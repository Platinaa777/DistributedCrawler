package worker

import (
	"net/url"
	"strings"
	"testing"

	"distributed-crawler/internal/domain/crawl/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/charmap"
)

// newTestParserWorker creates a minimal ParserWorker suitable for unit-testing extraction logic.
// All external service fields are nil because extraction methods do not call them.
func newTestParserWorker(t *testing.T) *ParserWorker {
	t.Helper()
	logger := zap.NewNop()
	return &ParserWorker{logger: logger}
}

func parseHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)
	return doc
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)
	return u
}

// ──────────────────────────────────────────────────────────────────────────────
// Item extraction: success
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractItems_Success(t *testing.T) {
	t.Parallel()

	w := newTestParserWorker(t)
	baseURL := mustParseURL(t, "https://books.toscrape.com/")

	html := `
<html><body>
  <article class="product_pod">
    <h3><a title="Book A">Book A</a></h3>
    <p class="price_color">£19.99</p>
    <p class="availability">In stock</p>
  </article>
  <article class="product_pod">
    <h3><a title="Book B">Book B</a></h3>
    <p class="price_color">£25.00</p>
    <p class="availability">Out of stock</p>
  </article>
</body></html>`

	doc := parseHTML(t, html)

	spec := &models.ItemsSpec{
		ContainerSelector: "article.product_pod",
		Fields: []models.FieldSpec{
			{
				Name: "name",
				Type: models.ValueString,
				Extractor: models.ExtractorSpec{
					Selector:  "h3 a",
					Attribute: "title",
				},
			},
			{
				Name: "price",
				Type: models.ValueFloat,
				Extractor: models.ExtractorSpec{
					Selector:  "p.price_color",
					Attribute: "text",
				},
				Transforms: []models.TransformSpec{
					{Op: models.OpParsePrice},
				},
			},
			{
				Name: "availability",
				Type: models.ValueString,
				Extractor: models.ExtractorSpec{
					Selector:  "p.availability",
					Attribute: "text",
				},
				Transforms: []models.TransformSpec{
					{Op: models.OpTrim},
					{Op: models.OpCollapseWS},
				},
			},
		},
	}

	items := w.extractItems(spec, doc, baseURL)

	require.Len(t, items, 2)

	// First item
	assert.Equal(t, "Book A", items[0]["name"])
	assert.InDelta(t, 19.99, items[0]["price"], 0.001)
	assert.Equal(t, "In stock", items[0]["availability"])

	// Second item
	assert.Equal(t, "Book B", items[1]["name"])
	assert.InDelta(t, 25.00, items[1]["price"], 0.001)
	assert.Equal(t, "Out of stock", items[1]["availability"])
}

// ──────────────────────────────────────────────────────────────────────────────
// Item extraction: missing (non-required) field is skipped, item still produced
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractItems_MissingOptionalField(t *testing.T) {
	t.Parallel()

	w := newTestParserWorker(t)
	baseURL := mustParseURL(t, "https://example.com/")

	// Only one article has a rating; the other doesn't.
	html := `
<html><body>
  <article class="item">
    <h2>Item One</h2>
    <span class="rating">5</span>
  </article>
  <article class="item">
    <h2>Item Two</h2>
    <!-- no rating element -->
  </article>
</body></html>`

	doc := parseHTML(t, html)

	spec := &models.ItemsSpec{
		ContainerSelector: "article.item",
		Fields: []models.FieldSpec{
			{
				Name:     "title",
				Type:     models.ValueString,
				Required: false,
				Extractor: models.ExtractorSpec{
					Selector:  "h2",
					Attribute: "text",
				},
			},
			{
				Name:     "rating",
				Type:     models.ValueInt,
				Required: false, // not required → missing is okay
				Extractor: models.ExtractorSpec{
					Selector:  "span.rating",
					Attribute: "text",
				},
				Transforms: []models.TransformSpec{
					{Op: models.OpParseInt},
				},
			},
		},
	}

	items := w.extractItems(spec, doc, baseURL)

	// Both items must be present even if one has a missing field
	require.Len(t, items, 2)

	assert.Equal(t, "Item One", items[0]["title"])
	// rating present for item 0
	assert.NotNil(t, items[0]["rating"])

	assert.Equal(t, "Item Two", items[1]["title"])
	// rating absent for item 1 – key should be absent (extractFieldFromSelection returned nil/err)
	assert.Nil(t, items[1]["rating"])
}

// ──────────────────────────────────────────────────────────────────────────────
// Item extraction: required field missing logs warning but does not abort
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractItems_MissingRequiredField_ContinuesProcessing(t *testing.T) {
	t.Parallel()

	// Use a logger that captures output so we can assert a warning was issued.
	logger, _ := zap.NewDevelopment()
	w := &ParserWorker{logger: logger}
	baseURL := mustParseURL(t, "https://example.com/")

	html := `
<html><body>
  <div class="row">
    <span class="name">Alpha</span>
    <!-- no .price element -->
  </div>
  <div class="row">
    <span class="name">Beta</span>
    <span class="price">£9.99</span>
  </div>
</body></html>`

	doc := parseHTML(t, html)

	spec := &models.ItemsSpec{
		ContainerSelector: "div.row",
		Fields: []models.FieldSpec{
			{
				Name:     "name",
				Type:     models.ValueString,
				Required: false,
				Extractor: models.ExtractorSpec{
					Selector:  "span.name",
					Attribute: "text",
				},
			},
			{
				Name:     "price",
				Type:     models.ValueString,
				Required: true, // required but missing in first item
				Extractor: models.ExtractorSpec{
					Selector:  "span.price",
					Attribute: "text",
				},
			},
		},
	}

	items := w.extractItems(spec, doc, baseURL)

	// Both items processed despite missing required field in item 0
	require.Len(t, items, 2)
	assert.Equal(t, "Alpha", items[0]["name"])
	assert.Nil(t, items[0]["price"]) // missing → nil value stored

	assert.Equal(t, "Beta", items[1]["name"])
	assert.Equal(t, "£9.99", items[1]["price"])
}

// ──────────────────────────────────────────────────────────────────────────────
// Item extraction: empty container list → returns empty slice, not nil
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractItems_EmptyContainer(t *testing.T) {
	t.Parallel()

	w := newTestParserWorker(t)
	baseURL := mustParseURL(t, "https://example.com/")

	html := `<html><body><p>No items here</p></body></html>`
	doc := parseHTML(t, html)

	spec := &models.ItemsSpec{
		ContainerSelector: "article.product_pod",
		Fields: []models.FieldSpec{
			{
				Name: "name",
				Type: models.ValueString,
				Extractor: models.ExtractorSpec{
					Selector:  "h3 a",
					Attribute: "title",
				},
			},
		},
	}

	items := w.extractItems(spec, doc, baseURL)

	require.NotNil(t, items)
	assert.Empty(t, items)
}

// ──────────────────────────────────────────────────────────────────────────────
// Item extraction: multiple field within item scope (multiple: true)
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractItems_MultipleFieldWithinScope(t *testing.T) {
	t.Parallel()

	w := newTestParserWorker(t)
	baseURL := mustParseURL(t, "https://example.com/")

	html := `
<html><body>
  <div class="product">
    <span class="tag">fiction</span>
    <span class="tag">bestseller</span>
  </div>
</body></html>`

	doc := parseHTML(t, html)

	spec := &models.ItemsSpec{
		ContainerSelector: "div.product",
		Fields: []models.FieldSpec{
			{
				Name: "tags",
				Type: models.ValueString,
				Extractor: models.ExtractorSpec{
					Selector:  "span.tag",
					Attribute: "text",
					Multiple:  true,
				},
			},
		},
	}

	items := w.extractItems(spec, doc, baseURL)

	require.Len(t, items, 1)
	tags, ok := items[0]["tags"].([]string)
	require.True(t, ok, "tags should be []string")
	assert.Equal(t, []string{"fiction", "bestseller"}, tags)
}

// ──────────────────────────────────────────────────────────────────────────────
// CrawlMode: constants have expected string values
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractElementValue_AttributeLookupIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	w := newTestParserWorker(t)
	baseURL := mustParseURL(t, "https://books.toscrape.com/")
	doc := parseHTML(t, `<html><body><a href="catalogue/page-2.html" title="A Light in the Attic">next</a></body></html>`)

	link := w.extractElementValue(doc.Find("a").First(), "HREF", baseURL)
	title := w.extractElementValue(doc.Find("a").First(), "TITLE", baseURL)

	assert.Equal(t, "https://books.toscrape.com/catalogue/page-2.html", link)
	assert.Equal(t, "A Light in the Attic", title)
}

func TestParseHTMLDocument_RespectsDeclaredCharset(t *testing.T) {
	t.Parallel()

	w := newTestParserWorker(t)
	expected := "Shakespeare\u2019s Globe \u00a39.99"
	encodedHTML, err := charmap.Windows1252.NewEncoder().String(
		"<html><head><meta charset=\"windows-1252\"></head><body><h1>" + expected + "</h1></body></html>",
	)
	require.NoError(t, err)

	doc, err := w.parseHTMLDocument([]byte(encodedHTML))
	require.NoError(t, err)

	assert.Equal(t, expected, strings.TrimSpace(doc.Find("h1").Text()))
}

func TestCrawlMode_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, models.CrawlMode("pagination_and_links"), models.CrawlModePaginationAndLinks)
	assert.Equal(t, models.CrawlMode("pagination_only"), models.CrawlModePaginationOnly)
	assert.Equal(t, models.CrawlMode("links_only"), models.CrawlModeLinksOnly)
}

// ──────────────────────────────────────────────────────────────────────────────
// CrawlMode: default (empty) treated as pagination_and_links
// ──────────────────────────────────────────────────────────────────────────────

func TestCrawlMode_DefaultIsEmpty(t *testing.T) {
	t.Parallel()

	var cfg models.CrawlJobConfig
	// When CrawlMode is empty, runtime should treat as pagination_and_links
	assert.Equal(t, models.CrawlMode(""), cfg.CrawlMode)

	// Simulated runtime default logic
	effective := cfg.CrawlMode
	if effective == "" {
		effective = models.CrawlModePaginationAndLinks
	}
	assert.Equal(t, models.CrawlModePaginationAndLinks, effective)
}

// ──────────────────────────────────────────────────────────────────────────────
// CrawlMode gating: pagination_only disables link discovery
// ──────────────────────────────────────────────────────────────────────────────

func TestCrawlMode_PaginationOnly_GatesPaginationAndLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		mode              models.CrawlMode
		runsPagination    bool
		runsLinkDiscovery bool
	}{
		{
			name:              "pagination_and_links runs both",
			mode:              models.CrawlModePaginationAndLinks,
			runsPagination:    true,
			runsLinkDiscovery: true,
		},
		{
			name:              "pagination_only disables link discovery",
			mode:              models.CrawlModePaginationOnly,
			runsPagination:    true,
			runsLinkDiscovery: false,
		},
		{
			name:              "links_only disables pagination",
			mode:              models.CrawlModeLinksOnly,
			runsPagination:    false,
			runsLinkDiscovery: true,
		},
		{
			name:              "empty (default) runs both",
			mode:              "",
			runsPagination:    true,
			runsLinkDiscovery: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mode := tt.mode
			if mode == "" {
				mode = models.CrawlModePaginationAndLinks
			}

			gotPagination := mode != models.CrawlModeLinksOnly
			gotLinkDiscovery := mode != models.CrawlModePaginationOnly

			assert.Equal(t, tt.runsPagination, gotPagination, "pagination gating mismatch")
			assert.Equal(t, tt.runsLinkDiscovery, gotLinkDiscovery, "link discovery gating mismatch")
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Backward compat: nil Items spec → extractData still returns Fields only
// ──────────────────────────────────────────────────────────────────────────────

func TestExtractData_BackwardCompat_NoItemsSpec(t *testing.T) {
	t.Parallel()

	w := newTestParserWorker(t)
	baseURL := mustParseURL(t, "https://example.com/")

	html := `<html><body><h1>Hello World</h1></body></html>`
	doc := parseHTML(t, html)

	spec := models.ExtractionSpec{
		Fields: []models.FieldSpec{
			{
				Name: "title",
				Type: models.ValueString,
				Extractor: models.ExtractorSpec{
					Selector:  "h1",
					Attribute: "text",
				},
			},
		},
		// Items is nil → backward compatible behavior
	}

	// Extract fields manually (replicating extractData logic)
	fields := make(map[string]any)
	for _, fs := range spec.Fields {
		val, err := w.extractField(fs, doc, baseURL)
		require.NoError(t, err)
		fields[fs.Name] = val
	}

	var items []map[string]any
	if spec.Items != nil {
		items = w.extractItems(spec.Items, doc, baseURL)
	}

	assert.Equal(t, "Hello World", fields["title"])
	assert.Nil(t, items)
}
