package models

type ScopeRules struct {
	MaxDepth        uint64
	AllowedDomains  []string
	DenyUrlPatterns []string
}
