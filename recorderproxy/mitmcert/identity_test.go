package mitmcert

import (
	"testing"
	"time"
)

func TestParseIdentityRejectsMissingAndMismatchedPEM(t *testing.T) {
	if _, err := ParseIdentity(nil, nil); err == nil {
		t.Fatal("empty identity was accepted")
	}
	certA, _, err := Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, keyB, err := Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseIdentity(certA, keyB); err == nil {
		t.Fatal("mismatched identity was accepted")
	}
}
