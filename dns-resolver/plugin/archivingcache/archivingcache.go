// Package archivingcache caches DNS responses in Olric and archives responses per collection.
package archivingcache

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/dns-resolver/plugin/resolve"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/plugin/metrics"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/plugin/pkg/nonwriter"
	"github.com/coredns/coredns/plugin/pkg/response"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
)

const (
	cacheKeyVersion         = "v3"
	archiveMarkerKeyVersion = "v1"
	maximumCacheTTL         = 7 * 24 * time.Hour
	failureCacheTTL         = 5 * time.Second
)

var log = clog.NewWithPlugin("archivingcache")

// ArchivingCache is a CoreDNS plugin.
type ArchivingCache struct {
	Next            plugin.Handler
	cache           Cachier
	olricAddresses  []string
	olricDMap       string
	contentWriter   *ContentWriterClient
	logWriter       *LogWriterClient
	archiveResponse func(*request.Request, *dns.Msg, string, string, string, time.Time) error
	now             func() time.Time
	singleflight.Group
}

// NewArchivingCache returns a new instance of ArchivingCache.
func NewArchivingCache(cache Cachier, lw *LogWriterClient, cw *ContentWriterClient) *ArchivingCache {
	a := &ArchivingCache{
		cache:         cache,
		logWriter:     lw,
		contentWriter: cw,
		now:           time.Now,
	}
	if cw != nil {
		a.archiveResponse = a.archive
	}
	return a
}

func (a *ArchivingCache) Ready() bool {
	return a.cache != nil && clientReady(a.logWriter) && clientReady(a.contentWriter)
}

// ServeDNS implements the plugin.Handler interface.
func (a *ArchivingCache) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	key, ok := cacheKey(r)
	if !ok {
		return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
	}

	requestStart := a.now().UTC()
	server := metrics.WithServer(ctx)
	entry, err := a.get(ctx, key, requestStart)
	if err == nil {
		log.Debugf("Cache hit: %s", key)
		CacheHits.WithLabelValues(server, Success).Inc()
	} else {
		log.Debugf("Cache miss: %s: %v", key, err)
		CacheMisses.WithLabelValues(server).Inc()

		value, lookupErr, _ := a.Do(key, func() (interface{}, error) {
			now := a.now().UTC()
			if cached, cacheErr := a.get(ctx, key, now); cacheErr == nil {
				return cached, nil
			}
			return a.fetch(ctx, w, r, key, now), nil
		})
		if lookupErr != nil {
			return dns.RcodeServerFailure, lookupErr
		}
		entry = value.(*CacheEntry)
	}

	now := a.now().UTC()
	msg := agedResponse(entry, now)
	msg.SetRcode(r, msg.Rcode)
	a.archiveForCollection(ctx, w, r, key, entry, msg, requestStart)

	_ = w.WriteMsg(msg)
	return 0, nil
}

func (a *ArchivingCache) fetch(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, key string, storedAt time.Time) *CacheEntry {
	state := &request.Request{Req: r, W: w}
	nw := nonwriter.New(w)
	_, nextErr := plugin.NextOrFailure(a.Name(), a.Next, ctx, nw, r)
	msg := nw.Msg
	if nextErr != nil {
		log.Errorf("DNS resolution failed for %s: %v", state.Name(), nextErr)
		msg = new(dns.Msg).SetRcode(r, dns.RcodeServerFailure)
	} else if msg == nil {
		msg = new(dns.Msg).SetRcode(r, dns.RcodeServerFailure)
	}

	msg = msg.Copy()
	msg.SetRcode(r, msg.Rcode)
	responseType, _ := response.Typify(msg, storedAt)
	ttl, cacheable := cacheTTL(msg, responseType)
	entry := &CacheEntry{
		StoredAt:  storedAt,
		ExpiresAt: storedAt,
		ProxyAddr: upstreamAddress(ctx),
		Msg:       msg,
	}
	if !cacheable {
		return entry
	}

	entry.ExpiresAt = storedAt.Add(ttl)
	if err := a.set(ctx, key, entry, ttl); err != nil {
		log.Errorf("%s: %v", key, err)
	}
	return entry
}

func (a *ArchivingCache) archiveForCollection(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, key string, entry *CacheEntry, msg *dns.Msg, fetchStart time.Time) {
	if a.archiveResponse == nil {
		return
	}
	collectionID, ok := ctx.Value(resolve.CollectionIdKey{}).(string)
	now := a.now().UTC()
	if !ok || collectionID == "" || !isArchivable(msg, now) {
		return
	}

	markerKey := archiveMarkerKey(key, collectionID)
	generation := strconv.FormatInt(entry.StoredAt.UnixNano(), 10)
	if entry.ExpiresAt.After(now) {
		marker, err := a.cache.Get(ctx, markerKey)
		switch {
		case err == nil && string(marker) == generation:
			return
		case err != nil && !errors.Is(err, ErrKeyNotFound):
			log.Errorf("failed to read archive marker %s: %v", markerKey, err)
		}
	}

	state := &request.Request{Req: r, W: w}
	executionID, _ := ctx.Value(resolve.ExecutionIdKey{}).(string)
	if err := a.archiveResponse(state, msg, executionID, collectionID, entry.ProxyAddr, fetchStart); err != nil {
		log.Errorf("%s: %v", key, err)
		return
	}

	remaining := entry.ExpiresAt.Sub(a.now().UTC())
	if remaining <= 0 {
		return
	}
	if err := a.cache.Set(ctx, markerKey, []byte(generation), remaining); err != nil {
		log.Errorf("failed to write archive marker %s: %v", markerKey, err)
	}
}

func (a *ArchivingCache) set(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error {
	packed, err := entry.pack()
	if err != nil {
		return fmt.Errorf("failed to serialize cache entry: %w", err)
	}

	if err := a.cache.Set(ctx, key, packed, ttl); err != nil {
		return fmt.Errorf("failed to cache entry: %s: %w", key, err)
	}
	log.Debugf("Cache set: %s, %v", key, entry)
	return nil
}

func (a *ArchivingCache) get(ctx context.Context, key string, now time.Time) (*CacheEntry, error) {
	packed, err := a.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	entry := new(CacheEntry)
	if err := entry.unpack(packed); err != nil {
		return nil, err
	}
	if !now.Before(entry.ExpiresAt) {
		return nil, ErrKeyNotFound
	}
	return entry, nil
}

func cacheKey(r *dns.Msg) (string, bool) {
	if r == nil || r.Opcode != dns.OpcodeQuery || len(r.Question) != 1 {
		return "", false
	}
	if opt := r.IsEdns0(); opt != nil && len(opt.Option) != 0 {
		return "", false
	}

	question := r.Question[0]
	do := false
	if opt := r.IsEdns0(); opt != nil {
		do = opt.Do()
	}
	return fmt.Sprintf("dns|%s|%s|%d|%d|%t|%t", cacheKeyVersion, strings.ToLower(dns.Fqdn(question.Name)), question.Qtype, question.Qclass, do, r.CheckingDisabled), true
}

func archiveMarkerKey(cacheKey, collectionID string) string {
	return fmt.Sprintf("dns-archive|%s|%s|%s", archiveMarkerKeyVersion, collectionID, cacheKey)
}

func cacheTTL(msg *dns.Msg, responseType response.Type) (time.Duration, bool) {
	if msg == nil || msg.Truncated {
		return 0, false
	}

	switch responseType {
	case response.ServerError:
		return failureCacheTTL, true
	case response.NameError, response.NoData:
		if !normalizeNegativeTTL(msg) {
			return 0, false
		}
	case response.NoError, response.Delegation:
		// Positive and delegation responses use their record TTLs below.
	default:
		return 0, false
	}

	capResponseTTLs(msg, maximumCacheTTL)
	ttl, ok := minimumRecordTTL(msg)
	if !ok || ttl <= 0 {
		return 0, false
	}
	return ttl, true
}

func normalizeNegativeTTL(msg *dns.Msg) bool {
	found := false
	for _, rr := range msg.Ns {
		soa, ok := rr.(*dns.SOA)
		if !ok {
			continue
		}
		found = true
		if soa.Minttl < soa.Hdr.Ttl {
			soa.Hdr.Ttl = soa.Minttl
		}
	}
	return found
}

func capResponseTTLs(msg *dns.Msg, maximum time.Duration) {
	maxSeconds := uint32(maximum / time.Second)
	for _, section := range [][]dns.RR{msg.Answer, msg.Ns, msg.Extra} {
		for _, rr := range section {
			if rr.Header().Rrtype != dns.TypeOPT && rr.Header().Ttl > maxSeconds {
				rr.Header().Ttl = maxSeconds
			}
		}
	}
}

func minimumRecordTTL(msg *dns.Msg) (time.Duration, bool) {
	var minimum uint32
	found := false
	for _, section := range [][]dns.RR{msg.Answer, msg.Ns, msg.Extra} {
		for _, rr := range section {
			if rr.Header().Rrtype == dns.TypeOPT {
				continue
			}
			if !found || rr.Header().Ttl < minimum {
				minimum = rr.Header().Ttl
				found = true
			}
		}
	}
	return time.Duration(minimum) * time.Second, found
}

func agedResponse(entry *CacheEntry, now time.Time) *dns.Msg {
	msg := entry.Msg.Copy()
	age := max(now.Sub(entry.StoredAt), 0)
	elapsed := uint32(age / time.Second)

	for _, section := range [][]dns.RR{msg.Answer, msg.Ns, msg.Extra} {
		for _, rr := range section {
			if rr.Header().Rrtype == dns.TypeOPT {
				continue
			}
			if rr.Header().Ttl > elapsed {
				rr.Header().Ttl -= elapsed
			} else {
				rr.Header().Ttl = 0
			}
		}
	}
	return msg
}

func isArchivable(msg *dns.Msg, now time.Time) bool {
	t, _ := response.Typify(msg, now)
	return t == response.NoError && len(msg.Answer) > 0
}

func upstreamAddress(ctx context.Context) string {
	upstream := metadata.ValueFunc(ctx, "forward/upstream")
	if upstream == nil {
		return ""
	}

	proxyAddr := upstream()
	if proxyAddr == "" {
		return ""
	}
	proxyIPAddr, err := parseHostPortOrIP(proxyAddr)
	if err != nil {
		log.Errorf("failed to parse proxy address %q as host:port pair or IP address: %v", proxyAddr, err)
		return ""
	}
	return proxyIPAddr
}

type readyClient interface {
	Ready() bool
}

func clientReady(client readyClient) bool {
	if client == nil {
		return true
	}
	return client.Ready()
}

// archive writes a WARC record and a crawl log.
func (a *ArchivingCache) archive(state *request.Request, msg *dns.Msg, executionID string, collectionID string, proxyAddr string, fetchStart time.Time) error {
	if a.contentWriter == nil {
		return nil
	}
	if len(msg.Answer) == 0 {
		return nil
	}

	fetchDuration := max(a.now().UTC().Sub(fetchStart), 0)
	fetchDurationMs := (fetchDuration.Nanoseconds() + 500000) / 1000000
	requestedHost := strings.TrimSuffix(state.Name(), ".")

	payload := fmt.Appendf(nil, "%d%02d%02d%02d%02d%02d\n%s\n",
		fetchStart.Year(), fetchStart.Month(), fetchStart.Day(),
		fetchStart.Hour(), fetchStart.Minute(), fetchStart.Second(), msg.Answer[0])
	size := len(payload)

	reply, err := a.contentWriter.WriteRecord(payload, fetchStart, requestedHost, proxyAddr, executionID, collectionID)
	if err != nil {
		return fmt.Errorf("failed to write WARC record: %w", err)
	}

	if a.logWriter == nil {
		return nil
	}
	if err := a.logWriter.WriteCrawlLog(reply.GetMeta().GetRecordMeta()[0], size, requestedHost, fetchStart, fetchDurationMs, proxyAddr, executionID); err != nil {
		return fmt.Errorf("failed to write crawl log: %w", err)
	}
	return nil
}

// Name implements the Handler interface.
func (a *ArchivingCache) Name() string { return "archivingcache" }

// parseHostPortOrIP parses a host:port pair or IP address into an IP address.
func parseHostPortOrIP(addr string) (string, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		ip := net.ParseIP(addr)
		if ip == nil {
			return "", err
		}
		return ip.String(), nil
	}
	return host, nil
}
