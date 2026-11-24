//go:build integration

package server

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
)

func TestDiscoveryAncestry(t *testing.T) {
	for _, scenario := range []string{"fragment", "redirect-chain", "nested", "redirected-style", "history", "no-referrer", "client-navigation", "reload", "http-script", "https-script"} {
		t.Run(scenario, func(t *testing.T) {
			mainPath := "L"
			expected := map[string]string{"/index": "L"}
			switch scenario {
			case "redirect-chain":
				mainPath = "LRR"
				expected["/middle"] = "LR"
				expected["/final"] = "LRR"
			case "nested":
				expected["/child"] = "LE"
				expected["/style.css"] = "LEE"
				expected["/image.svg"] = "LEEE"
			case "redirected-style":
				expected["/style.css"] = "LE"
				expected["/sheet.css"] = "LER"
				expected["/image.svg"] = "LERE"
			case "history", "no-referrer":
				mainPath = "LR"
				expected["/final"] = "LR"
				expected["/child"] = "LRE"
				expected["/image.svg"] = "LRE"
				if scenario == "history" {
					expected["/image.svg"] = "LREE"
				}
			case "client-navigation":
				mainPath = "LX"
				expected["/final"] = "LX"
			case "http-script":
				expected["/fetch"] = "LE"
				expected["/image.svg"] = "LE"
			case "https-script":
				expected["/fetch"] = "LX"
				expected["/image.svg"] = "LE"
			}
			origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "no-store")
				if scenario == "no-referrer" {
					w.Header().Set("Referrer-Policy", "no-referrer")
				}
				w.Header().Set("Content-Type", "text/html")
				switch r.URL.Path {
				case "/image.svg":
					w.Header().Set("Content-Type", "image/svg+xml")
					fmt.Fprint(w, `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><rect width="4" height="4"/></svg>`)
					return
				case "/fetch":
					fmt.Fprint(w, "ok")
					return
				case "/style.css", "/sheet.css":
					if scenario == "redirected-style" && r.URL.Path == "/style.css" {
						http.Redirect(w, r, "/sheet.css", 302)
						return
					}
					w.Header().Set("Content-Type", "text/css")
					fmt.Fprint(w, `body { background-image:url('/image.svg'); }`)
					return
				case "/child":
					if scenario == "nested" {
						fmt.Fprint(w, `<html><link rel="stylesheet" href="/style.css"><body>Child</body></html>`)
					} else {
						fmt.Fprint(w, `<html><img src="/image.svg"></html>`)
					}
					return
				case "/index":
					switch scenario {
					case "redirect-chain":
						http.Redirect(w, r, "middle", 302)
						return
					case "history", "no-referrer":
						http.Redirect(w, r, "/final", 302)
						return
					case "client-navigation":
						fmt.Fprint(w, `<html><script>location.href='/final'</script></html>`)
						return
					}
				case "/middle":
					http.Redirect(w, r, "/final#section", 302)
					return
				case "/final":
				default:
					w.WriteHeader(204)
					return
				}
				fmt.Fprint(w, `<html><head>`)
				if scenario == "no-referrer" {
					fmt.Fprint(w, `<meta name="referrer" content="no-referrer">`)
				}
				fmt.Fprint(w, `</head><body><a href="/link">Link</a>`)
				switch scenario {
				case "nested":
					fmt.Fprint(w, `<iframe src="/child"></iframe>`)
				case "redirected-style":
					fmt.Fprint(w, `<link rel="stylesheet" href="/style.css">`)
				case "history":
					fmt.Fprint(w, `<script>history.replaceState({},'', '/new-route?q=1');</script><iframe src="/child"></iframe>`)
				case "no-referrer":
					fmt.Fprint(w, `<iframe src="/child"></iframe>`)
				case "reload":
					fmt.Fprint(w, `<script>if (!sessionStorage.reloaded) {sessionStorage.reloaded='yes';location.reload();}</script>`)
				case "http-script", "https-script":
					fmt.Fprint(w, `<script>fetch('/fetch'); let image = new Image(); image.src='/image.svg'; document.body.appendChild(image);</script>`)
				}
				fmt.Fprint(w, `</body></html>`)
			}))
			_ = origin.Listener.Close()
			var err error
			origin.Listener, err = net.Listen("tcp", "0.0.0.0:0")
			if err != nil {
				t.Fatal(err)
			}
			scheme := "http"
			if scenario == "https-script" {
				scheme = "https"
				origin.StartTLS()
			} else {
				origin.Start()
			}
			defer origin.Close()
			_, port, _ := net.SplitHostPort(origin.Listener.Addr().String())
			base := fmt.Sprintf("%s://%s:%s", scheme, localhost, port)
			requested := base + "/index"
			if scenario == "fragment" {
				requested += "#section"
			}
			conf := &configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{
				BrowserConfigRef: &configV1.ConfigRef{Id: "browserConfig1"}, PolitenessRef: &configV1.ConfigRef{Id: "politenessConfig1"}, CollectionRef: &configV1.ConfigRef{Id: "collectionConfig1"}, Extra: &configV1.ExtraConfig{},
			}}}
			sess := session.New(1, sessionOptions...)
			result, err := sess.Fetch(t.Context(), &frontierV1.PageHarvestSpec{QueuedUri: &frontierV1.QueuedUri{Uri: requested, DiscoveryPath: "L", ExecutionId: "eid", JobExecutionId: "jid"}, CrawlConfig: conf})
			if err != nil {
				t.Fatal(err)
			}
			page := recordedPages.latest(requested)
			if page == nil {
				t.Fatal("missing page log")
			}
			for path, want := range expected {
				found := false
				for _, resource := range page.Resource {
					if resource.Uri == base+path {
						found = true
						if resource.DiscoveryPath != want {
							t.Errorf("resource %s path=%s want=%s", path, resource.DiscoveryPath, want)
						}
					}
				}
				if !found {
					t.Errorf("missing page resource %s", path)
				}
				recordedPages.mu.Lock()
				found = false
				for _, cl := range recordedPages.crawls {
					if cl.RequestedUri == base+path {
						found = true
						if cl.DiscoveryPath != want {
							t.Errorf("crawl %s path=%s want=%s", path, cl.DiscoveryPath, want)
						}
					}
				}
				recordedPages.mu.Unlock()
				if !found {
					t.Errorf("missing persisted crawl %s", path)
				}
			}
			if !hasOutlink(result.Outlinks, base+"/link") {
				t.Fatalf("missing link: %v", result)
			}
			for _, link := range result.Outlinks {
				if link.DiscoveryPath != mainPath+"L" {
					t.Errorf("outlink path=%s want=%s", link.DiscoveryPath, mainPath+"L")
				}
			}
		})
	}
}
