package main

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func testConfig(t *testing.T, env map[string]string, args ...string) (config, error) {
	t.Helper()
	lookup := func(name string) (string, bool) {
		value, ok := env[name]
		return value, ok
	}
	return loadConfig(lookup, args)
}

func TestConfigurationDefaults(t *testing.T) {
	c, err := testConfig(t, map[string]string{"POD_NAME": "my-db-0"})
	if err != nil {
		t.Fatal(err)
	}
	if c.pod.ordinal != "0" || c.pod.statefulSet != "my-db" || c.clusterPort != 29015 || c.discovery.attempts != 5 || c.discovery.delay != 2*time.Second || c.discovery.timeout != 2*time.Second {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if got := c.canonicalAddress(); got != "my-db-0.rethinkdb.default.svc.cluster.local:29015" {
		t.Fatal(got)
	}
}

func TestConfigurationErrors(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"POD_NAME", ""}, {"POD_IP", "not-an-ip"},
		{"RETHINKDB_DISCOVERY_ATTEMPTS", "0"}, {"RETHINKDB_DISCOVERY_ATTEMPTS", "abc"},
		{"RETHINKDB_DISCOVERY_ATTEMPTS", "1+1"}, {"RETHINKDB_DISCOVERY_ATTEMPTS", "9223372036854775808"},
		{"RETHINKDB_DISCOVERY_ATTEMPTS", "2147483648"},
		{"RETHINKDB_DISCOVERY_DELAY_SECONDS", "-1"}, {"RETHINKDB_DISCOVERY_DELAY_SECONDS", "1.5"},
		{"RETHINKDB_DISCOVERY_DNS_TIMEOUT_SECONDS", ""}, {"RETHINKDB_DISCOVERY_DNS_TIMEOUT_SECONDS", "0"},
		{"RETHINKDB_CLUSTER_PORT", "65536"}, {"RETHINKDB_CLUSTER_PORT", "+29015"},
		{"RETHINKDB_SEEDS", " , \n\t,"}, {"RETHINKDB_SEEDS", "peer:0"}, {"RETHINKDB_SEEDS", ":29015"},
	} {
		t.Run(tc.name+"="+tc.value, func(t *testing.T) {
			env := map[string]string{"POD_NAME": "rethinkdb-0"}
			env[tc.name] = tc.value
			_, err := testConfig(t, env)
			if err == nil || !strings.Contains(err.Error(), tc.name) {
				t.Fatalf("expected named configuration error, got %v", err)
			}
		})
	}
}

func TestPortOptionsAndForwarding(t *testing.T) {
	for _, tc := range []struct {
		name      string
		envPort   string
		args      []string
		wantPort  uint16
		wantArgs  []string
		wantError bool
	}{
		{name: "separate", args: []string{"--directory", "/data/a b", "--cluster-port", "0030015", "--no-http-admin"}, wantPort: 30015, wantArgs: []string{"--directory", "/data/a b", "--no-http-admin"}},
		{name: "equals", args: []string{"--cluster-port=30015"}, wantPort: 30015},
		{name: "matching environment", envPort: "030015", args: []string{"--cluster-port=30015"}, wantPort: 30015},
		{name: "conflict", envPort: "29015", args: []string{"--cluster-port=30015"}, wantError: true},
		{name: "repeated", args: []string{"--cluster-port=30015", "--cluster-port", "30015"}, wantError: true},
		{name: "missing", args: []string{"--cluster-port"}, wantError: true},
		{name: "empty", args: []string{"--cluster-port="}, wantError: true},
		{name: "out of range", args: []string{"--cluster-port=65536"}, wantError: true},
		{name: "end of options", args: []string{"--", "--cluster-port=30015"}, wantPort: 29015, wantArgs: []string{"--", "--cluster-port=30015"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{"POD_NAME": "rethinkdb-0"}
			if tc.envPort != "" {
				env["RETHINKDB_CLUSTER_PORT"] = tc.envPort
			}
			c, err := testConfig(t, env, tc.args...)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.clusterPort != tc.wantPort || !slices.Equal(c.extraArgs, tc.wantArgs) {
				t.Fatalf("port=%d args=%q", c.clusterPort, c.extraArgs)
			}
		})
	}
}

func TestSeedsAndCommand(t *testing.T) {
	env := map[string]string{
		"POD_NAME": "my-db-2", "POD_NAMESPACE": "archive", "RETHINKDB_SERVICE_NAME": "db",
		"RETHINKDB_CLUSTER_DOMAIN": "example.local", "RETHINKDB_PASSWORD": "secret",
		"RETHINKDB_DISCOVERY_DELAY_SECONDS": "000", "RETHINKDB_DISCOVERY_ATTEMPTS": "008",
		"RETHINKDB_SEEDS": "peer,\n\t10.0.0.2:029016 2001:db8::1 [2001:db8::2] [2001:db8::3]:30000",
	}
	c, err := testConfig(t, env, "--cluster-port=30015", "--server-tag", "tag with spaces")
	if err != nil {
		t.Fatal(err)
	}
	wantPeers := []string{"peer:30015", "10.0.0.2:29016", "[2001:db8::1]:30015", "[2001:db8::2]:30015", "[2001:db8::3]:30000"}
	if !slices.Equal(c.seeds, wantPeers) || c.discovery.delay != 0 || c.discovery.attempts != 8 {
		t.Fatalf("unexpected config: %+v", c)
	}
	want := []string{"rethinkdb", "--server-name", "my_db_2", "--cluster-port", "30015",
		"--canonical-address", "my-db-2.db.archive.svc.example.local:30015", "--initial-password", "secret"}
	for _, peer := range wantPeers {
		want = append(want, "--join", peer)
	}
	want = append(want, "--server-tag", "tag with spaces")
	if got := c.command(c.seeds); !slices.Equal(got, want) {
		t.Fatalf("got %q; want %q", got, want)
	}

	env["PROXY"] = "false" // Preserve the existing nonempty-value switch.
	env["POD_NAME"] = "admin"
	env["POD_IP"] = "2001:db8::9"
	c, err = testConfig(t, env)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.command(c.seeds); got[1] != "proxy" || c.canonicalAddress() != "[2001:db8::9]:29015" {
		t.Fatalf("invalid proxy command: %q", got)
	}
}
