package service

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
)

// Commands for Preview management

type CreatePreviewCommand struct {
	URL    string
}

// Queries for Preview

type GetPreviewQuery struct {
	ID string
}

// Service interface

type PreviewService interface {
	CreatePreview(ctx context.Context, cmd CreatePreviewCommand) (*models.Preview, error)
	GetPreview(ctx context.Context, query GetPreviewQuery) (*models.Preview, error)
}