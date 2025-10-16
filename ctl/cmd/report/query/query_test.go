package query

import (
	"testing"

	reportV1 "github.com/NationalLibraryOfNorway/veidemann/api/report"
	"github.com/stretchr/testify/assert"
)

func Test_queryCmdOptions_parseQuery(t *testing.T) {
	tests := []struct {
		name            string
		fieldsFromFlags *options
		args            []string
		want            *query
		wantErr         assert.ErrorAssertionFunc
	}{
		{"template file",
			&options{},
			[]string{"testdata/template1.yaml"},
			&query{
				Name:        "template1",
				Description: "Example template\n",
				Query:       "r.db('veidemann').table('config_crawl_entities')\n",
				Template:    "{{.id}} {{.meta.name}}\n",
				queryOrFile: "testdata/template1.yaml",
				queryArgs:   make([]any, 0),
				request: &reportV1.ExecuteDbQueryRequest{
					Query: "r.db('veidemann').table('config_crawl_entities')\n",
				},
				opts: &options{
					format: "template",
				},
			},
			assert.NoError},
		{"template file with template flag",
			&options{
				goTemplate: "{{.id}}",
			},
			[]string{"testdata/template1.yaml"},
			&query{
				Name:        "template1",
				Description: "Example template\n",
				Query:       "r.db('veidemann').table('config_crawl_entities')\n",
				Template:    "{{.id}}",
				queryOrFile: "testdata/template1.yaml",
				queryArgs:   make([]any, 0),
				request: &reportV1.ExecuteDbQueryRequest{
					Query: "r.db('veidemann').table('config_crawl_entities')\n",
				},
				opts: &options{
					goTemplate: "{{.id}}",
					format:     "template",
				},
			},
			assert.NoError},
		{"template file with format flag",
			&options{
				format: "yaml",
			},
			[]string{"testdata/template1.yaml"},
			&query{
				Name:        "template1",
				Description: "Example template\n",
				Query:       "r.db('veidemann').table('config_crawl_entities')\n",
				Template:    "{{.id}} {{.meta.name}}\n",
				queryOrFile: "testdata/template1.yaml",
				queryArgs:   make([]any, 0),
				request: &reportV1.ExecuteDbQueryRequest{
					Query: "r.db('veidemann').table('config_crawl_entities')\n",
				},
				opts: &options{
					format: "yaml",
				},
			},
			assert.NoError},
		{"nonexisting template file",
			&options{},
			[]string{"missing.yaml"},
			nil,
			assert.Error},
		{"query",
			&options{},
			[]string{"r.db('veidemann').table('config_crawl_entities')"},
			&query{
				Query:       "r.db('veidemann').table('config_crawl_entities')",
				queryOrFile: "r.db('veidemann').table('config_crawl_entities')",
				queryArgs:   make([]any, 0),
				request: &reportV1.ExecuteDbQueryRequest{
					Query: "r.db('veidemann').table('config_crawl_entities')",
				},
				opts: &options{
					format: "json",
				},
			},
			assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.fieldsFromFlags
			got, err := o.parseQuery(tt.args)
			if !tt.wantErr(t, err, "parseQuery()") {
				return
			}
			assert.Equalf(t, tt.want, got, "parseQuery()")
		})
	}
}
