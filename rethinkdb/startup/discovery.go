package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"time"
)

type addressLookup func(context.Context, string) ([]netip.Addr, error)

type peerDiscovery struct {
	lookup addressLookup
	log    *slog.Logger
}

// A snapshot describes one successful lookup. Failed lookups always return an
// empty snapshot so an earlier self-only response cannot authorize bootstrap.
type peerSnapshot struct {
	peers        []string
	containsSelf bool
}

func (d peerDiscovery) joinTargets(ctx context.Context, c config) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch {
	case len(c.seeds) > 0:
		d.log.InfoContext(ctx, "Using explicit join targets", "source", "RETHINKDB_SEEDS")
		return c.seeds, nil
	case c.pod.ordinal == "":
		d.log.InfoContext(ctx, "Using headless service as join target", "service", c.service)
		return []string{net.JoinHostPort(c.service, strconv.Itoa(int(c.clusterPort)))}, nil
	default:
		return d.discover(ctx, c)
	}
}

func (d peerDiscovery) discover(ctx context.Context, c config) ([]string, error) {
	logger := d.log.With("service", c.service, "max_attempts", c.discovery.attempts)
	logger.InfoContext(ctx, "Discovering cluster peers")

	var snapshot peerSnapshot
	for attempt := 1; attempt <= c.discovery.attempts; attempt++ {
		var err error
		snapshot, err = d.query(ctx, c)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err != nil {
			message := "DNS lookup failed"
			var dnsError *net.DNSError
			if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &dnsError) && dnsError.Timeout()) {
				message = "DNS lookup timed out"
			}
			logger.WarnContext(ctx, message, "attempt", attempt, "error", err)
		} else {
			logger.InfoContext(ctx, "DNS lookup succeeded",
				"attempt", attempt, "peer_count", len(snapshot.peers), "contains_self", snapshot.containsSelf)
			if !snapshot.containsSelf {
				logger.WarnContext(ctx, "Ignoring DNS response that omits this pod",
					"attempt", attempt, "pod_ip", c.pod.address)
			} else if len(snapshot.peers) > 0 {
				return snapshot.peers, nil
			}
		}
		if attempt < c.discovery.attempts {
			logger.InfoContext(ctx, "Retrying peer discovery", "attempt", attempt, "delay", c.discovery.delay)
			if err := waitForRetry(ctx, c.discovery.delay); err != nil {
				return nil, err
			}
		}
	}

	if c.canBootstrap() && snapshot.containsSelf {
		logger.WarnContext(ctx, "Bootstrapping from a self-only DNS response; cluster membership is unverified")
		return nil, nil
	}
	return nil, fmt.Errorf("could not discover join targets or establish bootstrap eligibility through %s; exiting so Kubernetes can restart the container and retry discovery", c.service)
}

func (d peerDiscovery) query(ctx context.Context, c config) (peerSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, c.discovery.timeout)
	defer cancel()

	addresses, err := d.lookup(ctx, c.service)
	if ctx.Err() != nil {
		return peerSnapshot{}, ctx.Err()
	}
	if err != nil {
		return peerSnapshot{}, err
	}
	return snapshotPeers(addresses, c.pod.address, c.clusterPort), nil
}

func snapshotPeers(addresses []netip.Addr, self netip.Addr, port uint16) peerSnapshot {
	var snapshot peerSnapshot
	seen := make(map[netip.Addr]struct{}, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if address == self.Unmap() {
			snapshot.containsSelf = true
			continue
		}
		if _, exists := seen[address]; exists {
			continue
		}
		seen[address] = struct{}{}
		snapshot.peers = append(snapshot.peers, netip.AddrPortFrom(address, port).String())
	}
	return snapshot
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
