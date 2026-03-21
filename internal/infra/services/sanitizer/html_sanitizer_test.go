package sanitizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTMLSanitizer_RemovesDangerousContentAndKeepsSafeMarkup(t *testing.T) {
	t.Parallel()

	s := NewHTMLSanitizer()
	require.NotNil(t, s)

	input := []byte(`<html><body><script>alert(1)</script><a href="https://example.com" onclick="x()">link</a><div class="x">safe</div></body></html>`)
	out := string(s.Sanitize(input))

	assert.NotContains(t, out, "<script>")
	assert.NotContains(t, out, "onclick=")
	assert.Contains(t, out, `href="https://example.com"`)
	assert.Contains(t, out, `<div class="x">safe</div>`)
}

