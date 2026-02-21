package repos

import (
	"context"

	sq "github.com/Masterminds/squirrel"

	"distributed-crawler/internal/application/service"
	"distributed-crawler/internal/domain/crawl/models"
	crawltask "distributed-crawler/internal/domain/crawl/repos/crawl_task"
	"distributed-crawler/internal/domain/crawl/valueobjects"
	"distributed-crawler/internal/infra/persistence"
	"distributed-crawler/internal/infra/persistence/postgres/converters"
	"distributed-crawler/internal/infra/persistence/postgres/snapshots"
)

const (
	taskTableName = "crawl_tasks"

	taskIDColumn             = "id"
	taskJobIDColumn          = "job_id"
	taskURLColumn            = "url"
	taskFinalURLColumn       = "final_url"
	taskStatusColumn         = "status"
	taskEnqueuedAtColumn     = "enqueued_at"
	taskDepthColumn          = "depth"
	taskMinioObjectKeyColumn = "minio_object_key"

	// Result persistence columns
	taskResultObjectKeyColumn   = "result_object_key"
	taskResultContentTypeColumn = "result_content_type"
	taskResultSizeBytesColumn   = "result_size_bytes"
	taskResultCreatedAtColumn   = "result_created_at"

	// Error message column
	taskErrorMessageColumn = "error_message"
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
		Columns(
			taskIDColumn, taskJobIDColumn, taskURLColumn, taskFinalURLColumn, taskStatusColumn, taskEnqueuedAtColumn,
			taskDepthColumn, taskMinioObjectKeyColumn,
			taskResultObjectKeyColumn, taskResultContentTypeColumn, taskResultSizeBytesColumn, taskResultCreatedAtColumn,
			taskErrorMessageColumn,
		).
		Values(
			dbEntity.ID, dbEntity.JobID, dbEntity.URL, dbEntity.FinalURL, dbEntity.Status, dbEntity.EnqueuedAt,
			dbEntity.Depth, dbEntity.MinioObjectKey,
			dbEntity.ResultObjectKey, dbEntity.ResultContentType, dbEntity.ResultSizeBytes, dbEntity.ResultCreatedAt,
			dbEntity.ErrorMessage,
		).
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
		Columns(
			taskIDColumn, taskJobIDColumn, taskURLColumn, taskFinalURLColumn, taskStatusColumn, taskEnqueuedAtColumn,
			taskDepthColumn, taskMinioObjectKeyColumn,
			taskResultObjectKeyColumn, taskResultContentTypeColumn, taskResultSizeBytesColumn, taskResultCreatedAtColumn,
			taskErrorMessageColumn,
		)

	for _, entity := range entities {
		dbEntity := converters.SaveCrawlTaskToSnapshot(entity)
		builder = builder.Values(
			dbEntity.ID, dbEntity.JobID, dbEntity.URL, dbEntity.FinalURL, dbEntity.Status, dbEntity.EnqueuedAt,
			dbEntity.Depth, dbEntity.MinioObjectKey,
			dbEntity.ResultObjectKey, dbEntity.ResultContentType, dbEntity.ResultSizeBytes, dbEntity.ResultCreatedAt,
			dbEntity.ErrorMessage,
		)
	}

	query, args, err := builder.
		Suffix("ON CONFLICT (job_id, url) DO NOTHING").
		ToSql()
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
		"t.id", "t.job_id", "t.url", "t.final_url", "t.status", "t.enqueued_at", "t.depth", "t.minio_object_key",
		"t.result_object_key", "t.result_content_type", "t.result_size_bytes", "t.result_created_at", "t.error_message",
		"j.id", "j.job_config_id", "j.status", "j.created_at", "j.completed_at",
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
		&taskSnapshot.FinalURL,
		&taskSnapshot.Status,
		&taskSnapshot.EnqueuedAt,
		&taskSnapshot.Depth,
		&taskSnapshot.MinioObjectKey,
		&taskSnapshot.ResultObjectKey,
		&taskSnapshot.ResultContentType,
		&taskSnapshot.ResultSizeBytes,
		&taskSnapshot.ResultCreatedAt,
		&taskSnapshot.ErrorMessage,
		&jobSnapshot.ID,
		&jobSnapshot.JobConfigID,
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
		Set(taskFinalURLColumn, dbEntity.FinalURL).
		Set(taskStatusColumn, dbEntity.Status).
		Set(taskDepthColumn, dbEntity.Depth).
		Set(taskMinioObjectKeyColumn, dbEntity.MinioObjectKey).
		Set(taskResultObjectKeyColumn, dbEntity.ResultObjectKey).
		Set(taskResultContentTypeColumn, dbEntity.ResultContentType).
		Set(taskResultSizeBytesColumn, dbEntity.ResultSizeBytes).
		Set(taskResultCreatedAtColumn, dbEntity.ResultCreatedAt).
		Set(taskErrorMessageColumn, dbEntity.ErrorMessage).
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
	builder := sq.Select(
		taskIDColumn, taskJobIDColumn, taskURLColumn, taskFinalURLColumn, taskStatusColumn, taskEnqueuedAtColumn,
		taskDepthColumn, taskMinioObjectKeyColumn,
		taskResultObjectKeyColumn, taskResultContentTypeColumn, taskResultSizeBytesColumn, taskResultCreatedAtColumn,
		taskErrorMessageColumn,
	).
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
	builder := sq.Select(
		taskIDColumn, taskJobIDColumn, taskURLColumn, taskFinalURLColumn, taskStatusColumn, taskEnqueuedAtColumn,
		taskDepthColumn, taskMinioObjectKeyColumn,
		taskResultObjectKeyColumn, taskResultContentTypeColumn, taskResultSizeBytesColumn, taskResultCreatedAtColumn,
		taskErrorMessageColumn,
	).
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

// SetTaskResult updates the result fields for a task (Part A - ParserWorker persistence)
func (c *crawlTaskRepository) SetTaskResult(ctx context.Context, taskID valueobjects.CrawlTaskID, objectKey string, contentType string, sizeBytes int64) error {
	builder := sq.Update(taskTableName).
		PlaceholderFormat(sq.Dollar).
		Set(taskResultObjectKeyColumn, objectKey).
		Set(taskResultContentTypeColumn, contentType).
		Set(taskResultSizeBytesColumn, sizeBytes).
		Set(taskResultCreatedAtColumn, "NOW()").
		Where(sq.Eq{taskIDColumn: taskID.String()})

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.SetTaskResult",
		QueryRaw: query,
	}

	_, err = c.client.DB().ExecContext(ctx, q, args...)
	return err
}

// ExistsByJobIDAndURL checks if a task with the given URL already exists for the job (URL deduplication)
func (c *crawlTaskRepository) ExistsByJobIDAndURL(ctx context.Context, jobID valueobjects.CrawlJobID, url string) (bool, error) {
	builder := sq.Select("1").
		PlaceholderFormat(sq.Dollar).
		From(taskTableName).
		Where(sq.And{
			sq.Eq{taskJobIDColumn: jobID.String()},
			sq.Eq{taskURLColumn: url},
			sq.NotEq{taskStatusColumn: models.TaskStatusInProgress.String() },
		}).
		Limit(1)

	query, args, err := builder.ToSql()
	if err != nil {
		return false, err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.ExistsByJobIDAndURL",
		QueryRaw: query,
	}

	var exists int
	err = c.client.DB().QueryRowContext(ctx, q, args...).Scan(&exists)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ListWithCursor returns tasks with cursor-based pagination and filtering
func (c *crawlTaskRepository) ListWithCursor(ctx context.Context, query service.ListTasksByJobQuery) (*service.ListTasksResult, error) {
	// Set defaults
	effectiveLimit := query.Limit
	if effectiveLimit == 0 {
		effectiveLimit = 20
	}
	if effectiveLimit > 100 {
		effectiveLimit = 100
	}

	// Fetch one extra row to determine if there are more results
	fetchLimit := effectiveLimit + 1

	builder := sq.Select(
		taskIDColumn, taskJobIDColumn, taskURLColumn, taskFinalURLColumn, taskStatusColumn, taskEnqueuedAtColumn,
		taskDepthColumn, taskMinioObjectKeyColumn,
		taskResultObjectKeyColumn, taskResultContentTypeColumn, taskResultSizeBytesColumn, taskResultCreatedAtColumn,
		taskErrorMessageColumn,
	).
		PlaceholderFormat(sq.Dollar).
		From(taskTableName)

	// Build WHERE conditions
	conditions := sq.And{
		sq.Eq{taskJobIDColumn: query.JobID},
	}

	// Filter by status (exact match)
	if query.Filter.Status != nil && *query.Filter.Status != "" {
		conditions = append(conditions, sq.Eq{taskStatusColumn: *query.Filter.Status})
	}

	// Filter by URL (partial match, case-insensitive)
	if query.Filter.URL != nil && *query.Filter.URL != "" {
		conditions = append(conditions, sq.ILike{taskURLColumn: "%" + *query.Filter.URL + "%"})
	}

	// Filter by depth range
	if query.Filter.MinDepth != nil {
		conditions = append(conditions, sq.GtOrEq{taskDepthColumn: *query.Filter.MinDepth})
	}
	if query.Filter.MaxDepth != nil {
		conditions = append(conditions, sq.LtOrEq{taskDepthColumn: *query.Filter.MaxDepth})
	}

	// Filter by enqueued_at range
	if query.Filter.EnqueuedFrom != nil {
		conditions = append(conditions, sq.GtOrEq{taskEnqueuedAtColumn: *query.Filter.EnqueuedFrom})
	}
	if query.Filter.EnqueuedTo != nil {
		conditions = append(conditions, sq.LtOrEq{taskEnqueuedAtColumn: *query.Filter.EnqueuedTo})
	}

	// Apply cursor condition (seek method for keyset pagination)
	// For ASC ordering: WHERE (enqueued_at, id) > (cursor.enqueued_at, cursor.id)
	if query.Cursor != nil {
		cursorCondition := sq.Expr(
			"("+taskEnqueuedAtColumn+", "+taskIDColumn+") > (?, ?)",
			query.Cursor.EnqueuedAt,
			query.Cursor.ID,
		)
		conditions = append(conditions, cursorCondition)
	}

	builder = builder.Where(conditions)

	// Order by enqueued_at ASC, id ASC for stable ordering
	builder = builder.OrderBy(taskEnqueuedAtColumn+" ASC", taskIDColumn+" ASC")
	builder = builder.Limit(uint64(fetchLimit))

	sqlQuery, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	q := persistence.Query{
		Name:     "crawl_task_repository.ListWithCursor",
		QueryRaw: sqlQuery,
	}

	var taskSnapshots []snapshots.CrawlTaskSnapshot
	err = c.client.DB().ScanAllContext(ctx, &taskSnapshots, q, args...)
	if err != nil {
		return nil, err
	}

	// Convert snapshots to models and trim to effective limit
	tasks := make([]*models.CrawlTask, 0, effectiveLimit)
	for _, snapshot := range taskSnapshots {
		if len(tasks) >= effectiveLimit {
			break
		}
		task, err := converters.RestoreCrawlTaskFromSnapshot(snapshot)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	// Determine if there are more results
	hasMore := len(taskSnapshots) > effectiveLimit

	// Build next cursor from last item
	var nextCursor *service.ListTasksCursor
	if hasMore && len(tasks) > 0 {
		lastTask := tasks[len(tasks)-1]
		nextCursor = &service.ListTasksCursor{
			EnqueuedAt: lastTask.EnqueuedAt,
			ID:         lastTask.ID.String(),
		}
	}

	return &service.ListTasksResult{
		Tasks:      tasks,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// GetAnalyticsByJob returns aggregated analytics for a job
func (c *crawlTaskRepository) GetAnalyticsByJob(ctx context.Context, jobID valueobjects.CrawlJobID) (*service.TaskAnalytics, error) {
	// Query for status counts
	statusBuilder := sq.Select(taskStatusColumn, "COUNT(*) as count").
		PlaceholderFormat(sq.Dollar).
		From(taskTableName).
		Where(sq.Eq{taskJobIDColumn: jobID.String()}).
		GroupBy(taskStatusColumn)

	statusQuery, statusArgs, err := statusBuilder.ToSql()
	if err != nil {
		return nil, err
	}

	statusQ := persistence.Query{
		Name:     "crawl_task_repository.GetAnalyticsByJob.status",
		QueryRaw: statusQuery,
	}

	rows, err := c.client.DB().QueryContext(ctx, statusQ, statusArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusCounts := make(map[string]int64)
	var totalCount int64
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		statusCounts[status] = count
		totalCount += count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Query for depth counts
	depthBuilder := sq.Select(taskDepthColumn, "COUNT(*) as count").
		PlaceholderFormat(sq.Dollar).
		From(taskTableName).
		Where(sq.Eq{taskJobIDColumn: jobID.String()}).
		GroupBy(taskDepthColumn).
		OrderBy(taskDepthColumn + " ASC")

	depthQuery, depthArgs, err := depthBuilder.ToSql()
	if err != nil {
		return nil, err
	}

	depthQ := persistence.Query{
		Name:     "crawl_task_repository.GetAnalyticsByJob.depth",
		QueryRaw: depthQuery,
	}

	depthRows, err := c.client.DB().QueryContext(ctx, depthQ, depthArgs...)
	if err != nil {
		return nil, err
	}
	defer depthRows.Close()

	depthCounts := make(map[uint64]int64)
	for depthRows.Next() {
		var depth uint64
		var count int64
		if err := depthRows.Scan(&depth, &count); err != nil {
			return nil, err
		}
		depthCounts[depth] = count
	}

	if err := depthRows.Err(); err != nil {
		return nil, err
	}

	return &service.TaskAnalytics{
		StatusCounts: statusCounts,
		DepthCounts:  depthCounts,
		TotalCount:   totalCount,
	}, nil
}
