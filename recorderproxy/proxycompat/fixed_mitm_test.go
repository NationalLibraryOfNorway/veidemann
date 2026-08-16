package proxycompat

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
)

func TestFixedMITMUsesOneCertificateForEverySNI(t *testing.T) {
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := ParseMITMIdentity(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	interceptor, err := NewFixedMITMInterceptor(identity, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	fixed := interceptor.(*fixedMITMInterceptor)

	first, err := fixed.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: "first.example"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5000; i++ {
		serverName := fmt.Sprintf("unique-%d.example", i)
		got, err := fixed.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{ServerName: serverName})
		if err != nil {
			t.Fatal(err)
		}
		if got != first {
			t.Fatal("fixed interceptor returned a different certificate")
		}
	}
	if _, err := fixed.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{}); err == nil {
		t.Fatal("empty SNI was accepted")
	}
}

func TestFixedMITMIdentityIsSharedAcrossElevenListeners(t *testing.T) {
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := ParseMITMIdentity(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}

	var first *tls.Certificate
	for listener := 0; listener < 11; listener++ {
		interceptor, err := NewFixedMITMInterceptor(identity, nil)
		if err != nil {
			t.Fatal(err)
		}
		fixed := interceptor.(*fixedMITMInterceptor)
		got, err := fixed.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{
			ServerName: fmt.Sprintf("listener-%d.example", listener),
		})
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = got
		} else if got != first {
			t.Fatal("listeners did not share the same certificate identity")
		}
	}
}

func TestParseMITMIdentityRejectsMissingAndMismatchedPEM(t *testing.T) {
	if _, err := ParseMITMIdentity(nil, nil); err == nil {
		t.Fatal("empty identity was accepted")
	}
	certA, _, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, keyB, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseMITMIdentity(certA, keyB); err == nil {
		t.Fatal("mismatched identity was accepted")
	}
}
