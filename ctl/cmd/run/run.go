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

package run

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
		Use:     "run JOB-ID [SEED-ID]",
		Short:   "Run a crawl job",
		Long: `Run a crawl job.
		
If a seed is provided, the job will be created and started with the seed only.
If the job is already running, the seed will be added to the running job.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			conn, err := connection.Connect()
			if err != nil {
				return fmt.Errorf("failed to connect")
			}
			defer conn.Close()

			client := controllerV1.NewControllerClient(conn)

			switch len(args) {
			case 1: // One argument (only jobId)
				request := controllerV1.RunCrawlRequest{JobId: args[0]}
				r, err := client.RunCrawl(context.Background(), &request)
				if err != nil {
					return fmt.Errorf("could not run job: %w", err)
				}

				fmt.Printf("Job Execution ID: %v\n", r.GetJobExecutionId())
			case 2: // Two arguments (jobId and seedId)
				request := controllerV1.RunCrawlRequest{JobId: args[0], SeedId: args[1]}
				r, err := client.RunCrawl(context.Background(), &request)
				if err != nil {
					return fmt.Errorf("could not run job: %w", err)
				}

				fmt.Printf("Job Execution ID: %v\n", r.GetJobExecutionId())
			}
			return nil
		},
	}
}
