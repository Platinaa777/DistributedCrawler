package preview

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
)

type PreviewRepository interface {
	Create(ctx context.Context, entity models.Preview) (valueobjects.PreviewID, error)
	Get(ctx context.Context, id valueobjects.PreviewID) (*models.Preview, error)
}