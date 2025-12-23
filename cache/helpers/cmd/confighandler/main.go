package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/cache/helpers/discovery"
	"github.com/NationalLibraryOfNorway/veidemann/cache/helpers/iputil"
)

func main() {
	var (
		isBalancer bool
		configPath string
		readyFile  string
		interval   time.Duration
		minReconf  time.Duration
	)

	flag.BoolVar(&isBalancer, "b", false, "Configure squid as balancer")
	flag.StringVar(&configPath, "config", "/etc/squid/conf.d/90-role.conf", "Output config path")
	flag.StringVar(&readyFile, "ready-file", "/run/confighandler.ready", "Write this file after initial successful render (empty disables)")
	flag.DurationVar(&interval, "interval", 5*time.Second, "Rewrite check interval")
	flag.DurationVar(&minReconf, "min-reconfigure-interval", 30*time.Second, "Minimum interval between squid reconfigure calls")
	flag.Parse()

	mode := "cache"
	if isBalancer {
		mode = "balancer"
	}

	log := slog.With("daemon", "confighandler", "pid", os.Getpid(), "mode", mode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := &rewriter{
		balancer:   isBalancer,
		configPath: configPath,
	}

	if r.balancer {
		disc, err := discovery.NewDiscovery()
		if err != nil {
			log.Error("Failed to create discovery", "error", err)
			os.Exit(1)
		}
		r.discovery = disc
		r.templatePath = "/etc/squid/squid-balancer.conf.template"
	} else {
		r.templatePath = "/etc/squid/squid.conf.template"
	}

	if err := run(ctx, log, r, interval, minReconf, readyFile); err != nil {
		// Context cancellation is a normal shutdown.
		if ctx.Err() != nil {
			log.Info("Shutting down", "reason", ctx.Err().Error())
			return
		}
		log.Error("Exited with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger, r *rewriter, interval, minReconf time.Duration, readyFile string) error {
	changed, err := r.rewriteConfig()
	if err != nil {
		return fmt.Errorf("initial rewrite failed: %w", err)
	}

	if changed {
		log.Info("Initial config rendered", "path", r.configPath)
	} else {
		log.Info("Initial config unchanged", "path", r.configPath)
	}

	// Signal readiness to entrypoint/supervisor.
	if readyFile != "" {
		if err := os.MkdirAll(filepath.Dir(readyFile), 0755); err != nil {
			return fmt.Errorf("create ready dir: %w", err)
		}
		if err := writeFileAtomic(readyFile, []byte("ok\n"), 0644); err != nil {
			return fmt.Errorf("write ready file: %w", err)
		}
	}

	// Decide initial reconfigure behavior:
	// - If you want to reconfigure immediately after first change, set lastReconf = time.Time{}.
	// - Here: avoid immediate reconfigure right after initial render.
	lastReconf := time.Now()

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-t.C:
			changed, err := r.rewriteConfig()
			if err != nil {
				log.Error("Rewrite failed", "error", err)
				continue
			}
			if !changed {
				continue
			}

			if time.Since(lastReconf) < minReconf {
				log.Debug("Change detected but reconfigure throttled", "since_last", time.Since(lastReconf).String())
				continue
			}

			lastReconf = time.Now()
			out, err := reconfigureSquid()
			if err != nil {
				log.Error("Squid reconfigure failed", "error", err, "output", out)
				continue
			}
			if out != "" {
				log.Info("Squid reconfigured", "output", out)
			} else {
				log.Info("Squid reconfigured")
			}
		}
	}
}

type rewriter struct {
	lastParents    string
	lastDnsServers string
	discovery      *discovery.Discovery
	balancer       bool
	templatePath   string
	configPath     string
}

func (r *rewriter) rewriteConfig() (bool, error) {
	dnsServers := r.getDnsServersString()
	if dnsServers == "" {
		return false, fmt.Errorf("no dns servers configured (DNS_SERVERS env empty/invalid)")
	}

	parents := ""
	if r.balancer {
		p, err := r.getParents()
		if err != nil {
			return false, fmt.Errorf("get parents: %w", err)
		}
		if p == "" {
			return false, fmt.Errorf("found no parents")
		}
		parents = p
	}

	if parents == r.lastParents && dnsServers == r.lastDnsServers {
		return false, nil
	}

	b, err := os.ReadFile(r.templatePath)
	if err != nil {
		return false, fmt.Errorf("read template (%s): %w", r.templatePath, err)
	}

	conf := string(b)
	conf = strings.ReplaceAll(conf, "${DNS_IP}", dnsServers)
	if r.balancer {
		conf = strings.ReplaceAll(conf, "${PARENTS}", parents)
	}

	if err := writeFileAtomic(r.configPath, []byte(conf), 0644); err != nil {
		return false, fmt.Errorf("write config (%s): %w", r.configPath, err)
	}

	r.lastParents = parents
	r.lastDnsServers = dnsServers
	return true, nil
}

func (r *rewriter) getParents() (string, error) {
	parents, err := r.discovery.GetParents()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, parent := range parents {
		fmt.Fprintf(&b, "cache_peer %v parent 3128 0 carp no-digest\n", parent)
	}
	return b.String(), nil
}

func (r *rewriter) getDnsServersString() string {
	fields := strings.Fields(os.Getenv("DNS_SERVERS"))
	ips := make([]string, 0, len(fields))
	for _, d := range fields {
		ip, _, err := iputil.IPAndPortForAddr(d, 53)
		if err == nil {
			ips = append(ips, ip)
		}
	}
	return strings.Join(ips, " ")
}

func reconfigureSquid() (string, error) {
	out, err := exec.Command("squid", "-k", "reconfigure").CombinedOutput()
	if len(out) > 0 {
		return string(out), err
	}
	return "", err
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
