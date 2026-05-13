package archivingcache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/dns-resolver/plugin/pkg/serviceconnections"
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

// init registers this plugin within the Caddy plugin framework. It uses "archivingcache" as the
// name, and couples it to the Action "setup".
func init() {
	plugin.Register("archivingcache", setup)
}

// setup is the function that gets called when the config parser see the token "archivingcache". Setup is responsible
// for parsing any extra options the archive plugin may have. The first token this function sees is "archivingcache".
func setup(c *caddy.Controller) error {
	a, err := parseArchivingCache(c)
	if err != nil {
		return plugin.Error("archivingcache", err)
	}

	c.OnStartup(a.OnStartup)
	c.OnShutdown(a.OnShutdown)

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		a.Next = next
		return a
	})

	return nil
}

// OnStartup connects to content writer and log writer.
func (a *ArchivingCache) OnStartup() error {
	if a.cache == nil {
		cache, err := NewOlricCache(a.olricAddresses, a.olricDMap, a.eviction)
		if err != nil {
			return fmt.Errorf("failed to connect to olric: %w", err)
		}
		a.cache = cache
		log.Infof("Connected to olric at: %s (dmap=%s)", strings.Join(a.olricAddresses, ","), a.olricDMap)
	}

	if a.contentWriter == nil {
		return nil
	}
	if err := a.contentWriter.Connect(); err != nil {
		return fmt.Errorf("failed to connect to cws: %w", err)
	}
	log.Infof("Connected to cws at: %s", a.contentWriter.Addr())

	if a.logWriter == nil {
		return nil
	}
	if err := a.logWriter.Connect(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Infof("Connected to log client at: %s", a.logWriter.Addr())

	return nil
}

// OnShutdown closes connections to content writer and log writer.
func (a *ArchivingCache) OnShutdown() (err error) {
	if a.contentWriter != nil {
		_ = a.contentWriter.Close()
	}
	if a.logWriter != nil {
		_ = a.logWriter.Close()
	}
	if a.cache != nil {
		_ = a.cache.Close(context.Background())
	}
	return
}

func parseArchivingCache(c *caddy.Controller) (*ArchivingCache, error) {
	eviction := defaultEviction
	olricAddresses := []string{defaultOlricAddress}
	olricAddressesConfigured := false
	olricDMap := defaultOlricDMap
	var contentWriterHost string
	var contentWriterPort int
	var logHost string
	var logPort int

	j := 0
	for c.Next() { // 'archivingcache'
		if j > 0 {
			return nil, plugin.ErrOnce
		}
		j++

		if len(c.RemainingArgs()) > 0 {
			return nil, c.Errf("unknown property '%s'", c.Val())
		}
		for c.NextBlock() {
			switch c.Val() {
			case "eviction":
				if arg, err := getArg(c); err != nil {
					return nil, err
				} else {
					duration, err := time.ParseDuration(arg)
					if err != nil {
						return nil, err
					}
					eviction = duration
				}
			case "olricAddress":
				args, err := getArgs(c)
				if err != nil {
					return nil, err
				}
				if !olricAddressesConfigured {
					olricAddresses = nil
					olricAddressesConfigured = true
				}
				for _, arg := range args {
					for _, address := range strings.Split(arg, ",") {
						address = strings.TrimSpace(address)
						if address != "" {
							olricAddresses = append(olricAddresses, address)
						}
					}
				}
				if len(olricAddresses) == 0 {
					return nil, fmt.Errorf("missing value for %q", c.Val())
				}
			case "olricDmap":
				if arg, err := getArg(c); err != nil {
					return nil, err
				} else {
					olricDMap = strings.TrimSpace(arg)
					if olricDMap == "" {
						return nil, fmt.Errorf("missing value for %q", c.Val())
					}
				}
			case "contentWriterHost":
				if arg, err := getArg(c); err != nil {
					return nil, err
				} else {
					contentWriterHost = arg
				}
			case "contentWriterPort":
				if arg, err := getArg(c); err != nil {
					return nil, err
				} else {
					contentWriterPort, err = strconv.Atoi(arg)
					if err != nil {
						return nil, err
					}
				}
			case "logHost":
				if arg, err := getArg(c); err != nil {
					return nil, err
				} else {
					logHost = arg
				}
			case "logPort":
				if arg, err := getArg(c); err != nil {
					return nil, err
				} else {
					logPort, err = strconv.Atoi(arg)
					if err != nil {
						return nil, err
					}
				}
			default:
				return nil, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	var lw *LogWriterClient
	var cw *ContentWriterClient

	if logHost != "" {
		lw = NewLogWriterClient(
			serviceconnections.WithHost(logHost),
			serviceconnections.WithPort(logPort),
		)
	}
	if contentWriterHost != "" {
		cw = NewContentWriterClient(
			serviceconnections.WithHost(contentWriterHost),
			serviceconnections.WithPort(contentWriterPort),
		)
	}
	a := NewArchivingCache(nil, lw, cw)
	a.eviction = eviction
	a.olricAddresses = olricAddresses
	a.olricDMap = olricDMap
	return a, nil
}

func getArg(c *caddy.Controller) (string, error) {
	args, err := getArgs(c)
	if err != nil {
		return "", err
	}
	return args[0], nil
}

func getArgs(c *caddy.Controller) ([]string, error) {
	args := c.RemainingArgs()
	if len(args) == 0 {
		return nil, fmt.Errorf("missing value for %q", c.Val())
	}
	if len(args) > 1 && c.Val() != "olricAddress" {
		return nil, c.ArgErr()
	}
	return args, nil
}

const (
	defaultEviction     = 1 * time.Hour
	defaultOlricAddress = "localhost:3320"
	defaultOlricDMap    = "dns-resolver-archivingcache"
)
