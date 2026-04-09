package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeContentStore struct {
	storeFn func(ctx context.Context, key string, content []byte, contentType string) error
}

func (f *fakeContentStore) Store(ctx context.Context, key string, content []byte, contentType string) error {
	if f.storeFn != nil {
		return f.storeFn(ctx, key, content, contentType)
	}
	return nil
}

func (f *fakeContentStore) Get(context.Context, string) ([]byte, error) { return nil, nil }
func (f *fakeContentStore) GetReader(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeContentStore) Delete(context.Context, string) error         { return nil }
func (f *fakeContentStore) Exists(context.Context, string) (bool, error) { return false, nil }

type fakeScopeValidator struct {
	validateFn func(url string, depth uint64, rules models.ScopeRules) error
}

func (f fakeScopeValidator) Validate(url string, depth uint64, rules models.ScopeRules) error {
	if f.validateFn != nil {
		return f.validateFn(url, depth, rules)
	}
	return nil
}

type fakeRobotsService struct {
	allowFn func(ctx context.Context, urlStr string, userAgent string) (bool, error)
	calls   int
}

func (f *fakeRobotsService) IsAllowed(ctx context.Context, urlStr string, userAgent string) (bool, error) {
	f.calls++
	if f.allowFn != nil {
		return f.allowFn(ctx, urlStr, userAgent)
	}
	return true, nil
}

func loadDocFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	path := filepath.Join(append([]string{"..", "..", "docs"}, parts...)...)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return content
}

func booksExampleSpec() models.ExtractionSpec {
	return models.ExtractionSpec{
		Fields: []models.FieldSpec{
			{
				Name: "page_title",
				Type: models.ValueString,
				Extractor: models.ExtractorSpec{
					Selector:  "div.page-header h1",
					Attribute: "text",
				},
				Transforms: []models.TransformSpec{
					{Op: models.OpTrim},
					{Op: models.OpCollapseWS},
				},
			},
		},
		Items: &models.ItemsSpec{
			ContainerSelector: "article.product_pod",
			Fields: []models.FieldSpec{
				{
					Name: "name",
					Type: models.ValueString,
					Extractor: models.ExtractorSpec{
						Selector:  "h3 a",
						Attribute: "title",
					},
					Transforms: []models.TransformSpec{
						{Op: models.OpTrim},
						{Op: models.OpCollapseWS},
					},
				},
				{
					Name: "price",
					Type: models.ValueFloat,
					Extractor: models.ExtractorSpec{
						Selector:  "p.price_color",
						Attribute: "text",
					},
					Transforms: []models.TransformSpec{{Op: models.OpParsePrice}},
				},
				{
					Name: "availability",
					Type: models.ValueString,
					Extractor: models.ExtractorSpec{
						Selector:  "p.instock.availability",
						Attribute: "text",
					},
					Transforms: []models.TransformSpec{
						{Op: models.OpTrim},
						{Op: models.OpCollapseWS},
					},
				},
				{
					Name: "url",
					Type: models.ValueURL,
					Extractor: models.ExtractorSpec{
						Selector:  "h3 a",
						Attribute: "href",
					},
					Transforms: []models.TransformSpec{{Op: models.OpNormalizeURL}},
				},
			},
		},
		Pagination: []models.PaginationSpec{
			{Name: "next_page", Selector: "ul.pager li.next a", Attribute: "href"},
		},
	}
}

func bigGeekListSpec() models.ExtractionSpec {
	return models.ExtractionSpec{
		Items: &models.ItemsSpec{
			ContainerSelector: "div.catalog-card",
			Fields: []models.FieldSpec{
				{
					Name: "name",
					Type: models.ValueString,
					Extractor: models.ExtractorSpec{
						Selector:  "a.catalog-card__title",
						Attribute: "text",
					},
					Transforms: []models.TransformSpec{{Op: models.OpTrim}, {Op: models.OpCollapseWS}},
				},
				{
					Name: "price",
					Type: models.ValueFloat,
					Extractor: models.ExtractorSpec{
						Selector:  "b.cart-modal-count",
						Attribute: "text",
					},
					Transforms: []models.TransformSpec{{Op: models.OpParsePrice}},
				},
				{
					Name: "url",
					Type: models.ValueURL,
					Extractor: models.ExtractorSpec{
						Selector:  "a.catalog-card__title",
						Attribute: "href",
					},
					Transforms: []models.TransformSpec{{Op: models.OpNormalizeURL}},
				},
				{
					Name: "thumbnail",
					Type: models.ValueURL,
					Extractor: models.ExtractorSpec{
						Selector:  "a.catalog-card__img img",
						Attribute: "src",
					},
					Transforms: []models.TransformSpec{{Op: models.OpNormalizeURL}},
				},
			},
		},
	}
}

func TestExtractData_BooksFixtureExtractsFieldsItemsAndResolvedURLs(t *testing.T) {
	t.Parallel()

	w := &ParserWorker{logger: zap.NewNop()}
	task := &models.CrawlTask{
		ID:  valueobjects.GenerateCrawlTaskID(),
		URL: "https://books.toscrape.com/",
	}

	result, err := w.extractData(context.Background(), task, booksExampleSpec(), loadDocFixture(t, "example1", "All products _ Books to Scrape - Sandbox.html"))
	require.NoError(t, err)

	require.NotNil(t, result.Fields)
	assert.Equal(t, "All products", result.Fields["page_title"])
	require.Len(t, result.Items, 20)
	assert.Equal(t, "A Light in the Attic", result.Items[0]["name"])
	assert.InDelta(t, 51.77, result.Items[0]["price"], 0.001)
	assert.Equal(t, "In stock", result.Items[0]["availability"])
	assert.Equal(t, "https://books.toscrape.com/catalogue/a-light-in-the-attic_1000/index.html", result.Items[0]["url"])
}

func TestExtractData_BigGeekFixtureExtractsCatalogCards(t *testing.T) {
	t.Parallel()

	w := &ParserWorker{logger: zap.NewNop()}
	task := &models.CrawlTask{
		ID:  valueobjects.GenerateCrawlTaskID(),
		URL: "https://biggeek.ru/catalog/macbook-pro-14?sort=price&f253=25996-26002",
	}

	result, err := w.extractData(context.Background(), task, bigGeekListSpec(), loadDocFixture(t, "example4", "items-list.html"))
	require.NoError(t, err)
	require.Len(t, result.Items, 24)

	first := result.Items[0]
	assert.NotEmpty(t, first["name"])
	assert.IsType(t, float64(0), first["price"])
	assert.Contains(t, first["url"], "https://biggeek.ru/products/")
	assert.Contains(t, first["thumbnail"], "https://")
}

func TestExtractWithCSS_MultipleSupportsNegativeIndex(t *testing.T) {
	t.Parallel()

	w := &ParserWorker{logger: zap.NewNop()}
	doc := parseHTML(t, `<html><body><ul><li>one</li><li>two</li><li>three</li></ul></body></html>`)
	index := -1

	value, err := w.extractWithCSS(models.ExtractorSpec{
		Selector:  "li",
		Attribute: "text",
		Multiple:  true,
		Index:     &index,
	}, doc, nil)
	require.NoError(t, err)
	assert.Equal(t, "three", value)
}

func TestExtractElementValue_HTMLPseudoAttributeReturnsInnerHTML(t *testing.T) {
	t.Parallel()

	w := &ParserWorker{logger: zap.NewNop()}
	doc := parseHTML(t, `<html><body><div class="content"><strong>Hello</strong> <em>world</em></div></body></html>`)

	value := w.extractElementValue(doc.Find("div.content").First(), "html", nil)

	assert.Contains(t, value, "<strong>Hello</strong>")
	assert.Contains(t, value, "<em>world</em>")
}

func TestUploadResultToS3_StoresExpectedPayload(t *testing.T) {
	t.Parallel()

	var storedKey string
	var storedType string
	var storedContent []byte

	w := &ParserWorker{
		logger: zap.NewNop(),
		contentStore: &fakeContentStore{storeFn: func(ctx context.Context, key string, content []byte, contentType string) error {
			storedKey = key
			storedType = contentType
			storedContent = append([]byte(nil), content...)
			return nil
		}},
	}

	objectKey, size, err := w.uploadResultToS3(context.Background(), "task-123", "https://example.com/page", &extractionResult{
		Fields: map[string]any{"title": "Example"},
		Items:  []map[string]any{{"name": "Item"}},
	})
	require.NoError(t, err)

	assert.Equal(t, "results/tasks/task-123.json", objectKey)
	assert.Equal(t, int64(len(storedContent)), size)
	assert.Equal(t, objectKey, storedKey)
	assert.Equal(t, "application/json", storedType)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(storedContent, &payload))
	assert.Equal(t, "task-123", payload["task_id"])
	assert.Equal(t, "https://example.com/page", payload["url"])
	assert.Contains(t, payload, "trace_context")
	assert.Contains(t, payload, "fields")
	assert.Contains(t, payload, "items")
}

func TestPreparePaginationLinks_FiltersByAllowedPatternsRobotsAndScheme(t *testing.T) {
	t.Parallel()

	jobID := valueobjects.GenerateCrawlJobID()
	task := &models.CrawlTask{
		ID:    valueobjects.GenerateCrawlTaskID(),
		JobID: jobID,
		URL:   "https://example.com/catalog",
		Depth: 0,
	}

	w := &ParserWorker{
		logger: zap.NewNop(),
		scopeValidator: fakeScopeValidator{
			validateFn: func(url string, depth uint64, rules models.ScopeRules) error {
				if url == "https://example.com/blocked-by-scope" {
					return errors.New("blocked")
				}
				return nil
			},
		},
		robotsTxtService: &fakeRobotsService{
			allowFn: func(ctx context.Context, urlStr string, userAgent string) (bool, error) {
				return urlStr != "https://example.com/blocked-by-robots", nil
			},
		},
	}

	html := []byte(`
		<html><body>
			<a class="page" href="/allowed?page=2">next</a>
			<a class="page" href="/blocked-by-robots">robots</a>
			<a class="page" href="/blocked-by-scope">scope</a>
			<a class="page" href="mailto:test@example.com">mail</a>
			<a class="page" href="/not-matching">other</a>
		</body></html>`)

	tasks, events, err := w.preparePaginationLinks(context.Background(), task, html, &models.CrawlJobConfig{
		RespectRobotsTxt: true,
		Scopes: models.ScopeRules{
			MaxDepth:           2,
			AllowedURLPatterns: []string{"https://example.com/allowed*"},
		},
		ExtractionSpec: models.ExtractionSpec{
			Pagination: []models.PaginationSpec{
				{Name: "next", Selector: "a.page", Attribute: "href", Multiple: true},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Len(t, events, 1)
	assert.Equal(t, "https://example.com/allowed?page=2", tasks[0].URL)
	assert.Equal(t, uint64(1), tasks[0].Depth)
	assert.Equal(t, tasks[0].ID.String(), events[0].AggregateID)
}

func TestPrepareDiscoveredLinks_DeduplicatesAndRespectsDepthLimit(t *testing.T) {
	t.Parallel()

	jobID := valueobjects.GenerateCrawlJobID()
	task := &models.CrawlTask{
		ID:    valueobjects.GenerateCrawlTaskID(),
		JobID: jobID,
		URL:   "https://example.com/catalog",
		Depth: 0,
	}

	w := &ParserWorker{
		logger:           zap.NewNop(),
		scopeValidator:   fakeScopeValidator{},
		robotsTxtService: &fakeRobotsService{},
	}

	html := []byte(`
		<html><body>
			<a href="/item/1#details">one</a>
			<a href="/item/1">duplicate</a>
			<a href="/item/2">two</a>
			<a href="javascript:void(0)">js</a>
		</body></html>`)

	tasks, events, err := w.prepareDiscoveredLinks(context.Background(), task, html, &models.CrawlJobConfig{
		Scopes: models.ScopeRules{MaxDepth: 2},
	})
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	require.Len(t, events, 2)

	urls := []string{tasks[0].URL, tasks[1].URL}
	assert.ElementsMatch(t, []string{
		"https://example.com/item/1",
		"https://example.com/item/2",
	}, urls)

	none, noneEvents, err := w.prepareDiscoveredLinks(context.Background(), &models.CrawlTask{
		ID:    valueobjects.GenerateCrawlTaskID(),
		JobID: jobID,
		URL:   "https://example.com/catalog",
		Depth: 2,
	}, html, &models.CrawlJobConfig{
		Scopes: models.ScopeRules{MaxDepth: 2},
	})
	require.NoError(t, err)
	assert.Nil(t, none)
	assert.Nil(t, noneEvents)
}

func TestPreparePaginationLinks_IgnoresRobotsWhenDisabled(t *testing.T) {
	t.Parallel()

	task := &models.CrawlTask{
		ID:    valueobjects.GenerateCrawlTaskID(),
		JobID: valueobjects.GenerateCrawlJobID(),
		URL:   "https://example.com/list",
		Depth: 0,
	}

	robots := &fakeRobotsService{
		allowFn: func(ctx context.Context, urlStr string, userAgent string) (bool, error) {
			return false, nil
		},
	}

	w := &ParserWorker{
		logger:           zap.NewNop(),
		scopeValidator:   fakeScopeValidator{},
		robotsTxtService: robots,
	}

	html := []byte(`<html><body><a class="page" href="/blocked">next</a></body></html>`)

	tasks, events, err := w.preparePaginationLinks(context.Background(), task, html, &models.CrawlJobConfig{
		RespectRobotsTxt: false,
		Scopes: models.ScopeRules{
			MaxDepth:           2,
			AllowedURLPatterns: []string{"https://example.com/*"},
		},
		ExtractionSpec: models.ExtractionSpec{
			Pagination: []models.PaginationSpec{
				{Name: "next", Selector: "a.page", Attribute: "href", Multiple: true},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Len(t, events, 1)
	assert.Equal(t, "https://example.com/blocked", tasks[0].URL)
	assert.Equal(t, 0, robots.calls)
}

func TestPrepareDiscoveredLinks_IgnoresRobotsWhenDisabled(t *testing.T) {
	t.Parallel()

	robots := &fakeRobotsService{
		allowFn: func(ctx context.Context, urlStr string, userAgent string) (bool, error) {
			return false, nil
		},
	}

	w := &ParserWorker{
		logger:           zap.NewNop(),
		scopeValidator:   fakeScopeValidator{},
		robotsTxtService: robots,
	}

	tasks, events, err := w.prepareDiscoveredLinks(context.Background(), &models.CrawlTask{
		ID:    valueobjects.GenerateCrawlTaskID(),
		JobID: valueobjects.GenerateCrawlJobID(),
		URL:   "https://example.com/catalog",
		Depth: 0,
	}, []byte(`<html><body><a href="/blocked">blocked</a></body></html>`), &models.CrawlJobConfig{
		RespectRobotsTxt: false,
		Scopes:           models.ScopeRules{MaxDepth: 1},
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Len(t, events, 1)
	assert.Equal(t, "https://example.com/blocked", tasks[0].URL)
	assert.Equal(t, 0, robots.calls)
}

func TestApplyTransform_LimitAcceptsJSONNumber(t *testing.T) {
	t.Parallel()

	w := &ParserWorker{logger: zap.NewNop()}
	value := w.applyTransform(models.TransformSpec{
		Op:  models.OpLimit,
		Arg: json.Number("2"),
	}, []string{"a", "b", "c"})

	assert.Equal(t, []string{"a", "b"}, value)
}

func TestConvertToType_StringPreservesSlices(t *testing.T) {
	t.Parallel()

	w := &ParserWorker{logger: zap.NewNop()}

	value, err := w.convertToType([]string{"a", "b"}, models.ValueString)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, value)
}
