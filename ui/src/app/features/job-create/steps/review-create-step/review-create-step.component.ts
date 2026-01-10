import { Component, OnInit, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { CrawlerApiService } from '../../../../core/services/api/crawler-api.service';
import { CrawlJobConfig } from '../../../../core/models/crawl-job.model';

@Component({
  selector: 'app-review-create-step',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    MatChipsModule,
    MatDividerModule
  ],
  template: `
    <div class="space-y-4">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Step 5: Review & Create</mat-card-title>
          <mat-card-subtitle>
            Review your job configuration before creating
          </mat-card-subtitle>
        </mat-card-header>
      </mat-card>

      <!-- Job Settings Summary -->
      <mat-card>
        <mat-card-header>
          <mat-card-title class="text-base">Job Settings</mat-card-title>
        </mat-card-header>
        <mat-card-content>
          <div class="space-y-3">
            <div>
              <p class="text-xs text-gray-600 font-semibold">Name</p>
              <p class="text-sm">{{ jobConfig.name || '(Not set)' }}</p>
            </div>

            <mat-divider></mat-divider>

            <div>
              <p class="text-xs text-gray-600 font-semibold">Seed URLs ({{ jobConfig.seeds.length }})</p>
              <div class="flex flex-wrap gap-1 mt-1">
                <mat-chip *ngFor="let seed of jobConfig.seeds" class="text-xs">
                  {{ seed.url }}
                </mat-chip>
              </div>
            </div>

            <mat-divider></mat-divider>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <p class="text-xs text-gray-600 font-semibold">Max Depth</p>
                <p class="text-sm">{{ jobConfig.scopes.max_depth }}</p>
              </div>
              <div>
                <p class="text-xs text-gray-600 font-semibold">Rate Limit (RPS)</p>
                <p class="text-sm">{{ jobConfig.rate_limit.rps }}</p>
              </div>
            </div>

            <mat-divider></mat-divider>

            <div *ngIf="jobConfig.scopes.allowed_domains.length > 0">
              <p class="text-xs text-gray-600 font-semibold">Allowed Domains ({{ jobConfig.scopes.allowed_domains.length }})</p>
              <div class="flex flex-wrap gap-1 mt-1">
                <mat-chip *ngFor="let domain of jobConfig.scopes.allowed_domains" class="text-xs">
                  {{ domain }}
                </mat-chip>
              </div>
            </div>
          </div>
        </mat-card-content>
      </mat-card>

      <!-- Extraction Spec Summary -->
      <mat-card>
        <mat-card-header>
          <mat-card-title class="text-base">Extraction Specification</mat-card-title>
        </mat-card-header>
        <mat-card-content>
          <div class="space-y-3">
            <div>
              <p class="text-xs text-gray-600 font-semibold mb-2">Fields ({{ jobConfig.extraction_spec.fields.length }})</p>
              <div class="space-y-2">
                <div
                  *ngFor="let field of jobConfig.extraction_spec.fields"
                  class="p-3 bg-gray-50 rounded border border-gray-200"
                >
                  <div class="flex items-center gap-2 mb-1">
                    <mat-chip class="text-xs">{{ field.name }}</mat-chip>
                    <mat-chip class="text-xs" color="primary">{{ field.type }}</mat-chip>
                    <mat-chip *ngIf="field.required" class="text-xs bg-red-100 text-red-800">Required</mat-chip>
                  </div>
                  <p class="text-xs font-mono text-gray-600 mt-1">{{ field.extractor.selector }}</p>
                  <p class="text-xs text-gray-500">
                    Attribute: <strong>{{ field.extractor.attribute }}</strong>
                    <span *ngIf="field.extractor.multiple"> • Multiple values</span>
                  </p>
                  <div *ngIf="field.transforms && field.transforms.length > 0" class="mt-1">
                    <p class="text-xs text-gray-500">
                      Transforms: {{ getTransformsList(field.transforms) }}
                    </p>
                  </div>
                </div>
              </div>

              <div *ngIf="jobConfig.extraction_spec.fields.length === 0" class="text-center py-4 text-gray-400 text-sm">
                No fields configured
              </div>
            </div>

            <mat-divider *ngIf="jobConfig.extraction_spec.metrics.length > 0"></mat-divider>

            <div *ngIf="jobConfig.extraction_spec.metrics.length > 0">
              <p class="text-xs text-gray-600 font-semibold mb-2">Metrics ({{ jobConfig.extraction_spec.metrics.length }})</p>
              <div class="space-y-2">
                <div
                  *ngFor="let metric of jobConfig.extraction_spec.metrics"
                  class="p-2 bg-gray-50 rounded border border-gray-200"
                >
                  <div class="flex items-center gap-2">
                    <mat-chip class="text-xs">{{ metric.name }}</mat-chip>
                    <mat-chip class="text-xs" color="accent">{{ metric.op }}</mat-chip>
                  </div>
                  <p class="text-xs text-gray-500 mt-1">Input: <strong>{{ metric.input }}</strong></p>
                </div>
              </div>
            </div>
          </div>
        </mat-card-content>
      </mat-card>

      <!-- Actions -->
      <mat-card>
        <mat-card-content>
          <div class="flex items-center justify-between">
            <div *ngIf="error" class="text-sm text-red-600 flex items-center gap-2">
              <mat-icon>error</mat-icon>
              {{ error }}
            </div>

            <div class="flex-1"></div>

            <div class="flex items-center gap-4">
              <mat-spinner *ngIf="creating" diameter="24"></mat-spinner>

              <button
                mat-raised-button
                color="primary"
                (click)="createJob()"
                [disabled]="creating || !isValid()"
                class="ml-auto"
              >
                <mat-icon>rocket_launch</mat-icon>
                Create Job
              </button>
            </div>
          </div>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class ReviewCreateStepComponent implements OnInit {
  @Output() jobCreated = new EventEmitter<string>();

  jobConfig!: CrawlJobConfig;
  creating = false;
  error: string | null = null;

  constructor(
    private stateService: JobCreateStateService,
    private crawlerApi: CrawlerApiService,
    private router: Router,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit(): void {
    this.jobConfig = this.stateService.buildJobConfig();
  }

  createJob(): void {
    this.creating = true;
    this.error = null;

    this.crawlerApi.createJob(this.jobConfig).subscribe({
      next: (response) => {
        this.creating = false;
        const jobId = response.id;

        // Show success message
        this.snackBar.open('Job created successfully!', 'Close', {
          duration: 3000,
          horizontalPosition: 'end',
          verticalPosition: 'top'
        });

        // Reset wizard state
        this.stateService.reset();

        // Emit event and navigate
        this.jobCreated.emit(jobId);
        this.router.navigate(['/jobs', jobId]);
      },
      error: (err) => {
        this.creating = false;
        this.error = err.message || 'Failed to create job';

        this.snackBar.open(`Error: ${this.error}`, 'Close', {
          duration: 5000,
          horizontalPosition: 'end',
          verticalPosition: 'top',
          panelClass: ['error-snackbar']
        });
      }
    });
  }

  isValid(): boolean {
    return (
      this.jobConfig.name.trim() !== '' &&
      this.jobConfig.seeds.length > 0 &&
      this.jobConfig.extraction_spec.fields.length > 0
    );
  }

  getTransformsList(transforms: any[]): string {
    return transforms.map(t => t.op).join(', ');
  }
}
