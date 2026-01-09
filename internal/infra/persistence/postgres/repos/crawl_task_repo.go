package repos

import (
	"context"
	"distributed-crawler/internal/domain/crawl/models"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"

	sq "github.com/Masterminds/squirrel"
)

const (
	taskTableName = "crawl_tasks"

	taskIDColumn         = "id"
	taskJobIDColumn      = "job_id"
	taskURLColumn        = "url"
	taskStatusColumn     = "status"
	taskEnqueuedAtColumn = "enqueued_at"
)

type crawlTaskRepository struct {
	client persistence.Client
}

func NewCrawlTaskRepository(client persistence.Client) crawltask.CrawlTaskRepository {
	return &crawlTaskRepository{client: client}
}

func (c *crawlTaskRepository) Create(ctx context.Context, entity models.CrawlTask) (valueobjects.CrawlTaskID, error) {
	dbEntity := converters.SaveCrawlTaskToSnapshot(entity)

	builder := sq.Insert(taskTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(taskIDColumn, taskJobIDColumn, taskURLColumn, taskStatusColumn, taskEnqueuedAtColumn).
		Values(dbEntity.ID, dbEntity.JobID, dbEntity.URL, dbEntity.Status, dbEntity.EnqueuedAt).
		Suffix("RETURNING id")

	query, args, err := builder.ToSql()
	if err != nil {
		return valueobjects.CrawlTaskID{}, err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.Create",
		QueryRaw: query,
	}

	var id string
	err = c.client.DB().QueryRowContext(ctx, q, args...).Scan(&id)
	if err != nil {
		return valueobjects.CrawlTaskID{}, err
	}

	return valueobjects.NewCrawlTaskID(id)
}

func (c *crawlTaskRepository) BulkCreate(ctx context.Context, entities []models.CrawlTask) error {
	if len(entities) == 0 {
		return nil
	}

	builder := sq.Insert(taskTableName).
		PlaceholderFormat(sq.Dollar).
		Columns(taskIDColumn, taskJobIDColumn, taskURLColumn, taskStatusColumn, taskEnqueuedAtColumn)

	for _, entity := range entities {
		dbEntity := converters.SaveCrawlTaskToSnapshot(entity)
		builder = builder.Values(dbEntity.ID, dbEntity.JobID, dbEntity.URL, dbEntity.Status, dbEntity.EnqueuedAt)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.BulkCreate",
		QueryRaw: query,
	}

	_, err = c.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (c *crawlTaskRepository) Get(ctx context.Context, id valueobjects.CrawlTaskID) (*models.CrawlTask, error) {
	builder := sq.Select(
		"t.id", "t.job_id", "t.url", "t.status", "t.enqueued_at",
		"j.id", "j.name", "j.status", "j.created_at", "j.completed_at",
	).
		PlaceholderFormat(sq.Dollar).
		From(taskTableName + " t").
		InnerJoin("crawl_jobs j ON t.job_id = j.id").
		Where(sq.Eq{"t.id": id.String()}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.Get",
		QueryRaw: query,
	}

	var taskSnapshot snapshots.CrawlTaskSnapshot
	var jobSnapshot snapshots.CrawlJobSnapshot

	err = c.client.DB().QueryRowContext(ctx, q, args...).Scan(
		&taskSnapshot.ID,
		&taskSnapshot.JobID,
		&taskSnapshot.URL,
		&taskSnapshot.Status,
		&taskSnapshot.EnqueuedAt,
		&jobSnapshot.ID,
		&jobSnapshot.Name,
		&jobSnapshot.Status,
		&jobSnapshot.CreatedAt,
		&jobSnapshot.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	taskWithJob := snapshots.CrawlTaskWithJobSnapshot{
		CrawlTaskSnapshot: taskSnapshot,
		Job:               &jobSnapshot,
	}

	return converters.RestoreCrawlTaskWithJobFromSnapshot(taskWithJob)
}

func (c *crawlTaskRepository) Update(ctx context.Context, entity models.CrawlTask) error {
	dbEntity := converters.SaveCrawlTaskToSnapshot(entity)

	builder := sq.Update(taskTableName).
		PlaceholderFormat(sq.Dollar).
		Set(taskURLColumn, dbEntity.URL).
		Set(taskStatusColumn, dbEntity.Status).
		Where(sq.Eq{taskIDColumn: dbEntity.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.Update",
		QueryRaw: query,
	}

	_, err = c.client.DB().ExecContext(ctx, q, args...)
	return err
}

func (c *crawlTaskRepository) ListByJob(ctx context.Context, jobID valueobjects.CrawlJobID) ([]*models.CrawlTask, error) {
	builder := sq.Select(taskIDColumn, taskJobIDColumn, taskURLColumn, taskStatusColumn, taskEnqueuedAtColumn).
		PlaceholderFormat(sq.Dollar).
		From(taskTableName).
		Where(sq.Eq{taskJobIDColumn: jobID.String()}).
		OrderBy(taskEnqueuedAtColumn + " ASC")

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.ListByJob",
		QueryRaw: query,
	}

	var taskSnapshots []snapshots.CrawlTaskSnapshot
	err = c.client.DB().ScanAllContext(ctx, &taskSnapshots, q, args...)
	if err != nil {
		return nil, err
	}

	tasks := make([]*models.CrawlTask, 0, len(taskSnapshots))
	for _, snapshot := range taskSnapshots {
		task, err := converters.RestoreCrawlTaskFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (c *crawlTaskRepository) ListByStatus(ctx context.Context, status models.TaskStatus, limit int) ([]*models.CrawlTask, error) {
	builder := sq.Select(taskIDColumn, taskJobIDColumn, taskURLColumn, taskStatusColumn, taskEnqueuedAtColumn).
		PlaceholderFormat(sq.Dollar).
		From(taskTableName).
		Where(sq.Eq{taskStatusColumn: status.String()}).
		OrderBy(taskEnqueuedAtColumn + " ASC").
		Limit(uint64(limit))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.ListByStatus",
		QueryRaw: query,
	}

	var taskSnapshots []snapshots.CrawlTaskSnapshot
	err = c.client.DB().ScanAllContext(ctx, &taskSnapshots, q, args...)
	if err != nil {
		return nil, err
	}

	tasks := make([]*models.CrawlTask, 0, len(taskSnapshots))
	for _, snapshot := range taskSnapshots {
		task, err := converters.RestoreCrawlTaskFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

