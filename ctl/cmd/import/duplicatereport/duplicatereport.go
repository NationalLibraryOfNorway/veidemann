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

package duplicatereport

import (
	"fmt"
	"io"
	"os"
	"path"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/connection"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/format"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/importutil"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// options is the options for the convert oos command
type options struct {
	Kind         configV1.Kind
	OutFile      string
	DbDir        string
	ResetDb      bool
	Toplevel     bool
	IgnoreScheme bool
	SkipImport   bool
}

func NewCmd() *cobra.Command {
	o := &options{}

	cmd := &cobra.Command{
		Use:   "duplicatereport KIND",
		Short: "List duplicated seeds or crawl entities in Veidemann",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		ValidArgs: []string{
			configV1.Kind_seed.String(),
			configV1.Kind_crawlEntity.String(),
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := format.GetKind(args[0])
			if kind == configV1.Kind_undefined {
				return fmt.Errorf("undefined kind: %v", kind)
			}
			o.Kind = kind

			// silence usage to prevent printing usage when error occurs
			cmd.SilenceUsage = true

			return run(o)
		},
	}

	cmd.Flags().StringVarP(&o.OutFile, "out-file", "o", "", "File to write output.")
	cmd.Flags().StringVarP(&o.DbDir, "db-dir", "b", "/tmp/veidemannctl", "Directory for storing state db")
	cmd.Flags().BoolVarP(&o.Toplevel, "toplevel", "", false, "Convert URI to toplevel by removing path before checking for duplicates.")
	cmd.Flags().BoolVarP(&o.IgnoreScheme, "ignore-scheme", "", false, "Ignore the URL's scheme when checking for duplicates.")
	cmd.Flags().BoolVar(&o.ResetDb, "truncate", false, "Truncate state database")
	cmd.Flags().BoolVar(&o.SkipImport, "skip-import", false, "Do not import existing seeds into state database")

	return cmd
}

type DuplicateReporter interface {
	Report(w io.Writer) error
}

// run runs the convert oos command with the given options
func run(o *options) error {
	// Create output writer (file or stdout)
	var out io.Writer
	if o.OutFile == "" {
		out = os.Stdout
	} else {
		f, err := os.Create(o.OutFile)
		if err != nil {
			return fmt.Errorf("unable to open output file: %v: %w", o.OutFile, err)
		}
		defer f.Close()
		out = f
	}

	// Connect to Veidemann
	conn, err := connection.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client := configV1.NewConfigClient(conn)

	// Create key normalizer
	var keyNormalizer importutil.KeyNormalizer
	if o.Kind == configV1.Kind_seed {
		keyNormalizer = &importutil.UriKeyNormalizer{Toplevel: o.Toplevel, IgnoreScheme: o.IgnoreScheme}
	}

	dbDir := path.Join(o.DbDir, config.GetContext(), o.Kind.String())

	// Create state Database of kind seed or kind crawlEntity from Veidemann
	stateDb, err := importutil.NewImportDb(dbDir, o.ResetDb)
	if err != nil {
		return fmt.Errorf("failed to initialize import db: %w", err)
	}
	defer stateDb.Close()

	// Import existing into state database
	if !o.SkipImport {
		log.Info().Str("kind", o.Kind.String()).Msg("Importing from Veidemann...")
		err = importutil.ImportExisting(stateDb, client, o.Kind, keyNormalizer)
		if err != nil {
			return fmt.Errorf("failed to import '%v': %w", o.Kind, err)
		}
	}

	var duplicateReporter DuplicateReporter

	if o.Kind == configV1.Kind_seed {
		duplicateReporter = importutil.SeedReporter{ImportDb: stateDb, Client: client}
	} else {
		duplicateReporter = importutil.DuplicateKindReporter{ImportDb: stateDb}
	}

	return duplicateReporter.Report(out)
}
