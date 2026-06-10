/*
 * Copyright 2020 National Library of Norway.
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

package session

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
)

func TestMatch(t *testing.T) {
	type sample struct {
		uri  string
		want bool
	}
	type test struct {
		pattern string
		samples []sample
	}
	tests := []test{
		{
			"^https://example[.]com/$",
			[]sample{
				{"https://example.com/", true},
				{"https://example.com/bad", false},
			},
		},
		{
			"^https://example[.]com/",
			[]sample{
				{"https://example.com/", true},
				{"https://example.com/bad", true},
			},
		},
	}

	for _, test := range tests {
		for _, sample := range test.samples {
			regexps := []string{test.pattern}
			t.Run(sample.uri, func(t *testing.T) {
				got := match(regexps, sample.uri)
				if got != sample.want {
					t.Errorf("Got %t; want %t", got, sample.want)
				}
			})
		}
	}
}

func TestBuildNewDocumentScriptSource(t *testing.T) {
	arguments := json.RawMessage(`{"flag":true}`)
	source := buildNewDocumentScriptSource("function init(args) { window.__flag = args.flag; }", arguments)

	checks := []string{
		"const __veidemannFn = (function init(args) { window.__flag = args.flag; });",
		"void __veidemannFn({\"flag\":true});",
	}

	for _, check := range checks {
		if !strings.Contains(source, check) {
			t.Fatalf("source %q does not contain %q", source, check)
		}
	}
}

func TestScriptPriority(t *testing.T) {
	tests := []struct {
		name         string
		configObject *configV1.ConfigObject
		want         int
	}{
		{
			name:         "missing meta defaults to zero",
			configObject: &configV1.ConfigObject{},
			want:         0,
		},
		{
			name:         "missing priority label defaults to zero",
			configObject: newBrowserScriptConfigObject("default", "default", configV1.BrowserScript_ON_LOAD),
			want:         0,
		},
		{
			name: "priority label is parsed",
			configObject: newBrowserScriptConfigObject(
				"prioritized",
				"prioritized",
				configV1.BrowserScript_ON_LOAD,
				&configV1.Label{Key: "priority", Value: "3"},
			),
			want: 3,
		},
		{
			name: "negative priority is parsed",
			configObject: newBrowserScriptConfigObject(
				"negative",
				"negative",
				configV1.BrowserScript_ON_LOAD,
				&configV1.Label{Key: "priority", Value: "-1"},
			),
			want: -1,
		},
		{
			name: "malformed priority defaults to zero",
			configObject: newBrowserScriptConfigObject(
				"invalid",
				"invalid",
				configV1.BrowserScript_ON_LOAD,
				&configV1.Label{Key: "priority", Value: "high"},
			),
			want: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := scriptPriority(test.configObject)
			if got != test.want {
				t.Fatalf("scriptPriority() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestOrderOnLoadScripts(t *testing.T) {
	scripts := []*configV1.ConfigObject{
		newBrowserScriptConfigObject("default-first", "default-first", configV1.BrowserScript_ON_LOAD),
		newBrowserScriptConfigObject(
			"high",
			"high",
			configV1.BrowserScript_ON_LOAD,
			&configV1.Label{Key: "priority", Value: "10"},
		),
		newBrowserScriptConfigObject(
			"default-second",
			"default-second",
			configV1.BrowserScript_ON_LOAD,
			&configV1.Label{Key: "category", Value: "test"},
		),
		newBrowserScriptConfigObject(
			"same-priority-a",
			"same-priority-a",
			configV1.BrowserScript_ON_LOAD,
			&configV1.Label{Key: "priority", Value: "5"},
		),
		newBrowserScriptConfigObject(
			"same-priority-b",
			"same-priority-b",
			configV1.BrowserScript_ON_LOAD,
			&configV1.Label{Key: "priority", Value: "5"},
		),
		newBrowserScriptConfigObject(
			"invalid-priority",
			"invalid-priority",
			configV1.BrowserScript_ON_LOAD,
			&configV1.Label{Key: "priority", Value: "high"},
		),
			newBrowserScriptConfigObject(
				"negative-priority",
				"negative-priority",
				configV1.BrowserScript_ON_LOAD,
				&configV1.Label{Key: "priority", Value: "-1"},
			),
	}

	orderOnLoadScripts(scripts)

	got := make([]string, len(scripts))
	for i, script := range scripts {
		got[i] = script.GetId()
	}

	want := []string{
		"high",
		"same-priority-a",
		"same-priority-b",
		"default-first",
		"default-second",
		"invalid-priority",
		"negative-priority",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ordered script ids = %v, want %v", got, want)
	}
}

func newBrowserScriptConfigObject(id, name string, scriptType configV1.BrowserScript_BrowserScriptType, labels ...*configV1.Label) *configV1.ConfigObject {
	return &configV1.ConfigObject{
		Id: id,
		Meta: &configV1.Meta{
			Name:  name,
			Label: labels,
		},
		Spec: &configV1.ConfigObject_BrowserScript{BrowserScript: &configV1.BrowserScript{BrowserScriptType: scriptType}},
	}
}
