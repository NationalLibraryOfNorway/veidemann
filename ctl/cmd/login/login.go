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

package login

import (
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	var manualLogin bool

	var cmd = &cobra.Command{
		GroupID:      "login",
		Use:          "login",
		Short:        "Log in to Veidemann",
		Long:         ``,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return connection.Login(manualLogin)
		},
	}

	cmd.Flags().BoolVarP(&manualLogin, "manual", "m", false,
		"Manually copy and paste login url and code. Used to log in from a remote terminal.")

	return cmd
}
