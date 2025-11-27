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

package apiutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// CreateSelector creates a label selector from a string.
func CreateSelector(labelString string) []string {
	if labelString != "" {
		return strings.Split(labelString, ",")
	}
	return nil
}

// stringSliceContains checks if a string slice contains a string.
func stringSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// CreateTemplateFilter creates a filter from a string by parsing the string and setting the value in the message.
//
// The filter string should have the format: <field>[.<field>]*[+|-]=<value>.
// The message should be a pointer to a proto message.
// The mask will be updated with the fields that are set.
func CreateTemplateFilter(filterString string, msg proto.Message, mask *commonsV1.FieldMask) error {
	path, value, ok := strings.Cut(filterString, "=")
	if !ok {
		return fmt.Errorf("invalid filter: %s", filterString)
	}
	if !stringSliceContains(mask.Paths, path) {
		mask.Paths = append(mask.Paths, path)
	}
	path = strings.TrimRight(path, "+-")
	tokens := strings.Split(path, ".")

	var fieldType protoreflect.FieldDescriptor
	msgType := msg.ProtoReflect().Descriptor()

	for _, token := range tokens {
		fieldType = msgType.Fields().ByJSONName(token)
		if fieldType == nil {
			sa := make([]string, msgType.Fields().Len())
			for i := 0; i < msgType.Fields().Len(); i++ {
				sa[i] = msgType.Fields().Get(i).JSONName()
			}
			return fmt.Errorf("no field with name '%s' in '%s'. Valid field names: %s", token, msgType.FullName(), strings.Join(sa, ", "))
		}
		switch fieldType.Kind() {
		case protoreflect.BoolKind:
			v, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("error converting %v to boolean: %w", value, err)
			}
			setValue(protoreflect.ValueOfBool(v), msg, fieldType)

		case protoreflect.Int32Kind:
			fallthrough
		case protoreflect.Sint32Kind:
			fallthrough
		case protoreflect.Sfixed32Kind:
			v, err := strconv.ParseInt(value, 0, 32)
			if err != nil {
				return fmt.Errorf("error converting %v to int: %w", value, err)
			}
			setValue(protoreflect.ValueOfInt32(int32(v)), msg, fieldType)

		case protoreflect.Int64Kind:
			fallthrough
		case protoreflect.Sint64Kind:
			fallthrough
		case protoreflect.Sfixed64Kind:
			v, err := strconv.ParseInt(value, 0, 64)
			if err != nil {
				return fmt.Errorf("error converting %v to int: %w", value, err)
			}
			setValue(protoreflect.ValueOfInt64(v), msg, fieldType)

		case protoreflect.Uint32Kind:
			fallthrough
		case protoreflect.Fixed32Kind:
			v, err := strconv.ParseUint(value, 0, 32)
			if err != nil {
				return fmt.Errorf("error converting %v to uint: %w", value, err)
			}
			setValue(protoreflect.ValueOfUint32(uint32(v)), msg, fieldType)

		case protoreflect.Uint64Kind:
			fallthrough
		case protoreflect.Fixed64Kind:
			v, err := strconv.ParseUint(value, 0, 64)
			if err != nil {
				return fmt.Errorf("error converting %v to uint: %w", value, err)
			}
			setValue(protoreflect.ValueOfUint64(v), msg, fieldType)

		case protoreflect.FloatKind:
			v, err := strconv.ParseFloat(value, 32)
			if err != nil {
				return fmt.Errorf("error converting %v to float: %w", value, err)
			}
			setValue(protoreflect.ValueOfFloat32(float32(v)), msg, fieldType)

		case protoreflect.DoubleKind:
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("error converting %v to float: %w", value, err)
			}
			setValue(protoreflect.ValueOfFloat64(v), msg, fieldType)

		case protoreflect.StringKind:
			setValue(protoreflect.ValueOfString(value), msg, fieldType)

		case protoreflect.BytesKind:
			setValue(protoreflect.ValueOfBytes([]byte(value)), msg, fieldType)

		case protoreflect.EnumKind:
			enumVal := fieldType.Enum().Values().ByName(protoreflect.Name(value))
			if enumVal == nil {
				sa := make([]string, fieldType.Enum().Values().Len())
				for i := 0; i < fieldType.Enum().Values().Len(); i++ {
					sa[i] = string(fieldType.Enum().Values().Get(i).Name())
				}
				return fmt.Errorf("not a valid enum value '%s' for '%s'. Valid values: %s", value, fieldType.FullName(), strings.Join(sa, ", "))
			}
			setValue(protoreflect.ValueOfEnum(enumVal.Number()), msg, fieldType)

		case protoreflect.GroupKind:
			// unsupported
		case protoreflect.MessageKind:
			var item protoreflect.Value
			isNewFieldValue := false
			if msg.ProtoReflect().Has(fieldType) {
				item = msg.ProtoReflect().Mutable(fieldType)
			} else {
				isNewFieldValue = true
				item = msg.ProtoReflect().NewField(fieldType)
			}

			var message protoreflect.Message
			if fieldType.IsList() {
				message = item.List().AppendMutable().Message()
			} else {
				message = item.Message()
				msgType = fieldType.Message()
			}

			switch m := message.Interface().(type) {
			case *configV1.ConfigRef:
				kind, id, ok := strings.Cut(value, ":")
				if kind == "" || id == "" || !ok {
					return fmt.Errorf("not a valid configRef value %v for: %v. ConfigRef should have format [kind]:[id]", value, fieldType.FullName())
				}
				m.Kind = configV1.Kind(configV1.Kind_value[kind])
				m.Id = id
			case *configV1.Label:
				k, v, ok := strings.Cut(value, ":")
				if k == "" || v == "" || !ok {
					return fmt.Errorf("not a valid label value %v for: %v. Label should have format [name]:[value]", value, fieldType.FullName())
				}
				m.Key = k
				m.Value = v
			}
			// Set the field value if it is a new field value
			if isNewFieldValue {
				msg.ProtoReflect().Set(fieldType, item)
			}
			msg = message.Interface()
		}
	}

	return nil
}

// setValue sets the value of a field in a proto message.
func setValue(v protoreflect.Value, obj proto.Message, fieldType protoreflect.FieldDescriptor) {
	if fieldType.IsList() {
		list := obj.ProtoReflect().NewField(fieldType)
		list.List().Append(v)
		v = list
	}
	obj.ProtoReflect().Set(fieldType, v)
}

// CreateListRequest creates a list request from the given parameters.
func CreateListRequest(kind configV1.Kind, ids []string, name string, labelString string, filters []string, pageSize int32, page int32) (*configV1.ListRequest, error) {
	var queryMask *commonsV1.FieldMask
	var queryTemplate *configV1.ConfigObject

	if filters != nil {
		queryMask = new(commonsV1.FieldMask)
		queryTemplate = new(configV1.ConfigObject)

		for _, filter := range filters {
			if err := CreateTemplateFilter(filter, queryTemplate, queryMask); err != nil {
				return nil, err
			}
		}
	}

	return &configV1.ListRequest{
		Kind:          kind,
		Id:            ids,
		NameRegex:     name,
		LabelSelector: CreateSelector(labelString),
		Offset:        page,
		PageSize:      pageSize,
		QueryTemplate: queryTemplate,
		QueryMask:     queryMask,
	}, nil
}

// CompleteName returns a list of names that matches the given name.
func CompleteName(kind string, name string) ([]string, error) {
	k := format.GetKind(kind)
	if k == configV1.Kind_undefined {
		return nil, errors.New("undefined kind")
	}

	request, err := CreateListRequest(k, nil, name, "", nil, 10, 0)
	if err != nil {
		return nil, err
	}

	request.ReturnedFieldsMask = &commonsV1.FieldMask{
		Paths: []string{"meta.name"},
	}

	conn, err := connection.Connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := configV1.NewConfigClient(conn)
	r, err := client.ListConfigObjects(context.Background(), request)
	if err != nil {
		return nil, err
	}

	var names []string
	for {
		msg, err := r.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		names = append(names, msg.Meta.Name)
	}
	return names, nil
}
