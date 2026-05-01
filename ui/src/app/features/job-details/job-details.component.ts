import { Component, OnDestroy, OnInit } from '@angular/core';
import { animate, state, style, transition, trigger } from '@angular/animations';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { TagModule } from 'primeng/tag';
import { TooltipModule } from 'primeng/tooltip';
import { DividerModule } from 'primeng/divider';
import { ChartModule } from 'primeng/chart';
import { DialogModule } from 'primeng/dialog';
import { catchError, forkJoin, interval, of, Subscription, switchMap } from 'rxjs';
import 'chart.js/auto';
import { ChartData, ChartOptions } from 'chart.js';
import { CrawlerApiService, TaskListFilter, TaskAnalytics, TaskSortParams } from '../../core/services/api/crawler-api.service';
import { CrawlJob, CrawlTask, FileType, JobExportFileType } from '../../core/models';
import { TaskFiltersComponent } from './components/task-filters.component';

@Component({
  selector: 'app-job-details',
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    TableModule,
    ButtonModule,
    ProgressSpinnerModule,
    TagModule,
    TooltipModule,
    DividerModule,
    ChartModule,
    DialogModule,
    TaskFiltersComponent
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="mb-4">
        <p-button [text]="true" (onClick)="goBack()">
          <i class="pi pi-arrow-left mr-2"></i>
          Back to Jobs
        </p-button>
      </div>

      <p-card *ngIf="loading" styleClass="text-center p-8">
        <p-progressSpinner />
        <p class="mt-4">Loading job details...</p>
      </p-card>

      <p-card *ngIf="error && !loading" styleClass="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </p-card>

      <div *ngIf="!loading && !error">
        <!-- Job Info Card -->
        <p-card styleClass="mb-6">
          <ng-template pTemplate="header">
            <div class="p-4 pb-0">
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">{{ job?.name || job?.job_config?.name || 'Unnamed Job' }}</h2>
              <p class="text-sm text-gray-500 dark:text-gray-400">Job ID: {{ job?.id }}</p>
            </div>
          </ng-template>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <p class="text-sm text-gray-600 dark:text-gray-400">Status</p>
              <p-tag [value]="job?.status || ''" [severity]="getStatusSeverity(job?.status || '')" />
            </div>
            <div>
              <p class="text-sm text-gray-600 dark:text-gray-400">Job Type</p>
              <p-tag
                [value]="job?.job_config?.job_type === 'JOB_TYPE_SCHEDULED' ? 'Scheduled' : 'One-time'"
                [severity]="job?.job_config?.job_type === 'JOB_TYPE_SCHEDULED' ? 'info' : 'secondary'" />
            </div>
            <div>
              <p class="text-sm text-gray-600 dark:text-gray-400">Created At</p>
              <p class="text-gray-900 dark:text-white">{{ job?.created_at | date:'medium' }}</p>
            </div>
            <div>
              <p class="text-sm text-gray-600 dark:text-gray-400">Max Depth</p>
              <p class="text-gray-900 dark:text-white">{{ job?.job_config?.scopes?.max_depth || 'N/A' }}</p>
            </div>
            <div>
              <p class="text-sm text-gray-600 dark:text-gray-400">Rate Limit (RPS)</p>
              <p class="text-gray-900 dark:text-white">{{ job?.job_config?.rate_limit?.rps || 'N/A' }}</p>
            </div>
            <div>
              <p class="text-sm text-gray-600 dark:text-gray-400">Allowed URL Patterns</p>
              <p class="text-gray-900 dark:text-white break-all">{{ job?.job_config?.scopes?.allowed_url_patterns?.join(', ') || 'N/A' }}</p>
            </div>
          </div>

          <div class="mt-4" *ngIf="job?.job_config?.seeds">
            <p class="text-sm text-gray-600 dark:text-gray-400">Seed URLs</p>
            <div class="flex flex-wrap gap-2 mt-2">
              <p-tag *ngFor="let seed of job?.job_config?.seeds" [value]="seed.url" severity="secondary" />
            </div>
          </div>


          <div class="mt-4" *ngIf="job?.job_config?.queue_weights?.length">
            <p class="text-sm text-gray-600 dark:text-gray-400">Queue Routing Weights</p>
            <div class="flex flex-wrap gap-2 mt-2">
              <p-tag
                *ngFor="let qw of job?.job_config?.queue_weights"
                [value]="qw.queue + ' ×' + qw.weight"
                severity="secondary" />
            </div>
          </div>

          <div class="mt-4" *ngIf="job?.job_config?.extraction_spec?.pagination?.length">
            <p class="text-sm text-gray-600 dark:text-gray-400">Pagination Selectors</p>
            <div class="mt-2 space-y-2">
              <div *ngFor="let pag of job?.job_config?.extraction_spec?.pagination" class="bg-gray-50 dark:bg-gray-700 rounded p-2 text-sm text-gray-900 dark:text-gray-100">
                <div class="flex items-center gap-4">
                  <span *ngIf="pag.name" class="font-medium">{{ pag.name }}</span>
                  <code class="bg-gray-200 dark:bg-gray-600 px-2 py-1 rounded text-xs">{{ pag.selector }}</code>
                  <span class="text-gray-500 dark:text-gray-400">attr: {{ pag.attribute || 'href' }}</span>
                  <p-tag *ngIf="pag.multiple" value="multiple" severity="info" styleClass="text-xs" />
                </div>
              </div>
            </div>
          </div>

          <p-divider />

          <div>
            <p class="text-sm text-gray-600 dark:text-gray-400">Export Results</p>
            <div class="flex flex-wrap items-center gap-2 mt-2">
              <p-tag [value]="job?.export_status || 'NOT_STARTED'" [severity]="getStatusSeverity(job?.export_status || 'NOT_STARTED')" />
              <span *ngIf="job?.exported_at" class="text-sm text-gray-500 dark:text-gray-400">
                Exported: {{ job?.exported_at | date:'short' }}
              </span>
              <p-button
                [outlined]="true"
                severity="secondary"
                [disabled]="!job?.export_json_key || loadingFile['job-export-json']"
                [pTooltip]="job?.export_json_key ? 'Download JSON export' : 'JSON export not available'"
                (onClick)="downloadJobExport('json')">
                <i *ngIf="!loadingFile['job-export-json']" class="pi pi-download mr-2"></i>
                <p-progressSpinner *ngIf="loadingFile['job-export-json']" [style]="{width: '20px', height: '20px'}" />
                JSON
              </p-button>
              <p-button
                [outlined]="true"
                severity="help"
                [disabled]="!job?.export_csv_key || loadingFile['job-export-csv']"
                [pTooltip]="job?.export_csv_key ? 'Download CSV export' : 'CSV export not available'"
                (onClick)="downloadJobExport('csv')">
                <i *ngIf="!loadingFile['job-export-csv']" class="pi pi-download mr-2"></i>
                <p-progressSpinner *ngIf="loadingFile['job-export-csv']" [style]="{width: '20px', height: '20px'}" />
                CSV
              </p-button>
            </div>
          </div>

          <p-divider />

          <div>
            <p-button
              [outlined]="true"
              severity="secondary"
              (onClick)="toggleConfig()">
              <i class="pi mr-2" [ngClass]="configExpanded ? 'pi-chevron-up' : 'pi-chevron-down'"></i>
              Job Config (auth hidden)
            </p-button>
            <div
              class="detail-wrapper mt-3"
              [@expandCollapse]="configExpanded ? 'expanded' : 'collapsed'"
              [attr.aria-hidden]="!configExpanded">
              <pre class="json-view" *ngIf="getJobConfigWithoutAuth(job) as config">{{ config | json }}</pre>
              <div class="text-gray-500 dark:text-gray-400" *ngIf="job && !job.job_config">No job configuration available.</div>
            </div>
          </div>
        </p-card>

        <!-- Charts -->
        <p-card *ngIf="analytics" styleClass="mb-6">
          <ng-template pTemplate="header">
            <div class="p-4 pb-0">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <h2 class="text-xl font-semibold text-gray-900 dark:text-white">Task Analytics</h2>
                  <p class="text-sm text-gray-500 dark:text-gray-400">Status and depth distribution ({{ analytics.total_count }} total tasks).</p>
                </div>
                <p-button
                  [outlined]="true"
                  severity="secondary"
                  (onClick)="toggleAnalytics()">
                  <i class="pi mr-2" [ngClass]="analyticsExpanded ? 'pi-chevron-up' : 'pi-chevron-down'"></i>
                  {{ analyticsExpanded ? 'Collapse' : 'Expand' }}
                </p-button>
              </div>
            </div>
          </ng-template>

          <div
            class="detail-wrapper"
            [@expandCollapse]="analyticsExpanded ? 'expanded' : 'collapsed'"
            [attr.aria-hidden]="!analyticsExpanded">
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div class="chart-card">
                <h3 class="text-sm font-medium text-gray-600 dark:text-gray-400 mb-3">By Status</h3>
                <p-chart *ngIf="statusChartData"
                  type="doughnut"
                  [data]="statusChartData"
                  [options]="statusChartOptions"></p-chart>
              </div>
              <div class="chart-card">
                <h3 class="text-sm font-medium text-gray-600 dark:text-gray-400 mb-3">By Depth</h3>
                <p-chart *ngIf="depthChartData"
                  type="doughnut"
                  [data]="depthChartData"
                  [options]="depthChartOptions"></p-chart>
              </div>
            </div>
          </div>
        </p-card>

        <!-- Tasks Table -->
        <p-card>
          <ng-template pTemplate="header">
            <div class="p-4 pb-0">
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">Tasks ({{ tasks.length }} of {{ analytics?.total_count ?? '?' }})</h2>
            </div>
          </ng-template>

          <!-- Task Filters -->
          <app-task-filters
            [maxDepthValue]="job?.job_config?.scopes?.max_depth ?? 10"
            (filterChange)="onFilterChange($event)">
          </app-task-filters>

          <p-table
            [value]="tasks"
            [lazy]="true"
            (onLazyLoad)="onTaskLazyLoad($event)"
            [sortField]="currentTaskSortField"
            [sortOrder]="currentTaskSortOrder"
            [tableStyle]="{'min-width': '60rem'}">
            <ng-template pTemplate="header">
              <tr>
                <th pSortableColumn="url">URL <p-sortIcon field="url" /></th>
                <th pSortableColumn="status">Status <p-sortIcon field="status" /></th>
                <th pSortableColumn="depth">Depth <p-sortIcon field="depth" /></th>
                <th pSortableColumn="enqueued_at">Enqueued At <p-sortIcon field="enqueued_at" /></th>
                <th>Body Hash</th>
                <th>Files</th>
              </tr>
            </ng-template>
            <ng-template pTemplate="body" let-task>
              <tr>
                <td class="truncate max-w-md">
                  <a [href]="task.url" target="_blank" rel="noopener noreferrer" class="text-blue-600 dark:text-blue-400 hover:underline">
                    {{ task.url }}
                  </a>
                </td>
                <td>
                  <p-tag [value]="task.status" [severity]="getStatusSeverity(task.status)" />
                </td>
                <td>{{ task.depth }}</td>
                <td>{{ task.enqueued_at | date:'medium' }}</td>
                <td class="font-mono text-xs break-all">
                  {{ task.body_hash ? (task.body_hash | slice:0:8) + '...' : '' }}
                </td>
                <td>
                  <div class="flex gap-1">
                    <p-button
                      [text]="true"
                      [rounded]="true"
                      severity="secondary"
                      [disabled]="!task.minio_object_key || loadingFile[task.id + '-pages']"
                      [pTooltip]="task.minio_object_key ? 'Download HTML page' : 'HTML not available'"
                      (onClick)="downloadTaskFile(task, 'pages')">
                      <i *ngIf="!loadingFile[task.id + '-pages']" class="pi pi-download"></i>
                      <p-progressSpinner *ngIf="loadingFile[task.id + '-pages']" [style]="{width: '20px', height: '20px'}" />
                    </p-button>
                    <p-button
                      [text]="true"
                      [rounded]="true"
                      severity="help"
                      [disabled]="!task.result_object_key || loadingFile[task.id + '-result']"
                      [pTooltip]="task.result_object_key ? 'Download JSON result' : 'Result not available'"
                      (onClick)="downloadTaskFile(task, 'result')">
                      <i *ngIf="!loadingFile[task.id + '-result']" class="pi pi-download"></i>
                      <p-progressSpinner *ngIf="loadingFile[task.id + '-result']" [style]="{width: '20px', height: '20px'}" />
                    </p-button>
                    <p-button
                      *ngIf="task.error_message"
                      [text]="true"
                      [rounded]="true"
                      severity="danger"
                      pTooltip="View error details"
                      (onClick)="showErrorDialog(task)">
                      <i class="pi pi-exclamation-triangle"></i>
                    </p-button>
                  </div>
                </td>
              </tr>
            </ng-template>
            <ng-template pTemplate="emptymessage">
              <tr>
                <td colspan="6" class="text-center p-8 text-gray-500 dark:text-gray-400">
                  <i class="pi pi-list text-6xl block mb-4"></i>
                  <p>No tasks found for this job.</p>
                </td>
              </tr>
            </ng-template>
          </p-table>

          <!-- Load More Button -->
          <div class="flex justify-center p-4" *ngIf="hasMoreTasks">
            <p-button
              [outlined]="true"
              severity="secondary"
              (onClick)="loadMoreTasks()"
              [disabled]="loadingMoreTasks">
              <p-progressSpinner *ngIf="loadingMoreTasks" [style]="{width: '18px', height: '18px'}" styleClass="mr-2" />
              <span>{{ loadingMoreTasks ? 'Loading...' : 'Load More' }}</span>
            </p-button>
          </div>
        </p-card>
      </div>

      <!-- Error Dialog -->
      <p-dialog
        header="Task Error Details"
        [(visible)]="errorDialogVisible"
        [modal]="true"
        [style]="{width: '600px'}"
        [breakpoints]="{'768px': '90vw'}">
        <div *ngIf="selectedErrorTask" class="space-y-4">
          <div>
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-1">Task ID</p>
            <code class="text-sm bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-gray-900 dark:text-gray-100">{{ selectedErrorTask.id }}</code>
          </div>
          <div>
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-1">URL</p>
            <a [href]="selectedErrorTask.url" target="_blank" rel="noopener noreferrer" class="text-blue-600 dark:text-blue-400 hover:underline text-sm break-all">
              {{ selectedErrorTask.url }}
            </a>
          </div>
          <div>
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-1">Error Message</p>
            <div class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded p-3">
              <pre class="text-sm text-red-700 dark:text-red-400 whitespace-pre-wrap break-words m-0">{{ selectedErrorTask.error_message }}</pre>
            </div>
          </div>
        </div>
        <ng-template pTemplate="footer">
          <p-button label="Close" (onClick)="errorDialogVisible = false" />
        </ng-template>
      </p-dialog>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    .detail-wrapper {
      padding: 16px 24px;
      background: #f8fafc;
      border-top: 1px solid #e5e7eb;
      overflow: hidden;
    }

    :host-context(.dark-mode) .detail-wrapper {
      background: #1f2937;
      border-top-color: #374151;
    }

    .json-view {
      margin: 0;
      padding: 12px;
      background: #0b1021;
      color: #d1e5ff;
      border-radius: 6px;
      overflow: auto;
      font-family: SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
      font-size: 13px;
    }

    .chart-card :is(canvas, .p-chart) {
      width: 100% !important;
      height: 260px !important;
    }

    .endpoint-card {
      border: 1px solid #e5e7eb;
      border-radius: 8px;
      padding: 12px;
      background: #f9fafb;
    }

    :host-context(.dark-mode) .endpoint-card {
      border-color: #374151;
      background: #1f2937;
    }

    .endpoint-meta {
      display: flex;
      align-items: center;
      gap: 5px;
      font-size: 11px;
      color: #6b7280;
      margin: 0;
    }

    :host-context(.dark-mode) .endpoint-meta {
      color: #9ca3af;
    }

    .endpoint-meta .pi {
      font-size: 10px;
      opacity: 0.7;
      flex-shrink: 0;
    }
  `],
  animations: [
    trigger('expandCollapse', [
      state('collapsed', style({
        height: '0px',
        opacity: 0,
        padding: '0 24px',
        borderTopColor: 'transparent'
      })),
      state('expanded', style({
        height: '*',
        opacity: 1,
        padding: '16px 24px',
        borderTopColor: '#e5e7eb'
      })),
      transition('expanded <=> collapsed', animate('200ms cubic-bezier(0.4, 0.0, 0.2, 1)'))
    ])
  ]
})
export class JobDetailsComponent implements OnInit, OnDestroy {
  job: CrawlJob | null = null;
  tasks: CrawlTask[] = [];
  loading = false;
  error: string | null = null;
  configExpanded = false;
  analyticsExpanded = false;
  loadingFile: { [key: string]: boolean } = {};
  statusChartData: ChartData<'doughnut', number[], string> | null = null;
  statusChartOptions: ChartOptions<'doughnut'> | null = null;
  depthChartData: ChartData<'doughnut', number[], string> | null = null;
  depthChartOptions: ChartOptions<'doughnut'> | null = null;
  private analyticsPollingSub: Subscription | null = null;
  private tasksPollingSub: Subscription | null = null;
  private jobPollingSub: Subscription | null = null;

  // Pagination state
  taskCursor: string | null = null;
  hasMoreTasks = false;
  loadingMoreTasks = false;
  currentFilter: TaskListFilter = {};
  private readonly pageSize = 20;

  // Sort state (PrimeNG: 1 = ASC, -1 = DESC)
  currentTaskSortField = 'enqueued_at';
  currentTaskSortOrder = 1;
  private currentSort: TaskSortParams = { sort_field: 'TASK_SORT_FIELD_ENQUEUED_AT', sort_order: 'SORT_ORDER_ASC' };
  private taskTableInitialized = false; // skip first (onLazyLoad) — tasks already loaded by forkJoin
  private readonly statusChartColors: Record<string, string> = {
    pending: '#f59e0b',
    queued: '#06b6d4',
    inprogress: '#3b82f6',
    fetched: '#14b8a6',
    parsed: '#8b5cf6',
    completed: '#22c55e',
    failed: '#ef4444',
    skipped: '#64748b',
  };

  // Server-side analytics
  analytics: TaskAnalytics | null = null;

  // Error dialog
  errorDialogVisible = false;
  selectedErrorTask: CrawlTask | null = null;

  constructor(
    private crawlerApi: CrawlerApiService,
    private route: ActivatedRoute,
    private router: Router
  ) {}

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.loadJobDetails(id);
      this.startAnalyticsPolling(id);
    }
  }

  ngOnDestroy(): void {
    this.stopAnalyticsPolling();
    this.stopTasksPolling();
    this.stopJobPolling();
  }

  loadJobDetails(id: string): void {
    this.loading = true;
    this.error = null;
    this.taskTableInitialized = false;

    forkJoin({
      job: this.crawlerApi.getJob(id),
      tasks: this.crawlerApi.listTasksByJob(id, { limit: this.pageSize, sort: this.currentSort }),
      analytics: this.crawlerApi.getTaskAnalytics(id)
    }).subscribe({
      next: (response) => {
        this.job = response.job.job;
        this.tasks = response.tasks.tasks;
        this.taskCursor = response.tasks.next_cursor || null;
        this.hasMoreTasks = response.tasks.has_more;
        this.analytics = response.analytics.analytics;
        this.buildChartsFromAnalytics();
        this.loading = false;
        this.startTasksPolling();
        this.startJobPolling(id);
      },
      error: (err) => {
        this.error = `Failed to load job details: ${err.message}`;
        this.loading = false;
      }
    });
  }

  onFilterChange(filter: TaskListFilter): void {
    this.currentFilter = filter;
    this.taskCursor = null;
    this.tasks = [];
    this.loadTasks();
  }

  private readonly taskSortFieldMap: Record<string, TaskSortParams['sort_field']> = {
    'enqueued_at': 'TASK_SORT_FIELD_ENQUEUED_AT',
    'url': 'TASK_SORT_FIELD_URL',
    'status': 'TASK_SORT_FIELD_STATUS',
    'depth': 'TASK_SORT_FIELD_DEPTH',
  };

  onTaskLazyLoad(event: { sortField?: string | string[] | null; sortOrder?: number | null }): void {
    // Skip the first firing — tasks are already loaded by the initial forkJoin
    if (!this.taskTableInitialized) {
      this.taskTableInitialized = true;
      return;
    }

    const field = (event.sortField as string) || 'enqueued_at';
    const order = event.sortOrder ?? 1;

    this.currentTaskSortField = field;
    this.currentTaskSortOrder = order;
    this.currentSort = {
      sort_field: this.taskSortFieldMap[field] ?? 'TASK_SORT_FIELD_ENQUEUED_AT',
      sort_order: order === 1 ? 'SORT_ORDER_ASC' : 'SORT_ORDER_DESC',
    };
    this.taskCursor = null;
    this.tasks = [];
    this.loadTasks();
  }

  loadTasks(): void {
    if (!this.job) return;

    this.crawlerApi.listTasksByJob(this.job.id, {
      limit: this.pageSize,
      filter: this.currentFilter,
      sort: this.currentSort
    }).subscribe({
      next: (response) => {
        this.tasks = response.tasks;
        this.taskCursor = response.next_cursor || null;
        this.hasMoreTasks = response.has_more;
      },
      error: (err) => {
        console.error('Failed to load tasks:', err);
      }
    });
  }

  loadMoreTasks(): void {
    if (!this.job || !this.hasMoreTasks || this.loadingMoreTasks) return;

    this.loadingMoreTasks = true;
    this.crawlerApi.listTasksByJob(this.job.id, {
      cursor: this.taskCursor ?? undefined,
      limit: this.pageSize,
      filter: this.currentFilter,
      sort: this.currentSort
    }).subscribe({
      next: (response) => {
        this.tasks = [...this.tasks, ...response.tasks];
        this.taskCursor = response.next_cursor || null;
        this.hasMoreTasks = response.has_more;
        this.loadingMoreTasks = false;
      },
      error: (err) => {
        console.error('Failed to load more tasks:', err);
        this.loadingMoreTasks = false;
      }
    });
  }

  private startAnalyticsPolling(id: string): void {
    this.stopAnalyticsPolling();
    this.analyticsPollingSub = interval(5000)
      .pipe(
        switchMap(() => this.crawlerApi.getTaskAnalytics(id).pipe(
          catchError((err) => {
            console.error(`Failed to poll analytics: ${err.message}`);
            return of({ analytics: this.analytics! });
          })
        ))
      )
      .subscribe({
        next: (response) => {
          this.analytics = response.analytics;
          this.buildChartsFromAnalytics();
        }
      });
  }

  private stopAnalyticsPolling(): void {
    this.analyticsPollingSub?.unsubscribe();
    this.analyticsPollingSub = null;
  }

  private startTasksPolling(): void {
    this.stopTasksPolling();
    if (!this.job || this.isTerminalStatus(this.job.status || '')) return;

    const id = this.job.id;
    this.tasksPollingSub = interval(5000)
      .pipe(
        switchMap(() => this.crawlerApi.listTasksByJob(id, {
          limit: this.pageSize,
          filter: this.currentFilter,
          sort: this.currentSort
        }).pipe(
          catchError((err) => {
            console.error(`Failed to poll tasks: ${err.message}`);
            return of({ tasks: this.tasks, next_cursor: this.taskCursor, has_more: this.hasMoreTasks });
          })
        ))
      )
      .subscribe({
        next: (response) => {
          this.applyPolledTasks(response);
        }
      });
  }

  private applyPolledTasks(response: { tasks: CrawlTask[]; next_cursor: string | null; has_more: boolean }): void {
    if (this.loadingMoreTasks) return;

    const isFirstPageView = this.tasks.length <= this.pageSize;
    if (isFirstPageView) {
      this.tasks = response.tasks;
      this.taskCursor = response.next_cursor || null;
      this.hasMoreTasks = response.has_more;
      return;
    }

    const polledTasksById = new Map(response.tasks.map(task => [task.id, task]));
    this.tasks = this.tasks.map(task => polledTasksById.get(task.id) ?? task);
  }

  private stopTasksPolling(): void {
    this.tasksPollingSub?.unsubscribe();
    this.tasksPollingSub = null;
  }

  private isTerminalStatus(status: string): boolean {
    const normalized = status.toLowerCase().replace(/_/g, '');
    return normalized === 'completed' || normalized === 'failed';
  }

  private startJobPolling(id: string): void {
    this.stopJobPolling();
    if (this.job && this.isTerminalStatus(this.job.status || '')) return;

    this.jobPollingSub = interval(5000)
      .pipe(
        switchMap(() => this.crawlerApi.getJob(id).pipe(
          catchError((err) => {
            console.error(`Failed to poll job: ${err.message}`);
            return of({ job: this.job! });
          })
        ))
      )
      .subscribe({
        next: (response) => {
          const updatedJob = response.job;
          const becameTerminal = !this.isTerminalStatus(this.job?.status || '') &&
                                 this.isTerminalStatus(updatedJob?.status || '');
          this.job = updatedJob;

          if (this.isTerminalStatus(updatedJob?.status || '')) {
            this.stopJobPolling();
            if (becameTerminal) {
              if (this.tasks.length <= this.pageSize) {
                this.loadTasks();
              }
              this.stopTasksPolling();
            }
          }
        }
      });
  }

  private stopJobPolling(): void {
    this.jobPollingSub?.unsubscribe();
    this.jobPollingSub = null;
  }

  goBack(): void {
    this.router.navigate(['/jobs']);
  }

  toggleConfig(): void {
    this.configExpanded = !this.configExpanded;
  }

  toggleAnalytics(): void {
    this.analyticsExpanded = !this.analyticsExpanded;
  }

  showErrorDialog(task: CrawlTask): void {
    this.selectedErrorTask = task;
    this.errorDialogVisible = true;
  }

  getJobConfigWithoutAuth(job: CrawlJob | null) {
    const config = job?.job_config;
    if (!config) {
      return null;
    }

    const { auth, ...safeConfig } = config;
    return safeConfig;
  }

  getBrokerLabel(brokerType: string): string {
    switch (brokerType) {
      case 'QUEUE_BROKER_TYPE_RABBITMQ': return 'RabbitMQ';
      case 'QUEUE_BROKER_TYPE_KAFKA': return 'Kafka';
      default: return brokerType;
    }
  }


  getStatusSeverity(status: string): 'success' | 'info' | 'warn' | 'danger' | 'secondary' {
    const normalized = status.toLowerCase().replace(/_/g, '');
    switch (normalized) {
      case 'completed':
        return 'success';
      case 'failed':
        return 'danger';
      case 'inprogress':
      case 'queued':
        return 'info';
      default:
        return 'secondary';
    }
  }

  private buildChartsFromAnalytics(): void {
    if (!this.analytics) return;

    this.buildStatusChart();
    this.buildDepthChart();
  }

  private buildStatusChart(): void {
    if (!this.analytics) return;

    const statusEntries = Object.entries(this.analytics.status_counts);

    this.statusChartData = {
      labels: statusEntries.map(([status]) => status.replace(/_/g, ' ')),
      datasets: [
        {
          data: statusEntries.map(([, count]) => count),
          backgroundColor: statusEntries.map(([status]) => this.getStatusChartColor(status)),
          borderColor: '#ffffff',
          borderWidth: 2
        }
      ]
    };
    this.statusChartOptions = {
      responsive: true,
      plugins: {
        legend: {
          position: 'bottom'
        }
      }
    };
  }

  private getStatusChartColor(status: string): string {
    return this.statusChartColors[this.normalizeStatusKey(status)] ?? '#94a3b8';
  }

  private normalizeStatusKey(status: string): string {
    return status.toLowerCase().replace(/[\s_-]/g, '');
  }

  private buildDepthChart(): void {
    if (!this.analytics) return;

    const depths = Object.keys(this.analytics.depth_counts).map(Number).sort((a, b) => a - b);
    const palette = ['#0ea5e9', '#22c55e', '#f97316', '#e879f9', '#f59e0b', '#14b8a6', '#6366f1'];

    this.depthChartData = {
      labels: depths.map(depth => `Depth ${depth}`),
      datasets: [
        {
          data: depths.map(depth => this.analytics!.depth_counts[depth.toString()] ?? 0),
          backgroundColor: depths.map((_, index) => palette[index % palette.length]),
          borderColor: '#ffffff',
          borderWidth: 2
        }
      ]
    };
    this.depthChartOptions = {
      responsive: true,
      plugins: {
        legend: {
          position: 'bottom'
        }
      }
    };
  }

  downloadTaskFile(task: CrawlTask, fileType: FileType): void {
    const loadingKey = `${task.id}-${fileType}`;
    this.loadingFile[loadingKey] = true;

    this.crawlerApi.getTaskFileURL(task.id, fileType).subscribe({
      next: (response) => {
        this.openPresignedDownload(response.url);
        this.loadingFile[loadingKey] = false;
      },
      error: (err) => {
        this.loadingFile[loadingKey] = false;
        console.error(`Failed to get file URL: ${err.message}`);
      }
    });
  }

  downloadJobExport(fileType: JobExportFileType): void {
    if (!this.job) {
      return;
    }

    const loadingKey = `job-export-${fileType}`;
    this.loadingFile[loadingKey] = true;

    this.crawlerApi.getJobExportFileURL(this.job.id, fileType).subscribe({
      next: (response) => {
        this.openPresignedDownload(response.url);
        this.loadingFile[loadingKey] = false;
      },
      error: (err) => {
        this.loadingFile[loadingKey] = false;
        console.error(`Failed to get export URL: ${err.message}`);
      }
    });
  }

  private openPresignedDownload(url: string): void {
    const link = document.createElement('a');
    link.href = url;
    link.target = '_blank';
    link.rel = 'noopener noreferrer';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }
}
