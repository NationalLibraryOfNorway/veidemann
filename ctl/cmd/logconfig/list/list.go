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

package list

import (
	"context"
	"fmt"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/spf13/cobra"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured loggers",
		Long:  `List configured loggers.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, err := connection.Connect()
			if err != nil {
				return fmt.Errorf("failed to connect: %w", err)
			}
			defer func() { _ = conn.Close() }()

			client := configV1.NewConfigClient(conn)

			r, err := client.GetLogConfig(context.Background(), &empty.Empty{})
			if err != nil {
				return fmt.Errorf("could not get log config: %w", err)
			}

			fmt.Printf("%-45s %s\n", "LOGGER", "LEVEL")
			for _, l := range r.LogLevel {
				fmt.Printf("%-45s %s\n", l.Logger, l.Level)
			}
			return nil
		},
	}
}
