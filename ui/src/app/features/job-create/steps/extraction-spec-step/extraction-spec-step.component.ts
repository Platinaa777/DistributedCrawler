import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { TabViewModule } from 'primeng/tabview';
import { FieldBuilderComponent } from '../../components/field-builder/field-builder.component';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { FieldSpec } from '../../../../core/models/extraction-spec.model';

@Component({
  selector: 'app-extraction-spec-step',
  standalone: true,
  imports: [
    CommonModule,
    CardModule,
    ButtonModule,
    TabViewModule,
    FieldBuilderComponent
  ],
  template: `
    <div class="space-y-4">
      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h2 class="text-xl font-semibold">Step 3: Build Extraction Spec</h2>
            <p class="text-sm text-gray-500">Define fields to extract from pages.</p>
          </div>
        </ng-template>
      </p-card>

      <p-tabView>
        <p-tabPanel header="Fields ({{ fields.length }})">
          <div class="p-4 space-y-4">
            <div class="flex items-center justify-between mb-4">
              <p class="text-sm text-gray-600">Define data fields to extract from each page.</p>
              <p-button (onClick)="addField()">
                <i class="pi pi-plus mr-2"></i>
                Add Field
              </p-button>
            </div>

            <div *ngIf="fields.length === 0" class="text-center py-12 bg-gray-50 rounded">
              <i class="pi pi-database text-gray-400 text-4xl mb-2"></i>
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
        </p-tabPanel>

      </p-tabView>

      <p-card>
        <ng-template pTemplate="header">
          <div class="p-4 pb-0">
            <h3 class="text-base font-semibold">Trial Run</h3>
            <p class="text-sm text-gray-500">Check what the backend would extract with the current spec.</p>
          </div>
        </ng-template>
        <div class="p-4 space-y-3">
          <p-button (onClick)="runTrial()">
            <i class="pi pi-play mr-2"></i>
            Check
          </p-button>
          <div *ngIf="trialResult" class="border rounded bg-gray-50 p-3">
            <p class="text-xs font-semibold text-gray-600 mb-2">Trial Result (mock)</p>
            <pre class="text-xs text-gray-800 overflow-auto">{{ trialResult }}</pre>
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
export class ExtractionSpecStepComponent implements OnInit {
  fields: FieldSpec[] = [];
  trialResult: string | null = null;

  constructor(private stateService: JobCreateStateService) {}

  ngOnInit(): void {
    // Load from state
    const state = this.stateService.getCurrentState();
    this.fields = [...state.extractionSpec.fields];
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

  runTrial(): void {
    console.log('Trial run payload:', { fields: this.fields });

    this.trialResult = JSON.stringify({
      status: 'ok',
      fields_count: this.fields.length,
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
