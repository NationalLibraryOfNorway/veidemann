package robots

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/cache"
	"github.com/jimsmart/grobotstxt"
	whatwgurl "github.com/nlnwa/whatwg-url/url"
)

const (
	executionID    = "veidemann_eid"
	jobExecutionID = "veidemann_jeid"
	collectionID   = "veidemann_cid"
)

type Evaluator struct {
	Cache  cache.Cachier
	Client *http.Client
}

type AllowedRequest struct {
	RobotsPolicy       configV1.PolitenessConfig_RobotsPolicy
	MinValiditySeconds int32
	Uri                string
	CustomRobots       string
	UserAgent          string
	CollectionId       string
	ExecutionId        string
	JobExecutionId     string
}

// IsAllowed implements the RobotsEvaluatorServer interface
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

		robotsTxt, err := e.fetchRobotsTxt(ctx, req)
		if err == nil {
			return grobotstxt.AgentAllowed(robotsTxt, req.UserAgent, req.Uri), nil
		}
		if !customIfMissing {
			return false, fmt.Errorf("failed to fetch robots.txt: %w", err)
		}
		fallthrough

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
	robotsTxt, err := e.fetchSitemap(ctx, req.Uri)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch robots.txt: %w", err)
	}
	return grobotstxt.Sitemaps(robotsTxt), nil
}

func (e *Evaluator) fetchSitemap(ctx context.Context, uri string) (string, error) {
	return e.fetchRobotsTxt(ctx, &AllowedRequest{
		Uri: uri,
	})
}

func (e *Evaluator) fetchRobotsTxt(ctx context.Context, req *AllowedRequest) (string, error) {
	base, err := whatwgurl.Parse(req.Uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}
	robotsUrl, err := base.Parse("/robots.txt")
	if err != nil {
		return "", fmt.Errorf("failed to parse robots.txt: %w", err)
	}
	robotsUri := robotsUrl.String()

	cacheKey := fmt.Sprintf("robots|%s|%s|%d|%s",
		robotsUrl.Scheme(),
		robotsUrl.Host(),
		robotsUrl.DecodedPort(),
		req.JobExecutionId,
	)

	cached, err := e.Cache.Get(ctx, cacheKey)
	if err == nil {
		slog.Debug("Cache hit for robots.txt", "key", cacheKey)
		return string(cached), nil
	}
	if !errors.Is(err, cache.ErrKeyNotFound) {
		slog.Debug("Failed to find robots.txt in cache", "key", cacheKey, "error", err)
	} else {
		slog.Debug("Cache miss for robots.txt", "key", cacheKey)
	}

	slog.Debug("Fetching robots.txt", "url", robotsUri)
	robotsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsUri, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if req.CollectionId != "" {
		robotsReq.Header.Set(collectionID, req.CollectionId)
	}
	if req.ExecutionId != "" {
		robotsReq.Header.Set(executionID, req.ExecutionId)
	}
	if req.JobExecutionId != "" {
		robotsReq.Header.Set(jobExecutionID, req.JobExecutionId)
	}

	resp, err := e.Client.Do(robotsReq)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("robots.txt returned status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	robotsTxt := string(b)

	if err := e.Cache.Put(ctx, cacheKey, b); err != nil {
		slog.Warn("Failed to cache robots.txt", "key", cacheKey, "error", err)
	}

	return robotsTxt, nil
}
