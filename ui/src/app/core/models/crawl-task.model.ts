import { CrawlJob } from './crawl-job.model';

export interface CrawlTask {
  id: string;
  job_id: string;
  job?: CrawlJob;
  url: string;
  final_url?: string;
  status: string;
  enqueued_at: string;
  depth: number;
  body_hash: string;
  minio_object_key: string;
  result_object_key?: string;
  error_message?: string;
}

export type FileType = 'pages' | 'result';

export interface TaskFileURLResponse {
  url: string;
  expires_in_seconds: number;
}
