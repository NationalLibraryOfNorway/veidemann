// Copyright © 2019 National Library of Norway
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
	"reflect"
	"testing"
)

func TestEncodeDecodeStringArray(t *testing.T) {
	tests := []struct {
		name  string
		value []string
	}{
		{"1", []string{"foo", "bar"}},
		{"2", []string{}},
		{"3", []string{}},
		{"4", []string{"", "foo", "bar"}},
		{"5", []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &ImportDb{}
			b := d.stringArrayToBytes(tt.value)
			if got := d.bytesToStringArray(b); !reflect.DeepEqual(got, tt.value) {
				t.Errorf("ImportDb.bytesToStringArray(ImportDb.stringArrayToBytes()) = %v, want %v", got, tt.value)
			}
		})
	}
}

func TestExistsCode_ExistsInVeidemann(t *testing.T) {
	tests := []struct {
		name string
		e    ExistsCode
		want bool
	}{
		{"error", Undefined, false},
		{"new", NewKey, false},
		{"exists_veidemann", NewId, true},
		{"duplicate_veidemann", Exists, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.ExistsInVeidemann(); got != tt.want {
				t.Errorf("ExistsCode.ExistsInVeidemann() = %v, want %v", got, tt.want)
			}
		})
	}
}
