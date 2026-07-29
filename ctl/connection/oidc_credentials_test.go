package connection

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestOIDCCredentialsRotateRefreshTokenOnceAcrossConcurrentClients(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer string
	var refreshCalls atomic.Int32
	var tokenMu sync.Mutex
	var refreshedIDToken string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(t, w, map[string]any{
				"issuer":                                issuer,
				"authorization_endpoint":                issuer + "/auth",
				"token_endpoint":                        issuer + "/token",
				"jwks_uri":                              issuer + "/keys",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/keys":
			writeJSON(t, w, map[string]any{"keys": []any{testJWK(&key.PublicKey)}})
		case "/token":
			refreshCalls.Add(1)
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if got := r.Form.Get("refresh_token"); got != "old-refresh" {
				t.Errorf("refresh token = %q, want old-refresh", got)
			}
			tokenMu.Lock()
			refreshedIDToken = signTestIDToken(t, key, issuer, time.Now().Add(time.Hour))
			issuedIDToken := refreshedIDToken
			tokenMu.Unlock()
			writeJSON(t, w, map[string]any{
				"access_token":  "unused-access-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "rotated-refresh",
				"id_token":      issuedIDToken,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL

	configFile := initializeTestConfig(t)
	expiredIDToken := signTestIDToken(t, key, issuer, time.Now().Add(-time.Minute))
	if err := config.SetAuthProvider(&config.AuthProvider{
		Name: config.ProviderOIDC,
		Config: config.OIDCConfig{
			ClientID:     "veidemann-cli",
			IdToken:      expiredIDToken,
			RefreshToken: "old-refresh",
			IdpIssuerUrl: issuer,
		},
	}); err != nil {
		t.Fatal(err)
	}

	first, err := newOIDCCredentials(mustOIDCConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := newOIDCCredentials(mustOIDCConfig(t))
	if err != nil {
		t.Fatal(err)
	}

	credentials := []*oidcCredentials{first, second}
	errs := make(chan error, len(credentials))
	var wg sync.WaitGroup
	for _, credential := range credentials {
		wg.Add(1)
		go func(credential *oidcCredentials) {
			defer wg.Done()
			metadata, err := credential.GetRequestMetadata(context.Background())
			tokenMu.Lock()
			expectedIDToken := refreshedIDToken
			tokenMu.Unlock()
			if err == nil && metadata["authorization"] != "Bearer "+expectedIDToken {
				err = fmt.Errorf("unexpected authorization metadata")
			}
			errs <- err
		}(credential)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := refreshCalls.Load(); got != 1 {
		t.Fatalf("refresh calls = %d, want 1", got)
	}
	stored := mustOIDCConfig(t)
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if stored.RefreshToken != "rotated-refresh" || stored.IdToken != refreshedIDToken {
		t.Fatalf("rotated credentials were not persisted: %+v", stored)
	}
	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config mode = %o, want 600", got)
	}
}

func TestOIDCCredentialsFailClosedWhenShortLivedSessionExpires(t *testing.T) {
	credentials := &oidcCredentials{
		idToken: "expired-token",
		expiry:  time.Now().Add(-time.Minute),
	}
	if _, err := credentials.GetRequestMetadata(context.Background()); err == nil {
		t.Fatal("expired short-lived session unexpectedly succeeded")
	}
}

func TestOIDCCredentialsUseUnexpiredShortLivedSession(t *testing.T) {
	credentials := &oidcCredentials{
		idToken: "current-token",
		expiry:  time.Now().Add(time.Hour),
	}
	metadata, err := credentials.GetRequestMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if metadata["authorization"] != "Bearer current-token" {
		t.Fatalf("authorization metadata = %q", metadata["authorization"])
	}
}

func initializeTestConfig(t *testing.T) string {
	t.Helper()
	viper.Reset()
	configFile := filepath.Join(t.TempDir(), "context.yaml")
	if err := os.WriteFile(configFile, []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("config", configFile, "")
	flags.String("context", "", "")
	flags.String("log-level", "error", "")
	flags.String("log-format", "json", "")
	flags.Bool("log-caller", false, "")
	if err := config.Init(flags); err != nil {
		t.Fatal(err)
	}
	return configFile
}

func mustOIDCConfig(t *testing.T) *config.OIDCConfig {
	t.Helper()
	value, err := config.GetOIDCConfig()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func signTestIDToken(t *testing.T, key *rsa.PrivateKey, issuer string, expiry time.Time) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "RS256", "kid": "test", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iss": issuer,
		"sub": "test-user",
		"aud": "veidemann-cli",
		"exp": expiry.Unix(),
		"iat": time.Now().Add(-time.Minute).Unix(),
	})
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func testJWK(key *rsa.PublicKey) map[string]any {
	exponent := big.NewInt(int64(key.E)).Bytes()
	return map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": "test",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(exponent),
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}
