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
	"os"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/spf13/cobra"
)

func newImportCaCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "import-ca FILENAME",
		Short:   "Import file with trusted certificate chains for the idp and controller.",
		Long:    `Import file with trusted certificate chains for the idp and controller. These are in addition to the default certs configured for the OS.`,
		Aliases: []string{"ca"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to avoid printing usage when returning an error
			cmd.SilenceUsage = true
			rootCA := args[0]
			rootCABytes, err := os.ReadFile(rootCA)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			if err := config.SetCaCert(string(rootCABytes)); err != nil {
				return err
			}
			fmt.Printf("Successfully imported root CA cert from file %s\n", rootCA)
			return nil
		},
	}
}
