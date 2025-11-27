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

package seeds

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/spf13/cobra"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/importutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type options struct {
	Toplevel        bool
	IgnoreScheme    bool
	CheckUri        bool
	Truncate        bool
	DryRun          bool
	SkipImport      bool
	CheckUriTimeout time.Duration
	Filename        string
	ErrorFile       string
	CrawlJobId      string
	DbDir           string
	Concurrency     int
}

func NewCmd() *cobra.Command {
	o := &options{}

	// cmd represents the import command
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Import seeds",
		Long: `Import new seeds and entities from a line oriented JSON file on the following format: 

{"entityName":"foo","uri":"https://www.example.com/","entityDescription":"desc","entityLabel":[{"key":"foo","value":"bar"}],"seedLabel":[{"key":"foo","value":"bar"},{"key":"foo2","value":"bar"}],"seedDescription":"foo"}

Every record must be formatted on a single line.


`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(o)
		},
	}

	// filename is required
	cmd.Flags().StringVarP(&o.Filename, "filename", "f", "", "Filename or directory to read from. "+
		"If input is a directory, all files ending in .yaml or .json will be tried. An input of '-' will read from stdin.")
	_ = cmd.MarkFlagRequired("filename")
	cmd.Flags().StringVarP(&o.ErrorFile, "err-file", "e", "-", "File to write errors to. \"-\" writes to stderr")
	cmd.Flags().BoolVarP(&o.Toplevel, "toplevel", "", false, "Convert URI by removing path")
	cmd.Flags().BoolVarP(&o.IgnoreScheme, "ignore-scheme", "", true, "Ignore the URL's scheme when checking if this URL is already imported")
	cmd.Flags().BoolVarP(&o.CheckUri, "check-uri", "", false, "Check the uri for liveness and follow permanent redirects")
	cmd.Flags().DurationVarP(&o.CheckUriTimeout, "check-uri-timeout", "", 2*time.Second, "Timeout duration when checking uri for liveness")
	cmd.Flags().StringVarP(&o.CrawlJobId, "crawljob-id", "", "", "Set crawlJob ID for new seeds")
	cmd.Flags().StringVarP(&o.DbDir, "db-dir", "b", "/tmp/veidemannctl", "Directory for storing state db")
	cmd.Flags().BoolVar(&o.Truncate, "truncate", false, "Truncate state database")
	cmd.Flags().BoolVarP(&o.DryRun, "dry-run", "", false, "Run without actually writing anything to Veidemann")
	cmd.Flags().BoolVar(&o.SkipImport, "skip-import", false, "Do not import existing seeds into state database")
	cmd.Flags().IntVarP(&o.Concurrency, "concurrency", "c", 16, "Number of concurrent workers")

	return cmd
}

func run(o *options) error {
	// Create error writer (file or stderr)
	var errFile io.Writer
	if o.ErrorFile == "" || o.ErrorFile == "-" {
		errFile = os.Stderr
	} else {
		f, err := os.Create(o.ErrorFile)
		if err != nil {
			return fmt.Errorf("unable to open error file '%v': %w", o.ErrorFile, err)
		}
		defer f.Close()
		errFile = f
	}

	// Create Veidemann config client
	conn, err := connection.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect %w", err)
	}
	defer conn.Close()
	client := configV1.NewConfigClient(conn)

	// Create/open state database for entities
	entityDbDir := path.Join(o.DbDir, config.GetContext(), configV1.Kind_crawlEntity.String())
	entityDb, err := importutil.NewImportDb(entityDbDir, o.Truncate)
	if err != nil {
		return fmt.Errorf("failed to initialize entity state db: %w", err)
	}
	defer entityDb.Close()

	uriNormalizer := &importutil.UriKeyNormalizer{IgnoreScheme: o.IgnoreScheme, Toplevel: o.Toplevel}

	// Create/open state database for seeds
	seedDbDir := path.Join(o.DbDir, config.GetContext(), configV1.Kind_seed.String())
	seedDb, err := importutil.NewImportDb(seedDbDir, o.Truncate)
	if err != nil {
		return fmt.Errorf("failed to initialize seed state db: %w", err)
	}
	defer seedDb.Close()

	if !o.SkipImport {
		// Import entities
		err = importutil.ImportExisting(entityDb, client, configV1.Kind_crawlEntity, nil)
		if err != nil {
			return fmt.Errorf("failed to import entities: %w", err)
		}

		// Import seeds
		err = importutil.ImportExisting(seedDb, client, configV1.Kind_seed, uriNormalizer)
		if err != nil {
			return fmt.Errorf("failed to import seeds: %w", err)
		}
	}

	// Create Record reader for file input
	rr, err := importutil.NewRecordReader(o.Filename, &importutil.JsonYamlDecoder{}, "*.json")
	if err != nil {
		return fmt.Errorf("failed to initialize reader: %w", err)
	}

	var uriChecker *importutil.UriChecker
	if o.CheckUri {
		uriChecker = &importutil.UriChecker{
			Client: importutil.NewHttpClient(o.CheckUriTimeout, false),
		}
	}

	crawlJobRef := []*configV1.ConfigRef{{Kind: configV1.Kind_crawlJob, Id: o.CrawlJobId}}

	var m sync.Mutex

	// Create error logger
	errorLog := log.Output(zerolog.ConsoleWriter{Out: errFile, TimeFormat: time.RFC3339})

	// Create processor function for each record in input file
	proc := func(sd *importutil.SeedDesc) error {
		if o.CrawlJobId != "" {
			sd.CrawlJobRef = crawlJobRef
		}

		if uriChecker != nil {
			// check liveness of uri and follow permanent redirects
			uri, err := uriChecker.Check(sd.Uri)
			if err != nil {
				return err
			}
			sd.Uri = uri
		}

		normalizedUri, err := uriNormalizer.Normalize(sd.Uri)
		if err != nil {
			return fmt.Errorf("failed to normalize URL '%s': %w", sd.Uri, err)
		}

		// Ensure every concurrent process can see the same state by locking
		// while checking and updating state database and Veidemann.
		m.Lock()
		defer m.Unlock()

		// Check if seed already exists in state database
		seedIds, err := seedDb.Get(normalizedUri)
		if err != nil {
			return err
		} else if len(seedIds) > 0 {
			return importutil.ErrAlreadyExists(normalizedUri)
		}

		if o.DryRun {
			return nil
		}

		var entity *configV1.ConfigObject

		if sd.EntityId == "" {
			entityIds, err := entityDb.Get(sd.EntityName)
			if err != nil {
				return fmt.Errorf("failed to get entity: %w", err)
			}

			if len(entityIds) > 0 {
				sd.EntityId = entityIds[0]
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				entity, err = client.SaveConfigObject(ctx, sd.ToEntity())
				if err != nil {
					return fmt.Errorf("failed to create entity in Veidemann: %w", err)
				}

				_, _, err := entityDb.Set(sd.EntityName, entity.Id)
				if err != nil {
					return fmt.Errorf("failed to save new entity to import db: %w", err)
				}

				sd.EntityId = entity.Id
				errorLog.Info().Str("entityId", entity.Id).Str("entityName", entity.Meta.Name).Msg("Created new entity in Veidemann")
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		seed, err := client.SaveConfigObject(ctx, sd.ToSeed())
		if err != nil {
			if entity != nil { // Delete created entity if seed creation failed
				if _, err := client.DeleteConfigObject(ctx, entity); err != nil {
					errorLog.Error().Err(err).
						Str("uri", sd.Uri).
						Str("entityId", entity.Id).
						Str("entityName", entity.Meta.Name).
						Msg("Failed to delete new entity from Veidemann after seed creation error")
				}
			}
			return fmt.Errorf("failed to create seed in Veidemann: %w", err)
		}
		errorLog.Info().Str("key", normalizedUri).Str("seedId", seed.Id).Str("uri", sd.Uri).Msg("Created new seed in Veidemann")

		_, _, err = seedDb.Set(normalizedUri, seed.Id)
		if err != nil {
			return fmt.Errorf("failed to save new seed to import db: %w", err)
		}

		return nil
	}

	errHandler := func(state importutil.Job[*importutil.SeedDesc]) {
		l := errorLog.With().
			Str("uri", state.Val.Uri).
			Str("filename", state.GetFilename()).
			Int("recNum", state.GetRecordNum()).Logger()

		var err importutil.ErrAlreadyExists
		if errors.As(state.GetError(), &err) {
			l.Warn().Msgf("Skipping: %v", err.Error())
		} else {
			l.Error().Err(state.GetError()).Msg("")
		}
	}

	executor := importutil.NewExecutor(o.Concurrency, proc, errHandler)

	// Process each record in input file and add to import db if not already present
	for {
		var sd importutil.SeedDesc
		state, err := rr.Next(&sd)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			errorLog.Error().Err(err).Msgf("error decoding record: %v", state)
			continue
		}
		executor.Queue <- importutil.Job[*importutil.SeedDesc]{State: state, Val: &sd}
	}

	count, success, failed := executor.Wait()

	errorLog.Info().Int("processed", count).Int("imported", success).Int("errors", failed).Msg("Import completed")

	return err
}
