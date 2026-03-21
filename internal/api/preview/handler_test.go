package preview

import (
	"context"
	"testing"
	"time"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	crawlergrpc "distributed-crawler/pkg/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type fakePreviewService struct {
	createFn func(ctx context.Context, cmd service.CreatePreviewCommand) (*models.Preview, error)
	getFn    func(ctx context.Context, query service.GetPreviewQuery) (*models.Preview, error)
}

func (f fakePreviewService) CreatePreview(ctx context.Context, cmd service.CreatePreviewCommand) (*models.Preview, error) {
	return f.createFn(ctx, cmd)
}
func (f fakePreviewService) GetPreview(ctx context.Context, query service.GetPreviewQuery) (*models.Preview, error) {
	return f.getFn(ctx, query)
}

func TestCreatePreview_ReadsCookieFromMetadata(t *testing.T) {
	t.Parallel()

	previewID := valueobjects.GeneratePreviewID()
	impl := NewImplementation(fakePreviewService{
		createFn: func(ctx context.Context, cmd service.CreatePreviewCommand) (*models.Preview, error) {
			assert.Equal(t, "https://example.com", cmd.URL)
			assert.Equal(t, "session=abc", cmd.Auth.Cookie)
			return &models.Preview{ID: previewID}, nil
		},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-preview-cookie", " session=abc "))
	resp, err := impl.CreatePreview(ctx, &crawlergrpc.CreatePreviewRequest{Url: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, previewID.String(), resp.Id)
}

func TestGetPreview_ConvertsDomainPreview(t *testing.T) {
	t.Parallel()

	previewID := valueobjects.GeneratePreviewID()
	finalURL := "https://example.com/final"
	expiresAt := time.Now().UTC().Add(10 * time.Minute).Round(0)
	createdAt := time.Now().UTC().Round(0)

	impl := NewImplementation(fakePreviewService{
		getFn: func(ctx context.Context, query service.GetPreviewQuery) (*models.Preview, error) {
			assert.Equal(t, "preview-id", query.ID)
			return &models.Preview{
				ID:          previewID,
				SourceURL:   "https://example.com/source",
				FinalURL:    &finalURL,
				MinioKey:    "previews/123.html",
				ContentType: "text/html",
				DownloadURL: "https://minio.local/file",
				CreatedAt:   createdAt,
				ExpiresAt:   &expiresAt,
			}, nil
		},
	})

	resp, err := impl.GetPreview(context.Background(), &crawlergrpc.GetPreviewRequest{Id: "preview-id"})
	require.NoError(t, err)
	require.NotNil(t, resp.Preview)
	assert.Equal(t, previewID.String(), resp.Preview.Id)
	assert.Equal(t, "https://example.com/source", resp.Preview.SourceUrl)
	assert.Equal(t, &finalURL, resp.Preview.FinalUrl)
	assert.Equal(t, "previews/123.html", resp.Preview.MinioKey)
	assert.True(t, resp.Preview.CreatedAt.AsTime().Equal(createdAt))
	assert.True(t, resp.Preview.ExpiresAt.AsTime().Equal(expiresAt))
}

