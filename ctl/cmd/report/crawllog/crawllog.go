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

package crawllog

import (
	"context"
	"errors"
	"fmt"
	"io"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/rs/zerolog/log"
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
		Use:   "crawllog [ID ...]",
		Short: "View crawl log",
		Long:  `View crawl log.`,
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
	conn, err := connection.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := logV1.NewLogClient(conn)

	request, err := createCrawlLogListRequest(o)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	r, err := client.ListCrawlLogs(context.Background(), request)
	if err != nil {
		return fmt.Errorf("error from controller: %w", err)
	}

	out, err := format.ResolveWriter(o.file)
	if err != nil {
		return fmt.Errorf("unable to open output file: %v: %w", o.file, err)
	}
	s, err := format.NewFormatter("CrawlLog", out, o.format, o.goTemplate)
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
		log.Debug().Msgf("Outputting crawl log record with WARC id '%s'", msg.WarcId)
		if err := s.WriteRecord(msg); err != nil {
			return err
		}
	}
	return nil
}

func createCrawlLogListRequest(o *options) (*logV1.CrawlLogListRequest, error) {
	request := &logV1.CrawlLogListRequest{}
	request.WarcId = o.ids
	request.Offset = o.page
	request.PageSize = o.pageSize

	if o.executionId != "" {
		queryMask := new(commonsV1.FieldMask)
		queryMask.Paths = append(queryMask.Paths, "executionId")
		queryTemplate := new(logV1.CrawlLog)
		queryTemplate.ExecutionId = o.executionId

		request.QueryMask = queryMask
		request.QueryTemplate = queryTemplate
	}

	return request, nil
}
