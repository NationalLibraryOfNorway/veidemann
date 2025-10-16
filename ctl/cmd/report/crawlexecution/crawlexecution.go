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

package crawlexecution

import (
	"context"
	"errors"
	"fmt"
	"io"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier"
	reportV1 "github.com/NationalLibraryOfNorway/veidemann/api/report"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/apiutil"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// crawlExecCmdOptions holds the crawl execution command line options
type options struct {
	ids         []string
	filters     []string
	pageSize    int32
	page        int32
	goTemplate  string
	format      string
	file        string
	orderByPath string
	orderDesc   bool
	from        *timestamppb.Timestamp
	to          *timestamppb.Timestamp
	watch       bool
	states      []string
}

// NewCmd creates the crawl execution command
func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "crawlexecution [ID ...]",
		Short: "Get current status for crawl executions",
		Long:  `Get current status for crawl executions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ids = args

			f := cmd.Flag("from")
			if f.Changed {
				o.from = timestamppb.New(cast.ToTime(f.Value.String()))
			}

			f = cmd.Flag("to")
			if f.Changed {
				o.to = timestamppb.New(cast.ToTime(f.Value.String()))
			}

			cmd.SilenceUsage = true

			return run(o)
		},
	}
	cmd.Flags().Int32VarP(&o.pageSize, "pagesize", "s", 10, "Number of objects to get")
	cmd.Flags().Int32VarP(&o.page, "page", "p", 0, "The page number")
	cmd.Flags().StringVarP(&o.format, "output", "o", "table", "Output format (table|wide|json|yaml|template|template-file)")
	cmd.Flags().StringVarP(&o.goTemplate, "template", "t", "", "A Go template used to format the output")
	cmd.Flags().StringSliceVarP(&o.filters, "filter", "q", nil, "Filter objects by field (i.e. meta.description=foo)")
	cmd.Flags().StringSliceVar(&o.states, "state", nil, "Filter objects by state. Valid states are UNDEFINED, FETCHING, SLEEPING, FINISHED or FAILED")
	cmd.Flags().StringVarP(&o.file, "filename", "f", "", "Filename to write to")
	cmd.Flags().StringVar(&o.orderByPath, "order-by", "", "Order by path")
	cmd.Flags().String("to", "", "To start time")
	cmd.Flags().String("from", "", "From start time")
	cmd.Flags().BoolVar(&o.orderDesc, "desc", false, "Order descending")
	cmd.Flags().BoolVarP(&o.watch, "watch", "w", false, "Get a continous stream of changes")

	return cmd
}

// run runs the crawl execution command
func run(o *options) error {
	conn, err := connection.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := reportV1.NewReportClient(conn)

	request, err := createCrawlExecutionsListRequest(o)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	r, err := client.ListExecutions(context.Background(), request)
	if err != nil {
		return fmt.Errorf("failed to list crawl executions: %w", err)
	}
	out, err := format.ResolveWriter(o.file)
	if err != nil {
		return fmt.Errorf("unable to open output file: %v: %w", o.file, err)
	}
	s, err := format.NewFormatter("CrawlExecutionStatus", out, o.format, o.goTemplate)
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

// createCrawlExecutionsListRequest creates a crawl execution list request
func createCrawlExecutionsListRequest(o *options) (*reportV1.CrawlExecutionsListRequest, error) {
	request := &reportV1.CrawlExecutionsListRequest{
		Id:              o.ids,
		Watch:           o.watch,
		PageSize:        o.pageSize,
		Offset:          o.page,
		OrderByPath:     o.orderByPath,
		OrderDescending: o.orderDesc,
	}

	if o.watch {
		request.PageSize = 0
	}

	if o.from != nil {
		request.StartTimeFrom = o.from
	}

	if o.to != nil {
		request.StartTimeTo = o.to
	}

	if len(o.states) > 0 {
		for _, state := range o.states {
			if s, ok := frontierV1.CrawlExecutionStatus_State_value[state]; !ok {
				return nil, fmt.Errorf("not a crawlexecution state: %s", state)
			} else {
				request.State = append(request.State, frontierV1.CrawlExecutionStatus_State(s))
			}
		}
	}

	if len(o.filters) > 0 {
		queryTemplate := new(frontierV1.CrawlExecutionStatus)
		queryMask := new(commonsV1.FieldMask)

		for _, filter := range o.filters {
			err := apiutil.CreateTemplateFilter(filter, queryTemplate, queryMask)
			if err != nil {
				return nil, err
			}
		}

		request.QueryMask = queryMask
		request.QueryTemplate = queryTemplate
	}

	return request, nil
}
