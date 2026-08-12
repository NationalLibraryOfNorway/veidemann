package main

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestParseFlags(t *testing.T) {
	setFlagState(t, []string{
		"--address", "127.0.0.1:50053",
		"--metrics-address", "127.0.0.1:9302",
		"--log-level", "debug",
		"--log-method",
		"--data-dir", "/mnt/oos",
	})

	if err := parseFlags(); err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	opts := Options{}
	if got := opts.Address(); got != "127.0.0.1:50053" {
		t.Errorf("Address() = %q, want %q", got, "127.0.0.1:50053")
	}
	if got := opts.TelemetryAddr(); got != "127.0.0.1:9302" {
		t.Errorf("TelemetryAddr() = %q, want %q", got, "127.0.0.1:9302")
	}
	if got := opts.LogLevel(); got != "debug" {
		t.Errorf("LogLevel() = %q, want %q", got, "debug")
	}
	if !opts.LogMethod() {
		t.Error("LogMethod() = false, want true")
	}
	if got := opts.DataDir(); got != "/mnt/oos" {
		t.Errorf("DataDir() = %q, want %q", got, "/mnt/oos")
	}
}

func TestParseFlagsUsesEnvironment(t *testing.T) {
	setFlagState(t, nil)
	t.Setenv("DATA_DIR", "/env/oos")

	if err := parseFlags(); err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	if got := (Options{}).DataDir(); got != "/env/oos" {
		t.Errorf("DataDir() = %q, want %q", got, "/env/oos")
	}
}

func setFlagState(t *testing.T, args []string) {
	t.Helper()

	oldCommandLine := pflag.CommandLine
	oldArgs := os.Args
	pflag.CommandLine = pflag.NewFlagSet(name, pflag.ContinueOnError)
	os.Args = append([]string{name}, args...)
	viper.Reset()

	t.Cleanup(func() {
		pflag.CommandLine = oldCommandLine
		os.Args = oldArgs
		viper.Reset()
	})
}
