package main

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

const defaultClusterPort = 29015

// config contains validated startup settings. Environment parsing stops here;
// discovery and command construction work with addresses, ports, and durations.
type config struct {
	pod         podIdentity
	service     string
	clusterPort uint16
	password    string
	proxy       bool
	seeds       []string
	discovery   discoveryPolicy
	extraArgs   []string
}

type podIdentity struct {
	name        string
	namespace   string
	address     netip.Addr
	ordinal     string // Empty for pods without a StatefulSet-style name.
	statefulSet string
}

type discoveryPolicy struct {
	attempts int
	delay    time.Duration
	timeout  time.Duration
}

type environment func(string) (string, bool)

func loadConfig(env environment, args []string) (config, error) {
	pod, err := env.pod()
	if err != nil {
		return config{}, err
	}
	policy, err := env.discoveryPolicy()
	if err != nil {
		return config{}, err
	}
	port, forwarded, err := resolveClusterPort(env, args)
	if err != nil {
		return config{}, err
	}
	seeds, err := parseSeeds(env.value("RETHINKDB_SEEDS", ""), port)
	if err != nil {
		return config{}, fmt.Errorf("RETHINKDB_SEEDS: %w", err)
	}
	service := env.value("RETHINKDB_SERVICE_NAME", "rethinkdb")
	domain := env.value("RETHINKDB_CLUSTER_DOMAIN", "cluster.local")
	return config{
		pod:         pod,
		service:     strings.Join([]string{service, pod.namespace, "svc", domain}, "."),
		clusterPort: port,
		password:    env.value("RETHINKDB_PASSWORD", "auto"),
		proxy:       env.value("PROXY", "") != "",
		seeds:       seeds,
		discovery:   policy,
		extraArgs:   forwarded,
	}, nil
}

func (env environment) pod() (podIdentity, error) {
	name := env.value("POD_NAME", "")
	if name == "" {
		return podIdentity{}, errors.New("POD_NAME must be set")
	}
	address, err := netip.ParseAddr(env.value("POD_IP", "127.0.0.1"))
	if err != nil {
		return podIdentity{}, fmt.Errorf("POD_IP must be an IP address: %w", err)
	}
	statefulSet, ordinal := "rethinkdb", ""
	if i := strings.LastIndexByte(name, '-'); i > 0 && decimalDigits(name[i+1:]) {
		statefulSet, ordinal = name[:i], name[i+1:]
	}
	return podIdentity{
		name:        name,
		namespace:   env.value("POD_NAMESPACE", "default"),
		address:     address.Unmap(),
		ordinal:     ordinal,
		statefulSet: env.value("RETHINKDB_STATEFULSET_NAME", statefulSet),
	}, nil
}

func (env environment) discoveryPolicy() (discoveryPolicy, error) {
	attempts, err := env.integer("RETHINKDB_DISCOVERY_ATTEMPTS", 5, 1, 2147483647)
	if err != nil {
		return discoveryPolicy{}, err
	}
	delay, err := env.integer("RETHINKDB_DISCOVERY_DELAY_SECONDS", 2, 0, 2147483647)
	if err != nil {
		return discoveryPolicy{}, err
	}
	timeout, err := env.integer("RETHINKDB_DISCOVERY_DNS_TIMEOUT_SECONDS", 2, 1, 2147483647)
	if err != nil {
		return discoveryPolicy{}, err
	}
	return discoveryPolicy{
		attempts: attempts,
		delay:    time.Duration(delay) * time.Second,
		timeout:  time.Duration(timeout) * time.Second,
	}, nil
}

// Empty string settings use their defaults; explicitly empty numeric settings
// are errors. Keep this distinction at the environment boundary.
func (env environment) value(name, fallback string) string {
	if value, _ := env(name); value != "" {
		return value
	}
	return fallback
}

func (env environment) integer(name string, fallback, minimum, maximum int) (int, error) {
	value, set := env(name)
	if !set {
		return fallback, nil
	}
	return parseInteger(name, value, minimum, maximum)
}

func parseInteger(name, value string, minimum, maximum int) (int, error) {
	n, err := strconv.ParseUint(value, 10, 32)
	if !decimalDigits(value) || err != nil || n < uint64(minimum) || n > uint64(maximum) {
		return 0, fmt.Errorf("%s must be a decimal integer (%d..%d)", name, minimum, maximum)
	}
	return int(n), nil
}

func decimalDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func (c config) canBootstrap() bool {
	return c.pod.ordinal == "0" && !c.proxy
}
