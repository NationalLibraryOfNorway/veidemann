package mitmcert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

// Identity is an immutable TLS server identity shared by proxy listeners.
type Identity struct {
	certificate tls.Certificate
}

// LoadIdentity loads and validates a non-expired TLS server identity.
func LoadIdentity(certFile, keyFile string) (*Identity, error) {
	if certFile == "" || keyFile == "" {
		return nil, errors.New("both MITM certificate and key files are required")
	}

	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("read MITM certificate: %w", err)
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read MITM private key: %w", err)
	}
	return ParseIdentity(certPEM, keyPEM)
}

// ParseIdentity validates a PEM-encoded TLS server identity.
func ParseIdentity(certPEM, keyPEM []byte) (*Identity, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("MITM certificate file does not contain a PEM certificate")
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse MITM certificate: %w", err)
	}

	now := time.Now()
	if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
		return nil, fmt.Errorf("MITM certificate is not currently valid (valid from %s through %s)", leaf.NotBefore, leaf.NotAfter)
	}
	if leaf.IsCA {
		return nil, errors.New("MITM certificate must be a server leaf, not a CA")
	}
	if !allowsServerAuth(leaf.ExtKeyUsage) {
		return nil, errors.New("MITM certificate is not valid for TLS server authentication")
	}
	if leaf.KeyUsage != 0 && leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return nil, errors.New("MITM certificate does not permit digital signatures")
	}

	keyPair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load MITM key pair: %w", err)
	}
	keyPair.Leaf = leaf
	return &Identity{certificate: keyPair}, nil
}

func allowsServerAuth(usages []x509.ExtKeyUsage) bool {
	if len(usages) == 0 {
		return true
	}
	for _, usage := range usages {
		if usage == x509.ExtKeyUsageAny || usage == x509.ExtKeyUsageServerAuth {
			return true
		}
	}
	return false
}

// Certificate returns the parsed public certificate for identity checks.
func (i *Identity) Certificate() *x509.Certificate {
	if i == nil || i.certificate.Leaf == nil {
		return nil
	}
	return i.certificate.Leaf
}

// TLSCertificate returns the immutable key pair used by TLS listeners.
func (i *Identity) TLSCertificate() *tls.Certificate {
	if i == nil || i.certificate.Leaf == nil {
		return nil
	}
	return &i.certificate
}
