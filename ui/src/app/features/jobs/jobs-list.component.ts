import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { TagModule } from 'primeng/tag';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { CrawlerApiService, JobListFilter } from '../../core/services/api/crawler-api.service';
import { AuthService } from '../../core/services/auth.service';
import { CrawlJob } from '../../core/models';
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
    JobFiltersComponent
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-3xl font-bold">Crawl Jobs</h1>
          <p class="text-sm text-gray-500 mt-1">Track status, inspect configs, and launch new crawls.</p>
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
      <app-job-filters (filterChange)="onFilterChange($event)"></app-job-filters>

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
          [tableStyle]="{'min-width': '60rem'}">
          <ng-template #header>
            <tr>
              <th style="width: 3rem"></th>
              <th>Name</th>
              <th>Status</th>
              <th>Created At</th>
              <th style="width: 8rem"></th>
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
              <td>{{ job.job_config?.name || 'Unnamed Job' }}</td>
              <td>
                <p-tag [value]="job.status" [severity]="getStatusSeverity(job.status)" />
              </td>
              <td>{{ job.created_at | date:'short' }}</td>
              <td class="text-right">
                <p-button
                  [outlined]="true"
                  severity="secondary"
                  (onClick)="viewJob(job)">
                  <i class="pi pi-external-link"></i>
                </p-button>
              </td>
            </tr>
          </ng-template>
          <ng-template #expandedrow let-job>
            <tr>
              <td [attr.colspan]="5">
                <div class="detail-wrapper">
                  <div class="detail-header">
                    <div class="detail-title">Job Config (auth hidden)</div>
                  </div>
                  <div class="pagination-info mb-3" *ngIf="job.job_config?.extraction_spec?.pagination?.length">
                    <p class="text-sm font-semibold mb-2">Pagination Selectors</p>
                    <div class="flex flex-wrap gap-2">
                      <div *ngFor="let pag of job.job_config?.extraction_spec?.pagination" class="bg-white rounded px-3 py-1 text-sm border">
                        <span *ngIf="pag.name" class="font-medium mr-2">{{ pag.name }}:</span>
                        <code class="text-xs">{{ pag.selector }}</code>
                        <span class="text-gray-500 ml-2">({{ pag.attribute || 'href' }})</span>
                        <span *ngIf="pag.multiple" class="text-blue-600 ml-1">[multiple]</span>
                      </div>
                    </div>
                  </div>
                  <pre class="json-view" *ngIf="getJobConfigWithoutAuth(job) as config">{{ config | json }}</pre>
                  <div class="text-gray-500" *ngIf="!job.job_config">No job configuration available.</div>
                </div>
              </td>
            </tr>
          </ng-template>
          <ng-template #emptymessage>
            <tr>
              <td colspan="5" class="text-center p-8 text-gray-500">
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

  private nextCursor: string | null = null;
  private currentFilter: JobListFilter = {};
  private readonly pageSize = 20;

  constructor(
    private crawlerApi: CrawlerApiService,
    private router: Router,
    private authService: AuthService
  ) {}

  ngOnInit(): void {
    this.loadJobs();
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

  loadJobs(): void {
    this.loading = true;
    this.error = null;

    this.crawlerApi.listJobs({
      limit: this.pageSize,
      filter: this.currentFilter
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
      filter: this.currentFilter
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

  get canCreateJobs(): boolean {
    return this.authService.hasMinimumRole('READ_WRITE');
  }

  get canManageUsers(): boolean {
    return this.authService.hasMinimumRole('ADMINISTRATOR');
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
