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

package abort

import (
	"context"
	"fmt"

	controllerV1 "github.com/NationalLibraryOfNorway/veidemann/api/controller"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		GroupID: "run",
		Use:     "abort CRAWL-EXECUTION-ID ...",
		Short:   "Abort crawl executions",
		Long:    `Abort one or more crawl executions.`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to avoid showing usage when an error occurs
			cmd.SilenceUsage = true

			conn, err := connection.Connect()
			if err != nil {
				return err
			}
			defer conn.Close()

			client := controllerV1.NewControllerClient(conn)

			// ceids is a list of crawl execution ids to abort
			for _, ceid := range args {
				request := controllerV1.ExecutionId{Id: ceid}
				_, err := client.AbortCrawlExecution(context.Background(), &request)
				if err != nil {
					return fmt.Errorf("failed to abort execution '%v': %w", ceid, err)
				}
			}
			return nil
		},
	}
}
