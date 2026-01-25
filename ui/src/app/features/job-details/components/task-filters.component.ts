import { Component, EventEmitter, OnInit, OnDestroy, Output, inject, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { DatePickerModule } from 'primeng/datepicker';
import { ButtonModule } from 'primeng/button';
import { IconFieldModule } from 'primeng/iconfield';
import { InputIconModule } from 'primeng/inputicon';
import { InputNumberModule } from 'primeng/inputnumber';
import { Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged, takeUntil } from 'rxjs/operators';
import { TASK_STATUSES, TaskStatus } from '../../../core/models/crawl-task.model';
import { TaskListFilter } from '../../../core/services/api/crawler-api.service';

@Component({
  selector: 'app-task-filters',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    InputTextModule,
    SelectModule,
    DatePickerModule,
    ButtonModule,
    IconFieldModule,
    InputIconModule,
    InputNumberModule
  ],
  template: `
    <div class="filter-container flex flex-wrap gap-4 items-end p-4 bg-gray-50 dark:bg-gray-800 rounded-lg mb-4 border border-gray-200 dark:border-gray-700">
      <!-- URL Search -->
      <div class="flex-1 min-w-48">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Search by URL</label>
        <p-iconfield>
          <p-inputicon styleClass="pi pi-search" />
          <input
            pInputText
            [formControl]="filterForm.controls.url"
            placeholder="Enter URL..."
            class="w-full" />
        </p-iconfield>
      </div>

      <!-- Status Filter -->
      <div class="w-40">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Status</label>
        <p-select
          [options]="statusOptions"
          [formControl]="filterForm.controls.status"
          placeholder="All"
          styleClass="w-full" />
      </div>

      <!-- Depth Range -->
      <div class="w-28">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Min Depth</label>
        <p-inputnumber
          [formControl]="filterForm.controls.minDepth"
          [min]="0"
          [max]="maxDepthValue"
          placeholder="0"
          styleClass="w-full"
          inputStyleClass="w-full" />
      </div>

      <div class="w-28">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Depth</label>
        <p-inputnumber
          [formControl]="filterForm.controls.maxDepth"
          [min]="0"
          [max]="maxDepthValue"
          placeholder="Any"
          styleClass="w-full"
          inputStyleClass="w-full" />
      </div>

      <!-- Date Range -->
      <div class="w-40">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">From date</label>
        <p-datepicker
          [formControl]="filterForm.controls.enqueuedFrom"
          [showIcon]="true"
          dateFormat="yy-mm-dd"
          placeholder="Select date"
          styleClass="w-full" />
      </div>

      <div class="w-40">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">To date</label>
        <p-datepicker
          [formControl]="filterForm.controls.enqueuedTo"
          [showIcon]="true"
          dateFormat="yy-mm-dd"
          placeholder="Select date"
          styleClass="w-full" />
      </div>

      <!-- Clear Filters -->
      <p-button
        [outlined]="true"
        severity="secondary"
        (onClick)="clearFilters()">
        <i class="pi pi-times mr-2"></i>
        Clear
      </p-button>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class TaskFiltersComponent implements OnInit, OnDestroy {
  @Input() maxDepthValue = 10;
  @Output() filterChange = new EventEmitter<TaskListFilter>();

  private destroy$ = new Subject<void>();
  private fb = inject(FormBuilder);

  statuses: TaskStatus[] = TASK_STATUSES;
  statusOptions = [
    { label: 'All', value: null },
    ...TASK_STATUSES.map(status => ({ label: status, value: status }))
  ];

  filterForm = this.fb.group({
    url: [''],
    status: [null as TaskStatus | null],
    minDepth: [null as number | null],
    maxDepth: [null as number | null],
    enqueuedFrom: [null as Date | null],
    enqueuedTo: [null as Date | null]
  });

  ngOnInit(): void {
    // Emit filter changes with debounce
    this.filterForm.valueChanges.pipe(
      takeUntil(this.destroy$),
      debounceTime(300),
      distinctUntilChanged((prev, curr) => JSON.stringify(prev) === JSON.stringify(curr))
    ).subscribe(value => {
      this.emitFilters(value);
    });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  private emitFilters(value: typeof this.filterForm.value): void {
    const filter: TaskListFilter = {};

    if (value.url?.trim()) {
      filter.url = value.url.trim();
    }
    if (value.status) {
      filter.status = value.status;
    }
    if (value.minDepth !== null && value.minDepth !== undefined) {
      filter.min_depth = value.minDepth;
    }
    if (value.maxDepth !== null && value.maxDepth !== undefined) {
      filter.max_depth = value.maxDepth;
    }
    if (value.enqueuedFrom) {
      filter.enqueued_from = value.enqueuedFrom.toISOString();
    }
    if (value.enqueuedTo) {
      const endOfDay = new Date(value.enqueuedTo);
      endOfDay.setHours(23, 59, 59, 999);
      filter.enqueued_to = endOfDay.toISOString();
    }

    this.filterChange.emit(filter);
  }

  clearFilters(): void {
    this.filterForm.reset();
  }
}
