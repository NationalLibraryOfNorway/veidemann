package main

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"unicode"
)

func resolveClusterPort(env environment, args []string) (uint16, []string, error) {
	port, err := env.integer("RETHINKDB_CLUSTER_PORT", defaultClusterPort, 1, 65535)
	if err != nil {
		return 0, nil, err
	}
	override, forwarded, err := parsePortOption(args)
	if err != nil {
		return 0, nil, err
	}
	if override != 0 {
		if _, set := env("RETHINKDB_CLUSTER_PORT"); set && override != port {
			return 0, nil, errors.New("--cluster-port conflicts with RETHINKDB_CLUSTER_PORT")
		}
		port = override
	}
	return uint16(port), forwarded, nil
}

// RethinkDB owns the rest of the command line. flag.FlagSet cannot parse our
// option among arbitrary RethinkDB options without consuming or rejecting them.
func parsePortOption(args []string) (int, []string, error) {
	var port int
	var forwarded []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			forwarded = append(forwarded, args[i:]...)
			break
		}
		name, value, hasValue := strings.Cut(args[i], "=")
		if name != "--cluster-port" {
			forwarded = append(forwarded, args[i])
			continue
		}
		if port != 0 {
			return 0, nil, errors.New("--cluster-port must be supplied only once")
		}
		if !hasValue {
			i++
			if i == len(args) {
				return 0, nil, errors.New("--cluster-port requires a value")
			}
			value = args[i]
		}
		var err error
		port, err = parseInteger("--cluster-port", value, 1, 65535)
		if err != nil {
			return 0, nil, err
		}
	}
	return port, forwarded, nil
}

func parseSeeds(value string, port uint16) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	seeds := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	if len(seeds) == 0 {
		return nil, errors.New("nonempty value contains no endpoints")
	}
	for i, seed := range seeds {
		endpoint, err := joinEndpoint(seed, port)
		if err != nil {
			return nil, err
		}
		seeds[i] = endpoint
	}
	return seeds, nil
}

func joinEndpoint(seed string, defaultPort uint16) (string, error) {
	// A bare IP (including IPv6) has no explicit port.
	if address, err := netip.ParseAddr(seed); err == nil {
		return netip.AddrPortFrom(address, defaultPort).String(), nil
	}
	if strings.HasPrefix(seed, "[") && strings.HasSuffix(seed, "]") {
		address, err := netip.ParseAddr(seed[1 : len(seed)-1])
		if err != nil {
			return "", fmt.Errorf("invalid bracketed IP address %q", seed)
		}
		return netip.AddrPortFrom(address, defaultPort).String(), nil
	}
	if !strings.ContainsAny(seed, ":[]") {
		if seed == "" {
			return "", errors.New("join endpoint has no host")
		}
		return net.JoinHostPort(seed, strconv.Itoa(int(defaultPort))), nil
	}
	host, value, err := net.SplitHostPort(seed)
	if err != nil {
		return "", fmt.Errorf("invalid join endpoint %q: %w", seed, err)
	}
	if host == "" {
		return "", fmt.Errorf("join endpoint %q has no host", seed)
	}
	port, err := parseInteger("seed port", value, 1, 65535)
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func (c config) canonicalAddress() string {
	// RethinkDB resolves advertised hostnames during handshakes and retains the
	// resolved addresses for reconnects. Advertise this process's actual pod IP
	// so stale DNS during a rollout cannot poison a peer's routing information.
	return netip.AddrPortFrom(c.pod.address, c.clusterPort).String()
}

func (c config) command(peers []string) []string {
	args := []string{"rethinkdb"}
	if c.proxy {
		args = append(args, "proxy")
	} else {
		args = append(args, "--server-name", strings.ReplaceAll(c.pod.name, "-", "_"))
	}
	args = append(args,
		"--cluster-port", strconv.Itoa(int(c.clusterPort)),
		"--canonical-address", c.canonicalAddress(),
		"--initial-password", c.password,
	)
	for _, peer := range peers {
		args = append(args, "--join", peer)
	}
	return append(args, c.extraArgs...)
}
