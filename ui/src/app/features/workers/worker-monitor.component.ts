import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatTableModule } from '@angular/material/table';
import { MatChipsModule } from '@angular/material/chips';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Subject, interval } from 'rxjs';
import { startWith, switchMap, takeUntil } from 'rxjs/operators';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { WorkerInfo } from '../../core/models';

@Component({
  selector: 'app-worker-monitor',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatTableModule,
    MatChipsModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-3xl font-bold">Worker Health</h1>
          <p class="text-sm text-gray-500 mt-1">Live heartbeats and capacity signals from the fleet</p>
        </div>
        <button mat-stroked-button color="primary" (click)="refresh()">
          <mat-icon>refresh</mat-icon>
          Refresh
        </button>
      </div>

      <div class="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <mat-card class="p-4">
          <div class="text-sm text-gray-500">Total Workers</div>
          <div class="text-2xl font-semibold mt-1">{{ workers.length }}</div>
        </mat-card>
        <mat-card class="p-4">
          <div class="text-sm text-gray-500">Active</div>
          <div class="text-2xl font-semibold mt-1 text-green-600">{{ countByStatus('ACTIVE') }}</div>
        </mat-card>
        <mat-card class="p-4">
          <div class="text-sm text-gray-500">Draining</div>
          <div class="text-2xl font-semibold mt-1 text-amber-600">{{ countByStatus('DRAINING') }}</div>
        </mat-card>
        <mat-card class="p-4">
          <div class="text-sm text-gray-500">Inactive</div>
          <div class="text-2xl font-semibold mt-1 text-red-600">{{ countByStatus('INACTIVE') }}</div>
        </mat-card>
      </div>

      <mat-card *ngIf="loading" class="text-center p-8">
        <mat-spinner class="mx-auto"></mat-spinner>
        <p class="mt-4">Fetching worker status...</p>
      </mat-card>

      <mat-card *ngIf="error && !loading" class="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </mat-card>

      <mat-card *ngIf="!loading">
        <table mat-table [dataSource]="workers" class="w-full">
          <ng-container matColumnDef="worker_id">
            <th mat-header-cell *matHeaderCellDef>Worker ID</th>
            <td mat-cell *matCellDef="let worker" class="font-mono text-xs">{{ worker.worker_id }}</td>
          </ng-container>

          <ng-container matColumnDef="worker_type">
            <th mat-header-cell *matHeaderCellDef>Type</th>
            <td mat-cell *matCellDef="let worker">{{ worker.worker_type || 'unknown' }}</td>
          </ng-container>

          <ng-container matColumnDef="status">
            <th mat-header-cell *matHeaderCellDef>Status</th>
            <td mat-cell *matCellDef="let worker">
              <mat-chip [class]="getStatusClass(worker.status)">
                {{ normalizeStatus(worker.status) }}
              </mat-chip>
            </td>
          </ng-container>

          <ng-container matColumnDef="active_tasks">
            <th mat-header-cell *matHeaderCellDef>Active Tasks</th>
            <td mat-cell *matCellDef="let worker">{{ worker.active_tasks }}</td>
          </ng-container>

          <ng-container matColumnDef="last_heartbeat_at">
            <th mat-header-cell *matHeaderCellDef>Last Heartbeat</th>
            <td mat-cell *matCellDef="let worker">
              {{ worker.last_heartbeat_at ? (worker.last_heartbeat_at | date:'short') : '—' }}
            </td>
          </ng-container>

          <ng-container matColumnDef="uptime">
            <th mat-header-cell *matHeaderCellDef>Uptime</th>
            <td mat-cell *matCellDef="let worker">{{ formatUptime(worker.uptime) }}</td>
          </ng-container>

          <tr mat-header-row *matHeaderRowDef="displayedColumns"></tr>
          <tr mat-row *matRowDef="let row; columns: displayedColumns;"></tr>
        </table>

        <div *ngIf="workers.length === 0 && !loading" class="text-center p-8 text-gray-500">
          <mat-icon class="text-6xl">insights</mat-icon>
          <p class="mt-4">No workers have reported heartbeats yet.</p>
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
export class WorkerMonitorComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  workers: WorkerInfo[] = [];
  displayedColumns: string[] = [
    'worker_id',
    'worker_type',
    'status',
    'active_tasks',
    'last_heartbeat_at',
    'uptime'
  ];
  loading = false;
  error: string | null = null;

  constructor(private crawlerApi: CrawlerApiService) {}

  ngOnInit(): void {
    this.startPolling();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  refresh(): void {
    this.fetchWorkers();
  }

  private startPolling(): void {
    this.loading = true;
    interval(5000).pipe(
      startWith(0),
      switchMap(() => this.crawlerApi.listWorkers()),
      takeUntil(this.destroy$)
    ).subscribe({
      next: (response) => {
        this.workers = response.workers || [];
        this.loading = false;
        this.error = null;
      },
      error: (err) => {
        this.error = `Failed to load workers: ${err.message}`;
        this.loading = false;
      }
    });
  }

  private fetchWorkers(): void {
    this.loading = true;
    this.crawlerApi.listWorkers().pipe(takeUntil(this.destroy$)).subscribe({
      next: (response) => {
        this.workers = response.workers || [];
        this.loading = false;
        this.error = null;
      },
      error: (err) => {
        this.error = `Failed to load workers: ${err.message}`;
        this.loading = false;
      }
    });
  }

  countByStatus(status: string): number {
    return this.workers.filter(worker => this.normalizeStatus(worker.status).toUpperCase() === status).length;
  }

  getStatusClass(status: string): string {
    switch (this.normalizeStatus(status).toUpperCase()) {
      case 'ACTIVE':
        return 'bg-green-100 text-green-800';
      case 'DRAINING':
        return 'bg-amber-100 text-amber-800';
      case 'INACTIVE':
        return 'bg-red-100 text-red-800';
      case 'DEAD':
        return 'bg-gray-900 text-white';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  }

  normalizeStatus(status?: string): string {
    if (!status) {
      return 'UNKNOWN';
    }

    if (status.startsWith('WORKER_STATUS_')) {
      return status.replace('WORKER_STATUS_', '');
    }

    return status;
  }

  formatUptime(uptime?: string): string {
    if (!uptime) {
      return '—';
    }

    const match = uptime.match(/([\d.]+)s/);
    if (!match) {
      return uptime;
    }

    const totalSeconds = Math.floor(parseFloat(match[1]));
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    const parts: string[] = [];
    if (hours > 0) parts.push(`${hours}h`);
    if (minutes > 0 || hours > 0) parts.push(`${minutes}m`);
    parts.push(`${seconds}s`);

    return parts.join(' ');
  }
}
