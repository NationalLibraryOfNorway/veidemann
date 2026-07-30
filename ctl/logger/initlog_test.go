/*
 * Copyright 2026 National Library of Norway.
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

package logger

import (
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestInitLoggerDoesNotLog(t *testing.T) {
	oldStderr := os.Stderr
	oldLogger := slog.Default()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = oldStderr
		slog.SetDefault(oldLogger)
	})

	InitLogger("info", "pretty", false)

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if len(output) != 0 {
		t.Fatalf("InitLogger() wrote %q to stderr", output)
	}
}
