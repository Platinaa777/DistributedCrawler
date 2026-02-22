
import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, FormArray, FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { PanelModule } from 'primeng/panel';
import { InputTextModule } from 'primeng/inputtext';
import { InputNumberModule } from 'primeng/inputnumber';
import { ButtonModule } from 'primeng/button';
import { DividerModule } from 'primeng/divider';
import { CheckboxModule } from 'primeng/checkbox';
import { SelectModule } from 'primeng/select';
import { ToastModule } from 'primeng/toast';
import { MessageService } from 'primeng/api';
import { CrawlerApiService } from '../../core/services/api/crawler-api.service';
import { CrawlJobConfig, RetryPolicy, ScheduleOptions, JobType, JOB_TYPES, CrawlMode, CRAWL_MODES } from '../../core/models/crawl-job.model';
import { FieldSpec, ItemsSpec, PaginationSpec, TransformSpec } from '../../core/models/extraction-spec.model';

interface SimpleJobFormValue {
  name: string;
  seeds: { url: string }[];
  allowed_domains: string[];
  deny_url_patterns: string[];
  allowed_url_patterns: string[];
  max_depth: number;
  rps: number;
  retries: RetryPolicy;
  auth: {
    basic_user?: string;
    basic_password?: string;
    bearer_token?: string;
    cookie?: string;
  };
  schedule: ScheduleOptions;
  job_type: JobType;
  crawl_mode: CrawlMode;
  items_enabled: boolean;
  items_container_selector: string;
}

@Component({
  selector: 'app-simple-job-create',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    CardModule,
    PanelModule,
    InputTextModule,
    InputNumberModule,
    ButtonModule,
    DividerModule,
    CheckboxModule,
    SelectModule,
    ToastModule
  ],
  providers: [MessageService],
  template: `
    <p-toast position="top-right" />

    <form class="container mx-auto p-6 space-y-8" [formGroup]="jobForm">
      <div class="flex flex-wrap items-center justify-between gap-3 mb-6">
        <div class="flex items-center gap-2">
          <p-button [text]="true" (onClick)="goBack()" type="button">
            <i class="pi pi-arrow-left mr-2"></i>
            Back to Jobs
          </p-button>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Simple Crawl Job</h1>
        </div>
        <div class="flex items-center gap-2">
          <p-button
            [outlined]="true"
            severity="secondary"
            type="button"
            (onClick)="toggleImportPanel()">
            <i class="pi pi-file-import mr-2"></i>
            {{ showImportPanel ? 'Close Import' : 'Import JSON' }}
          </p-button>
          <p-button
            type="button"
            (onClick)="submit()"
            [disabled]="creating || !canSubmit()">
            <i class="pi pi-send mr-2"></i>
            {{ creating ? 'Creating...' : 'Create Job' }}
          </p-button>
        </div>
      </div>

      <p-card *ngIf="showImportPanel" styleClass="mb-6">
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Import from JSON</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">Paste a job config JSON to fill all fields automatically.</p>
          </div>
        </ng-template>
        <div class="p-4 space-y-3">
          <textarea
            [(ngModel)]="importJsonText"
            [ngModelOptions]="{standalone: true}"
            placeholder='{ "config": { "name": "...", "seeds": [...], ... } }'
            rows="10"
            class="w-full font-mono text-xs p-3 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 resize-y"
          ></textarea>
          <div *ngIf="importError" class="text-red-600 text-sm flex items-center gap-2">
            <i class="pi pi-times-circle"></i>
            {{ importError }}
          </div>
          <div class="flex justify-end gap-2">
            <p-button [outlined]="true" severity="secondary" type="button" (onClick)="toggleImportPanel()">
              Cancel
            </p-button>
            <p-button type="button" (onClick)="importFromJson()" [disabled]="!importJsonText.trim()">
              <i class="pi pi-check mr-2"></i>
              Apply
            </p-button>
          </div>
        </div>
      </p-card>

      <p-card styleClass="mb-6">
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Job Basics</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">Minimal settings to register a crawl job.</p>
          </div>
        </ng-template>
        <div class="p-4 space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name</label>
            <input pInputText formControlName="name" placeholder="Example crawl job" class="w-full" />
            <small *ngIf="jobForm.get('name')?.hasError('required')" class="text-red-500">
              Name is required
            </small>
          </div>
        </div>
      </p-card>

      <p-panel header="Seeds & Scope" [toggleable]="true" [collapsed]="true" styleClass="mb-6">
        <div class="p-4 space-y-6">
          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Seed URLs</p>
              <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addSeed()">
                <i class="pi pi-plus mr-2"></i>
                Add Seed
              </p-button>
            </div>
            <div formArrayName="seeds" class="space-y-2">
              <div
                *ngFor="let seed of seeds.controls; let i = index"
                [formGroupName]="i"
                class="flex items-center gap-2"
              >
                <div class="flex-1">
                  <input pInputText formControlName="url" placeholder="https://example.com" class="w-full" />
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
                  type="button"
                  (onClick)="removeSeed(i)"
                  [disabled]="seeds.length === 1">
                  <i class="pi pi-trash"></i>
                </p-button>
              </div>
            </div>
          </div>

          <p-divider></p-divider>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Depth</label>
              <p-inputNumber formControlName="max_depth" [min]="0" styleClass="w-full"></p-inputNumber>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">RPS</label>
              <p-inputNumber formControlName="rps" [min]="0.1" [step]="0.1" mode="decimal" styleClass="w-full"></p-inputNumber>
            </div>
          </div>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Allowed Domains</p>
              <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addAllowedDomain()">
                <i class="pi pi-plus mr-2"></i>
                Add
              </p-button>
            </div>
            <div formArrayName="allowed_domains" class="space-y-2">
              <div *ngFor="let domain of allowedDomains.controls; let i = index" class="flex items-center gap-2">
                <input pInputText [formControlName]="i" placeholder="example.com" class="flex-1" />
                <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removeAllowedDomain(i)">
                  <i class="pi pi-trash"></i>
                </p-button>
              </div>
            </div>
          </div>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Allowed URL Patterns (wildcard)</p>
              <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addAllowedUrlPattern()">
                <i class="pi pi-plus mr-2"></i>
                Add
              </p-button>
            </div>
            <div formArrayName="allowed_url_patterns" class="space-y-2">
              <div *ngFor="let pattern of allowedUrlPatterns.controls; let i = index" class="flex items-center gap-2">
                <input pInputText [formControlName]="i" placeholder="https://example.com/*" class="flex-1" />
                <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removeAllowedUrlPattern(i)">
                  <i class="pi pi-trash"></i>
                </p-button>
              </div>
            </div>
            <small class="text-xs text-gray-500 dark:text-gray-400">
              Optional. Uses * wildcard and filters discovered/pagination URLs in parser worker.
            </small>
          </div>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Deny URL Patterns</p>
              <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addDenyPattern()">
                <i class="pi pi-plus mr-2"></i>
                Add
              </p-button>
            </div>
            <div formArrayName="deny_url_patterns" class="space-y-2">
              <div *ngFor="let pattern of denyPatterns.controls; let i = index" class="flex items-center gap-2">
                <input pInputText [formControlName]="i" placeholder="/login" class="flex-1" />
                <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removeDenyPattern(i)">
                  <i class="pi pi-trash"></i>
                </p-button>
              </div>
            </div>
          </div>
        </div>
      </p-panel>

      <p-panel header="Rate Limit, Retries & Schedule" [toggleable]="true" [collapsed]="true" styleClass="mb-6">
        <div class="p-4 grid grid-cols-1 md:grid-cols-3 gap-6">
          <div [formGroup]="retriesGroup" class="space-y-4">
            <p class="text-sm font-semibold">Retry Policy</p>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Attempts</label>
              <p-inputNumber formControlName="max_attempts" [min]="0" styleClass="w-full"></p-inputNumber>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Backoff Initial (ms)</label>
              <p-inputNumber formControlName="backoff_initial_ms" [min]="0" styleClass="w-full"></p-inputNumber>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Backoff Multiplier</label>
              <p-inputNumber formControlName="backoff_multiplier" [min]="0" mode="decimal" styleClass="w-full"></p-inputNumber>
            </div>
          </div>

          <div class="space-y-4">
            <p class="text-sm font-semibold">Schedule</p>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Job Type</label>
              <p-select
                [options]="jobTypeSelectOptions"
                optionLabel="label"
                optionValue="value"
                formControlName="job_type"
                styleClass="w-full">
              </p-select>
              <small class="text-gray-500 dark:text-gray-400">
                {{ jobForm.get('job_type')?.value === 'JOB_TYPE_SCHEDULED' ? 'Runs on a recurring schedule' : 'Runs exactly once' }}
              </small>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Crawl Mode</label>
              <p-select
                [options]="crawlModeSelectOptions"
                optionLabel="label"
                optionValue="value"
                formControlName="crawl_mode"
                styleClass="w-full">
              </p-select>
              <small class="text-gray-500 dark:text-gray-400">{{ crawlModeDescription }}</small>
            </div>
            <div [formGroup]="scheduleGroup">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Cron</label>
              <input pInputText formControlName="cron" placeholder="0 9 * * 1" class="w-full" />
            </div>
          </div>

          <div [formGroup]="authGroup" class="space-y-4">
            <p class="text-sm font-semibold">Auth (optional)</p>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Basic User</label>
              <input pInputText formControlName="basic_user" class="w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Basic Password</label>
              <input pInputText type="password" formControlName="basic_password" class="w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Bearer Token</label>
              <input pInputText formControlName="bearer_token" class="w-full" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Cookie</label>
              <input pInputText formControlName="cookie" class="w-full" />
            </div>
          </div>
        </div>
      </p-panel>

      <p-panel header="Extraction Spec" [toggleable]="true" styleClass="mb-6">
        <div class="p-4 space-y-6">
          <div class="flex items-center justify-between">
            <p class="text-sm font-semibold">Fields</p>
            <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addExtractionField()">
              <i class="pi pi-plus mr-2"></i>
              Add Field
            </p-button>
          </div>

          <div formArrayName="extraction_fields" class="space-y-6">
            <div
              *ngFor="let field of extractionFields.controls; let i = index"
              [formGroupName]="i"
              class="border border-gray-200 dark:border-gray-700 rounded p-4 space-y-3"
            >
              <div class="flex items-center justify-between">
                <div class="font-semibold">Field #{{ i + 1 }}</div>
                <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removeExtractionField(i)">
                  <i class="pi pi-trash"></i>
                </p-button>
              </div>

              <div class="grid grid-cols-1 gap-3">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name</label>
                  <input pInputText formControlName="name" class="w-full" />
                </div>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type</label>
                  <p-select [options]="fieldTypeSelectOptions" optionLabel="label" optionValue="value" formControlName="type" styleClass="w-full"></p-select>
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Attribute</label>
                  <p-select [options]="attributeSelectOptions" optionLabel="label" optionValue="value" formControlName="attribute" styleClass="w-full"></p-select>
                </div>
              </div>

              <div class="grid grid-cols-1 gap-3 items-center">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Selector</label>
                  <input pInputText formControlName="selector" class="w-full" />
                </div>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-3 gap-3 items-center">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Index</label>
                  <p-inputNumber
                    formControlName="index"
                    [disabled]="!field.get('multiple')?.value"
                    styleClass="w-full">
                  </p-inputNumber>
                  <small class="text-xs text-gray-500 dark:text-gray-400">Works only with Multiple: true</small>
                </div>
                <div class="flex items-center gap-2 mt-6">
                  <p-checkbox
                    formControlName="multiple"
                    [binary]="true"
                    inputId="multiple-{{ i }}"
                    (onChange)="handleMultipleToggle(i)">
                  </p-checkbox>
                  <label for="multiple-{{ i }}" class="text-sm text-gray-700 dark:text-gray-300">Multiple</label>
                </div>
                <div class="flex items-center gap-2 mt-6">
                  <p-checkbox formControlName="required" [binary]="true" inputId="required-{{ i }}"></p-checkbox>
                  <label for="required-{{ i }}" class="text-sm text-gray-700 dark:text-gray-300">Required</label>
                </div>
              </div>

              <div formArrayName="transforms" class="space-y-2">
                <div class="flex items-center justify-between">
                  <p class="text-sm font-semibold">Transforms</p>
                  <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addTransform(i)">
                    <i class="pi pi-plus mr-2"></i>
                    Add Transform
                  </p-button>
                </div>
                <div
                  *ngFor="let transform of getTransforms(i).controls; let tIdx = index"
                  [formGroupName]="tIdx"
                  class="grid grid-cols-1 md:grid-cols-2 gap-2 items-center"
                >
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Op</label>
                    <p-select [options]="transformOpSelectOptions" optionLabel="label" optionValue="value" formControlName="op" styleClass="w-full"></p-select>
                  </div>
                  <div class="flex items-center gap-2">
                    <div class="flex-1">
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Arg</label>
                      <input pInputText formControlName="arg" class="w-full" />
                    </div>
                    <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removeTransform(i, tIdx)">
                      <i class="pi pi-trash"></i>
                    </p-button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <p-divider></p-divider>

          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <p class="text-sm font-semibold">Items Extraction</p>
              <div class="flex items-center gap-2">
                <p-checkbox formControlName="items_enabled" [binary]="true" inputId="items-enabled"></p-checkbox>
                <label for="items-enabled" class="text-sm text-gray-700 dark:text-gray-300">Enable</label>
              </div>
            </div>

            <div *ngIf="jobForm.get('items_enabled')?.value" class="space-y-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Container Selector</label>
                <input pInputText formControlName="items_container_selector" placeholder="article.product_pod" class="w-full" />
                <small class="text-xs text-gray-500 dark:text-gray-400">Each matched element becomes one item object.</small>
              </div>

              <div class="flex items-center justify-between">
                <p class="text-sm font-semibold">Item Fields</p>
                <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addItemField()">
                  <i class="pi pi-plus mr-2"></i>
                  Add Item Field
                </p-button>
              </div>

              <div formArrayName="items_fields" class="space-y-6">
                <div
                  *ngFor="let field of itemFields.controls; let i = index"
                  [formGroupName]="i"
                  class="border border-gray-200 dark:border-gray-700 rounded p-4 space-y-3"
                >
                  <div class="flex items-center justify-between">
                    <div class="font-semibold">Item Field #{{ i + 1 }}</div>
                    <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removeItemField(i)">
                      <i class="pi pi-trash"></i>
                    </p-button>
                  </div>

                  <div class="grid grid-cols-1 gap-3">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name</label>
                      <input pInputText formControlName="name" class="w-full" />
                    </div>
                  </div>

                  <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type</label>
                      <p-select [options]="fieldTypeSelectOptions" optionLabel="label" optionValue="value" formControlName="type" styleClass="w-full"></p-select>
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Attribute</label>
                      <p-select [options]="attributeSelectOptions" optionLabel="label" optionValue="value" formControlName="attribute" styleClass="w-full"></p-select>
                    </div>
                  </div>

                  <div class="grid grid-cols-1 gap-3 items-center">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Selector</label>
                      <input pInputText formControlName="selector" class="w-full" />
                    </div>
                  </div>

                  <div class="grid grid-cols-1 md:grid-cols-3 gap-3 items-center">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Index</label>
                      <p-inputNumber
                        formControlName="index"
                        [disabled]="!field.get('multiple')?.value"
                        styleClass="w-full">
                      </p-inputNumber>
                      <small class="text-xs text-gray-500 dark:text-gray-400">Works only with Multiple: true</small>
                    </div>
                    <div class="flex items-center gap-2 mt-6">
                      <p-checkbox
                        formControlName="multiple"
                        [binary]="true"
                        inputId="item-multiple-{{ i }}"
                        (onChange)="handleItemMultipleToggle(i)">
                      </p-checkbox>
                      <label for="item-multiple-{{ i }}" class="text-sm text-gray-700 dark:text-gray-300">Multiple</label>
                    </div>
                    <div class="flex items-center gap-2 mt-6">
                      <p-checkbox formControlName="required" [binary]="true" inputId="item-required-{{ i }}"></p-checkbox>
                      <label for="item-required-{{ i }}" class="text-sm text-gray-700 dark:text-gray-300">Required</label>
                    </div>
                  </div>

                  <div formArrayName="transforms" class="space-y-2">
                    <div class="flex items-center justify-between">
                      <p class="text-sm font-semibold">Transforms</p>
                      <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addItemTransform(i)">
                        <i class="pi pi-plus mr-2"></i>
                        Add Transform
                      </p-button>
                    </div>
                    <div
                      *ngFor="let transform of getItemTransforms(i).controls; let tIdx = index"
                      [formGroupName]="tIdx"
                      class="grid grid-cols-1 md:grid-cols-2 gap-2 items-center"
                    >
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Op</label>
                        <p-select [options]="transformOpSelectOptions" optionLabel="label" optionValue="value" formControlName="op" styleClass="w-full"></p-select>
                      </div>
                      <div class="flex items-center gap-2">
                        <div class="flex-1">
                          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Arg</label>
                          <input pInputText formControlName="arg" class="w-full" />
                        </div>
                        <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removeItemTransform(i, tIdx)">
                          <i class="pi pi-trash"></i>
                        </p-button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div *ngIf="itemFields.length === 0" class="text-gray-500 dark:text-gray-400 text-sm">
                Add at least one item field to extract structured arrays.
              </div>
            </div>
          </div>

          <p-divider></p-divider>

          <div>
            <div class="flex items-center justify-between mb-2">
              <p class="text-sm font-semibold">Pagination</p>
              <p-button [outlined]="true" severity="secondary" type="button" (onClick)="addPagination()">
                <i class="pi pi-plus mr-2"></i>
                Add Pagination
              </p-button>
            </div>
            <div formArrayName="pagination" class="space-y-4">
              <div
                *ngFor="let pag of pagination.controls; let p = index"
                [formGroupName]="p"
                class="border border-gray-200 dark:border-gray-700 rounded p-3 grid grid-cols-1 md:grid-cols-4 gap-3 items-start"
              >
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name</label>
                  <input pInputText formControlName="name" placeholder="next_page" class="w-full" />
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Selector</label>
                  <input pInputText formControlName="selector" placeholder="a.next-page" class="w-full" />
                </div>
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Attribute</label>
                  <p-select [options]="paginationAttributeSelectOptions" optionLabel="label" optionValue="value" formControlName="attribute" styleClass="w-full"></p-select>
                </div>
                <div class="flex items-center gap-2 mt-6">
                  <p-checkbox formControlName="multiple" [binary]="true" inputId="pagination-multiple-{{ p }}"></p-checkbox>
                  <label for="pagination-multiple-{{ p }}" class="text-sm text-gray-700 dark:text-gray-300">Multiple</label>
                  <p-button [text]="true" [rounded]="true" severity="danger" type="button" (onClick)="removePagination(p)">
                    <i class="pi pi-trash"></i>
                  </p-button>
                </div>
              </div>
            </div>
            <div *ngIf="pagination.length === 0" class="text-gray-500 dark:text-gray-400 text-sm mt-2">
              No pagination selectors configured. Add pagination to follow next-page links.
            </div>
          </div>
        </div>
      </p-panel>

      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">Preview Payload</h3>
          </div>
        </ng-template>
        <div class="p-4">
          <pre class="bg-gray-100 dark:bg-gray-800 p-3 rounded text-xs overflow-auto text-gray-900 dark:text-gray-100">{{ previewJson }}</pre>
          <div *ngIf="error" class="text-red-600 text-sm flex items-center gap-2 mt-2">
            <i class="pi pi-times-circle"></i>
            {{ error }}
          </div>
        </div>
      </p-card>
    </form>
  `,
  styles: [`
    :host {
      display: block;
    }

    pre {
      white-space: pre-wrap;
      word-break: break-word;
    }
  `]
})
export class SimpleJobCreateComponent implements OnInit {
  jobForm!: FormGroup;
  creating = false;
  error?: string;
  previewJson = '';
  showImportPanel = false;
  importJsonText = '';
  importError?: string;
  readonly fieldTypeOptions: FieldSpec['type'][] = ['string', 'int', 'float', 'bool', 'url', 'json'];
  readonly attributeOptions = ['text', 'html', 'href', 'src', 'content'];
  readonly transformOpOptions: TransformSpec['op'][] = [
    'trim',
    'lower',
    'upper',
    'normalize_url',
    'unique',
    'limit',
    'to_int',
    'to_float',
    'parse_price',
    'html_to_text',
    'collapse_ws',
    'sha256'
  ];

  fieldTypeSelectOptions = this.fieldTypeOptions.map(option => ({ label: option, value: option }));
  attributeSelectOptions = this.attributeOptions.map(option => ({ label: option, value: option }));
  transformOpSelectOptions = this.transformOpOptions.map(option => ({ label: option, value: option }));
  paginationAttributeSelectOptions = ['href', 'src', 'data-url', 'content'].map(option => ({ label: option, value: option }));
  jobTypeSelectOptions = JOB_TYPES.map(option => ({ label: option.label, value: option.value }));
  crawlModeSelectOptions = CRAWL_MODES.map(option => ({ label: option.label, value: option.value }));

  private readonly sampleConfig: CrawlJobConfig = {
    name: 'Example Crawl Job',
    seeds: [{ url: 'https://bool.dev/blog/detail/voprosy-na-sobesedovanii-dlya-senior-net-developer' }],
    scopes: {
      allowed_domains: ['bool.dev'],
      deny_url_patterns: ['/login', '/register'],
      allowed_url_patterns: ['https://bool.dev/blog/*'],
      max_depth: 0
    },
    rate_limit: { rps: 1 },
    retries: { max_attempts: 3, backoff_initial_ms: 500, backoff_multiplier: 2 },
    schedule: { cron: '0 9 * * 1' },
    auth: { basic_user: '', basic_password: '', bearer_token: '', cookie: '' },
    extraction_spec: {
      fields: [
        {
          name: 'page_url',
          type: 'string',
          required: true,
          extractor: {
            selector: '',
            attribute: 'text',
            multiple: false
          },
          transforms: []
        },
        {
          name: 'title',
          type: 'string',
          required: true,
          extractor: {
            selector: "h1[itemprop='headline']",
            attribute: 'text',
            multiple: false
          },
          transforms: []
        }
      ],
    }
  };

  constructor(
    private fb: FormBuilder,
    private crawlerApi: CrawlerApiService,
    private router: Router,
    private messageService: MessageService
  ) {}

  ngOnInit(): void {
    this.jobForm = this.fb.group({
      name: ['', Validators.required],
      job_type: ['JOB_TYPE_ONCE', Validators.required],
      crawl_mode: ['CRAWL_MODE_PAGINATION_AND_LINKS', Validators.required],
      seeds: this.fb.array([this.createSeedGroup()]),
      allowed_domains: this.fb.array([]),
      deny_url_patterns: this.fb.array([]),
      allowed_url_patterns: this.fb.array([]),
      max_depth: [0, [Validators.required, Validators.min(0)]],
      rps: [1, [Validators.required, Validators.min(0.1)]],
      retries: this.fb.group({
        max_attempts: [3],
        backoff_initial_ms: [500],
        backoff_multiplier: [2]
      }),
      auth: this.fb.group({
        basic_user: [''],
        basic_password: [''],
        bearer_token: [''],
        cookie: ['']
      }),
      schedule: this.fb.group({
        cron: ['']
      }),
      extraction_fields: this.fb.array([]),
      items_enabled: [false],
      items_container_selector: [''],
      items_fields: this.fb.array([]),
      pagination: this.fb.array([])
    });

    this.jobForm.valueChanges.subscribe(() => this.updatePreview());

    this.updatePreview();
  }

  get seeds(): FormArray {
    return this.jobForm.get('seeds') as FormArray;
  }

  get allowedDomains(): FormArray {
    return this.jobForm.get('allowed_domains') as FormArray;
  }

  get denyPatterns(): FormArray {
    return this.jobForm.get('deny_url_patterns') as FormArray;
  }

  get allowedUrlPatterns(): FormArray {
    return this.jobForm.get('allowed_url_patterns') as FormArray;
  }

  get extractionFields(): FormArray {
    return this.jobForm.get('extraction_fields') as FormArray;
  }

  get pagination(): FormArray {
    return this.jobForm.get('pagination') as FormArray;
  }

  get itemFields(): FormArray {
    return this.jobForm.get('items_fields') as FormArray;
  }

  get retriesGroup(): FormGroup {
    return this.jobForm.get('retries') as FormGroup;
  }

  get scheduleGroup(): FormGroup {
    return this.jobForm.get('schedule') as FormGroup;
  }

  get authGroup(): FormGroup {
    return this.jobForm.get('auth') as FormGroup;
  }

  get crawlModeDescription(): string {
    const mode = this.jobForm.get('crawl_mode')?.value as CrawlMode;
    return CRAWL_MODES.find(m => m.value === mode)?.description ?? '';
  }

  goBack(): void {
    this.router.navigate(['/jobs']);
  }

  toggleImportPanel(): void {
    this.showImportPanel = !this.showImportPanel;
    if (!this.showImportPanel) {
      this.importJsonText = '';
      this.importError = undefined;
    }
  }

  importFromJson(): void {
    this.importError = undefined;
    let parsed: any;
    try {
      parsed = JSON.parse(this.importJsonText);
    } catch (e) {
      this.importError = 'Invalid JSON: ' + (e as Error).message;
      return;
    }

    const config = parsed.config ?? parsed;

    try {
      this.jobForm.patchValue({
        name: config.name ?? '',
        job_type: (config.job_type as JobType) ?? 'JOB_TYPE_ONCE',
        crawl_mode: config.crawl_mode ?? 'CRAWL_MODE_PAGINATION_AND_LINKS',
        max_depth: config.scopes?.max_depth ?? 0,
        rps: config.rate_limit?.rps ?? 1,
        retries: config.retries ?? {},
        auth: config.auth ?? {},
        schedule: config.schedule ?? {},
        items_enabled: !!config.extraction_spec?.items,
        items_container_selector: config.extraction_spec?.items?.container_selector ?? ''
      });

      if (config.seeds?.length > 0) {
        this.resetArray(this.seeds, config.seeds.map((s: any) => this.createSeedGroup(s.url ?? '')));
      }

      const domains: string[] = config.scopes?.allowed_domains ?? [];
      this.resetArray(this.allowedDomains, domains.map(d => this.fb.control(d)));

      const patterns: string[] = config.scopes?.deny_url_patterns ?? [];
      this.resetArray(this.denyPatterns, patterns.map(p => this.fb.control(p)));

      const allowedUrlPatterns: string[] = config.scopes?.allowed_url_patterns ?? [];
      this.resetArray(this.allowedUrlPatterns, allowedUrlPatterns.map(p => this.fb.control(p)));

      const fields: any[] = config.extraction_spec?.fields ?? [];
      this.resetArray(this.extractionFields, fields.map(f => this.createExtractionFieldGroup(f as Partial<FieldSpec>)));

      const itemFields: any[] = config.extraction_spec?.items?.fields ?? [];
      this.resetArray(this.itemFields, itemFields.map(f => this.createExtractionFieldGroup(f as Partial<FieldSpec>)));

      const paginationItems: any[] = config.extraction_spec?.pagination ?? [];
      this.resetArray(this.pagination, paginationItems.map(p => this.createPaginationGroup(p)));

      this.updatePreview();
      this.showImportPanel = false;
      this.importJsonText = '';
      this.messageService.add({
        severity: 'success',
        summary: 'Imported',
        detail: 'Configuration loaded from JSON',
        life: 3000
      });
    } catch (e) {
      this.importError = 'Failed to apply config: ' + (e as Error).message;
    }
  }

  addSeed(url = ''): void {
    this.seeds.push(this.createSeedGroup(url));
    this.updatePreview();
  }

  removeSeed(index: number): void {
    if (this.seeds.length > 1) {
      this.seeds.removeAt(index);
      this.updatePreview();
    }
  }

  addAllowedDomain(domain = ''): void {
    this.allowedDomains.push(this.fb.control(domain));
    this.updatePreview();
  }

  removeAllowedDomain(index: number): void {
    this.allowedDomains.removeAt(index);
    this.updatePreview();
  }

  addDenyPattern(pattern = ''): void {
    this.denyPatterns.push(this.fb.control(pattern));
    this.updatePreview();
  }

  removeDenyPattern(index: number): void {
    this.denyPatterns.removeAt(index);
    this.updatePreview();
  }

  addAllowedUrlPattern(pattern = ''): void {
    this.allowedUrlPatterns.push(this.fb.control(pattern));
    this.updatePreview();
  }

  removeAllowedUrlPattern(index: number): void {
    this.allowedUrlPatterns.removeAt(index);
    this.updatePreview();
  }

  addExtractionField(field?: Partial<FieldSpec>): void {
    this.extractionFields.push(this.createExtractionFieldGroup(field));
    this.updatePreview();
  }

  removeExtractionField(index: number): void {
    this.extractionFields.removeAt(index);
    this.updatePreview();
  }

  addTransform(fieldIndex: number, transform?: TransformSpec): void {
    this.getTransforms(fieldIndex).push(this.createTransformGroup(transform));
    this.updatePreview();
  }

  removeTransform(fieldIndex: number, transformIndex: number): void {
    this.getTransforms(fieldIndex).removeAt(transformIndex);
    this.updatePreview();
  }

  addPagination(pagination?: Partial<PaginationSpec>): void {
    this.pagination.push(this.createPaginationGroup(pagination));
    this.updatePreview();
  }

  removePagination(index: number): void {
    this.pagination.removeAt(index);
    this.updatePreview();
  }

  getTransforms(fieldIndex: number): FormArray {
    return (this.extractionFields.at(fieldIndex) as FormGroup).get('transforms') as FormArray;
  }

  addItemField(field?: Partial<FieldSpec>): void {
    this.itemFields.push(this.createExtractionFieldGroup(field));
    this.updatePreview();
  }

  removeItemField(index: number): void {
    this.itemFields.removeAt(index);
    this.updatePreview();
  }

  addItemTransform(fieldIndex: number, transform?: TransformSpec): void {
    this.getItemTransforms(fieldIndex).push(this.createTransformGroup(transform));
    this.updatePreview();
  }

  removeItemTransform(fieldIndex: number, transformIndex: number): void {
    this.getItemTransforms(fieldIndex).removeAt(transformIndex);
    this.updatePreview();
  }

  getItemTransforms(fieldIndex: number): FormArray {
    return (this.itemFields.at(fieldIndex) as FormGroup).get('transforms') as FormArray;
  }

  handleMultipleToggle(fieldIndex: number): void {
    const fieldGroup = this.extractionFields.at(fieldIndex) as FormGroup;
    const multipleCtrl = fieldGroup.get('multiple');
    const indexCtrl = fieldGroup.get('index');

    if (!multipleCtrl?.value) {
      indexCtrl?.setValue(null);
    }
  }

  handleItemMultipleToggle(fieldIndex: number): void {
    const fieldGroup = this.itemFields.at(fieldIndex) as FormGroup;
    const multipleCtrl = fieldGroup.get('multiple');
    const indexCtrl = fieldGroup.get('index');

    if (!multipleCtrl?.value) {
      indexCtrl?.setValue(null);
    }
  }

  resetToSample(): void {
    const s = this.sampleConfig;
    this.jobForm.patchValue({
      name: s.name,
      max_depth: s.scopes.max_depth,
      rps: s.rate_limit.rps,
      retries: s.retries,
      schedule: s.schedule,
      auth: s.auth
    });

    this.resetArray(this.seeds, s.seeds.map(seed => this.createSeedGroup(seed.url)));
    this.resetArray(this.allowedDomains, s.scopes.allowed_domains.map(d => this.fb.control(d)));
    this.resetArray(this.denyPatterns, (s.scopes.deny_url_patterns || []).map(p => this.fb.control(p)));
    this.resetArray(this.allowedUrlPatterns, (s.scopes.allowed_url_patterns || []).map(p => this.fb.control(p)));

    this.resetArray(
      this.extractionFields,
      s.extraction_spec.fields.map(f => this.createExtractionFieldGroup(f as Partial<FieldSpec>))
    );
    this.jobForm.patchValue({
      items_enabled: false,
      items_container_selector: ''
    });
    this.resetArray(this.itemFields, []);

    this.updatePreview();
  }

  submit(): void {
    if (!this.canSubmit()) {
      this.error = 'Fill all required fields before creating a job.';
      return;
    }

    this.creating = true;
      this.error = undefined;
    const payload = { config: this.buildConfig() };

    this.crawlerApi.createJob(payload.config as CrawlJobConfig).subscribe({
      next: (response) => {
        this.creating = false;
        this.messageService.add({
          severity: 'success',
          summary: 'Success',
          detail: 'Job created successfully',
          life: 3000
        });
        this.router.navigate(['/jobs', response.id]);
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

  canSubmit(): boolean {
    const hasPageFields = this.extractionFields.length > 0;
    const itemsEnabled = !!this.jobForm.get('items_enabled')?.value;
    const itemsSelector = (this.jobForm.get('items_container_selector')?.value || '').trim();
    const hasValidItems = itemsEnabled && itemsSelector.length > 0 && this.itemFields.length > 0;

    return this.jobForm.valid && (hasPageFields || hasValidItems);
  }

  private createSeedGroup(url = ''): FormGroup {
    return this.fb.group({
      url: [url, [Validators.required, Validators.pattern(/^https?:\/\/.+/)]]
    });
  }

  private createExtractionFieldGroup(field?: Partial<FieldSpec>): FormGroup {
    return this.fb.group({
      name: [field?.name || '', Validators.required],
      type: [((field?.type || 'string') as FieldSpec['type']), Validators.required],
      required: [field?.required ?? false],
      selector: [field?.extractor?.selector || '', Validators.required],
      attribute: [field?.extractor?.attribute || 'text'],
      multiple: [field?.extractor?.multiple ?? false],
      index: [field?.extractor?.multiple ? field?.extractor?.index ?? null : null],
      transforms: this.fb.array(
        field?.transforms?.map(t => this.createTransformGroup(t)) || []
      )
    });
  }

  private createTransformGroup(transform?: TransformSpec): FormGroup {
    return this.fb.group({
      op: [transform?.op || 'trim', Validators.required],
      arg: [transform?.arg || '']
    });
  }

  private createPaginationGroup(pagination?: Partial<PaginationSpec>): FormGroup {
    return this.fb.group({
      name: [pagination?.name || ''],
      selector: [pagination?.selector || '', Validators.required],
      attribute: [pagination?.attribute || 'href'],
      multiple: [pagination?.multiple ?? false]
    });
  }

  private resetArray(target: FormArray, items: (FormGroup | any)[]): void {
    while (target.length > 0) {
      target.removeAt(0);
    }
    items.forEach(item => target.push(item));
  }

  private buildConfig(): CrawlJobConfig {
    const raw: SimpleJobFormValue = this.jobForm.getRawValue();
    const extractionFields = this.buildFieldSpecs(this.extractionFields);
    const itemFields = this.buildFieldSpecs(this.itemFields);

    const paginationSpecs: PaginationSpec[] = this.pagination.controls.map(ctrl => {
      const value = ctrl.value;
      return {
        name: value.name || undefined,
        selector: value.selector,
        attribute: value.attribute || 'href',
        multiple: !!value.multiple
      };
    }).filter(p => p.selector);

    const hasItems = !!raw.items_enabled && raw.items_container_selector?.trim() && itemFields.length > 0;
    const itemsSpec: ItemsSpec | undefined = hasItems
      ? {
          container_selector: raw.items_container_selector.trim(),
          fields: itemFields
        }
      : undefined;

    return {
      name: raw.name,
      job_type: raw.job_type,
      crawl_mode: raw.crawl_mode,
      seeds: raw.seeds.map(seed => ({ url: seed.url })),
      scopes: {
        max_depth: Number(raw.max_depth),
        allowed_domains: raw.allowed_domains.filter(d => d && d.trim() !== ''),
        deny_url_patterns: raw.deny_url_patterns.filter(p => p && p.trim() !== ''),
        allowed_url_patterns: raw.allowed_url_patterns.filter(p => p && p.trim() !== '')
      },
      rate_limit: {
        rps: Number(raw.rps)
      },
      retries: raw.retries,
      auth: raw.auth,
      schedule: raw.schedule,
      extraction_spec: {
        fields: extractionFields,
        items: itemsSpec,
        pagination: paginationSpecs.length > 0 ? paginationSpecs : undefined
      }
    };
  }

  private buildFieldSpecs(fieldArray: FormArray): FieldSpec[] {
    return fieldArray.controls.map(ctrl => {
      const value = ctrl.value;
      const isMultiple = !!value.multiple;
      const extractor: any = {
        selector: value.selector,
        attribute: value.attribute,
        multiple: isMultiple
      };

      if (isMultiple) {
        extractor.index =
          value.index === null || value.index === undefined || value.index === ''
            ? null
            : Number(value.index);
      }

      return {
        name: value.name,
        type: value.type,
        required: !!value.required,
        extractor,
        transforms: value.transforms.filter((t: TransformSpec) => t?.op)
      };
    });
  }

  private updatePreview(): void {
    const config = this.buildConfig();
    this.previewJson = JSON.stringify({ config }, null, 2);
  }
}
