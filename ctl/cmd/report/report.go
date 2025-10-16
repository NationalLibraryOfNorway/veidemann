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

package report

import (
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/report/crawlexecution"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/report/crawllog"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/report/jobexecution"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/report/pagelog"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/report/query"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		GroupID: "advanced",
		Use:     "report",
		Short:   "Request a report",
	}

	cmd.AddCommand(jobexecution.NewCmd())   // jobexecution
	cmd.AddCommand(crawlexecution.NewCmd()) // crawlexecution
	cmd.AddCommand(crawllog.NewCmd())       // crawllog
	cmd.AddCommand(query.NewCmd())          // query
	cmd.AddCommand(pagelog.NewCmd())        // pagelog

	return cmd
}
