// Copyright © 2023 National Library of Norway
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

package importutil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// LineAsStringDecoder is a decoder that reads a line from the input as a string
type LineAsStringDecoder struct {
	r *bufio.Reader
}

func (l *LineAsStringDecoder) Init(r io.Reader, suffix string) {
	l.r = bufio.NewReader(r)
}

func (l *LineAsStringDecoder) Read(v interface{}) error {
	s, err := l.r.ReadString('\n')
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("invalid target: %v", reflect.TypeOf(v))
	}

	s = strings.TrimSpace(s)
	rv.Elem().Set(reflect.ValueOf(&s).Elem())

	return nil
}

type decoder interface {
	Decode(v interface{}) error
}

// JsonYamlDecoder is a decoder that reads json or yaml from the input and decodes it into a struct
type JsonYamlDecoder struct {
	decoder
}

func (j *JsonYamlDecoder) Init(r io.Reader, suffix string) {
	if suffix == ".yaml" || suffix == ".yml" {
		dec := yaml.NewDecoder(r)
		// TODO check this
		dec.KnownFields(true)
		j.decoder = dec
	} else {
		dec := json.NewDecoder(r)
		dec.DisallowUnknownFields()
		j.decoder = dec
	}
}

func (j *JsonYamlDecoder) Read(v interface{}) error {
	return j.Decode(v)
}
