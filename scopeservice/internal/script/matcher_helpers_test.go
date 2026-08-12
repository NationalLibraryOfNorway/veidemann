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
	tests := []struct {
		name              string
		host              string
		seedHost          string
		includeSubdomains bool
		want              bool
	}{
		{name: "exact host", host: "example.com", seedHost: "example.com", want: true},
		{name: "subdomain enabled", host: "sub.example.com", seedHost: "example.com", includeSubdomains: true, want: true},
		{name: "subdomain disabled", host: "sub.example.com", seedHost: "example.com", want: false},
		{name: "apex to www", host: "www.example.com", seedHost: "example.com", includeSubdomains: true, want: true},
		{name: "www to apex", host: "example.com", seedHost: "www.example.com", includeSubdomains: true, want: true},
		{name: "numbered www to sibling", host: "news.example.com", seedHost: "www2.example.com", includeSubdomains: true, want: true},
		{name: "www exact mode", host: "example.com", seedHost: "www.example.com", want: false},
		{name: "unrelated domain", host: "other.example", seedHost: "www.example.com", includeSubdomains: true, want: false},
		{name: "suffix lookalike", host: "notexample.com", seedHost: "example.com", includeSubdomains: true, want: false},
		{name: "non www label", host: "example.com", seedHost: "wwwx.example.com", includeSubdomains: true, want: false},
		{name: "public suffix", host: "co.uk", seedHost: "www.co.uk", includeSubdomains: true, want: false},
		{name: "remove one www label", host: "news.www2.example.com", seedHost: "www1.www2.example.com", includeSubdomains: true, want: true},
		{name: "do not remove two www labels", host: "example.com", seedHost: "www1.www2.example.com", includeSubdomains: true, want: false},
		{name: "ip exact", host: "192.0.2.1", seedHost: "192.0.2.1", includeSubdomains: true, want: true},
		{name: "ip different", host: "192.0.2.2", seedHost: "192.0.2.1", includeSubdomains: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchSameHost(tt.host, tt.seedHost, tt.includeSubdomains); got != tt.want {
				t.Fatalf("matchSameHost(%q, %q, %v) = %v, want %v", tt.host, tt.seedHost, tt.includeSubdomains, got, tt.want)
			}
		})
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
