import { ExtractionSpec } from './extraction-spec.model';

export interface CrawlJob {
  id: string;
  job_config_id: string;
  job_config?: CrawlJobConfig;
  status: string;
  created_at: string;
  completed_at?: string;
  error?: string;
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
