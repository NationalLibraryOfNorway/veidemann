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

package grobotstxt_test

import (
	"slices"
	"strings"

	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/grobotstxt"
	"testing"
)

func TestRobots(t *testing.T) {

	// Line :30
	IsUserAgentAllowed := func(robotstxt, userAgent, url string) bool {
		matcher := grobotstxt.NewRobotsMatcher()
		return matcher.AgentAllowed(robotstxt, userAgent, url)
	}

	AllowedByRobots := func(robotstxt, userAgents, url string) bool {
		userAgentList := strings.Split(userAgents, ",")
		matcher := grobotstxt.NewRobotsMatcher()
		return matcher.AgentsAllowed(robotstxt, userAgentList, url)
	}

	// Google-specific: system test.
	t.Run("GoogleOnly_SystemTest", func(t *testing.T) {
		// Line :37
		const robotstxt = "user-agent: FooBot\n" +
			"disallow: /\n"
		// Empty robots.txt: everything allowed.
		expectEqual(t, true, IsUserAgentAllowed("", "FooBot", "https://example.com/"))

		// Empty user-agent to be matched: everything allowed.
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "", "https://example.com/"))

		// A root path matches the disallow rule.
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "https://example.com/"))

		// All params empty: same as robots.txt empty, everything allowed.
		expectEqual(t, true, IsUserAgentAllowed("", "", "https://example.com/"))
	})

	// Rules are colon separated name-value pairs. The following names are
	// provisioned:
	//     user-agent: <value>
	//     allow: <value>
	//     disallow: <value>
	// See REP I-D section "Protocol Definition".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.1
	//
	// Google specific: webmasters sometimes miss the colon separator, but it's
	// obvious what they mean by "disallow /", so we assume the colon if it's
	// missing.
	t.Run("ID_LineSyntax_Line", func(t *testing.T) {
		// Line :65
		const robotstxt_correct = "user-agent: FooBot\n" +
			"disallow: /\n"
		const robotstxt_incorrect = "foo: FooBot\n" +
			"bar: /\n"
		const robotstxt_incorrect_accepted = "user-agent FooBot\n" +
			"disallow /\n"
		const url = "http://foo.bar/x/y"
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_correct, "FooBot", url))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_incorrect, "FooBot", url))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_incorrect_accepted, "FooBot", url))
	})

	// A group is one or more user-agent line followed by rules, and terminated
	// by a another user-agent line. Rules for same user-agents are combined
	// opaquely into one group. Rules outside groups are ignored.
	// See REP I-D section "Protocol Definition".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.1
	t.Run("ID_LineSyntax_Groups", func(t *testing.T) {
		// Line :87
		const robotstxt = "allow: /foo/bar/\n" +
			"\n" +
			"user-agent: FooBot\n" +
			"disallow: /\n" +
			"allow: /x/\n" +
			"user-agent: BarBot\n" +
			"disallow: /\n" +
			"allow: /y/\n" +
			"\n" +
			"\n" +
			"allow: /w/\n" +
			"user-agent: BazBot\n" +
			"\n" +
			"user-agent: FooBot\n" +
			"allow: /z/\n" +
			"disallow: /\n"

		const url_w = "http://foo.bar/w/a"
		const url_x = "http://foo.bar/x/b"
		const url_y = "http://foo.bar/y/c"
		const url_z = "http://foo.bar/z/d"
		const url_foo = "http://foo.bar/foo/bar/"

		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url_x))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url_z))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url_y))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "BarBot", url_y))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "BarBot", url_w))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "BarBot", url_z))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "BazBot", url_z))

		// // Lines with rules outside groups are ignored.
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url_foo))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "BarBot", url_foo))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "BazBot", url_foo))
	})

	// Group must not be closed by rules not explicitly defined in the REP I-D.
	// See REP I-D section "Protocol Definition".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.1
	t.Run("ID_LineSyntax_Groups_OtherRules", func(t *testing.T) {
		{
			const robotstxt = "User-agent: BarBot\n" +
				"Sitemap: https://foo.bar/sitemap\n" +
				"User-agent: *\n" +
				"Disallow: /\n"
			const url = "http://foo.bar/"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "BarBot", url))
		}
		{
			const robotstxt = "User-agent: FooBot\n" +
				"Invalid-Unknown-Line: unknown\n" +
				"User-agent: *\n" +
				"Disallow: /\n"
			const url = "http://foo.bar/"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "BarBot", url))
		}
	})

	// Combine rules for matching aliases of a single crawler.
	t.Run("ID_Multiple_Useragents", func(t *testing.T) {
		const robotstxt = "user-agent: googlebot-news\n" +
			"Disallow: /bar/\n" +
			"\n" +
			"user-agent: *\n" +
			"Disallow: /baz/\n" +
			"\n\n" +
			"user-agent: googlebot\n" +
			"Disallow: /foo/\n"

		const url_foo = "http://foo.bar/foo/"
		const url_bar = "http://foo.bar/bar/"
		const url_baz = "http://foo.bar/baz/"
		const url_qux = "http://foo.bar/qux/"

		expectEqual(t, false, AllowedByRobots(robotstxt, "googlebot,googlebot-news", url_foo))
		expectEqual(t, false, AllowedByRobots(robotstxt, "googlebot,googlebot-news", url_bar))
		expectEqual(t, true, AllowedByRobots(robotstxt, "googlebot,googlebot-news", url_baz))
		expectEqual(t, true, AllowedByRobots(robotstxt, "googlebot,googlebot-news", url_qux))
	})

	// REP lines are case insensitive. See REP I-D section "Protocol Definition".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.1
	t.Run("ID_REPLineNamesCaseInsensitive", func(t *testing.T) {
		// Line :128
		const robotstxt_upper = "USER-AGENT: FooBot\n" +
			"ALLOW: /x/\n" +
			"DISALLOW: /\n"
		const robotstxt_lower = "user-agent: FooBot\n" +
			"allow: /x/\n" +
			"disallow: /\n"
		const robotstxt_camel = "uSeR-aGeNt: FooBot\n" +
			"AlLoW: /x/\n" +
			"dIsAlLoW: /\n"
		const url_allowed = "http://foo.bar/x/y"
		const url_disallowed = "http://foo.bar/a/b"

		expectEqual(t, true, IsUserAgentAllowed(robotstxt_upper, "FooBot", url_allowed))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_lower, "FooBot", url_allowed))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_camel, "FooBot", url_allowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_upper, "FooBot", url_disallowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_lower, "FooBot", url_disallowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_camel, "FooBot", url_disallowed))
	})

	// A user-agent line is expected to contain only [a-zA-Z_-] characters and must
	// not be empty. See REP I-D section "The user-agent line".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.1
	t.Run("ID_VerifyValidUserAgentsToObey", func(t *testing.T) {
		// Line :155
		expectEqual(t, true, grobotstxt.IsValidUserAgentToObey("Foobot"))
		expectEqual(t, true, grobotstxt.IsValidUserAgentToObey("Foobot-Bar"))
		expectEqual(t, true, grobotstxt.IsValidUserAgentToObey("Foo_Bar"))

		//   expectEqual(t, false, grobotstxt.IsValidUserAgentToObey(absl::string_view()));
		expectEqual(t, false, grobotstxt.IsValidUserAgentToObey(""))
		expectEqual(t, false, grobotstxt.IsValidUserAgentToObey("ツ"))

		// expectEqual(t, false, grobotstxt.IsValidUserAgentToObey("Foobot*")) // Allowed by RFC 7231.
		expectEqual(t, false, grobotstxt.IsValidUserAgentToObey(" Foobot "))
		expectEqual(t, false, grobotstxt.IsValidUserAgentToObey("Foobot/2.1"))

		expectEqual(t, false, grobotstxt.IsValidUserAgentToObey("Foobot Bar"))
	})

	// A user-agent line can in fact contain a wider range of characters
	// than the original spec referenced above, according to RFC7231.
	// https://httpwg.org/specs/rfc7231.html#header.user-agent
	t.Run("ID_VerifyValidUserAgentsToObey", func(t *testing.T) {
		expectEqual(t, true, grobotstxt.IsValidUserAgentToObey("Foo12bot"))
		expectEqual(t, true, grobotstxt.IsValidUserAgentToObey("Foobot~Bar"))
	})

	// User-agent line values are case insensitive. See REP I-D section "The
	// user-agent line".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.1
	t.Run("ID_UserAgentValueCaseInsensitive", func(t *testing.T) {
		// Line :174
		const robotstxt_upper = "User-Agent: FOO BAR\n" +
			"Allow: /x/\n" +
			"Disallow: /\n"
		const robotstxt_lower = "User-Agent: foo bar\n" +
			"Allow: /x/\n" +
			"Disallow: /\n"
		const robotstxt_camel = "User-Agent: FoO bAr\n" +
			"Allow: /x/\n" +
			"Disallow: /\n"
		const url_allowed = "http://foo.bar/x/y"
		const url_disallowed = "http://foo.bar/a/b"

		expectEqual(t, true, IsUserAgentAllowed(robotstxt_upper, "Foo", url_allowed))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_lower, "Foo", url_allowed))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_camel, "Foo", url_allowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_upper, "Foo", url_disallowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_lower, "Foo", url_disallowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_camel, "Foo", url_disallowed))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_upper, "foo", url_allowed))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_lower, "foo", url_allowed))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_camel, "foo", url_allowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_upper, "foo", url_disallowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_lower, "foo", url_disallowed))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_camel, "foo", url_disallowed))
	})

	// Google specific: accept user-agent value up to the first space. Space is not
	// allowed in user-agent values, but that doesn't stop webmasters from using
	// them. This is more restrictive than the I-D, since in case of the bad value
	// "Googlebot Images" we'd still obey the rules with "Googlebot".
	// Extends REP I-D section "The user-agent line"
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.1
	t.Run("GoogleOnly_AcceptUserAgentUpToFirstSpace", func(t *testing.T) {
		// Line :210
		expectEqual(t, false, grobotstxt.IsValidUserAgentToObey("Foobot Bar"))
		const robotstxt = "User-Agent: *\n" +
			"Disallow: /\n" +
			"User-Agent: Foo Bar\n" +
			"Allow: /x/\n" +
			"Disallow: /\n"
		const url = "http://foo.bar/x/y"

		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "Foo", url))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "Foo Bar", url))
	})

	// If no group matches the user-agent, crawlers must obey the first group with a
	// user-agent line with a "*" value, if present. If no group satisfies either
	// condition, or no groups are present at all, no rules apply.
	// See REP I-D section "The user-agent line".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.1
	t.Run("ID_GlobalGroups_Secondary", func(t *testing.T) {
		// Line :229
		const robotstxt_empty = ""
		const robotstxt_global = "user-agent: *\n" +
			"allow: /\n" +
			"user-agent: FooBot\n" +
			"disallow: /\n"
		const robotstxt_only_specific = "user-agent: FooBot\n" +
			"allow: /\n" +
			"user-agent: BarBot\n" +
			"disallow: /\n" +
			"user-agent: BazBot\n" +
			"disallow: /\n"
		const url = "http://foo.bar/x/y"

		expectEqual(t, true, IsUserAgentAllowed(robotstxt_empty, "FooBot", url))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt_global, "FooBot", url))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_global, "BarBot", url))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_only_specific, "QuxBot", url))
	})

	// Matching rules against URIs is case sensitive.
	// See REP I-D section "The Allow and Disallow lines".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.2
	t.Run("ID_AllowDisallow_Value_CaseSensitive", func(t *testing.T) {
		// Line :254
		const robotstxt_lowercase_url = "user-agent: FooBot\n" +
			"disallow: /x/\n"
		const robotstxt_uppercase_url = "user-agent: FooBot\n" +
			"disallow: /X/\n"
		const url = "http://foo.bar/x/y"

		expectEqual(t, false, IsUserAgentAllowed(robotstxt_lowercase_url, "FooBot", url))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt_uppercase_url, "FooBot", url))
	})

	// The most specific match found MUST be used. The most specific match is the
	// match that has the most octets. In case of multiple rules with the same
	// length, the least strict rule must be used.
	// See REP I-D section "The Allow and Disallow lines".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.2
	t.Run("ID_LongestMatch", func(t *testing.T) {
		// Line :272

		const url = "http://foo.bar/x/page.html"
		func() {
			const robotstxt = "user-agent: FooBot\n" +
				"disallow: /x/page.html\n" +
				"allow: /x/\n"

			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url))
		}()
		func() {
			const robotstxt = "user-agent: FooBot\n" +
				"allow: /x/page.html\n" +
				"disallow: /x/\n"

			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/x/"))
		}()
		func() {
			const robotstxt = "user-agent: FooBot\n" +
				"disallow: \n" +
				"allow: \n"
			// In case of equivalent disallow and allow patterns for the same
			// user-agent, allow is used.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
		}()
		func() {
			const robotstxt = "user-agent: FooBot\n" +
				"disallow: /\n" +
				"allow: /\n"
			// In case of equivalent disallow and allow patterns for the same
			// user-agent, allow is used.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
		}()
		func() {
			const url_a = "http://foo.bar/x"
			const url_b = "http://foo.bar/x/"
			const robotstxt = "user-agent: FooBot\n" +
				"disallow: /x\n" +
				"allow: /x/\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url_a))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url_b))
		}()

		func() {
			const robotstxt = "user-agent: FooBot\n" +
				"disallow: /x/page.html\n" +
				"allow: /x/page.html\n"
			// In case of equivalent disallow and allow patterns for the same
			// user-agent, allow is used.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
		}()
		func() {
			const robotstxt = "user-agent: FooBot\n" +
				"allow: /page\n" +
				"disallow: /*.html\n"
			// Longest match wins.
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/page.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/page"))
		}()
		func() {
			const robotstxt = "user-agent: FooBot\n" +
				"allow: /x/page.\n" +
				"disallow: /*.html\n"
			// Longest match wins.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/x/y.html"))
		}()
		func() {
			const robotstxt = "User-agent: *\n" +
				"Disallow: /x/\n" +
				"User-agent: FooBot\n" +
				"Disallow: /y/\n"
			// Most specific group for FooBot allows implicitly /x/page.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/x/page"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/y/page"))
		}()
	})

	// Octets in the URI and robots.txt paths outside the range of the US-ASCII
	// coded character set, and those in the reserved range defined by RFC3986,
	// MUST be percent-encoded as defined by RFC3986 prior to comparison.
	// See REP I-D section "The Allow and Disallow lines".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.2
	//
	// NOTE: It's up to the caller to percent encode a URL before passing it to the
	// parser. Percent encoding URIs in the rules is unnecessary.
	t.Run("ID_Encoding", func(t *testing.T) {
		// Line :372
		const robotstxt = "User-agent: FooBot\n" +
			"Disallow: /\n" +
			"Allow: /foo/bar?qux=taz&baz=http://foo.bar?tar&par\n"
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar?qux=taz&baz=http://foo.bar?tar&par"))
	})

	// 3 byte character: /foo/bar/ツ -> /foo/bar/%E3%83%84
	t.Run("handle a 3 byte character", func(t *testing.T) {
		// Line :385
		const robotstxt = "User-agent: FooBot\n" +
			"Disallow: /\n" +
			"Allow: /foo/bar/ツ\n"
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/%E3%83%84"))
		// // The parser encodes the 3-byte character, but the URL is not %-encoded.
		// expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/ツ"))

		// Departing from Googlebot behaviour:
		// we perform URI normalisation internally now.
		// Due to that, this now returns true here, not false.
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/ツ"))
	})

	// Percent encoded 3 byte character: /foo/bar/%E3%83%84 -> /foo/bar/%E3%83%84
	t.Run("handle a percent encoded 3 byte character", func(t *testing.T) {
		// Line :397
		const robotstxt = "User-agent: FooBot\n" +
			"Disallow: /\n" +
			"Allow: /foo/bar/%E3%83%84\n"
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/%E3%83%84"))
		// expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/ツ"))

		// Departing from Googlebot behaviour:
		// we perform URI normalisation internally now.
		// Due to that, this now returns true here, not false.
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/ツ"))
	})

	// RFC 9309 matches unreserved ASCII in its literal and percent-encoded forms.
	t.Run("handle a percent encoded unreserved US-ASCII", func(t *testing.T) {
		// Line :410
		const robotstxt = "User-agent: FooBot\n" +
			"Disallow: /\n" +
			"Allow: /foo/bar/%62%61%7A\n"
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/baz"))
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/%62%61%7A"))
	})

	// The REP I-D defines the following characters that have special meaning in
	// robots.txt:
	// # - inline comment.
	// $ - end of pattern.
	// * - any number of characters.
	// See REP I-D section "Special Characters".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.3
	t.Run("ID_SpecialCharacters", func(t *testing.T) {
		// Line :429
		func() {
			const robotstxt = "User-agent: FooBot\n" +
				"Disallow: /foo/bar/quz\n" +
				"Allow: /foo/*/qux\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/quz"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/quz"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo//quz"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bax/quz"))
		}()
		func() {
			const robotstxt = "User-agent: FooBot\n" +
				"Disallow: /foo/bar$\n" +
				"Allow: /foo/bar/qux\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/qux"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar/baz"))
		}()
		func() {
			const robotstxt = "User-agent: FooBot\n" +
				"# Disallow: /\n" +
				"Disallow: /foo/quz#qux\n" +
				"Allow: /\n"
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/bar"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/foo/quz"))
		}()
	})

	// An explicit index file may also allow its parent directory.
	t.Run("GoogleOnly_IndexHTMLisDirectory", func(t *testing.T) {
		// Line :473
		const robotstxt = "User-Agent: *\n" +
			"Allow: /allowed-slash/index.html\n" +
			"Disallow: /\n"
		// If index.html is allowed, we interpret this as / being allowed too.
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "foobot", "http://foo.com/allowed-slash/"))
		// Does not exatly match.
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "foobot", "http://foo.com/allowed-slash/index.htm"))
		// Exact match.
		expectEqual(t, true, IsUserAgentAllowed(robotstxt, "foobot", "http://foo.com/allowed-slash/index.html"))
		expectEqual(t, false, IsUserAgentAllowed(robotstxt, "foobot", "http://foo.com/anyother-url"))
	})

	// Google-specific: long lines are ignored after 8 * 2083 bytes. See comment in
	// RobotsTxtParser::Parse().
	t.Run("GoogleOnly_LineTooLong", func(t *testing.T) {
		// Line :493
		eolLen := len("\n")
		const maxLineLen = 2083 * 8
		allow := "allow: "
		disallow := "disallow: "

		// Disallow rule pattern matches the URL after being cut off at kMaxLineLen.
		func() {
			robotstxt := "user-agent: FooBot\n"
			longline := "/x/"
			maxLength := maxLineLen - len(longline) - len(disallow) + eolLen
			for len(longline) < maxLength {
				// absl::StrAppend(&longline, "a");
				longline += "a"
			}
			//   absl::StrAppend(&robotstxt, disallow, longline, "/qux\n");
			robotstxt += disallow + longline + "/qux\n"

			// Matches nothing, so URL is allowed.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fux"))
			// Matches cut off disallow rule.
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar"+longline+"/fux"))
		}()

		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"disallow: /\n"
			longlineA := "/x/"
			longlineB := "/x/"
			maxLength := maxLineLen - len(longlineA) - len(allow) + eolLen
			for len(longlineA) < maxLength {
				longlineA += "a"
				longlineB += "b"
				// absl::StrAppend(&longline_a, "a");
				// absl::StrAppend(&longline_b, "b");
			}
			//   absl::StrAppend(&robotstxt, allow, longline_a, "/qux\n");
			//   absl::StrAppend(&robotstxt, allow, longline_b, "/qux\n");
			robotstxt += allow + longlineA + "/qux\n"
			robotstxt += allow + longlineB + "/qux\n"

			// URL matches the disallow rule.
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/"))
			// Matches the allow rule exactly.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar"+longlineA+"/qux"))
			// Matches cut off allow rule.
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar"+longlineB+"/fux"))
		}()
	})

	t.Run("GoogleOnly_DocumentationChecks", func(t *testing.T) {
		// Line :545
		// Test documentation from
		// https://developers.google.com/search/reference/robots_txt
		// Section "URL matching based on path values".
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"disallow: /\n" +
					"allow: /fish\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/bar"))

			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish/salmon.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fishheads"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fishheads/yummy.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish.html?id=anything"))

			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/Fish.asp"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/catfish"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/?id=fish"))
		}()
		// "/fish*" equals "/fish"
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"disallow: /\n" +
					"allow: /fish*\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/bar"))

			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish/salmon.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fishheads"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fishheads/yummy.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish.html?id=anything"))

			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/Fish.bar"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/catfish"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/?id=fish"))
		}()
		// "/fish/" does not equal "/fish"
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"disallow: /\n" +
					"allow: /fish/\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/bar"))

			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish/"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish/salmon"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish/?salmon"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish/salmon.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish/?id=anything"))

			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish.html"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/Fish/Salmon.html"))
		}()
		// "/*.php"
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"disallow: /\n" +
					"allow: /*.php\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/bar"))

			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/filename.php"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/folder/filename.php"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/folder/filename.php?parameters"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar//folder/any.php.file.html"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/filename.php/"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/index?f=filename.php/"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/php/"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/index?php"))

			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/windows.PHP"))
		}()
		// "/*.php$"
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"disallow: /\n" +
					"allow: /*.php$\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/bar"))

			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/filename.php"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/folder/filename.php"))

			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/filename.php?parameters"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/filename.php/"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/filename.php5"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/php/"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/filename?php"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/aaaphpaaa"))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar//windows.PHP"))
		}()
		// "/fish*.php"
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"disallow: /\n" +
					"allow: /fish*.php\n"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/bar"))

			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fish.php"))
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/fishheads/catfish.php?parameters"))

			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", "http://foo.bar/Fish.PHP"))
		}()
		// Section "Order of precedence for group-member records".
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"allow: /p\n" +
					"disallow: /\n"
			const url = "http://example.com/page"
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
		}()
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"allow: /folder\n" +
					"disallow: /folder\n"
			const url = "http://example.com/folder/page"
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
		}()
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"allow: /page\n" +
					"disallow: /*.htm\n"
			const url = "http://example.com/page.htm"
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url))
		}()
		func() {
			robotstxt :=
				"user-agent: FooBot\n" +
					"allow: /$\n" +
					"disallow: /\n"
			const url = "http://example.com/"
			const url_page = "http://example.com/page.html"
			expectEqual(t, true, IsUserAgentAllowed(robotstxt, "FooBot", url))
			expectEqual(t, false, IsUserAgentAllowed(robotstxt, "FooBot", url_page))
		}()
	})

	// Different kinds of line endings are all supported: %x0D / %x0A / %x0D.0A
	t.Run("ID_LinesNumbersAreCountedCorrectly", func(t *testing.T) {
		// Line :796
		report := &robotsStatsReporter{}
		const unixFile = "User-Agent: foo\n" +
			"Allow: /some/path\n" +
			"User-Agent: bar\n" +
			"\n" +
			"\n" +
			"Disallow: /\n"
		grobotstxt.Parse(unixFile, report)
		expectEqual(t, 4, report.validDirectives)
		expectEqual(t, 6, report.lastLineSeen)

		const dosFile = "User-Agent: foo\r\n" +
			"Allow: /some/path\r\n" +
			"User-Agent: bar\r\n" +
			"\r\n" +
			"\r\n" +
			"Disallow: /\r\n"
		grobotstxt.Parse(dosFile, report)
		expectEqual(t, 4, report.validDirectives)
		expectEqual(t, 6, report.lastLineSeen)

		const macFile = "User-Agent: foo\r" +
			"Allow: /some/path\r" +
			"User-Agent: bar\r" +
			"\r" +
			"\r" +
			"Disallow: /\r"
		grobotstxt.Parse(macFile, report)
		expectEqual(t, 4, report.validDirectives)
		expectEqual(t, 6, report.lastLineSeen)

		const noFinalNewline = "User-Agent: foo\n" +
			"Allow: /some/path\n" +
			"User-Agent: bar\n" +
			"\n" +
			"\n" +
			"Disallow: /"
		grobotstxt.Parse(noFinalNewline, report)
		expectEqual(t, 4, report.validDirectives)
		expectEqual(t, 6, report.lastLineSeen)

		const mixedFile = "User-Agent: foo\n" +
			"Allow: /some/path\r\n" +
			"User-Agent: bar\n" +
			"\r\n" +
			"\n" +
			"Disallow: /"
		grobotstxt.Parse(mixedFile, report)
		expectEqual(t, 4, report.validDirectives)
		expectEqual(t, 6, report.lastLineSeen)
	})

	// BOM characters are unparseable and thus skipped. The rules following the line
	// are used.
	t.Run("ID_UTF8ByteOrderMarkIsSkipped", func(t *testing.T) {
		// Line :856
		report := &robotsStatsReporter{}
		const utf8FileFullBOM = "\xEF\xBB\xBF" +
			"User-Agent: foo\n" +
			"Allow: /AnyValue\n"
		grobotstxt.Parse(utf8FileFullBOM, report)
		expectEqual(t, 2, report.validDirectives)
		expectEqual(t, 0, report.unknownDirectives)

		// We allow as well partial ByteOrderMarks.
		const utf8FilePartial2BOM = "\xEF\xBB" +
			"User-Agent: foo\n" +
			"Allow: /AnyValue\n"
		grobotstxt.Parse(utf8FilePartial2BOM, report)
		expectEqual(t, 2, report.validDirectives)
		expectEqual(t, 0, report.unknownDirectives)

		const utf8FilePartial1BOM = "\xEF" +
			"User-Agent: foo\n" +
			"Allow: /AnyValue\n"
		grobotstxt.Parse(utf8FilePartial1BOM, report)
		expectEqual(t, 2, report.validDirectives)
		expectEqual(t, 0, report.unknownDirectives)

		// If the BOM is not the right sequence, the first line looks like garbage
		// that is skipped (we essentially see "\x11\xBFUser-Agent").
		const utf8FileBrokenBOM = "\xEF\x11\xBF" +
			"User-Agent: foo\n" +
			"Allow: /AnyValue\n"
		grobotstxt.Parse(utf8FileBrokenBOM, report)
		expectEqual(t, 1, report.validDirectives)
		expectEqual(t, 1, report.unknownDirectives) // We get one broken line.

		// Some other messed up file: BOMs only valid in the beginning of the file.
		const utf8BOMSomewhereInMiddleOfFile = "User-Agent: foo\n" +
			"\xEF\xBB\xBF" +
			"Allow: /AnyValue\n"
		grobotstxt.Parse(utf8BOMSomewhereInMiddleOfFile, report)
		expectEqual(t, 1, report.validDirectives)
		expectEqual(t, 1, report.unknownDirectives)
	})

	// Google specific: the I-D allows any line that crawlers might need, such as
	// sitemaps, which Google supports.
	// See REP I-D section "Other records".
	// https://www.rfc-editor.org/rfc/rfc9309.html#section-2.2.4
	t.Run("ID_NonStandardLineExample_Sitemap", func(t *testing.T) {
		// Line :907
		report := &robotsStatsReporter{}
		func() {
			const sitemap_loc = "http://foo.bar/sitemap.xml"
			robotstxt :=
				"User-Agent: foo\n" +
					"Allow: /some/path\n" +
					"User-Agent: bar\n" +
					"\n" +
					"\n"
			robotstxt += "Sitemap: " + sitemap_loc + "\n"

			grobotstxt.Parse(robotstxt, report)
			expectEqual(t, sitemap_loc, report.sitemap)
		}()
		// A sitemap line may appear anywhere in the file.
		func() {
			robotstxt := ""
			const sitemap_loc = "http://foo.bar/sitemap.xml"
			const robotstxt_temp = "User-Agent: foo\n" +
				"Allow: /some/path\n" +
				"User-Agent: bar\n" +
				"\n" +
				"\n"
			robotstxt += "Sitemap: " + sitemap_loc + "\n" + robotstxt_temp

			grobotstxt.Parse(robotstxt, report)
			expectEqual(t, sitemap_loc, report.sitemap)
		}()
	})

}

type robotsStatsReporter struct {
	// Line :739
	lastLineSeen      int
	validDirectives   int
	unknownDirectives int
	sitemap           string
}

func (r *robotsStatsReporter) HandleRobotsStart() {
	r.lastLineSeen = 0
	r.validDirectives = 0
	r.unknownDirectives = 0
	r.sitemap = ""
}

func (r *robotsStatsReporter) HandleRobotsEnd() {}

func (r *robotsStatsReporter) HandleUserAgent(lineNum int, value string) {
	r.digest(lineNum)
}

func (r *robotsStatsReporter) HandleAllow(lineNum int, value string) {
	r.digest(lineNum)
}

func (r *robotsStatsReporter) HandleDisallow(lineNum int, value string) {
	r.digest(lineNum)
}

func (r *robotsStatsReporter) HandleSitemap(lineNum int, value string) {
	r.digest(lineNum)
	r.sitemap += value
}

func (r *robotsStatsReporter) HandleUnknownAction(lineNum int, action, value string) {
	r.lastLineSeen = lineNum
	r.unknownDirectives++
}

func (r *robotsStatsReporter) digest(lineNum int) {
	if lineNum < r.lastLineSeen {
		panic("Bad lineNum")
	}
	r.lastLineSeen = lineNum
	r.validDirectives++
}

func expectEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

type callbackPatterns struct {
	*grobotstxt.RobotsMatcher
	allow, disallow, unknown string
}

func (h *callbackPatterns) HandleAllow(_ int, s string)            { h.allow = s }
func (h *callbackPatterns) HandleDisallow(_ int, s string)         { h.disallow = s }
func (h *callbackPatterns) HandleUnknownAction(_ int, _, s string) { h.unknown = s }

type strategyPatterns struct {
	allow, disallow string
}

func (s *strategyPatterns) MatchAllow(_, pattern string) int {
	s.allow = pattern
	return -1
}

func (s *strategyPatterns) MatchDisallow(_, pattern string) int {
	s.disallow = pattern
	return -1
}

func TestRuleRepresentations(t *testing.T) {
	const pattern = "/café/%63%2f"
	const rules = "User-agent: *\nAllow: " + pattern + "\nDisallow: " + pattern + "\nOther: " + pattern
	callbacks := &callbackPatterns{RobotsMatcher: grobotstxt.NewRobotsMatcher()}
	grobotstxt.Parse(rules, callbacks)
	const escaped = "/caf%C3%A9/%63%2F"
	if callbacks.allow != escaped || callbacks.disallow != escaped || callbacks.unknown != escaped {
		t.Fatalf("public callbacks: allow %q, disallow %q, unknown %q; want %q", callbacks.allow, callbacks.disallow, callbacks.unknown, escaped)
	}

	m := grobotstxt.NewRobotsMatcher()
	strategy := &strategyPatterns{}
	m.MatchStrategy = strategy
	m.AgentAllowed(rules, "Bot", "https://example.org/")
	const normalized = "/caf%C3%A9/c%2F"
	if strategy.allow != normalized || strategy.disallow != normalized {
		t.Fatalf("strategy: allow %q, disallow %q; want %q", strategy.allow, strategy.disallow, normalized)
	}
}

func TestSitemapsOnly(t *testing.T) {
	const first = "https://example.org/café/%63.xml"
	const second = "https://example.org/two.xml"
	const rules = "Sitemap: " + first + " # comment\nUser-agent: Bot\nAllow: /é\nSite-map: " + second +
		"\nOther: /%63\nDisallow: /%2f\n# Sitemap: ignored\nSitemap: " + first
	if got := grobotstxt.Sitemaps(rules); !slices.Equal(got, []string{first, second, first}) {
		t.Errorf("sitemaps = %q, want %q, %q, %q", got, first, second, first)
	}
	if got := grobotstxt.Sitemaps("User-agent: *\nDisallow: /"); got != nil {
		t.Errorf("sitemaps = %q, want nil", got)
	}
}
