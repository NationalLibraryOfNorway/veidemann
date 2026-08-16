package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type certificateOptions struct {
	notBefore time.Time
	notAfter  time.Time
	isCA      bool
	extUsage  []x509.ExtKeyUsage
	keyUsage  x509.KeyUsage
}

func newCertificate(t *testing.T, opts certificateOptions) ([]byte, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "cache"},
		DNSNames:              []string{"cache"},
		NotBefore:             opts.notBefore,
		NotAfter:              opts.notAfter,
		BasicConstraintsValid: true,
		IsCA:                  opts.isCA,
		ExtKeyUsage:           opts.extUsage,
		KeyUsage:              opts.keyUsage,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
}

func validCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()
	now := time.Now()
	return newCertificate(t, certificateOptions{
		notBefore: now.Add(-time.Hour),
		notAfter:  now.Add(time.Hour),
		extUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		keyUsage:  x509.KeyUsageDigitalSignature,
	})
}

func writePair(t *testing.T, dir string, certPEM, keyPEM []byte) (string, string) {
	t.Helper()
	certFile := filepath.Join(dir, "tls.crt")
	keyFile := filepath.Join(dir, "tls.key")
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func TestValidateTLSFilesAcceptsValidPair(t *testing.T) {
	now := time.Now()
	certPEM, keyPEM := validCertificate(t)
	certFile, keyFile := writePair(t, t.TempDir(), certPEM, keyPEM)

	fingerprint, err := validateTLSFiles(certFile, keyFile, now)
	if err != nil {
		t.Fatalf("validateTLSFiles() error = %v", err)
	}
	if len(strings.Split(fingerprint, ":")) != 2 {
		t.Fatalf("fingerprint = %q, want certificate:key SHA-256 values", fingerprint)
	}
}

func TestValidateTLSFilesRejectsInvalidMaterial(t *testing.T) {
	now := time.Now()
	validCert, validKey := validCertificate(t)
	_, otherKey := validCertificate(t)
	expiredCert, expiredKey := newCertificate(t, certificateOptions{
		notBefore: now.Add(-2 * time.Hour),
		notAfter:  now.Add(-time.Hour),
		extUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		keyUsage:  x509.KeyUsageDigitalSignature,
	})
	futureCert, futureKey := newCertificate(t, certificateOptions{
		notBefore: now.Add(time.Hour),
		notAfter:  now.Add(2 * time.Hour),
		extUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		keyUsage:  x509.KeyUsageDigitalSignature,
	})
	caCert, caKey := newCertificate(t, certificateOptions{
		notBefore: now.Add(-time.Hour),
		notAfter:  now.Add(time.Hour),
		isCA:      true,
		extUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		keyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	})
	clientCert, clientKey := newCertificate(t, certificateOptions{
		notBefore: now.Add(-time.Hour),
		notAfter:  now.Add(time.Hour),
		extUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		keyUsage:  x509.KeyUsageDigitalSignature,
	})
	noSigningCert, noSigningKey := newCertificate(t, certificateOptions{
		notBefore: now.Add(-time.Hour),
		notAfter:  now.Add(time.Hour),
		extUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		keyUsage:  x509.KeyUsageKeyEncipherment,
	})

	tests := []struct {
		name    string
		certPEM []byte
		keyPEM  []byte
		missing string
		want    string
	}{
		{name: "missing certificate", keyPEM: validKey, missing: "cert", want: "read certificate"},
		{name: "missing key", certPEM: validCert, missing: "key", want: "read private key"},
		{name: "empty certificate", certPEM: []byte{}, keyPEM: validKey, want: "certificate"},
		{name: "empty key", certPEM: validCert, keyPEM: []byte{}, want: "private key"},
		{name: "malformed", certPEM: []byte("not PEM"), keyPEM: validKey, want: "load certificate/key pair"},
		{name: "mismatched", certPEM: validCert, keyPEM: otherKey, want: "private key does not match"},
		{name: "expired", certPEM: expiredCert, keyPEM: expiredKey, want: "expired"},
		{name: "not yet valid", certPEM: futureCert, keyPEM: futureKey, want: "not valid before"},
		{name: "CA", certPEM: caCert, keyPEM: caKey, want: "must not be a CA"},
		{name: "client only", certPEM: clientCert, keyPEM: clientKey, want: "server authentication"},
		{name: "cannot sign", certPEM: noSigningCert, keyPEM: noSigningKey, want: "digital signatures"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			certFile := filepath.Join(dir, "tls.crt")
			keyFile := filepath.Join(dir, "tls.key")
			if tt.missing != "cert" {
				if err := os.WriteFile(certFile, tt.certPEM, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.missing != "key" {
				if err := os.WriteFile(keyFile, tt.keyPEM, 0600); err != nil {
					t.Fatal(err)
				}
			}

			_, err := validateTLSFiles(certFile, keyFile, now)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateTLSFiles() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

type fakeSquidRunner struct {
	mu                sync.Mutex
	validateCalls     int
	reconfigureCalls  int
	validateErrors    []error
	reconfigureErrors []error
	reconfigureCallCh chan time.Time
}

func (f *fakeSquidRunner) validateConfig() (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.validateCalls++
	call := f.validateCalls
	if call <= len(f.validateErrors) {
		return "", f.validateErrors[call-1]
	}
	return "", nil
}

func (f *fakeSquidRunner) reconfigure() (string, error) {
	f.mu.Lock()
	f.reconfigureCalls++
	call := f.reconfigureCalls
	var err error
	if call <= len(f.reconfigureErrors) {
		err = f.reconfigureErrors[call-1]
	}
	f.mu.Unlock()
	if f.reconfigureCallCh != nil {
		f.reconfigureCallCh <- time.Now()
	}
	return "", err
}

func (f *fakeSquidRunner) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.validateCalls, f.reconfigureCalls
}

func newTestRewriter(t *testing.T, runner squidRunner) (*rewriter, string) {
	t.Helper()
	t.Setenv("DNS_SERVERS", "192.0.2.53")
	dir := t.TempDir()
	certPEM, keyPEM := validCertificate(t)
	certFile, keyFile := writePair(t, dir, certPEM, keyPEM)
	templatePath := filepath.Join(dir, "squid.conf.template")
	if err := os.WriteFile(templatePath, []byte("dns_nameservers ${DNS_IP}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return &rewriter{
		templatePath: templatePath,
		configPath:   filepath.Join(dir, "90-role.conf"),
		tlsCertFile:  certFile,
		tlsKeyFile:   keyFile,
		runner:       runner,
		now:          time.Now,
	}, dir
}

func TestBalancerRewriteDoesNotReadTLSFiles(t *testing.T) {
	t.Setenv("DNS_SERVERS", "192.0.2.53")
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "squid-balancer.conf.template")
	if err := os.WriteFile(templatePath, []byte("${PARENTS}dns_nameservers ${DNS_IP}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runner := &fakeSquidRunner{}
	r := &rewriter{
		balancer:     true,
		templatePath: templatePath,
		configPath:   filepath.Join(dir, "90-role.conf"),
		tlsCertFile:  filepath.Join(dir, "missing.crt"),
		tlsKeyFile:   filepath.Join(dir, "missing.key"),
		getParentsFunc: func() (string, error) {
			return "cache_peer 192.0.2.1 parent 3128 0\n", nil
		},
		runner: runner,
		now:    time.Now,
	}

	changed, err := r.rewriteConfig()
	if err != nil {
		t.Fatalf("rewriteConfig() error = %v", err)
	}
	if !changed {
		t.Fatal("rewriteConfig() changed = false, want true")
	}
}

func TestParentRewriteTracksTLSFingerprintAndDNS(t *testing.T) {
	runner := &fakeSquidRunner{}
	r, dir := newTestRewriter(t, runner)

	changed, err := r.rewriteConfig()
	if err != nil || !changed {
		t.Fatalf("initial rewriteConfig() = %v, %v; want true, nil", changed, err)
	}
	changed, err = r.rewriteConfig()
	if err != nil || changed {
		t.Fatalf("unchanged rewriteConfig() = %v, %v; want false, nil", changed, err)
	}

	certPEM, keyPEM := validCertificate(t)
	writePair(t, dir, certPEM, keyPEM)
	changed, err = r.rewriteConfig()
	if err != nil || !changed {
		t.Fatalf("TLS rewriteConfig() = %v, %v; want true, nil", changed, err)
	}

	t.Setenv("DNS_SERVERS", "192.0.2.54")
	changed, err = r.rewriteConfig()
	if err != nil || !changed {
		t.Fatalf("DNS rewriteConfig() = %v, %v; want true, nil", changed, err)
	}

	validateCalls, _ := runner.counts()
	if validateCalls != 3 {
		t.Fatalf("validate calls = %d, want 3", validateCalls)
	}
}

func TestBalancerRewriteTracksParentsAndDNS(t *testing.T) {
	t.Setenv("DNS_SERVERS", "192.0.2.53")
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "squid-balancer.conf.template")
	if err := os.WriteFile(templatePath, []byte("${PARENTS}dns_nameservers ${DNS_IP}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	parents := "cache_peer 192.0.2.1 parent 3128 0\n"
	runner := &fakeSquidRunner{}
	r := &rewriter{
		balancer:     true,
		templatePath: templatePath,
		configPath:   filepath.Join(dir, "90-role.conf"),
		getParentsFunc: func() (string, error) {
			return parents, nil
		},
		runner: runner,
		now:    time.Now,
	}

	if changed, err := r.rewriteConfig(); err != nil || !changed {
		t.Fatalf("initial rewriteConfig() = %v, %v; want true, nil", changed, err)
	}
	if changed, err := r.rewriteConfig(); err != nil || changed {
		t.Fatalf("unchanged rewriteConfig() = %v, %v; want false, nil", changed, err)
	}

	parents = "cache_peer 192.0.2.2 parent 3128 0\n"
	if changed, err := r.rewriteConfig(); err != nil || !changed {
		t.Fatalf("parent rewriteConfig() = %v, %v; want true, nil", changed, err)
	}
	t.Setenv("DNS_SERVERS", "192.0.2.54")
	if changed, err := r.rewriteConfig(); err != nil || !changed {
		t.Fatalf("DNS rewriteConfig() = %v, %v; want true, nil", changed, err)
	}

	validateCalls, _ := runner.counts()
	if validateCalls != 3 {
		t.Fatalf("validate calls = %d, want 3", validateCalls)
	}
}

func TestRewriteRetriesSquidValidationFailure(t *testing.T) {
	runner := &fakeSquidRunner{
		validateErrors: []error{nil, errors.New("parse failed")},
	}
	r, dir := newTestRewriter(t, runner)
	if changed, err := r.rewriteConfig(); err != nil || !changed {
		t.Fatalf("initial rewriteConfig() = %v, %v; want true, nil", changed, err)
	}
	acceptedFingerprint := r.lastTLSFingerprint

	certPEM, keyPEM := validCertificate(t)
	writePair(t, dir, certPEM, keyPEM)
	if changed, err := r.rewriteConfig(); err == nil || changed {
		t.Fatalf("failed rewriteConfig() = %v, %v; want false, error", changed, err)
	}
	if r.lastTLSFingerprint != acceptedFingerprint {
		t.Fatal("failed Squid validation accepted the new TLS fingerprint")
	}

	if changed, err := r.rewriteConfig(); err != nil || !changed {
		t.Fatalf("retried rewriteConfig() = %v, %v; want true, nil", changed, err)
	}
	if r.lastTLSFingerprint == acceptedFingerprint {
		t.Fatal("successful retry did not accept the new TLS fingerprint")
	}
}

func TestRunReconfiguresOnceForChangedTLSFingerprint(t *testing.T) {
	runner := &fakeSquidRunner{reconfigureCallCh: make(chan time.Time, 4)}
	r, dir := newTestRewriter(t, runner)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		done <- run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), r, 5*time.Millisecond, 30*time.Millisecond, "")
	}()

	waitForValidation(t, runner, 1)
	certPEM, keyPEM := validCertificate(t)
	writePair(t, dir, certPEM, keyPEM)

	select {
	case callTime := <-runner.reconfigureCallCh:
		if callTime.Sub(start) < 25*time.Millisecond {
			t.Fatalf("reconfigure happened after %s, before minimum interval", callTime.Sub(start))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reconfigure")
	}
	time.Sleep(50 * time.Millisecond)
	_, reconfigureCalls := runner.counts()
	if reconfigureCalls != 1 {
		t.Fatalf("reconfigure calls = %d, want 1", reconfigureCalls)
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v, want context canceled", err)
	}
}

func TestRunRetriesFailedReconfiguration(t *testing.T) {
	runner := &fakeSquidRunner{
		reconfigureErrors: []error{errors.New("first attempt failed")},
		reconfigureCallCh: make(chan time.Time, 4),
	}
	r, dir := newTestRewriter(t, runner)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), r, 5*time.Millisecond, 0, "")
	}()

	waitForValidation(t, runner, 1)
	certPEM, keyPEM := validCertificate(t)
	writePair(t, dir, certPEM, keyPEM)
	for range 2 {
		select {
		case <-runner.reconfigureCallCh:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for reconfigure retry")
		}
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v, want context canceled", err)
	}
	_, reconfigureCalls := runner.counts()
	if reconfigureCalls != 2 {
		t.Fatalf("reconfigure calls = %d, want 2", reconfigureCalls)
	}
}

func TestRunRetriesInvalidChangedTLSMaterial(t *testing.T) {
	runner := &fakeSquidRunner{reconfigureCallCh: make(chan time.Time, 4)}
	r, dir := newTestRewriter(t, runner)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), r, 5*time.Millisecond, 0, "")
	}()

	waitForValidation(t, runner, 1)
	if err := os.WriteFile(filepath.Join(dir, "tls.crt"), []byte("invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	_, reconfigureCalls := runner.counts()
	if reconfigureCalls != 0 {
		t.Fatalf("reconfigure calls with invalid certificate = %d, want 0", reconfigureCalls)
	}

	certPEM, keyPEM := validCertificate(t)
	writePair(t, dir, certPEM, keyPEM)
	select {
	case <-runner.reconfigureCallCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reconfigure after valid material appeared")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("run() error = %v, want context canceled", err)
	}
}

func TestRoleTemplatesHaveExpectedTLSDependencies(t *testing.T) {
	parent, err := os.ReadFile("../../../squid.conf.template")
	if err != nil {
		t.Fatal(err)
	}
	parentConfig := string(parent)
	for _, want := range []string{
		"tls-cert=${TLS_CERT_FILE}",
		"tls-key=${TLS_KEY_FILE}",
		"generate-host-certificates=off",
		"dynamic_cert_mem_cache_size=0KB",
	} {
		if !strings.Contains(parentConfig, want) {
			t.Errorf("parent template missing %q", want)
		}
	}
	if strings.Contains(parentConfig, "sslcrtd_program") {
		t.Error("parent template still contains sslcrtd_program")
	}

	child, err := os.ReadFile("../../../squid-balancer.conf.template")
	if err != nil {
		t.Fatal(err)
	}
	childConfig := string(child)
	if strings.Contains(childConfig, "tls-certificates") || strings.Contains(childConfig, "sslcrtd") {
		t.Error("child template depends on TLS certificate generation material")
	}
}

func TestParentRewriteRendersConfiguredTLSPaths(t *testing.T) {
	runner := &fakeSquidRunner{}
	r, _ := newTestRewriter(t, runner)
	r.templatePath = "../../../squid.conf.template"

	changed, err := r.rewriteConfig()
	if err != nil {
		t.Fatalf("rewriteConfig() error = %v", err)
	}
	if !changed {
		t.Fatal("rewriteConfig() changed = false, want true")
	}

	config, err := os.ReadFile(r.configPath)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(config)
	for _, want := range []string{
		"tls-cert=" + r.tlsCertFile,
		"tls-key=" + r.tlsKeyFile,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered parent config missing %q", want)
		}
	}
	if strings.Contains(rendered, "${TLS_") {
		t.Error("rendered parent config contains unresolved TLS placeholders")
	}
}

func waitForValidation(t *testing.T, runner *fakeSquidRunner, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		calls, _ := runner.counts()
		if calls >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d validation calls", want)
}
