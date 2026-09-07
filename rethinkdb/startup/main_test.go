package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type processReport struct {
	PID    int
	Args   []string
	Marker string
}

// The test executable also stands in for rethinkdb, allowing us to verify an
// actual exec replacement, inherited environment, argv, and exit status.
func TestMain(m *testing.M) {
	if os.Getenv("STARTUP_EXEC_TEST") == "1" {
		if os.Args[0] == "rethinkdb" {
			_ = json.NewEncoder(os.Stdout).Encode(processReport{os.Getpid(), os.Args, os.Getenv("STARTUP_EXEC_MARKER")})
			os.Exit(23)
		}
		os.Args = []string{"rethinkdb-startup", "--server-tag", "tag with spaces"}
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestProcessReplacement(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(executable, filepath.Join(dir, "rethinkdb")); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable)
	cmd.Env = []string{"PATH=" + dir, "STARTUP_EXEC_TEST=1", "STARTUP_EXEC_MARKER=preserved",
		"POD_NAME=rethinkdb-1", "RETHINKDB_SEEDS=peer", "RETHINKDB_PASSWORD=secret-for-exec-test"}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if err == nil || cmd.ProcessState.ExitCode() != 23 {
		t.Fatalf("exit=%v stderr=%s", err, &stderr)
	}
	var report processReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.PID != cmd.Process.Pid || report.Marker != "preserved" {
		t.Fatalf("exec did not preserve process/environment: %+v", report)
	}
	if !slices.Equal(report.Args[len(report.Args)-2:], []string{"--server-tag", "tag with spaces"}) {
		t.Fatalf("argv=%q", report.Args)
	}
	if strings.Contains(stderr.String(), "secret-for-exec-test") {
		t.Fatal("password leaked to startup logs")
	}
	if !strings.Contains(stderr.String(), `level=INFO msg="Starting RethinkDB"`) {
		t.Fatalf("expected the standard slog text format, got %s", &stderr)
	}
}
