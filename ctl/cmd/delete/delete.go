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

package del

import (
	"context"
	"errors"
	"fmt"
	"io"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/apiutil"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type options struct {
	kind    configV1.Kind
	ids     []string
	label   string
	filters []string
	dryRun  bool
}

func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		GroupID: "basic",
		Use:     "delete KIND ID ...",
		Short:   "Delete config objects",
		Long: `Delete one or many config objects.

` +
			format.ListObjectNames() +
			`Examples:
  # Delete a seed.
  veidemannctl delete seed 407a9600-4f25-4f17-8cff-ee1b8ee950f6`,
		Args: cobra.MatchAll(
			cobra.MinimumNArgs(1),
			func(cmd *cobra.Command, args []string) error {
				return cobra.OnlyValidArgs(cmd, args[:1])
			}),
		ValidArgs: format.GetObjectNames(),
		RunE: func(cmd *cobra.Command, args []string) error {
			k := args[0]

			o.kind = format.GetKind(k)
			o.ids = args[1:]

			if o.kind == configV1.Kind_undefined {
				return fmt.Errorf("undefined kind '%s'", k)
			}

			if len(o.ids) == 0 && o.filters == nil && o.label == "" {
				return fmt.Errorf("Either the -f or -q flags, or one or more id's must be provided\n")
			}

			// set silence usage to true to avoid printing usage when an error occurs
			cmd.SilenceUsage = true

			return run(o)
		},
	}
	cmd.PersistentFlags().StringVarP(&o.label, "label", "l", "", "Delete objects by label {TYPE:VALUE | VALUE}")
	cmd.PersistentFlags().StringArrayVarP(&o.filters, "filter", "q", nil, "Delete objects by field (i.e. meta.description=foo)")
	cmd.PersistentFlags().BoolVarP(&o.dryRun, "dry-run", "", true, "Set to false to execute delete")

	return cmd
}

// run runs the delete command which deletes one or more config objects
func run(o *options) error {
	conn, err := connection.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := configV1.NewConfigClient(conn)

	selector, err := apiutil.CreateListRequest(o.kind, o.ids, "", o.label, o.filters, 0, 0)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	r, err := client.ListConfigObjects(context.Background(), selector)
	if err != nil {
		return fmt.Errorf("could not list objects: %w", err)
	}

	count, err := client.CountConfigObjects(context.Background(), selector)
	if err != nil {
		return fmt.Errorf("could not count objects: %w", err)
	}

	if o.dryRun {
		for {
			msg, err := r.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return fmt.Errorf("error getting object: %w", err)
			}
			log.Debug().Msgf("Outputing record of kind '%s' with name '%s'", msg.Kind, msg.Meta.Name)
			fmt.Printf("%s\n", msg.Meta.Name)
		}
		fmt.Printf("Requested count: %v\nTo actually delete, add: --dry-run=false\n", count.Count)

		return nil
	}

	var deleted int
	for {
		msg, err := r.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("error getting object: %w", err)
		}
		log.Debug().Msgf("Deleting record of kind '%s' with name '%s'", msg.Kind, msg.Meta.Name)

		request := &configV1.ConfigObject{
			ApiVersion: "v1",
			Kind:       o.kind,
			Id:         msg.Id,
		}

		r, err := client.DeleteConfigObject(context.Background(), request)
		if err != nil {
			log.Error().Err(err).Str("id", msg.Id).Msgf("Could not delete object")
			continue
		}
		if r.Deleted {
			deleted++
		}
	}
	log.Info().Msgf("Deleted %d objects of %d selected", deleted, count.Count)

	return nil
}
