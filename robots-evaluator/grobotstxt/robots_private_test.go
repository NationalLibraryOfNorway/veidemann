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
	"strings"
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

// getPathParamsQuery extracts path (with params) and query part from the given URI.
// Removes scheme, authority, and fragment. Result always starts with "/".
// Returns "/" if the URI doesn't have a path or is not valid.
func getPathParamsQuery(uri string) string {
	// Line :117

	// path := ""

	// Initial two slashes are ignored.
	searchStart := 0
	if len(uri) >= 2 && uri[0] == '/' && uri[1] == '/' {
		searchStart = 2
	}

	earlyPath := findFirstOf(uri, "/?;", searchStart)
	protocolEnd := find(uri, "://", searchStart)
	if earlyPath != -1 && earlyPath < protocolEnd {
		// If path, param or query starts before ://, :// doesn't indicate protocol.
		protocolEnd = -1
	}
	if protocolEnd == -1 {
		protocolEnd = searchStart
	} else {
		protocolEnd += 3
	}

	pathStart := findFirstOf(uri, "/?;", protocolEnd)
	if pathStart != -1 {
		hashPos := findByte(uri, '#', searchStart)
		if hashPos != -1 && hashPos < pathStart {
			return "/"
		}
		pathEnd := hashPos
		if hashPos == -1 {
			pathEnd = len(uri)
		}
		if uri[pathStart] != '/' {
			return "/" + uri[pathStart:pathEnd]
		}
		return uri[pathStart:pathEnd]
	}

	return "/"
}

func findFirstOf(s, match string, i int) int {
	j := strings.IndexAny(s[i:], match)
	if j != -1 {
		j += i
	}
	return j
}

func find(s, match string, i int) int {
	j := strings.Index(s[i:], match)
	if j != -1 {
		j += i
	}
	return j
}

func findByte(s string, b byte, i int) int {
	j := strings.IndexByte(s[i:], b)
	if j != -1 {
		j += i
	}
	return j
}

func TestMatcherOnEscapedPath(t *testing.T) {
	const rules = "User-agent: FooBot\nDisallow: /\nAllow: /example?x=1$\n"
	matcher := NewRobotsMatcher()
	for _, tt := range []struct {
		uri  string
		want bool
	}{
		{"", false},
		{"https://example.com/example?x=1#fragment", true},
		{"https://example.com/example?x=1&y=2", false},
	} {
		if got := matcher.agentsAllowedOnPath(rules, []string{"FooBot"}, getPathParamsQuery(tt.uri)); got != tt.want {
			t.Errorf("matching escaped URL %q = %t, want %t", tt.uri, got, tt.want)
		}
	}
}

func TestWHATWGURLMatching(t *testing.T) {
	const rules = "User-agent: FooBot\nDisallow: /\nAllow: /example?x=1&y=2$\nAllow: /empty?$\nAllow: /escaped%2Fpath$\nAllow: /caf%C3%A9$\n"
	for _, tt := range []struct {
		name string
		uri  string
		want bool
	}{
		{"dot segments and fragment", "https://example.com/dir/%2e%2e/example?x=1&y=2#fragment", true},
		{"query ordering", "https://example.com/example?y=2&x=1", false},
		{"explicit empty query", "https://example.com/empty?#fragment", true},
		{"absent query", "https://example.com/empty#fragment", false},
		{"escaped delimiter", "https://example.com/escaped%2Fpath", true},
		{"literal delimiter", "https://example.com/escaped/path", false},
		{"Unicode", "https://bücher.example/café", true},
		{"invalid host", "https://[invalid]/example?x=1&y=2", false},
		{"relative", "/example?x=1&y=2", false},
		{"opaque path", "data:example?x=1&y=2", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := AgentAllowed(rules, "FooBot", tt.uri); got != tt.want {
				t.Errorf("AgentAllowed(%q) = %t, want %t", tt.uri, got, tt.want)
			}
		})
	}
}
