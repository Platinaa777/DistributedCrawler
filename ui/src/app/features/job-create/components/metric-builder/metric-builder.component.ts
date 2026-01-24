import { Component, Input, Output, EventEmitter, OnInit, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { CardModule } from 'primeng/card';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { ButtonModule } from 'primeng/button';
import { MetricSpec } from '../../../../core/models/extraction-spec.model';

@Component({
  selector: 'app-metric-builder',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    CardModule,
    InputTextModule,
    SelectModule,
    ButtonModule
  ],
  template: `
    <p-card styleClass="mb-4">
      <form [formGroup]="metricForm" class="p-4 space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Metric Name</label>
            <input pInputText formControlName="name" placeholder="total_items" class="w-full" />
            <small *ngIf="metricForm.get('name')?.hasError('required')" class="text-red-500">
              Name is required
            </small>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Operation</label>
            <p-select
              [options]="operationOptions"
              optionLabel="label"
              optionValue="value"
              formControlName="op"
              styleClass="w-full">
            </p-select>
            <small *ngIf="metricForm.get('op')?.hasError('required')" class="text-red-500">
              Operation is required
            </small>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Input Field</label>
            <p-select
              [options]="fieldSelectOptions"
              optionLabel="label"
              optionValue="value"
              formControlName="input"
              styleClass="w-full"
              placeholder="Select field">
            </p-select>
            <small *ngIf="!fieldOptions.length" class="text-xs text-gray-500">Add a field to enable metrics</small>
            <small *ngIf="metricForm.get('input')?.hasError('required')" class="text-red-500 block">
              Input is required
            </small>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Argument (Optional)</label>
            <input pInputText formControlName="arg" class="w-full" />
          </div>
        </div>

        <div class="flex justify-end">
          <p-button
            [text]="true"
            [rounded]="true"
            severity="danger"
            (onClick)="removeMetric()">
            <i class="pi pi-trash"></i>
          </p-button>
        </div>
      </form>
    </p-card>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class MetricBuilderComponent implements OnInit, OnChanges {
  @Input() metric?: MetricSpec;
  @Input() availableFields: string[] = [];
  @Output() metricChange = new EventEmitter<MetricSpec>();
  @Output() remove = new EventEmitter<void>();

  metricForm!: FormGroup;
  fieldOptions: string[] = [];
  operationOptions = [
    { label: 'Length', value: 'len' },
    { label: 'Count', value: 'count' },
    { label: 'Word Count', value: 'word_count' },
    { label: 'Field Present', value: 'field_present' },
    { label: 'Status Is Error', value: 'status_is_error' },
    { label: 'Count External Links', value: 'count_external_links' }
  ];
  fieldSelectOptions: { label: string; value: string }[] = [];

  constructor(private fb: FormBuilder) {}

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['metric'] && this.metricForm) {
      this.metricForm.patchValue({
        name: this.metric?.name || '',
        op: this.metric?.op || 'count',
        input: this.metric?.input || '',
        arg: this.metric?.arg || ''
      }, { emitEvent: false });
    }

    if (changes['availableFields'] && this.metricForm) {
      this.refreshFieldOptions();
    }
  }

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

    this.refreshFieldOptions();
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

  private refreshFieldOptions(): void {
    this.fieldOptions = [...this.availableFields];
    this.fieldSelectOptions = this.fieldOptions.map(field => ({ label: field, value: field }));

    if (!this.metricForm) {
      return;
    }

    const currentInput = this.metricForm.get('input')?.value;
    const isValid = currentInput && this.fieldOptions.includes(currentInput);

    if (!isValid && currentInput) {
      this.metricForm.get('input')?.setValue('', { emitEvent: false });
      this.metricChange.emit(this.buildMetricSpec());
    }
  }
}
