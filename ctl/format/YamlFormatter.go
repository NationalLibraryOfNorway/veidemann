// Copyright © 2017 National Library of Norway
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
	"fmt"
	"io"
	"reflect"

	"github.com/invopop/yaml"
	"google.golang.org/protobuf/proto"
)

// yamlFormatter is a formatter that writes records as yaml
type yamlFormatter struct {
	*MarshalSpec
}

// newYamlFormatter creates a new yaml formatter
func newYamlFormatter(s *MarshalSpec) Formatter {
	return &preFormatter{
		&yamlFormatter{
			MarshalSpec: s,
		},
	}
}

// WriteRecord writes a record to the formatters writer
func (yf *yamlFormatter) WriteRecord(record interface{}) error {
	switch v := record.(type) {
	case proto.Message:
		var values reflect.Value
		values = reflect.ValueOf(v).Elem().FieldByName("Value")
		if values.IsValid() {
			if values.Len() == 0 {
				fmt.Println("Empty result")
				return nil
			}

			for i := 0; i < values.Len(); i++ {
				var err error
				if i > 0 {
					_, err = yf.rWriter.Write([]byte("---\n"))
				}
				if err != nil {
					return err
				}

				m, ok := values.Index(i).Interface().(proto.Message)
				if !ok {
					return fmt.Errorf("illegal record type '%T'", record)
				}
				err = marshalElementYaml(yf.rWriter, m)
				if err != nil {
					return err
				}
			}
		} else {
			err := marshalElementYaml(yf.rWriter, v)
			if err != nil {
				return err
			}
			_, err = yf.rWriter.Write([]byte("---\n"))
			return err
		}
	default:
		final, err := yaml.Marshal(record)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return err
		}

		_, err = fmt.Fprint(yf.rWriter, string(final))
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(yf.rWriter, "---")
		if err != nil {
			return err
		}
	}
	return nil
}

// marshalElementYaml marshals a proto message to yaml
func marshalElementYaml(w io.Writer, msg proto.Message) error {
	r, err := jsonMarshaler.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to convert %v to JSON: %w", msg, err)
	}

	final, err := yaml.JSONToYAML(r)
	if err != nil {
		return fmt.Errorf("failed to convert %v to YAML: %w", msg, err)
	}

	_, err = fmt.Fprintln(w, string(final))
	return err
}
