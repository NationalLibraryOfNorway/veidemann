package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/cache/helpers/discovery"
	"github.com/NationalLibraryOfNorway/veidemann/cache/helpers/iputil"
)

func main() {
	var (
		isBalancer  bool
		configPath  string
		readyFile   string
		tlsCertFile string
		tlsKeyFile  string
		interval    time.Duration
		minReconf   time.Duration
	)

	flag.BoolVar(&isBalancer, "b", false, "Configure squid as balancer")
	flag.StringVar(&configPath, "config", "/etc/squid/conf.d/90-role.conf", "Output config path")
	flag.StringVar(&readyFile, "ready-file", "/run/confighandler.ready", "Write this file after initial successful render (empty disables)")
	flag.StringVar(&tlsCertFile, "tls-cert-file", "/tls-certificates/tls.crt", "Parent TLS certificate path")
	flag.StringVar(&tlsKeyFile, "tls-key-file", "/tls-certificates/tls.key", "Parent TLS private key path")
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
		balancer:    isBalancer,
		configPath:  configPath,
		tlsCertFile: tlsCertFile,
		tlsKeyFile:  tlsKeyFile,
		runner:      execSquidRunner{},
		now:         time.Now,
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

	t := time.NewTicker(interval)
	defer t.Stop()

	pendingReconfigure := false
	lastReconf := time.Now()

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

			if changed {
				pendingReconfigure = true
			}

			if !pendingReconfigure {
				continue
			}

			remaining := minReconf - time.Since(lastReconf)
			if remaining > 0 {
				log.Debug(
					"Reconfigure pending",
					"remaining", remaining.String(),
				)
				continue
			}

			out, err := r.runner.reconfigure()
			if err != nil {
				log.Error(
					"Squid reconfigure failed",
					"error", err,
					"output", out,
				)
				// Keep pendingReconfigure=true so it retries.
				continue
			}

			pendingReconfigure = false
			lastReconf = time.Now()

			if out != "" {
				log.Info("Squid reconfigured", "output", out)
			} else {
				log.Info("Squid reconfigured")
			}
		}
	}
}

type rewriter struct {
	lastParents        string
	lastDnsServers     string
	lastTLSFingerprint string
	discovery          *discovery.Discovery
	getParentsFunc     func() (string, error)
	balancer           bool
	templatePath       string
	configPath         string
	tlsCertFile        string
	tlsKeyFile         string
	runner             squidRunner
	now                func() time.Time
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

	tlsFingerprint := ""
	if !r.balancer {
		var err error
		tlsFingerprint, err = validateTLSFiles(
			r.tlsCertFile,
			r.tlsKeyFile,
			r.now(),
		)
		if err != nil {
			return false, fmt.Errorf("validate parent TLS material: %w", err)
		}
	}

	if parents == r.lastParents &&
		dnsServers == r.lastDnsServers &&
		tlsFingerprint == r.lastTLSFingerprint {
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
	} else {
		conf = strings.ReplaceAll(conf, "${TLS_CERT_FILE}", r.tlsCertFile)
		conf = strings.ReplaceAll(conf, "${TLS_KEY_FILE}", r.tlsKeyFile)
	}

	previous, readErr := os.ReadFile(r.configPath)
	previousExists := readErr == nil

	if readErr != nil && !os.IsNotExist(readErr) {
		return false, fmt.Errorf(
			"read existing config (%s): %w",
			r.configPath,
			readErr,
		)
	}

	if err := writeFileAtomic(r.configPath, []byte(conf), 0644); err != nil {
		return false, fmt.Errorf(
			"write config (%s): %w",
			r.configPath,
			err,
		)
	}

	validationOutput, err := r.runner.validateConfig()
	if err != nil {
		if previousExists {
			if restoreErr := writeFileAtomic(
				r.configPath,
				previous,
				0644,
			); restoreErr != nil {
				return false, fmt.Errorf(
					"generated config invalid: %w; output: %q; "+
						"failed restoring previous config: %v",
					err,
					validationOutput,
					restoreErr,
				)
			}
		} else {
			if removeErr := os.Remove(r.configPath); removeErr != nil &&
				!os.IsNotExist(removeErr) {
				return false, fmt.Errorf(
					"generated config invalid: %w; output: %q; "+
						"failed removing invalid config: %v",
					err,
					validationOutput,
					removeErr,
				)
			}
		}

		return false, fmt.Errorf(
			"generated config invalid: %w; output: %q",
			err,
			validationOutput,
		)
	}

	r.lastParents = parents
	r.lastDnsServers = dnsServers
	r.lastTLSFingerprint = tlsFingerprint
	return true, nil
}

func (r *rewriter) getParents() (string, error) {
	if r.getParentsFunc != nil {
		return r.getParentsFunc()
	}

	parents, err := r.discovery.GetParents()
	if err != nil {
		return "", err
	}

	// Stable output prevents unnecessary reconfiguration when discovery
	// returns the same parents in a different order.
	slices.Sort(parents)
	parents = slices.Compact(parents)

	var b strings.Builder
	for _, parent := range parents {
		fmt.Fprintf(
			&b,
			"cache_peer %s parent 3128 0 "+
				"carp no-query no-digest proxy-only no-netdb-exchange "+
				"connect-timeout=5 connect-fail-limit=2\n",
			parent,
		)
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

type squidRunner interface {
	validateConfig() (string, error)
	reconfigure() (string, error)
}

type execSquidRunner struct{}

func (execSquidRunner) validateConfig() (string, error) {
	return validateSquidConfig()
}

func validateSquidConfig() (string, error) {
	out, err := exec.Command(
		"squid",
		"-k",
		"parse",
		"-f",
		"/etc/squid/squid.conf",
	).CombinedOutput()

	return strings.TrimSpace(string(out)), err
}

func (execSquidRunner) reconfigure() (string, error) {
	return reconfigureSquid()
}

func reconfigureSquid() (string, error) {
	out, err := exec.Command("squid", "-k", "reconfigure").CombinedOutput()
	if len(out) > 0 {
		return string(out), err
	}
	return "", err
}

func validateTLSFiles(certFile, keyFile string, now time.Time) (string, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return "", fmt.Errorf("read certificate (%s): %w", certFile, err)
	}
	if len(certPEM) == 0 {
		return "", fmt.Errorf("certificate (%s) is empty", certFile)
	}

	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return "", fmt.Errorf("read private key (%s): %w", keyFile, err)
	}
	if len(keyPEM) == 0 {
		return "", fmt.Errorf("private key (%s) is empty", keyFile)
	}

	if err := validateTLSMaterial(certPEM, keyPEM, now); err != nil {
		return "", err
	}

	certFingerprint := sha256.Sum256(certPEM)
	keyFingerprint := sha256.Sum256(keyPEM)
	return hex.EncodeToString(certFingerprint[:]) + ":" +
		hex.EncodeToString(keyFingerprint[:]), nil
}

func validateTLSMaterial(certPEM, keyPEM []byte, now time.Time) error {
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("load certificate/key pair: %w", err)
	}
	if len(pair.Certificate) == 0 {
		return fmt.Errorf("certificate file contains no certificates")
	}

	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return fmt.Errorf("parse leaf certificate: %w", err)
	}
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not valid before %s", cert.NotBefore.Format(time.RFC3339))
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired at %s", cert.NotAfter.Format(time.RFC3339))
	}
	if cert.IsCA {
		return fmt.Errorf("certificate must not be a CA")
	}

	serverAuth := false
	for _, usage := range cert.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth || usage == x509.ExtKeyUsageAny {
			serverAuth = true
			break
		}
	}
	if !serverAuth {
		return fmt.Errorf("certificate does not permit TLS server authentication")
	}
	if cert.KeyUsage != 0 && cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return fmt.Errorf("certificate does not permit digital signatures")
	}

	return nil
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
