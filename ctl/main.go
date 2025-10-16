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

package main

import (
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/cmd"

	v "github.com/NationalLibraryOfNorway/veidemann/ctl/version"
)

var (
	version = "dev"
	commit  = "none"
	date    = time.Now().Format(time.RFC3339)
)

//go:generate go run ./docs
func main() {
	v.ClientVersion.Set(version, commit, date)
	cmd.Execute()
}
