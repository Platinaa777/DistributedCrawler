package models

import (
	"regexp"
	"strings"
)

type ScopeRules struct {
	MaxDepth           uint64
	AllowedDomains     []string
	DenyUrlPatterns    []string
	AllowedURLPatterns []string
}

// CompileAllowedURLPattern compiles a wildcard pattern (supports "*") into a regexp.
// Empty patterns return nil, nil and should be ignored by callers.
func CompileAllowedURLPattern(pattern string) (*regexp.Regexp, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, nil
	}

	regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
	return regexp.Compile(regexPattern)
}

// CompileAllowedURLPatterns compiles all non-empty wildcard URL patterns.
func CompileAllowedURLPatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := CompileAllowedURLPattern(pattern)
		if err != nil {
			return nil, err
		}
		if re != nil {
			compiled = append(compiled, re)
		}
	}
	return compiled, nil
}
