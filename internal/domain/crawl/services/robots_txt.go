package services

import "context"

// RobotsTxtService checks URLs against robots.txt rules
type RobotsTxtService interface {
	// IsAllowed checks if the given URL is allowed to be crawled
	// according to robots.txt rules for the specified user-agent.
	// Returns true if allowed, false if disallowed.
	// If robots.txt cannot be fetched, it defaults to allowing the URL.
	IsAllowed(ctx context.Context, urlStr string, userAgent string) (bool, error)
}
