package main

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestOptionsValidateModes(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		address string
		wantErr string
	}{
		{name: "combined", mode: modeCombined},
		{name: "recent", mode: modeRecent},
		{name: "writer archive only", mode: modeWriter},
		{name: "writer forwarding", mode: modeWriter, address: "log-service:8080"},
		{name: "invalid mode", mode: "unknown", wantErr: "mode must be one of"},
		{name: "forwarding outside writer", mode: modeRecent, address: "log-service:8080", wantErr: "only valid in writer mode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidOptionValues(t, tt.mode)
			viper.Set("recent-log-service-address", tt.address)
			err := (Options{}).Validate()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("expected valid options, got %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestWriterOptionsRequireBoundedForwardingSettings(t *testing.T) {
	setValidOptionValues(t, modeWriter)
	viper.Set("recent-forward-queue-size", 0)
	if err := (Options{}).Validate(); err == nil || !strings.Contains(err.Error(), "queue-size") {
		t.Fatalf("expected queue-size validation error, got %v", err)
	}
}

func setValidOptionValues(t *testing.T, mode string) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("mode", mode)
	viper.Set("parquet-dir", t.TempDir())
	viper.Set("recent-log-db-path", t.TempDir()+"/logs.db")
	viper.Set("recent-forward-queue-size", 1024)
	viper.Set("recent-forward-workers", 2)
	viper.Set("recent-forward-timeout", 5*time.Second)
	viper.Set("recent-forward-shutdown-timeout", 30*time.Second)
}
