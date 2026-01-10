import { Component, Input, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MetricSpec } from '../../../../core/models/extraction-spec.model';

@Component({
  selector: 'app-metric-builder',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatAutocompleteModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule
  ],
  template: `
    <mat-card class="mb-4">
      <mat-card-content>
        <form [formGroup]="metricForm" class="space-y-4">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <mat-form-field appearance="fill">
              <mat-label>Metric Name</mat-label>
              <input matInput formControlName="name" placeholder="total_items" />
              <mat-error *ngIf="metricForm.get('name')?.hasError('required')">
                Name is required
              </mat-error>
            </mat-form-field>

            <mat-form-field appearance="fill">
              <mat-label>Operation</mat-label>
              <mat-select formControlName="op">
                <mat-option value="len">Length</mat-option>
                <mat-option value="count">Count</mat-option>
                <mat-option value="word_count">Word Count</mat-option>
                <mat-option value="field_present">Field Present</mat-option>
                <mat-option value="status_is_error">Status Is Error</mat-option>
                <mat-option value="count_external_links">Count External Links</mat-option>
              </mat-select>
              <mat-error *ngIf="metricForm.get('op')?.hasError('required')">
                Operation is required
              </mat-error>
            </mat-form-field>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <mat-form-field appearance="fill">
              <mat-label>Input Field</mat-label>
              <input
                matInput
                formControlName="input"
                placeholder="field_name"
                [matAutocomplete]="fieldAuto"
              />
              <mat-autocomplete #fieldAuto="matAutocomplete" autoActiveFirstOption>
                <mat-option *ngFor="let field of availableFields" [value]="field">
                  {{ field }}
                </mat-option>
              </mat-autocomplete>
              <mat-error *ngIf="metricForm.get('input')?.hasError('required')">
                Input is required
              </mat-error>
            </mat-form-field>

            <mat-form-field appearance="fill">
              <mat-label>Argument (Optional)</mat-label>
              <input matInput formControlName="arg" />
            </mat-form-field>
          </div>

          <div class="flex justify-end">
            <button mat-stroked-button color="warn" (click)="removeMetric()" type="button">
              <mat-icon>delete</mat-icon>
              Remove Metric
            </button>
          </div>
        </form>
      </mat-card-content>
    </mat-card>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class MetricBuilderComponent implements OnInit {
  @Input() metric?: MetricSpec;
  @Input() availableFields: string[] = [];
  @Output() metricChange = new EventEmitter<MetricSpec>();
  @Output() remove = new EventEmitter<void>();

  metricForm!: FormGroup;

  constructor(private fb: FormBuilder) {}

  ngOnInit(): void {
    this.metricForm = this.fb.group({
      name: [this.metric?.name || '', Validators.required],
      op: [this.metric?.op || 'count', Validators.required],
      input: [this.metric?.input || '', Validators.required],
      arg: [this.metric?.arg || '']
    });

    // Emit changes
    this.metricForm.valueChanges.subscribe(() => {
      if (this.metricForm.valid) {
        this.metricChange.emit(this.buildMetricSpec());
      }
    });
  }

  removeMetric(): void {
    this.remove.emit();
  }

  buildMetricSpec(): MetricSpec {
    const formValue = this.metricForm.value;
    return {
      name: formValue.name,
      op: formValue.op,
      input: formValue.input,
      arg: formValue.arg || undefined
    };
  }

  isValid(): boolean {
    return this.metricForm.valid;
  }
}
