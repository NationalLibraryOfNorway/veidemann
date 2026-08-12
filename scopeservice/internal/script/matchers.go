package script

import (
	"errors"
	"fmt"
	"strings"

	"go.starlark.net/starlark"
)

func init() {
	starlark.Universe["test"] = starlark.NewBuiltin("test", test)
	starlark.Universe["isScheme"] = starlark.NewBuiltin("isScheme", isScheme)
	starlark.Universe["isSameHost"] = starlark.NewBuiltin("isSameHost", isSameHost)
	starlark.Universe["maxHopsFromSeed"] = starlark.NewBuiltin("maxHopsFromSeed", maxHopsFromSeed)
	starlark.Universe["isUrl"] = starlark.NewBuiltin("isUrl", isUrl)
	starlark.Universe["isPathPrefix"] = starlark.NewBuiltin("isPathPrefix", isPathPrefix)
	starlark.Universe["isReferrer"] = starlark.NewBuiltin("isReferrer", isReferrer)
}

func test(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var m starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "match", &m); err != nil {
		return nil, err
	}
	match := Match(parameterAsBool(m))
	printDebug(thread, b, args, kwargs, fmt.Sprintf("match=%v", match))
	return match, nil
}

func isScheme(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var scheme string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &scheme); err != nil {
		return nil, err
	}
	qUrl, ok := thread.Local(urlKey).(*UrlValue)
	if !ok {
		return nil, fmt.Errorf("url not set")
	}
	s := strings.TrimRight(qUrl.parsedUri.Protocol(), ":")
	scheme = strings.ToLower(scheme)
	match := Match(matchToken(s, scheme))

	printDebugf(thread, b, args, kwargs, "scheme=%v, wantScheme=%v, match=%v", s, scheme, match)

	return match, nil
}

func isReferrer(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var referrer string
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &referrer); err != nil {
		return nil, err
	}
	qUrl, ok := thread.Local(urlKey).(*UrlValue)
	if !ok {
		return nil, fmt.Errorf("url not set")
	}
	s := strings.TrimSpace(qUrl.qUri.Referrer)
	referrer = strings.ToLower(referrer)
	match := Match(matchToken(s, referrer))

	printDebugf(thread, b, args, kwargs, "referrer=%v, wantReferrer=%v, match=%v", s, referrer, match)

	return match, nil
}

func isSameHost(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var includeSubdomains starlark.Value
	var altSeeds string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "includeSubdomains?", &includeSubdomains, "altSeeds?", &altSeeds); err != nil {
		return nil, err
	}

	qUrl := thread.Local(urlKey).(*UrlValue)
	host := qUrl.parsedUri.Hostname()

	seedHosts, err := canonicalSeedHosts(qUrl.qUri.SeedUri, altSeeds)
	if err != nil {
		seedText := strings.SplitN(err.Error(), ":", 2)[0]
		printDebugf(thread, b, args, kwargs, "Could not parse seed '%v'", seedText)
		return nil, IllegalUri.asError(fmt.Sprintf("Could not parse seed '%v'", seedText))
	}

	includeSubs := parameterAsBool(includeSubdomains)
	for _, seed := range seedHosts {
		match := matchSameHost(host, seed.Host, includeSubs)
		anchor := seed.Host
		if includeSubs {
			anchor = scopeAnchorHost(seed.Host)
		}
		printDebugf(thread, b, args, kwargs, "host=%v, seedHost=%v, scopeAnchor=%v, match=%v", host, seed.Host, anchor, match)
		if match {
			return True, nil
		}
	}

	return False, nil
}

func maxHopsFromSeed(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var maxHops starlark.Value
	var includeRedirects starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "hops", &maxHops, "includeRedirects?", &includeRedirects); err != nil {
		return nil, err
	}
	qUrl := thread.Local(urlKey).(*UrlValue)
	discoveryPath := qUrl.qUri.GetDiscoveryPath()

	var match bool

	if h, err := parameterAsInt64(maxHops); err == nil {
		match = exceedsMaxHops(qUrl.qUri.GetDiscoveryPath(), h, parameterAsBool(includeRedirects))
		discoveryPath = normalizeDiscoveryPath(discoveryPath, parameterAsBool(includeRedirects))
	} else {
		if errors.Is(err, None) {
			return nil, err
		}
	}
	printDebugf(thread, b, args, kwargs, "discoveryPath=%v, hops=%v, match=%v", discoveryPath, len(discoveryPath), match)
	return Match(match), nil
}

func isUrl(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var u string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "url", &u); err != nil {
		return nil, err
	}
	qUrl := thread.Local(urlKey).(*UrlValue)

	matched, err := matchCanonicalURL(qUrl.String(), u)
	if err != nil {
		return nil, err
	}
	match := Match(matched)

	printDebugf(thread, b, args, kwargs, "test='%v', url=%v, match=%v", u, qUrl.String(), match)

	return match, nil
}

func isPathPrefix(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var prefixes string
	if err := starlark.UnpackArgs(b.Name(), args, kwargs, "prefix", &prefixes); err != nil {
		return nil, err
	}
	qUrl := thread.Local(urlKey).(*UrlValue)
	path := qUrl.parsedUri.Pathname()

	match := Match(matchPathPrefix(path, prefixes))

	printDebugf(thread, b, args, kwargs, "path=%v, prefixes=%v, match=%v", path, prefixes, match)

	return match, nil
}
