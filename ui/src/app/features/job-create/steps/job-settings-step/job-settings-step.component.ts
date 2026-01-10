import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormArray, Validators, ReactiveFormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { Seed, ScopeRules, RateLimitPolicy } from '../../../../core/models/crawl-job.model';

@Component({
  selector: 'app-job-settings-step',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatChipsModule
  ],
  template: `
    <div class="space-y-4">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Step 4: Job Settings</mat-card-title>
          <mat-card-subtitle>
            Configure crawl scope, rate limits, and initial seeds
          </mat-card-subtitle>
        </mat-card-header>

        <mat-card-content>
          <form [formGroup]="settingsForm" class="space-y-6">
            <!-- Job Name -->
            <div>
              <h3 class="text-sm font-semibold mb-3">Job Information</h3>
              <mat-form-field appearance="outline" class="w-full">
                <mat-label>Job Name</mat-label>
                <input matInput formControlName="jobName" placeholder="My Crawl Job" />
                <mat-error *ngIf="settingsForm.get('jobName')?.hasError('required')">
                  Job name is required
                </mat-error>
              </mat-form-field>
            </div>

            <!-- Seeds -->
            <div>
              <div class="flex items-center justify-between mb-3">
                <h3 class="text-sm font-semibold">Seed URLs</h3>
                <button mat-raised-button color="primary" (click)="addSeed()" type="button">
                  <mat-icon>add</mat-icon>
                  Add Seed
                </button>
              </div>

              <div formArrayName="seeds" class="space-y-2">
                <div
                  *ngFor="let seed of seeds.controls; let i = index"
                  [formGroupName]="i"
                  class="flex items-center gap-2"
                >
                  <mat-form-field appearance="outline" class="flex-1">
                    <mat-label>Seed URL {{ i + 1 }}</mat-label>
                    <input matInput formControlName="url" placeholder="https://example.com" />
                    <mat-icon matPrefix>link</mat-icon>
                    <mat-error *ngIf="seed.get('url')?.hasError('required')">
                      URL is required
                    </mat-error>
                    <mat-error *ngIf="seed.get('url')?.hasError('pattern')">
                      Must be a valid URL
                    </mat-error>
                  </mat-form-field>

                  <button
                    mat-icon-button
                    color="warn"
                    (click)="removeSeed(i)"
                    type="button"
                    [disabled]="seeds.length === 1"
                  >
                    <mat-icon>delete</mat-icon>
                  </button>
                </div>
              </div>
            </div>

            <!-- Scope Rules -->
            <div formGroupName="scopeRules">
              <h3 class="text-sm font-semibold mb-3">Crawl Scope</h3>
              <div class="space-y-4">
                <mat-form-field appearance="outline" class="w-full">
                  <mat-label>Maximum Depth</mat-label>
                  <input matInput type="number" formControlName="max_depth" />
                  <mat-hint>0 = seeds only, 1 = seeds + direct links, etc.</mat-hint>
                  <mat-error *ngIf="settingsForm.get('scopeRules.max_depth')?.hasError('min')">
                    Must be at least 0
                  </mat-error>
                </mat-form-field>

                <div>
                  <div class="flex items-center justify-between mb-2">
                    <label class="text-sm font-medium">Allowed Domains</label>
                    <button mat-stroked-button (click)="addDomain()" type="button">
                      <mat-icon>add</mat-icon>
                      Add Domain
                    </button>
                  </div>

                  <div formArrayName="allowed_domains" class="space-y-2">
                    <div
                      *ngFor="let domain of allowedDomains.controls; let i = index"
                      class="flex items-center gap-2"
                    >
                      <mat-form-field appearance="outline" class="flex-1">
                        <mat-label>Domain {{ i + 1 }}</mat-label>
                        <input matInput [formControlName]="i" placeholder="example.com" />
                        <mat-icon matPrefix>public</mat-icon>
                      </mat-form-field>

                      <button mat-icon-button color="warn" (click)="removeDomain(i)" type="button">
                        <mat-icon>delete</mat-icon>
                      </button>
                    </div>
                  </div>

                  <p class="text-xs text-gray-500 mt-2">
                    Leave empty to allow all domains
                  </p>
                </div>
              </div>
            </div>

            <!-- Rate Limit -->
            <div formGroupName="rateLimit">
              <h3 class="text-sm font-semibold mb-3">Rate Limiting</h3>
              <mat-form-field appearance="outline" class="w-full">
                <mat-label>Requests Per Second (RPS)</mat-label>
                <input matInput type="number" formControlName="rps" />
                <mat-hint>Number of requests per second to send</mat-hint>
                <mat-error *ngIf="settingsForm.get('rateLimit.rps')?.hasError('min')">
                  RPS must be at least 0.1
                </mat-error>
              </mat-form-field>
            </div>
          </form>
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
export class JobSettingsStepComponent implements OnInit {
  settingsForm!: FormGroup;

  constructor(
    private fb: FormBuilder,
    private stateService: JobCreateStateService
  ) {}

  ngOnInit(): void {
    const state = this.stateService.getCurrentState();

    this.settingsForm = this.fb.group({
      jobName: [state.jobName || '', Validators.required],
      seeds: this.fb.array(
        state.seeds.length > 0
          ? state.seeds.map(s => this.createSeed(s))
          : [this.createSeed()]
      ),
      scopeRules: this.fb.group({
        max_depth: [state.scopeRules.max_depth || 2, [Validators.required, Validators.min(0)]],
        allowed_domains: this.fb.array(
          state.scopeRules.allowed_domains.length > 0
            ? state.scopeRules.allowed_domains.map(d => this.fb.control(d))
            : []
        )
      }),
      rateLimit: this.fb.group({
        rps: [state.rateLimit.rps || 1, [Validators.required, Validators.min(0.1)]]
      })
    });

    // Auto-save to state on changes
    this.settingsForm.valueChanges.subscribe(() => {
      if (this.settingsForm.valid) {
        this.saveToState();
      }
    });

    // Pre-fill first seed URL from preview if available
    if (state.previewUrl && state.seeds.length === 0) {
      this.seeds.at(0)?.patchValue({ url: state.previewUrl });
    }
  }

  get seeds(): FormArray {
    return this.settingsForm.get('seeds') as FormArray;
  }

  get allowedDomains(): FormArray {
    return this.settingsForm.get('scopeRules.allowed_domains') as FormArray;
  }

  createSeed(seed?: Seed): FormGroup {
    return this.fb.group({
      url: [
        seed?.url || '',
        [Validators.required, Validators.pattern(/^https?:\/\/.+/)]
      ]
    });
  }

  addSeed(): void {
    this.seeds.push(this.createSeed());
  }

  removeSeed(index: number): void {
    if (this.seeds.length > 1) {
      this.seeds.removeAt(index);
    }
  }

  addDomain(): void {
    this.allowedDomains.push(this.fb.control(''));
  }

  removeDomain(index: number): void {
    this.allowedDomains.removeAt(index);
  }

  saveToState(): void {
    const formValue = this.settingsForm.value;

    const seeds: Seed[] = formValue.seeds.map((s: any) => ({ url: s.url }));
    const scopeRules: ScopeRules = {
      max_depth: formValue.scopeRules.max_depth,
      allowed_domains: formValue.scopeRules.allowed_domains.filter((d: string) => d.trim() !== '')
    };
    const rateLimit: RateLimitPolicy = {
      rps: formValue.rateLimit.rps
    };

    this.stateService.setJobSettings(
      formValue.jobName,
      seeds,
      scopeRules,
      rateLimit
    );
  }

  isValid(): boolean {
    return this.settingsForm ? this.settingsForm.valid : false;
  }
}
