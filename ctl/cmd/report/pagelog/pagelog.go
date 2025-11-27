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

package pagelog

import (
	"context"
	"errors"
	"fmt"
	"io"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/spf13/cobra"
)

type options struct {
	ids         []string
	executionId string
	pageSize    int32
	page        int32
	goTemplate  string
	format      string
	file        string
}

func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "pagelog [ID ...]",
		Short: "View page log",
		Long:  `View page log.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ids = args
			if len(o.ids) == 0 && o.executionId == "" {
				return fmt.Errorf("request must provide either warcId or executionId")
			}

			// set silence usage to true to avoid printing usage when an error occurs
			cmd.SilenceUsage = true
			return run(o)
		},
	}

	cmd.Flags().Int32VarP(&o.pageSize, "pagesize", "s", 10, "Number of objects to get")
	cmd.Flags().Int32VarP(&o.page, "page", "p", 0, "The page number")
	cmd.Flags().StringVarP(&o.format, "output", "o", "table", "Output format (table|wide|json|yaml|template|template-file)")
	cmd.Flags().StringVarP(&o.goTemplate, "template", "t", "", "A Go template used to format the output")
	cmd.Flags().StringVarP(&o.file, "filename", "f", "", "Filename to write to")
	cmd.Flags().StringVar(&o.executionId, "execution-id", "", "Execution ID")

	return cmd
}

func run(o *options) error {
	// initialize output writer
	out, err := format.ResolveWriter(o.file)
	if err != nil {
		return fmt.Errorf("error opening output file: %w", err)
	}
	s, err := format.NewFormatter("PageLog", out, o.format, o.goTemplate)
	if err != nil {
		return err
	}
	defer s.Close()

	// connect to grpc server
	conn, err := connection.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := logV1.NewLogClient(conn)

	// create request
	request := createPageLogListRequest(o)

	r, err := client.ListPageLogs(context.Background(), request)
	if err != nil {
		return fmt.Errorf("could not get page log: %w", err)
	}

	for {
		msg, err := r.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error getting object: %w", err)
		}
		if err := s.WriteRecord(msg); err != nil {
			return err
		}
	}
	return nil
}

func createPageLogListRequest(o *options) *logV1.PageLogListRequest {
	request := &logV1.PageLogListRequest{}
	request.WarcId = o.ids
	request.Offset = o.page
	request.PageSize = o.pageSize

	if o.executionId != "" {
		queryMask := new(commonsV1.FieldMask)
		queryMask.Paths = append(queryMask.Paths, "executionId")
		queryTemplate := new(logV1.PageLog)
		queryTemplate.ExecutionId = o.executionId

		request.QueryMask = queryMask
		request.QueryTemplate = queryTemplate
	}

	return request
}
