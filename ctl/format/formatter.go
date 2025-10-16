// Copyright © 2017 National Library of Norway.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package format

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/api/config"
	"github.com/invopop/yaml"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var jsonMarshaler = &protojson.MarshalOptions{EmitUnpopulated: true}
var jsonUnMarshaler = &protojson.UnmarshalOptions{DiscardUnknown: true}

//go:embed res
var res embed.FS

// templateDir is the directory where the templates are located
const templateDir = "res/"

// Formatter is the interface for formatters.
// Formatters should not be instantiated directly, but through the NewFormatter function.
type Formatter interface {
	// WriteRecord writes a record to the formatter.
	// The input record is guaranteed to be a protobuf message
	// or the result of parsing a json formatted string into an interface (see https://pkg.go.dev/encoding/json#Unmarshal).
	WriteRecord(interface{}) error
	Close() error
}

// anyElement is a struct that can hold any type of element.
// It is used as input to the json unmarshaler.
type anyElement struct {
	v interface{}
}

// UnmarshalJSON implements the encoding/json.Unmarshaler interface.
// The function uses encoding.json.Unmarshal to unmarshal the input.
// If the input is a map[string]interface{}, the map is traversed to find any RethinkDb dates
// which are converted to RFC3339 formatted strings.
func (r *anyElement) UnmarshalJSON(b []byte) error {
	var i interface{}
	err := json.Unmarshal(b, &i)
	if err != nil {
		return err
	}

	switch j := i.(type) {
	case map[string]interface{}:
		if d, ok := r.formatDate(j); ok {
			r.v = d
			return nil
		}

		r.traverseMap(&j)
		r.v = j
	default:
		r.v = i
	}

	return err
}

func (r *anyElement) traverseMap(i *map[string]interface{}) {
	for k, v := range *i {
		if m, ok := v.(map[string]interface{}); ok {
			if d, ok := r.formatDate(m); ok {
				(*i)[k] = d
			} else {
				r.traverseMap(&m)
			}
		}
	}
}

// getAsInt returns the value as an int if it is a float64 or int
func getAsInt(v interface{}) (int, bool) {
	switch i := v.(type) {
	case float64:
		return int(i), true
	case int:
		return i, true
	default:
		return 0, false
	}
}

// formatDate first checks if i is a RethinkDb date.
// If so, date is the result of converting the date to a RFC3339 formatted string and isDate is true.
// Otherwise, date is the empty string and isDate is false.
func (r *anyElement) formatDate(element map[string]interface{}) (date string, isDate bool) {
	var year, month, day, hour, minute, second, nano, offset int

	if dateTime, ok := element["dateTime"].(map[string]interface{}); !ok {
		return "", false
	} else {
		if date, ok := dateTime["date"].(map[string]interface{}); !ok {
			return "", false
		} else {
			if year, ok = getAsInt(date["year"]); !ok {
				return "", false
			}
			if month, ok = getAsInt(date["month"]); !ok {
				return "", false
			}
			if day, ok = getAsInt(date["day"]); !ok {
				return "", false
			}
		}
		if tm, ok := dateTime["time"].(map[string]interface{}); !ok {
			return "", false
		} else {
			if hour, ok = getAsInt(tm["hour"]); !ok {
				return "", false
			}
			if minute, ok = getAsInt(tm["minute"]); !ok {
				return "", false
			}
			if second, ok = getAsInt(tm["second"]); !ok {
				return "", false
			}
			if nano, ok = getAsInt(tm["nano"]); !ok {
				return "", false
			}
		}
	}
	if of, ok := element["offset"].(map[string]interface{}); !ok {
		return "", false
	} else {
		if offset, ok = getAsInt(of["totalSeconds"]); !ok {
			return "", false
		}
	}
	tz := time.UTC
	if offset != 0 {
		tz = time.FixedZone(fmt.Sprintf("OFF%.d", offset), offset)
	}
	d := time.Date(year, time.Month(month), day, hour, minute, second, nano, tz)
	return d.Format(time.RFC3339Nano), true
}

// preFormatter wraps a formatter and converts json strings to objects
type preFormatter struct {
	formatter Formatter
}

// WriteRecord implements the Formatter interface.
// If the input record is a string, it is parsed as json and the result is passed to the wrapped formatter.
// If the input record is a protobuf message, it is passed to the wrapped formatter.
// Otherwise, an error is returned.
func (p *preFormatter) WriteRecord(record interface{}) error {
	switch v := record.(type) {
	case string:
		var j anyElement
		err := json.Unmarshal([]byte(v), &j)
		if err != nil {
			return fmt.Errorf("failed to parse json: %w", err)
		}
		record = j.v
	case proto.Message:
		// Do nothing, just pass through
	default:
		return fmt.Errorf("unsupported type '%T'", v)
	}

	return p.formatter.WriteRecord(record)
}

func (p *preFormatter) Close() error {
	return p.formatter.Close()
}

// MarshalSpec is the specification for a formatter
type MarshalSpec struct {
	ObjectType          string
	Format              string
	Template            string
	defaultTemplateName string

	rFormat   string
	rTemplate string
	rWriter   io.Writer
	resolved  bool
}

// NewFormatter creates a new formatter
func NewFormatter(objectType string, out io.Writer, format string, template string) (formatter Formatter, err error) {
	s := &MarshalSpec{
		ObjectType: objectType,
		Format:     format,
		Template:   template,
		rWriter:    out,
	}

	if err := s.resolve(); err != nil {
		return nil, fmt.Errorf("failed to resolve formatter: %w", err)
	}

	switch s.rFormat {
	case "json":
		formatter = newJsonFormatter(s)
	case "yaml":
		formatter = newYamlFormatter(s)
	case "template":
		formatter, err = newTemplateFormatter(s)
	case "table":
		formatter, err = newTemplateFormatter(s)
	case "wide":
		formatter, err = newTemplateFormatter(s)
	default:
		return nil, fmt.Errorf("illegal or missing format '%s'", s.rFormat)
	}
	return
}

// ResolveWriter creates a file and returns an io.Writer for the file.
// If filename is empty, os.StdOut is returned
func ResolveWriter(filename string) (io.WriteCloser, error) {
	if filename == "" || filename == "-" {
		return os.Stdout, nil
	} else {
		f, err := os.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create file '%s': %w", filename, err)
		}
		return f, nil
	}
}

// resolve resolves the template and format
func (s *MarshalSpec) resolve() (err error) {
	if !s.resolved {
		switch s.Format {
		case "template":
			if s.Template == "" {
				if s.defaultTemplateName != "" {
					data, err := res.ReadFile(templateDir + s.defaultTemplateName)
					if err != nil {
						return err
					}
					s.rTemplate = string(data)
				} else {
					return errors.New("Format is 'template', but template is missing")
				}
			} else {
				s.rTemplate = s.Template
			}
			s.rFormat = s.Format
		case "template-file":
			if s.Template == "" {
				return errors.New("format is 'template-file', but template is missing")
			}
			data, err := os.ReadFile(s.Template)
			if err != nil {
				return fmt.Errorf("template not found: %w", err)
			}
			s.rTemplate = string(data)
			s.rFormat = s.Format
		case "json":
			s.rTemplate = ""
			s.rFormat = s.Format
		case "yaml":
			s.rTemplate = ""
			s.rFormat = s.Format
		case "table":
			if s.ObjectType == "" {
				return fmt.Errorf("format is table, but object type is missing")
			}

			templateName := s.ObjectType + "_table.template"
			data, err := res.ReadFile(templateDir + templateName)
			if err != nil {
				return err
			}
			s.rTemplate = string(data)
			s.rFormat = s.Format
		case "wide":
			if s.ObjectType == "" {
				return fmt.Errorf("format is wide, but object type is missing")
			}

			templateName := s.ObjectType + "_wide.template"
			data, err := res.ReadFile(templateDir + templateName)
			if err != nil {
				return err
			}
			s.rTemplate = string(data)
			s.rFormat = s.Format
		default:
			s.rTemplate = s.Template
			s.rFormat = s.Format
		}
	}
	s.resolved = true
	return nil
}

// Close closes the formatter
func (s *MarshalSpec) Close() error {
	if s == nil {
		return nil
	}
	if closer, ok := s.rWriter.(io.Closer); ok && closer != os.Stdout {
		return closer.Close()
	}
	return nil
}

type fileType string

const jsonFile fileType = "json"
const yamlFile fileType = "yaml"

// unmarshal unmarshals a file into a channel of ConfigObjects based on the file type
func unmarshal(r io.Reader, result chan<- *config.ConfigObject, done <-chan struct{}, t fileType) error {
	var err error

	switch t {
	case jsonFile:
		err = unmarshalJson(r, result, done)
	case yamlFile:
		err = unmarshalYaml(r, result, done)
	default:
		err = fmt.Errorf("unknown file type '%s'", t)
	}

	if err != nil {
		return err
	}
	return nil
}

// Unmarshal unmarshals a file into a channel of ConfigObjects
func Unmarshal(ctx context.Context, filename string, result chan<- *config.ConfigObject) error {
	var err error
	var f *os.File
	var t fileType

	if filename == "" {
		r := bufio.NewReader(os.Stdin)

		// read one byte to determine if json or yaml
		b, err := r.Peek(1)
		if err != nil {
			return err
		}
		if b[0] == '{' {
			t = jsonFile
		} else {
			t = yamlFile
		}

		go func() {
			defer close(result)
			err := unmarshal(r, result, ctx.Done(), t)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal")
			}
		}()

		return nil
	}

	f, err = os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file '%s': %w", filename, err)
	}

	if fi, _ := f.Stat(); !fi.IsDir() {
		if HasSuffix(f.Name(), ".yaml", ".yml") {
			t = yamlFile
		} else if HasSuffix(f.Name(), ".json", ".jsonl") {
			t = jsonFile
		}

		go func() {
			defer close(result)
			err := unmarshal(f, result, ctx.Done(), t)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal")
			}
		}()

		return nil
	}

	des, err := f.ReadDir(0)
	if err != nil {
		return fmt.Errorf("failed to read directory '%s': %w", filename, err)
	}

	var wg sync.WaitGroup
	for _, fi := range des {
		if fi.IsDir() {
			continue
		}

		if HasSuffix(fi.Name(), ".yaml", ".yml") {
			t = yamlFile
		} else if HasSuffix(fi.Name(), ".json", ".jsonl") {
			t = jsonFile
		}

		f, err = os.Open(fi.Name())
		if err != nil {
			log.Error().Err(err).Msg("failed to open file")
			continue
		}

		wg.Add(1)
		go func() {
			f := f
			defer wg.Done()
			defer f.Close()
			err := unmarshal(f, result, ctx.Done(), t)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal")
			}
		}()
	}
	// wait for all goroutines to finish
	wg.Wait()
	// close the channel to end the loop in the caller
	close(result)

	return nil
}

type yamlReader struct {
	*bufio.Reader
}

func newYamlReader(r io.Reader) yamlReader {
	return yamlReader{Reader: bufio.NewReader(r)}
}

// readYaml reads a yaml document from the reader and returns it as a byte array
func (yr yamlReader) readYaml() ([]byte, error) {
	delim := []byte{'-', '-', '-'}
	var inDoc bool = true
	var err error
	var l, doc []byte

	for inDoc && err == nil {
		isPrefix := true
		ln := []byte{}
		for isPrefix && err == nil {
			l, isPrefix, err = yr.ReadLine()
			ln = append(ln, l...)
		}

		if len(ln) >= 3 && bytes.Equal(delim, ln[:3]) {
			inDoc = false
		} else if len(ln) > 0 {
			doc = append(doc, ln...)
			doc = append(doc, '\n')
		}
	}

	return doc, err
}

// readJson reads a yaml document from the reader and returns it as a json byte array
func (yr yamlReader) readJson() ([]byte, error) {
	data, err := yr.readYaml()
	if err != nil {
		return nil, err
	}

	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}

	// convert yaml to json before unmarshaling because protojson doesn't support yaml
	return yaml.YAMLToJSON(data)
}

// unmarshalYaml unmarshals a yaml stream into ConfigObjects and sends them to the result channel
func unmarshalYaml(r io.Reader, result chan<- *config.ConfigObject, done <-chan struct{}) error {
	yr := newYamlReader(r)

	var b []byte
	var err error

	for {
		b, err = yr.readJson()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := &config.ConfigObject{}
		err = jsonUnMarshaler.Unmarshal(b, target)
		if err != nil {
			return err
		}
		select {
		case <-done:
			return nil
		case result <- target:
		}
	}
}

// unmarshalJson unmarshals a json stream into ConfigObjects and sends them to the result channel
func unmarshalJson(r io.Reader, result chan<- *config.ConfigObject, done <-chan struct{}) error {
	dec := json.NewDecoder(r)
	var err error
	var msg json.RawMessage

	for {
		err = dec.Decode(&msg)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := &config.ConfigObject{}
		err = jsonUnMarshaler.Unmarshal(msg, target)
		if err != nil {
			return err
		}
		select {
		case <-done:
			return nil
		case result <- target:
		}
	}
}

// HasSuffix checks if a string has one of the suffixes given as input
func HasSuffix(s string, suffix ...string) bool {
	for _, suf := range suffix {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}
