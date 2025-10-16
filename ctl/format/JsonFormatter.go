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
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// jsonFormatter is a formatter that writes records as json
type jsonFormatter struct {
	*MarshalSpec
}

// newJsonFormatter creates a new json formatter
func newJsonFormatter(s *MarshalSpec) Formatter {
	return &preFormatter{
		&jsonFormatter{
			MarshalSpec: s,
		},
	}
}

// WriteRecord writes a record to the formatters writer
func (jf *jsonFormatter) WriteRecord(record interface{}) error {
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
				m, ok := values.Index(i).Interface().(proto.Message)
				if !ok {
					return fmt.Errorf("illegal record type '%T'", record)
				}
				err := marshalElementJson(jf.rWriter, m)
				if err != nil {
					return err
				}
			}
		} else {
			err := marshalElementJson(jf.rWriter, v)
			if err != nil {
				return err
			}
		}
	default:
		j, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(jf.rWriter, string(j))
		return err
	}
	return nil
}

// marshalElementJson marshals a proto message to json
func marshalElementJson(w io.Writer, msg proto.Message) error {
	if b, err := jsonMarshaler.Marshal(msg); err != nil {
		return fmt.Errorf("could not convert %v to JSON: %w", msg, err)
	} else {
		_, err := fmt.Fprintln(w, string(b))
		return err
	}
}
