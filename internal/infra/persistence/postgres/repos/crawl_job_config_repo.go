package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	crawljobconfig "distributed-crawler/internal/domain/crawl/repos/crawl_job_config"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"

	sq "github.com/Masterminds/squirrel"
)

const (
	configTableName = "crawl_job_configs"

	configIDColumn               = "id"
	configNameColumn             = "name"
	configExtractionSpecColumn   = "extraction_spec"
	configScopesColumn           = "scopes"
	configSeedsColumn            = "seeds"
	configRateLimitColumn        = "rate_limit"
	configRetriesColumn          = "retries"
	configAuthColumn             = "auth"
	configScheduleColumn         = "schedule"
	configRespectRobotsTxtColumn = "respect_robots_txt"
)

type crawlJobConfigRepository struct {
	client persistence.Client
}

func NewCrawlJobConfigRepository(client persistence.Client) crawljobconfig.CrawlJobConfigRepository {
	return &crawlJobConfigRepository{client: client}
}

func (r *crawlJobConfigRepository) Create(ctx context.Context, entity models.CrawlJobConfig) (valueobjects.ID, error) {
	dbEntity, err := converters.SaveCrawlJobConfigToSnapshot(entity)
	if err != nil {
		return valueobjects.ID{}, err
	}

	builder := sq.Insert(configTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(
			configIDColumn,
			configNameColumn,
			configExtractionSpecColumn,
			configScopesColumn,
			configSeedsColumn,
			configRateLimitColumn,
			configRetriesColumn,
			configAuthColumn,
			configScheduleColumn,
			configRespectRobotsTxtColumn,
		).
		Values(
			dbEntity.ID,
			dbEntity.Name,
			dbEntity.ExtractionSpec,
			dbEntity.Scopes,
			dbEntity.Seeds,
			dbEntity.RateLimit,
			dbEntity.Retries,
			dbEntity.Auth,
			dbEntity.Schedule,
			dbEntity.RespectRobotsTxt,
		).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return valueobjects.ID{}, err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Create",
		QueryRaw: query,
	}

	var id string
	err = r.client.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		return valueobjects.ID{}, err
	}

	return valueobjects.NewID(id)
}

func (r *crawlJobConfigRepository) Get(ctx context.Context, id valueobjects.ID) (*models.CrawlJobConfig, error) {
	builder := sq.Select(
		configIDColumn,
		configNameColumn,
		configExtractionSpecColumn,
		configScopesColumn,
		configSeedsColumn,
		configRateLimitColumn,
		configRetriesColumn,
		configAuthColumn,
		configScheduleColumn,
		configRespectRobotsTxtColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(configTableName).
		Where(sq.Eq{configIDColumn: id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Get",
		QueryRaw: query,
	}

	var snapshot snapshots.CrawlJobConfigSnapshot
	err = r.client.DB().ScanOneContext(ctx, &snapshot, q, args...)
	if err != nil {
		return nil, err
	}

	return converters.RestoreCrawlJobConfigFromSnapshot(snapshot)
}

func (r *crawlJobConfigRepository) Update(ctx context.Context, entity models.CrawlJobConfig) error {
	dbEntity, err := converters.SaveCrawlJobConfigToSnapshot(entity)
	if err != nil {
		return err
	}

	builder := sq.Update(configTableName).
		PlaceholderFormat(sq.Dollar).
		Set(configNameColumn, dbEntity.Name).
		Set(configExtractionSpecColumn, dbEntity.ExtractionSpec).
		Set(configScopesColumn, dbEntity.Scopes).
		Set(configSeedsColumn, dbEntity.Seeds).
		Set(configRateLimitColumn, dbEntity.RateLimit).
		Set(configRetriesColumn, dbEntity.Retries).
		Set(configAuthColumn, dbEntity.Auth).
		Set(configScheduleColumn, dbEntity.Schedule).
		Set(configRespectRobotsTxtColumn, dbEntity.RespectRobotsTxt).
		Where(sq.Eq{configIDColumn: dbEntity.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Update",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (r *crawlJobConfigRepository) Delete(ctx context.Context, id valueobjects.ID) error {
	builder := sq.Delete(configTableName).
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{configIDColumn: id.String()})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.Delete",
		QueryRaw: query,
	}

	_, err = r.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (r *crawlJobConfigRepository) ListAll(ctx context.Context, limit, offset int) ([]*models.CrawlJobConfig, error) {
	builder := sq.Select(
		configIDColumn,
		configNameColumn,
		configExtractionSpecColumn,
		configScopesColumn,
		configSeedsColumn,
		configRateLimitColumn,
		configRetriesColumn,
		configAuthColumn,
		configScheduleColumn,
		configRespectRobotsTxtColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(configTableName).
		OrderBy(configNameColumn + " ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_job_config_repository.ListAll",
		QueryRaw: query,
	}

	var snapshots []snapshots.CrawlJobConfigSnapshot
	err = r.client.DB().ScanAllContext(ctx, &snapshots, q, args...)
	if err != nil {
		return nil, err
	}

	configs := make([]*models.CrawlJobConfig, 0, len(snapshots))
	for _, snapshot := range snapshots {
		config, err := converters.RestoreCrawlJobConfigFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	return configs, nil
}
