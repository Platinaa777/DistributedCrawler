import { Component, EventEmitter, OnInit, OnDestroy, Output, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatNativeDateModule } from '@angular/material/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
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
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatDatepickerModule,
    MatNativeDateModule,
    MatButtonModule,
    MatIconModule
  ],
  template: `
    <div class="filter-container flex flex-wrap gap-4 items-end p-4 bg-gray-50 rounded-lg mb-4">
      <!-- Name Search -->
      <mat-form-field class="flex-1 min-w-48">
        <mat-label>Search by name</mat-label>
        <input matInput [formControl]="filterForm.controls.name" placeholder="Enter job name...">
        <mat-icon matSuffix>search</mat-icon>
      </mat-form-field>

      <!-- Status Filter -->
      <mat-form-field class="w-40">
        <mat-label>Status</mat-label>
        <mat-select [formControl]="filterForm.controls.status">
          <mat-option [value]="null">All</mat-option>
          <mat-option *ngFor="let status of statuses" [value]="status">
            {{ status }}
          </mat-option>
        </mat-select>
      </mat-form-field>

      <!-- Date Range -->
      <mat-form-field class="w-40">
        <mat-label>From date</mat-label>
        <input matInput [matDatepicker]="fromPicker" [formControl]="filterForm.controls.createdFrom">
        <mat-datepicker-toggle matSuffix [for]="fromPicker"></mat-datepicker-toggle>
        <mat-datepicker #fromPicker></mat-datepicker>
      </mat-form-field>

      <mat-form-field class="w-40">
        <mat-label>To date</mat-label>
        <input matInput [matDatepicker]="toPicker" [formControl]="filterForm.controls.createdTo">
        <mat-datepicker-toggle matSuffix [for]="toPicker"></mat-datepicker-toggle>
        <mat-datepicker #toPicker></mat-datepicker>
      </mat-form-field>

      <!-- Clear Filters -->
      <button mat-stroked-button (click)="clearFilters()" class="h-14">
        <mat-icon>clear</mat-icon>
        Clear
      </button>
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

    mat-form-field {
      margin-bottom: 0;
    }

    ::ng-deep .mat-mdc-form-field-subscript-wrapper {
      display: none;
    }
  `]
})
export class JobFiltersComponent implements OnInit, OnDestroy {
  @Output() filterChange = new EventEmitter<JobListFilter>();

  private destroy$ = new Subject<void>();
  private fb = inject(FormBuilder);

  statuses: JobStatus[] = JOB_STATUSES;

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
