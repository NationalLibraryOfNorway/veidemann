package script

import (
	"testing"

	"github.com/nationallibraryofnorway/whatwg-url/url"
)

func TestCanonicalizationProfiles(t *testing.T) {
	t.Cleanup(func() { InitializeCanonicalizationProfiles(false) })
	InitializeCanonicalizationProfiles(false)
	tests := []struct {
		name, input, scope, crawl string
	}{
		{
			name:  "default scheme and slash collapsing",
			input: "Example.COM/a//b", scope: "http://example.com/a/b", crawl: "http://example.com/a/b",
		},
		{
			name:  "credentials fragment and default port",
			input: "https://user:secret@Example.COM:443/a#fragment", scope: "https://example.com/a", crawl: "https://example.com/a",
		},
		{
			name:  "repeated decoding only for scope",
			input: "https://example.com/%2541?b=%2542&a=1", scope: "https://example.com/A?a=1&b=B", crawl: "https://example.com/%2541?a=1&b=%2542",
		},
		{
			name:  "stable duplicate keys bare and empty values",
			input: "https://example.com/?b=2&flag=&a=z&flag&a=a", scope: "https://example.com/?a=z&a=a&b=2&flag&flag", crawl: "https://example.com/?a=z&a=a&b=2&flag&flag",
		},
		{
			name:  "empty values become bare parameters",
			input: "https://example.com/?flag=&flag&x=", scope: "https://example.com/?flag&flag&x", crawl: "https://example.com/?flag&flag&x",
		},
		{
			name:  "empty names and terms retain query presence",
			input: "https://example.com/?=&flag=&&", scope: "https://example.com/?&&&flag", crawl: "https://example.com/?&&&flag",
		},
		{
			name:  "empty name and value",
			input: "https://example.com/?=", scope: "https://example.com/?", crawl: "https://example.com/?",
		},
		{
			name:  "nonempty values retain equals",
			input: "https://example.com/?a=0&b=+&c=%20&d=%3D&=value", scope: "https://example.com/?=value&a=0&b=+&c=%20&d=%3D", crawl: "https://example.com/?=value&a=0&b=+&c=%20&d=%3D",
		},
		{
			name:  "empty query terms",
			input: "https://example.com/?b=2&&a=1&", scope: "https://example.com/?&&a=1&b=2", crawl: "https://example.com/?&&a=1&b=2",
		},
		{
			name:  "literal and encoded plus",
			input: "https://example.com/?q=a+b%2Bc", scope: "https://example.com/?q=a+b+c", crawl: "https://example.com/?q=a+b%2Bc",
		},
		{
			name:  "encoded delimiters remain within query terms",
			input: "https://example.com/?b=%2526%253D&a=1", scope: "https://example.com/?a=1&b=%26%3D", crawl: "https://example.com/?a=1&b=%2526%253D",
		},
		{
			name:  "malformed UTF-8 bytes stay escaped",
			input: "https://example.com/?b=%FC&a=%FF", scope: "https://example.com/?a=%FF&b=%FC", crawl: "https://example.com/?a=%FF&b=%FC",
		},
		{
			name:  "present empty query",
			input: "https://example.com/?", scope: "https://example.com/?", crawl: "https://example.com/?",
		},
		{
			name:  "absent query",
			input: "https://example.com/", scope: "https://example.com/", crawl: "https://example.com/",
		},
		{
			name:  "raw query key sorting",
			input: "https://example.com/?b=1&%61=2", scope: "https://example.com/?a=2&b=1", crawl: "https://example.com/?%61=2&b=1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, profile := range []struct {
				name   string
				parser url.Parser
				want   string
			}{
				{"scope", ScopeCanonicalizationProfile, tt.scope},
				{"crawl", CrawlCanonicalizationProfile, tt.crawl},
			} {
				t.Run(profile.name, func(t *testing.T) {
					got, err := profile.parser.Parse(tt.input)
					if err != nil {
						t.Fatal(err)
					}
					if got.String() != profile.want {
						t.Fatalf("Parse(%q) = %q, want %q", tt.input, got.String(), profile.want)
					}
				})
			}
		})
	}
}

func TestCanonicalizationProfilesFragments(t *testing.T) {
	t.Cleanup(func() { InitializeCanonicalizationProfiles(false) })
	for _, includeFragment := range []bool{false, true} {
		InitializeCanonicalizationProfiles(includeFragment)
		for _, parser := range []url.Parser{ScopeCanonicalizationProfile, CrawlCanonicalizationProfile} {
			for _, fragment := range []string{"#", "#section"} {
				got, err := parser.Parse("https://user:secret@example.com/" + fragment)
				if err != nil {
					t.Fatal(err)
				}
				want := "https://example.com/"
				if includeFragment {
					want += fragment
				}
				if got.String() != want {
					t.Fatalf("includeFragment=%v: got %q, want %q", includeFragment, got.String(), want)
				}
			}
		}
	}
}

func TestCanonicalizationProfilesRejectInvalidURL(t *testing.T) {
	InitializeCanonicalizationProfiles(false)
	for _, parser := range []url.Parser{ScopeCanonicalizationProfile, CrawlCanonicalizationProfile} {
		for _, input := range []string{"http://[invalid]/", "http://example.com:99999/"} {
			if _, err := parser.Parse(input); err == nil {
				t.Errorf("Parse(%q) succeeded, want an error", input)
			}
		}
	}
}
