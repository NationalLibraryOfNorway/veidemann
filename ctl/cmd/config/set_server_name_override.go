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
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/spf13/cobra"
)

func newSetServerNameOverrideCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-server-name-override HOSTNAME",
		Short: "Set the server name override",
		Long: `Set the server name override.

Use this when there is a mismatch between the exposed server name for the cluster and the certificate. The use is a security
risk and is only recommended for testing.

Examples:
  # Sets the server name override to test.local
  veidemannctl config set-server-name-override test.local
`,
		Aliases: []string{"set-servername"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to avoid printing usage when returning an error
			cmd.SilenceUsage = true

			return config.SetServerNameOverride(args[0])
		},
	}
}
