package robots

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/NationalLibraryOfNorway/veidemann/api/config"
	"github.com/NationalLibraryOfNorway/veidemann/robots-evaluator/cache"
	"github.com/jimsmart/grobotstxt"
	whatwgurl "github.com/nlnwa/whatwg-url/url"
)

type Evaluator struct {
	Cache  cache.Cachier
	Client *http.Client
}

type AllowedRequest struct {
	RobotsPolicy       config.PolitenessConfig_RobotsPolicy
	MinValiditySeconds int32
	Uri                string
	CustomRobots       string
	UserAgent          string
}

// IsAllowed implements the RobotsEvaluatorServer interface
func (e *Evaluator) IsAllowed(ctx context.Context, req *AllowedRequest) (bool, error) {
	switch req.RobotsPolicy {

	case config.PolitenessConfig_IGNORE_ROBOTS:
		return true, nil

	case config.PolitenessConfig_OBEY_ROBOTS,
		config.PolitenessConfig_OBEY_ROBOTS_CLASSIC,
		config.PolitenessConfig_CUSTOM_IF_MISSING,
		config.PolitenessConfig_CUSTOM_IF_MISSING_CLASSIC:

		customIfMissing := req.RobotsPolicy == config.PolitenessConfig_CUSTOM_IF_MISSING ||
			req.RobotsPolicy == config.PolitenessConfig_CUSTOM_IF_MISSING_CLASSIC

		robotsTxt, err := e.fetchRobotsTxt(ctx, req.Uri)
		if err == nil {
			return grobotstxt.AgentAllowed(robotsTxt, req.UserAgent, req.Uri), nil
		}
		if err != nil && !customIfMissing {
			return false, fmt.Errorf("failed to fetch robots.txt: %w", err)
		}
		fallthrough

	case config.PolitenessConfig_CUSTOM_ROBOTS,
		config.PolitenessConfig_CUSTOM_ROBOTS_CLASSIC:

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
	robotsTxt, err := e.fetchRobotsTxt(ctx, req.Uri)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch robots.txt: %w", err)
	}
	return grobotstxt.Sitemaps(robotsTxt), nil
}

func (e *Evaluator) fetchRobotsTxt(ctx context.Context, uri string) (string, error) {
	base, err := whatwgurl.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}
	robotsUrl, err := base.Parse("/robots.txt")
	if err != nil {
		return "", fmt.Errorf("failed to parse robots.txt URI: %w", err)
	}
	robotsUri := robotsUrl.String()

	cached, err := e.Cache.Get(ctx, robotsUri)
	if err == nil {
		slog.Debug("Cache hit for robots.txt", "robotsUri", robotsUri)
		return string(cached), nil
	}
	if !errors.Is(err, cache.ErrKeyNotFound) {
		slog.Warn("Failed to get robots.txt from cache", "robotsUri", robotsUri, "error", err)
	} else {
		slog.Debug("Cache miss for robots.txt", "robotsUri", robotsUri)
	}

	slog.Debug("Fetching robots.txt", "robotsUri", robotsUri)

	robotsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsUri, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := e.Client.Do(robotsReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("robots.txt returned status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	robotsTxt := string(b)

	if err := e.Cache.Put(ctx, robotsUri, b); err != nil {
		slog.Warn("Failed to put robots.txt in cache", "robotsUri", robotsUri, "error", err)
	}

	return robotsTxt, nil
}
