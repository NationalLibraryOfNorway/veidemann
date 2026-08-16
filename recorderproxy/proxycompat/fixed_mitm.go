package proxycompat

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/getlantern/netx"
	"github.com/getlantern/reconn"
)

const (
	maxTLSRecordSize = 2 << 15
	recordTypeAlert  = 21
)

// MITMIdentity is an immutable TLS identity shared by every proxy listener.
type MITMIdentity struct {
	certificate tls.Certificate
}

// LoadMITMIdentity loads and validates a non-expired TLS server identity.
func LoadMITMIdentity(certFile, keyFile string) (*MITMIdentity, error) {
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
	return ParseMITMIdentity(certPEM, keyPEM)
}

// ParseMITMIdentity validates a PEM-encoded TLS server identity.
func ParseMITMIdentity(certPEM, keyPEM []byte) (*MITMIdentity, error) {
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
	return &MITMIdentity{certificate: keyPair}, nil
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
func (i *MITMIdentity) Certificate() *x509.Certificate {
	if i == nil || i.certificate.Leaf == nil {
		return nil
	}
	return i.certificate.Leaf
}

// NewFixedMITMInterceptor creates an interceptor with no hostname-keyed state.
func NewFixedMITMInterceptor(identity *MITMIdentity, clientTLSConfig *tls.Config) (MITMInterceptor, error) {
	if identity == nil || identity.certificate.Leaf == nil {
		return nil, errors.New("MITM identity is required")
	}

	serverTLSConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			if hello.ServerName == "" {
				return nil, errors.New("no ServerName provided")
			}
			return &identity.certificate, nil
		},
	}
	if clientTLSConfig == nil {
		clientTLSConfig = &tls.Config{}
	}

	return &fixedMITMInterceptor{
		serverTLSConfig: serverTLSConfig,
		clientTLSConfig: clientTLSConfig.Clone(),
	}, nil
}

type fixedMITMInterceptor struct {
	serverTLSConfig *tls.Config
	clientTLSConfig *tls.Config
}

func (i *fixedMITMInterceptor) MITM(downstream net.Conn, upstream net.Conn) (newDown net.Conn, newUp net.Conn, success bool, err error) {
	rc := reconn.Wrap(downstream, maxTLSRecordSize)
	alertDown := &alertDetectingConn{Conn: rc}
	tlsDown := tls.Server(alertDown, i.serverTLSConfig)
	if handshakeErr := tlsDown.Handshake(); handshakeErr != nil {
		if alertDown.sawAlert() || errors.As(handshakeErr, new(tls.RecordHeaderError)) {
			rereader, rereadErr := rc.Rereader()
			if rereadErr != nil {
				return nil, nil, false, fmt.Errorf("unable to re-attempt TLS connection to upstream: %w", rereadErr)
			}
			if _, copyErr := io.Copy(upstream, rereader); copyErr != nil {
				return nil, nil, false, copyErr
			}
			return rc, upstream, false, nil
		}
		return nil, nil, false, handshakeErr
	}

	skipTLS := false
	netx.WalkWrapped(upstream, func(conn net.Conn) bool {
		_, skipTLS = conn.(interface{ MITMSkipEncryption() })
		return !skipTLS
	})
	if skipTLS {
		return tlsDown, upstream, true, nil
	}

	upstreamTLSConfig := i.clientTLSConfig.Clone()
	upstreamTLSConfig.ServerName = tlsDown.ConnectionState().ServerName
	tlsUp := tls.Client(upstream, upstreamTLSConfig)
	return tlsDown, tlsUp, true, tlsUp.Handshake()
}

type alertDetectingConn struct {
	net.Conn
	initialized atomic.Bool
	alerted     atomic.Bool
}

func (c *alertDetectingConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if c.initialized.CompareAndSwap(false, true) && p[0] == recordTypeAlert {
		c.alerted.Store(true)
	}
	if c.sawAlert() {
		return 0, nil
	}
	return c.Conn.Write(p)
}

func (c *alertDetectingConn) sawAlert() bool {
	return c.alerted.Load()
}
