// Copyright 2026 National Library of Norway.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerdictAndExitCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "robots.txt")
	if err := os.WriteFile(path, []byte("User-agent: Bot\nDisallow: /books"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		uri     string
		code    int
		verdict string
	}{
		{"https://example.org/", 0, "ALLOWED"},
		{"https://example.org/books", 1, "DISALLOWED"},
		{"relative", 2, ""},
		{"data:text/plain,books", 2, ""},
	} {
		var stdout, stderr bytes.Buffer
		code := run([]string{"icanhasrobot", path, "Reader,Bot/1", tt.uri}, &stdout, &stderr)
		var wantOut string
		if tt.verdict != "" {
			wantOut = "user-agent 'Reader,Bot/1' with URI '" + tt.uri + "': " + tt.verdict + "\n"
		}
		if code != tt.code || stdout.String() != wantOut {
			t.Errorf("URI %q: exit %d, stdout %q; want %d, %q", tt.uri, code, &stdout, tt.code, wantOut)
		}
		if tt.code == 2 {
			if !strings.Contains(stderr.String(), "invalid URL") {
				t.Errorf("URI %q: stderr = %q", tt.uri, &stderr)
			}
		} else if stderr.Len() != 0 {
			t.Errorf("URI %q: unexpected stderr %q", tt.uri, &stderr)
		}
	}
}
