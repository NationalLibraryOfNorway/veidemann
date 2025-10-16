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

package version

import (
	"fmt"
	"runtime"
)

// Version contains version information
type Version struct {
	ClientVersion string
	GitCommit     string
	BuildDate     string
}

// String returns the version information as a string
func (v *Version) String() string {
	return fmt.Sprintf("Client version: %s, Git commit: %s, Build date: %s, Go version: %s, Platform: %s/%s",
		v.ClientVersion,
		v.GitCommit,
		v.BuildDate,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH)
}

// Set sets the version information
func (v *Version) Set(gitVersion, gitCommit, buildDate string) {
	v.ClientVersion = gitVersion
	v.GitCommit = gitCommit
	v.BuildDate = buildDate
}

var ClientVersion = &Version{}
