package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	"distributed-crawler/internal/domain/crawl/repos/preview"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"

	sq "github.com/Masterminds/squirrel"
)

const (
	previewTableName         = "previews"
	previewIDColumn          = "id"
	previewSourceURLColumn   = "source_url"
	previewFinalURLColumn    = "final_url"
	previewMinioKeyColumn    = "minio_key"
	previewContentTypeColumn = "content_type"
	previewDownloadURLColumn = "download_url"
	previewCreatedAtColumn   = "created_at"
	previewExpiresAtColumn   = "expires_at"
)

type previewRepository struct {
	client persistence.Client
}

func NewPreviewRepository(client persistence.Client) preview.PreviewRepository {
	return &previewRepository{client: client}
}

func (r *previewRepository) Create(ctx context.Context, entity models.Preview) (valueobjects.PreviewID, error) {
	dbEntity := converters.SavePreviewToSnapshot(entity)

	builder := sq.Insert(previewTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(
			previewIDColumn,
			previewSourceURLColumn,
			previewFinalURLColumn,
			previewMinioKeyColumn,
			previewContentTypeColumn,
			previewDownloadURLColumn,
			previewCreatedAtColumn,
			previewExpiresAtColumn,
		).
		Values(
			dbEntity.ID,
			dbEntity.SourceURL,
			dbEntity.FinalURL,
			dbEntity.MinioKey,
			dbEntity.ContentType,
			dbEntity.DownloadURL,
			dbEntity.CreatedAt,
			dbEntity.ExpiresAt,
		).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return valueobjects.PreviewID{}, err
	}

	q := persistence.Query{
		Name:     "preview_repository.Create",
		QueryRaw: query,
	}

	var id string
	err = r.client.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		return valueobjects.PreviewID{}, err
	}

	return valueobjects.NewPreviewID(id)
}

func (r *previewRepository) Get(ctx context.Context, id valueobjects.PreviewID) (*models.Preview, error) {
	builder := sq.Select(
		previewIDColumn,
		previewSourceURLColumn,
		previewFinalURLColumn,
		previewMinioKeyColumn,
		previewContentTypeColumn,
		previewDownloadURLColumn,
		previewCreatedAtColumn,
		previewExpiresAtColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(previewTableName).
		Where(sq.Eq{previewIDColumn: id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "preview_repository.Get",
		QueryRaw: query,
	}

	var previewSnapshot snapshots.PreviewSnapshot
	err = r.client.DB().ScanOneContext(ctx, &previewSnapshot, q, args...)
	if err != nil {
		return nil, err
	}

	return converters.RestorePreviewFromSnapshot(previewSnapshot)
}
