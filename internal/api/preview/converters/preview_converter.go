package converters

import (
	"distributed-crawler/internal/domain/crawl/models"
	crawlergrpc "distributed-crawler/pkg/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToProtoPreview converts domain Preview to proto Preview
func ToProtoPreview(preview *models.Preview) *crawlergrpc.Preview {
	if preview == nil {
		return nil
	}

	proto := &crawlergrpc.Preview{
		Id:          preview.ID.String(),
		SourceUrl:   preview.SourceURL,
		MinioKey:    preview.MinioKey,
		ContentType: preview.ContentType,
		CreatedAt:   timestamppb.New(preview.CreatedAt),
		DownloadUrl: preview.DownloadURL,
	}

	// Handle optional FinalURL
	if preview.FinalURL != nil {
		proto.FinalUrl = preview.FinalURL
	}

	// Handle optional ExpiresAt
	if preview.ExpiresAt != nil {
		expiresAt := timestamppb.New(*preview.ExpiresAt)
		proto.ExpiresAt = expiresAt
	}

	return proto
}
