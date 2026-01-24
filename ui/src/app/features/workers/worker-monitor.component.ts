import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { ButtonModule } from 'primeng/button';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { Subject, interval } from 'rxjs';
import { startWith, switchMap, takeUntil } from 'rxjs/operators';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { WorkerInfo } from '../../core/models';

@Component({
  selector: 'app-worker-monitor',
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    TableModule,
    TagModule,
    ButtonModule,
    ProgressSpinnerModule
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-3xl font-bold">Worker Health</h1>
          <p class="text-sm text-gray-500 mt-1">Live heartbeats and capacity signals from the fleet</p>
        </div>
        <div class="flex items-center gap-3">
          <p-button
            [outlined]="true"
            severity="secondary"
            (onClick)="goBack()">
            <i class="pi pi-arrow-left mr-2"></i>
            Back to Jobs
          </p-button>
          <p-button
            [outlined]="true"
            severity="secondary"
            (onClick)="refresh()">
            <i class="pi pi-refresh mr-2"></i>
            Refresh
          </p-button>
        </div>
      </div>

      <div class="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <p-card styleClass="p-4">
          <div class="text-sm text-gray-500">Total Workers</div>
          <div class="text-2xl font-semibold mt-1">{{ workers.length }}</div>
        </p-card>
        <p-card styleClass="p-4">
          <div class="text-sm text-gray-500">Active</div>
          <div class="text-2xl font-semibold mt-1 text-green-600">{{ countByStatus('ACTIVE') }}</div>
        </p-card>
        <p-card styleClass="p-4">
          <div class="text-sm text-gray-500">Draining</div>
          <div class="text-2xl font-semibold mt-1 text-amber-600">{{ countByStatus('DRAINING') }}</div>
        </p-card>
        <p-card styleClass="p-4">
          <div class="text-sm text-gray-500">Inactive</div>
          <div class="text-2xl font-semibold mt-1 text-red-600">{{ countByStatus('INACTIVE') }}</div>
        </p-card>
      </div>

      <p-card *ngIf="loading" styleClass="text-center p-8">
        <p-progressSpinner />
        <p class="mt-4">Fetching worker status...</p>
      </p-card>

      <p-card *ngIf="error && !loading" styleClass="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </p-card>

      <p-card *ngIf="!loading">
        <p-table [value]="workers" [tableStyle]="{'min-width': '60rem'}">
          <ng-template pTemplate="header">
            <tr>
              <th>Worker ID</th>
              <th>Type</th>
              <th>Status</th>
              <th>Last Heartbeat</th>
              <th>Uptime</th>
            </tr>
          </ng-template>
          <ng-template pTemplate="body" let-worker>
            <tr>
              <td class="font-mono text-xs">{{ worker.worker_id }}</td>
              <td>{{ worker.worker_type || 'unknown' }}</td>
              <td>
                <p-tag
                  [value]="normalizeStatus(worker.status)"
                  [severity]="getStatusSeverity(worker.status)" />
              </td>
              <td>{{ worker.last_heartbeat_at ? (worker.last_heartbeat_at | date:'short') : '—' }}</td>
              <td>{{ formatUptime(worker.uptime) }}</td>
            </tr>
          </ng-template>
          <ng-template pTemplate="emptymessage">
            <tr>
              <td colspan="5" class="text-center p-8 text-gray-500">
                <i class="pi pi-chart-line text-6xl block mb-4"></i>
                <p>No workers have reported heartbeats yet.</p>
              </td>
            </tr>
          </ng-template>
        </p-table>
      </p-card>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class WorkerMonitorComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  workers: WorkerInfo[] = [];
  loading = false;
  error: string | null = null;

  constructor(
    private crawlerApi: CrawlerApiService,
    private router: Router
  ) {}

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

  goBack(): void {
    this.router.navigate(['/jobs']);
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

  getStatusSeverity(status: string): 'success' | 'info' | 'warn' | 'danger' | 'secondary' {
    switch (this.normalizeStatus(status).toUpperCase()) {
      case 'ACTIVE':
        return 'success';
      case 'DRAINING':
        return 'warn';
      case 'INACTIVE':
      case 'DEAD':
        return 'danger';
      default:
        return 'secondary';
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
