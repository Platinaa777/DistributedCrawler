package converters

import (
	"testing"
	"time"

	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToProtoPreview_ConvertsOptionalFields(t *testing.T) {
	t.Parallel()

	id := valueobjects.GeneratePreviewID()
	finalURL := "https://example.com/final"
	expiresAt := time.Now().UTC().Add(time.Hour).Round(0)
	createdAt := time.Now().UTC().Round(0)

	proto := ToProtoPreview(&models.Preview{
		ID:          id,
		SourceURL:   "https://example.com",
		FinalURL:    &finalURL,
		MinioKey:    "preview.html",
		ContentType: "text/html",
		DownloadURL: "https://signed",
		CreatedAt:   createdAt,
		ExpiresAt:   &expiresAt,
	})
	require.NotNil(t, proto)
	assert.Equal(t, id.String(), proto.Id)
	assert.Equal(t, &finalURL, proto.FinalUrl)
	assert.True(t, proto.CreatedAt.AsTime().Equal(createdAt))
	assert.True(t, proto.ExpiresAt.AsTime().Equal(expiresAt))
	assert.Nil(t, ToProtoPreview(nil))
}

