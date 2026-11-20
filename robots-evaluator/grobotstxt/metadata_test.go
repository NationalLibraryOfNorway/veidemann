package grobotstxt

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

type metadataRecorder struct {
	sitemapExtractor
	events []string
	lines  []LineMetadata
}

func (r *metadataRecorder) HandleRobotsStart() { r.events = append(r.events, "start") }
func (r *metadataRecorder) HandleRobotsEnd()   { r.events = append(r.events, "end") }

func (r *metadataRecorder) HandleDisallow(lineNum int, value string) {
	r.events = append(r.events, fmt.Sprintf("disallow %d %s", lineNum, value))
}

func (r *metadataRecorder) ReportLineMetadata(lineNum int, metadata LineMetadata) {
	r.events = append(r.events, fmt.Sprintf("line %d", lineNum))
	r.lines = append(r.lines, metadata)
}

func TestLineMetadata(t *testing.T) {
	r := &metadataRecorder{}
	Parse("\xEF\xBB\xBF# first\r\n \rDisalow /a # note\nno valid directive\n", r)
	wantEvents := []string{"start", "line 1", "line 2", "disallow 3 /a", "line 3", "line 4", "line 5", "end"}
	wantLines := []LineMetadata{
		{HasComment: true, IsComment: true},
		{IsEmpty: true},
		{HasComment: true, HasDirective: true, IsAcceptableTypo: true, IsMissingColonSeparator: true},
		{},
		{IsEmpty: true},
	}
	if !slices.Equal(r.events, wantEvents) || !slices.Equal(r.lines, wantLines) {
		t.Fatalf("got events %v and metadata %+v", r.events, r.lines)
	}
}

func TestLineLengthMetadata(t *testing.T) {
	const limit = 2083*8 - 1
	r := &metadataRecorder{}
	Parse(strings.Repeat("x", limit)+"\r\n"+strings.Repeat("x", limit+1), r)
	if want := []LineMetadata{{}, {IsLineTooLong: true}}; !slices.Equal(r.lines, want) {
		t.Fatalf("got %+v, want %+v", r.lines, want)
	}
}

func TestGetKeyType(t *testing.T) {
	for _, tt := range []struct {
		key  string
		kind KeyType
		typo bool
	}{
		{"USER-AGENT", KeyUserAgent, false},
		{"ALLOW", KeyAllow, false},
		{"DiSalow", KeyDisallow, true},
		{"Sitemap", KeySitemap, false},
		{"site-map", KeySitemap, true},
		{"Crawl-delay", KeyUnknown, false},
	} {
		if kind, typo := GetKeyType(tt.key); kind != tt.kind || typo != tt.typo {
			t.Errorf("GetKeyType(%q) = (%v, %v), want (%v, %v)", tt.key, kind, typo, tt.kind, tt.typo)
		}
	}
}

func TestEscapedDirectiveCallback(t *testing.T) {
	r := &metadataRecorder{}
	Parse("Disallow: /%63afé%20%2f", r)
	want := []string{"start", "disallow 1 /%63af%C3%A9%20%2F", "line 1", "end"}
	if !slices.Equal(r.events, want) {
		t.Fatalf("events = %q, want %q", r.events, want)
	}
}
