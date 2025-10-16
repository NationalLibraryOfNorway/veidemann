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

package config

import (
	"fmt"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/spf13/cobra"
)

// createContextCmd represents the create-context command
func newCreateContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create-context NAME",
		Short: "Create a new context",
		Long: `Create a new context

Examples:
  # Create context for the prod cluster
  veidemannctl config create-context prod
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to avoid printing usage when returning an error
			cmd.SilenceUsage = true

			name := args[0]
			err := config.CreateContext(name)
			if err != nil {
				return err
			}
			fmt.Printf("Created context \"%v\"\n", name)

			return nil
		},
	}
}
