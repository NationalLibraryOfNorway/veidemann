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

package create

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/spf13/cobra"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type options struct {
	filename    string
	concurrency int
}

func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		GroupID: "basic",
		Use:     "create",
		Short:   "Create or update config objects",
		Long:    `Create or update one or many config objects`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to prevent printing usage when an error occurs
			cmd.SilenceUsage = true

			return run(o)
		},
	}

	// filename is a required flag
	cmd.Flags().StringVarP(&o.filename, "filename", "f", "", "Filename or directory to read from. "+
		"If input is a directory, all files ending in .yaml or .json will be tried. An input of '-' will read from stdin.")
	_ = cmd.MarkFlagRequired("filename")
	cmd.Flags().IntVarP(&o.concurrency, "concurrency", "c", 32, "Number of concurrent requests")

	return cmd
}

func run(o *options) error {
	if o.filename == "-" {
		o.filename = ""
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigs := make(chan os.Signal, 2)
		signal.Notify(sigs, os.Interrupt)
		sig := <-sigs
		log.Debug().Msgf("Received %s signal, aborting...", sig)
		cancel()
	}()

	result := make(chan *configV1.ConfigObject, 256)
	err := format.Unmarshal(ctx, o.filename, result)
	if err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}

	conn, err := connection.Connect()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	client := configV1.NewConfigClient(conn)

	validate := func(co *configV1.ConfigObject) error {
		if co.ApiVersion == "" {
			return fmt.Errorf("missing apiVersion")
		}
		if co.Kind == configV1.Kind_undefined {
			return fmt.Errorf("missing kind")
		}
		if co.GetMeta().GetName() == "" {
			return fmt.Errorf("missing metadata.name")
		}
		return nil
	}

	handleError := func(co *configV1.ConfigObject, err error) {
		log.Error().Err(err).
			Str("kind", co.GetKind().String()).
			Str("meta.name", co.GetMeta().Name).
			Str("id", co.GetId()).
			Msg("Failed to save config object")
	}

	var wg sync.WaitGroup
	for i := 0; i < o.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for co := range result {
				// validate
				err = validate(co)
				if err != nil {
					handleError(co, fmt.Errorf("validation failed: %w", err))
					continue
				}
				// save
				for attempts := 0; attempts < 3; attempts++ {
					r, err := client.SaveConfigObject(context.Background(), co)
					s, ok := status.FromError(err)
					if ok && s.Code() == codes.Unauthenticated {
						// retry if unauthenticated
						log.Debug().Int("attempts", attempts).Msg("Unauthenticated, retrying...")
						continue
					}
					if err != nil {
						handleError(co, err)
						break
					}
					log.Info().Str("kind", r.GetKind().String()).Str("meta.name", r.Meta.Name).Str("id", r.Id).Msg("Saved config object")
					break
				}
			}
		}()
	}
	wg.Wait()

	return nil
}
