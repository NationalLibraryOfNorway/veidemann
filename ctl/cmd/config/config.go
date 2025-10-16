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
	"github.com/spf13/cobra"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Modify or view configuration files",
	}

	cmd.AddCommand(newUseContextCmd())            // use-context
	cmd.AddCommand(newCreateContextCmd())         // create-context
	cmd.AddCommand(newCurrentContextCmd())        // current-context
	cmd.AddCommand(newListContextsCmd())          // list-contexts
	cmd.AddCommand(newImportCaCmd())              // import-ca
	cmd.AddCommand(newSetServerNameOverrideCmd()) // set-server-name-override
	cmd.AddCommand(newGetServerNameOverrideCmd()) // get-server-name-override
	cmd.AddCommand(newSetApiKeyCmd())             // set-apikey
	cmd.AddCommand(newGetApiKeyCmd())             // get-apikey
	cmd.AddCommand(newSetAddressCmd())            // set-address
	cmd.AddCommand(newGetAddressCmd())            // get-address
	cmd.AddCommand(newViewConfigCmd())            // view

	return cmd
}
