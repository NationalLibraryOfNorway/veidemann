package recorderproxy

import (
	"net/http"
	"net/url"

	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	whatwg "github.com/nationallibraryofnorway/whatwg-url/url"
)

// discoveryIndex is protected by harvestAttempt.mu. Entries are immutable and
// independent of terminal outcomes: a streaming parent can already have children.
type discoveryIndex struct {
	byURL   map[string][]discoveryTransfer
	pending map[string][]discoveryRedirect
	main    *discoveryTransfer
}

type discoveryTransfer struct {
	id                      uint64
	uri, method, dest, path string
	main                    bool
}

type discoveryRedirect struct {
	parent discoveryTransfer
	method string
}

// wireDiscovery is called with a.mu held, as is ResolveDiscovery during Init.
// Only ObserveResponse takes the mutex itself, after downstream filters return.
func (a *harvestAttempt) wireDiscovery(rc *rpcontext.RecordContext, id uint64) {
	var transfer discoveryTransfer
	var initialized, observed bool
	rc.ResolveDiscovery = func(req *http.Request, uri *url.URL, main bool) string {
		transfer = a.discovery.admit(id, req, uri.String(), main, a.spec.DiscoveryPath)
		initialized = true
		return transfer.path
	}
	rc.ObserveResponse = func(resp *http.Response) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.closing || a.sealed || !initialized || observed {
			return
		}
		observed = true
		a.discovery.response(transfer, resp)
	}
}

func (d *discoveryIndex) admit(id uint64, req *http.Request, raw string, main bool, seed string) discoveryTransfer {
	if d.byURL == nil {
		d.byURL = make(map[string][]discoveryTransfer)
		d.pending = make(map[string][]discoveryRedirect)
	}
	t := discoveryTransfer{id: id, uri: discoveryURI(raw), method: req.Method, dest: req.Header.Get("Sec-Fetch-Dest"), main: main}
	base := seed
	if d.main != nil {
		base = d.main.path
	}
	// Only exact referrers with an unambiguous path supply a parent. In
	// particular, never expand an origin-only referrer into a same-origin guess.
	if ref := discoveryURI(req.Referer()); ref != "" {
		parents := d.byURL[ref]
		if len(parents) > 0 {
			path, agrees := parents[0].path, true
			for _, parent := range parents[1:] {
				if parent.path != path {
					agrees = false
					break
				}
			}
			if agrees {
				base = path
			}
		}
	}
	switch {
	case main && d.main == nil:
		t.path = seed
	case main && t.uri != "" && t.uri == d.main.uri:
		t.path = d.main.path // Reloads do not add a discovery hop.
	case main || t.dest == "empty":
		t.path = base + "X"
	default:
		t.path = base + "E"
	}
	// A real HTTP redirect takes precedence, including a redirect to the same
	// URL. Consume all compatible candidates; ambiguity must not survive to
	// become a spurious unique match for a later unrelated request.
	var matches []discoveryRedirect
	pending := d.pending[t.uri]
	remaining := pending[:0]
	for _, candidate := range pending {
		p := candidate.parent
		if p.main == t.main && candidate.method == t.method && (p.dest == "" || t.dest == "" || p.dest == t.dest) {
			matches = append(matches, candidate)
		} else {
			remaining = append(remaining, candidate)
		}
	}
	if len(remaining) == 0 {
		delete(d.pending, t.uri)
	} else {
		d.pending[t.uri] = remaining
	}
	if len(matches) == 1 {
		t.path = matches[0].parent.path + "R"
	}
	if t.uri != "" {
		d.byURL[t.uri] = append(d.byURL[t.uri], t)
	}
	if main {
		d.main = &t
	}
	return t
}

func (d *discoveryIndex) response(t discoveryTransfer, resp *http.Response) {
	if t.uri == "" || resp == nil || resp.Header.Get("Location") == "" {
		return
	}
	method := t.method
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound:
		if method == http.MethodPost {
			method = http.MethodGet
		}
	case http.StatusSeeOther:
		if method != http.MethodGet && method != http.MethodHead {
			method = http.MethodGet
		}
	case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
	default:
		return
	}
	target, err := whatwg.ParseRef(t.uri, resp.Header.Get("Location"))
	if err != nil {
		return
	}
	uri := discoveryURI(target.String())
	if uri != "" {
		d.pending[uri] = append(d.pending[uri], discoveryRedirect{parent: t, method: method})
	}
}

// Malformed and non-network URLs cannot establish ancestry. Valid URLs use the
// same normalization as Finish's main selection; logged URLs remain untouched.
func discoveryURI(raw string) string {
	parsed, err := whatwg.Parse(raw)
	if err != nil || (parsed.Scheme() != "http" && parsed.Scheme() != "https") {
		return ""
	}
	return networkURI(raw)
}
