package connection

import (
	"encoding/base64"
	"slices"
	"testing"
)

func TestSecureRandomString(t *testing.T) {
	first, err := secureRandomString(32)
	if err != nil {
		t.Fatal(err)
	}
	second, err := secureRandomString(32)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("independent random values unexpectedly matched")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("random value is not raw URL-safe base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("decoded length = %d, want 32", len(decoded))
	}
}

func TestOIDCScopesOfflineAccessIsOptIn(t *testing.T) {
	if slices.Contains(oidcScopes(nil, false), "offline_access") {
		t.Fatal("ordinary login unexpectedly requests offline_access")
	}
	if !slices.Contains(oidcScopes(nil, true), "offline_access") {
		t.Fatal("offline login does not request offline_access")
	}
}

func TestOIDCScopesAreConfigurable(t *testing.T) {
	configured := []string{"openid", "profile", "offline_access"}
	got := oidcScopes(configured, false)
	if !slices.Equal(got, []string{"openid", "profile"}) {
		t.Fatalf("scopes = %v, want configurable scopes without offline_access", got)
	}
	if slices.Contains(got, "audience:server:client_id:veidemann-api") {
		t.Fatal("configured scopes unexpectedly include the Dex audience scope")
	}
}

func TestRefreshTokenForLogin(t *testing.T) {
	if got, err := refreshTokenForLogin("unexpected-token", false); err != nil || got != "" {
		t.Fatalf("ordinary login retained refresh token %q, error %v", got, err)
	}
	if _, err := refreshTokenForLogin("", true); err == nil {
		t.Fatal("offline login succeeded without a refresh token")
	}
	if got, err := refreshTokenForLogin("refresh-token", true); err != nil || got != "refresh-token" {
		t.Fatalf("offline refresh token = %q, error %v", got, err)
	}
}
