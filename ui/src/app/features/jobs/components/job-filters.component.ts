import { Component, EventEmitter, OnInit, OnDestroy, Output, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { DatePickerModule } from 'primeng/datepicker';
import { ButtonModule } from 'primeng/button';
import { IconFieldModule } from 'primeng/iconfield';
import { InputIconModule } from 'primeng/inputicon';
import { Subject } from 'rxjs';
import { debounceTime, distinctUntilChanged, takeUntil } from 'rxjs/operators';
import { JOB_STATUSES, JobStatus } from '../../../core/models';
import { JobListFilter } from '../../../core/services/api/crawler-api.service';

@Component({
  selector: 'app-job-filters',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    InputTextModule,
    SelectModule,
    DatePickerModule,
    ButtonModule,
    IconFieldModule,
    InputIconModule
  ],
  template: `
    <div class="filter-container flex flex-wrap gap-4 items-end p-4 bg-gray-50 rounded-lg mb-4">
      <!-- Name Search -->
      <div class="flex-1 min-w-48">
        <label class="block text-sm font-medium text-gray-700 mb-1">Search by name</label>
        <p-iconfield>
          <p-inputicon styleClass="pi pi-search" />
          <input
            pInputText
            [formControl]="filterForm.controls.name"
            placeholder="Enter job name..."
            class="w-full" />
        </p-iconfield>
      </div>

      <!-- Status Filter -->
      <div class="w-40">
        <label class="block text-sm font-medium text-gray-700 mb-1">Status</label>
        <p-select
          [options]="statusOptions"
          [formControl]="filterForm.controls.status"
          placeholder="All"
          styleClass="w-full" />
      </div>

      <!-- Date Range -->
      <div class="w-40">
        <label class="block text-sm font-medium text-gray-700 mb-1">From date</label>
        <p-datepicker
          [formControl]="filterForm.controls.createdFrom"
          [showIcon]="true"
          dateFormat="yy-mm-dd"
          placeholder="Select date"
          styleClass="w-full" />
      </div>

      <div class="w-40">
        <label class="block text-sm font-medium text-gray-700 mb-1">To date</label>
        <p-datepicker
          [formControl]="filterForm.controls.createdTo"
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

    .filter-container {
      background: #f8fafc;
      border: 1px solid #e2e8f0;
    }
  `]
})
export class JobFiltersComponent implements OnInit, OnDestroy {
  @Output() filterChange = new EventEmitter<JobListFilter>();

  private destroy$ = new Subject<void>();
  private fb = inject(FormBuilder);

  statuses: JobStatus[] = JOB_STATUSES;
  statusOptions = [
    { label: 'All', value: null },
    ...JOB_STATUSES.map(status => ({ label: status, value: status }))
  ];

  filterForm = this.fb.group({
    name: [''],
    status: [null as JobStatus | null],
    createdFrom: [null as Date | null],
    createdTo: [null as Date | null]
  });

  ngOnInit(): void {
    // Emit filter changes with debounce for name input
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
    const filter: JobListFilter = {};

    if (value.name?.trim()) {
      filter.name = value.name.trim();
    }
    if (value.status) {
      filter.status = value.status;
    }
    if (value.createdFrom) {
      filter.created_from = value.createdFrom.toISOString();
    }
    if (value.createdTo) {
      // Set to end of day
      const endOfDay = new Date(value.createdTo);
      endOfDay.setHours(23, 59, 59, 999);
      filter.created_to = endOfDay.toISOString();
    }

    this.filterChange.emit(filter);
  }

  clearFilters(): void {
    this.filterForm.reset();
  }
}
