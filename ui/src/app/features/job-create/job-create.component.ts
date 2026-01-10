import { Component, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatStepperModule } from '@angular/material/stepper';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { UrlPreviewStepComponent } from './steps/url-preview-step/url-preview-step.component';
import { ElementPickerStepComponent } from './steps/element-picker-step/element-picker-step.component';
import { ExtractionSpecStepComponent } from './steps/extraction-spec-step/extraction-spec-step.component';
import { JobSettingsStepComponent } from './steps/job-settings-step/job-settings-step.component';
import { ReviewCreateStepComponent } from './steps/review-create-step/review-create-step.component';

@Component({
  selector: 'app-job-create',
  standalone: true,
  imports: [
    CommonModule,
    MatStepperModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    UrlPreviewStepComponent,
    ElementPickerStepComponent,
    ExtractionSpecStepComponent,
    JobSettingsStepComponent,
    ReviewCreateStepComponent
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="mb-4">
        <button mat-button (click)="goBack()">
          <mat-icon>arrow_back</mat-icon>
          Back to Jobs
        </button>
      </div>

      <mat-card class="mb-4">
        <mat-card-header>
          <mat-card-title class="text-2xl">Create New Crawl Job</mat-card-title>
          <mat-card-subtitle>
            Follow the wizard to configure and launch your crawl job
          </mat-card-subtitle>
        </mat-card-header>
      </mat-card>

      <mat-stepper #stepper linear>
        <mat-step [completed]="previewStep?.isValid()">
          <ng-template matStepLabel>URL Preview</ng-template>
          <app-url-preview-step #previewStep></app-url-preview-step>
          <div class="flex justify-end mt-4 gap-2">
            <button mat-raised-button matStepperNext [disabled]="!previewStep?.isValid()">
              Next
              <mat-icon>navigate_next</mat-icon>
            </button>
          </div>
        </mat-step>

        <mat-step [completed]="pickerStep?.isValid()">
          <ng-template matStepLabel>Pick Elements</ng-template>
          <app-element-picker-step #pickerStep></app-element-picker-step>
          <div class="flex justify-between mt-4">
            <button mat-button matStepperPrevious>
              <mat-icon>navigate_before</mat-icon>
              Previous
            </button>
            <button mat-raised-button matStepperNext [disabled]="!pickerStep?.isValid()">
              Next
              <mat-icon>navigate_next</mat-icon>
            </button>
          </div>
        </mat-step>

        <mat-step [completed]="extractionStep?.isValid()">
          <ng-template matStepLabel>Extraction Spec</ng-template>
          <app-extraction-spec-step #extractionStep></app-extraction-spec-step>
          <div class="flex justify-between mt-4">
            <button mat-button matStepperPrevious>
              <mat-icon>navigate_before</mat-icon>
              Previous
            </button>
            <button mat-raised-button matStepperNext [disabled]="!extractionStep?.isValid()">
              Next
              <mat-icon>navigate_next</mat-icon>
            </button>
          </div>
        </mat-step>

        <mat-step [completed]="settingsStep?.isValid()">
          <ng-template matStepLabel>Job Settings</ng-template>
          <app-job-settings-step #settingsStep></app-job-settings-step>
          <div class="flex justify-between mt-4">
            <button mat-button matStepperPrevious>
              <mat-icon>navigate_before</mat-icon>
              Previous
            </button>
            <button mat-raised-button matStepperNext [disabled]="!settingsStep?.isValid()">
              Next
              <mat-icon>navigate_next</mat-icon>
            </button>
          </div>
        </mat-step>

        <mat-step>
          <ng-template matStepLabel>Review & Create</ng-template>
          <app-review-create-step (jobCreated)="onJobCreated($event)"></app-review-create-step>
          <div class="flex justify-start mt-4">
            <button mat-button matStepperPrevious>
              <mat-icon>navigate_before</mat-icon>
              Previous
            </button>
          </div>
        </mat-step>
      </mat-stepper>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    mat-stepper {
      background: transparent;
    }
  `]
})
export class JobCreateComponent {
  @ViewChild('previewStep') previewStep?: UrlPreviewStepComponent;
  @ViewChild('pickerStep') pickerStep?: ElementPickerStepComponent;
  @ViewChild('extractionStep') extractionStep?: ExtractionSpecStepComponent;
  @ViewChild('settingsStep') settingsStep?: JobSettingsStepComponent;

  constructor(private router: Router) {}

  goBack(): void {
    this.router.navigate(['/jobs']);
  }

  onJobCreated(jobId: string): void {
    // Navigation is handled by the review-create step
    console.log('Job created with ID:', jobId);
  }
}
