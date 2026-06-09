package script

import "testing"

func Test_matchToken(t *testing.T) {
	if !matchToken("http", "https http") {
		t.Fatalf("expected token match")
	}
	if matchToken("ftp", "https http") {
		t.Fatalf("unexpected token match")
	}
}

func Test_canonicalSeedHosts(t *testing.T) {
	InitializeCanonicalizationProfiles(false)

	got, err := canonicalSeedHosts("https://seed.example/path", "http://alt.example")
	if err != nil {
		t.Fatalf("canonicalSeedHosts() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("canonicalSeedHosts() len = %d, want 2", len(got))
	}
	if got[0].Host != "alt.example" || got[1].Host != "seed.example" {
		t.Fatalf("canonicalSeedHosts() hosts = %#v", got)
	}
}

func Test_matchSameHost(t *testing.T) {
	if !matchSameHost("example.com", "example.com", false) {
		t.Fatalf("expected exact host match")
	}
	if !matchSameHost("sub.example.com", "example.com", true) {
		t.Fatalf("expected subdomain match")
	}
	if matchSameHost("sub.example.com", "example.com", false) {
		t.Fatalf("unexpected subdomain match")
	}
}

func Test_exceedsMaxHops(t *testing.T) {
	if !exceedsMaxHops("RLERLR", 2, false) {
		t.Fatalf("expected max hops to be exceeded without redirects")
	}
	if exceedsMaxHops("RLERLR", 4, false) {
		t.Fatalf("unexpected max hops exceedance without redirects")
	}
	if !exceedsMaxHops("RLERLR", 3, true) {
		t.Fatalf("expected max hops to be exceeded with redirects")
	}
}

func Test_matchCanonicalURL(t *testing.T) {
	InitializeCanonicalizationProfiles(false)

	matched, err := matchCanonicalURL(
		"http://foo.bar/aa/bb/cc?a=c&a=b&foo&jsessionid=1",
		"foo.bar/aa/ff/../bb/cc?foo&a=c&jsessionid=1&a=b",
	)
	if err != nil {
		t.Fatalf("matchCanonicalURL() error = %v", err)
	}
	if !matched {
		t.Fatalf("expected canonical URL match")
	}
	matched, err = matchCanonicalURL("http://foo.bar/aa/", "http://foo.bar/bb/")
	if err != nil {
		t.Fatalf("matchCanonicalURL() error = %v", err)
	}
	if matched {
		t.Fatalf("unexpected canonical URL match")
	}
}

func Test_matchPathPrefix(t *testing.T) {
	if !matchPathPrefix("/aa%20bb/cc", "/zz /aa%20bb") {
		t.Fatalf("expected path prefix match")
	}
	if matchPathPrefix("/aa%20bb/cc", "/zz /bb") {
		t.Fatalf("unexpected path prefix match")
	}
}
