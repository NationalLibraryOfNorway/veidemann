package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/mitmcert"
)

func TestHandlerUsesOneCertificateForEverySNI(t *testing.T) {
	certPEM, keyPEM, err := mitmcert.Generate(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	identity, err := mitmcert.ParseIdentity(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(Config{
		Identity: identity,
		Filter: FilterFunc(func(s *State, req *http.Request, next Next) (*http.Response, *State, error) {
			return next(s, req)
		}),
		Dial: func(context.Context, bool, string, string) (net.Conn, error) {
			return nil, fmt.Errorf("unused")
		},
		OnError: func(*State, *http.Request, ErrorPhase, error) *http.Response { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	var first *tls.Certificate
	for i := range 5000 {
		certificate, err := handler.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{
			ServerName: fmt.Sprintf("unique-%d.example", i),
		})
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = certificate
		} else if certificate != first {
			t.Fatal("handler returned a different certificate")
		}
	}
	if _, err := handler.serverTLSConfig.GetCertificate(&tls.ClientHelloInfo{}); err == nil {
		t.Fatal("empty SNI was accepted")
	}
}
