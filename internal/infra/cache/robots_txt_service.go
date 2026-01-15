package cache

import (
	"context"
	"distributed-crawler/internal/domain/crawl/services"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/temoto/robotstxt"
	"go.uber.org/zap"
)

// CachedRobotsTxtService implements RobotsTxtService with Redis caching
type CachedRobotsTxtService struct {
	client     *redis.Client
	httpClient *http.Client
	ttl        time.Duration
	logger     *zap.Logger
}

// NewCachedRobotsTxtService creates a new cached robots.txt service
func NewCachedRobotsTxtService(client *redis.Client, ttl time.Duration, logger *zap.Logger) services.RobotsTxtService {
	if ttl == 0 {
		ttl = 24 * time.Hour // Default: cache robots.txt for 24 hours
	}

	return &CachedRobotsTxtService{
		client: client,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ttl:    ttl,
		logger: logger,
	}
}

// IsAllowed checks if the given URL is allowed to be crawled according to robots.txt rules
func (s *CachedRobotsTxtService) IsAllowed(ctx context.Context, urlStr string, userAgent string) (bool, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Build robots.txt URL
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)
	cacheKey := fmt.Sprintf("robots:%s", parsedURL.Host)

	// Try to get from cache
	robotsData, err := s.client.Get(ctx, cacheKey).Bytes()
	if err != nil && err != redis.Nil {
		s.logger.Warn("Failed to get robots.txt from cache",
			zap.String("host", parsedURL.Host),
			zap.Error(err),
		)
	}

	if err == redis.Nil || len(robotsData) == 0 {
		// Fetch robots.txt
		robotsData, err = s.fetchRobotsTxt(ctx, robotsURL)
		if err != nil {
			s.logger.Debug("Failed to fetch robots.txt, allowing URL",
				zap.String("robots_url", robotsURL),
				zap.Error(err),
			)
			// If we can't fetch robots.txt, allow the URL (permissive default)
			return true, nil
		}

		// Cache the result
		if setErr := s.client.Set(ctx, cacheKey, robotsData, s.ttl).Err(); setErr != nil {
			s.logger.Warn("Failed to cache robots.txt",
				zap.String("host", parsedURL.Host),
				zap.Error(setErr),
			)
		}
	}

	// Parse robots.txt
	robots, err := robotstxt.FromBytes(robotsData)
	if err != nil {
		s.logger.Debug("Failed to parse robots.txt, allowing URL",
			zap.String("host", parsedURL.Host),
			zap.Error(err),
		)
		// If we can't parse robots.txt, allow the URL
		return true, nil
	}

	// Check if the path is allowed
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}
	if parsedURL.RawQuery != "" {
		path = path + "?" + parsedURL.RawQuery
	}

	group := robots.FindGroup(userAgent)
	if group == nil {
		// No matching group, allow by default
		return true, nil
	}

	allowed := group.Test(path)
	if !allowed {
		s.logger.Debug("URL disallowed by robots.txt",
			zap.String("url", urlStr),
			zap.String("user_agent", userAgent),
		)
	}

	return allowed, nil
}

// fetchRobotsTxt fetches the robots.txt file from the given URL
func (s *CachedRobotsTxtService) fetchRobotsTxt(ctx context.Context, robotsURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "DistributedCrawler/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch robots.txt: %w", err)
	}
	defer resp.Body.Close()

	// If robots.txt doesn't exist (404) or other error, return empty robots.txt
	if resp.StatusCode != http.StatusOK {
		// Return empty robots.txt (allows everything)
		return []byte{}, nil
	}

	// Limit reading to 512KB to prevent abuse
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read robots.txt body: %w", err)
	}

	return body, nil
}
