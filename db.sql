// Use DBML to define your simplified database structure

//////////////////////////////////////////////////////
// Enums
//////////////////////////////////////////////////////

Enum task_status {
  Pending
  Running
  Completed
  Failed
}

//////////////////////////////////////////////////////
// Core entities
//////////////////////////////////////////////////////

Table crawl_job {
  id uuid [pk]
  name varchar [not null]
  status task_status [not null]
  created_at timestamp [not null]
  completed_at timestamp                      // nullable
}

Table crawl_task {
  id uuid [pk]
  job_id uuid [not null]                      // -> crawl_job.id
  url text [not null]
  status task_status [not null]
  enqueued_at timestamp [not null]
}

Table page_snapshot {
  id uuid [pk]
  task_id uuid [not null]                     // -> crawl_task.id
  url text [not null]
  http_status int [not null]
  content_type varchar
  storage_key varchar [not null]
  fetched_at timestamp [not null]
}

Table extracted_record {
  id uuid [pk]
  task_id uuid [not null]                     // -> crawl_task.id
  source_url text [not null]
  data json [not null]
  parsed_at timestamp [not null]
}

//////////////////////////////////////////////////////
// Relationships
//////////////////////////////////////////////////////

Ref: crawl_task.job_id > crawl_job.id
Ref: page_snapshot.task_id > crawl_task.id
Ref: extracted_record.task_id > crawl_task.id
