import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatSelectModule } from '@angular/material/select';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatCardModule } from '@angular/material/card';
import { Router } from '@angular/router';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { UserApiService } from '../../core/services/api/user-api.service';
import { User, UserRole } from '../../core/models';

@Component({
  selector: 'app-users-list',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatSelectModule,
    MatFormFieldModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatCardModule
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-3xl font-bold">Users</h1>
          <p class="text-sm text-gray-500 mt-1">Only administrators can grant READ_WRITE access.</p>
        </div>
        <button mat-stroked-button color="primary" (click)="goBack()">
          <mat-icon>arrow_back</mat-icon>
          Back to Jobs
        </button>
      </div>

      <mat-card *ngIf="loading" class="text-center p-8">
        <mat-spinner class="mx-auto"></mat-spinner>
        <p class="mt-4">Loading users...</p>
      </mat-card>

      <mat-card *ngIf="error && !loading" class="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </mat-card>

      <mat-card *ngIf="!loading">
        <table mat-table [dataSource]="users" class="w-full">
          <ng-container matColumnDef="email">
            <th mat-header-cell *matHeaderCellDef>Email</th>
            <td mat-cell *matCellDef="let user">{{ user.email }}</td>
          </ng-container>

          <ng-container matColumnDef="role">
            <th mat-header-cell *matHeaderCellDef>Role</th>
            <td mat-cell *matCellDef="let user">
              <mat-form-field appearance="outline" class="role-select">
                <mat-select
                  [value]="user.role"
                  (selectionChange)="updateRole(user, $event.value)"
                  [disabled]="isUpdating(user.id) || user.role === 'ADMINISTRATOR'">
                  <mat-option *ngFor="let role of assignableRoles" [value]="role">
                    {{ role }}
                  </mat-option>
                  <mat-option *ngIf="user.role === 'ADMINISTRATOR'" [value]="user.role">
                    {{ user.role }}
                  </mat-option>
                </mat-select>
              </mat-form-field>
              <mat-spinner *ngIf="isUpdating(user.id)" diameter="18" class="inline-spinner"></mat-spinner>
            </td>
          </ng-container>

          <ng-container matColumnDef="created">
            <th mat-header-cell *matHeaderCellDef>Created</th>
            <td mat-cell *matCellDef="let user">{{ user.created_at | date:'short' }}</td>
          </ng-container>

          <tr mat-header-row *matHeaderRowDef="displayedColumns"></tr>
          <tr mat-row *matRowDef="let row; columns: displayedColumns;"></tr>
        </table>

        <div *ngIf="users.length === 0" class="text-center p-8 text-gray-500">
          <mat-icon class="text-6xl">person_outline</mat-icon>
          <p class="mt-4">No users found.</p>
        </div>
      </mat-card>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }

    table {
      width: 100%;
    }

    .role-select {
      width: 180px;
    }

    .inline-spinner {
      display: inline-block;
      vertical-align: middle;
      margin-left: 8px;
    }
  `]
})
export class UsersListComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  users: User[] = [];
  displayedColumns = ['email', 'role', 'created'];
  loading = false;
  error: string | null = null;
  private updating = new Set<string>();
  readonly assignableRoles: UserRole[] = ['READ', 'READ_WRITE'];

  constructor(
    private userApi: UserApiService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.loadUsers();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadUsers(): void {
    this.loading = true;
    this.error = null;

    this.userApi.listUsers()
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (response) => {
          this.users = response.users || [];
          this.loading = false;
        },
        error: (err) => {
          this.error = `Failed to load users: ${err.message}`;
          this.loading = false;
        }
      });
  }

  updateRole(user: User, role: UserRole): void {
    if (user.role === role || this.isUpdating(user.id)) {
      return;
    }

    this.updating.add(user.id);
    this.userApi.updateUserRole(user.id, role)
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: () => {
          user.role = role;
          this.updating.delete(user.id);
        },
        error: (err) => {
          this.error = `Failed to update role: ${err.message}`;
          this.updating.delete(user.id);
        }
      });
  }

  isUpdating(id: string): boolean {
    return this.updating.has(id);
  }

  goBack(): void {
    this.router.navigate(['/jobs']);
  }
}
