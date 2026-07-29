package cmd

import "testing"

func TestRootCommandDoesNotExposeAPIKeyFlag(t *testing.T) {
	if flag := NewRootCmd().PersistentFlags().Lookup("api-key"); flag != nil {
		t.Fatal("--api-key must not be exposed")
	}
}
