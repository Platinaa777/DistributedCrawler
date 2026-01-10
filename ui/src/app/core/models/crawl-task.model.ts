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
}
