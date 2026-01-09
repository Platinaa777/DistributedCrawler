package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/page_extract"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

const (
	pageExtractTable = "page_extract"
	pageLinkTable    = "page_link"
	pageImageTable   = "page_image"

	peTaskID            = "task_id"
	peTitle             = "title"
	peMetaDescription   = "meta_description"
	peCanonicalURL      = "canonical_url"
	peMetadata          = "metadata"
	peLinkCount         = "link_count"
	peImageCount        = "image_count"
	peExternalLinkCount = "external_link_count"
	peWordCount         = "word_count"
	peParsedAt          = "parsed_at"
	peCreatedAt         = "created_at"

	plID         = "id"
	plTaskID     = "task_id"
	plURL        = "url"
	plAnchorText = "anchor_text"
	plIsExternal = "is_external"
	plCreatedAt  = "created_at"

	piID        = "id"
	piTaskID    = "task_id"
	piURL       = "url"
	piAltText   = "alt_text"
	piCreatedAt = "created_at"
)

type pageExtractRepository struct {
	client persistence.Client
}

func NewPageExtractRepository(client persistence.Client) page_extract.PageExtractRepository {
	return &pageExtractRepository{client: client}
}

// Save creates or updates a page extract record (UPSERT for idempotency)
func (r *pageExtractRepository) Save(ctx context.Context, extract *models.PageExtract) error {
	snapshot, err := converters.SavePageExtractToSnapshot(extract)
	if err != nil {
		return fmt.Errorf("failed to convert to snapshot: %w", err)
	}

	builder := sq.Insert(pageExtractTable).
		PlaceholderFormat(sq.Dollar).
		Columns(
			peTaskID, peTitle, peMetaDescription, peCanonicalURL, peMetadata,
			peLinkCount, peImageCount, peExternalLinkCount, peWordCount,
			peParsedAt, peCreatedAt,
		).
		Values(
			snapshot.TaskID, snapshot.Title, snapshot.MetaDescription,
			snapshot.CanonicalURL, snapshot.Metadata, snapshot.LinkCount,
			snapshot.ImageCount, snapshot.ExternalLinkCount, snapshot.WordCount,
			snapshot.ParsedAt, snapshot.CreatedAt,
		).
		Suffix(`
			ON CONFLICT (task_id) DO UPDATE SET
				title = EXCLUDED.title,
				meta_description = EXCLUDED.meta_description,
				canonical_url = EXCLUDED.canonical_url,
				metadata = EXCLUDED.metadata,
				link_count = EXCLUDED.link_count,
				image_count = EXCLUDED.image_count,
				external_link_count = EXCLUDED.external_link_count,
				word_count = EXCLUDED.word_count,
				parsed_at = EXCLUDED.parsed_at
		`)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_extract_repository.Save",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("failed to save page extract: %w", err)
	}

	return nil
}

// GetByTaskID retrieves a page extract by task ID
func (r *pageExtractRepository) GetByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) (*models.PageExtract, error) {
	builder := sq.Select(
		peTaskID, peTitle, peMetaDescription, peCanonicalURL, peMetadata,
		peLinkCount, peImageCount, peExternalLinkCount, peWordCount,
		peParsedAt, peCreatedAt,
	).
		PlaceholderFormat(sq.Dollar).
		From(pageExtractTable).
		Where(sq.Eq{peTaskID: taskID.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_extract_repository.GetByTaskID",
		QueryRaw: query,
	}

	var snapshot snapshots.PageExtractSnapshot
	err = r.client.DB().ScanOneContext(ctx, &snapshot, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get page extract: %w", err)
	}

	return converters.RestorePageExtractFromSnapshot(&snapshot)
}

// SaveLinks saves extracted links (bulk insert with ON CONFLICT DO NOTHING)
func (r *pageExtractRepository) SaveLinks(ctx context.Context, links []*models.PageLink) error {
	if len(links) == 0 {
		return nil
	}

	builder := sq.Insert(pageLinkTable).
		PlaceholderFormat(sq.Dollar).
		Columns(plID, plTaskID, plURL, plAnchorText, plIsExternal, plCreatedAt)

	for _, link := range links {
		snapshot := converters.SavePageLinkToSnapshot(link)
		builder = builder.Values(
			snapshot.ID, snapshot.TaskID, snapshot.URL,
			snapshot.AnchorText, snapshot.IsExternal, snapshot.CreatedAt,
		)
	}

	builder = builder.Suffix("ON CONFLICT (task_id, url) DO NOTHING")

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_extract_repository.SaveLinks",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("failed to save links: %w", err)
	}

	return nil
}

// SaveImages saves extracted images (bulk insert with ON CONFLICT DO NOTHING)
func (r *pageExtractRepository) SaveImages(ctx context.Context, images []*models.PageImage) error {
	if len(images) == 0 {
		return nil
	}

	builder := sq.Insert(pageImageTable).
		PlaceholderFormat(sq.Dollar).
		Columns(piID, piTaskID, piURL, piAltText, piCreatedAt)

	for _, image := range images {
		snapshot := converters.SavePageImageToSnapshot(image)
		builder = builder.Values(
			snapshot.ID, snapshot.TaskID, snapshot.URL,
			snapshot.AltText, snapshot.CreatedAt,
		)
	}

	builder = builder.Suffix("ON CONFLICT (task_id, url) DO NOTHING")

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_extract_repository.SaveImages",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("failed to save images: %w", err)
	}

	return nil
}

// GetLinksByTaskID retrieves all links for a task
func (r *pageExtractRepository) GetLinksByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) ([]*models.PageLink, error) {
	builder := sq.Select(plID, plTaskID, plURL, plAnchorText, plIsExternal, plCreatedAt).
		PlaceholderFormat(sq.Dollar).
		From(pageLinkTable).
		Where(sq.Eq{plTaskID: taskID.String()}).
		OrderBy(plCreatedAt + " ASC")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_extract_repository.GetLinksByTaskID",
		QueryRaw: query,
	}

	var snapshots []snapshots.PageLinkSnapshot
	err = r.client.DB().ScanAllContext(ctx, &snapshots, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get links: %w", err)
	}

	links := make([]*models.PageLink, 0, len(snapshots))
	for _, snapshot := range snapshots {
		link, err := converters.RestorePageLinkFromSnapshot(&snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to restore link: %w", err)
		}
		links = append(links, link)
	}

	return links, nil
}

// GetImagesByTaskID retrieves all images for a task
func (r *pageExtractRepository) GetImagesByTaskID(ctx context.Context, taskID valueobjects.CrawlTaskID) ([]*models.PageImage, error) {
	builder := sq.Select(piID, piTaskID, piURL, piAltText, piCreatedAt).
		PlaceholderFormat(sq.Dollar).
		From(pageImageTable).
		Where(sq.Eq{piTaskID: taskID.String()}).
		OrderBy(piCreatedAt + " ASC")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	q := persistence.Query{
		Name:     "page_extract_repository.GetImagesByTaskID",
		QueryRaw: query,
	}

	var snapshots []snapshots.PageImageSnapshot
	err = r.client.DB().ScanAllContext(ctx, &snapshots, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get images: %w", err)
	}

	images := make([]*models.PageImage, 0, len(snapshots))
	for _, snapshot := range snapshots {
		image, err := converters.RestorePageImageFromSnapshot(&snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to restore image: %w", err)
		}
		images = append(images, image)
	}

	return images, nil
}
