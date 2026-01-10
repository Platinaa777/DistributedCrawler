import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { forkJoin } from 'rxjs';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { CrawlJob, CrawlTask } from '../../core/models';

@Component({
  selector: 'app-job-details',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatTableModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatIconModule
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
          </mat-card-content>
        </mat-card>

        <!-- Tasks Table -->
        <mat-card>
          <mat-card-header>
            <mat-card-title>Tasks ({{ tasks.length }})</mat-card-title>
          </mat-card-header>
          <mat-card-content>
            <table mat-table [dataSource]="tasks" class="w-full">
              <!-- URL Column -->
              <ng-container matColumnDef="url">
                <th mat-header-cell *matHeaderCellDef>URL</th>
                <td mat-cell *matCellDef="let task" class="truncate max-w-md">{{ task.url }}</td>
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

              <tr mat-header-row *matHeaderRowDef="taskColumns"></tr>
              <tr mat-row *matRowDef="let row; columns: taskColumns;"></tr>
            </table>

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
  `]
})
export class JobDetailsComponent implements OnInit {
  job: CrawlJob | null = null;
  tasks: CrawlTask[] = [];
  taskColumns: string[] = ['url', 'status', 'depth', 'enqueued_at'];
  loading = false;
  error: string | null = null;

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
}
