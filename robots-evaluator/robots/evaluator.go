package robots

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/cache"
	"github.com/jimsmart/grobotstxt"
	whatwgurl "github.com/nlnwa/whatwg-url/url"
	"golang.org/x/sync/singleflight"
)

const (
	executionID       = "veidemann_eid"
	jobExecutionID    = "veidemann_jeid"
	collectionID      = "veidemann_cid"
	unscopedCacheKey  = "unscoped"
	cacheEntryVersion = 1
)

var errRobotsUnreachable = errors.New("robots.txt is unreachable")

type Evaluator struct {
	cache                    cache.Cachier
	client                   *http.Client
	cacheFreshness           time.Duration
	unreachableRetryInterval time.Duration
	now                      func() time.Time
	refreshGroup             singleflight.Group
}

type AllowedRequest struct {
	RobotsPolicy   configV1.PolitenessConfig_RobotsPolicy
	Uri            string
	CustomRobots   string
	UserAgent      string
	CollectionId   string
	ExecutionId    string
	JobExecutionId string
}

type cacheEntry struct {
	Version    int       `json:"version"`
	Rules      string    `json:"rules,omitempty"`
	HasRules   bool      `json:"hasRules"`
	FetchedAt  time.Time `json:"fetchedAt,omitempty"`
	FreshUntil time.Time `json:"freshUntil,omitempty"`
	RetryAfter time.Time `json:"retryAfter,omitempty"`
}

type robotsResult struct {
	rules    string
	hasRules bool
}

func NewEvaluator(
	cache cache.Cachier,
	client *http.Client,
	cacheFreshness time.Duration,
	unreachableRetryInterval time.Duration,
) *Evaluator {
	return &Evaluator{
		cache:                    cache,
		client:                   client,
		cacheFreshness:           cacheFreshness,
		unreachableRetryInterval: unreachableRetryInterval,
		now:                      time.Now,
	}
}

// IsAllowed evaluates robots.txt for a URI.
func (e *Evaluator) IsAllowed(ctx context.Context, req *AllowedRequest) (bool, error) {
	switch req.RobotsPolicy {

	case configV1.PolitenessConfig_IGNORE_ROBOTS:
		return true, nil

	case configV1.PolitenessConfig_OBEY_ROBOTS,
		configV1.PolitenessConfig_OBEY_ROBOTS_CLASSIC,
		configV1.PolitenessConfig_CUSTOM_IF_MISSING,
		configV1.PolitenessConfig_CUSTOM_IF_MISSING_CLASSIC:

		customIfMissing := req.RobotsPolicy == configV1.PolitenessConfig_CUSTOM_IF_MISSING ||
			req.RobotsPolicy == configV1.PolitenessConfig_CUSTOM_IF_MISSING_CLASSIC

		result, err := e.fetchRobotsTxt(ctx, req)
		if err == nil {
			if result.hasRules {
				return grobotstxt.AgentAllowed(result.rules, req.UserAgent, req.Uri), nil
			}
			if !customIfMissing {
				return true, nil
			}
			return grobotstxt.AgentAllowed(req.CustomRobots, req.UserAgent, req.Uri), nil
		}

		if customIfMissing {
			return grobotstxt.AgentAllowed(req.CustomRobots, req.UserAgent, req.Uri), nil
		}
		if errors.Is(err, errRobotsUnreachable) {
			slog.Warn("Robots.txt is unreachable and no cached rules are available; disallowing request",
				"uri", req.Uri,
				"collectionId", req.CollectionId,
				"error", err,
			)
			return false, nil
		}
		return false, fmt.Errorf("failed to fetch robots.txt: %w", err)

	case configV1.PolitenessConfig_CUSTOM_ROBOTS,
		configV1.PolitenessConfig_CUSTOM_ROBOTS_CLASSIC:

		return grobotstxt.AgentAllowed(req.CustomRobots, req.UserAgent, req.Uri), nil

	default:
		slog.Warn("invalid robots policy", "policy", req.RobotsPolicy)
		return true, nil
	}
}

type SitemapRequest struct {
	Uri string
}

func (e *Evaluator) Sitemap(ctx context.Context, req *SitemapRequest) ([]string, error) {
	result, err := e.fetchRobotsTxt(ctx, &AllowedRequest{Uri: req.Uri})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch robots.txt: %w", err)
	}
	if !result.hasRules {
		return nil, nil
	}
	return grobotstxt.Sitemaps(result.rules), nil
}

func (e *Evaluator) fetchRobotsTxt(ctx context.Context, req *AllowedRequest) (robotsResult, error) {
	robotsURI, cacheKey, err := robotsLocation(req)
	if err != nil {
		return robotsResult{}, err
	}

	if result, err, usable := e.cachedResult(ctx, cacheKey, e.now().UTC()); usable {
		return result, err
	}

	value, err, _ := e.refreshGroup.Do(cacheKey, func() (any, error) {
		now := e.now().UTC()
		if result, cachedErr, usable := e.cachedResult(ctx, cacheKey, now); usable {
			return result, cachedErr
		}

		entry, _ := e.loadCacheEntry(ctx, cacheKey)
		return e.refreshRobotsTxt(ctx, cacheKey, robotsURI, req, entry, now)
	})
	if err != nil {
		return robotsResult{}, err
	}
	return value.(robotsResult), nil
}

func robotsLocation(req *AllowedRequest) (string, string, error) {
	base, err := whatwgurl.Parse(req.Uri)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse URI: %w", err)
	}
	robotsURL, err := base.Parse("/robots.txt")
	if err != nil {
		return "", "", fmt.Errorf("failed to parse robots.txt: %w", err)
	}

	collection := req.CollectionId
	if collection == "" {
		collection = unscopedCacheKey
		slog.Warn("Robots request has no collection ID; using unscoped cache partition", "uri", req.Uri)
	}

	cacheKey := fmt.Sprintf("robots|%s|%s|%s|%d",
		collection,
		robotsURL.Scheme(),
		robotsURL.Hostname(),
		robotsURL.DecodedPort(),
	)
	return robotsURL.String(), cacheKey, nil
}

func (e *Evaluator) cachedResult(ctx context.Context, key string, now time.Time) (robotsResult, error, bool) {
	entry, found := e.loadCacheEntry(ctx, key)
	if !found {
		return robotsResult{}, nil, false
	}

	if now.Before(entry.FreshUntil) {
		slog.Debug("Fresh robots cache hit", "key", key, "freshUntil", entry.FreshUntil)
		return robotsResult{rules: entry.Rules, hasRules: entry.HasRules}, nil, true
	}
	if now.Before(entry.RetryAfter) {
		slog.Debug("Robots refresh suppressed after unreachable result", "key", key, "retryAfter", entry.RetryAfter)
		if entry.HasRules {
			return robotsResult{rules: entry.Rules, hasRules: true}, nil, true
		}
		return robotsResult{}, errRobotsUnreachable, true
	}
	return robotsResult{}, nil, false
}

func (e *Evaluator) loadCacheEntry(ctx context.Context, key string) (*cacheEntry, bool) {
	b, err := e.cache.Get(ctx, key)
	if errors.Is(err, cache.ErrKeyNotFound) {
		slog.Debug("Robots cache miss", "key", key)
		return nil, false
	}
	if err != nil {
		slog.Warn("Failed to read robots cache", "key", key, "error", err)
		return nil, false
	}

	entry := new(cacheEntry)
	if err := json.Unmarshal(b, entry); err != nil || entry.Version != cacheEntryVersion {
		slog.Warn("Ignoring incompatible robots cache entry", "key", key, "error", err, "version", entry.Version)
		return nil, false
	}
	return entry, true
}

func (e *Evaluator) refreshRobotsTxt(
	ctx context.Context,
	cacheKey string,
	robotsURI string,
	req *AllowedRequest,
	stale *cacheEntry,
	now time.Time,
) (robotsResult, error) {
	slog.Debug("Fetching robots.txt", "url", robotsURI, "key", cacheKey)
	robotsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURI, nil)
	if err != nil {
		return robotsResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	robotsReq.Header.Set("Cache-Control", "no-cache, no-store")
	if req.CollectionId != "" {
		robotsReq.Header.Set(collectionID, req.CollectionId)
	}
	if req.ExecutionId != "" {
		robotsReq.Header.Set(executionID, req.ExecutionId)
	}
	if req.JobExecutionId != "" {
		robotsReq.Header.Set(jobExecutionID, req.JobExecutionId)
	}

	resp, err := e.client.Do(robotsReq)
	if err != nil {
		return e.handleUnreachable(ctx, cacheKey, stale, now, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return e.handleUnreachable(ctx, cacheKey, stale, now,
				fmt.Errorf("failed to read response body: %w", err))
		}

		entry := &cacheEntry{
			Version:    cacheEntryVersion,
			Rules:      string(b),
			HasRules:   true,
			FetchedAt:  now,
			FreshUntil: now.Add(e.cacheFreshness),
		}
		e.storeCacheEntry(ctx, cacheKey, entry)
		return robotsResult{rules: entry.Rules, hasRules: true}, nil

	case resp.StatusCode >= 300 && resp.StatusCode < 500:
		_, _ = io.Copy(io.Discard, resp.Body)
		entry := &cacheEntry{
			Version:    cacheEntryVersion,
			FetchedAt:  now,
			FreshUntil: now.Add(e.cacheFreshness),
		}
		e.storeCacheEntry(ctx, cacheKey, entry)
		slog.Debug("Robots.txt is unavailable; caching missing result",
			"url", robotsURI,
			"statusCode", resp.StatusCode,
			"freshUntil", entry.FreshUntil,
		)
		return robotsResult{}, nil

	default:
		_, _ = io.Copy(io.Discard, resp.Body)
		return e.handleUnreachable(ctx, cacheKey, stale, now,
			fmt.Errorf("robots.txt returned status code: %d", resp.StatusCode))
	}
}

func (e *Evaluator) handleUnreachable(
	ctx context.Context,
	cacheKey string,
	stale *cacheEntry,
	now time.Time,
	cause error,
) (robotsResult, error) {
	entry := stale
	if entry == nil || !entry.HasRules {
		entry = &cacheEntry{Version: cacheEntryVersion}
	}
	entry.FreshUntil = time.Time{}
	entry.RetryAfter = now.Add(e.unreachableRetryInterval)
	e.storeCacheEntry(ctx, cacheKey, entry)

	if entry.HasRules {
		slog.Warn("Robots.txt is unreachable; using stale rules",
			"key", cacheKey,
			"retryAfter", entry.RetryAfter,
			"error", cause,
		)
		return robotsResult{rules: entry.Rules, hasRules: true}, nil
	}
	return robotsResult{}, fmt.Errorf("%w: %v", errRobotsUnreachable, cause)
}

func (e *Evaluator) storeCacheEntry(ctx context.Context, key string, entry *cacheEntry) {
	b, err := json.Marshal(entry)
	if err != nil {
		slog.Warn("Failed to encode robots cache entry", "key", key, "error", err)
		return
	}
	if err := e.cache.Put(ctx, key, b); err != nil {
		slog.Warn("Failed to store robots cache entry", "key", key, "error", err)
	}
}
