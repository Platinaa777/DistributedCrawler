package preview

import (
	"context"
	"errors"
	"io"
	"testing"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	previewrepo "distributed-crawler/internal/domain/crawl/repos/preview"
	"distributed-crawler/internal/domain/crawl/services"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type previewRepoFake struct {
	createFn func(ctx context.Context, entity models.Preview) (valueobjects.PreviewID, error)
	getFn    func(ctx context.Context, id valueobjects.PreviewID) (*models.Preview, error)
}

func (f previewRepoFake) Create(ctx context.Context, entity models.Preview) (valueobjects.PreviewID, error) {
	return f.createFn(ctx, entity)
}
func (f previewRepoFake) Get(ctx context.Context, id valueobjects.PreviewID) (*models.Preview, error) {
	return f.getFn(ctx, id)
}

type fetcherFactoryFake struct {
	createFn func(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher
}

func (f fetcherFactoryFake) CreateFetcher(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher {
	return f.createFn(auth, retry)
}

type fetcherFake struct {
	fetchFn func(ctx context.Context, url string) (*services.FetchResult, error)
}

func (f fetcherFake) Fetch(ctx context.Context, url string) (*services.FetchResult, error) {
	return f.fetchFn(ctx, url)
}

type contentStoreFake struct {
	storeFn func(ctx context.Context, key string, content []byte, contentType string) error
}

func (f contentStoreFake) Store(ctx context.Context, key string, content []byte, contentType string) error {
	return f.storeFn(ctx, key, content, contentType)
}
func (f contentStoreFake) Get(context.Context, string) ([]byte, error)                   { return nil, nil }
func (f contentStoreFake) GetReader(context.Context, string) (io.ReadCloser, error)      { return nil, nil }
func (f contentStoreFake) Delete(context.Context, string) error                           { return nil }
func (f contentStoreFake) Exists(context.Context, string) (bool, error)                   { return false, nil }

type sanitizerFake struct {
	sanitizeFn func(html []byte) []byte
}

func (f sanitizerFake) Sanitize(html []byte) []byte {
	return f.sanitizeFn(html)
}

type urlGenFake struct {
	presignFn func(key string, ttlMinutes int) (string, error)
}

func (f urlGenFake) PresignGetURL(key string, ttlMinutes int) (string, error) {
	return f.presignFn(key, ttlMinutes)
}

type txManagerFake struct {
	runFn func(ctx context.Context, exec persistence.Handler) error
}

func (f txManagerFake) ReadCommitted(ctx context.Context, exec persistence.Handler) error {
	return f.runFn(ctx, exec)
}

var _ previewrepo.PreviewRepository = previewRepoFake{}

func TestCreatePreview_SanitizesStoresAndPersistsMetadata(t *testing.T) {
	t.Parallel()

	savedID := valueobjects.GeneratePreviewID()
	svc := NewService(
		previewRepoFake{
			createFn: func(ctx context.Context, entity models.Preview) (valueobjects.PreviewID, error) {
				assert.Equal(t, "https://example.com", entity.SourceURL)
				assert.Equal(t, "text/html; charset=utf-8", entity.ContentType)
				assert.NotEmpty(t, entity.MinioKey)
				assert.Equal(t, "https://signed.local/preview", entity.DownloadURL)
				require.NotNil(t, entity.FinalURL)
				assert.Equal(t, "https://example.com/final", *entity.FinalURL)
				return savedID, nil
			},
			getFn: func(ctx context.Context, id valueobjects.PreviewID) (*models.Preview, error) { return nil, nil },
		},
		fetcherFactoryFake{
			createFn: func(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher {
				assert.Equal(t, "cookie=value", auth.Cookie)
				assert.EqualValues(t, 3, retry.MaxAttempts)
				return fetcherFake{
					fetchFn: func(ctx context.Context, url string) (*services.FetchResult, error) {
						assert.Equal(t, "https://example.com", url)
						return &services.FetchResult{
							Body:     []byte("<html><script>x</script><body>ok</body></html>"),
							FinalURL: "https://example.com/final",
						}, nil
					},
				}
			},
		},
		contentStoreFake{
			storeFn: func(ctx context.Context, key string, content []byte, contentType string) error {
				assert.Contains(t, key, "previews/")
				assert.Contains(t, key, ".html")
				assert.Equal(t, []byte("<html><body>clean</body></html>"), content)
				assert.Equal(t, "text/html; charset=utf-8", contentType)
				return nil
			},
		},
		sanitizerFake{sanitizeFn: func(html []byte) []byte {
			assert.Contains(t, string(html), "<script>")
			return []byte("<html><body>clean</body></html>")
		}},
		urlGenFake{presignFn: func(key string, ttlMinutes int) (string, error) {
			assert.Equal(t, UrlTTL, ttlMinutes)
			return "https://signed.local/preview", nil
		}},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error {
			return exec(ctx)
		}},
		models.RetryPolicy{MaxAttempts: 3},
	)

	preview, err := svc.CreatePreview(context.Background(), service.CreatePreviewCommand{
		URL:  "https://example.com",
		Auth: models.AuthOptions{Cookie: "cookie=value"},
	})
	require.NoError(t, err)
	assert.Equal(t, savedID, preview.ID)
	assert.Equal(t, "https://signed.local/preview", preview.DownloadURL)
}

func TestCreatePreview_ReturnsTransactionWrappedErrors(t *testing.T) {
	t.Parallel()

	svc := NewService(
		previewRepoFake{
			createFn: func(ctx context.Context, entity models.Preview) (valueobjects.PreviewID, error) {
				return valueobjects.PreviewID{}, nil
			},
			getFn: func(ctx context.Context, id valueobjects.PreviewID) (*models.Preview, error) { return nil, nil },
		},
		fetcherFactoryFake{createFn: func(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher {
			return fetcherFake{fetchFn: func(ctx context.Context, url string) (*services.FetchResult, error) {
				return &services.FetchResult{Body: []byte("x"), FinalURL: url}, nil
			}}
		}},
		contentStoreFake{storeFn: func(ctx context.Context, key string, content []byte, contentType string) error {
			return errors.New("store failed")
		}},
		sanitizerFake{sanitizeFn: func(html []byte) []byte { return html }},
		urlGenFake{presignFn: func(key string, ttlMinutes int) (string, error) { return "", nil }},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		models.RetryPolicy{},
	)

	preview, err := svc.CreatePreview(context.Background(), service.CreatePreviewCommand{URL: "https://example.com"})
	require.Error(t, err)
	assert.Nil(t, preview)
	assert.Contains(t, err.Error(), "failed to store HTML in MinIO")
}

func TestGetPreview_ValidatesIDAndWrapsRepoError(t *testing.T) {
	t.Parallel()

	svc := NewService(
		previewRepoFake{
			createFn: func(ctx context.Context, entity models.Preview) (valueobjects.PreviewID, error) {
				return valueobjects.PreviewID{}, nil
			},
			getFn: func(ctx context.Context, id valueobjects.PreviewID) (*models.Preview, error) {
				return nil, errors.New("db down")
			},
		},
		fetcherFactoryFake{createFn: func(auth models.AuthOptions, retry models.RetryPolicy) services.Fetcher { return nil }},
		contentStoreFake{storeFn: func(ctx context.Context, key string, content []byte, contentType string) error { return nil }},
		sanitizerFake{sanitizeFn: func(html []byte) []byte { return html }},
		urlGenFake{presignFn: func(key string, ttlMinutes int) (string, error) { return "", nil }},
		txManagerFake{runFn: func(ctx context.Context, exec persistence.Handler) error { return exec(ctx) }},
		models.RetryPolicy{},
	)

	got, err := svc.GetPreview(context.Background(), service.GetPreviewQuery{ID: "bad"})
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "invalid preview_id")

	validID := valueobjects.GeneratePreviewID()
	got, err = svc.GetPreview(context.Background(), service.GetPreviewQuery{ID: validID.String()})
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "failed to get preview")
}
