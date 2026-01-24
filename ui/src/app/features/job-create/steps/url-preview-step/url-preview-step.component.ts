import { Component, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { InputTextModule } from 'primeng/inputtext';
import { TextareaModule } from 'primeng/textarea';
import { ButtonModule } from 'primeng/button';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { CardModule } from 'primeng/card';
import { FloatLabelModule } from 'primeng/floatlabel';
import { IconFieldModule } from 'primeng/iconfield';
import { InputIconModule } from 'primeng/inputicon';
import { PreviewLoaderService } from '../../services/preview-loader.service';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { PreviewIframeComponent } from '../../components/preview-iframe/preview-iframe.component';

@Component({
  selector: 'app-url-preview-step',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    InputTextModule,
    TextareaModule,
    ButtonModule,
    ProgressSpinnerModule,
    CardModule,
    FloatLabelModule,
    IconFieldModule,
    InputIconModule,
    PreviewIframeComponent
  ],
  template: `
    <div class="space-y-4">
      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h2 class="text-xl font-semibold">Step 1: Load URL Preview</h2>
            <p class="text-sm text-gray-500">Enter the URL you want to crawl and inspect</p>
          </div>
        </ng-template>

        <form [formGroup]="urlForm" (ngSubmit)="loadPreview()" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Target URL</label>
            <p-iconfield>
              <p-inputicon styleClass="pi pi-link" />
              <input
                pInputText
                formControlName="url"
                placeholder="https://example.com"
                type="url"
                class="w-full" />
            </p-iconfield>
            <small *ngIf="urlForm.get('url')?.hasError('required') && urlForm.get('url')?.touched" class="text-red-500">
              URL is required
            </small>
            <small *ngIf="urlForm.get('url')?.hasError('pattern') && urlForm.get('url')?.touched" class="text-red-500">
              Please enter a valid URL
            </small>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Cookie (optional)</label>
            <textarea
              pTextarea
              formControlName="cookie"
              placeholder="Paste Cookie header value from your browser"
              rows="3"
              class="w-full"></textarea>
          </div>

          <div class="flex items-center gap-4">
            <p-button
              type="submit"
              [disabled]="urlForm.invalid || loading">
              <i class="pi pi-refresh mr-2"></i>
              Load Preview
            </p-button>

            <p-progressSpinner *ngIf="loading" [style]="{width: '24px', height: '24px'}" />

            <span *ngIf="previewHtml" class="text-sm text-green-600 flex items-center gap-1">
              <i class="pi pi-check-circle text-sm"></i>
              Preview loaded
            </span>

            <span *ngIf="error" class="text-sm text-red-600 flex items-center gap-1">
              <i class="pi pi-times-circle text-sm"></i>
              {{ error }}
            </span>
          </div>
        </form>
      </p-card>

      <p-card *ngIf="previewHtml" styleClass="flex-1">
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h3 class="text-base font-semibold">HTML Preview</h3>
            <p class="text-xs text-gray-500">{{ finalUrl }}</p>
          </div>
        </ng-template>
        <div class="h-[500px]">
          <app-preview-iframe
            [html]="previewHtml"
            (frameReady)="onFrameReady($event)">
          </app-preview-iframe>
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
export class UrlPreviewStepComponent implements OnInit {
  @Output() previewLoaded = new EventEmitter<HTMLIFrameElement>();

  urlForm: FormGroup;
  loading = false;
  error: string | null = null;
  previewHtml: string | null = null;
  finalUrl: string | null = null;

  constructor(
    private fb: FormBuilder,
    private previewLoader: PreviewLoaderService,
    private stateService: JobCreateStateService
  ) {
    this.urlForm = this.fb.group({
      url: ['', [
        Validators.required,
        Validators.pattern(/^https?:\/\/.+/)
      ]],
      cookie: ['']
    });
  }

  ngOnInit(): void {
    // Restore state if available
    const state = this.stateService.getCurrentState();
    if (state.previewUrl) {
      this.urlForm.patchValue({ url: state.previewUrl });
      this.previewHtml = state.previewHtml;
      this.finalUrl = state.previewUrl;
    }
  }

  loadPreview(): void {
    if (this.urlForm.invalid) {
      return;
    }

    const url = this.urlForm.value.url;
    const cookie = this.urlForm.value.cookie?.trim();
    this.loading = true;
    this.error = null;

    this.previewLoader.loadPreview(url, cookie || undefined).subscribe({
      next: (result) => {
        this.loading = false;
        this.previewHtml = result.html;
        this.finalUrl = result.url;
        this.error = null;

        // Explicitly save to state service to ensure persistence
        this.stateService.setPreview(url, result.previewId, result.html);

        console.log('UrlPreviewStep - preview loaded:', {
          url,
          previewId: result.previewId,
          htmlLength: result.html?.length || 0,
          hasHtml: !!result.html
        });
      },
      error: (err) => {
        this.loading = false;
        this.error = err.message || 'Failed to load preview';
        this.previewHtml = null;
      }
    });
  }

  onFrameReady(iframe: HTMLIFrameElement): void {
    this.previewLoaded.emit(iframe);
  }

  isValid(): boolean {
    return this.previewHtml !== null;
  }
}
