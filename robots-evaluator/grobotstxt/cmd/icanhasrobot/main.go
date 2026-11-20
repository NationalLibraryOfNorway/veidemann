// Copyright 2020 Jim Smart
// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// -----------------------------------------------------------------------------
// File: robots_main.cc
// -----------------------------------------------------------------------------
//
// Simple binary to assess whether a URL is accessible to a set of user-agents
// according to records found in a local robots.txt file, based on Google's
// robots.txt parsing and matching algorithms.
// Usage:
//     robots_main <local_path_to_robotstxt> <user_agents> <url>
// Arguments:
// local_path_to_robotstxt: local path to a file containing robots.txt records.
//   For example: /home/users/username/robots.txt
// user_agents: a token to be matched against records in the robots.txt (or
// a comma-separated list of aliases for one crawler)
//   For example: Googlebot or Googlebot,Googlebot-image
// url: an absolute hierarchical URL to match against the robots.txt records.
//   For example: https://example.com/accessible/url.html
// Returns: Prints a sentence with verdict about whether 'user_agent' is allowed
// to access 'url' based on records in 'local_path_to_robotstxt'. Exits with status
// code 0 if allowed, 1 if disallowed, or 2 otherwise (e.g. bad inputs).
//

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/grobotstxt"
)

func loadFile(filename string) (string, error) {
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// TODO(js) Can we use the flag package here, instead of manually handling params and help/usage.

func showHelp(argv []string, stderr io.Writer) {
	_, _ = fmt.Fprint(stderr, "Shows whether the given user_agent(s) and URI combination"+
		" is allowed or disallowed by the given robots.txt file.\n\n")
	_, _ = fmt.Fprint(stderr, "Usage:\n"+
		"  "+argv[0]+" <robots.txt filename> <user_agents> <URI>\n\n")
	_, _ = fmt.Fprint(stderr, "The URI must be an absolute URL with a hierarchical path.\n\n")
	_, _ = fmt.Fprint(stderr, "The user_agents argument accepts comma-separated aliases of one crawler. Rules for matching aliases are combined.\n\n")
	_, _ = fmt.Fprint(stderr, "Example:\n"+
		"  "+argv[0]+" robots.txt FooBot http://example.com/foo\n")
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr io.Writer) int {
	filename := ""
	if len(argv) >= 2 {
		filename = argv[1]
	}
	if filename == "-h" || filename == "-help" || filename == "--help" {
		showHelp(argv, stderr)
		return 2
	}

	if len(argv) != 4 {
		_, _ = fmt.Fprint(stderr, "Invalid amount of arguments. Showing help.\n\n")
		showHelp(argv, stderr)
		return 2
	}

	robotsContent, err := loadFile(filename)
	if err != nil {
		_, _ = fmt.Fprint(stderr, "failed to read file \""+filename+"\"\n")
		return 2
	}

	userAgent := argv[2]
	userAgentList := strings.Split(userAgent, ",")
	uri := argv[3]

	allowed, err := grobotstxt.NewRobotsMatcher().AgentsAllowedWithError(robotsContent, userAgentList, uri)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid URL %q: %v\n", uri, err)
		return 2
	}

	m := "user-agent '" + userAgent + "' with URI '" + uri + "': "
	if allowed {
		m += "ALLOWED"
	} else {
		m += "DISALLOWED"
	}
	m += "\n"
	_, _ = fmt.Fprint(stdout, m)
	if len(robotsContent) == 0 && allowed {
		_, _ = fmt.Fprint(stdout, "notice: robots file is empty so all user-agents are allowed\n")
	}

	if allowed {
		return 0
	}
	return 1
}
