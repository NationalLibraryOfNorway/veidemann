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
// This file tests the robots.txt parsing and matching code found in robots.cc
// against the current Robots Exclusion Protocol (REP), RFC 9309.
// https://www.rfc-editor.org/rfc/rfc9309.html

//

// Modified for Veidemann to use the standard Go testing package.
// Converted 2020-04-21, from https://github.com/google/robotstxt/blob/master/robots_test.cc

package grobotstxt

import (
	"testing"
)

func TestParserHelpers(t *testing.T) {

	// Line :948
	TestPath := func(t *testing.T, uri, expected string) {
		x := getPathParamsQuery(uri)
		t.Helper()
		if x != expected {
			t.Errorf("%q: got %q, want %q", uri, x, expected)
		}
	}

	TestEscape := func(t *testing.T, uri, expected string) {
		x := escapePattern(uri)
		t.Helper()
		if x != expected {
			t.Errorf("%q: got %q, want %q", uri, x, expected)
		}
	}

	t.Run("TestGetPathParamsQuery", func(t *testing.T) {
		// Only testing URLs that are already correctly escaped here.
		TestPath(t, "", "/")
		TestPath(t, "http://www.example.com", "/")
		TestPath(t, "http://www.example.com/", "/")
		TestPath(t, "http://www.example.com/a", "/a")
		TestPath(t, "http://www.example.com/a/", "/a/")
		TestPath(t, "http://www.example.com/a/b?c=http://d.e/", "/a/b?c=http://d.e/")
		TestPath(t, "http://www.example.com/a/b?c=d&e=f#fragment", "/a/b?c=d&e=f")
		TestPath(t, "example.com", "/")
		TestPath(t, "example.com/", "/")
		TestPath(t, "example.com/a", "/a")
		TestPath(t, "example.com/a/", "/a/")
		TestPath(t, "example.com/a/b?c=d&e=f#fragment", "/a/b?c=d&e=f")
		TestPath(t, "a", "/")
		TestPath(t, "a/", "/")
		TestPath(t, "/a", "/a")
		TestPath(t, "a/b", "/b")
		TestPath(t, "example.com?a", "/?a")
		TestPath(t, "example.com/a;b#c", "/a;b")
		TestPath(t, "//a/b/c", "/b/c")
	})

	t.Run("TestMaybeEscapePattern", func(t *testing.T) {
		TestEscape(t, "http://www.example.com", "http://www.example.com")
		TestEscape(t, "/a/b/c", "/a/b/c")
		TestEscape(t, "á", "%C3%A1")
		TestEscape(t, "%aa", "%AA")
	})

}
