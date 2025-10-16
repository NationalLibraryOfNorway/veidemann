// Copyright © 2017 National Library of Norway.
// Licensed under the Apache License, GitVersion 2.0 (the "License");
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

package cmd

import (
	"fmt"
	"os"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/abort"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/abortjobexecution"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/activeroles"
	configcmd "github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/config"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/create"
	deletecmd "github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/delete"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/get"
	importcmd "github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/import"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/logconfig"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/login"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/logout"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/pause"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/report"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/run"
	scriptparameters "github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/script_parameters"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/status"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/unpause"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd/update"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/version"

	"github.com/spf13/cobra"
)

// NewRootCmd returns the root command.
func NewRootCmd() *cobra.Command {
	cobra.EnableCommandSorting = false

	cmd := &cobra.Command{
		Use:               "veidemannctl",
		Short:             "veidemannctl controls the Veidemann web crawler",
		Long:              "veidemannctl controls the Veidemann web crawler",
		DisableAutoGenTag: true,
		Version:           version.ClientVersion.String(),
	}

	// Add global flags
	cmd.PersistentFlags().String("config", "", "Path to the config file to use (By default configuration file is stored under $HOME/.veidemann/contexts/")
	cmd.PersistentFlags().String("context", "", "The name of the context to use")
	cmd.PersistentFlags().String("server", "", "The address of the Veidemann server to use")
	cmd.PersistentFlags().String("server-name-override", "",
		"If set, it will override the virtual host name of authority (e.g. :authority header field) in requests")
	cmd.PersistentFlags().String("api-key", "",
		"If set, it will be used as the bearer token for authentication")
	cmd.PersistentFlags().Bool("insecure", false, "If set, it will use an insecure connection")
	cmd.PersistentFlags().String("log-level", "info", `set log level, available levels are "panic", "fatal", "error", "warn", "info", "debug" and "trace"`)
	cmd.PersistentFlags().String("log-format", "pretty", `set log format, available formats are: "pretty" or "json"`)
	cmd.PersistentFlags().Bool("log-caller", false, "include information about caller in log output")

	// Add subcommands
	cmd.AddCommand(configcmd.NewConfigCmd()) // config

	cmd.AddGroup(&cobra.Group{
		ID:    "basic",
		Title: "Basic Commands:",
	})
	cmd.AddCommand(get.NewCmd())       // get
	cmd.AddCommand(create.NewCmd())    // create
	cmd.AddCommand(update.NewCmd())    // update
	cmd.AddCommand(deletecmd.NewCmd()) // delete

	cmd.AddGroup(&cobra.Group{
		ID:    "advanced",
		Title: "Advanced Commands:",
	})
	cmd.AddCommand(report.NewCmd())    // report
	cmd.AddCommand(importcmd.NewCmd()) // import

	cmd.AddGroup(&cobra.Group{
		ID:    "run",
		Title: "Crawl Commands:",
	})
	cmd.AddCommand(run.NewCmd())               // run
	cmd.AddCommand(abort.NewCmd())             // abort
	cmd.AddCommand(abortjobexecution.NewCmd()) // abortjobexecution

	cmd.AddGroup(&cobra.Group{
		ID:    "status",
		Title: "Management Commands:",
	})
	cmd.AddCommand(status.NewCmd())  // status
	cmd.AddCommand(pause.NewCmd())   // pause
	cmd.AddCommand(unpause.NewCmd()) // unpause

	cmd.AddGroup(&cobra.Group{
		ID:    "login",
		Title: "Authentication Commands:",
	})
	cmd.AddCommand(login.NewCmd())  // login
	cmd.AddCommand(logout.NewCmd()) // logout

	cmd.AddGroup(&cobra.Group{
		ID:    "debug",
		Title: "Troubleshooting and Debug Commands:",
	})
	cmd.AddCommand(scriptparameters.NewCmd()) // script-parameters
	cmd.AddCommand(logconfig.NewCmd())        // logconfig
	cmd.AddCommand(activeroles.NewCmd())      // activeroles

	return cmd
}

// Execute initializes the root command and executes it.
func Execute() {
	// Initialize root command
	cmd := NewRootCmd()

	// Register function to run after command is initialized
	cobra.OnInitialize(func() {
		// Initialize config from flags
		err := config.Init(cmd.PersistentFlags())
		if err != nil {
			fmt.Printf("Initialization failed: %v\n", err)
			os.Exit(1)
		}
	})

	// Execute root command
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
