import { Component, ViewChild, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { StepsModule } from 'primeng/steps';
import { MenuItem } from 'primeng/api';
import { UrlPreviewStepComponent } from './steps/url-preview-step/url-preview-step.component';
import { ElementPickerStepComponent } from './steps/element-picker-step/element-picker-step.component';
import { JobSettingsStepComponent } from './steps/job-settings-step/job-settings-step.component';
import { ReviewCreateStepComponent } from './steps/review-create-step/review-create-step.component';

@Component({
  selector: 'app-job-create',
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    ButtonModule,
    StepsModule,
    UrlPreviewStepComponent,
    ElementPickerStepComponent,
    JobSettingsStepComponent,
    ReviewCreateStepComponent
  ],
  template: `
    <div class="container mx-auto p-6 space-y-4">
      <div>
        <p-button [text]="true" (onClick)="goBack()">
          <i class="pi pi-arrow-left mr-2"></i>
          Back to Jobs
        </p-button>
      </div>

      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h2 class="text-2xl font-semibold">Create New Crawl Job</h2>
            <p class="text-sm text-gray-500">Follow the wizard to configure and launch your crawl job.</p>
          </div>
        </ng-template>
      </p-card>

      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <p-steps [model]="steps" [activeIndex]="activeStep" styleClass="w-full"></p-steps>
          </div>
        </ng-template>

        <div class="p-4">
          <ng-container [ngSwitch]="activeStep">
            <app-url-preview-step *ngSwitchCase="0" #previewStep></app-url-preview-step>
            <app-element-picker-step *ngSwitchCase="1" #pickerStep></app-element-picker-step>
            <app-job-settings-step *ngSwitchCase="2" #settingsStep></app-job-settings-step>
            <app-review-create-step *ngSwitchCase="3" (jobCreated)="onJobCreated($event)"></app-review-create-step>
          </ng-container>

          <div class="flex items-center justify-between mt-6">
            <p-button
              [outlined]="true"
              severity="secondary"
              (onClick)="prevStep()"
              [disabled]="activeStep === 0">
              <i class="pi pi-chevron-left mr-2"></i>
              Previous
            </p-button>

            <p-button
              *ngIf="activeStep < 3"
              (onClick)="nextStep()"
              [disabled]="!canProceed(activeStep)">
              Next
              <i class="pi pi-chevron-right ml-2"></i>
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
export class JobCreateComponent implements OnInit {
  @ViewChild('previewStep') previewStep?: UrlPreviewStepComponent;
  @ViewChild('pickerStep') pickerStep?: ElementPickerStepComponent;
  @ViewChild('settingsStep') settingsStep?: JobSettingsStepComponent;

  activeStep = 0;
  steps: MenuItem[] = [];

  constructor(private router: Router) {}

  goBack(): void {
    this.router.navigate(['/jobs']);
  }

  onJobCreated(jobId: string): void {
    // Navigation is handled by the review-create step
    console.log('Job created with ID:', jobId);
  }

  canProceed(stepIndex: number): boolean {
    switch (stepIndex) {
      case 0:
        return this.previewStep?.isValid() ?? false;
      case 1:
        return this.pickerStep?.isValid() ?? false;
      case 2:
        return this.settingsStep?.isValid() ?? false;
      default:
        return true;
    }
  }

  nextStep(): void {
    if (this.activeStep >= 3) return;
    if (!this.canProceed(this.activeStep)) return;
    this.activeStep += 1;
  }

  prevStep(): void {
    if (this.activeStep <= 0) return;
    this.activeStep -= 1;
  }

  private setActiveStep(index: number): void {
    if (index === this.activeStep) {
      return;
    }

    if (index < this.activeStep) {
      this.activeStep = index;
      return;
    }

    if (index === this.activeStep + 1 && this.canProceed(this.activeStep)) {
      this.activeStep = index;
    }
  }

  private buildSteps(): MenuItem[] {
    return [
      {
        label: 'URL Preview',
        command: () => this.setActiveStep(0)
      },
      {
        label: 'Element Picker',
        command: () => this.setActiveStep(1)
      },
      {
        label: 'Job Settings',
        command: () => this.setActiveStep(2)
      },
      {
        label: 'Review & Create',
        command: () => this.setActiveStep(3)
      }
    ];
  }

  ngOnInit(): void {
    this.steps = this.buildSteps();
  }
}
