package sanitizer

import (
	"distributed-crawler/internal/application/service/preview"

	"github.com/microcosm-cc/bluemonday"
)

var _ preview.HTMLSanitizer = (*HTMLSanitizerImpl)(nil)

// HTMLSanitizerImpl sanitizes HTML for safe iframe rendering
type HTMLSanitizerImpl struct {
	policy *bluemonday.Policy
}

// NewHTMLSanitizer creates a new HTML sanitizer
func NewHTMLSanitizer() *HTMLSanitizerImpl {
	// Create strict policy that removes all dangerous elements
	policy := bluemonday.NewPolicy()

	// Allow basic structural elements
	policy.AllowElements("html", "head", "body", "title", "meta", "link")
	policy.AllowElements("div", "span", "p", "br", "hr")
	policy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")
	policy.AllowElements("ul", "ol", "li", "dl", "dt", "dd")
	policy.AllowElements("table", "thead", "tbody", "tfoot", "tr", "th", "td")
	policy.AllowElements("section", "article", "aside", "header", "footer", "nav", "main")

	// Allow text formatting
	policy.AllowElements("strong", "b", "em", "i", "u", "s", "strike", "small", "mark")
	policy.AllowElements("code", "pre", "kbd", "samp", "var")
	policy.AllowElements("blockquote", "q", "cite", "abbr", "address")

	// Allow images (but src will be sanitized)
	policy.AllowElements("img")
	policy.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")

	// Allow links (but href will be sanitized)
	policy.AllowElements("a")
	policy.AllowAttrs("href", "title").OnElements("a")

	// Allow form elements (but without action/onsubmit)
	policy.AllowElements("form", "input", "textarea", "button", "select", "option", "label", "fieldset", "legend")
	policy.AllowAttrs("type", "name", "value", "placeholder", "disabled", "readonly").OnElements("input")
	policy.AllowAttrs("name", "disabled", "readonly").OnElements("textarea")

	// Allow common attributes
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("style").Globally() // CSS will be sanitized

	// Explicitly deny dangerous elements
	policy.SkipElementsContent("script", "style", "iframe", "object", "embed", "applet", "frame", "frameset")

	// Remove all event handlers (onclick, onload, etc.)
	policy.AllowNoAttrs().OnElements("script")

	return &HTMLSanitizerImpl{
		policy: policy,
	}
}

// Sanitize removes dangerous HTML elements and attributes
func (s *HTMLSanitizerImpl) Sanitize(html []byte) []byte {
	return s.policy.SanitizeBytes(html)
}
