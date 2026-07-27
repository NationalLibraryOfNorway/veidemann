/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestHandler_Handle(t *testing.T) {
	testDir := t.TempDir()

	oos, err := NewHandler(testDir)
	if err != nil {
		t.Fatalf("Could not create OOS handler: %v", err)
	}

	u, _, _ := oos.parseUriAndGroup("http://example1.com")
	oos.bloomContains(u)

	type args struct {
		uri string
	}
	tests := []struct {
		name       string
		o          *Handler
		args       args
		wantExists bool
	}{
		{"1-1", oos, args{"http://example1.com"}, false},
		{"1-2", oos, args{"http://example2.com"}, false},
		{"1-3", oos, args{"http://example3.com"}, false},
		{"1-4", oos, args{"http://example2.com"}, true},
		{"1-5", oos, args{"http://example1.com"}, true},
		{"1-6", oos, args{"http://example3.com"}, true},
		{"2-1", oos, args{"http://example1.no"}, false},
		{"2-2", oos, args{"http://example2.no"}, false},
		{"3-1", oos, args{"http://bølle.no"}, false},
		{"3-2", oos, args{"http://bolle.no"}, false},
		{"3-3", oos, args{"https://bømållag.no"}, false},
		{"4-1", oos, args{"http://191.69.10.1"}, false},
		{"4-2", oos, args{"http://127.0.0.1"}, false},
		{"5-1", oos, args{"http://www.mindandlife.org%20/"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.o
			exists := o.Handle(tt.args.uri)
			if exists != tt.wantExists {
				t.Errorf("Expected %v, but got %v", tt.wantExists, exists)
			}
		})
	}

	fileValueTests := []struct {
		filename string
		values   []string
	}{
		{"uri_com_ex.txt", []string{"http://example1.com", "http://example2.com", "http://example3.com"}},
		{"uri_no_ex.txt", []string{"http://example1.no", "http://example2.no"}},
		{"uri_no_bo.txt", []string{"http://bølle.no", "http://bolle.no", "https://bømållag.no"}},
	}
	for _, ff := range fileValueTests {
		t.Run(ff.filename, func(t *testing.T) {
			f, err := os.Open(filepath.Join(testDir, ff.filename))
			if err != nil {
				t.Errorf("Could not open file '%v'", ff.filename)
			}
			defer func() { _ = f.Close() }()

			i := 0
			buf := bufio.NewReader(f)
			for {
				l, err := buf.ReadString('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("Error reading from file: %v", err)
					break
				}

				line := strings.Trim(l, "\n")
				if line == "" {
					break
				}
				if line != ff.values[i] {
					t.Errorf("Expected '%v', but got '%v'", ff.values[i], line)
				}
				i++
			}

			if i != len(ff.values) {
				t.Errorf("Expected %v values in file '%v', got %v", len(ff.values), ff.filename, i)
			}
		})
	}
}

const testDataDir = "testdata"

func TestOosHandler_Import(t *testing.T) {
	t.Parallel()

	oos, err := NewHandler(filepath.Join(testDataDir, "preimport"))
	if err != nil {
		t.Fatalf("NewHandler() failed: %v", err)
	}

	lines := readNonEmptyLines(t, filepath.Join(testDataDir, "seeds.txt"))

	var wg sync.WaitGroup
	results := make(chan bool, len(lines))

	for _, line := range lines {
		wg.Go(func() {
			results <- oos.Handle(line)
		})
	}

	wg.Wait()
	close(results)

	var duplicates int
	for exists := range results {
		if exists {
			duplicates++
		}
	}

	if duplicates != len(lines) {
		t.Fatalf("Handle() reported %d/%d pre-imported URIs; want all", duplicates, len(lines))
	}
}

func readNonEmptyLines(t *testing.T, path string) []string {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("os.Open(%q) failed: %v", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("closing %q failed: %v", path, err)
		}
	}()

	var lines []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("reading %q failed: %v", path, err)
	}

	if len(lines) == 0 {
		t.Fatalf("%q contained no non-empty lines", path)
	}

	return lines
}
