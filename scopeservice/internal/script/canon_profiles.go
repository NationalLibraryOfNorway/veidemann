package script

import (
	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

var ScopeCanonicalizationProfile url.Parser
var CrawlCanonicalizationProfile url.Parser

func InitializeCanonicalizationProfiles(includeFragment bool) {
	parser := url.NewParser(url.WithCollapseConsecutiveSlashes())
	rules := []canonicalizer.Rule{canonicalizer.RemoveUserInfo}
	if !includeFragment {
		rules = append(rules, canonicalizer.RemoveFragment)
	}
	// Sort serialized query terms without form-decoding them, preserving literal
	// '+' and empty terms. Retain the policy that "flag=" canonicalizes to "flag".
	rules = append(rules, canonicalizer.SortQuery, canonicalizer.RemoveEmptyQueryEquals)
	CrawlCanonicalizationProfile = canonicalizer.New(rules...).
		WithParser(parser).WithDefaultScheme("https")
	ScopeCanonicalizationProfile = canonicalizer.New(canonicalizer.RepeatedPercentDecoding()).
		Then(rules...).WithParser(parser).WithDefaultScheme("https")
}
