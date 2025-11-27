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
	"fmt"
	"sort"
	"strings"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
)

// GetKind returns the Kind for the given name.
func GetKind(name string) configV1.Kind {
	name = strings.ToLower(name)
	for _, k := range configV1.Kind_name {
		if strings.ToLower(k) == name {
			return configV1.Kind(configV1.Kind_value[k])
		}
	}
	return configV1.Kind_undefined
}

// GetObjectNames returns a sorted list of object names.
func GetObjectNames() []string {
	result := make([]string, len(configV1.Kind_name)-1)
	idx := 0
	for _, n := range configV1.Kind_name {
		if n != "undefined" {
			result[idx] = n
			idx++
		}
	}
	sort.Strings(result)
	return result
}

// ListObjectNames returns a string with a list of object names.
func ListObjectNames() string {
	var names string
	for _, v := range GetObjectNames() {
		names += fmt.Sprintf(" - %s\n", v)
	}
	return fmt.Sprintln(names)
}
