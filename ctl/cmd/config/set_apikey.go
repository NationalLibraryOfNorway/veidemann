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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// newSetApiKeyCmd represents the set-apikey command
func newSetApiKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-apikey",
		Short: "Set the api-key to use for authentication",
		Long: "Set the api-key to use for authentication\n\n" +
			"Examples:\n" +
			"  # Enter an API key without echoing it\n" +
			"  veidemannctl config set-apikey\n\n" +
			"  # Read an API key from a secret manager\n" +
			"  secret-tool lookup service veidemann | veidemannctl config set-apikey",
		Aliases: []string{"set-apikey", "set-api-key"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// silence usage to avoid printing usage when returning an error
			cmd.SilenceUsage = true

			apiKey, err := readAPIKey(cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			return config.SetApiKey(apiKey)
		},
	}
}

func readAPIKey(input io.Reader, output io.Writer) (string, error) {
	var value []byte
	if file, ok := input.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		if _, err := fmt.Fprint(output, "API key: "); err != nil {
			return "", err
		}
		var err error
		value, err = term.ReadPassword(int(file.Fd()))
		_, _ = fmt.Fprintln(output)
		if err != nil {
			return "", fmt.Errorf("read API key: %w", err)
		}
	} else {
		var err error
		value, err = io.ReadAll(bufio.NewReader(input))
		if err != nil {
			return "", fmt.Errorf("read API key: %w", err)
		}
		value = []byte(strings.TrimSuffix(strings.TrimSuffix(string(value), "\n"), "\r"))
	}

	if len(value) == 0 {
		return "", errors.New("API key must not be empty")
	}
	if strings.ContainsAny(string(value), "\r\n\x00") {
		return "", errors.New("API key must be a single line without NUL bytes")
	}
	return string(value), nil
}
