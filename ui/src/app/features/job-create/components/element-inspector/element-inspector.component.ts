import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { SelectedElementData } from '../../services/job-create-state.service';

@Component({
  selector: 'app-element-inspector',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule
  ],
  template: `
    <mat-card class="inspector-card flex flex-col overflow-hidden">
      <mat-card-header>
        <mat-card-title class="text-lg flex items-center justify-between w-full">
          <span>Element Inspector</span>
          <span class="text-sm text-gray-500 ml-2 flex-shrink-0">{{ selectedElements.length }} selected</span>
        </mat-card-title>
      </mat-card-header>

      <mat-card-content class="overflow-y-auto flex-1 pr-1">
        <div *ngIf="hoveredElement" class="mb-4 p-3 bg-blue-50 rounded border border-blue-200 shadow-sm">
          <p class="text-xs font-semibold text-blue-800 mb-2">HOVER</p>
          <div class="space-y-1">
            <div class="flex items-start">
              <span class="text-xs font-mono text-blue-600 mr-2">Tag:</span>
              <span class="text-xs font-mono">{{ hoveredElement.elementTag }}</span>
            </div>
            <div class="flex items-start">
              <span class="text-xs font-mono text-blue-600 mr-2">Selector:</span>
              <span class="text-xs font-mono break-all">{{ hoveredElement.selector }}</span>
            </div>
            <div *ngIf="hoveredElement.value" class="flex items-start">
              <span class="text-xs font-mono text-blue-600 mr-2">Value:</span>
              <span class="text-xs truncate">{{ hoveredElement.value }}</span>
            </div>
          </div>
        </div>

        <div *ngIf="selectedElements.length === 0 && !hoveredElement" class="text-center py-10 text-gray-500">
          <mat-icon class="text-gray-400 text-4xl mb-2">mouse</mat-icon>
          <p class="text-sm">Hover over elements in the preview to inspect</p>
          <p class="text-xs mt-1">Click to select</p>
        </div>

        <div *ngIf="selectedElements.length > 0" class="space-y-3 max-h-[360px] overflow-y-auto pr-1">
          <div
            *ngFor="let element of selectedElements; let i = index"
            class="p-3 rounded border border-gray-200 bg-gray-50 shadow-xs"
          >
            <div class="flex items-start justify-between mb-2">
              <div class="flex items-center gap-2">
                <mat-chip-set aria-label="Element selection">
                  <mat-chip class="text-xs">{{ element.elementTag }}</mat-chip>
                  <mat-chip class="text-xs" color="primary">{{ element.attribute }}</mat-chip>
                </mat-chip-set>
              </div>
              <button mat-icon-button color="warn" (click)="removeElement(i)">
                <mat-icon>delete</mat-icon>
              </button>
            </div>

            <mat-form-field appearance="outline" class="w-full mb-2">
              <mat-label>Selector (editable)</mat-label>
              <input
                matInput
                [ngModel]="element.selector"
                (ngModelChange)="updateSelector(i, $event)"
                placeholder=".title"
              />
            </mat-form-field>

            <div class="text-xs text-gray-600 truncate">
              <strong>Value:</strong> {{ element.value || '(empty)' }}
            </div>
          </div>
        </div>
      </mat-card-content>

      <mat-card-actions *ngIf="selectedElements.length > 0">
        <button mat-stroked-button color="warn" (click)="clearAll()">
          <mat-icon>clear_all</mat-icon>
          Clear All
        </button>
      </mat-card-actions>
    </mat-card>
  `,
  styles: [`
    :host {
      display: block;
      height: 100%;
    }

    mat-card {
      display: flex;
      flex-direction: column;
    }

    .inspector-card {
      height: 100%;
      min-height: 0;
    }

    mat-card-content {
      flex: 1;
      overflow-y: auto;
    }
  `]
})
export class ElementInspectorComponent {
  @Input() selectedElements: SelectedElementData[] = [];
  @Input() hoveredElement: SelectedElementData | null = null;

  @Output() elementRemoved = new EventEmitter<number>();
  @Output() elementUpdated = new EventEmitter<{ index: number; selector: string }>();
  @Output() clearAllElements = new EventEmitter<void>();

  removeElement(index: number): void {
    this.elementRemoved.emit(index);
  }

  updateSelector(index: number, selector: string): void {
    this.elementUpdated.emit({ index, selector });
  }

  clearAll(): void {
    this.clearAllElements.emit();
  }
}
