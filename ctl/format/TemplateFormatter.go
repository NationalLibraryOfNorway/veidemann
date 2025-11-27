/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// templateFormatter is a formatter that uses a template to format the output
type templateFormatter struct {
	*MarshalSpec
	headerWritten  bool
	parsedTemplate *template.Template
}

// newTemplateFormatter creates a new template formatter
func newTemplateFormatter(s *MarshalSpec) (Formatter, error) {
	t := &templateFormatter{
		MarshalSpec: s,
	}
	pt, err := parseTemplate(t.rTemplate)
	if err != nil {
		return nil, err
	}
	t.parsedTemplate = pt
	return &preFormatter{t}, nil
}

// WriteRecord writes a record to the formatters writer
func (tf *templateFormatter) WriteRecord(record interface{}) error {
	if !tf.headerWritten {
		tf.headerWritten = true
		tpl := tf.parsedTemplate.Lookup("HEADER")
		if tpl != nil {
			err := tpl.Execute(tf.rWriter, nil)
			if err != nil {
				return fmt.Errorf("failed applying header template: %w", err)
			}
		}
	}

	tpl := tf.parsedTemplate
	if tpl != nil {
		if r, ok := record.(string); ok {
			var j interface{}
			err := json.Unmarshal([]byte(r), &j)
			if err != nil {
				return fmt.Errorf("failed to parse json: %w", err)
			}
			record = j
		}
		err := tpl.Execute(tf.rWriter, record)
		if err != nil {
			return fmt.Errorf("failed applying template to '%v': %w", record, err)
		}
	}
	return nil
}

// parseTemplate parses a template string and returns a template
func parseTemplate(templateString string) (*template.Template, error) {
	ESC := string(rune(0x1b))
	funcMap := template.FuncMap{
		"reset":         func() string { return ESC + "[0m" },
		"bold":          func() string { return ESC + "[1m" },
		"inverse":       func() string { return ESC + "[7m" },
		"red":           func() string { return ESC + "[31m" },
		"green":         func() string { return ESC + "[32m" },
		"yellow":        func() string { return ESC + "[33m" },
		"blue":          func() string { return ESC + "[34m" },
		"magenta":       func() string { return ESC + "[35m" },
		"cyan":          func() string { return ESC + "[36m" },
		"brightred":     func() string { return ESC + "[1;31m" },
		"brightgreen":   func() string { return ESC + "[1;32m" },
		"brightyellow":  func() string { return ESC + "[1;33m" },
		"brightblue":    func() string { return ESC + "[1;34m" },
		"brightmagenta": func() string { return ESC + "[1;35m" },
		"brightcyan":    func() string { return ESC + "[1;36m" },
		"bgwhite":       func() string { return ESC + "[47m" },
		"bgbrightblack": func() string { return ESC + "[100m" },
		"time": func(ts *timestamppb.Timestamp) string {
			if ts == nil {
				return "                        "
			} else {
				return fmt.Sprintf("%-24.24s", ts.AsTime().Format(time.RFC3339))
			}
		},
		"json": func(v interface{}) (string, error) {
			if v == nil {
				return "", nil
			} else {
				var buf bytes.Buffer
				encoder := json.NewEncoder(&buf)
				encoder.SetEscapeHTML(false)
				err := encoder.Encode(v)
				if err != nil {
					return "", err
				}
				return buf.String(), nil
			}
		},
		"prettyJson": func(v interface{}) (string, error) {
			if v == nil {
				return "", nil
			} else {
				var buf bytes.Buffer
				encoder := json.NewEncoder(&buf)
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "  ")
				err := encoder.Encode(v)
				if err != nil {
					return "", err
				}
				return buf.String(), nil
			}
		},
		"nl": func() string { return "\n" },
		"join": func(sep string, v interface{}) string {
			a := reflect.ValueOf(v)
			if a.Kind() != reflect.Slice {
				s, _ := v.(string)
				return s
			}
			var b strings.Builder
			if a.Len() == 0 {
				return ""
			}
			b.WriteString(fmt.Sprintf("%s", a.Index(0)))
			for i := 1; i < a.Len(); i++ {
				b.WriteString(sep)
				b.WriteString(fmt.Sprintf("%s", a.Index(i)))
			}
			return b.String()
		},
		"flatMap": func(v interface{}, field ...string) []interface{} {
			slice := reflect.ValueOf(v)
			if slice.Kind() != reflect.Slice {
				return nil
			}
			n := slice.Len()
			fieldCount := len(field)
			if fieldCount == 0 {
				fieldCount = 1
			}
			res := make([]interface{}, n*fieldCount)

			for i := 0; i < n; i++ {
				val := reflect.ValueOf(v).Index(i)
				if len(field) == 0 {
					res[i] = fmt.Sprintf("%s", val)
				} else {
					if val.Kind() == reflect.Ptr {
						val = val.Elem()
					}
					for j, f := range field {
						res[i*fieldCount+j] = fmt.Sprintf("%s", val.FieldByName(f))
					}
				}
			}

			return res
		},
		"printLabels": func(v interface{}) string {
			if labels, ok := v.([]*configV1.Label); ok {
				b := strings.Builder{}
				// b.WriteString("[")
				for i, l := range labels {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(l.Key)
					b.WriteString(":")
					b.WriteString(l.Value)
					// b.WriteString("'")
				}
				// b.WriteString("]")
				return b.String()
			}
			return ""
		},
	}

	return template.New("Template").Funcs(funcMap).Parse(templateString)
}
