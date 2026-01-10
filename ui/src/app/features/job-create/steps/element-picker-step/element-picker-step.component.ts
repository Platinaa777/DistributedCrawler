import { Component, OnInit, OnDestroy, ViewChild, ElementRef, NgZone } from '@angular/core';
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
              <div #previewContainer class="relative h-[600px]">
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

                <!-- Highlight overlay -->
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

    const elementData: SelectedElementData = {
      selector,
      value: value.substring(0, 200),
      attribute,
      elementTag: target.tagName.toLowerCase()
    };

    const alreadySelected = this.selectedElements.some(
      existing => existing.selector === elementData.selector && existing.attribute === elementData.attribute
    );
    if (alreadySelected) {
      return;
    }

    this.zone.run(() => {
      // Add to state
      this.stateService.addSelectedElement(elementData);
    });
  }

  removeElement(index: number): void {
    this.stateService.removeSelectedElement(index);
  }

  clearAllElements(): void {
    this.stateService.clearSelectedElements();
  }

  isValid(): boolean {
    return this.selectedElements.length > 0;
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
}
