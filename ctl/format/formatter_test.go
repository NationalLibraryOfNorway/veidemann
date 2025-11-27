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
	"bytes"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHasSuffix(t *testing.T) {
	type args struct {
		s      string
		suffix []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"1", args{"foo.yml", []string{".yml"}}, true},
		{"2", args{"foo.yml", []string{".yaml", ".yml"}}, true},
		{"3", args{"foo.yml", []string{".yaml", ".json"}}, false},
		{"4", args{"foo.yml", []string{".yaml"}}, false},
		{"5", args{"foo.yml", []string{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasSuffix(tt.args.s, tt.args.suffix...); got != tt.want {
				t.Errorf("HasSuffix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFormatter(t *testing.T) {
	ts, _ := time.Parse(time.RFC3339, "2019-12-24T17:00:00Z")
	startTime := timestamppb.New(ts)
	ts, _ = time.Parse(time.RFC3339, "2019-12-31T23:59:59Z")
	endTime := timestamppb.New(ts)

	jobExec := &frontierV1.JobExecutionStatus{Id: "id1", StartTime: startTime, EndTime: endTime}

	seed := &configV1.ConfigObject{
		ApiVersion: "v1",
		Kind:       configV1.Kind_seed,
		Id:         "id1",
		Meta:       &configV1.Meta{Name: "http://www.example.com"},
		Spec:       &configV1.ConfigObject_Seed{Seed: &configV1.Seed{EntityRef: &configV1.ConfigRef{Kind: configV1.Kind_crawlEntity, Id: "entityRef"}}}}

	json := "{\"foo\": \"bar\", \"val\": 42}"

	type args struct {
		kind     string
		format   string
		template string
	}
	tests := []struct {
		name   string
		args   args
		record interface{}
		want   string
	}{
		{"JobExec-json", args{format: "json"},
			jobExec,
			`{
  "bytesCrawled": "0",
  "desiredState": "UNDEFINED",
  "documentsCrawled": "0",
  "documentsDenied": "0",
  "documentsFailed": "0",
  "documentsOutOfScope": "0",
  "documentsRetried": "0",
  "endTime": "2019-12-31T23:59:59Z",
  "error": null,
  "executionsState": {},
  "id": "id1",
  "jobId": "",
  "startTime": "2019-12-24T17:00:00Z",
  "state": "UNDEFINED",
  "urisCrawled": "0"
}`},

		{"JobExec-yaml", args{format: "yaml"},
			jobExec,
			`bytesCrawled: "0"
documentsCrawled: "0"
documentsDenied: "0"
documentsFailed: "0"
documentsOutOfScope: "0"
documentsRetried: "0"
endTime: "2019-12-31T23:59:59Z"
error: null
executionsState: {}
id: id1
jobId: ""
startTime: "2019-12-24T17:00:00Z"
state: UNDEFINED
desiredState: UNDEFINED
urisCrawled: "0"
`},

		{"JobExec-table", args{format: "table", kind: "JobExecutionStatus"},
			jobExec,
			`Id                                             State     Docs     Uris Start time               End time                
                                 id1       UNDEFINED        0        0 2019-12-24T17:00:00Z     2019-12-31T23:59:59Z    
`},

		{"JobExec-template", args{format: "template", template: "ID: {{.Id}}, State: {{.State}}"},
			jobExec,
			`ID: id1, State: UNDEFINED`},

		{"Seed-json", args{format: "json"},
			seed,
			`{
  "apiVersion": "v1",
  "id": "id1",
  "kind": "seed",
  "meta": {
    "created": null,
    "createdBy": "",
    "description": "",
    "label": [],
	"annotation": [],
    "lastModified": null,
    "lastModifiedBy": "",
    "name": "http://www.example.com"
  },
  "seed": {
    "disabled": false,
    "entityRef": {
      "id": "entityRef",
      "kind": "crawlEntity"
    },
    "jobRef": []
  }
}`},

		{"Seed-yaml", args{format: "yaml"},
			seed,
			`apiVersion: v1
id: id1
kind: seed
meta:
  created: null
  createdBy: ""
  description: ""
  label: []
  annotation: []
  lastModified: null
  lastModifiedBy: ""
  name: http://www.example.com
seed:
  disabled: false
  entityRef:
    id: entityRef
    kind: crawlEntity
  jobRef: []

---
`},

		{"Seed-table", args{format: "table", kind: "seed"},
			seed,
			`Id                                   Url                                      EntityId                             Disabled
                                 id1 http://www.example.com                                              entityRef    false
`},

		{"Seed-template", args{format: "template", template: "ID: {{.Id}}, Url: {{.Meta.Name}}"},
			seed,
			`ID: id1, Url: http://www.example.com`},

		{"Json-json", args{format: "json"},
			json,
			`{"foo": "bar", "val": 42}`},

		{"Json-yaml", args{format: "yaml"},
			json,
			`foo: bar
val: 42

---
`},

		{"Json-template", args{format: "template", template: "ID: {{.foo}}, Val: {{.val}}"},
			json,
			`ID: bar, Val: 42`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := bytes.NewBuffer([]byte{})
			f, err := NewFormatter(tt.args.kind, out, tt.args.format, tt.args.template)
			if assert.NoError(t, err) {
				err = f.WriteRecord(tt.record)
				_ = f.Close()
				if assert.NoError(t, err) {
					switch tt.args.format {
					case "json":
						assert.JSONEq(t, tt.want, out.String(), "Got: '%s'", out.String())
					case "yaml":
						assert.YAMLEq(t, tt.want, out.String(), "Got: '%s'", out.String())
					default:
						assert.Equal(t, tt.want, out.String(), "Got: '%s'", out.String())
					}
				}
			}
		})
	}
}
