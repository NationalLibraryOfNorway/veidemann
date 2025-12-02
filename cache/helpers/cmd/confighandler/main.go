package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/cache/helpers/discovery"
	"github.com/NationalLibraryOfNorway/veidemann/cache/helpers/iputil"
	"github.com/sevlyar/go-daemon"
)

func main() {
	r := new(rewriter)

	logger := log.New(os.Stderr, "[ConfigHandler] ", log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)

	flag.BoolVar(&r.balancer, "b", false, "Set to true to configure squid as balancer")
	flag.Parse()

	// configure rewriter
	r.configPath = "/etc/squid/conf.d/veidemann.conf"
	if r.balancer {
		logger.Print("Configuring squid as balancer")
		r.templatePath = "/etc/squid/squid-balancer.conf.template"
		if d, err := discovery.NewDiscovery(); err != nil {
			logger.Fatalf("Failed to initialize discovery: %v", err)
		} else {
			r.discovery = d
		}
	} else {
		logger.Println("Configuring squid as cache")
		r.templatePath = "/etc/squid/squid.conf.template"
	}

	// initial config rewrite
	// Note: this will run twice (both in the parent and the child process)
	if err := r.rewriteConfig(); err != nil {
		logger.Fatalf("Failed to initialize configuration: %v", err)
	}

	context := &daemon.Context{LogFileName: "/dev/stderr"}
	child, err := context.Reborn()
	if err != nil {
		logger.Fatalf("Failed to create daemon process: %v", err)
	}

	if child != nil {
		// This code is run in parent process
		logger.Printf("Configuration initialized (%s)", r.configPath)
	} else {
		// This code is run in forked child
		logger.Println("Daemon started")
		defer func() {
			_ = context.Release()
			logger.Println("Daemon stopped")
		}()
		for {
			time.Sleep(5 * time.Second)
			if err := r.rewriteConfig(); err != nil {
				logger.Printf("Failed to rewrite configuration: %v", err)
			}
			if r.changes {
				logger.Printf("Reconfiguring squid...")
				if err := reconfigureSquid(); err != nil {
					logger.Printf("Error reconfiguring squid: %v", err)
				}
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
	changes        bool
}

func (r *rewriter) rewriteConfig() error {
	r.changes = false

	dnsServers := r.getDnsServersString()
	if dnsServers == "" {
		return fmt.Errorf("no dns servers configured")
	}

	parents := ""
	if r.balancer {
		var err error
		parents, err = r.getParents()
		if err != nil {
			return fmt.Errorf("failed to get parents: %w", err)
		}
		if parents == "" {
			return fmt.Errorf("found no parents")
		}
	}
	if parents != r.lastParents || dnsServers != r.lastDnsServers {
		// read template
		b, err := os.ReadFile(r.templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template (%s): %w", r.templatePath, err)
		}
		// substitute template variables
		conf := string(b)
		conf = strings.Replace(conf, "${DNS_IP}", dnsServers, 1)
		if r.balancer {
			conf = strings.Replace(conf, "${PARENTS}", parents, 1)
		}
		// write config
		if err := os.WriteFile(r.configPath, []byte(conf), 0644); err != nil {
			return fmt.Errorf("failed to write config (%s): %w", r.configPath, err)
		}
		r.changes = true
	}

	r.lastParents = parents
	r.lastDnsServers = dnsServers
	return nil
}

func (r *rewriter) getParents() (string, error) {
	parents, err := r.discovery.GetParents()
	if err != nil {
		return "", err
	}
	var peers string
	for _, parent := range parents {
		peers += fmt.Sprintf("cache_peer %v parent 3128 0 carp no-digest\n", parent)
	}
	return peers, nil
}

func (r *rewriter) getDnsServersString() string {
	var dnsServers string

	dnsEnv, _ := os.LookupEnv("DNS_SERVERS")
	dns := strings.Split(dnsEnv, " ")

	for _, d := range dns {
		ip, _, err := iputil.IPAndPortForAddr(strings.TrimSpace(d), 53)
		if err == nil {
			dnsServers += ip + " "
		}
	}
	return dnsServers
}

func reconfigureSquid() error {
	cmd := exec.Command("squid", "-k", "reconfigure")
	// ignore error returned if wait was already called
	defer func() { _ = cmd.Wait() }()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stderr [%s]: %w", cmd.String(), err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stdout [%s]: %w", cmd.String(), err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start [%s]", cmd.String())
	}
	if slurp, err := io.ReadAll(stdout); err != nil {
		return fmt.Errorf("failed to read standard out [%s]", cmd.String())
	} else if len(slurp) > 0 {
		log.Printf("%s", slurp)
	}
	if slurp, err := io.ReadAll(stderr); err != nil {
		return fmt.Errorf("failed to read standard err [%s]", cmd.String())
	} else if len(slurp) > 0 {
		log.Printf("%s", slurp)
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
