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
	"slices"
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

func TestRuleOctetComparison(t *testing.T) {
	for _, tt := range []struct {
		pattern, path string
		match         bool
	}{
		{"/catalog", "/%63atalog", true},
		{"/%63atalog", "/catalog", true},
		{"/%61%5a%30%2d%2e%5f%7e", "/aZ0-._~", true},
		{"/a%2fb", "/a%2Fb", true},
		{"/a%2Fb", "/a%2fb", true},
		{"/a%2Fb", "/a/b", false},
		{"/a%252Fb", "/a%2Fb", false},
		{"/%2563atalog", "/%63atalog", false},
		{"/café%20x%39", "/caf%c3%a9%20x9", true},
		{"/x?q=%61+b", "/x?q=a+b", true},
		{"/x?q=a+b", "/x?q=a%20b", false},
		{"/x%3fy", "/x?y", false},
		{"/x%2ay", "/xANYy", false},
		{"/x%24", "/x", false},
		{"/x%zz", "/x%zz", true},
	} {
		rules := "User-agent: *\nDisallow: " + tt.pattern + "$"
		if got := AgentAllowed(rules, "Bot", "https://example.org"+tt.path); got == tt.match {
			t.Errorf("Disallow %q on %q: allowed = %t, want %t", tt.pattern, tt.path, got, !tt.match)
		}
	}

	const rules = "User-agent: *\nDisallow: /%63atalog\nAllow: /catalog"
	m := NewRobotsMatcher()
	if !m.AgentAllowed(rules, "Bot", "https://example.org/catalog") || m.MatchingLine() != 3 {
		t.Errorf("equivalent patterns should select Allow on line 3, got line %d", m.MatchingLine())
	}
}

func TestDirectoryIndexAllow(t *testing.T) {
	for _, tt := range []struct {
		suffix string
		want   bool
	}{
		{"index.htm", true}, {"index.html", true}, {"index.htm$", true}, {"index.html$", true},
		{"index.html.bak", false}, {"index.html*", false}, {"index.html?lang=en", false},
		{"?next=/index.html", false}, {"index.html$$", false},
	} {
		rules := "User-agent: *\nDisallow: /docs/\nAllow: /docs/" + tt.suffix
		if got := AgentAllowed(rules, "Bot", "https://example.org/docs/"); got != tt.want {
			t.Errorf("Allow suffix %q: allowed = %t, want %t", tt.suffix, got, tt.want)
		}
	}
	const rules = "User-agent: *\nDisallow: /docs/\nAllow: /docs/?next=/index.html"
	if AgentAllowed(rules, "Bot", "https://example.org/docs/?next=/") {
		t.Error("index filename inside the query must not grant a directory match")
	}
}

func TestProductTokensAndAliases(t *testing.T) {
	const uri = "https://example.org/book"
	const rules = "User-agent: *\nAllow: /\nUser-agent: A&B\nDisallow: /book\nUser-agent: C!D\nAllow: /"
	for _, tt := range []struct {
		aliases []string
		want    bool
	}{
		{[]string{"a&b/1.0"}, false},
		{[]string{"A"}, true},
		{[]string{"A&B/1", "C!D/2"}, false},
		{[]string{"C!D/2", "A&B/1"}, false},
	} {
		original := slices.Clone(tt.aliases)
		if got := AgentsAllowed(rules, tt.aliases, uri); got != tt.want || !slices.Equal(tt.aliases, original) {
			t.Errorf("AgentsAllowed(%v) = %t, want %t; aliases after call = %v", original, got, tt.want, tt.aliases)
		}
	}
	if AgentAllowed("User-agent: C!D\nDisallow: /", "C!D/1", uri) {
		t.Error("missed exact product token containing !")
	}
	if AgentAllowed("User-agent: *\nDisallow: /\nUser-agent: \nAllow: /", "", uri) {
		t.Error("empty identifier should use wildcard group")
	}
}

func TestRobotsFileImplicitAllow(t *testing.T) {
	m := NewRobotsMatcher()
	const rules = "User-agent: Bot\nDisallow: /\nDisallow: /robots.txt"
	for _, tc := range []struct {
		path  string
		allow bool
	}{
		{"/blocked", false}, {"/robots.txt", true}, {"/robots.txt#fragment", true},
		{"/%72obots.txt", true}, {"/Robots.txt", false}, {"/sub/robots.txt", false},
		{"/robots.txt/", false}, {"/robots.txt?x=1", false}, {"/robots.txt?", false},
	} {
		if got := m.AgentAllowed(rules, "Bot", "https://example.org"+tc.path); got != tc.allow {
			t.Errorf("%s: allowed = %t", tc.path, got)
		}
		if tc.allow && (m.MatchingLine() != 0 || m.Disallowed() || m.EverSeenSpecificAgent() || m.DisallowedIgnoreGlobal()) {
			t.Errorf("%s: stale match diagnostics", tc.path)
		}
	}
}

func TestLiteralRobotsOperators(t *testing.T) {
	for _, tc := range []struct {
		pattern, path string
		match         bool
	}{
		{"/file-%2A.html$", "/file-*.html", true}, {"/file-%2a.html$", "/file-%2A.html", true},
		{"/file-%2A.html$", "/file-any.html", false}, {"/foo-%24$", "/foo-$", true},
		{"/foo-%24$", "/foo-%24", true}, {"/foo-%24$", "/foo-$more", false},
		{"/*%2a$", "/abc*", true}, {"/*2A$", "/abc%2A", false},
		{"/%2A$", "/%252A", false}, {"/%24$", "/%2524", false},
		{"/a$b$", "/a%24b", true},
	} {
		got := AgentAllowed("User-agent: *\nDisallow: "+tc.pattern, "Bot", "https://example.org"+tc.path)
		if got == tc.match {
			t.Errorf("Disallow %q on %q: allowed = %t", tc.pattern, tc.path, got)
		}
	}
	const rules = "User-agent: *\nDisallow: /%24x\nAllow: /$x"
	if !AgentAllowed(rules, "Bot", "https://example.org/%24x") {
		t.Error("equal literal specificity should select Allow")
	}
}
