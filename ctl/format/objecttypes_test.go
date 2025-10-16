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
	"reflect"
	"testing"

	"github.com/magiconair/properties/assert"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config"
)

func TestGetKind(t *testing.T) {
	tests := []struct {
		name     string
		kindName string
		want     configV1.Kind
	}{
		{"1", "sEed", configV1.Kind_seed},
		{"2", "crawlentity", configV1.Kind_crawlEntity},
		{"3", "crawlcoonfig", configV1.Kind_undefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetKind(tt.kindName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetObjectNames(t *testing.T) {
	assert.Equal(t, GetObjectNames(), []string{"browserConfig", "browserScript", "collection", "crawlConfig", "crawlEntity", "crawlHostGroupConfig", "crawlJob", "crawlScheduleConfig", "politenessConfig", "roleMapping", "seed"})
}
