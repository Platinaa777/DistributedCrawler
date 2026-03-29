import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { TagModule } from 'primeng/tag';
import { DialogModule } from 'primeng/dialog';
import { TooltipModule } from 'primeng/tooltip';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { CrawlerApiService, JobListFilter, JobSortParams } from '../../core/services/api/crawler-api.service';

import { AuthService } from '../../core/services/auth.service';
import { CrawlJob, CrawlJobConfig } from '../../core/models';

import { JobFiltersComponent } from './components/job-filters.component';

@Component({
  selector: 'app-jobs-list',
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    TableModule,
    ButtonModule,
    ProgressSpinnerModule,
    TagModule,
    DialogModule,
    TooltipModule,
    JobFiltersComponent
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Crawl Jobs</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Track status, inspect configs, and launch new crawls.</p>
        </div>
        <div class="flex items-center gap-3">
          <p-button
            *ngIf="canManageUsers"
            [outlined]="true"
            severity="secondary"
            (onClick)="goToUsers()">
            <i class="pi pi-users mr-2"></i>
            Users
          </p-button>
          <p-button
            *ngIf="canManageUsers"
            [outlined]="true"
            severity="secondary"
            (onClick)="goToMonitoring()">
            <i class="pi pi-chart-line mr-2"></i>
            Monitoring
          </p-button>
          <p-button
            severity="primary"
            (onClick)="createJob()"
            [disabled]="!canCreateJobs">
            <i class="pi pi-plus mr-2"></i>
            Create Job
          </p-button>
        </div>
      </div>

      <!-- Filters -->
      <app-job-filters
        [currentUserEmail]="currentUserEmail"
        (filterChange)="onFilterChange($event)"></app-job-filters>

      <p-card *ngIf="loading && jobs.length === 0" styleClass="text-center p-8">
        <p-progressSpinner />
        <p class="mt-4">Loading jobs...</p>
      </p-card>

      <p-card *ngIf="error && !loading" styleClass="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </p-card>

      <p-card *ngIf="!loading || jobs.length > 0">
        <p-table
          [value]="jobs"
          dataKey="id"
          [expandedRowKeys]="expandedRows"
          (onRowExpand)="onRowExpand($event)"
          (onRowCollapse)="onRowCollapse($event)"
          [lazy]="true"
          (onLazyLoad)="onLazyLoad($event)"
          [sortField]="currentSortField"
          [sortOrder]="currentSortOrder"
          [tableStyle]="{'min-width': '60rem'}">
          <ng-template #header>
            <tr>
              <th style="width: 3rem"></th>
              <th pSortableColumn="name">Name <p-sortIcon field="name" /></th>
              <th>Type</th>
              <th pSortableColumn="status">Status <p-sortIcon field="status" /></th>
              <th pSortableColumn="created_at">Created At <p-sortIcon field="created_at" /></th>
              <th style="width: 16rem"></th>
            </tr>
          </ng-template>
          <ng-template #body let-job let-expanded="expanded">
            <tr>
              <td>
                <p-button
                  type="button"
                  [pRowToggler]="job"
                  [text]="true"
                  [rounded]="true"
                  [plain]="true"
                  [icon]="expanded ? 'pi pi-chevron-down' : 'pi pi-chevron-right'" />
              </td>
              <td>{{ job.name || job.job_config?.name || 'Unnamed Job' }}</td>
              <td>
                <p-tag
                  [value]="job.job_config?.job_type === 'JOB_TYPE_SCHEDULED' ? 'Scheduled' : 'One-time'"
                  [severity]="job.job_config?.job_type === 'JOB_TYPE_SCHEDULED' ? 'info' : 'secondary'"
                  styleClass="text-xs" />
              </td>
              <td>
                <p-tag [value]="job.status" [severity]="getStatusSeverity(job.status)" />
              </td>
              <td>{{ job.created_at | date:'short' }}</td>
              <td class="text-right">
                <div class="flex items-center justify-end gap-2">
                  <p-button
                    *ngIf="canCreateJobs"
                    [outlined]="true"
                    severity="secondary"
                    [disabled]="!job.job_config"
                    (onClick)="copyJob(job)">
                    <i class="pi pi-copy mr-2"></i>
                    Copy
                  </p-button>
                  <p-button
                    [outlined]="true"
                    severity="secondary"
                    (onClick)="viewJob(job)">
                    <i class="pi pi-external-link mr-2"></i>
                    Open
                  </p-button>
                  <p-button
                    *ngIf="canDeleteJobs"
                    [outlined]="true"
                    severity="danger"
                    styleClass="job-delete-button"
                    icon="pi pi-trash"
                    pTooltip="Delete job and its config (all runs for scheduled jobs)"
                    (onClick)="confirmDelete(job)" />
                </div>
              </td>
            </tr>
          </ng-template>
          <ng-template #expandedrow let-job>
            <tr>
              <td [attr.colspan]="6">
                <div class="detail-wrapper">
                  <div class="detail-header">
                    <div class="detail-title">Job Config (auth hidden)</div>
                  </div>
                  <div class="mb-3" *ngIf="job.job_config?.scopes?.allowed_url_patterns?.length">
                    <p class="text-sm font-semibold mb-2 text-gray-900 dark:text-white">Allowed URL Patterns</p>
                    <code class="text-xs bg-white dark:bg-gray-700 rounded px-3 py-2 border border-gray-200 dark:border-gray-600 text-gray-900 dark:text-gray-100 inline-block">
                      {{ job.job_config?.scopes?.allowed_url_patterns?.join(', ') }}
                    </code>
                  </div>
                  <div class="pagination-info mb-3" *ngIf="job.job_config?.extraction_spec?.pagination?.length">
                    <p class="text-sm font-semibold mb-2 text-gray-900 dark:text-white">Pagination Selectors</p>
                    <div class="flex flex-wrap gap-2">
                      <div *ngFor="let pag of job.job_config?.extraction_spec?.pagination" class="bg-white dark:bg-gray-700 rounded px-3 py-1 text-sm border border-gray-200 dark:border-gray-600 text-gray-900 dark:text-gray-100">
                        <span *ngIf="pag.name" class="font-medium mr-2">{{ pag.name }}:</span>
                        <code class="text-xs">{{ pag.selector }}</code>
                        <span class="text-gray-500 dark:text-gray-400 ml-2">({{ pag.attribute || 'href' }})</span>
                        <span *ngIf="pag.multiple" class="text-blue-600 dark:text-blue-400 ml-1">[multiple]</span>
                      </div>
                    </div>
                  </div>
                  <pre class="json-view" *ngIf="getJobConfigWithoutAuth(job) as config">{{ config | json }}</pre>
                  <div class="text-gray-500 dark:text-gray-400" *ngIf="!job.job_config">No job configuration available.</div>
                </div>
              </td>
            </tr>
          </ng-template>
          <ng-template #emptymessage>
            <tr>
              <td colspan="6" class="text-center p-8 text-gray-500 dark:text-gray-400">
                <i class="pi pi-briefcase text-6xl block mb-4"></i>
                <p>No jobs found.</p>
              </td>
            </tr>
          </ng-template>
        </p-table>

        <!-- Load More Button -->
        <div class="flex justify-center p-4" *ngIf="hasMore">
          <p-button
            severity="primary"
            (onClick)="loadMore()"
            [disabled]="loadingMore">
            <p-progressSpinner *ngIf="loadingMore" [style]="{width: '18px', height: '18px'}" />
            <span class="ml-2">{{ loadingMore ? 'Loading...' : 'Load More' }}</span>
          </p-button>
        </div>
      </p-card>
    </div>

    <!-- Delete Confirm Dialog -->
    <p-dialog
      header="Delete Job"
      [(visible)]="showDeleteDialog"
      [modal]="true"
      [style]="{ width: '420px' }"
      [closable]="!deleting">
      <div class="py-2">
        <p class="text-gray-700 dark:text-gray-300 mb-2">
          Are you sure you want to delete this job?
        </p>
        <p class="text-sm text-amber-600 dark:text-amber-400" *ngIf="jobToDelete?.job_config?.job_type === 'JOB_TYPE_SCHEDULED'">
          <i class="pi pi-exclamation-triangle mr-1"></i>
          This is a <strong>scheduled job</strong>. Deleting it will remove the schedule config and <strong>all runs</strong> associated with it.
        </p>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-2" *ngIf="jobToDelete">
          Job: <strong>{{ jobToDelete.name || jobToDelete.job_config?.name || 'Unnamed Job' }}</strong>
        </p>
      </div>
      <ng-template pTemplate="footer">
        <div class="flex justify-end gap-2">
          <p-button
            label="Cancel"
            severity="secondary"
            [outlined]="true"
            [disabled]="deleting"
            (onClick)="cancelDelete()" />
          <p-button
            label="Delete"
            severity="danger"
            [loading]="deleting"
            (onClick)="executeDelete()" />
        </div>
      </ng-template>
    </p-dialog>
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

    .detail-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 12px;
      gap: 12px;
    }

    .detail-title {
      font-weight: 600;
      color: #111827;
    }

    :host-context(.dark-mode) .detail-title {
      color: #f3f4f6;
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

    :host ::ng-deep .job-delete-button.p-button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 2.5rem;
      width: 2.5rem;
      height: 2.5rem;
      padding-inline: 0;
    }

    :host ::ng-deep .job-delete-button .pi {
      line-height: 1;
    }
  `]
})
export class JobsListComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  jobs: CrawlJob[] = [];
  loading = false;
  loadingMore = false;
  error: string | null = null;
  expandedRows: { [key: string]: boolean } = {};
  hasMore = false;
  // Delete state
  showDeleteDialog = false;
  jobToDelete: CrawlJob | null = null;
  deleting = false;

  // Sort state (PrimeNG: 1 = ASC, -1 = DESC)
  currentSortField = 'created_at';
  currentSortOrder = -1;

  private nextCursor: string | null = null;
  private currentFilter: JobListFilter = {};
  private currentSort: JobSortParams = { sort_field: 'JOB_SORT_FIELD_CREATED_AT', sort_order: 'SORT_ORDER_DESC' };
  private readonly pageSize = 20;

  constructor(
    private crawlerApi: CrawlerApiService,
    private router: Router,
    private authService: AuthService
  ) {}

  ngOnInit(): void {
    // Initial job load is triggered by (onLazyLoad) from p-table init
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  onFilterChange(filter: JobListFilter): void {
    this.currentFilter = filter;
    this.jobs = [];
    this.nextCursor = null;
    this.loadJobs();
  }

  private readonly jobSortFieldMap: Record<string, JobSortParams['sort_field']> = {
    'name': 'JOB_SORT_FIELD_NAME',
    'status': 'JOB_SORT_FIELD_STATUS',
    'created_at': 'JOB_SORT_FIELD_CREATED_AT',
  };

  onLazyLoad(event: { sortField?: string | string[] | null; sortOrder?: number | null }): void {
    const field = (event.sortField as string) || 'created_at';
    const order = event.sortOrder ?? -1;

    const sortChanged = field !== this.currentSortField || order !== this.currentSortOrder;
    this.currentSortField = field;
    this.currentSortOrder = order;
    this.currentSort = {
      sort_field: this.jobSortFieldMap[field] ?? 'JOB_SORT_FIELD_CREATED_AT',
      sort_order: order === 1 ? 'SORT_ORDER_ASC' : 'SORT_ORDER_DESC',
    };

    if (sortChanged) {
      this.jobs = [];
      this.nextCursor = null;
    }
    this.loadJobs();
  }

  loadJobs(): void {
    this.loading = true;
    this.error = null;

    this.crawlerApi.listJobs({
      limit: this.pageSize,
      filter: this.currentFilter,
      sort: this.currentSort
    }).pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: (response) => {
        this.jobs = response.jobs || [];
        this.nextCursor = response.next_cursor || null;
        this.hasMore = response.has_more;
        this.loading = false;
      },
      error: (err) => {
        this.error = `Failed to load jobs: ${err.message}`;
        this.loading = false;
      }
    });
  }

  loadMore(): void {
    if (!this.nextCursor || this.loadingMore) return;

    this.loadingMore = true;
    this.crawlerApi.listJobs({
      cursor: this.nextCursor,
      limit: this.pageSize,
      filter: this.currentFilter,
      sort: this.currentSort
    }).pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: (response) => {
        this.jobs = [...this.jobs, ...(response.jobs || [])];
        this.nextCursor = response.next_cursor || null;
        this.hasMore = response.has_more;
        this.loadingMore = false;
      },
      error: (err) => {
        this.error = `Failed to load more jobs: ${err.message}`;
        this.loadingMore = false;
      }
    });
  }

  onRowExpand(event: { data: CrawlJob }): void {
    this.expandedRows = { [event.data.id]: true };
  }

  onRowCollapse(event: { data: CrawlJob }): void {
    this.expandedRows = {};
  }

  viewJob(job: CrawlJob): void {
    this.router.navigate(['/jobs', job.id]);
  }

  copyJob(job: CrawlJob): void {
    if (!job.job_config) {
      return;
    }

    this.router.navigate(['/jobs/simple-create'], {
      state: {
        initialConfig: this.cloneJobConfig(job.job_config),
        copySourceJobId: job.id,
        copySourceJobName: job.name || job.job_config.name || 'Unnamed Job'
      }
    });
  }

  getJobConfigWithoutAuth(job: CrawlJob) {
    const config = job.job_config;
    if (!config) {
      return null;
    }

    const { auth, ...safeConfig } = config;
    return safeConfig;
  }

  createJob(): void {
    this.router.navigate(['/jobs/simple-create']);
  }

  private cloneJobConfig(config: CrawlJobConfig): CrawlJobConfig {
    return JSON.parse(JSON.stringify(config)) as CrawlJobConfig;
  }

  get canCreateJobs(): boolean {
    return this.authService.hasMinimumRole('READ_WRITE');
  }

  get canDeleteJobs(): boolean {
    return this.authService.hasMinimumRole('READ_WRITE');
  }

  confirmDelete(job: CrawlJob): void {
    this.jobToDelete = job;
    this.showDeleteDialog = true;
  }

  cancelDelete(): void {
    this.showDeleteDialog = false;
    this.jobToDelete = null;
  }

  executeDelete(): void {
    if (!this.jobToDelete) return;
    this.deleting = true;
    this.crawlerApi.deleteJob(this.jobToDelete.id).pipe(
      takeUntil(this.destroy$)
    ).subscribe({
      next: () => {
        this.showDeleteDialog = false;
        this.jobToDelete = null;
        this.deleting = false;
        this.jobs = [];
        this.nextCursor = null;
        this.loadJobs();
      },
      error: (err) => {
        this.error = `Failed to delete job: ${err.message}`;
        this.deleting = false;
        this.showDeleteDialog = false;
      }
    });
  }

  get canManageUsers(): boolean {
    return this.authService.hasMinimumRole('ADMINISTRATOR');
  }

  get currentUserEmail(): string | undefined {
    return this.authService.email;
  }

  goToUsers(): void {
    this.router.navigate(['/users']);
  }

  goToMonitoring(): void {
    this.router.navigate(['/workers']);
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
}
