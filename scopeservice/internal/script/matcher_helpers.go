package script

import (
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"
)

type seedHost struct {
	Raw  string
	Host string
}

func matchToken(actual, candidates string) bool {
	for _, token := range strings.Fields(candidates) {
		if token == actual {
			return true
		}
	}
	return false
}

func canonicalSeedHosts(seedURI, altSeeds string) ([]seedHost, error) {
	rawSeeds := append(strings.Fields(altSeeds), seedURI)
	seedHosts := make([]seedHost, 0, len(rawSeeds))
	for _, raw := range rawSeeds {
		seed, err := ScopeCanonicalizationProfile.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", raw, err)
		}
		seedHosts = append(seedHosts, seedHost{Raw: raw, Host: seed.Hostname()})
	}
	return seedHosts, nil
}

func matchSameHost(host, seedHost string, includeSubdomains bool) bool {
	if host == seedHost {
		return true
	}
	if !includeSubdomains {
		return false
	}

	anchor := scopeAnchorHost(seedHost)
	return host == anchor || strings.HasSuffix(host, "."+anchor)
}

// scopeAnchorHost removes one conventional www or numbered-www label from a
// public DNS host. The public suffix check prevents aliases such as www.co.uk
// from widening scope to an entire suffix.
func scopeAnchorHost(host string) string {
	dot := strings.IndexByte(host, '.')
	if dot < 0 || !isWwwLabel(host[:dot]) {
		return host
	}

	anchor := host[dot+1:]
	if _, err := publicsuffix.EffectiveTLDPlusOne(anchor); err != nil {
		return host
	}
	return anchor
}

func isWwwLabel(label string) bool {
	label = strings.ToLower(label)
	if label == "www" {
		return true
	}
	if !strings.HasPrefix(label, "www") {
		return false
	}
	for _, r := range label[len("www"):] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func normalizeDiscoveryPath(discoveryPath string, includeRedirects bool) string {
	if includeRedirects {
		return discoveryPath
	}
	return strings.ReplaceAll(discoveryPath, "R", "")
}

func exceedsMaxHops(discoveryPath string, maxHops int64, includeRedirects bool) bool {
	return len(normalizeDiscoveryPath(discoveryPath, includeRedirects)) > int(maxHops)
}

func matchCanonicalURL(candidate, urls string) (bool, error) {
	for _, rawURL := range strings.Fields(urls) {
		canon, err := ScopeCanonicalizationProfile.Parse(rawURL)
		if err != nil {
			return false, err
		}
		if candidate == canon.String() {
			return true, nil
		}
	}
	return false, nil
}

func matchPathPrefix(path, prefixes string) bool {
	for _, prefix := range strings.Fields(prefixes) {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
