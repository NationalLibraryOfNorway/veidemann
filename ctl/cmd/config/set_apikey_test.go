package config

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadAPIKeyFromPipe(t *testing.T) {
	got, err := readAPIKey(strings.NewReader("secret-value\r\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "secret-value" {
		t.Fatalf("API key = %q, want %q", got, "secret-value")
	}
}

func TestReadAPIKeyRejectsUnsafeInput(t *testing.T) {
	tests := []string{"", "\n", "first\nsecond", "secret\x00value", "secret\n\n"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := readAPIKey(strings.NewReader(input), &bytes.Buffer{}); err == nil {
				t.Fatal("expected input to be rejected")
			}
		})
	}
}
