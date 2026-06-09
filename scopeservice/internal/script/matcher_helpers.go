package script

import (
	"fmt"
	"strings"
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
	return includeSubdomains && strings.HasSuffix(host, "."+seedHost)
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
