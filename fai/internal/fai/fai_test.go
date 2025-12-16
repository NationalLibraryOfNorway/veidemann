package fai

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	// Test glob pattern is invalid
	_, err := New(
		WithGlobPattern("["),
	)
	if err == nil {
		t.Error("expected error when glob pattern is invalid")
	}
}

func TestRun(t *testing.T) {
	sourceDir := t.TempDir()
	var testFiles []*os.File

	for range 10 {
		f, _ := os.CreateTemp(sourceDir, "testfile")
		testFiles = append(testFiles, f)
	}

	fai, err := New(
		WithSleep(0), // single pass
		WithSourceDir(sourceDir),
		WithInspector(func(_ context.Context, path string) error {
			return os.Remove(path)
		}),
		WithGlobPattern("testfile*"),
	)
	if err != nil {
		t.Fatalf("failed to create fai: %v", err)
	}

	// run fai (add files to queue)
	err = fai.Run(context.Background())
	if err != nil {
		t.Fatalf("failed to run fai: %v", err)
	}

	for _, testFile := range testFiles {
		// assert that test file is removed (as per worker function)
		_, err = os.Stat(testFile.Name())
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("test file still exists: %s", testFile.Name())
		}
	}
}
