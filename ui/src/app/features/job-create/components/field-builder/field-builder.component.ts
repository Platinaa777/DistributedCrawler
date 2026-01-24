import { Component, Input, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormArray, Validators, ReactiveFormsModule } from '@angular/forms';
import { CardModule } from 'primeng/card';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { CheckboxModule } from 'primeng/checkbox';
import { ButtonModule } from 'primeng/button';
import { FieldSpec, TransformSpec } from '../../../../core/models/extraction-spec.model';

@Component({
  selector: 'app-field-builder',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    CardModule,
    InputTextModule,
    SelectModule,
    CheckboxModule,
    ButtonModule
  ],
  template: `
    <p-card styleClass="mb-4">
      <ng-template pTemplate="header">
        <div class="p-3 flex items-center justify-between">
          <div class="text-base font-semibold">Field: {{ fieldForm.value.name || 'Unnamed' }}</div>
          <p-button
            [text]="true"
            [rounded]="true"
            severity="danger"
            (onClick)="removeField()">
            <i class="pi pi-trash"></i>
          </p-button>
        </div>
      </ng-template>

      <form [formGroup]="fieldForm" class="p-4 space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Field Name</label>
            <input pInputText formControlName="name" placeholder="title" class="w-full" />
            <small *ngIf="fieldForm.get('name')?.hasError('required')" class="text-red-500">Name is required</small>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Type</label>
            <p-select
              [options]="typeOptions"
              optionLabel="label"
              optionValue="value"
              formControlName="type"
              styleClass="w-full">
            </p-select>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-6">
          <p-checkbox formControlName="required" [binary]="true" inputId="required-{{ fieldForm.value.name }}"></p-checkbox>
          <label for="required-{{ fieldForm.value.name }}" class="text-sm text-gray-700">Required</label>

          <div formGroupName="extractor" class="flex items-center gap-2">
            <p-checkbox formControlName="multiple" [binary]="true" inputId="multiple-{{ fieldForm.value.name }}"></p-checkbox>
            <label for="multiple-{{ fieldForm.value.name }}" class="text-sm text-gray-700">Multiple</label>
          </div>
        </div>

        <div class="border-t pt-4">
          <h4 class="text-sm font-semibold mb-3">Extractor Configuration</h4>
          <div formGroupName="extractor" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">CSS Selector</label>
              <input pInputText formControlName="selector" placeholder=".title" class="w-full" />
              <small *ngIf="fieldForm.get('extractor.selector')?.hasError('required')" class="text-red-500">
                Selector is required
              </small>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Attribute</label>
                <p-select
                  [options]="attributeOptions"
                  optionLabel="label"
                  optionValue="value"
                  formControlName="attribute"
                  styleClass="w-full">
                </p-select>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Default Value (Optional)</label>
                <input pInputText formControlName="default_value" class="w-full" />
              </div>
            </div>
          </div>
        </div>

        <div class="border-t pt-4">
          <div class="flex items-center justify-between mb-3">
            <h4 class="text-sm font-semibold">Transforms</h4>
            <p-button [outlined]="true" severity="secondary" (onClick)="addTransform()">
              <i class="pi pi-plus mr-2"></i>
              Add Transform
            </p-button>
          </div>

          <div formArrayName="transforms" class="space-y-3">
            <div
              *ngFor="let transform of transforms.controls; let i = index"
              [formGroupName]="i"
              class="transform-row"
            >
              <div class="transform-top">
                <div class="flex-1">
                  <label class="block text-sm font-medium text-gray-700 mb-1">Operation</label>
                  <p-select
                    [options]="transformOptions"
                    optionLabel="label"
                    optionValue="value"
                    formControlName="op"
                    styleClass="w-full">
                  </p-select>
                </div>

                <p-button
                  [outlined]="true"
                  severity="danger"
                  class="transform-remove"
                  (onClick)="removeTransform(i)">
                  <i class="pi pi-times mr-2"></i>
                  Remove
                </p-button>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Argument (Optional)</label>
                <input pInputText formControlName="arg" class="w-full" />
              </div>
            </div>
          </div>
        </div>
      </form>
    </p-card>
  `,
  styles: [`
    :host {
      display: block;
    }

    .transform-row {
      display: flex;
      flex-direction: column;
      gap: 10px;
      padding: 12px 14px;
      border: 1px solid #e5e7eb;
      border-radius: 10px;
      background: #f9fafb;
    }

    .transform-top {
      display: flex;
      gap: 12px;
      align-items: stretch;
    }

    .transform-remove {
      white-space: nowrap;
      padding: 0 12px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      min-height: 56px;
      box-sizing: border-box;
      font-size: 13px;
    }
  `]
})
export class FieldBuilderComponent implements OnInit {
  @Input() field?: FieldSpec;
  @Input() prefilledSelector?: string;
  @Output() fieldChange = new EventEmitter<FieldSpec>();
  @Output() remove = new EventEmitter<void>();

  fieldForm!: FormGroup;
  typeOptions = [
    { label: 'String', value: 'string' },
    { label: 'Integer', value: 'int' },
    { label: 'Float', value: 'float' },
    { label: 'Boolean', value: 'bool' },
    { label: 'URL', value: 'url' },
    { label: 'JSON', value: 'json' }
  ];
  attributeOptions = [
    { label: 'Text Content', value: 'text' },
    { label: 'href', value: 'href' },
    { label: 'src', value: 'src' },
    { label: 'alt', value: 'alt' },
    { label: 'title', value: 'title' },
    { label: 'data-*', value: 'data-*' }
  ];
  transformOptions = [
    { label: 'Trim', value: 'trim' },
    { label: 'Lowercase', value: 'lower' },
    { label: 'Uppercase', value: 'upper' },
    { label: 'Normalize URL', value: 'normalize_url' },
    { label: 'HTML to Text', value: 'html_to_text' },
    { label: 'Collapse Whitespace', value: 'collapse_ws' },
    { label: 'To Integer', value: 'to_int' },
    { label: 'To Float', value: 'to_float' },
    { label: 'Parse Price', value: 'parse_price' }
  ];

  constructor(private fb: FormBuilder) {}

  ngOnInit(): void {
    this.fieldForm = this.fb.group({
      name: [this.field?.name || '', Validators.required],
      type: [this.field?.type || 'string', Validators.required],
      required: [this.field?.required || false],
      label: [this.field?.label || ''],
      extractor: this.fb.group({
        source: ['html'],
        selector_type: ['css'],
        selector: [this.field?.extractor?.selector || this.prefilledSelector || '', Validators.required],
        attribute: [this.field?.extractor?.attribute || 'text'],
        multiple: [this.field?.extractor?.multiple || false],
        default_value: [this.field?.extractor?.default_value || '']
      }),
      transforms: this.fb.array(
        this.field?.transforms?.map(t => this.createTransform(t)) || []
      )
    });

    // Emit changes
    this.fieldForm.valueChanges.subscribe(value => {
      if (this.fieldForm.valid) {
        this.fieldChange.emit(this.buildFieldSpec());
      }
    });
  }

  get transforms(): FormArray {
    return this.fieldForm.get('transforms') as FormArray;
  }

  get multiple(): boolean {
    return this.fieldForm.get('extractor.multiple')?.value || false;
  }

  createTransform(transform?: TransformSpec): FormGroup {
    return this.fb.group({
      op: [transform?.op || 'trim', Validators.required],
      arg: [transform?.arg || '']
    });
  }

  addTransform(): void {
    this.transforms.push(this.createTransform());
  }

  removeTransform(index: number): void {
    this.transforms.removeAt(index);
  }

  removeField(): void {
    this.remove.emit();
  }

  buildFieldSpec(): FieldSpec {
    const formValue = this.fieldForm.value;
    return {
      name: formValue.name,
      type: formValue.type,
      required: formValue.required,
      label: formValue.label || undefined,
      extractor: {
        source: 'html',
        selector_type: 'css',
        selector: formValue.extractor.selector,
        attribute: formValue.extractor.attribute,
        multiple: formValue.extractor.multiple,
        default_value: formValue.extractor.default_value || undefined
      },
      transforms: formValue.transforms.filter((t: TransformSpec) => t.op)
    };
  }

  isValid(): boolean {
    return this.fieldForm.valid;
  }
}
