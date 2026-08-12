package resolve

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	"github.com/coredns/coredns/plugin/test"
	"github.com/miekg/dns"
	"google.golang.org/grpc/peer"
)

func TestExample(t *testing.T) {
	server := NewResolver("8053")
	defer func() { _ = server.OnStop() }()
	a, _ := net.ResolveTCPAddr("tcp", "127.0.0.1")
	p := &peer.Peer{Addr: a}
	ctx := peer.NewContext(context.TODO(), p)

	server.Next = MsgHandler(test.A("example.org. IN A 127.0.0.1"))
	reply, err := server.Resolve(ctx, &dnsresolverV1.ResolveRequest{Host: "example.org"})
	if err != nil {
		t.Error(err)
	}

	if reply.Host != "example.org" {
		t.Errorf("Expected Host to be example.org, got: %s", reply.Host)
	}
	if reply.Port != 0 {
		t.Errorf("Expected deprecated Port to be unset, got: %d", reply.Port)
	}
	if reply.TextualIp != "127.0.0.1" {
		t.Errorf("Expected TextualIp to be 127.0.0.1, got: %s", reply.TextualIp)
	}
	expectedBytes := []byte{127, 0, 0, 1}
	if !bytes.Equal(reply.RawIp, expectedBytes) {
		t.Errorf("Expected RawIp to be %v, got: %v", expectedBytes, reply.RawIp)
	}
}

func TestDeprecatedPortIsEchoedForLegacyClients(t *testing.T) {
	server := NewResolver("8053")
	defer func() { _ = server.OnStop() }()
	server.Next = MsgHandler(test.A("example.org. IN A 127.0.0.1"))

	reply, err := server.Resolve(testPeerContext(t), &dnsresolverV1.ResolveRequest{
		Host: "example.org",
		Port: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Port != 80 {
		t.Errorf("Expected deprecated Port compatibility echo to be 80, got: %d", reply.Port)
	}
}

func TestResolveReturnsStructuredDNSErrors(t *testing.T) {
	server := NewResolver("8053")
	defer func() { _ = server.OnStop() }()
	ctx := testPeerContext(t)

	soa, err := dns.NewRR("example.org. 120 IN SOA ns.example.org. hostmaster.example.org. 1 3600 600 86400 30")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		handler     pluginHandler
		wantCode    int32
		wantMessage string
	}{
		{
			name: "NXDOMAIN",
			handler: func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
				msg := new(dns.Msg).SetRcode(r, dns.RcodeNameError)
				_ = w.WriteMsg(msg)
				return msg.Rcode, nil
			},
			wantCode:    dns.RcodeNameError,
			wantMessage: "NXDOMAIN",
		},
		{
			name: "NODATA",
			handler: func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
				msg := new(dns.Msg).SetReply(r)
				msg.Ns = []dns.RR{soa}
				_ = w.WriteMsg(msg)
				return msg.Rcode, nil
			},
			wantCode:    dns.RcodeSuccess,
			wantMessage: "NODATA",
		},
		{
			name: "SERVFAIL",
			handler: func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
				msg := new(dns.Msg).SetRcode(r, dns.RcodeServerFailure)
				_ = w.WriteMsg(msg)
				return msg.Rcode, nil
			},
			wantCode:    dns.RcodeServerFailure,
			wantMessage: "SERVFAIL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.Next = test.HandlerFunc(tt.handler)
			reply, err := server.Resolve(ctx, &dnsresolverV1.ResolveRequest{Host: "example.org"})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if reply.GetError().GetCode() != tt.wantCode {
				t.Fatalf("error code = %d, want %d", reply.GetError().GetCode(), tt.wantCode)
			}
			if reply.GetError().GetMsg() != tt.wantMessage {
				t.Fatalf("error message = %q, want %q", reply.GetError().GetMsg(), tt.wantMessage)
			}
			if !strings.Contains(reply.GetError().GetDetail(), "example.org") {
				t.Fatalf("error detail = %q, want host", reply.GetError().GetDetail())
			}
		})
	}
}

func TestResolveKeepsInvalidOrInternalResponsesRetryable(t *testing.T) {
	server := NewResolver("8053")
	defer func() { _ = server.OnStop() }()
	ctx := testPeerContext(t)

	tests := []struct {
		name    string
		handler pluginHandler
	}{
		{
			name: "missing response",
			handler: func(_ context.Context, _ dns.ResponseWriter, _ *dns.Msg) (int, error) {
				return dns.RcodeSuccess, nil
			},
		},
		{
			name: "truncated response",
			handler: func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
				msg := new(dns.Msg).SetReply(r)
				msg.Truncated = true
				_ = w.WriteMsg(msg)
				return dns.RcodeSuccess, nil
			},
		},
		{
			name: "handler error",
			handler: func(_ context.Context, _ dns.ResponseWriter, _ *dns.Msg) (int, error) {
				return dns.RcodeServerFailure, errors.New("upstream unavailable")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.Next = test.HandlerFunc(tt.handler)
			reply, err := server.Resolve(ctx, &dnsresolverV1.ResolveRequest{Host: "example.org"})
			if err == nil {
				t.Fatalf("Resolve() reply = %v, want error", reply)
			}
		})
	}
}

type pluginHandler func(context.Context, dns.ResponseWriter, *dns.Msg) (int, error)

func testPeerContext(t *testing.T) context.Context {
	t.Helper()
	a, err := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
	if err != nil {
		t.Fatal(err)
	}
	return peer.NewContext(context.Background(), &peer.Peer{Addr: a})
}

// MsgHandler returns a Handler that adds answer to request and writes it to w.
func MsgHandler(answer dns.RR) test.Handler {
	return test.HandlerFunc(func(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		reply := new(dns.Msg)
		reply.SetReply(r)
		reply.Answer = append(reply.Answer, answer)
		_ = w.WriteMsg(reply)
		return reply.Rcode, nil
	})
}
