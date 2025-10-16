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

func newGetApiKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-apikey",
		Short: "Display Veidemann authentication api-key",
		Long: `Display Veidemann authentication api-key

Examples:
  # Display Veidemann authentication api-key
  veidemannctl config get-apikey
`,
		Aliases: []string{"get-apikey", "apikey", "get-api-key", "api-key"},
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.GetApiKey())
		},
	}
}
