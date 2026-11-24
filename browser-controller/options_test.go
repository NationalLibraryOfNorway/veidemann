package main

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"
)

func TestCompletionTimeoutSettings(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		args                 []string
		media, drain         string
		wantMedia, wantDrain time.Duration
		invalid              bool
	}{
		{name: "defaults", wantMedia: 5 * time.Second, wantDrain: 45 * time.Second},
		{name: "environment", media: "0s", drain: "12s", wantDrain: 12 * time.Second},
		{name: "flag overrides environment", args: []string{"--media-settle-timeout=2s", "--completion-drain-timeout=8s"}, media: "1s", drain: "9s", wantMedia: 2 * time.Second, wantDrain: 8 * time.Second},
		{name: "negative media", media: "-1s", drain: "45s", invalid: true},
		{name: "zero drain", media: "5s", drain: "0s", invalid: true},
		{name: "negative drain", media: "5s", drain: "-1s", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MEDIA_SETTLE_TIMEOUT", tc.media)
			t.Setenv("COMPLETION_DRAIN_TIMEOUT", tc.drain)
			oldFlags, oldArgs := pflag.CommandLine, os.Args
			t.Cleanup(func() { pflag.CommandLine, os.Args = oldFlags, oldArgs })
			pflag.CommandLine = pflag.NewFlagSet(tc.name, pflag.ContinueOnError)
			os.Args = append([]string{"browser-controller"}, tc.args...)
			opts, err := parseFlags()
			if err != nil {
				t.Fatal(err)
			}
			if err := opts.ValidateCompletionTimeouts(); (err != nil) != tc.invalid {
				t.Fatalf("validation=%v, invalid=%v", err, tc.invalid)
			}
			if !tc.invalid && (opts.MediaSettleTimeout() != tc.wantMedia || opts.CompletionDrainTimeout() != tc.wantDrain) {
				t.Fatalf("timeouts=%v/%v, want %v/%v", opts.MediaSettleTimeout(), opts.CompletionDrainTimeout(), tc.wantMedia, tc.wantDrain)
			}
		})
	}
}
