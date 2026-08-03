package robots

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/cache"
)

const (
	blockedRules = "User-agent: *\nDisallow: /blocked\n"
	allowedRules = "User-agent: *\nAllow: /\n"
)

type memoryCache struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMemoryCache() *memoryCache {
	return &memoryCache{data: make(map[string][]byte)}
}

func (c *memoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, ok := c.data[key]
	if !ok {
		return nil, cache.ErrKeyNotFound
	}
	return bytes.Clone(b), nil
}

func (c *memoryCache) Put(_ context.Context, key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = bytes.Clone(value)
	return nil
}

func (c *memoryCache) Close(context.Context) error { return nil }

func (c *memoryCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.data)
}

type originRequest struct {
	cacheControl string
	collection   string
	execution    string
	jobExecution string
}

type testOrigin struct {
	mu       sync.Mutex
	status   int
	body     string
	requests []originRequest
}

func (o *testOrigin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.mu.Lock()
	o.requests = append(o.requests, originRequest{
		cacheControl: r.Header.Get("Cache-Control"),
		collection:   r.Header.Get(collectionID),
		execution:    r.Header.Get(executionID),
		jobExecution: r.Header.Get(jobExecutionID),
	})
	status := o.status
	body := o.body
	o.mu.Unlock()

	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func (o *testOrigin) setResponse(status int, body string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.status = status
	o.body = body
}

func (o *testOrigin) requestSnapshot() []originRequest {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]originRequest(nil), o.requests...)
}

func newTestEvaluator(c cache.Cachier, client *http.Client, now *time.Time) *Evaluator {
	e := NewEvaluator(c, client, 24*time.Hour, time.Hour)
	e.now = func() time.Time { return *now }
	return e
}

func allowedRequest(uri, collection, execution, job string) *AllowedRequest {
	return &AllowedRequest{
		RobotsPolicy:   configV1.PolitenessConfig_OBEY_ROBOTS,
		Uri:            uri,
		UserAgent:      "testcrawler",
		CollectionId:   collection,
		ExecutionId:    execution,
		JobExecutionId: job,
	}
}

func TestCollectionScopedCacheAndRequestHeaders(t *testing.T) {
	origin := &testOrigin{status: http.StatusOK, body: blockedRules}
	server := httptest.NewServer(origin)
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	c := newMemoryCache()
	e := newTestEvaluator(c, server.Client(), &now)
	uri := server.URL + "/blocked"

	for _, req := range []*AllowedRequest{
		allowedRequest(uri, "collection-a", "execution-a1", "job-a1"),
		allowedRequest(uri, "collection-a", "execution-a2", "job-a2"),
		allowedRequest(uri, "collection-b", "execution-b1", "job-b1"),
	} {
		allowed, err := e.IsAllowed(context.Background(), req)
		if err != nil {
			t.Fatalf("IsAllowed returned error: %v", err)
		}
		if allowed {
			t.Fatalf("IsAllowed(%q) = true, want false", req.CollectionId)
		}
	}

	requests := origin.requestSnapshot()
	if len(requests) != 2 {
		t.Fatalf("origin requests = %d, want 2", len(requests))
	}
	if c.len() != 2 {
		t.Fatalf("cache entries = %d, want 2", c.len())
	}

	wantRequests := []originRequest{
		{cacheControl: "no-cache, no-store", collection: "collection-a", execution: "execution-a1", jobExecution: "job-a1"},
		{cacheControl: "no-cache, no-store", collection: "collection-b", execution: "execution-b1", jobExecution: "job-b1"},
	}
	for i, want := range wantRequests {
		if requests[i] != want {
			t.Errorf("origin request %d = %#v, want %#v", i, requests[i], want)
		}
	}
}

func TestLogicalFreshnessTriggersRefresh(t *testing.T) {
	origin := &testOrigin{status: http.StatusOK, body: allowedRules}
	server := httptest.NewServer(origin)
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	e := newTestEvaluator(newMemoryCache(), server.Client(), &now)
	req := allowedRequest(server.URL+"/page", "collection-a", "execution-1", "job-1")

	for _, advance := range []time.Duration{0, 23 * time.Hour, time.Hour} {
		now = now.Add(advance)
		if _, err := e.IsAllowed(context.Background(), req); err != nil {
			t.Fatalf("IsAllowed returned error: %v", err)
		}
	}

	if got := len(origin.requestSnapshot()); got != 2 {
		t.Fatalf("origin requests = %d, want 2", got)
	}
}

func TestUnreachableUsesStaleRulesAndSuppressesRetry(t *testing.T) {
	origin := &testOrigin{status: http.StatusOK, body: blockedRules}
	server := httptest.NewServer(origin)
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	e := newTestEvaluator(newMemoryCache(), server.Client(), &now)
	req := allowedRequest(server.URL+"/blocked", "collection-a", "execution-1", "job-1")

	allowed, err := e.IsAllowed(context.Background(), req)
	if err != nil || allowed {
		t.Fatalf("initial IsAllowed = (%v, %v), want (false, nil)", allowed, err)
	}

	now = now.Add(25 * time.Hour)
	origin.setResponse(http.StatusServiceUnavailable, "")
	allowed, err = e.IsAllowed(context.Background(), req)
	if err != nil || allowed {
		t.Fatalf("stale IsAllowed = (%v, %v), want (false, nil)", allowed, err)
	}

	now = now.Add(30 * time.Minute)
	allowed, err = e.IsAllowed(context.Background(), req)
	if err != nil || allowed {
		t.Fatalf("suppressed IsAllowed = (%v, %v), want (false, nil)", allowed, err)
	}
	if got := len(origin.requestSnapshot()); got != 2 {
		t.Fatalf("origin requests during retry backoff = %d, want 2", got)
	}

	now = now.Add(30 * time.Minute)
	origin.setResponse(http.StatusOK, allowedRules)
	allowed, err = e.IsAllowed(context.Background(), req)
	if err != nil || !allowed {
		t.Fatalf("refreshed IsAllowed = (%v, %v), want (true, nil)", allowed, err)
	}
	if got := len(origin.requestSnapshot()); got != 3 {
		t.Fatalf("origin requests after retry = %d, want 3", got)
	}
}

func TestCollectionCannotUseAnotherCollectionsStaleRules(t *testing.T) {
	origin := &testOrigin{status: http.StatusOK, body: allowedRules}
	server := httptest.NewServer(origin)
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	e := newTestEvaluator(newMemoryCache(), server.Client(), &now)
	uri := server.URL + "/blocked"

	allowed, err := e.IsAllowed(context.Background(), allowedRequest(uri, "collection-a", "execution-a", "job-a"))
	if err != nil || !allowed {
		t.Fatalf("initial collection-a IsAllowed = (%v, %v), want (true, nil)", allowed, err)
	}

	now = now.Add(25 * time.Hour)
	origin.setResponse(http.StatusServiceUnavailable, "")
	allowed, err = e.IsAllowed(context.Background(), allowedRequest(uri, "collection-b", "execution-b", "job-b"))
	if err != nil || allowed {
		t.Fatalf("collection-b IsAllowed = (%v, %v), want (false, nil)", allowed, err)
	}

	allowed, err = e.IsAllowed(context.Background(), allowedRequest(uri, "collection-a", "execution-a2", "job-a2"))
	if err != nil || !allowed {
		t.Fatalf("stale collection-a IsAllowed = (%v, %v), want (true, nil)", allowed, err)
	}
}

func TestConcurrentMissesUseSingleOriginFetch(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	var once sync.Once
	var requests int
	var requestsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestsMu.Lock()
		requests++
		requestsMu.Unlock()
		once.Do(func() { close(requestStarted) })
		<-releaseRequest
		_, _ = io.WriteString(w, allowedRules)
	}))
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	e := newTestEvaluator(newMemoryCache(), server.Client(), &now)
	req := allowedRequest(server.URL+"/page", "collection-a", "execution-1", "job-1")

	const callers = 10
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.IsAllowed(context.Background(), req)
			errs <- err
		}()
	}

	<-requestStarted
	close(releaseRequest)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("IsAllowed returned error: %v", err)
		}
	}

	requestsMu.Lock()
	defer requestsMu.Unlock()
	if requests != 1 {
		t.Fatalf("origin requests = %d, want 1", requests)
	}
}

func TestUnreachableWithoutCachedRulesDisallowsAndCustomFallsBack(t *testing.T) {
	origin := &testOrigin{status: http.StatusServiceUnavailable}
	server := httptest.NewServer(origin)
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	e := newTestEvaluator(newMemoryCache(), server.Client(), &now)
	uri := server.URL + "/blocked"

	req := allowedRequest(uri, "collection-a", "execution-1", "job-1")
	allowed, err := e.IsAllowed(context.Background(), req)
	if err != nil || allowed {
		t.Fatalf("OBEY_ROBOTS IsAllowed = (%v, %v), want (false, nil)", allowed, err)
	}

	customReq := allowedRequest(uri, "collection-a", "execution-2", "job-2")
	customReq.RobotsPolicy = configV1.PolitenessConfig_CUSTOM_IF_MISSING
	customReq.CustomRobots = blockedRules
	allowed, err = e.IsAllowed(context.Background(), customReq)
	if err != nil || allowed {
		t.Fatalf("CUSTOM_IF_MISSING IsAllowed = (%v, %v), want (false, nil)", allowed, err)
	}
	if got := len(origin.requestSnapshot()); got != 1 {
		t.Fatalf("origin requests = %d, want 1 during retry backoff", got)
	}
}

func TestUnavailableResultUsesPolicyFallback(t *testing.T) {
	origin := &testOrigin{status: http.StatusNotFound}
	server := httptest.NewServer(origin)
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	e := newTestEvaluator(newMemoryCache(), server.Client(), &now)
	uri := server.URL + "/blocked"

	allowed, err := e.IsAllowed(context.Background(), allowedRequest(uri, "collection-a", "execution-1", "job-1"))
	if err != nil || !allowed {
		t.Fatalf("OBEY_ROBOTS IsAllowed = (%v, %v), want (true, nil)", allowed, err)
	}

	customReq := allowedRequest(uri, "collection-a", "execution-2", "job-2")
	customReq.RobotsPolicy = configV1.PolitenessConfig_CUSTOM_IF_MISSING
	customReq.CustomRobots = blockedRules
	allowed, err = e.IsAllowed(context.Background(), customReq)
	if err != nil || allowed {
		t.Fatalf("CUSTOM_IF_MISSING IsAllowed = (%v, %v), want (false, nil)", allowed, err)
	}
	if got := len(origin.requestSnapshot()); got != 1 {
		t.Fatalf("origin requests = %d, want 1", got)
	}
}

func TestIncompatibleCacheEntryIsReplaced(t *testing.T) {
	origin := &testOrigin{status: http.StatusOK, body: allowedRules}
	server := httptest.NewServer(origin)
	defer server.Close()

	now := time.Date(2026, time.August, 3, 12, 0, 0, 0, time.UTC)
	c := newMemoryCache()
	e := newTestEvaluator(c, server.Client(), &now)
	req := allowedRequest(server.URL+"/page", "collection-a", "execution-1", "job-1")
	_, key, err := robotsLocation(req)
	if err != nil {
		t.Fatal(err)
	}
	c.data[key] = []byte("legacy raw robots body")

	if _, err := e.IsAllowed(context.Background(), req); err != nil {
		t.Fatalf("IsAllowed returned error: %v", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(c.data[key], &entry); err != nil {
		t.Fatalf("replacement cache entry is not JSON: %v", err)
	}
	if entry.Version != cacheEntryVersion || !entry.HasRules {
		t.Fatalf("replacement cache entry = %#v", entry)
	}
}

func TestMissingCollectionUsesUnscopedPartitionAndWarns(t *testing.T) {
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(previousLogger)

	req := allowedRequest("https://example.com/page", "", "execution-1", "job-1")
	_, key, err := robotsLocation(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "robots|unscoped|") {
		t.Fatalf("cache key = %q, want unscoped partition", key)
	}
	if !strings.Contains(logs.String(), "using unscoped cache partition") {
		t.Fatalf("warning not logged: %s", logs.String())
	}
}

func TestCacheKeySeparatesHostnameAndPort(t *testing.T) {
	req := allowedRequest("https://Example.COM:8443/page", "collection-a", "execution-1", "job-1")
	_, key, err := robotsLocation(req)
	if err != nil {
		t.Fatal(err)
	}
	if want := "robots|collection-a|https|example.com|8443"; key != want {
		t.Fatalf("cache key = %q, want %q", key, want)
	}
}
