import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
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
    MatCardModule
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
        <table mat-table [dataSource]="jobs" class="w-full">
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

          <tr mat-header-row *matHeaderRowDef="displayedColumns"></tr>
          <tr mat-row *matRowDef="let row; columns: displayedColumns;"
              class="cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800"
              (click)="viewJob(row)"></tr>
        </table>

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
  `]
})
export class JobsListComponent implements OnInit {
  jobs: CrawlJob[] = [];
  displayedColumns: string[] = ['name', 'status', 'created_at'];
  loading = false;
  error: string | null = null;

  constructor(
    private crawlerApi: CrawlerApiService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.loadJobs();
  }

  loadJobs(): void {
    this.loading = true;
    this.error = null;

    this.crawlerApi.listJobs().subscribe({
      next: (response) => {
        this.jobs = response.jobs;
        this.loading = false;
      },
      error: (err) => {
        this.error = `Failed to load jobs: ${err.message}`;
        this.loading = false;
      }
    });
  }

  viewJob(job: CrawlJob): void {
    this.router.navigate(['/jobs', job.id]);
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
