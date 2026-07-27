package script

import (
	"github.com/nlnwa/whatwg-url/canonicalizer"
	"github.com/nlnwa/whatwg-url/url"
)

var ScopeCanonicalizationProfile url.Parser
var CrawlCanonicalizationProfile url.Parser

func InitializeCanonicalizationProfiles(includeFragment bool) {
	opts := []url.ParserOption{
		url.WithCollapseConsecutiveSlashes(),
		url.WithSkipEqualsForEmptySearchParamsValue(),
		canonicalizer.WithRemoveUserInfo(),
		canonicalizer.WithRepeatedPercentDecoding(),
		canonicalizer.WithSortQuery(canonicalizer.SortKeys),
		canonicalizer.WithDefaultScheme("http"),
	}
	if !includeFragment {
		opts = append(opts, canonicalizer.WithRemoveFragment())
	}
	ScopeCanonicalizationProfile = canonicalizer.New(opts...)

	opts = []url.ParserOption{
		url.WithCollapseConsecutiveSlashes(),
		url.WithSkipEqualsForEmptySearchParamsValue(),
		canonicalizer.WithRemoveUserInfo(),
		canonicalizer.WithSortQuery(canonicalizer.SortKeys),
		canonicalizer.WithDefaultScheme("http"),
	}
	if !includeFragment {
		opts = append(opts, canonicalizer.WithRemoveFragment())
	}
	CrawlCanonicalizationProfile = canonicalizer.New(opts...)
}
