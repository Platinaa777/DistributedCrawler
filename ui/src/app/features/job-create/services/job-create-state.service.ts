import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { CrawlJobConfig, Seed, ScopeRules, RateLimitPolicy } from '../../../core/models/crawl-job.model';
import { ExtractionSpec, FieldSpec, MetricSpec } from '../../../core/models/extraction-spec.model';

export interface JobCreateState {
  // Step A - Preview
  previewUrl: string;
  previewId: string | null;
  previewHtml: string | null;

  // Step C - ExtractionSpec
  extractionSpec: ExtractionSpec;

  // Step D - Job Settings
  jobName: string;
  seeds: Seed[];
  scopeRules: ScopeRules;
  rateLimit: RateLimitPolicy;
}

const initialState: JobCreateState = {
  previewUrl: '',
  previewId: null,
  previewHtml: null,
  extractionSpec: {
    fields: [],
    metrics: []
  },
  jobName: '',
  seeds: [],
  scopeRules: {
    max_depth: 2,
    allowed_domains: []
  },
  rateLimit: {
    rps: 1
  }
};

@Injectable({
  providedIn: 'root'
})
export class JobCreateStateService {
  private state$ = new BehaviorSubject<JobCreateState>({ ...initialState });

  constructor() {}

  getState(): Observable<JobCreateState> {
    return this.state$.asObservable();
  }

  getCurrentState(): JobCreateState {
    return this.state$.value;
  }

  // Step A - Preview
  setPreview(url: string, previewId: string, html: string): void {
    this.state$.next({
      ...this.state$.value,
      previewUrl: url,
      previewId,
      previewHtml: html
    });
  }

  // Step C - ExtractionSpec
  setExtractionSpec(spec: ExtractionSpec): void {
    this.state$.next({
      ...this.state$.value,
      extractionSpec: spec
    });
  }

  addField(field: FieldSpec): void {
    const current = this.state$.value;
    this.state$.next({
      ...current,
      extractionSpec: {
        ...current.extractionSpec,
        fields: [...current.extractionSpec.fields, field]
      }
    });
  }

  updateField(index: number, field: FieldSpec): void {
    const current = this.state$.value;
    const fields = [...current.extractionSpec.fields];
    fields[index] = field;
    this.state$.next({
      ...current,
      extractionSpec: {
        ...current.extractionSpec,
        fields
      }
    });
  }

  removeField(index: number): void {
    const current = this.state$.value;
    this.state$.next({
      ...current,
      extractionSpec: {
        ...current.extractionSpec,
        fields: current.extractionSpec.fields.filter((_, i) => i !== index)
      }
    });
  }

  addMetric(metric: MetricSpec): void {
    const current = this.state$.value;
    this.state$.next({
      ...current,
      extractionSpec: {
        ...current.extractionSpec,
        metrics: [...current.extractionSpec.metrics, metric]
      }
    });
  }

  removeMetric(index: number): void {
    const current = this.state$.value;
    this.state$.next({
      ...current,
      extractionSpec: {
        ...current.extractionSpec,
        metrics: current.extractionSpec.metrics.filter((_, i) => i !== index)
      }
    });
  }

  // Step D - Job Settings
  setJobSettings(
    name: string,
    seeds: Seed[],
    scopeRules: ScopeRules,
    rateLimit: RateLimitPolicy
  ): void {
    this.state$.next({
      ...this.state$.value,
      jobName: name,
      seeds,
      scopeRules,
      rateLimit
    });
  }

  // Build final CrawlJobConfig
  buildJobConfig(): CrawlJobConfig {
    const state = this.state$.value;
    return {
      name: state.jobName,
      extraction_spec: state.extractionSpec,
      scopes: state.scopeRules,
      seeds: state.seeds,
      rate_limit: state.rateLimit
    };
  }

  // Reset wizard
  reset(): void {
    this.state$.next({ ...initialState });
  }
}
