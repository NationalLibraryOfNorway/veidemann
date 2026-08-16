package mitmcert

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteCreatesValidatedLeafAndModes(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "tls.crt")
	keyFile := filepath.Join(dir, "tls.key")
	now := time.Date(2026, time.August, 16, 12, 0, 0, 0, time.UTC)

	if err := Write(certFile, keyFile, now); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	certInfo, err := os.Stat(certFile)
	if err != nil {
		t.Fatal(err)
	}
	keyInfo, err := os.Stat(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := certInfo.Mode().Perm(); got != 0o644 {
		t.Fatalf("certificate mode = %o, want 644", got)
	}
	if got := keyInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("key mode = %o, want 600", got)
	}

	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("certificate PEM did not decode")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if cert.IsCA {
		t.Fatal("generated certificate is a CA")
	}
	if err := cert.VerifyHostname(ServerName); err != nil {
		t.Fatalf("VerifyHostname() error = %v", err)
	}
	if got := cert.NotAfter; !got.Equal(now.AddDate(10, 0, 0)) {
		t.Fatalf("NotAfter = %s, want %s", got, now.AddDate(10, 0, 0))
	}
}
