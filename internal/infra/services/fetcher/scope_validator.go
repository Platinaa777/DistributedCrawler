package fetcher

import (
	"distributed-crawler/internal/domain/crawl/models"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// DomainScopeValidator implements the ScopeValidator interface
type DomainScopeValidator struct{}

// NewDomainScopeValidator creates a new domain scope validator
func NewDomainScopeValidator() *DomainScopeValidator {
	return &DomainScopeValidator{}
}

// Validate checks if a URL is within the allowed scope
func (v *DomainScopeValidator) Validate(urlStr string, depth uint64, rules models.ScopeRules) error {
	// Check depth
	if depth > rules.MaxDepth {
		return fmt.Errorf("depth %d exceeds max depth %d", depth, rules.MaxDepth)
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check allowed domains (only if whitelist is specified)
	if len(rules.AllowedDomains) > 0 {
		allowed := false
		host := parsedURL.Host

		for _, domain := range rules.AllowedDomains {
			// Exact match or subdomain match
			if host == domain || strings.HasSuffix(host, "."+domain) {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("domain %s not in allowed domains", host)
		}
	}

	// Check denied URL patterns
	for _, pattern := range rules.DenyUrlPatterns {
		matched, err := regexp.MatchString(pattern, urlStr)
		if err != nil {
			return fmt.Errorf("invalid deny pattern %s: %w", pattern, err)
		}
		if matched {
			return fmt.Errorf("URL matches denied pattern: %s", pattern)
		}
	}

	return nil
}
