import { ExtractionSpec } from './extraction-spec.model';

export interface CrawlJob {
  id: string;
  job_config_id: string;
  job_config?: CrawlJobConfig;
  status: string;
  created_at: string;
  completed_at?: string;
  export_json_key?: string;
  export_csv_key?: string;
  exported_at?: string;
  export_status?: string;
}

// JobType determines whether a crawl job runs once or on a schedule
export type JobType = 'JOB_TYPE_ONCE' | 'JOB_TYPE_SCHEDULED';

// CrawlMode controls which link-following strategy the crawler uses
export type CrawlMode = 'CRAWL_MODE_UNSPECIFIED' | 'CRAWL_MODE_PAGINATION_AND_LINKS' | 'CRAWL_MODE_PAGINATION_ONLY' | 'CRAWL_MODE_LINKS_ONLY';

export const CRAWL_MODES: { value: CrawlMode; label: string; description: string }[] = [
  { value: 'CRAWL_MODE_PAGINATION_AND_LINKS', label: 'Pagination & Links', description: 'Follow both pagination and regular links (default)' },
  { value: 'CRAWL_MODE_PAGINATION_ONLY', label: 'Pagination Only', description: 'Follow only pagination links' },
  { value: 'CRAWL_MODE_LINKS_ONLY', label: 'Links Only', description: 'Follow only regular <a href> links' }
];

export const JOB_TYPES: { value: JobType; label: string; description: string }[] = [
  { value: 'JOB_TYPE_ONCE', label: 'One-time', description: 'Run exactly once' },
  { value: 'JOB_TYPE_SCHEDULED', label: 'Scheduled', description: 'Run on a recurring schedule' }
];

export interface CrawlJobConfig {
  id?: string;
  name: string;
  extraction_spec: ExtractionSpec;
  scopes: ScopeRules;
  seeds: Seed[];
  rate_limit: RateLimitPolicy;
  retries?: RetryPolicy;
  auth?: AuthOptions;
  schedule?: ScheduleOptions;
  // Determines if this is a one-time job or a scheduled recurring job
  job_type?: JobType;
  // Controls link-following behavior
  crawl_mode?: CrawlMode;
  // If true, crawler follows robots.txt rules; if false, robots.txt is ignored
  respect_robots_txt?: boolean;
}

export interface ScopeRules {
  max_depth: number;
  allowed_domains: string[];
  deny_url_patterns?: string[];
}

export interface Seed {
  url: string;
}

export interface RateLimitPolicy {
  rps: number;
}

export interface RetryPolicy {
  max_attempts: number;
  backoff_initial_ms: number;
  backoff_multiplier: number;
}

export interface AuthOptions {
  cookie?: string;
  basic_user?: string;
  basic_password?: string;
  bearer_token?: string;
}

export interface ScheduleOptions {
  cron?: string;
}

export type JobExportFileType = 'json' | 'csv';

export interface JobExportFileURLResponse {
  url: string;
  expires_in_seconds: number;
}

// Job status constants
export type JobStatus = 'InProgress' | 'Parsed' | 'Completed' | 'Failed' | 'Skipped';

export const JOB_STATUSES: JobStatus[] = [
  'InProgress',
  'Parsed',
  'Completed',
  'Failed',
  'Skipped'
];
