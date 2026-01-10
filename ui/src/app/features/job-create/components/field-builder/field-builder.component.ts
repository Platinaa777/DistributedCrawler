import { Component, Input, Output, EventEmitter, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormArray, Validators, ReactiveFormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { FieldSpec, TransformSpec } from '../../../../core/models/extraction-spec.model';

@Component({
  selector: 'app-field-builder',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatChipsModule
  ],
  template: `
    <mat-card class="mb-4">
      <mat-card-header>
        <mat-card-title class="text-base flex items-center justify-between w-full">
          <span>Field: {{ fieldForm.value.name || 'Unnamed' }}</span>
          <button mat-icon-button color="warn" (click)="removeField()" type="button">
            <mat-icon>delete</mat-icon>
          </button>
        </mat-card-title>
      </mat-card-header>

      <mat-card-content>
        <form [formGroup]="fieldForm" class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <mat-form-field appearance="fill">
              <mat-label>Field Name</mat-label>
              <input matInput formControlName="name" placeholder="title" />
              <mat-error *ngIf="fieldForm.get('name')?.hasError('required')">
                Name is required
              </mat-error>
            </mat-form-field>

            <mat-form-field appearance="fill">
              <mat-label>Type</mat-label>
              <mat-select formControlName="type">
                <mat-option value="string">String</mat-option>
                <mat-option value="int">Integer</mat-option>
                <mat-option value="float">Float</mat-option>
                <mat-option value="bool">Boolean</mat-option>
                <mat-option value="url">URL</mat-option>
                <mat-option value="json">JSON</mat-option>
              </mat-select>
            </mat-form-field>
          </div>

          <div class="flex items-center gap-4">
            <mat-checkbox formControlName="required">Required</mat-checkbox>
            <div formGroupName="extractor" class="flex items-center gap-4">
              <mat-checkbox formControlName="multiple">Multiple</mat-checkbox>
            </div>
          </div>

          <div class="border-t pt-4">
            <h4 class="text-sm font-semibold mb-3">Extractor Configuration</h4>
            <div formGroupName="extractor" class="space-y-4">
              <mat-form-field appearance="fill" class="w-full">
                <mat-label>CSS Selector</mat-label>
                <input matInput formControlName="selector" placeholder=".title" />
                <mat-error *ngIf="fieldForm.get('extractor.selector')?.hasError('required')">
                  Selector is required
                </mat-error>
              </mat-form-field>

              <div class="grid grid-cols-2 gap-4">
                <mat-form-field appearance="fill">
                  <mat-label>Attribute</mat-label>
                  <mat-select formControlName="attribute">
                    <mat-option value="text">Text Content</mat-option>
                    <mat-option value="href">href</mat-option>
                    <mat-option value="src">src</mat-option>
                    <mat-option value="alt">alt</mat-option>
                    <mat-option value="title">title</mat-option>
                    <mat-option value="data-*">data-*</mat-option>
                  </mat-select>
                </mat-form-field>

                <mat-form-field appearance="fill">
                  <mat-label>Default Value (Optional)</mat-label>
                  <input matInput formControlName="default_value" />
                </mat-form-field>
              </div>
            </div>
          </div>

          <div class="border-t pt-4">
            <div class="flex items-center justify-between mb-3">
              <h4 class="text-sm font-semibold">Transforms</h4>
              <button mat-stroked-button (click)="addTransform()" type="button">
                <mat-icon>add</mat-icon>
                Add Transform
              </button>
            </div>

            <div formArrayName="transforms" class="space-y-3">
              <div
                *ngFor="let transform of transforms.controls; let i = index"
                [formGroupName]="i"
                class="transform-row"
              >
                <div class="transform-top">
                  <mat-form-field appearance="fill" class="flex-1">
                    <mat-label>Operation</mat-label>
                    <mat-select formControlName="op">
                      <mat-option value="trim">Trim</mat-option>
                      <mat-option value="lower">Lowercase</mat-option>
                      <mat-option value="upper">Uppercase</mat-option>
                      <mat-option value="normalize_url">Normalize URL</mat-option>
                      <mat-option value="html_to_text">HTML to Text</mat-option>
                      <mat-option value="collapse_ws">Collapse Whitespace</mat-option>
                      <mat-option value="to_int">To Integer</mat-option>
                      <mat-option value="to_float">To Float</mat-option>
                      <mat-option value="parse_price">Parse Price</mat-option>
                    </mat-select>
                  </mat-form-field>

                  <button
                    mat-stroked-button
                    color="warn"
                    class="transform-remove"
                    (click)="removeTransform(i)"
                    type="button"
                  >
                    <mat-icon>close</mat-icon>
                    Remove
                  </button>
                </div>

                <mat-form-field appearance="fill" class="w-full">
                  <mat-label>Argument (Optional)</mat-label>
                  <input matInput formControlName="arg" />
                </mat-form-field>
              </div>
            </div>
          </div>
        </form>
      </mat-card-content>
    </mat-card>
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
