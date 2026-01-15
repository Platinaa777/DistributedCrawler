import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormArray, FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDividerModule } from '@angular/material/divider';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatSelectModule } from '@angular/material/select';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { CrawlJobConfig, RetryPolicy, ScheduleOptions } from '../../core/models/crawl-job.model';
import { FieldSpec, MetricSpec, PaginationSpec, TransformSpec } from '../../core/models/extraction-spec.model';

interface SimpleJobFormValue {
  name: string;
  seeds: { url: string }[];
  allowed_domains: string[];
  deny_url_patterns: string[];
  max_depth: number;
  rps: number;
  retries: RetryPolicy;
  auth: {
    basic_user?: string;
    basic_password?: string;
    bearer_token?: string;
    cookie?: string;
  };
  schedule: ScheduleOptions;
}

@Component({
  selector: 'app-simple-job-create',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    MatDividerModule,
    MatCheckboxModule,
    MatSnackBarModule,
    MatSelectModule
  ],
  template: `
    <form class="container mx-auto p-6 space-y-4" [formGroup]="jobForm">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <button mat-button type="button" (click)="goBack()">
            <mat-icon>arrow_back</mat-icon>
            Back to Jobs
          </button>
          <h1 class="text-2xl font-semibold">Simple Crawl Job</h1>
        </div>
        <div class="flex items-center gap-2">
          <button
            mat-raised-button
            color="primary"
            type="button"
            (click)="submit()"
            [disabled]="creating || !canSubmit()"
          >
            <mat-icon>play_arrow</mat-icon>
            Create Job
          </button>
        </div>
      </div>

      <mat-card>
        <mat-card-header>
          <mat-card-title>Job Basics</mat-card-title>
          <mat-card-subtitle>Minimal settings to register a crawl job</mat-card-subtitle>
        </mat-card-header>
        <mat-card-content>
          <mat-form-field class="w-full" [matTooltip]="'Human-friendly job name'">
            <mat-label>Name</mat-label>
            <input matInput formControlName="name" placeholder="Example crawl job" />
            <mat-icon matSuffix>info_outline</mat-icon>
            <mat-error *ngIf="jobForm.get('name')?.hasError('required')">
              Name is required
            </mat-error>
          </mat-form-field>
        </mat-card-content>
      </mat-card>

      <mat-card>
        <mat-card-header class="flex items-center justify-between">
          <div>
            <mat-card-title>Seeds & Scope</mat-card-title>
            <mat-card-subtitle>Where to start and what to allow</mat-card-subtitle>
          </div>
          <button
            mat-icon-button
            type="button"
            (click)="toggleSeedsScope()"
            [attr.aria-label]="seedsScopeExpanded ? 'Collapse Seeds & Scope' : 'Expand Seeds & Scope'"
          >
            <mat-icon>{{ seedsScopeExpanded ? 'expand_less' : 'expand_more' }}</mat-icon>
          </button>
        </mat-card-header>
        <mat-card-content class="space-y-4" *ngIf="seedsScopeExpanded">
          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Seed URLs</p>
              <button mat-stroked-button color="primary" type="button" (click)="addSeed()">
                <mat-icon>add</mat-icon>
                Add Seed
              </button>
            </div>
            <div formArrayName="seeds" class="space-y-2">
              <div
                *ngFor="let seed of seeds.controls; let i = index"
                [formGroupName]="i"
                class="flex items-center gap-2"
              >
                <mat-form-field
                 
                  class="flex-1"
                  [matTooltip]="'Starting URL for the crawler'"
                >
                  <mat-label>Seed {{ i + 1 }}</mat-label>
                  <input matInput formControlName="url" placeholder="https://example.com" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <button
                  mat-icon-button
                  color="warn"
                  type="button"
                  (click)="removeSeed(i)"
                  [disabled]="seeds.length === 1"
                  aria-label="Remove seed"
                >
                  <mat-icon>delete</mat-icon>
                </button>
              </div>
            </div>
          </div>

          <mat-divider></mat-divider>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <mat-form-field class="w-full" [matTooltip]="'How deep to follow links (0 = only seeds)'">
              <mat-label>Max Depth</mat-label>
              <input matInput type="number" formControlName="max_depth" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>

            <mat-form-field class="w-full" [matTooltip]="'Requests per second limit for this job'">
              <mat-label>RPS</mat-label>
              <input matInput type="number" formControlName="rps" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
          </div>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Allowed Domains</p>
              <button mat-stroked-button type="button" (click)="addAllowedDomain()">
                <mat-icon>add</mat-icon>
                Add
              </button>
            </div>
            <div formArrayName="allowed_domains" class="space-y-2">
              <div *ngFor="let domain of allowedDomains.controls; let i = index" class="flex items-center gap-2">
                <mat-form-field class="flex-1" [matTooltip]="'Only crawl URLs inside these domains'">
                  <mat-label>Domain {{ i + 1 }}</mat-label>
                  <input matInput [formControlName]="i" placeholder="example.com" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <button mat-icon-button color="warn" type="button" (click)="removeAllowedDomain(i)">
                  <mat-icon>delete</mat-icon>
                </button>
              </div>
            </div>
          </div>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Deny URL Patterns</p>
              <button mat-stroked-button type="button" (click)="addDenyPattern()">
                <mat-icon>add</mat-icon>
                Add
              </button>
            </div>
            <div formArrayName="deny_url_patterns" class="space-y-2">
              <div *ngFor="let pattern of denyPatterns.controls; let i = index" class="flex items-center gap-2">
                <mat-form-field class="flex-1" [matTooltip]="'Paths to skip during crawl'">
                  <mat-label>Pattern {{ i + 1 }}</mat-label>
                  <input matInput [formControlName]="i" placeholder="/login" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <button mat-icon-button color="warn" type="button" (click)="removeDenyPattern(i)">
                  <mat-icon>delete</mat-icon>
                </button>
              </div>
            </div>
          </div>
        </mat-card-content>
      </mat-card>
      <mat-card>
        <mat-card-header class="flex items-center justify-between">
          <mat-card-title>Rate Limit, Retries & Schedule</mat-card-title>
          <button
            mat-icon-button
            type="button"
            (click)="toggleRateLimit()"
            [attr.aria-label]="rateLimitExpanded ? 'Collapse Rate Limit, Retries & Schedule' : 'Expand Rate Limit, Retries & Schedule'"
          >
            <mat-icon>{{ rateLimitExpanded ? 'expand_less' : 'expand_more' }}</mat-icon>
          </button>
        </mat-card-header>
        <mat-card-content class="grid grid-cols-1 md:grid-cols-3 gap-4" *ngIf="rateLimitExpanded">
          <div [formGroup]="retriesGroup" class="space-y-4">
            <p class="text-sm font-semibold">Retry Policy</p>
            <mat-form-field class="w-full" [matTooltip]="'Maximum retry attempts'">
              <mat-label>Max Attempts</mat-label>
              <input matInput type="number" formControlName="max_attempts" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
            <mat-form-field class="w-full" [matTooltip]="'First retry backoff in ms'">
              <mat-label>Backoff Initial (ms)</mat-label>
              <input matInput type="number" formControlName="backoff_initial_ms" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
            <mat-form-field class="w-full" [matTooltip]="'Multiplier applied per retry attempt'">
              <mat-label>Backoff Multiplier</mat-label>
              <input matInput type="number" formControlName="backoff_multiplier" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
          </div>

          <div [formGroup]="scheduleGroup" class="space-y-4">
            <p class="text-sm font-semibold">Schedule</p>
            <mat-form-field class="w-full" [matTooltip]="'Cron expression for periodic runs (optional)'">
              <mat-label>Cron</mat-label>
              <input matInput formControlName="cron" placeholder="0 9 * * 1" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
          </div>

          <div [formGroup]="authGroup" class="space-y-4">
            <p class="text-sm font-semibold">Auth (optional)</p>
            <mat-form-field class="w-full" [matTooltip]="'Basic auth username'">
              <mat-label>Basic User</mat-label>
              <input matInput formControlName="basic_user" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
            <mat-form-field class="w-full" [matTooltip]="'Basic auth password'">
              <mat-label>Basic Password</mat-label>
              <input matInput type="password" formControlName="basic_password" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
            <mat-form-field class="w-full" [matTooltip]="'Bearer token for Authorization header'">
              <mat-label>Bearer Token</mat-label>
              <input matInput formControlName="bearer_token" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
            <mat-form-field class="w-full" [matTooltip]="'Cookie header value'">
              <mat-label>Cookie</mat-label>
              <input matInput formControlName="cookie" />
              <mat-icon matSuffix>info_outline</mat-icon>
            </mat-form-field>
          </div>
        </mat-card-content>
      </mat-card>
      <mat-card>
        <mat-card-header class="flex items-center justify-between">
          <div>
            <mat-card-title>Extraction Spec</mat-card-title>
            <mat-card-subtitle>Fields and metrics that mirror the backend payload</mat-card-subtitle>
          </div>
          <button
            mat-icon-button
            type="button"
            (click)="toggleExtraction()"
            [attr.aria-label]="extractionExpanded ? 'Collapse Extraction Spec' : 'Expand Extraction Spec'"
          >
            <mat-icon>{{ extractionExpanded ? 'expand_less' : 'expand_more' }}</mat-icon>
          </button>
        </mat-card-header>
        <mat-card-content class="space-y-4" *ngIf="extractionExpanded">
          <div class="flex items-center justify-between">
            <p class="text-sm font-semibold">Fields</p>
            <button mat-stroked-button color="primary" type="button" (click)="addExtractionField()">
              <mat-icon>add</mat-icon>
              Add Field
            </button>
          </div>

          <div formArrayName="extraction_fields" class="space-y-4">
            <div
              *ngFor="let field of extractionFields.controls; let i = index"
              [formGroupName]="i"
              class="border rounded p-4 space-y-3"
            >
              <div class="flex items-center justify-between">
                <div class="font-semibold">Field #{{ i + 1 }}</div>
                <button mat-icon-button color="warn" type="button" (click)="removeExtractionField(i)">
                  <mat-icon>delete</mat-icon>
                </button>
              </div>

              <div class="grid grid-cols-1 gap-3">
                <mat-form-field class="w-full" [matTooltip]="'Key saved to the output JSON and referenced by metrics'">
                  <mat-label>Name</mat-label>
                  <input matInput formControlName="name" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <mat-form-field class="w-full" [matTooltip]="'Target type enforced by the parser (string/int/float/bool/url/json)'">
                  <mat-label>Type</mat-label>
                  <mat-select formControlName="type">
                    <mat-option *ngFor="let option of fieldTypeOptions" [value]="option">{{ option }}</mat-option>
                  </mat-select>
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <mat-form-field class="w-full" [matTooltip]="'Attribute to return (text/html/href/src/content). href/src are resolved against the page URL.'">
                  <mat-label>Attribute</mat-label>
                  <mat-select formControlName="attribute">
                    <mat-option *ngFor="let option of attributeOptions" [value]="option">{{ option }}</mat-option>
                  </mat-select>
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
              </div>

              <div class="grid grid-cols-1 gap-3 items-center">
                <mat-form-field class="w-full" [matTooltip]="'Selector or pattern used for extraction (CSS rule, regex body, meta name, etc.)'">
                  <mat-label>Selector</mat-label>
                  <input matInput formControlName="selector" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-3 gap-3 items-center">
                <mat-form-field
                  class="w-full"
                  [matTooltip]="'Index is used only when Multiple is enabled; pick a specific match (0-based, negative from the end)'"
                >
                  <mat-label>Index</mat-label>
                  <input
                    matInput
                    type="number"
                    formControlName="index"
                    [disabled]="!field.get('multiple')?.value"
                    [readonly]="!field.get('multiple')?.value"
                    [ngClass]="{
                      'bg-gray-100 text-gray-500 cursor-not-allowed': !field.get('multiple')?.value
                    }"
                  />
                  <mat-icon matSuffix>info_outline</mat-icon>
                  <mat-hint>Works only with Multiple: true</mat-hint>
                </mat-form-field>
                <mat-checkbox
                  formControlName="multiple"
                  [matTooltip]="'Return all matches as an array instead of a single value'"
                  class="mt-2"
                  (change)="handleMultipleToggle(i)"
                >
                  Multiple
                </mat-checkbox>
                <mat-checkbox formControlName="required" [matTooltip]="'If true, missing extraction is logged as a warning but does not stop the job'" class="mt-2">
                  Required
                </mat-checkbox>
              </div>

              <div formArrayName="transforms" class="space-y-2">
                <div class="flex items-center justify-between">
                  <p class="text-sm font-semibold">Transforms</p>
                  <button mat-stroked-button type="button" (click)="addTransform(i)">
                    <mat-icon>add</mat-icon>
                    Add Transform
                  </button>
                </div>
                <div
                  *ngFor="let transform of getTransforms(i).controls; let tIdx = index"
                  [formGroupName]="tIdx"
                  class="grid grid-cols-1 md:grid-cols-2 gap-2 items-center"
                >
                  <mat-form-field class="w-full" [matTooltip]="'Transform operation, e.g., trim, lower, limit'">
                    <mat-label>Op</mat-label>
                    <mat-select formControlName="op">
                      <mat-option *ngFor="let option of transformOpOptions" [value]="option">{{ option }}</mat-option>
                    </mat-select>
                    <mat-icon matSuffix>info_outline</mat-icon>
                  </mat-form-field>
                  <div class="flex items-center gap-2">
                    <mat-form-field class="flex-1" [matTooltip]="'Optional transform argument'">
                      <mat-label>Arg</mat-label>
                      <input matInput formControlName="arg" />
                      <mat-icon matSuffix>info_outline</mat-icon>
                    </mat-form-field>
                    <button mat-icon-button color="warn" type="button" (click)="removeTransform(i, tIdx)">
                      <mat-icon>delete</mat-icon>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <mat-divider></mat-divider>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Metrics</p>
              <button mat-stroked-button color="primary" type="button" (click)="addMetric()">
                <mat-icon>add</mat-icon>
                Add Metric
              </button>
            </div>
            <div formArrayName="metrics" class="space-y-3">
              <div
                *ngFor="let metric of metrics.controls; let m = index"
                [formGroupName]="m"
                class="border rounded p-3 grid grid-cols-1 md:grid-cols-3 gap-3 items-start"
              >
                <mat-form-field class="w-full" [matTooltip]="'Metric key saved to the output JSON'">
                  <mat-label>Name</mat-label>
                  <input matInput formControlName="name" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <mat-form-field class="w-full" [matTooltip]="'Parser operations: len (string length), count (array size), word_count (split by whitespace), field_present (bool), count_external_links'">
                  <mat-label>Op</mat-label>
                  <mat-select formControlName="op">
                    <mat-option *ngFor="let option of metricOpOptions" [value]="option">{{ option }}</mat-option>
                  </mat-select>
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <mat-form-field class="w-full" [matTooltip]="'Field name to run the metric on; for word_count you may also use body_text to count the whole page'">
                  <mat-label>Input</mat-label>
                  <mat-select formControlName="input">
                    <mat-option *ngFor="let option of fieldNameOptions" [value]="option">
                      {{ option }}
                    </mat-option>
                  </mat-select>
                  <mat-hint *ngIf="fieldNameOptions.length === 0">Add a field first</mat-hint>
                </mat-form-field>
                <button mat-icon-button color="warn" type="button" (click)="removeMetric(m)">
                  <mat-icon>delete</mat-icon>
                </button>
              </div>
            </div>
          </div>

          <mat-divider></mat-divider>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Pagination</p>
              <button mat-stroked-button color="primary" type="button" (click)="addPagination()">
                <mat-icon>add</mat-icon>
                Add Pagination
              </button>
            </div>
            <div formArrayName="pagination" class="space-y-3">
              <div
                *ngFor="let pag of pagination.controls; let p = index"
                [formGroupName]="p"
                class="border rounded p-3 grid grid-cols-1 md:grid-cols-4 gap-3 items-start"
              >
                <mat-form-field class="w-full" [matTooltip]="'Optional name for the pagination source (e.g., next_page, load_more)'">
                  <mat-label>Name</mat-label>
                  <input matInput formControlName="name" placeholder="next_page" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <mat-form-field class="w-full" [matTooltip]="'CSS selector for pagination elements (e.g., a.next-page, .pagination a)'">
                  <mat-label>Selector</mat-label>
                  <input matInput formControlName="selector" placeholder="a.next-page" />
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <mat-form-field class="w-full" [matTooltip]="'Attribute to extract URL from (default: href)'">
                  <mat-label>Attribute</mat-label>
                  <mat-select formControlName="attribute">
                    <mat-option value="href">href</mat-option>
                    <mat-option value="src">src</mat-option>
                    <mat-option value="data-url">data-url</mat-option>
                    <mat-option value="content">content</mat-option>
                  </mat-select>
                  <mat-icon matSuffix>info_outline</mat-icon>
                </mat-form-field>
                <div class="flex items-center gap-2">
                  <mat-checkbox formControlName="multiple" [matTooltip]="'Extract all matching elements (true) or just first (false)'">
                    Multiple
                  </mat-checkbox>
                  <button mat-icon-button color="warn" type="button" (click)="removePagination(p)">
                    <mat-icon>delete</mat-icon>
                  </button>
                </div>
              </div>
            </div>
            <div *ngIf="pagination.length === 0" class="text-gray-500 text-sm mt-2">
              No pagination selectors configured. Add pagination to follow next-page links.
            </div>
          </div>
        </mat-card-content>
      </mat-card>
      <mat-card>
        <mat-card-header>
          <mat-card-title>Preview Payload</mat-card-title>
        </mat-card-header>
        <mat-card-content>
          <pre class="bg-gray-100 p-3 rounded text-xs overflow-auto">{{ previewJson }}</pre>
          <div *ngIf="error" class="text-red-600 text-sm flex items-center gap-2 mt-2">
            <mat-icon>error</mat-icon>
            {{ error }}
          </div>
        </mat-card-content>
      </mat-card>
    </form>
  `,
  styles: [`
    :host {
      display: block;
    }

    pre {
      white-space: pre-wrap;
      word-break: break-word;
    }
  `]
})
export class SimpleJobCreateComponent implements OnInit {
  jobForm!: FormGroup;
  creating = false;
  error: string | null = null;
  previewJson = '';
  seedsScopeExpanded = false;
  rateLimitExpanded = false;
  extractionExpanded = true;
  readonly fieldTypeOptions: FieldSpec['type'][] = ['string', 'int', 'float', 'bool', 'url', 'json'];
  readonly attributeOptions = ['text', 'html', 'href', 'src', 'content'];
  readonly metricOpOptions: MetricSpec['op'][] = ['len', 'count', 'word_count', 'field_present', 'status_is_error', 'count_external_links'];
  readonly transformOpOptions: TransformSpec['op'][] = [
    'trim',
    'lower',
    'upper',
    'normalize_url',
    'unique',
    'limit',
    'to_int',
    'to_float',
    'parse_price',
    'html_to_text',
    'collapse_ws',
    'sha256'
  ];

  private readonly sampleConfig: CrawlJobConfig = {
    name: 'Example Crawl Job',
    seeds: [{ url: 'https://bool.dev/blog/detail/voprosy-na-sobesedovanii-dlya-senior-net-developer' }],
    scopes: {
      allowed_domains: ['bool.dev'],
      deny_url_patterns: ['/login', '/register'],
      max_depth: 0
    },
    rate_limit: { rps: 1 },
    retries: { max_attempts: 3, backoff_initial_ms: 500, backoff_multiplier: 2 },
    schedule: { cron: '0 9 * * 1' },
    auth: { basic_user: '', basic_password: '', bearer_token: '', cookie: '' },
    extraction_spec: {
      fields: [
        {
          name: 'page_url',
          type: 'string',
          required: true,
          extractor: {
            selector: '',
            attribute: 'text',
            multiple: false
          },
          transforms: []
        },
        {
          name: 'title',
          type: 'string',
          required: true,
          extractor: {
            selector: "h1[itemprop='headline']",
            attribute: 'text',
            multiple: false
          },
          transforms: []
        }
      ],
      metrics: [
        {
          name: 'questions_count',
          op: 'count',
          input: 'questions_h3'
        }
      ]
    }
  };

  constructor(
    private fb: FormBuilder,
    private crawlerApi: CrawlerApiService,
    private router: Router,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit(): void {
    this.jobForm = this.fb.group({
      name: ['', Validators.required],
      seeds: this.fb.array([this.createSeedGroup()]),
      allowed_domains: this.fb.array([]),
      deny_url_patterns: this.fb.array([]),
      max_depth: [0, [Validators.required, Validators.min(0)]],
      rps: [1, [Validators.required, Validators.min(0.1)]],
      retries: this.fb.group({
        max_attempts: [3],
        backoff_initial_ms: [500],
        backoff_multiplier: [2]
      }),
      auth: this.fb.group({
        basic_user: [''],
        basic_password: [''],
        bearer_token: [''],
        cookie: ['']
      }),
      schedule: this.fb.group({
        cron: ['']
      }),
      extraction_fields: this.fb.array([]),
      metrics: this.fb.array([]),
      pagination: this.fb.array([])
    });

    this.jobForm.valueChanges.subscribe(() => this.updatePreview());

    this.extractionFields.valueChanges.subscribe(() => {
      this.syncMetricInputs();
    });

    this.updatePreview();
  }

  get seeds(): FormArray {
    return this.jobForm.get('seeds') as FormArray;
  }

  get allowedDomains(): FormArray {
    return this.jobForm.get('allowed_domains') as FormArray;
  }

  get denyPatterns(): FormArray {
    return this.jobForm.get('deny_url_patterns') as FormArray;
  }

  get extractionFields(): FormArray {
    return this.jobForm.get('extraction_fields') as FormArray;
  }

  get metrics(): FormArray {
    return this.jobForm.get('metrics') as FormArray;
  }

  get pagination(): FormArray {
    return this.jobForm.get('pagination') as FormArray;
  }

  get retriesGroup(): FormGroup {
    return this.jobForm.get('retries') as FormGroup;
  }

  get scheduleGroup(): FormGroup {
    return this.jobForm.get('schedule') as FormGroup;
  }

  get authGroup(): FormGroup {
    return this.jobForm.get('auth') as FormGroup;
  }

  get fieldNameOptions(): string[] {
    return this.extractionFields.controls
      .map(ctrl => (ctrl.get('name')?.value || '').toString())
      .filter(name => name.trim() !== '');
  }

  goBack(): void {
    this.router.navigate(['/jobs']);
  }

  toggleSeedsScope(): void {
    this.seedsScopeExpanded = !this.seedsScopeExpanded;
  }

  toggleRateLimit(): void {
    this.rateLimitExpanded = !this.rateLimitExpanded;
  }

  toggleExtraction(): void {
    this.extractionExpanded = !this.extractionExpanded;
  }

  addSeed(url = ''): void {
    this.seeds.push(this.createSeedGroup(url));
    this.updatePreview();
  }

  removeSeed(index: number): void {
    if (this.seeds.length > 1) {
      this.seeds.removeAt(index);
      this.updatePreview();
    }
  }

  addAllowedDomain(domain = ''): void {
    this.allowedDomains.push(this.fb.control(domain));
    this.updatePreview();
  }

  removeAllowedDomain(index: number): void {
    this.allowedDomains.removeAt(index);
    this.updatePreview();
  }

  addDenyPattern(pattern = ''): void {
    this.denyPatterns.push(this.fb.control(pattern));
    this.updatePreview();
  }

  removeDenyPattern(index: number): void {
    this.denyPatterns.removeAt(index);
    this.updatePreview();
  }

  addExtractionField(field?: Partial<FieldSpec>): void {
    this.extractionFields.push(this.createExtractionFieldGroup(field));
    this.updatePreview();
  }

  removeExtractionField(index: number): void {
    this.extractionFields.removeAt(index);
    this.updatePreview();
  }

  addTransform(fieldIndex: number, transform?: TransformSpec): void {
    this.getTransforms(fieldIndex).push(this.createTransformGroup(transform));
    this.updatePreview();
  }

  removeTransform(fieldIndex: number, transformIndex: number): void {
    this.getTransforms(fieldIndex).removeAt(transformIndex);
    this.updatePreview();
  }

  addMetric(metric?: Partial<MetricSpec>): void {
    this.metrics.push(this.createMetricGroup(metric));
    this.updatePreview();
  }

  removeMetric(index: number): void {
    this.metrics.removeAt(index);
    this.updatePreview();
  }

  addPagination(pagination?: Partial<PaginationSpec>): void {
    this.pagination.push(this.createPaginationGroup(pagination));
    this.updatePreview();
  }

  removePagination(index: number): void {
    this.pagination.removeAt(index);
    this.updatePreview();
  }

  getTransforms(fieldIndex: number): FormArray {
    return (this.extractionFields.at(fieldIndex) as FormGroup).get('transforms') as FormArray;
  }

  handleMultipleToggle(fieldIndex: number): void {
    const fieldGroup = this.extractionFields.at(fieldIndex) as FormGroup;
    const multipleCtrl = fieldGroup.get('multiple');
    const indexCtrl = fieldGroup.get('index');

    if (!multipleCtrl?.value) {
      indexCtrl?.setValue(null);
    }
  }

  resetToSample(): void {
    const s = this.sampleConfig;
    this.jobForm.patchValue({
      name: s.name,
      max_depth: s.scopes.max_depth,
      rps: s.rate_limit.rps,
      retries: s.retries,
      schedule: s.schedule,
      auth: s.auth
    });

    this.resetArray(this.seeds, s.seeds.map(seed => this.createSeedGroup(seed.url)));
    this.resetArray(this.allowedDomains, s.scopes.allowed_domains.map(d => this.fb.control(d)));
    this.resetArray(this.denyPatterns, (s.scopes.deny_url_patterns || []).map(p => this.fb.control(p)));

    this.resetArray(
      this.extractionFields,
      s.extraction_spec.fields.map(f => this.createExtractionFieldGroup(f as Partial<FieldSpec>))
    );

    this.resetArray(
      this.metrics,
      s.extraction_spec.metrics.map(m => this.createMetricGroup(m as Partial<MetricSpec>))
    );

    this.updatePreview();
  }

  submit(): void {
    if (!this.canSubmit()) {
      this.error = 'Fill all required fields before creating a job.';
      return;
    }

    this.creating = true;
    this.error = null;
    const payload = { config: this.buildConfig() };

    this.crawlerApi.createJob(payload.config as CrawlJobConfig).subscribe({
      next: (response) => {
        this.creating = false;
        this.snackBar.open('Job created successfully', 'Close', { duration: 3000 });
        this.router.navigate(['/jobs', response.id]);
      },
      error: (err) => {
        this.creating = false;
        this.error = err.message || 'Failed to create job';
      }
    });
  }

  canSubmit(): boolean {
    return this.jobForm.valid && this.extractionFields.length > 0;
  }

  private createSeedGroup(url = ''): FormGroup {
    return this.fb.group({
      url: [url, [Validators.required, Validators.pattern(/^https?:\/\/.+/)]]
    });
  }

  private createExtractionFieldGroup(field?: Partial<FieldSpec>): FormGroup {
    return this.fb.group({
      name: [field?.name || '', Validators.required],
      type: [((field?.type || 'string') as FieldSpec['type']), Validators.required],
      required: [field?.required ?? false],
      selector: [field?.extractor?.selector || '', Validators.required],
      attribute: [field?.extractor?.attribute || 'text'],
      multiple: [field?.extractor?.multiple ?? false],
      index: [field?.extractor?.multiple ? field?.extractor?.index ?? null : null],
      transforms: this.fb.array(
        field?.transforms?.map(t => this.createTransformGroup(t)) || []
      )
    });
  }

  private createTransformGroup(transform?: TransformSpec): FormGroup {
    return this.fb.group({
      op: [transform?.op || 'trim', Validators.required],
      arg: [transform?.arg || '']
    });
  }

  private createMetricGroup(metric?: Partial<MetricSpec>): FormGroup {
    return this.fb.group({
      name: [metric?.name || '', Validators.required],
      op: [(metric?.op as MetricSpec['op']) || 'count', Validators.required],
      input: [metric?.input || '', Validators.required]
    });
  }

  private createPaginationGroup(pagination?: Partial<PaginationSpec>): FormGroup {
    return this.fb.group({
      name: [pagination?.name || ''],
      selector: [pagination?.selector || '', Validators.required],
      attribute: [pagination?.attribute || 'href'],
      multiple: [pagination?.multiple ?? false]
    });
  }

  private resetArray(target: FormArray, items: (FormGroup | any)[]): void {
    while (target.length > 0) {
      target.removeAt(0);
    }
    items.forEach(item => target.push(item));
  }

  private syncMetricInputs(): void {
    const options = new Set(this.fieldNameOptions);
    this.metrics.controls.forEach(ctrl => {
      const inputCtrl = ctrl.get('input');
      const current = (inputCtrl?.value || '').toString();
      if (current && !options.has(current)) {
        inputCtrl?.setValue('');
      }
    });
  }

  private buildConfig(): CrawlJobConfig {
    const raw: SimpleJobFormValue = this.jobForm.getRawValue();

    const extractionFields: FieldSpec[] = this.extractionFields.controls.map(ctrl => {
      const value = ctrl.value;
      const isMultiple = !!value.multiple;
      const extractor: any = {
        selector: value.selector,
        attribute: value.attribute,
        multiple: isMultiple
      };

      if (isMultiple) {
        extractor.index =
          value.index === null || value.index === undefined || value.index === ''
            ? null
            : Number(value.index);
      }

      return {
        name: value.name,
        type: value.type,
        required: !!value.required,
        extractor,
        transforms: value.transforms.filter((t: TransformSpec) => t?.op)
      };
    });

    const metrics: MetricSpec[] = this.metrics.controls.map(ctrl => {
      const value = ctrl.value;
      return {
        name: value.name,
        op: value.op,
        input: value.input
      };
    });

    const paginationSpecs: PaginationSpec[] = this.pagination.controls.map(ctrl => {
      const value = ctrl.value;
      return {
        name: value.name || undefined,
        selector: value.selector,
        attribute: value.attribute || 'href',
        multiple: !!value.multiple
      };
    }).filter(p => p.selector);

    return {
      name: raw.name,
      seeds: raw.seeds.map(seed => ({ url: seed.url })),
      scopes: {
        max_depth: Number(raw.max_depth),
        allowed_domains: raw.allowed_domains.filter(d => d && d.trim() !== ''),
        deny_url_patterns: raw.deny_url_patterns.filter(p => p && p.trim() !== '')
      },
      rate_limit: {
        rps: Number(raw.rps)
      },
      retries: raw.retries,
      auth: raw.auth,
      schedule: raw.schedule,
      extraction_spec: {
        fields: extractionFields,
        metrics,
        pagination: paginationSpecs.length > 0 ? paginationSpecs : undefined
      }
    };
  }

  private updatePreview(): void {
    const config = this.buildConfig();
    this.previewJson = JSON.stringify({ config }, null, 2);
  }
}
