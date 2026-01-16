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
