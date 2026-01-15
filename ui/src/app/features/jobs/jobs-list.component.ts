import { Component, OnInit, AfterViewInit, ViewChild } from '@angular/core';
import { animate, state, style, transition, trigger } from '@angular/animations';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatPaginator, MatPaginatorModule } from '@angular/material/paginator';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { CrawlJob } from '../../core/models';

@Component({
  selector: 'app-jobs-list',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatIconModule,
    MatCardModule,
    MatPaginatorModule
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold">Crawl Jobs</h1>
        <button mat-raised-button color="primary" (click)="createJob()">
          <mat-icon>add</mat-icon>
          Create Job
        </button>
      </div>

      <mat-card *ngIf="loading" class="text-center p-8">
        <mat-spinner class="mx-auto"></mat-spinner>
        <p class="mt-4">Loading jobs...</p>
      </mat-card>

      <mat-card *ngIf="error && !loading" class="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </mat-card>

      <mat-card *ngIf="!loading && !error">
        <table mat-table [dataSource]="dataSource" class="w-full" multiTemplateDataRows>
          <!-- Expand Toggle Column -->
          <ng-container matColumnDef="expand">
            <th mat-header-cell *matHeaderCellDef></th>
            <td mat-cell *matCellDef="let job">
              <button mat-icon-button aria-label="Toggle details"
                      [attr.aria-expanded]="expandedJobId === job.id"
                      (click)="toggleExpand(job, $event)">
                <mat-icon>{{ expandedJobId === job.id ? 'expand_less' : 'expand_more' }}</mat-icon>
              </button>
            </td>
          </ng-container>

          <!-- Name Column -->
          <ng-container matColumnDef="name">
            <th mat-header-cell *matHeaderCellDef>Name</th>
            <td mat-cell *matCellDef="let job">{{ job.job_config?.name || 'Unnamed Job' }}</td>
          </ng-container>

          <!-- Status Column -->
          <ng-container matColumnDef="status">
            <th mat-header-cell *matHeaderCellDef>Status</th>
            <td mat-cell *matCellDef="let job">
              <mat-chip [class]="getStatusClass(job.status)">
                {{ job.status }}
              </mat-chip>
            </td>
          </ng-container>

          <!-- Created At Column -->
          <ng-container matColumnDef="created_at">
            <th mat-header-cell *matHeaderCellDef>Created At</th>
            <td mat-cell *matCellDef="let job">{{ job.created_at | date:'short' }}</td>
          </ng-container>

          <!-- Detail Row -->
          <ng-container matColumnDef="detail">
            <td mat-cell *matCellDef="let job" [attr.colspan]="displayedColumns.length">
              <div
                class="detail-wrapper"
                [@expandCollapse]="expandedJobId === job.id ? 'expanded' : 'collapsed'"
                [attr.aria-hidden]="expandedJobId !== job.id">
                <div class="detail-header">
                  <div class="detail-title">Job Config (auth hidden)</div>
                  <button mat-stroked-button color="primary" (click)="viewJob(job)">
                    <mat-icon>open_in_new</mat-icon>
                    Open Job
                  </button>
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
          </ng-container>

          <tr mat-header-row *matHeaderRowDef="displayedColumns"></tr>
          <tr mat-row *matRowDef="let row; columns: displayedColumns;"
              class="cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800"
              (click)="toggleExpand(row)"></tr>
          <tr mat-row *matRowDef="let row; columns: detailColumns"
              class="detail-row"
              [class.detail-open]="expandedJobId === row.id">
          </tr>
        </table>
        <mat-paginator
          [length]="jobs.length"
          [pageSize]="10"
          [pageSizeOptions]="[5, 10, 25, 50]"
          showFirstLastButtons>
        </mat-paginator>

        <div *ngIf="jobs.length === 0" class="text-center p-8 text-gray-500">
          <mat-icon class="text-6xl">work_outline</mat-icon>
          <p class="mt-4">No jobs found. Create your first crawl job!</p>
        </div>
      </mat-card>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    table {
      width: 100%;
    }

    .detail-row td {
      padding: 0;
      border: none;
    }

    .detail-wrapper {
      padding: 16px 24px;
      background: #f8fafc;
      border-top: 1px solid #e5e7eb;
      overflow: hidden;
    }

    .detail-row {
      height: 0 !important;
      min-height: 0 !important;
    }

    .detail-row.detail-open {
      height: auto !important;
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
export class JobsListComponent implements OnInit, AfterViewInit {
  @ViewChild(MatPaginator) paginator?: MatPaginator;

  jobs: CrawlJob[] = [];
  dataSource = new MatTableDataSource<CrawlJob>([]);
  displayedColumns: string[] = ['expand', 'name', 'status', 'created_at'];
  detailColumns: string[] = ['detail'];
  loading = false;
  error: string | null = null;
  expandedJobId: string | null = null;

  constructor(
    private crawlerApi: CrawlerApiService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.loadJobs();
  }

  ngAfterViewInit(): void {
    if (this.paginator) {
      this.dataSource.paginator = this.paginator;
    }
  }

  loadJobs(): void {
    this.loading = true;
    this.error = null;

    this.crawlerApi.listJobs().subscribe({
      next: (response) => {
        this.jobs = response.jobs;
        this.dataSource.data = response.jobs;
        if (this.paginator) {
          this.dataSource.paginator = this.paginator;
        }
        this.loading = false;
      },
      error: (err) => {
        this.error = `Failed to load jobs: ${err.message}`;
        this.loading = false;
      }
    });
  }

  toggleExpand(job: CrawlJob, event?: Event): void {
    event?.stopPropagation();
    this.expandedJobId = this.expandedJobId === job.id ? null : job.id;
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

  getStatusClass(status: string): string {
    switch (status.toLowerCase()) {
      case 'completed':
        return 'bg-green-100 text-green-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      case 'inprogress':
        return 'bg-blue-100 text-blue-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  }
}
