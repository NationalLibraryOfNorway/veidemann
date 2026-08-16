package session

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestCompileBrowserWebsocketEndpointScopesRecorderCertificateTrust(t *testing.T) {
	const spki = "recorder-public-key-hash"
	sess := newSession(3,
		WithBrowserHost("browser"),
		WithBrowserPort(3000),
		WithProxyHost("recorder"),
		WithProxyPort(9900),
		WithRecorderCertificateSPKI(spki),
	)

	endpoint, err := sess.compileBrowserWebsocketEndpoint()
	if err != nil {
		t.Fatalf("compileBrowserWebsocketEndpoint() error = %v", err)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	launchJSON, err := base64.StdEncoding.DecodeString(parsed.Query().Get("launch"))
	if err != nil {
		t.Fatalf("decode launch config: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(launchJSON, &got); err != nil {
		t.Fatalf("decode launch JSON: %v", err)
	}
	if _, ok := got["acceptInsecureCerts"]; ok {
		t.Fatal("launch contains acceptInsecureCerts")
	}

	args, ok := got["args"].([]any)
	if !ok {
		t.Fatalf("launch args type = %T", got["args"])
	}
	wantScoped := "--ignore-certificate-errors-spki-list=" + spki
	foundScoped := false
	for _, value := range args {
		arg, _ := value.(string)
		if arg == "--ignore-certificate-errors" {
			t.Fatal("launch contains broad certificate-error suppression")
		}
		if arg == wantScoped {
			foundScoped = true
		}
	}
	if !foundScoped {
		t.Fatalf("launch args do not contain %q: %v", wantScoped, args)
	}
}

func TestCompileBrowserWebsocketEndpointRequiresSPKIForProxy(t *testing.T) {
	sess := newDefaultSession(WithProxyHost("recorder"))
	_, err := sess.compileBrowserWebsocketEndpoint()
	if err == nil || !strings.Contains(err.Error(), "SPKI is required") {
		t.Fatalf("compileBrowserWebsocketEndpoint() error = %v, want missing SPKI", err)
	}
}

func TestCompileBrowserWebsocketEndpointDirectHasNoCertificateException(t *testing.T) {
	sess := newDefaultSession(WithProxyHost(""))
	endpoint, err := sess.compileBrowserWebsocketEndpoint()
	if err != nil {
		t.Fatalf("compileBrowserWebsocketEndpoint() error = %v", err)
	}
	if strings.Contains(endpoint, "ignore-certificate") {
		t.Fatalf("direct endpoint contains certificate exception: %s", endpoint)
	}
}
