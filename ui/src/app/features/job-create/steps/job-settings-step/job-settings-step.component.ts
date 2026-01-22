import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormArray, Validators, ReactiveFormsModule } from '@angular/forms';
import { InputTextModule } from 'primeng/inputtext';
import { InputNumberModule } from 'primeng/inputnumber';
import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { FloatLabelModule } from 'primeng/floatlabel';
import { IconFieldModule } from 'primeng/iconfield';
import { InputIconModule } from 'primeng/inputicon';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { Seed, ScopeRules, RateLimitPolicy } from '../../../../core/models/crawl-job.model';

@Component({
  selector: 'app-job-settings-step',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    InputTextModule,
    InputNumberModule,
    ButtonModule,
    CardModule,
    FloatLabelModule,
    IconFieldModule,
    InputIconModule
  ],
  template: `
    <div class="space-y-4">
      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h2 class="text-xl font-semibold">Step 3: Job Settings</h2>
            <p class="text-sm text-gray-500">Configure crawl scope, rate limits, and initial seeds</p>
          </div>
        </ng-template>

        <form [formGroup]="settingsForm" class="space-y-6">
          <!-- Job Name -->
          <div>
            <h3 class="text-sm font-semibold mb-3">Job Information</h3>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Job Name</label>
              <input pInputText formControlName="jobName" placeholder="My Crawl Job" class="w-full" />
              <small *ngIf="settingsForm.get('jobName')?.hasError('required') && settingsForm.get('jobName')?.touched" class="text-red-500">
                Job name is required
              </small>
            </div>
          </div>

          <!-- Seeds -->
          <div>
            <div class="flex items-center justify-between mb-3">
              <h3 class="text-sm font-semibold">Seed URLs</h3>
              <p-button (onClick)="addSeed()" severity="secondary" [outlined]="true">
                <i class="pi pi-plus mr-2"></i>
                Add Seed
              </p-button>
            </div>

            <div formArrayName="seeds" class="space-y-2">
              <div
                *ngFor="let seed of seeds.controls; let i = index"
                [formGroupName]="i"
                class="flex items-center gap-2">
                <div class="flex-1">
                  <p-iconfield>
                    <p-inputicon styleClass="pi pi-link" />
                    <input
                      pInputText
                      formControlName="url"
                      [placeholder]="'https://example.com'"
                      class="w-full" />
                  </p-iconfield>
                  <small *ngIf="seed.get('url')?.hasError('required') && seed.get('url')?.touched" class="text-red-500">
                    URL is required
                  </small>
                  <small *ngIf="seed.get('url')?.hasError('pattern') && seed.get('url')?.touched" class="text-red-500">
                    Must be a valid URL
                  </small>
                </div>

                <p-button
                  [text]="true"
                  [rounded]="true"
                  severity="danger"
                  (onClick)="removeSeed(i)"
                  [disabled]="seeds.length === 1">
                  <i class="pi pi-trash"></i>
                </p-button>
              </div>
            </div>
          </div>

          <!-- Scope Rules -->
          <div formGroupName="scopeRules">
            <h3 class="text-sm font-semibold mb-3">Crawl Scope</h3>
            <div class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Maximum Depth</label>
                <p-inputNumber
                  formControlName="max_depth"
                  [showButtons]="true"
                  [min]="0"
                  styleClass="w-full" />
                <small class="text-gray-500">0 = seeds only, 1 = seeds + direct links, etc.</small>
                <small *ngIf="settingsForm.get('scopeRules.max_depth')?.hasError('min')" class="text-red-500 block">
                  Must be at least 0
                </small>
              </div>

              <div>
                <div class="flex items-center justify-between mb-2">
                  <label class="text-sm font-medium">Allowed Domains</label>
                  <p-button [outlined]="true" severity="secondary" (onClick)="addDomain()">
                    <i class="pi pi-plus mr-2"></i>
                    Add Domain
                  </p-button>
                </div>

                <div formArrayName="allowed_domains" class="space-y-2">
                  <div
                    *ngFor="let domain of allowedDomains.controls; let i = index"
                    class="flex items-center gap-2">
                    <div class="flex-1">
                      <p-iconfield>
                        <p-inputicon styleClass="pi pi-globe" />
                        <input
                          pInputText
                          [formControlName]="i"
                          placeholder="example.com"
                          class="w-full" />
                      </p-iconfield>
                    </div>

                    <p-button
                      [text]="true"
                      [rounded]="true"
                      severity="danger"
                      (onClick)="removeDomain(i)">
                      <i class="pi pi-trash"></i>
                    </p-button>
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
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Requests Per Second (RPS)</label>
              <p-inputNumber
                formControlName="rps"
                [showButtons]="true"
                [min]="0.1"
                [step]="0.1"
                mode="decimal"
                styleClass="w-full" />
              <small class="text-gray-500">Number of requests per second to send</small>
              <small *ngIf="settingsForm.get('rateLimit.rps')?.hasError('min')" class="text-red-500 block">
                RPS must be at least 0.1
              </small>
            </div>
          </div>
        </form>
      </p-card>
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
