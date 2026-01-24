import { Component, OnDestroy, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { SelectModule } from 'primeng/select';
import { CardModule } from 'primeng/card';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { Router } from '@angular/router';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { UserApiService } from '../../core/services/api/user-api.service';
import { AuthService } from '../../core/services/auth.service';
import { User, UserRole } from '../../core/models';

@Component({
  selector: 'app-users-list',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    TableModule,
    ButtonModule,
    SelectModule,
    CardModule,
    ProgressSpinnerModule
  ],
  template: `
    <div class="container mx-auto p-6">
      <div class="flex justify-between items-center mb-6">
        <div>
          <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Users</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">Administrators can manage user roles (except their own).</p>
        </div>
        <p-button
          [outlined]="true"
          severity="secondary"
          (onClick)="goBack()">
          <i class="pi pi-arrow-left mr-2"></i>
          Back to Jobs
        </p-button>
      </div>

      <p-card *ngIf="loading" styleClass="text-center p-8">
        <p-progressSpinner />
        <p class="mt-4">Loading users...</p>
      </p-card>

      <p-card *ngIf="error && !loading" styleClass="bg-red-50 p-4">
        <p class="text-red-700">{{ error }}</p>
      </p-card>

      <p-card *ngIf="!loading">
        <p-table [value]="users" [tableStyle]="{'min-width': '50rem'}">
          <ng-template pTemplate="header">
            <tr>
              <th>Email</th>
              <th>Role</th>
              <th>Created</th>
            </tr>
          </ng-template>
          <ng-template pTemplate="body" let-user>
            <tr>
              <td class="text-gray-900 dark:text-white">
                {{ user.email }}
                <span *ngIf="isCurrentUser(user.id)" class="text-gray-500 dark:text-gray-400 text-sm ml-2">(You)</span>
              </td>
              <td>
                <div class="flex items-center gap-2">
                  <p-select
                    [options]="getRoleOptions(user)"
                    [(ngModel)]="user.role"
                    (onChange)="updateRole(user, $event.value)"
                    [disabled]="isUpdating(user.id) || user.role === 'ADMINISTRATOR' || isCurrentUser(user.id)"
                    appendTo="body"
                    styleClass="w-44" />
                  <p-progressSpinner *ngIf="isUpdating(user.id)" [style]="{width: '18px', height: '18px'}" />
                </div>
              </td>
              <td>{{ user.created_at | date:'short' }}</td>
            </tr>
          </ng-template>
          <ng-template pTemplate="emptymessage">
            <tr>
              <td colspan="3" class="text-center p-8 text-gray-500 dark:text-gray-400">
                <i class="pi pi-user text-6xl block mb-4"></i>
                <p>No users found.</p>
              </td>
            </tr>
          </ng-template>
        </p-table>
      </p-card>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class UsersListComponent implements OnInit, OnDestroy {
  private destroy$ = new Subject<void>();

  users: User[] = [];
  loading = false;
  error: string | null = null;
  private updating = new Set<string>();
  readonly assignableRoles: UserRole[] = ['READ', 'READ_WRITE'];

  constructor(
    private userApi: UserApiService,
    private authService: AuthService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.loadUsers();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  getRoleOptions(user: User): { label: string; value: string }[] {
    const options = this.assignableRoles.map(role => ({ label: role, value: role }));
    if (user.role === 'ADMINISTRATOR') {
      options.push({ label: 'ADMINISTRATOR', value: 'ADMINISTRATOR' });
    }
    return options;
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

  isCurrentUser(id: string): boolean {
    return this.authService.userId === id;
  }

  goBack(): void {
    this.router.navigate(['/jobs']);
  }
}
