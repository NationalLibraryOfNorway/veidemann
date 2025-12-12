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

package query

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	reportV1 "github.com/NationalLibraryOfNorway/veidemann/api/report/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/invopop/yaml"
	"github.com/rs/zerolog/log"
)

type options struct {
	pageSize   int32
	page       int32
	goTemplate string
	file       string
	format     string
}

func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "query (QUERY-STRING | FILENAME) [ARGS ...]",
		Short: "Run a database query",
		Long:  `Run a database query. The query should be a JavaScript string like the ones used by RethinkDB JavaScript driver.`,
		Args:  cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			d, err := config.GetConfigPath("query")
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			q := listStoredQueries(d)
			var names []string
			for _, s := range q {
				if strings.HasPrefix(s.Name, toComplete) {
					names = append(names, s.Name+"\t"+s.Description)
				}
			}
			return names, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := o.parseQuery(args)
			if err != nil {
				return fmt.Errorf("failed to parse query: %w", err)
			}

			// set silence usage to true to avoid printing usage when an error occurs
			cmd.SilenceUsage = true

			return q.run()
		},
	}

	cmd.Flags().Int32VarP(&o.pageSize, "pagesize", "s", 10, "Number of objects to get")
	cmd.Flags().Int32VarP(&o.page, "page", "p", 0, "The page number")
	cmd.Flags().StringVarP(&o.format, "output", "o", "", "Output format (json|yaml|template|template-file) (default \"json\")")
	cmd.Flags().StringVarP(&o.goTemplate, "template", "t", "", "A Go template used to format the output")
	cmd.Flags().StringVarP(&o.file, "filename", "f", "", "Filename to write to")

	return cmd
}

// query is a struct for holding query definitions
type query struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Query       string `json:"query"`
	Template    string `json:"template"`

	opts        *options
	queryOrFile string
	queryArgs   []any
	request     *reportV1.ExecuteDbQueryRequest
}

// run runs the query command
func (q *query) run() error {
	log.Debug().Msgf("Executing query: %s", q.request.GetQuery())

	var w io.Writer
	var err error

	if q.opts.file == "" || q.opts.file == "-" {
		w = os.Stdout
	} else {
		w, err = os.Create(q.opts.file)
		if err != nil {
			return fmt.Errorf("failed to create file: %s: %w", q.opts.file, err)
		}
	}

	var formatter format.Formatter
	formatter, err = format.NewFormatter("", w, q.opts.format, q.Template)
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}

	conn, err := connection.Connect()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	client := reportV1.NewReportClient(conn)

	stream, err := client.ExecuteDbQuery(context.Background(), q.request)
	if err != nil {
		return fmt.Errorf("failed executing query: %w", err)
	}

	for {
		value, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to receive a value: %w", err)
		}
		if err := formatter.WriteRecord(value.GetRecord()); err != nil {
			return err
		}
	}

	return nil
}

func (o *options) parseQuery(args []string) (*query, error) {
	q := &query{
		queryOrFile: args[0],
		queryArgs:   make([]any, len(args[1:])),
		opts:        o,
	}
	for i, arg := range args[1:] {
		q.queryArgs[i] = arg
	}

	if strings.HasPrefix(q.queryOrFile, "r.") {
		q.Query = q.queryOrFile
	} else {
		filename, err := findQueryFile(q.queryOrFile)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("Using query definition from file '%s'", filename)
		err = readQuery(filename, q)
		if err != nil {
			return nil, err
		}
		if o.goTemplate != "" {
			q.Template = o.goTemplate
		}
	}
	if o.format == "" {
		if q.Template != "" {
			o.format = "template"
		} else {
			o.format = "json"
		}
	}

	q.request = &reportV1.ExecuteDbQueryRequest{
		Query: fmt.Sprintf(q.Query, q.queryArgs...),
		Limit: o.pageSize,
	}

	return q, nil
}

// findQueryFile finds a query file
func findQueryFile(name string) (string, error) {
	filename := name
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return filename, nil
	}

	queryDir, err := config.GetConfigPath("query")
	if err != nil {
		return "", fmt.Errorf("failed to resolve query dir: %w", err)
	}

	filename = filepath.Join(queryDir, name)
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return filename, nil
	}
	filename = filepath.Join(queryDir, name) + ".yml"
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return filename, nil
	}
	filename = filepath.Join(queryDir, name) + ".yaml"
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		return filename, nil
	}
	return "", fmt.Errorf("query not found: %s", name)
}

// readQuery reads a query definition from a file
func readQuery(name string, q *query) error {
	data, err := os.ReadFile(name)
	if err != nil {
		return fmt.Errorf("failed to read file: %s: %w", name, err)
	}
	// Found file
	if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
		err := yaml.Unmarshal(data, q)
		if err != nil {
			return err
		}
	} else {
		q.Query = string(data)
	}
	q.Name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))

	return nil
}

// listStoredQueries returns a list of query definitions stored in the query directory
func listStoredQueries(path string) []*query {
	var r []*query

	if files, err := os.ReadDir(path); err == nil {
		for _, f := range files {
			if !f.IsDir() {
				var q query
				err := readQuery(filepath.Join(path, f.Name()), &q)
				if err != nil {
					return nil
				}
				r = append(r, &q)
			}
		}
	}

	return r
}
