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

func newListContextsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-contexts",
		Short: "Display the known contexts",
		Long: `Display the known contexts.

Examples:
  # Get a list of known contexts
  veidemannctl config list-contexts
`,
		Aliases: []string{"contexts"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to avoid printing usage when returning an error
			cmd.SilenceUsage = true

			cs, err := config.ListContexts()
			if err != nil {
				return fmt.Errorf("failed to list contexts: %w", err)
			}
			for _, c := range cs {
				fmt.Println(c)
			}
			return nil
		},
	}
}
