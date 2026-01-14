You are working on a distributed crawler system with MinIO used as an object storage.

Context

There are two MinIO buckets:

pages — stores downloaded HTML pages

result — stores JSON files produced by the ParserWorker

Each crawl task may have:

1 HTML file (if the page was downloaded)

1 JSON result file (if the parser worker processed it)

The backend uses Go and follows a domain-driven structure.

Domain Model (relevant fields)
type CrawlTask struct {
	ID valueobjects.CrawlTaskID

	JobID valueobjects.CrawlJobID
	Job   *CrawlJob

	URL        string
	FinalURL   *string
	Status     TaskStatus
	EnqueuedAt time.Time

	Depth          uint64
	BodyHash       *string
	MinioObjectKey string // HTML object key (bucket: pages)

	// Result persistence fields (ParserWorker)
	ResultObjectKey   *string    // JSON object key (bucket: result)
	ResultContentType *string
	ResultSizeBytes   *int64
	ResultCreatedAt   *time.Time
}

Goal

Implement a feature that allows the UI to request read-access links (URLs) to files stored in MinIO:

HTML page from the pages bucket

JSON result from the result bucket

These links should be temporary pre-signed URLs, suitable for direct browser access.

Backend Requirements

Create a new HTTP handler that:

Accepts:

task_id

file_type (pages or result)

Validates:

Task exists

Requested object key exists for the task

Generates a pre-signed GET URL from MinIO

Returns the URL in JSON

Bucket mapping

pages → use CrawlTask.MinioObjectKey

result → use CrawlTask.ResultObjectKey

Behavior

If the requested file does not exist → return 404

URL expiration should be limited (e.g. 5–15 minutes)

No file proxying — only generate MinIO signed URLs

Security

Validate that the task belongs to the requested job (if job context is provided)

Do not expose raw MinIO credentials

Frontend Requirements (Angular)

File: job-details.component.ts

In the tasks table, add:

Pages button

Enabled only if MinioObjectKey exists

Result button

Enabled only if ResultObjectKey exists

On button click:

Call the backend handler

Receive the signed URL

Open it in a new browser tab

UX:

Disable buttons if the file is not available

Handle loading & error states gracefully

Deliverables

Backend:

Handler implementation

MinIO presigned URL generation logic

Proper error handling

Frontend:

UI buttons

API integration

Opening files via signed URLs

Constraints

Do NOT download or stream files through backend

Use MinIO pre-signed URLs only

Keep changes minimal and consistent with existing architecture