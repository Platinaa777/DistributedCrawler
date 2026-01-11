import { Component, OnInit, OnDestroy, ViewChild, ElementRef, NgZone } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { PreviewIframeComponent } from '../../components/preview-iframe/preview-iframe.component';
import { JobCreateStateService } from '../../services/job-create-state.service';
import { SelectorGeneratorService } from '../../../../core/services/selector-generator.service';
import { FieldBuilderComponent } from '../../components/field-builder/field-builder.component';
import { MetricBuilderComponent } from '../../components/metric-builder/metric-builder.component';
import { FieldSpec, MetricSpec } from '../../../../core/models/extraction-spec.model';
import { Subscription } from 'rxjs';

interface PickerElementData {
  selector: string;
  value: string;
  attribute: string;
  elementTag: string;
}

@Component({
  selector: 'app-element-picker-step',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatSlideToggleModule,
    PreviewIframeComponent,
    FieldBuilderComponent,
    MetricBuilderComponent
  ],
  template: `
    <div class="space-y-4 h-full">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Step 2: Pick Elements & Build Spec</mat-card-title>
          <mat-card-subtitle>
            Hover and click on elements in the preview to extract data and build your extraction spec
          </mat-card-subtitle>
        </mat-card-header>
        <mat-card-content class="space-y-4">
          <div class="flex items-center gap-4">
            <mat-slide-toggle
              [(ngModel)]="pickerEnabled"
              color="primary"
              (change)="togglePicker()"
            >
              {{ pickerEnabled ? 'Picker Active' : 'Picker Inactive' }}
            </mat-slide-toggle>

            <div class="text-sm text-gray-600">
              <mat-icon class="text-sm align-middle">info</mat-icon>
              {{
                pickerEnabled
                  ? 'Hover over elements and click to select'
                  : 'Enable picker to select elements'
              }}
            </div>
          </div>
        </mat-card-content>
      </mat-card>

      <div class="layout-grid">
        <mat-card class="fill-card preview-card">
          <mat-card-header>
            <mat-card-title class="text-base">Page Preview</mat-card-title>
          </mat-card-header>
          <mat-card-content class="relative flex-1 min-h-0">
            <div #previewContainer class="relative h-full min-h-[360px]">
              <app-preview-iframe
                [html]="previewHtml"
                (frameReady)="onFrameReady($event)"
              ></app-preview-iframe>

              <div
                *ngIf="pickerEnabled"
                class="absolute inset-0 z-10 cursor-crosshair"
                (mousemove)="onOverlayMouseMove($event)"
                (click)="onOverlayClick($event)"
                (wheel)="onOverlayWheel($event)"
              ></div>

              <div
                *ngIf="pickerEnabled && highlightBox"
                class="absolute z-20 pointer-events-none border-2 border-blue-500 bg-blue-500 bg-opacity-10"
                [style.left.px]="highlightBox.left"
                [style.top.px]="highlightBox.top"
                [style.width.px]="highlightBox.width"
                [style.height.px]="highlightBox.height"
              >
                <div class="absolute -top-6 left-0 bg-blue-500 text-white text-xs px-2 py-1 rounded">
                  {{ hoveredElement?.elementTag }}
                </div>
              </div>
            </div>
          </mat-card-content>
        </mat-card>

        <mat-card class="fill-card">
          <mat-card-header class="flex items-center justify-between gap-3">
            <div>
              <mat-card-title class="text-base">Fields ({{ fields.length }})</mat-card-title>
              <mat-card-subtitle>Define data fields to extract</mat-card-subtitle>
            </div>
            <button mat-stroked-button color="primary" (click)="addField()">
              <mat-icon>add</mat-icon>
              Add Field
            </button>
          </mat-card-header>
          <mat-card-content class="card-content-scroll">
            <div class="flex-1 overflow-y-auto space-y-3 pr-1">
              <div
                *ngIf="fields.length === 0"
                class="text-center py-10 bg-gray-50 rounded border border-dashed border-gray-200"
              >
                <mat-icon class="text-gray-400 text-5xl mb-2">data_object</mat-icon>
                <p class="text-gray-500">No fields defined yet</p>
                <p class="text-gray-400 text-sm mt-1">Click elements in the preview or add one manually</p>
              </div>

              <app-field-builder
                *ngFor="let field of fields; let i = index"
                [field]="field"
                (fieldChange)="updateField(i, $event)"
                (remove)="removeField(i)"
              ></app-field-builder>
            </div>
          </mat-card-content>
        </mat-card>

        <mat-card class="fill-card">
          <mat-card-header class="flex items-center justify-between gap-3">
            <div>
              <mat-card-title class="text-base">Metrics ({{ metrics.length }})</mat-card-title>
              <mat-card-subtitle>Define metrics to calculate from extracted data</mat-card-subtitle>
            </div>
            <button mat-stroked-button color="primary" (click)="addMetric()">
              <mat-icon>add</mat-icon>
              Add Metric
            </button>
          </mat-card-header>
          <mat-card-content class="card-content-scroll space-y-3">
            <div class="flex-1 overflow-y-auto space-y-3 pr-1">
              <div
                *ngIf="metrics.length === 0"
                class="text-center py-10 bg-gray-50 rounded border border-dashed border-gray-200"
              >
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

            <div class="space-y-2">
              <button mat-raised-button color="primary" (click)="runTrial()">
                <mat-icon>play_arrow</mat-icon>
                Check (mock)
              </button>
              <div *ngIf="trialResult" class="border rounded bg-gray-50 p-3 max-h-48 overflow-auto">
                <p class="text-xs font-semibold text-gray-600 mb-2">Trial Result</p>
                <pre class="text-xs text-gray-800">{{ trialResult }}</pre>
              </div>
            </div>
          </mat-card-content>
        </mat-card>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
      height: 100%;
      min-height: 0;
    }

    .layout-grid {
      display: grid;
      grid-template-columns: 2fr 1fr;
      grid-auto-rows: minmax(0, 1fr);
      gap: 1rem;
      min-height: 720px;
      height: calc(100vh - 280px);
    }

    .preview-card {
      grid-row: span 2;
    }

    .fill-card {
      display: flex;
      flex-direction: column;
      height: 100%;
      min-height: 0;
    }

    .fill-card mat-card-content {
      display: flex;
      flex-direction: column;
      flex: 1;
      min-height: 0;
    }
  `]
})
export class ElementPickerStepComponent implements OnInit, OnDestroy {
  previewHtml: string | null = null;
  iframe: HTMLIFrameElement | null = null;
  pickerEnabled = false;

  hoveredElement: PickerElementData | null = null;
  highlightBox: { left: number; top: number; width: number; height: number } | null = null;
  fields: FieldSpec[] = [];
  metrics: MetricSpec[] = [];
  trialResult: string | null = null;

  private iframeDoc: Document | null = null;
  private stateSubscription: Subscription | null = null;

  @ViewChild('previewContainer') previewContainer!: ElementRef<HTMLDivElement>;

  constructor(
    private stateService: JobCreateStateService,
    private selectorGenerator: SelectorGeneratorService,
    private zone: NgZone
  ) {}

  ngOnInit(): void {
    // Subscribe to state changes instead of reading once
    this.stateSubscription = this.stateService.getState().subscribe(state => {
      this.previewHtml = state.previewHtml;
      this.fields = [...state.extractionSpec.fields];
      this.metrics = [...state.extractionSpec.metrics];

      console.log('ElementPickerStep - state updated:', {
        hasPreviewHtml: !!this.previewHtml,
        previewHtmlLength: this.previewHtml?.length || 0,
        fieldsCount: this.fields.length,
        metricsCount: this.metrics.length
      });
    });
  }

  ngOnDestroy(): void {
    this.detachListeners();
    this.stateSubscription?.unsubscribe();
  }

  onFrameReady(iframe: HTMLIFrameElement): void {
    this.iframe = iframe;
    this.iframeDoc = iframe.contentDocument || iframe.contentWindow?.document || null;
  }

  togglePicker(): void {
    if (!this.pickerEnabled) {
      this.detachListeners();
      this.highlightBox = null;
      this.hoveredElement = null;
    }
  }

  private attachListeners(): void {
    // no-op: handled by overlay events
  }

  private detachListeners(): void {
    // no-op: handled by overlay events
  }

  onOverlayMouseMove(event: MouseEvent): void {
    if (!this.pickerEnabled) return;
    const target = this.getElementFromOverlayEvent(event);
    if (!target) {
      this.highlightBox = null;
      this.hoveredElement = null;
      return;
    }

    const selector = this.selectorGenerator.generate(target);
    const value = this.selectorGenerator.extractValue(target, 'text');

    this.hoveredElement = {
      selector,
      value: value.substring(0, 100),
      attribute: 'text',
      elementTag: target.tagName.toLowerCase()
    };

    this.highlightBox = this.getOverlayBox(target);
  }

  onOverlayClick(event: MouseEvent): void {
    if (!this.pickerEnabled) return;
    if (event.button !== 0) return;

    event.preventDefault();
    event.stopPropagation();
    event.stopImmediatePropagation();

    const target = this.getElementFromOverlayEvent(event);
    if (!target) return;

    // Generate selector
    const selector = this.selectorGenerator.generate(target);

    // Determine attribute based on element type
    const attribute = this.getPreferredAttribute(target);

    const value = this.selectorGenerator.extractValue(target, attribute);

    const elementData: PickerElementData = {
      selector,
      value: value.substring(0, 200),
      attribute,
      elementTag: target.tagName.toLowerCase()
    };

    const alreadySelected = this.fields.some(
      existing =>
        existing.extractor.selector === elementData.selector &&
        existing.extractor.attribute === elementData.attribute
    );
    if (alreadySelected) {
      return;
    }

    this.zone.run(() => {
      this.addFieldFromElement(elementData);
    });
  }

  isValid(): boolean {
    return this.fields.length > 0;
  }

  onOverlayWheel(event: WheelEvent): void {
    if (!this.pickerEnabled || !this.iframe?.contentWindow) return;
    event.preventDefault();
    this.iframe.contentWindow.scrollBy({
      top: event.deltaY,
      left: event.deltaX
    });
  }

  private getOverlayBox(target: Element): { left: number; top: number; width: number; height: number } | null {
    if (!this.iframe || !this.previewContainer) return null;

    const rect = target.getBoundingClientRect();
    const iframeRect = this.iframe.getBoundingClientRect();
    const containerRect = this.previewContainer.nativeElement.getBoundingClientRect();

    return {
      left: iframeRect.left - containerRect.left + rect.left,
      top: iframeRect.top - containerRect.top + rect.top,
      width: rect.width,
      height: rect.height
    };
  }

  private getPreferredAttribute(element: Element): string {
    if (element.hasAttribute('href')) {
      return 'href';
    }
    if (element.hasAttribute('src')) {
      return 'src';
    }
    if (element.hasAttribute('alt')) {
      return 'alt';
    }
    if (element.hasAttribute('title')) {
      return 'title';
    }
    return 'text';
  }

  private getElementFromOverlayEvent(event: MouseEvent): Element | null {
    if (!this.iframe || !this.iframeDoc) return null;

    const iframeRect = this.iframe.getBoundingClientRect();
    const x = event.clientX - iframeRect.left;
    const y = event.clientY - iframeRect.top;

    if (x < 0 || y < 0 || x > iframeRect.width || y > iframeRect.height) {
      return null;
    }

    const target = this.iframeDoc.elementFromPoint(x, y);
    if (!target) return null;
    if (target === this.iframeDoc.documentElement || target === this.iframeDoc.body) {
      return null;
    }
    return target;
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

    this.stateService.addField(newField);
  }

  updateField(index: number, field: FieldSpec): void {
    this.stateService.updateField(index, field);
  }

  removeField(index: number): void {
    this.stateService.removeField(index);
  }

  addMetric(): void {
    const newMetric: MetricSpec = {
      name: `metric_${this.metrics.length + 1}`,
      op: 'count',
      input: ''
    };

    this.stateService.addMetric(newMetric);
  }

  updateMetric(index: number, metric: MetricSpec): void {
    // State service lacks update helper, so remove and re-add
    this.stateService.removeMetric(index);
    this.stateService.addMetric(metric);
  }

  removeMetric(index: number): void {
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

  private addFieldFromElement(element: PickerElementData): void {
    const exists = this.fields.some(
      field =>
        field.extractor.selector === element.selector &&
        field.extractor.attribute === element.attribute
    );
    if (exists) return;

    const newField: FieldSpec = {
      name: element.elementTag,
      type: 'string',
      required: false,
      extractor: {
        source: 'html',
        selector_type: 'css',
        selector: element.selector,
        attribute: element.attribute,
        multiple: false
      },
      transforms: []
    };

    this.stateService.addField(newField);
  }
}

