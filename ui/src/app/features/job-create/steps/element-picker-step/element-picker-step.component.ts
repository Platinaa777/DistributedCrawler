import { Component, OnInit, OnDestroy, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { PreviewIframeComponent } from '../../components/preview-iframe/preview-iframe.component';
import { ElementInspectorComponent } from '../../components/element-inspector/element-inspector.component';
import { JobCreateStateService, SelectedElementData } from '../../services/job-create-state.service';
import { SelectorGeneratorService } from '../../../../core/services/selector-generator.service';
import { Subscription } from 'rxjs';

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
    ElementInspectorComponent
  ],
  template: `
    <div class="space-y-4">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Step 2: Pick Elements</mat-card-title>
          <mat-card-subtitle>
            Hover and click on elements in the preview to extract data
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
              {{ pickerEnabled ? 'Hover over elements and click to select' : 'Enable picker to select elements' }}
            </div>
          </div>
        </mat-card-content>
      </mat-card>

      <div class="grid grid-cols-3 gap-4">
        <div class="col-span-2">
          <mat-card>
            <mat-card-header>
              <mat-card-title class="text-base">Page Preview</mat-card-title>
            </mat-card-header>
            <mat-card-content class="relative">
              <div class="relative h-[600px]">
                <app-preview-iframe
                  [html]="previewHtml"
                  (frameReady)="onFrameReady($event)"
                ></app-preview-iframe>

                <!-- Highlight overlay -->
                <div
                  *ngIf="pickerEnabled && highlightBox"
                  class="absolute pointer-events-none border-2 border-blue-500 bg-blue-500 bg-opacity-10"
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
        </div>

        <div class="col-span-1">
          <app-element-inspector
            [selectedElements]="selectedElements"
            [hoveredElement]="hoveredElement"
            (elementRemoved)="removeElement($event)"
            (clearAllElements)="clearAllElements()"
          ></app-element-inspector>
        </div>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class ElementPickerStepComponent implements OnInit, OnDestroy {
  previewHtml: string | null = null;
  iframe: HTMLIFrameElement | null = null;
  pickerEnabled = false;

  selectedElements: SelectedElementData[] = [];
  hoveredElement: SelectedElementData | null = null;
  highlightBox: { left: number; top: number; width: number; height: number } | null = null;

  private iframeDoc: Document | null = null;
  private mouseMoveListener: ((e: MouseEvent) => void) | null = null;
  private clickListener: ((e: MouseEvent) => void) | null = null;
  private stateSubscription: Subscription | null = null;

  constructor(
    private stateService: JobCreateStateService,
    private selectorGenerator: SelectorGeneratorService
  ) {}

  ngOnInit(): void {
    // Subscribe to state changes instead of reading once
    this.stateSubscription = this.stateService.getState().subscribe(state => {
      this.previewHtml = state.previewHtml;
      this.selectedElements = [...state.selectedElements];

      console.log('ElementPickerStep - state updated:', {
        hasPreviewHtml: !!this.previewHtml,
        previewHtmlLength: this.previewHtml?.length || 0,
        selectedElementsCount: this.selectedElements.length
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

    if (this.pickerEnabled) {
      this.attachListeners();
    }
  }

  togglePicker(): void {
    if (this.pickerEnabled) {
      this.attachListeners();
    } else {
      this.detachListeners();
      this.highlightBox = null;
      this.hoveredElement = null;
    }
  }

  private attachListeners(): void {
    if (!this.iframeDoc) return;

    this.mouseMoveListener = this.onIframeMouseMove.bind(this);
    this.clickListener = this.onIframeClick.bind(this);

    this.iframeDoc.addEventListener('mousemove', this.mouseMoveListener);
    this.iframeDoc.addEventListener('click', this.clickListener);
  }

  private detachListeners(): void {
    if (!this.iframeDoc) return;

    if (this.mouseMoveListener) {
      this.iframeDoc.removeEventListener('mousemove', this.mouseMoveListener);
    }
    if (this.clickListener) {
      this.iframeDoc.removeEventListener('click', this.clickListener);
    }
  }

  private onIframeMouseMove(event: MouseEvent): void {
    if (!this.pickerEnabled || !this.iframe) return;

    const target = event.target as Element;
    if (!target || target.nodeName === '#document') return;

    // Generate selector
    const selector = this.selectorGenerator.generate(target);
    const value = this.selectorGenerator.extractValue(target, 'text');

    // Update hovered element
    this.hoveredElement = {
      selector,
      value: value.substring(0, 100), // Limit preview
      attribute: 'text',
      elementTag: target.tagName.toLowerCase()
    };

    // Calculate highlight box position relative to iframe
    const rect = target.getBoundingClientRect();
    const iframeRect = this.iframe.getBoundingClientRect();

    this.highlightBox = {
      left: rect.left,
      top: rect.top,
      width: rect.width,
      height: rect.height
    };
  }

  private onIframeClick(event: MouseEvent): void {
    if (!this.pickerEnabled) return;

    event.preventDefault();
    event.stopPropagation();

    const target = event.target as Element;
    if (!target) return;

    // Generate selector
    const selector = this.selectorGenerator.generate(target);

    // Determine attribute based on element type
    let attribute = 'text';
    if (target.hasAttribute('href')) {
      attribute = 'href';
    } else if (target.hasAttribute('src')) {
      attribute = 'src';
    }

    const value = this.selectorGenerator.extractValue(target, attribute);

    const elementData: SelectedElementData = {
      selector,
      value: value.substring(0, 200),
      attribute,
      elementTag: target.tagName.toLowerCase()
    };

    // Add to state
    this.stateService.addSelectedElement(elementData);
    this.selectedElements.push(elementData);
  }

  removeElement(index: number): void {
    this.stateService.removeSelectedElement(index);
    this.selectedElements.splice(index, 1);
  }

  clearAllElements(): void {
    this.stateService.clearSelectedElements();
    this.selectedElements = [];
  }

  isValid(): boolean {
    return this.selectedElements.length > 0;
  }
}
