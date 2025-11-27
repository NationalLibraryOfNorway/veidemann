// Copyright © 2017 National Library of Norway.
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

package activeroles

import (
	"context"
	"fmt"

	controllerV1 "github.com/NationalLibraryOfNorway/veidemann/api/controller/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/spf13/cobra"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		GroupID:      "debug",
		Use:          "activeroles",
		Short:        "Get the active roles for the currently logged in user",
		Long:         `Get the active roles for the currently logged in user.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, err := connection.Connect()
			if err != nil {
				return err
			}
			defer conn.Close()

			client := controllerV1.NewControllerClient(conn)

			r, err := client.GetRolesForActiveUser(context.Background(), &empty.Empty{})
			if err != nil {
				return err
			}

			for _, role := range r.Role {
				fmt.Println(role)
			}
			return nil
		},
	}
}
