import { Component, OnInit } from '@angular/core';
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
import { forkJoin } from 'rxjs';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { CrawlJob, CrawlTask, FileType, JobExportFileType } from '../../core/models';

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
    DividerModule
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
              <h2 class="text-xl font-semibold">{{ job?.job_config?.name || 'Unnamed Job' }}</h2>
              <p class="text-sm text-gray-500">Job ID: {{ job?.id }}</p>
            </div>
          </ng-template>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <p class="text-sm text-gray-600">Status</p>
              <p-tag [value]="job?.status || ''" [severity]="getStatusSeverity(job?.status || '')" />
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
              <p-tag *ngFor="let seed of job?.job_config?.seeds" [value]="seed.url" severity="secondary" />
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
                  <p-tag *ngIf="pag.multiple" value="multiple" severity="info" styleClass="text-xs" />
                </div>
              </div>
            </div>
          </div>

          <p-divider />

          <div>
            <p class="text-sm text-gray-600">Export Results</p>
            <div class="flex flex-wrap items-center gap-2 mt-2">
              <p-tag [value]="job?.export_status || 'NOT_STARTED'" [severity]="getStatusSeverity(job?.export_status || 'NOT_STARTED')" />
              <span *ngIf="job?.exported_at" class="text-sm text-gray-500">
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
              <i class="pi" [ngClass]="configExpanded ? 'pi-chevron-up' : 'pi-chevron-down'" class="mr-2"></i>
              Job Config (auth hidden)
            </p-button>
            <div
              class="detail-wrapper mt-3"
              [@expandCollapse]="configExpanded ? 'expanded' : 'collapsed'"
              [attr.aria-hidden]="!configExpanded">
              <pre class="json-view" *ngIf="getJobConfigWithoutAuth(job) as config">{{ config | json }}</pre>
              <div class="text-gray-500" *ngIf="job && !job.job_config">No job configuration available.</div>
            </div>
          </div>
        </p-card>

        <!-- Tasks Table -->
        <p-card>
          <ng-template pTemplate="header">
            <div class="p-4 pb-0">
              <h2 class="text-xl font-semibold">Tasks ({{ tasks.length }})</h2>
            </div>
          </ng-template>

          <p-table
            [value]="tasks"
            [paginator]="true"
            [rows]="10"
            [rowsPerPageOptions]="[5, 10, 25]"
            [showFirstLastIcon]="true"
            [tableStyle]="{'min-width': '60rem'}">
            <ng-template pTemplate="header">
              <tr>
                <th>URL</th>
                <th>Status</th>
                <th>Depth</th>
                <th>Enqueued At</th>
                <th>Body Hash</th>
                <th>Files</th>
              </tr>
            </ng-template>
            <ng-template pTemplate="body" let-task>
              <tr>
                <td class="truncate max-w-md">
                  <a [href]="task.url" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">
                    {{ task.url }}
                  </a>
                </td>
                <td>
                  <p-tag [value]="task.status" [severity]="getStatusSeverity(task.status)" />
                </td>
                <td>{{ task.depth }}</td>
                <td>{{ task.enqueued_at | date:'short' }}</td>
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
                  </div>
                </td>
              </tr>
            </ng-template>
            <ng-template pTemplate="emptymessage">
              <tr>
                <td colspan="6" class="text-center p-8 text-gray-500">
                  <i class="pi pi-list text-6xl block mb-4"></i>
                  <p>No tasks found for this job.</p>
                </td>
              </tr>
            </ng-template>
          </p-table>
        </p-card>
      </div>
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
export class JobDetailsComponent implements OnInit {
  job: CrawlJob | null = null;
  tasks: CrawlTask[] = [];
  loading = false;
  error: string | null = null;
  configExpanded = false;
  loadingFile: { [key: string]: boolean } = {};

  constructor(
    private crawlerApi: CrawlerApiService,
    private route: ActivatedRoute,
    private router: Router
  ) {}

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

  downloadTaskFile(task: CrawlTask, fileType: FileType): void {
    const loadingKey = `${task.id}-${fileType}`;
    this.loadingFile[loadingKey] = true;

    this.crawlerApi.getTaskFileURL(task.id, fileType).subscribe({
      next: (response) => {
        fetch(response.url)
          .then(res => {
            if (!res.ok) {
              throw new Error(`HTTP error! status: ${res.status}`);
            }
            return res.blob();
          })
          .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;

            const extension = fileType === 'pages' ? 'html' : 'json';
            link.download = `task-${task.id}.${extension}`;

            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);

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

  downloadJobExport(fileType: JobExportFileType): void {
    if (!this.job) {
      return;
    }

    const loadingKey = `job-export-${fileType}`;
    this.loadingFile[loadingKey] = true;

    this.crawlerApi.getJobExportFileURL(this.job.id, fileType).subscribe({
      next: (response) => {
        fetch(response.url)
          .then(res => {
            if (!res.ok) {
              throw new Error(`HTTP error! status: ${res.status}`);
            }
            return res.blob();
          })
          .then(blob => {
            const url = window.URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = url;
            link.download = `job-${this.job?.id}-export.${fileType}`;

            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);

            window.URL.revokeObjectURL(url);
            this.loadingFile[loadingKey] = false;
          })
          .catch(err => {
            this.loadingFile[loadingKey] = false;
            console.error(`Failed to download export: ${err.message}`);
          });
      },
      error: (err) => {
        this.loadingFile[loadingKey] = false;
        console.error(`Failed to get export URL: ${err.message}`);
      }
    });
  }
}
