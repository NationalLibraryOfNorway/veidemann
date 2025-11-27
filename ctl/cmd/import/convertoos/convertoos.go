// Copyright © 2023 National Library of Norway
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

package convertoos

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/importutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ConvertOosCmdOptions is the options for the convert oos command
type options struct {
	Filename        string
	ErrorFile       string
	OutFile         string
	Toplevel        bool
	IgnoreScheme    bool
	CheckUri        bool
	CheckUriTimeout time.Duration
	DbDir           string
	ResetDb         bool
	Concurrency     int
	SkipImport      bool
	EntityId        string
	EntityName      string
	EntityLabels    []string
	SeedLabels      []string
}

// NewCmd creates the convert oos command
func NewCmd() *cobra.Command {
	o := &options{}

	var cmd = &cobra.Command{
		Use:   "convertoos",
		Short: "Convert Out of Scope file(s) to seed import file",
		Long:  ``,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(o)
		},
	}

	cmd.Flags().StringVarP(&o.Filename, "filename", "f", "", "Filename or directory to read from. "+
		"If input is a directory, all files ending in .yaml or .json will be tried. An input of '-' will read from stdin.")
	_ = cmd.MarkFlagRequired("filename")
	cmd.Flags().StringVarP(&o.ErrorFile, "err-file", "e", "-", "File to write errors to. '-' writes to stderr.")
	cmd.Flags().StringVarP(&o.OutFile, "out-file", "o", "-", "File to write result to. '-' writes to stdout.")
	cmd.Flags().BoolVar(&o.Toplevel, "toplevel", true, "Convert URI to toplevel by removing path")
	cmd.Flags().BoolVar(&o.IgnoreScheme, "ignore-scheme", true, "Ignore the URL's scheme when checking if this URL is already imported.")
	cmd.Flags().BoolVarP(&o.CheckUri, "check-uri", "", true, "Check the uri for liveness and follow 301")
	cmd.Flags().DurationVarP(&o.CheckUriTimeout, "check-uri-timeout", "", 2*time.Second, "Timeout when checking uri for liveness")
	cmd.Flags().StringVarP(&o.DbDir, "db-dir", "b", "/tmp/veidemannctl", "Directory for storing state db")
	cmd.Flags().BoolVar(&o.ResetDb, "truncate", false, "Truncate state database")
	cmd.Flags().IntVarP(&o.Concurrency, "concurrency", "c", 16, "Number of concurrent workers")
	cmd.Flags().BoolVar(&o.SkipImport, "skip-import", false, "Do not import existing seeds into state database")
	cmd.Flags().StringVar(&o.EntityId, "entity-id", "", "Entity id to use for all seeds (overrides entity-name and entity-label)")
	cmd.Flags().StringVar(&o.EntityName, "entity-name", "", "Entity name to use for all seeds")
	cmd.Flags().StringSliceVar(&o.EntityLabels, "entity-label", []string{"source:oos"}, "Entity labels to use for all seeds")
	cmd.Flags().StringSliceVar(&o.SeedLabels, "seed-label", []string{"source:oos"}, "Seed labels to use for all seeds")

	return cmd
}

// run runs the convert oos command
func run(o *options) error {
	// Create output writer (file or stdout)
	out, err := format.ResolveWriter(o.OutFile)
	if err != nil {
		return fmt.Errorf("unable to open output file: %v: %w", o.OutFile, err)
	}
	defer out.Close()

	// Create error writer (file or stderr)
	var errFile io.WriteCloser
	if o.ErrorFile == "" || o.ErrorFile == "-" {
		errFile = os.Stderr
	} else {
		f, err := os.Create(o.ErrorFile)
		if err != nil {
			return fmt.Errorf("unable to open error file: %v: %w", o.ErrorFile, err)
		}
		defer f.Close()
	}

	// Connect to Veidemann API server
	conn, err := connection.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Create Veidemann config client
	client := configV1.NewConfigClient(conn)

	// Create uri checker for checking liveness of uri
	var uriChecker *importutil.UriChecker
	if o.CheckUri {
		uriChecker = &importutil.UriChecker{
			Client: importutil.NewHttpClient(o.CheckUriTimeout, false),
		}
	}

	// Create key normalizer for state database
	uriNormalizer := &importutil.UriKeyNormalizer{IgnoreScheme: o.IgnoreScheme, Toplevel: o.Toplevel}

	dbDir := path.Join(o.DbDir, config.GetContext(), configV1.Kind_seed.String())

	// Create database for storing state
	seedDb, err := importutil.NewImportDb(dbDir, o.ResetDb)
	if err != nil {
		return fmt.Errorf("failed to initialize import db: %w", err)
	}
	defer seedDb.Close()

	// Import existing seeds into state database
	if !o.SkipImport {
		log.Info().Msg("Importing existing seeds...")
		err = importutil.ImportExisting(seedDb, client, configV1.Kind_seed, uriNormalizer)
		if err != nil {
			return fmt.Errorf("failed to import existing seeds: %w", err)
		}
	}

	// Create Record reader for file input
	rr, err := importutil.NewRecordReader(o.Filename, &importutil.LineAsStringDecoder{}, "*.txt")
	if err != nil {
		return fmt.Errorf("unable to open file '%v': %w", o.Filename, err)
	}

	entityId := o.EntityId
	entityName := o.EntityName
	var entityLabels []*configV1.Label
	if len(o.EntityLabels) > 0 {
		for _, l := range o.EntityLabels {
			kv := strings.SplitN(l, ":", 2)
			if len(kv) != 2 {
				return fmt.Errorf("invalid entity label: %s", l)
			}
			entityLabels = append(entityLabels, &configV1.Label{Key: kv[0], Value: kv[1]})
		}
	}

	// If entityId is set, ignore entityName and entityLabels
	if entityId != "" {
		entityName = ""
		entityLabels = nil
	}

	var seedLabels []*configV1.Label
	if len(o.SeedLabels) > 0 {
		for _, l := range o.SeedLabels {
			kv := strings.SplitN(l, ":", 2)
			if len(kv) != 2 {
				return fmt.Errorf("invalid seed label: %s", l)
			}
			seedLabels = append(seedLabels, &configV1.Label{Key: kv[0], Value: kv[1]})
		}
	}

	// Processor for converting oos records into import records
	proc := func(uri string) error {
		seed := &importutil.SeedDesc{
			EntityId:    entityId,
			EntityName:  entityName,
			EntityLabel: entityLabels,
			Uri:         uri,
			SeedLabel:   seedLabels,
		}
		// If entityId or entityName is not set, use uri as entityName
		if entityId == "" && entityName == "" {
			seed.EntityName = uri
		}

		if uriChecker != nil {
			uri, err := uriChecker.Check(uri)
			if err != nil {
				return fmt.Errorf("failed URL check: %w", err)
			}
			seed.Uri = uri
		}

		normalizedUri, err := uriNormalizer.Normalize(seed.Uri)
		if err != nil {
			return fmt.Errorf("failed to normalize URL '%s': %w", uri, err)
		}

		if ids, err := seedDb.Get(normalizedUri); err != nil {
			return err
		} else if len(ids) > 0 {
			return importutil.ErrAlreadyExists(normalizedUri)
		}

		j, err := json.Marshal(seed)
		if err != nil {
			return err
		}
		if _, err = fmt.Fprintf(out, "%s\n", j); err != nil {
			return err
		}
		return nil
	}

	// Create error logger
	errorLog := log.Output(zerolog.ConsoleWriter{Out: errFile, TimeFormat: time.RFC3339})

	// Error handler for executor
	errHandler := func(state importutil.Job[string]) {
		l := errorLog.With().
			Str("uri", state.Val).
			Str("filename", state.GetFilename()).
			Int("recNum", state.GetRecordNum()).Logger()

		var err importutil.ErrAlreadyExists
		if errors.As(state.GetError(), &err) {
			l.Info().Msg(err.Error())
		} else {
			l.Error().Err(state.GetError()).Msg("")
		}
	}

	// Create cuncurrent executor for processing records
	executor := importutil.NewExecutor(o.Concurrency, proc, errHandler)

	// Read records from file and queue for processing
	for {
		var uri string
		state, err := rr.Next(&uri)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			errorLog.Error().Err(err).Msgf("error decoding record: %v", state)
			continue
		}
		// ignore empty lines
		if uri == "" {
			continue
		}
		executor.Queue <- importutil.Job[string]{State: state, Val: uri}
	}

	// Wait for all records to be processed
	count, success, failed := executor.Wait()

	errorLog.Info().Int("total", count).Int("success", success).Int("failed", failed).Msg("Finished converting records")

	return nil
}
