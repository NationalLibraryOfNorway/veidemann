package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

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
	if err := stopAfterFiveRedirects(nil, make([]*http.Request, 4)); err != nil {
		t.Fatalf("four redirects returned error: %v", err)
	}
	if err := stopAfterFiveRedirects(nil, make([]*http.Request, 5)); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("five redirects returned %v, want http.ErrUseLastResponse", err)
	}
}
