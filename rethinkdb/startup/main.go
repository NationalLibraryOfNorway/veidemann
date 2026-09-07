package main

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.LookupEnv, os.Args[1:], logger); err != nil {
		logger.Error("RethinkDB startup failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, env environment, args []string, logger *slog.Logger) error {
	cfg, err := loadConfig(env, args)
	if err != nil {
		return err
	}
	logger = logger.With("pod", cfg.pod.name, "namespace", cfg.pod.namespace)
	logger.InfoContext(ctx, "Preparing RethinkDB startup",
		"pod_ip", cfg.pod.address, "statefulset", cfg.pod.statefulSet,
		"proxy", cfg.proxy, "cluster_port", cfg.clusterPort,
		"canonical_address", cfg.canonicalAddress())

	// StrictErrors rejects partial answers when either A or AAAA queries suffer
	// temporary failures. PreferGo keeps lookup cancellation independent of libc.
	resolver := &net.Resolver{PreferGo: true, StrictErrors: true}
	discovery := peerDiscovery{
		lookup: func(ctx context.Context, host string) ([]netip.Addr, error) {
			return resolver.LookupNetIP(ctx, "ip", host)
		},
		log: logger,
	}
	peers, err := discovery.joinTargets(ctx, cfg)
	if err != nil {
		return err
	}
	path, err := exec.LookPath("rethinkdb")
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	logger.InfoContext(ctx, "Starting RethinkDB", "join_endpoints", peers)
	// Replace this process so tini supervises RethinkDB directly. Never log the
	// full command or config: both contain the initial password.
	return syscall.Exec(path, cfg.command(peers), os.Environ())
}
