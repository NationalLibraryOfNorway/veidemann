package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"
)

type answer struct {
	addresses []netip.Addr
	err       error
}

func ips(values ...string) []netip.Addr {
	var result []netip.Addr
	for _, value := range values {
		result = append(result, netip.MustParseAddr(value))
	}
	return result
}

func TestDiscoveryPolicy(t *testing.T) {
	self := answer{addresses: ips("10.0.0.1")}
	peer := answer{addresses: ips("10.0.0.1", "10.0.0.2", "::ffff:10.0.0.2", "2001:db8::2")}
	stale := answer{addresses: ips("10.0.0.99", "10.0.0.2")}
	failure := answer{err: errors.New("DNS unavailable")}
	for _, tc := range []struct {
		name, pod string
		proxy     bool
		answers   []answer
		want      []string
		wantError bool
	}{
		{name: "bootstrap zero", answers: []answer{self, self}},
		{name: "higher ordinal cannot bootstrap", pod: "rethinkdb-1", answers: []answer{self}, wantError: true},
		{name: "proxy cannot bootstrap", proxy: true, answers: []answer{self}, wantError: true},
		{name: "empty success cannot bootstrap", answers: []answer{{}}, wantError: true},
		{name: "lookup failure cannot bootstrap", answers: []answer{failure}, wantError: true},
		{name: "final failure overrides earlier self", answers: []answer{self, failure}, wantError: true},
		{name: "final success recovers", answers: []answer{failure, self}},
		{name: "partial failure discarded", answers: []answer{{addresses: peer.addresses, err: failure.err}}, wantError: true},
		{name: "peers without current pod cannot authorize startup", answers: []answer{stale, stale}, wantError: true},
		{name: "higher ordinal rejects peers without current pod", pod: "rethinkdb-2", answers: []answer{stale}, wantError: true},
		{name: "wait for DNS to include current pod", answers: []answer{stale, peer}, want: []string{"10.0.0.2:29015", "[2001:db8::2]:29015"}},
		{name: "stale final response overrides self-only success", answers: []answer{self, stale}, wantError: true},
		{name: "retry then join", answers: []answer{failure, {}, self, peer}, want: []string{"10.0.0.2:29015", "[2001:db8::2]:29015"}},
		{name: "ordinal zero joins existing members", answers: []answer{peer}, want: []string{"10.0.0.2:29015", "[2001:db8::2]:29015"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{"POD_NAME": "rethinkdb-0", "POD_IP": "10.0.0.1", "RETHINKDB_DISCOVERY_DELAY_SECONDS": "0"}
			if tc.pod != "" {
				env["POD_NAME"] = tc.pod
			}
			if tc.proxy {
				env["PROXY"] = "true"
			}
			c, err := testConfig(t, env)
			if err != nil {
				t.Fatal(err)
			}
			c.discovery.attempts = len(tc.answers)
			if len(tc.want) > 0 {
				c.discovery.attempts += 2 // Finding peers must stop discovery early.
			}
			calls := 0
			lookup := func(ctx context.Context, host string) ([]netip.Addr, error) {
				if host != "rethinkdb.default.svc.cluster.local" {
					t.Fatalf("unexpected lookup: %s", host)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("lookup has no deadline")
				}
				if calls >= len(tc.answers) {
					t.Fatal("too many lookups")
				}
				a := tc.answers[calls]
				calls++
				return a.addresses, a.err
			}
			var output bytes.Buffer
			discovery := peerDiscovery{
				lookup: lookup,
				log:    slog.New(slog.NewJSONHandler(&output, nil)),
			}
			got, err := discovery.joinTargets(t.Context(), c)
			if (err != nil) != tc.wantError || !slices.Equal(got, tc.want) {
				t.Fatalf("peers=%v err=%v logs=%s", got, err, &output)
			}
			if calls != len(tc.answers) {
				t.Fatalf("got %d lookups", calls)
			}
		})
	}
}

func TestDiscoveryBypass(t *testing.T) {
	for _, tc := range []struct{ name, seeds, want string }{
		{"rethinkdb-2", "peer:30015", "peer:30015"},
		{"admin", "", "rethinkdb.default.svc.cluster.local:29015"},
	} {
		c, err := testConfig(t, map[string]string{"POD_NAME": tc.name, "RETHINKDB_SEEDS": tc.seeds})
		if err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		discovery := peerDiscovery{
			lookup: func(context.Context, string) ([]netip.Addr, error) {
				t.Fatal("unexpected DNS lookup")
				return nil, nil
			},
			log: slog.New(slog.NewJSONHandler(&output, nil)),
		}
		got, err := discovery.joinTargets(t.Context(), c)
		if err != nil || !slices.Equal(got, []string{tc.want}) {
			t.Fatalf("got %v, %v", got, err)
		}
	}
}

func TestDiscoveryTimeoutDiscardsPartialAnswers(t *testing.T) {
	c, err := testConfig(t, map[string]string{"POD_NAME": "rethinkdb-0", "POD_IP": "10.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	c.discovery.attempts, c.discovery.timeout = 1, time.Millisecond
	var output bytes.Buffer
	discovery := peerDiscovery{
		lookup: func(ctx context.Context, _ string) ([]netip.Addr, error) {
			<-ctx.Done()
			return ips("10.0.0.1", "10.0.0.2"), nil
		},
		log: slog.New(slog.NewJSONHandler(&output, nil)),
	}
	peers, err := discovery.joinTargets(t.Context(), c)
	if err == nil || len(peers) != 0 || !strings.Contains(output.String(), "timed out") {
		t.Fatalf("peers=%v err=%v logs=%s", peers, err, &output)
	}
}

func TestDiscoveryCancellation(t *testing.T) {
	c, err := testConfig(t, map[string]string{"POD_NAME": "rethinkdb-0"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var output bytes.Buffer
	discovery := peerDiscovery{
		lookup: func(context.Context, string) ([]netip.Addr, error) {
			cancel()
			return ips("127.0.0.1"), nil
		},
		log: slog.New(slog.NewJSONHandler(&output, nil)),
	}
	_, err = discovery.joinTargets(ctx, c)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if err := waitForRetry(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("wait: %v", err)
	}
}

func TestDiscoveryStructuredLogs(t *testing.T) {
	c, err := testConfig(t, map[string]string{
		"POD_NAME": "rethinkdb-0", "POD_IP": "10.0.0.1",
		"RETHINKDB_DISCOVERY_ATTEMPTS": "2", "RETHINKDB_DISCOVERY_DELAY_SECONDS": "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	calls := 0
	discovery := peerDiscovery{
		lookup: func(context.Context, string) ([]netip.Addr, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("resolver unavailable")
			}
			return ips("10.0.0.1"), nil
		},
		log: slog.New(slog.NewJSONHandler(&output, nil)).With("pod", c.pod.name),
	}
	if _, err := discovery.joinTargets(t.Context(), c); err != nil {
		t.Fatal(err)
	}
	var failureSeen, successSeen, bootstrapSeen bool
	decoder := json.NewDecoder(&output)
	for decoder.More() {
		var record struct {
			Level        string `json:"level"`
			Message      string `json:"msg"`
			Pod          string `json:"pod"`
			Service      string `json:"service"`
			Attempt      int    `json:"attempt"`
			MaxAttempts  int    `json:"max_attempts"`
			ContainsSelf bool   `json:"contains_self"`
			Error        string `json:"error"`
		}
		if err := decoder.Decode(&record); err != nil {
			t.Fatal(err)
		}
		if record.Pod != c.pod.name || record.Service != c.service || record.MaxAttempts != 2 {
			t.Fatalf("missing discovery context: %+v", record)
		}
		switch record.Message {
		case "DNS lookup failed":
			failureSeen = record.Level == "WARN" && record.Attempt == 1 && record.Error == "resolver unavailable"
		case "DNS lookup succeeded":
			successSeen = record.Level == "INFO" && record.Attempt == 2 && record.ContainsSelf
		case "Bootstrapping from a self-only DNS response; cluster membership is unverified":
			bootstrapSeen = record.Level == "WARN"
		}
	}
	if !failureSeen || !successSeen || !bootstrapSeen {
		t.Fatalf("missing structured events: failure=%t success=%t bootstrap=%t", failureSeen, successSeen, bootstrapSeen)
	}
}
