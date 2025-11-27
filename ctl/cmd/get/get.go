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

package get

import (
	"errors"
	"fmt"
	"io"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/apiutil"
	"github.com/spf13/cobra"

	"context"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
)

func NewCmd() *cobra.Command {
	o := &opts{}

	cmd := &cobra.Command{
		GroupID: "basic",
		Use:     "get KIND [ID ...]",
		Short:   "Display config objects",
		Long: `Display one or many config objects.

` +
			`Valid object types:
` +
			format.ListObjectNames() +
			`Examples:
  # List all seeds.
  veidemannctl get seed

  # List all seeds in yaml output format.
  veidemannctl get seed -o yaml`,
		ValidArgs: format.GetObjectNames(),
		Args:      cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ids = args[1:]
			o.kind = format.GetKind(args[0])
			if o.kind == configV1.Kind_undefined {
				return fmt.Errorf(`undefined kind "%v"`, args[0])
			}

			// set SilenceUsage to true to prevent printing usage when an error occurs
			cmd.SilenceUsage = true

			return run(o)
		},
	}

	cmd.Flags().StringVarP(&o.label, "label", "l", "", "List objects by label {TYPE:VALUE | VALUE}")
	cmd.Flags().StringVarP(&o.name, "name", "n", "", "List objects by name (accepts regular expressions)")
	_ = cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		names, err := apiutil.CompleteName(args[0], toComplete)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveDefault
	})
	cmd.Flags().StringArrayVarP(&o.filters, "filter", "q", nil, "Filter objects by field (i.e. meta.description=foo)")
	cmd.Flags().StringVarP(&o.format, "output", "o", "table", "Output format (table|wide|json|yaml|template|template-file)")
	_ = cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "table", "yaml", "wide", "template", "template-file"}, cobra.ShellCompDirectiveDefault
	})
	cmd.Flags().StringVarP(&o.goTemplate, "template", "t", "", "A Go template used to format the output")
	cmd.Flags().StringVarP(&o.filename, "filename", "f", "", "Filename to write to")
	cmd.Flags().Int32VarP(&o.pageSize, "pagesize", "s", 10, "Number of objects to get")
	cmd.Flags().Int32VarP(&o.page, "page", "p", 0, "The page number")

	return cmd
}

type opts struct {
	kind       configV1.Kind
	ids        []string
	label      string
	name       string
	filters    []string
	filename   string
	format     string
	goTemplate string
	pageSize   int32
	page       int32
}

func run(o *opts) error {
	request, err := apiutil.CreateListRequest(o.kind, o.ids, o.name, o.label, o.filters, o.pageSize, o.page)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	conn, err := connection.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := configV1.NewConfigClient(conn)

	r, err := client.ListConfigObjects(context.Background(), request)
	if err != nil {
		return fmt.Errorf("error from controller: %w", err)
	}

	out, err := format.ResolveWriter(o.filename)
	if err != nil {
		return fmt.Errorf("could not resolve output file '%v': %w", o.filename, err)
	}

	s, err := format.NewFormatter(o.kind.String(), out, o.format, o.goTemplate)
	if err != nil {
		return err
	}
	defer s.Close()

	for {
		msg, err := r.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if err := s.WriteRecord(msg); err != nil {
			return err
		}
	}
	return nil
}
