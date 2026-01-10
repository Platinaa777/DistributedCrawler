import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTabsModule } from '@angular/material/tabs';
import { FieldBuilderComponent } from '../../components/field-builder/field-builder.component';
import { MetricBuilderComponent } from '../../components/metric-builder/metric-builder.component';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { FieldSpec, MetricSpec } from '../../../../core/models/extraction-spec.model';

@Component({
  selector: 'app-extraction-spec-step',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatTabsModule,
    FieldBuilderComponent,
    MetricBuilderComponent
  ],
  template: `
    <div class="space-y-4">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Step 3: Build Extraction Spec</mat-card-title>
          <mat-card-subtitle>
            Define fields and metrics to extract from pages
          </mat-card-subtitle>
        </mat-card-header>
      </mat-card>

      <mat-tab-group>
        <mat-tab label="Fields ({{ fields.length }})">
          <div class="p-4 space-y-4">
            <div class="flex items-center justify-between mb-4">
              <p class="text-sm text-gray-600">
            Define data fields to extract from each page
          </p>
          <button mat-raised-button color="primary" (click)="addField()">
            <mat-icon>add</mat-icon>
            Add Field
          </button>
        </div>

        <div *ngIf="fields.length === 0" class="text-center py-12 bg-gray-50 rounded">
          <mat-icon class="text-gray-400 text-5xl mb-2">data_object</mat-icon>
          <p class="text-gray-500">No fields defined yet</p>
          <p class="text-gray-400 text-sm mt-1">Add a field to start extracting data</p>
            </div>

            <app-field-builder
              *ngFor="let field of fields; let i = index"
              [field]="field"
              (fieldChange)="updateField(i, $event)"
              (remove)="removeField(i)"
            ></app-field-builder>
          </div>
        </mat-tab>

        <mat-tab label="Metrics ({{ metrics.length }})">
          <div class="p-4 space-y-4">
            <div class="flex items-center justify-between mb-4">
              <p class="text-sm text-gray-600">
                Define metrics to calculate from extracted data
              </p>
              <button mat-raised-button color="primary" (click)="addMetric()">
                <mat-icon>add</mat-icon>
                Add Metric
              </button>
            </div>

            <div *ngIf="metrics.length === 0" class="text-center py-12 bg-gray-50 rounded">
              <mat-icon class="text-gray-400 text-5xl mb-2">analytics</mat-icon>
              <p class="text-gray-500">No metrics defined yet</p>
              <p class="text-gray-400 text-sm mt-1">Add a metric to track data quality</p>
            </div>

            <app-metric-builder
              *ngFor="let metric of metrics; let i = index"
              [metric]="metric"
              (metricChange)="updateMetric(i, $event)"
              (remove)="removeMetric(i)"
            ></app-metric-builder>
          </div>
        </mat-tab>
      </mat-tab-group>

      <mat-card>
        <mat-card-header>
          <mat-card-title class="text-base">Trial Run</mat-card-title>
          <mat-card-subtitle>
            Check what the backend would extract with the current spec
          </mat-card-subtitle>
        </mat-card-header>
        <mat-card-content class="space-y-3">
          <button mat-raised-button color="primary" (click)="runTrial()">
            <mat-icon>play_arrow</mat-icon>
            Check
          </button>
          <div *ngIf="trialResult" class="border rounded bg-gray-50 p-3">
            <p class="text-xs font-semibold text-gray-600 mb-2">Trial Result (mock)</p>
            <pre class="text-xs text-gray-800 overflow-auto">{{ trialResult }}</pre>
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
export class ExtractionSpecStepComponent implements OnInit {
  fields: FieldSpec[] = [];
  metrics: MetricSpec[] = [];
  trialResult: string | null = null;

  constructor(private stateService: JobCreateStateService) {}

  ngOnInit(): void {
    // Load from state
    const state = this.stateService.getCurrentState();
    this.fields = [...state.extractionSpec.fields];
    this.metrics = [...state.extractionSpec.metrics];
  }

  addField(): void {
    const newField: FieldSpec = {
      name: `field_${this.fields.length + 1}`,
      type: 'string',
      required: false,
      extractor: {
        source: 'html',
        selector_type: 'css',
        selector: '',
        attribute: 'text',
        multiple: false
      },
      transforms: []
    };

    this.fields.push(newField);
    this.stateService.addField(newField);
  }

  updateField(index: number, field: FieldSpec): void {
    this.fields[index] = field;
    this.stateService.updateField(index, field);
  }

  removeField(index: number): void {
    this.fields.splice(index, 1);
    this.stateService.removeField(index);
  }

  addMetric(): void {
    const newMetric: MetricSpec = {
      name: `metric_${this.metrics.length + 1}`,
      op: 'count',
      input: ''
    };

    this.metrics.push(newMetric);
    this.stateService.addMetric(newMetric);
  }

  updateMetric(index: number, metric: MetricSpec): void {
    this.metrics[index] = metric;
    // Note: State service doesn't have updateMetric, we'll need to remove and re-add
    this.stateService.removeMetric(index);
    this.stateService.addMetric(metric);
  }

  removeMetric(index: number): void {
    this.metrics.splice(index, 1);
    this.stateService.removeMetric(index);
  }

  runTrial(): void {
    const payload = {
      fields: this.fields,
      metrics: this.metrics
    };

    console.log('Trial run payload:', payload);

    this.trialResult = JSON.stringify({
      status: 'ok',
      fields_count: this.fields.length,
      metrics_count: this.metrics.length,
      sample: this.fields.slice(0, 2).map(field => ({
        name: field.name,
        selector: field.extractor.selector,
        value: '(mocked)'
      }))
    }, null, 2);
  }

  isValid(): boolean {
    return this.fields.length > 0;
  }
}
