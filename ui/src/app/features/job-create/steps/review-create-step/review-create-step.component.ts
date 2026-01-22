import { Component, OnInit, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { ToastModule } from 'primeng/toast';
import { TagModule } from 'primeng/tag';
import { DividerModule } from 'primeng/divider';
import { MessageService } from 'primeng/api';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { CrawlerApiService } from '../../../../core/services/api/crawler-api.service';
import { CrawlJobConfig } from '../../../../core/models/crawl-job.model';

@Component({
  selector: 'app-review-create-step',
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    ButtonModule,
    ProgressSpinnerModule,
    ToastModule,
    TagModule,
    DividerModule
  ],
  providers: [MessageService],
  template: `
    <p-toast position="top-right" />

    <div class="space-y-4">
      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h2 class="text-xl font-semibold">Step 4: Review & Create</h2>
            <p class="text-sm text-gray-500">Review your job configuration before creating</p>
          </div>
        </ng-template>
      </p-card>

      <!-- Job Settings Summary -->
      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h3 class="text-base font-semibold">Job Settings</h3>
          </div>
        </ng-template>

        <div class="space-y-3">
          <div>
            <p class="text-xs text-gray-600 font-semibold">Name</p>
            <p class="text-sm">{{ jobConfig.name || '(Not set)' }}</p>
          </div>

          <p-divider />

          <div>
            <p class="text-xs text-gray-600 font-semibold">Seed URLs ({{ jobConfig.seeds.length }})</p>
            <div class="flex flex-wrap gap-1 mt-1">
              <p-tag *ngFor="let seed of jobConfig.seeds" [value]="seed.url" severity="secondary" styleClass="text-xs" />
            </div>
          </div>

          <p-divider />

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

          <p-divider *ngIf="jobConfig.scopes.allowed_domains.length > 0" />

          <div *ngIf="jobConfig.scopes.allowed_domains.length > 0">
            <p class="text-xs text-gray-600 font-semibold">Allowed Domains ({{ jobConfig.scopes.allowed_domains.length }})</p>
            <div class="flex flex-wrap gap-1 mt-1">
              <p-tag *ngFor="let domain of jobConfig.scopes.allowed_domains" [value]="domain" severity="secondary" styleClass="text-xs" />
            </div>
          </div>
        </div>
      </p-card>

      <!-- Extraction Spec Summary -->
      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h3 class="text-base font-semibold">Extraction Specification</h3>
          </div>
        </ng-template>

        <div class="space-y-3">
          <div>
            <p class="text-xs text-gray-600 font-semibold mb-2">Fields ({{ jobConfig.extraction_spec.fields.length }})</p>
            <div class="space-y-2">
              <div
                *ngFor="let field of jobConfig.extraction_spec.fields"
                class="p-3 bg-gray-50 rounded border border-gray-200">
                <div class="flex items-center gap-2 mb-1">
                  <p-tag [value]="field.name" severity="secondary" styleClass="text-xs" />
                  <p-tag [value]="field.type" severity="info" styleClass="text-xs" />
                  <p-tag *ngIf="field.required" value="Required" severity="danger" styleClass="text-xs" />
                </div>
                <p class="text-xs font-mono text-gray-600 mt-1">{{ field.extractor.selector }}</p>
                <p class="text-xs text-gray-500">
                  Attribute: <strong>{{ field.extractor.attribute }}</strong>
                  <span *ngIf="field.extractor.multiple"> - Multiple values</span>
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

          <p-divider *ngIf="jobConfig.extraction_spec.metrics.length > 0" />

          <div *ngIf="jobConfig.extraction_spec.metrics.length > 0">
            <p class="text-xs text-gray-600 font-semibold mb-2">Metrics ({{ jobConfig.extraction_spec.metrics.length }})</p>
            <div class="space-y-2">
              <div
                *ngFor="let metric of jobConfig.extraction_spec.metrics"
                class="p-2 bg-gray-50 rounded border border-gray-200">
                <div class="flex items-center gap-2">
                  <p-tag [value]="metric.name" severity="secondary" styleClass="text-xs" />
                  <p-tag [value]="metric.op" severity="help" styleClass="text-xs" />
                </div>
                <p class="text-xs text-gray-500 mt-1">Input: <strong>{{ metric.input }}</strong></p>
              </div>
            </div>
          </div>
        </div>
      </p-card>

      <!-- Actions -->
      <p-card>
        <div class="flex items-center justify-between">
          <div *ngIf="error" class="text-sm text-red-600 flex items-center gap-2">
            <i class="pi pi-times-circle"></i>
            {{ error }}
          </div>

          <div class="flex-1"></div>

          <div class="flex items-center gap-4">
            <p-progressSpinner *ngIf="creating" [style]="{width: '24px', height: '24px'}" />

            <p-button
              (onClick)="createJob()"
              [disabled]="creating || !isValid()"
              styleClass="ml-auto">
              <i class="pi pi-send mr-2"></i>
              Create Job
            </p-button>
          </div>
        </div>
      </p-card>
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
  error?: string;

  constructor(
    private stateService: JobCreateStateService,
    private crawlerApi: CrawlerApiService,
    private router: Router,
    private messageService: MessageService
  ) {}

  ngOnInit(): void {
    this.jobConfig = this.stateService.buildJobConfig();
  }

  createJob(): void {
    this.creating = true;
    this.error = undefined;

    this.crawlerApi.createJob(this.jobConfig).subscribe({
      next: (response) => {
        this.creating = false;
        const jobId = response.id;

        // Show success message
        this.messageService.add({
          severity: 'success',
          summary: 'Success',
          detail: 'Job created successfully!',
          life: 3000
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

        this.messageService.add({
          severity: 'error',
          summary: 'Error',
          detail: this.error ?? 'Failed to create job',
          life: 5000
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
