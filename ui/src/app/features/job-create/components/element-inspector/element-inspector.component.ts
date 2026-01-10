import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatListModule } from '@angular/material/list';
import { SelectedElementData } from '../../services/job-create-state.service';

@Component({
  selector: 'app-element-inspector',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    MatListModule
  ],
  template: `
    <mat-card class="h-full">
      <mat-card-header>
        <mat-card-title class="text-lg">Element Inspector</mat-card-title>
        <mat-card-subtitle>
          {{ selectedElements.length }} element(s) selected
        </mat-card-subtitle>
      </mat-card-header>

      <mat-card-content class="overflow-y-auto max-h-96">
        <div *ngIf="hoveredElement" class="mb-4 p-3 bg-blue-50 rounded border border-blue-200">
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

        <div *ngIf="selectedElements.length === 0 && !hoveredElement" class="text-center py-8">
          <mat-icon class="text-gray-400 text-4xl mb-2">mouse</mat-icon>
          <p class="text-gray-500 text-sm">Hover over elements in the preview to inspect</p>
          <p class="text-gray-500 text-xs mt-1">Click to select</p>
        </div>

        <mat-list *ngIf="selectedElements.length > 0">
          <mat-list-item *ngFor="let element of selectedElements; let i = index" class="border-b border-gray-200">
            <div class="flex items-start justify-between w-full py-2">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <mat-chip class="text-xs">{{ element.elementTag }}</mat-chip>
                  <mat-chip class="text-xs" color="primary">{{ element.attribute }}</mat-chip>
                </div>
                <p class="text-xs font-mono text-gray-600 break-all mb-1">
                  {{ element.selector }}
                </p>
                <p class="text-xs text-gray-500 truncate">
                  <strong>Value:</strong> {{ element.value || '(empty)' }}
                </p>
              </div>
              <button
                mat-icon-button
                color="warn"
                (click)="removeElement(i)"
                class="ml-2"
              >
                <mat-icon>delete</mat-icon>
              </button>
            </div>
          </mat-list-item>
        </mat-list>
      </mat-card-content>

      <mat-card-actions *ngIf="selectedElements.length > 0">
        <button mat-button color="warn" (click)="clearAll()">
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
  @Output() clearAllElements = new EventEmitter<void>();

  removeElement(index: number): void {
    this.elementRemoved.emit(index);
  }

  clearAll(): void {
    this.clearAllElements.emit();
  }
}
