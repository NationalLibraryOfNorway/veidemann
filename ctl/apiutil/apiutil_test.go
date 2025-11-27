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
	"reflect"
	"testing"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"google.golang.org/protobuf/proto"

	api "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
)

func TestCreateSelector(t *testing.T) {
	type args struct {
		labelString string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Empty label",
			args: args{""},
			want: nil,
		},
		{
			name: "Single label",
			args: args{"foo:bar"},
			want: []string{"foo:bar"},
		},
		{
			name: "Multiple labels",
			args: args{"foo:bar,lab"},
			want: []string{"foo:bar", "lab"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateSelector(tt.args.labelString); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateListRequest(t *testing.T) {
	type args struct {
		ids         []string
		name        string
		labelString string
		pageSize    int32
		page        int32
	}
	tests := []struct {
		name string
		args args
		want *api.ListRequest
	}{
		{
			name: "One Id",
			args: args{
				[]string{"id1"},
				"",
				"",
				0,
				0,
			},
			want: &api.ListRequest{Kind: api.Kind_browserConfig, Id: []string{"id1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateListRequest(api.Kind_browserConfig, tt.args.ids, tt.args.name, tt.args.labelString, nil, tt.args.pageSize, tt.args.page)
			if err != nil {
				t.Errorf("Error in CreateListRequest(): %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateListRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateTemplateFilter(t *testing.T) {
	type args struct {
		filterString string
		templateObj  proto.Message
	}
	tests := []struct {
		name         string
		args         args
		wantMask     *commonsV1.FieldMask
		wantTemplate proto.Message
		wantErr      bool
	}{
		{"message/string",
			args{filterString: "meta.name=foo", templateObj: &configV1.ConfigObject{}},
			&commonsV1.FieldMask{Paths: []string{"meta.name"}},
			&configV1.ConfigObject{Meta: &configV1.Meta{Name: "foo"}},
			false,
		},
		{"message/storageRef",
			args{filterString: "crawlConfig.browserConfigRef=browserConfig:foo", templateObj: &configV1.ConfigObject{}},
			&commonsV1.FieldMask{Paths: []string{"crawlConfig.browserConfigRef"}},
			&configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{BrowserConfigRef: &configV1.ConfigRef{Kind: configV1.Kind_browserConfig, Id: "foo"}}}},
			false,
		},
		{"illegal message/storageRef",
			args{filterString: "crawlConfig.browserConfigRef=foo", templateObj: &configV1.ConfigObject{}},
			nil,
			&configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{}}},
			true,
		},
		{"message/label",
			args{filterString: "meta.label=foo:bar", templateObj: &configV1.ConfigObject{}},
			&commonsV1.FieldMask{Paths: []string{"meta.label"}},
			&configV1.ConfigObject{Meta: &configV1.Meta{Label: []*api.Label{{Key: "foo", Value: "bar"}}}},
			false,
		},
		{"oneof/int",
			args{filterString: "browserConfig.pageLoadTimeoutMs=100", templateObj: &configV1.ConfigObject{}},
			&commonsV1.FieldMask{Paths: []string{"browserConfig.pageLoadTimeoutMs"}},
			&configV1.ConfigObject{Spec: &configV1.ConfigObject_BrowserConfig{BrowserConfig: &configV1.BrowserConfig{PageLoadTimeoutMs: 100}}},
			false,
		},
		{"illegal filter for template",
			args{filterString: "meta.name=foo", templateObj: &frontierV1.CrawlExecutionStatus{}},
			nil,
			&frontierV1.CrawlExecutionStatus{},
			true,
		},
		{"enum",
			args{filterString: "state=FINISHED", templateObj: &frontierV1.CrawlExecutionStatus{}},
			&commonsV1.FieldMask{Paths: []string{"state"}},
			&frontierV1.CrawlExecutionStatus{State: frontierV1.CrawlExecutionStatus_FINISHED},
			false,
		},
		{"illegal enum value",
			args{filterString: "state=FOO", templateObj: &frontierV1.CrawlExecutionStatus{}},
			nil,
			&frontierV1.CrawlExecutionStatus{},
			true,
		},
		{"array/set",
			args{filterString: "browserScript.urlRegexp=foo", templateObj: &configV1.ConfigObject{}},
			&commonsV1.FieldMask{Paths: []string{"browserScript.urlRegexp"}},
			&configV1.ConfigObject{Spec: &configV1.ConfigObject_BrowserScript{BrowserScript: &configV1.BrowserScript{UrlRegexp: []string{"foo"}}}},
			false,
		},
		{"array/add",
			args{filterString: "browserScript.urlRegexp+=foo", templateObj: &configV1.ConfigObject{}},
			&commonsV1.FieldMask{Paths: []string{"browserScript.urlRegexp+"}},
			&configV1.ConfigObject{Spec: &configV1.ConfigObject_BrowserScript{BrowserScript: &configV1.BrowserScript{UrlRegexp: []string{"foo"}}}},
			false,
		},
		{"array/remove",
			args{filterString: "browserScript.urlRegexp-=foo", templateObj: &configV1.ConfigObject{}},
			&commonsV1.FieldMask{Paths: []string{"browserScript.urlRegexp-"}},
			&configV1.ConfigObject{Spec: &configV1.ConfigObject_BrowserScript{BrowserScript: &configV1.BrowserScript{UrlRegexp: []string{"foo"}}}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMask := new(commonsV1.FieldMask)
			gotTemplate := tt.args.templateObj

			err := CreateTemplateFilter(tt.args.filterString, gotTemplate, gotMask)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplateFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantMask != nil && !proto.Equal(gotMask, tt.wantMask) {
				t.Errorf("CreateTemplateFilter() gotMask = %v, wantMask %v", gotMask, tt.wantMask)
			}
			if !proto.Equal(gotTemplate, tt.wantTemplate) {
				t.Errorf("CreateTemplateFilter() gotTemplate = %v, wantTemplate %v", gotTemplate, tt.wantTemplate)
			}
		})
	}
}
