package archivingcache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/dns-resolver/plugin/resolve"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

func TestPositiveCacheAgesTTLWithoutWriting(t *testing.T) {
	cache := newTestCache()
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	upstreamCalls := 0

	a := NewArchivingCache(cache, nil, nil)
	a.now = func() time.Time { return now }
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		upstreamCalls++
		msg := new(dns.Msg).SetReply(r)
		msg.Answer = []dns.RR{test.A("example.org. 120 IN A 192.0.2.1")}
		return 0, w.WriteMsg(msg)
	})

	first := query(t, a, context.Background(), "example.org.", dns.TypeA)
	assertAnswerTTL(t, first, 120)
	if cache.sets != 1 {
		t.Fatalf("cache writes after miss = %d, want 1", cache.sets)
	}

	now = now.Add(30 * time.Second)
	second := query(t, a, context.Background(), "example.org.", dns.TypeA)
	assertAnswerTTL(t, second, 90)
	if cache.sets != 1 {
		t.Fatalf("cache writes after hit = %d, want 1", cache.sets)
	}
	if upstreamCalls != 1 {
		t.Fatalf("upstream calls after hit = %d, want 1", upstreamCalls)
	}

	now = now.Add(91 * time.Second)
	third := query(t, a, context.Background(), "example.org.", dns.TypeA)
	assertAnswerTTL(t, third, 120)
	if cache.sets != 2 {
		t.Fatalf("cache writes after refresh = %d, want 2", cache.sets)
	}
	if upstreamCalls != 2 {
		t.Fatalf("upstream calls after refresh = %d, want 2", upstreamCalls)
	}
}

func TestPositiveCacheCapsTTLAtSevenDays(t *testing.T) {
	cache := newTestCache()
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	a := NewArchivingCache(cache, nil, nil)
	a.now = func() time.Time { return now }
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		msg := new(dns.Msg).SetReply(r)
		msg.Answer = []dns.RR{test.A("example.org. 864000 IN A 192.0.2.1")}
		return 0, w.WriteMsg(msg)
	})

	msg := query(t, a, context.Background(), "example.org.", dns.TypeA)
	assertAnswerTTL(t, msg, uint32(maximumCacheTTL/time.Second))
	key, _ := cacheKey(new(dns.Msg).SetQuestion("example.org.", dns.TypeA))
	if got := cache.ttls[key]; got != maximumCacheTTL {
		t.Fatalf("cache TTL = %v, want %v", got, maximumCacheTTL)
	}
}

func TestZeroTTLIsNotCached(t *testing.T) {
	cache := newTestCache()
	upstreamCalls := 0
	a := NewArchivingCache(cache, nil, nil)
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		upstreamCalls++
		msg := new(dns.Msg).SetReply(r)
		msg.Answer = []dns.RR{test.A("example.org. 0 IN A 192.0.2.1")}
		return 0, w.WriteMsg(msg)
	})

	query(t, a, context.Background(), "example.org.", dns.TypeA)
	query(t, a, context.Background(), "example.org.", dns.TypeA)
	if cache.sets != 0 {
		t.Fatalf("cache writes = %d, want 0", cache.sets)
	}
	if upstreamCalls != 2 {
		t.Fatalf("upstream calls = %d, want 2", upstreamCalls)
	}
}

func TestNegativeCacheUsesSOAMinimum(t *testing.T) {
	cache := newTestCache()
	a := NewArchivingCache(cache, nil, nil)
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		msg := new(dns.Msg).SetRcode(r, dns.RcodeNameError)
		soa, err := dns.NewRR("example.org. 120 IN SOA ns.example.org. hostmaster.example.org. 1 3600 600 86400 30")
		if err != nil {
			return dns.RcodeServerFailure, err
		}
		msg.Ns = []dns.RR{soa}
		return 0, w.WriteMsg(msg)
	})

	msg := query(t, a, context.Background(), "missing.example.org.", dns.TypeA)
	if len(msg.Ns) != 1 || msg.Ns[0].Header().Ttl != 30 {
		t.Fatalf("negative SOA TTL = %v, want 30", msg.Ns)
	}
	key, _ := cacheKey(new(dns.Msg).SetQuestion("missing.example.org.", dns.TypeA))
	if got := cache.ttls[key]; got != 30*time.Second {
		t.Fatalf("cache TTL = %v, want 30s", got)
	}
}

func TestNegativeResponseWithoutSOAIsNotCached(t *testing.T) {
	cache := newTestCache()
	upstreamCalls := 0
	a := NewArchivingCache(cache, nil, nil)
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		upstreamCalls++
		return 0, w.WriteMsg(new(dns.Msg).SetRcode(r, dns.RcodeNameError))
	})

	query(t, a, context.Background(), "missing.example.org.", dns.TypeA)
	query(t, a, context.Background(), "missing.example.org.", dns.TypeA)
	if cache.sets != 0 || upstreamCalls != 2 {
		t.Fatalf("cache writes/upstream calls = %d/%d, want 0/2", cache.sets, upstreamCalls)
	}
}

func TestServerFailureIsCachedForFiveSeconds(t *testing.T) {
	cache := newTestCache()
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	upstreamCalls := 0
	a := NewArchivingCache(cache, nil, nil)
	a.now = func() time.Time { return now }
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		upstreamCalls++
		return dns.RcodeServerFailure, w.WriteMsg(new(dns.Msg).SetRcode(r, dns.RcodeServerFailure))
	})

	query(t, a, context.Background(), "example.org.", dns.TypeA)
	now = now.Add(4 * time.Second)
	query(t, a, context.Background(), "example.org.", dns.TypeA)
	if upstreamCalls != 1 || cache.sets != 1 {
		t.Fatalf("within failure TTL calls/writes = %d/%d, want 1/1", upstreamCalls, cache.sets)
	}

	now = now.Add(2 * time.Second)
	query(t, a, context.Background(), "example.org.", dns.TypeA)
	if upstreamCalls != 2 || cache.sets != 2 {
		t.Fatalf("after failure TTL calls/writes = %d/%d, want 2/2", upstreamCalls, cache.sets)
	}
}

func TestUpstreamErrorIsCachedAsServerFailure(t *testing.T) {
	cache := newTestCache()
	upstreamCalls := 0
	a := NewArchivingCache(cache, nil, nil)
	a.Next = test.HandlerFunc(func(_ context.Context, _ dns.ResponseWriter, _ *dns.Msg) (int, error) {
		upstreamCalls++
		return dns.RcodeServerFailure, errors.New("upstream unavailable")
	})

	first := query(t, a, context.Background(), "example.org.", dns.TypeA)
	second := query(t, a, context.Background(), "example.org.", dns.TypeA)
	if first.Rcode != dns.RcodeServerFailure || second.Rcode != dns.RcodeServerFailure {
		t.Fatalf("response codes = %d/%d, want SERVFAIL", first.Rcode, second.Rcode)
	}
	if upstreamCalls != 1 || cache.sets != 1 {
		t.Fatalf("upstream calls/cache writes = %d/%d, want 1/1", upstreamCalls, cache.sets)
	}
}

func TestCollectionArchiveMarkersPreserveImmutableEntry(t *testing.T) {
	cache := newTestCache()
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	upstreamCalls := 0
	archives := 0
	a := NewArchivingCache(cache, nil, nil)
	a.archiveResponse = func(_ *request.Request, _ *dns.Msg, _, _, _ string, _ time.Time) error {
		archives++
		return nil
	}
	a.now = func() time.Time { return now }
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		upstreamCalls++
		msg := new(dns.Msg).SetReply(r)
		msg.Answer = []dns.RR{test.A("example.org. 120 IN A 192.0.2.1")}
		return 0, w.WriteMsg(msg)
	})

	// An ordinary DNS lookup populates the shared entry without archiving.
	query(t, a, context.Background(), "example.org.", dns.TypeA)
	if cache.sets != 1 {
		t.Fatalf("cache writes after DNS miss = %d, want 1", cache.sets)
	}
	key, _ := cacheKey(new(dns.Msg).SetQuestion("example.org.", dns.TypeA))
	originalEntry := append([]byte(nil), cache.entries[key]...)
	entry := new(CacheEntry)
	if err := entry.unpack(originalEntry); err != nil {
		t.Fatal(err)
	}

	now = now.Add(20 * time.Second)
	collectionCtx := context.WithValue(context.Background(), resolve.CollectionIdKey{}, "collection-a")
	query(t, a, collectionCtx, "example.org.", dns.TypeA)
	if cache.sets != 2 {
		t.Fatalf("writes after first collection = %d, want DNS entry plus marker", cache.sets)
	}
	if string(cache.entries[key]) != string(originalEntry) {
		t.Fatal("collection archival rewrote the DNS cache entry")
	}
	markerA := archiveMarkerKey(key, "collection-a")
	if got := cache.ttls[markerA]; got != 100*time.Second {
		t.Fatalf("marker TTL = %v, want 100s", got)
	}
	if got, want := string(cache.entries[markerA]), fmt.Sprint(entry.StoredAt.UnixNano()); got != want {
		t.Fatalf("marker generation = %q, want %q", got, want)
	}

	query(t, a, collectionCtx, "example.org.", dns.TypeA)
	if cache.sets != 2 {
		t.Fatalf("writes after repeated collection hit = %d, want 2", cache.sets)
	}

	now = now.Add(10 * time.Second)
	secondCollectionCtx := context.WithValue(context.Background(), resolve.CollectionIdKey{}, "collection-b")
	query(t, a, secondCollectionCtx, "example.org.", dns.TypeA)
	if cache.sets != 3 {
		t.Fatalf("writes after second collection = %d, want 3", cache.sets)
	}
	markerB := archiveMarkerKey(key, "collection-b")
	if got := cache.ttls[markerB]; got != 90*time.Second {
		t.Fatalf("second marker TTL = %v, want 90s", got)
	}
	if upstreamCalls != 1 {
		t.Fatalf("upstream calls = %d, want 1", upstreamCalls)
	}
	if archives != 2 {
		t.Fatalf("archives = %d, want one per collection", archives)
	}

	// The next DNS generation is archived again and replaces the old marker value.
	oldGeneration := string(cache.entries[markerA])
	now = now.Add(91 * time.Second)
	query(t, a, collectionCtx, "example.org.", dns.TypeA)
	if upstreamCalls != 2 {
		t.Fatalf("upstream calls after refresh = %d, want 2", upstreamCalls)
	}
	if got := string(cache.entries[markerA]); got == oldGeneration {
		t.Fatalf("marker generation was not refreshed: %q", got)
	}
	if archives != 3 {
		t.Fatalf("archives after refresh = %d, want 3", archives)
	}
}

func TestArchiveFailureLeavesNoMarkerAndRetries(t *testing.T) {
	cache := newTestCache()
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	archiveCalls := 0
	a := NewArchivingCache(cache, nil, nil)
	a.now = func() time.Time { return now }
	a.archiveResponse = func(_ *request.Request, _ *dns.Msg, _, _, _ string, _ time.Time) error {
		archiveCalls++
		if archiveCalls == 1 {
			return errors.New("archive unavailable")
		}
		return nil
	}
	a.Next = positiveHandler("example.org. 120 IN A 192.0.2.1")
	ctx := context.WithValue(context.Background(), resolve.CollectionIdKey{}, "collection-a")
	key, _ := cacheKey(new(dns.Msg).SetQuestion("example.org.", dns.TypeA))
	markerKey := archiveMarkerKey(key, "collection-a")

	query(t, a, ctx, "example.org.", dns.TypeA)
	if _, ok := cache.entries[markerKey]; ok {
		t.Fatal("archive failure wrote a marker")
	}
	query(t, a, ctx, "example.org.", dns.TypeA)
	if _, ok := cache.entries[markerKey]; !ok {
		t.Fatal("successful retry did not write a marker")
	}
	query(t, a, ctx, "example.org.", dns.TypeA)
	if archiveCalls != 2 {
		t.Fatalf("archive calls = %d, want failed attempt plus one retry", archiveCalls)
	}
}

func TestArchiveMarkerUsesLifetimeRemainingAfterArchive(t *testing.T) {
	cache := newTestCache()
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	a := NewArchivingCache(cache, nil, nil)
	a.now = func() time.Time { return now }
	a.archiveResponse = func(_ *request.Request, _ *dns.Msg, _, _, _ string, _ time.Time) error {
		now = now.Add(5 * time.Second)
		return nil
	}
	a.Next = positiveHandler("example.org. 120 IN A 192.0.2.1")
	ctx := context.WithValue(context.Background(), resolve.CollectionIdKey{}, "collection-a")
	key, _ := cacheKey(new(dns.Msg).SetQuestion("example.org.", dns.TypeA))

	query(t, a, ctx, "example.org.", dns.TypeA)
	markerKey := archiveMarkerKey(key, "collection-a")
	if got := cache.ttls[markerKey]; got != 115*time.Second {
		t.Fatalf("marker TTL = %v, want 115s remaining after archival", got)
	}
}

func TestArchiveMarkerErrorsDoNotSuppressArchival(t *testing.T) {
	cache := newTestCache()
	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	archiveCalls := 0
	a := NewArchivingCache(cache, nil, nil)
	a.now = func() time.Time { return now }
	a.archiveResponse = func(_ *request.Request, _ *dns.Msg, _, _, _ string, _ time.Time) error {
		archiveCalls++
		return nil
	}
	a.Next = positiveHandler("example.org. 120 IN A 192.0.2.1")
	ctx := context.WithValue(context.Background(), resolve.CollectionIdKey{}, "collection-a")
	key, _ := cacheKey(new(dns.Msg).SetQuestion("example.org.", dns.TypeA))
	markerKey := archiveMarkerKey(key, "collection-a")

	cache.setError[markerKey] = errors.New("marker write unavailable")
	query(t, a, ctx, "example.org.", dns.TypeA)
	if _, ok := cache.entries[markerKey]; ok {
		t.Fatal("failed marker write unexpectedly stored a marker")
	}
	delete(cache.setError, markerKey)
	query(t, a, ctx, "example.org.", dns.TypeA)
	if archiveCalls != 2 {
		t.Fatalf("archive calls after marker write failure = %d, want 2", archiveCalls)
	}

	cache.getError[markerKey] = errors.New("marker read unavailable")
	query(t, a, ctx, "example.org.", dns.TypeA)
	if archiveCalls != 3 {
		t.Fatalf("archive calls after marker read failure = %d, want 3", archiveCalls)
	}
}

func TestUncacheablePositiveResponseArchivesWithoutMarker(t *testing.T) {
	cache := newTestCache()
	archiveCalls := 0
	a := NewArchivingCache(cache, nil, nil)
	a.archiveResponse = func(_ *request.Request, _ *dns.Msg, _, _, _ string, _ time.Time) error {
		archiveCalls++
		return nil
	}
	a.Next = positiveHandler("example.org. 0 IN A 192.0.2.1")
	ctx := context.WithValue(context.Background(), resolve.CollectionIdKey{}, "collection-a")

	query(t, a, ctx, "example.org.", dns.TypeA)
	query(t, a, ctx, "example.org.", dns.TypeA)
	if cache.sets != 0 {
		t.Fatalf("cache writes = %d, want no DNS entry or marker", cache.sets)
	}
	if archiveCalls != 2 {
		t.Fatalf("archive calls = %d, want one per uncacheable response", archiveCalls)
	}
}

func TestConcurrentMissesShareOneUpstreamLookup(t *testing.T) {
	const concurrency = 20
	baseCache := newTestCache()
	cache := &countingCache{
		testCache: baseCache,
		waitFor:   concurrency + 1, // One initial read per caller plus the flight's double-check.
		allRead:   make(chan struct{}),
	}
	var upstreamCalls atomic.Int32
	releaseUpstream := make(chan struct{})
	a := NewArchivingCache(cache, nil, nil)
	a.Next = test.HandlerFunc(func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		upstreamCalls.Add(1)
		<-releaseUpstream
		msg := new(dns.Msg).SetReply(r)
		msg.Answer = []dns.RR{test.A("example.org. 120 IN A 192.0.2.1")}
		return 0, w.WriteMsg(msg)
	})

	start := make(chan struct{})
	errs := make(chan error, concurrency)
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(concurrency)
	done.Add(concurrency)
	for range concurrency {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			recorder := dnstest.NewRecorder(new(test.ResponseWriter))
			request := new(dns.Msg).SetQuestion("example.org.", dns.TypeA)
			_, err := a.ServeDNS(context.Background(), recorder, request)
			errs <- err
		}()
	}
	ready.Wait()
	close(start)
	select {
	case <-cache.allRead:
	case <-time.After(2 * time.Second):
		close(releaseUpstream)
		done.Wait()
		t.Fatal("concurrent callers did not reach the cache lookup")
	}
	close(releaseUpstream)
	done.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("ServeDNS() error = %v", err)
		}
	}
	if got := upstreamCalls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
	if baseCache.sets != 1 {
		t.Fatalf("cache writes = %d, want 1", baseCache.sets)
	}
}

type countingCache struct {
	*testCache
	reads   atomic.Int32
	waitFor int32
	allRead chan struct{}
	once    sync.Once
}

func (c *countingCache) Get(ctx context.Context, key string) ([]byte, error) {
	if c.reads.Add(1) == c.waitFor {
		c.once.Do(func() { close(c.allRead) })
	}
	return c.testCache.Get(ctx, key)
}

func positiveHandler(answer string) test.HandlerFunc {
	return func(_ context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		msg := new(dns.Msg).SetReply(r)
		msg.Answer = []dns.RR{test.A(answer)}
		return 0, w.WriteMsg(msg)
	}
}

func query(t *testing.T, a *ArchivingCache, ctx context.Context, name string, qtype uint16) *dns.Msg {
	t.Helper()
	recorder := dnstest.NewRecorder(new(test.ResponseWriter))
	request := new(dns.Msg).SetQuestion(name, qtype)
	if _, err := a.ServeDNS(ctx, recorder, request); err != nil {
		t.Fatalf("ServeDNS() error = %v", err)
	}
	if recorder.Msg == nil {
		t.Fatal("ServeDNS() returned no message")
	}
	return recorder.Msg
}

func assertAnswerTTL(t *testing.T, msg *dns.Msg, want uint32) {
	t.Helper()
	if len(msg.Answer) != 1 {
		t.Fatalf("answer count = %d, want 1", len(msg.Answer))
	}
	if got := msg.Answer[0].Header().Ttl; got != want {
		t.Fatalf("answer TTL = %d, want %d", got, want)
	}
}
