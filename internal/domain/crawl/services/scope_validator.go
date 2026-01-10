package services

import "distributed-crawler/internal/domain/crawl/models"

// ScopeValidator validates URLs against crawl scope rules
type ScopeValidator interface {
	Validate(url string, depth uint64, rules models.ScopeRules) error
}
