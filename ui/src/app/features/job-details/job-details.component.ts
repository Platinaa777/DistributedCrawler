import { AfterViewInit, Component, OnInit, ViewChild } from '@angular/core';
import { animate, state, style, transition, trigger } from '@angular/animations';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { MatPaginator, MatPaginatorModule } from '@angular/material/paginator';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { forkJoin } from 'rxjs';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { CrawlJob, CrawlTask, FileType } from '../../core/models';

@Component({
  selector: 'app-job-details',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatTableModule,
    MatPaginatorModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatIconModule,
    MatTooltipModule
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="mb-4">
        <button mat-button (click)="goBack()">
          <mat-icon>arrow_back</mat-icon>
          Back to Jobs
        </button>
      </div>

      <mat-card *ngIf="loading" class="text-center p-8">
        <mat-spinner class="mx-auto"></mat-spinner>
        <p class="mt-4">Loading job details...</p>
      </mat-card>

      <mat-card *ngIf="error && !loading" class="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </mat-card>

      <div *ngIf="!loading && !error">
        <!-- Job Info Card -->
        <mat-card class="mb-6">
          <mat-card-header>
            <mat-card-title>{{ job?.job_config?.name || 'Unnamed Job' }}</mat-card-title>
            <mat-card-subtitle>Job ID: {{ job?.id }}</mat-card-subtitle>
          </mat-card-header>
          <mat-card-content>
            <div class="grid grid-cols-2 gap-4 mt-4">
              <div>
                <p class="text-sm text-gray-600">Status</p>
                <mat-chip [class]="getStatusClass(job?.status || '')">
                  {{ job?.status }}
                </mat-chip>
              </div>
              <div>
                <p class="text-sm text-gray-600">Created At</p>
                <p>{{ job?.created_at | date:'medium' }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-600">Max Depth</p>
                <p>{{ job?.job_config?.scopes?.max_depth || 'N/A' }}</p>
              </div>
              <div>
                <p class="text-sm text-gray-600">Rate Limit (RPS)</p>
                <p>{{ job?.job_config?.rate_limit?.rps || 'N/A' }}</p>
              </div>
            </div>

            <div class="mt-4" *ngIf="job?.job_config?.seeds">
              <p class="text-sm text-gray-600">Seed URLs</p>
              <div class="flex flex-wrap gap-2 mt-2">
                <mat-chip *ngFor="let seed of job?.job_config?.seeds">
                  {{ seed.url }}
                </mat-chip>
              </div>
            </div>

            <div class="mt-4" *ngIf="job?.job_config?.extraction_spec?.pagination?.length">
              <p class="text-sm text-gray-600">Pagination Selectors</p>
              <div class="mt-2 space-y-2">
                <div *ngFor="let pag of job?.job_config?.extraction_spec?.pagination" class="bg-gray-50 rounded p-2 text-sm">
                  <div class="flex items-center gap-4">
                    <span *ngIf="pag.name" class="font-medium">{{ pag.name }}</span>
                    <code class="bg-gray-200 px-2 py-1 rounded text-xs">{{ pag.selector }}</code>
                    <span class="text-gray-500">attr: {{ pag.attribute || 'href' }}</span>
                    <mat-chip *ngIf="pag.multiple" class="text-xs">multiple</mat-chip>
                  </div>
                </div>
              </div>
            </div>

            <div class="mt-6">
              <button mat-stroked-button color="primary" (click)="toggleConfig()">
                <mat-icon>{{ configExpanded ? 'expand_less' : 'expand_more' }}</mat-icon>
                Job Config (auth hidden)
              </button>
              <div
                class="detail-wrapper mt-3"
                [@expandCollapse]="configExpanded ? 'expanded' : 'collapsed'"
                [attr.aria-hidden]="!configExpanded">
                <pre class="json-view" *ngIf="getJobConfigWithoutAuth(job) as config">{{ config | json }}</pre>
                <div class="text-gray-500" *ngIf="job && !job.job_config">No job configuration available.</div>
              </div>
            </div>
          </mat-card-content>
        </mat-card>

        <!-- Tasks Table -->
        <mat-card>
          <mat-card-header>
            <mat-card-title>Tasks ({{ tasks.length }})</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <table mat-table [dataSource]="dataSource" class="w-full">
              <!-- URL Column -->
              <ng-container matColumnDef="url">
                <th mat-header-cell *matHeaderCellDef>URL</th>
                <td mat-cell *matCellDef="let task" class="truncate max-w-md">
                  <a [href]="task.url" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">
                    {{ task.url }}
                  </a>
                </td>
              </ng-container>

              <!-- Status Column -->
              <ng-container matColumnDef="status">
                <th mat-header-cell *matHeaderCellDef>Status</th>
                <td mat-cell *matCellDef="let task">
                  <mat-chip [class]="getStatusClass(task.status)">
                    {{ task.status }}
                  </mat-chip>
                </td>
              </ng-container>

              <!-- Depth Column -->
              <ng-container matColumnDef="depth">
                <th mat-header-cell *matHeaderCellDef>Depth</th>
                <td mat-cell *matCellDef="let task">{{ task.depth }}</td>
              </ng-container>

              <!-- Enqueued At Column -->
              <ng-container matColumnDef="enqueued_at">
                <th mat-header-cell *matHeaderCellDef>Enqueued At</th>
                <td mat-cell *matCellDef="let task">{{ task.enqueued_at | date:'short' }}</td>
              </ng-container>

              <!-- Body Hash Column -->
              <ng-container matColumnDef="body_hash">
                <th mat-header-cell *matHeaderCellDef>Body Hash</th>
                <td mat-cell *matCellDef="let task" class="font-mono text-xs break-all">
                  {{ task.body_hash ? (task.body_hash | slice:0:8) + '...' : '' }}
                </td>
              </ng-container>

              <!-- Actions Column -->
              <ng-container matColumnDef="actions">
                <th mat-header-cell *matHeaderCellDef>Files</th>
                <td mat-cell *matCellDef="let task">
                  <div class="flex gap-1">
                    <button
                      mat-icon-button
                      color="primary"
                      [disabled]="!task.minio_object_key || loadingFile[task.id + '-pages']"
                      [matTooltip]="task.minio_object_key ? 'Download HTML page' : 'HTML not available'"
                      (click)="downloadTaskFile(task, 'pages')">
                      <mat-icon *ngIf="!loadingFile[task.id + '-pages']">download</mat-icon>
                      <mat-spinner *ngIf="loadingFile[task.id + '-pages']" diameter="20"></mat-spinner>
                    </button>
                    <button
                      mat-icon-button
                      color="accent"
                      [disabled]="!task.result_object_key || loadingFile[task.id + '-result']"
                      [matTooltip]="task.result_object_key ? 'Download JSON result' : 'Result not available'"
                      (click)="downloadTaskFile(task, 'result')">
                      <mat-icon *ngIf="!loadingFile[task.id + '-result']">download</mat-icon>
                      <mat-spinner *ngIf="loadingFile[task.id + '-result']" diameter="20"></mat-spinner>
                    </button>
                  </div>
                </td>
              </ng-container>

              <tr mat-header-row *matHeaderRowDef="taskColumns"></tr>
              <tr mat-row *matRowDef="let row; columns: taskColumns;"></tr>
            </table>

            <mat-paginator
              [pageSizeOptions]="[5, 10, 25]"
              showFirstLastButtons
              [pageSize]="10">
            </mat-paginator>

            <div *ngIf="tasks.length === 0" class="text-center p-8 text-gray-500">
              <mat-icon class="text-6xl">list_alt</mat-icon>
              <p class="mt-4">No tasks found for this job.</p>
            </div>
          </mat-card-content>
        </mat-card>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    table {
      width: 100%;
    }

    .detail-wrapper {
      padding: 16px 24px;
      background: #f8fafc;
      border-top: 1px solid #e5e7eb;
      overflow: hidden;
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
export class JobDetailsComponent implements OnInit, AfterViewInit {
  job: CrawlJob | null = null;
  tasks: CrawlTask[] = [];
  taskColumns: string[] = ['url', 'status', 'depth', 'enqueued_at', 'body_hash', 'actions'];
  dataSource = new MatTableDataSource<CrawlTask>([]);
  loading = false;
  error: string | null = null;
  configExpanded = false;
  loadingFile: { [key: string]: boolean } = {};
  @ViewChild(MatPaginator) paginator!: MatPaginator;

  constructor(
    private crawlerApi: CrawlerApiService,
    private route: ActivatedRoute,
    private router: Router
  ) {}

  ngAfterViewInit(): void {
    if (this.paginator) {
      this.dataSource.paginator = this.paginator;
    }
  }

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.loadJobDetails(id);
    }
  }

  loadJobDetails(id: string): void {
    this.loading = true;
    this.error = null;

    forkJoin({
      job: this.crawlerApi.getJob(id),
      tasks: this.crawlerApi.listTasksByJob(id)
    }).subscribe({
      next: (response) => {
        this.job = response.job.job;
        this.tasks = response.tasks.tasks;
        this.dataSource.data = response.tasks.tasks;
        if (this.paginator) {
          this.dataSource.paginator = this.paginator;
        }
        this.loading = false;
      },
      error: (err) => {
        this.error = `Failed to load job details: ${err.message}`;
        this.loading = false;
      }
    });
  }

  goBack(): void {
    this.router.navigate(['/jobs']);
  }

  toggleConfig(): void {
    this.configExpanded = !this.configExpanded;
  }

  getJobConfigWithoutAuth(job: CrawlJob | null) {
    const config = job?.job_config;
    if (!config) {
      return null;
    }

    const { auth, ...safeConfig } = config;
    return safeConfig;
  }

  getStatusClass(status: string): string {
    switch (status.toLowerCase()) {
      case 'completed':
        return 'bg-green-100 text-green-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      case 'inprogress':
      case 'queued':
        return 'bg-blue-100 text-blue-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  }

  downloadTaskFile(task: CrawlTask, fileType: FileType): void {
    const loadingKey = `${task.id}-${fileType}`;
    this.loadingFile[loadingKey] = true;

    this.crawlerApi.getTaskFileURL(task.id, fileType).subscribe({
      next: (response) => {
        // Fetch the file content from the presigned URL
        fetch(response.url)
          .then(res => {
            if (!res.ok) {
              throw new Error(`HTTP error! status: ${res.status}`);
            }
            return res.blob();
          })
          .then(blob => {
            // Create download link
            const url = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;

            // Set filename based on file type
            const extension = fileType === 'pages' ? 'html' : 'json';
            link.download = `task-${task.id}.${extension}`;

            // Trigger download
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);

            // Cleanup
            window.URL.revokeObjectURL(url);
            this.loadingFile[loadingKey] = false;
          })
          .catch(err => {
            this.loadingFile[loadingKey] = false;
            console.error(`Failed to download file: ${err.message}`);
          });
      },
      error: (err) => {
        this.loadingFile[loadingKey] = false;
        console.error(`Failed to get file URL: ${err.message}`);
      }
    });
  }
}
