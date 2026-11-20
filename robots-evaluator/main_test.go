package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/grobotstxt"
	"github.com/spf13/viper"
)

func TestValidateRobotsCacheSettings(t *testing.T) {
	tests := []struct {
		name      string
		freshness time.Duration
		retry     time.Duration
		wantError bool
	}{
		{name: "defaults", freshness: 24 * time.Hour, retry: time.Hour},
		{name: "shorter freshness", freshness: 6 * time.Hour, retry: 15 * time.Minute},
		{name: "zero freshness", retry: time.Hour, wantError: true},
		{name: "freshness over 24 hours", freshness: 24*time.Hour + time.Second, retry: time.Hour, wantError: true},
		{name: "zero retry", freshness: 24 * time.Hour, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRobotsCacheSettings(tt.freshness, tt.retry)
			if (err != nil) != tt.wantError {
				t.Fatalf("validateRobotsCacheSettings() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestRobotsCacheSettingsFromEnvironment(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	t.Setenv("ROBOTS_CACHE_FRESHNESS", "6h")
	t.Setenv("ROBOTS_UNREACHABLE_RETRY_INTERVAL", "45m")

	freshness, retry, err := robotsCacheSettingsFromViper()
	if err != nil {
		t.Fatalf("robotsCacheSettingsFromViper() returned error: %v", err)
	}
	if freshness != 6*time.Hour {
		t.Errorf("freshness = %v, want 6h", freshness)
	}
	if retry != 45*time.Minute {
		t.Errorf("retry interval = %v, want 45m", retry)
	}
}

func TestStopAfterFiveRedirects(t *testing.T) {
	for _, hops := range []int{5, 6} {
		t.Run(fmt.Sprint(hops), func(t *testing.T) {
			var reached atomic.Bool
			destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached.Store(true)
				_, _ = io.WriteString(w, "User-agent: *\nDisallow: /archive")
			}))
			defer destination.Close()
			var redirects atomic.Int32
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := int(redirects.Add(1))
				next := fmt.Sprintf("/hop/%d", n)
				if n == hops {
					next = destination.URL + "/rules"
				}
				http.Redirect(w, r, next, http.StatusFound)
			}))
			defer origin.Close()
			client := origin.Client()
			client.CheckRedirect = stopAfterFiveRedirects
			resp, err := client.Get(origin.URL + "/robots.txt")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			if hops == 6 {
				if reached.Load() || resp.StatusCode != http.StatusFound || redirects.Load() != 6 {
					t.Fatal("expected to stop before following the sixth redirect")
				}
				return
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !reached.Load() || resp.StatusCode != http.StatusOK || grobotstxt.AgentAllowed(string(body), "Bot", origin.URL+"/archive") {
				t.Fatal("expected rules reached after five redirects to deny /archive on the original origin")
			}
		})
	}
}
