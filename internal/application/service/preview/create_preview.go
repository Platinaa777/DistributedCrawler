package preview

import (
	"context"
	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"fmt"
	"time"
)

const (
	UrlTTL = 60 // 60 minutes TTL
)

func (s *previewServ) CreatePreview(ctx context.Context, cmd service.CreatePreviewCommand) (*models.Preview, error) {
	fetcher := s.fetcherFactory.CreateFetcher(cmd.Auth, s.retryPolicy)

	// Fetch HTML from URL
	fetchResult, err := fetcher.Fetch(ctx, cmd.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", cmd.URL, err)
	}

	// Sanitize HTML for safe iframe rendering
	sanitizedHTML := s.sanitizer.Sanitize(fetchResult.Body)

	// Generate preview ID and MinIO key
	previewID := valueobjects.GeneratePreviewID()
	minioKey := fmt.Sprintf("previews/%s.html", previewID.String())

	var preview *models.Preview

	// Use transaction to ensure atomicity
	err = s.txManager.ReadCommitted(ctx, func(ctx context.Context) error {
		// Store sanitized HTML in MinIO
		if err := s.contentStore.Store(ctx, minioKey, sanitizedHTML, "text/html; charset=utf-8"); err != nil {
			return fmt.Errorf("failed to store HTML in MinIO: %w", err)
		}

		// Generate presigned URL (inside transaction)
		downloadURL, err := s.urlGenerator.PresignGetURL(minioKey, UrlTTL)
		if err != nil {
			return fmt.Errorf("failed to generate presigned URL: %w", err)
		}

		// Create domain model
		preview = &models.Preview{
			ID:          previewID,
			SourceURL:   cmd.URL,
			MinioKey:    minioKey,
			ContentType: "text/html; charset=utf-8",
			DownloadURL: downloadURL,
			CreatedAt:   time.Now(),
		}

		// Set FinalURL if different from source
		if fetchResult.FinalURL != cmd.URL {
			preview.FinalURL = &fetchResult.FinalURL
		}

		// Persist preview metadata
		createdID, err := s.previewRepo.Create(ctx, *preview)
		if err != nil {
			return fmt.Errorf("failed to create preview: %w", err)
		}

		preview.ID = createdID

		return nil
	})

	if err != nil {
		return nil, err
	}

	return preview, nil
}
