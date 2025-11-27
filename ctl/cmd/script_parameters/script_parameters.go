// Copyright © 2020 National Library of Norway
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

package scriptparameters

import (
	"context"
	"fmt"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	// scriptParametersCmd represents the script-parameters command
	return &cobra.Command{
		GroupID: "debug",
		Use:     "script-parameters JOB-ID [SEED-ID]",
		Short:   "Get the effective script parameters for a crawl job",
		Long: `Get the effective script parameters for a crawl job and optionally a seed in the context of the crawl job.

Examples:
  # See active script parameters for a Crawl Job
  veidemannctl script-parameters 5604f0cc-315d-4091-8d6e-1b17a7eb990b

  # Get effective script parameters for a Seed in the context of a Crawl Job
  veidemannctl script-parameters 5604f0cc-315d-4091-8d6e-1b17a7eb990b 9f89ca44-afe0-4f8f-808f-9df1a0fe64c9
`,
		Aliases: []string{"params"},
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to prevent printing usage when an error occurs
			cmd.SilenceUsage = true

			conn, err := connection.Connect()
			if err != nil {
				return fmt.Errorf("failed to connect: %w", err)
			}
			defer conn.Close()

			configClient := configV1.NewConfigClient(conn)

			request := &configV1.GetScriptAnnotationsRequest{
				Job: &configV1.ConfigRef{Kind: configV1.Kind_crawlJob, Id: args[0]},
			}
			if len(args) == 2 {
				request.Seed = &configV1.ConfigRef{Kind: configV1.Kind_seed, Id: args[1]}
			}

			response, err := configClient.GetScriptAnnotations(context.Background(), request)
			if err != nil {
				return fmt.Errorf("failed getting parameters for %v: %w", args[0], err)
			}

			for _, a := range response.GetAnnotation() {
				fmt.Printf("Param: %s = '%s'\n", a.Key, a.Value)
			}
			return nil
		},
	}
}
