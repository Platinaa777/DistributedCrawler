import { Component, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatCardModule } from '@angular/material/card';
import { PreviewLoaderService } from '../../services/preview-loader.service';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { PreviewIframeComponent } from '../../components/preview-iframe/preview-iframe.component';

@Component({
  selector: 'app-url-preview-step',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatCardModule,
    PreviewIframeComponent
  ],
  template: `
    <div class="space-y-4">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Step 1: Load URL Preview</mat-card-title>
          <mat-card-subtitle>
            Enter the URL you want to crawl and inspect
          </mat-card-subtitle>
        </mat-card-header>

        <mat-card-content>
          <form [formGroup]="urlForm" (ngSubmit)="loadPreview()" class="space-y-4">
            <mat-form-field appearance="outline" class="w-full">
              <mat-label>Target URL</mat-label>
              <input
                matInput
                formControlName="url"
                placeholder="https://example.com"
                type="url"
              />
              <mat-icon matPrefix>link</mat-icon>
              <mat-error *ngIf="urlForm.get('url')?.hasError('required')">
                URL is required
              </mat-error>
              <mat-error *ngIf="urlForm.get('url')?.hasError('pattern')">
                Please enter a valid URL
              </mat-error>
            </mat-form-field>

            <div class="flex items-center gap-4">
              <button
                mat-raised-button
                color="primary"
                type="submit"
                [disabled]="urlForm.invalid || loading"
              >
                <mat-icon>refresh</mat-icon>
                Load Preview
              </button>

              <mat-spinner *ngIf="loading" diameter="24"></mat-spinner>

              <span *ngIf="previewHtml" class="text-sm text-green-600 flex items-center gap-1">
                <mat-icon class="text-sm">check_circle</mat-icon>
                Preview loaded
              </span>

              <span *ngIf="error" class="text-sm text-red-600 flex items-center gap-1">
                <mat-icon class="text-sm">error</mat-icon>
                {{ error }}
              </span>
            </div>
          </form>
        </mat-card-content>
      </mat-card>

      <mat-card *ngIf="previewHtml" class="flex-1">
        <mat-card-header>
          <mat-card-title class="text-base">HTML Preview</mat-card-title>
          <mat-card-subtitle class="text-xs">{{ finalUrl }}</mat-card-subtitle>
        </mat-card-header>
        <mat-card-content class="h-[500px]">
          <app-preview-iframe
            [html]="previewHtml"
            (frameReady)="onFrameReady($event)"
          ></app-preview-iframe>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    mat-form-field {
      width: 100%;
    }

    input[type="url"] {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
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
      ]]
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
    this.loading = true;
    this.error = null;

    this.previewLoader.loadPreview(url).subscribe({
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
