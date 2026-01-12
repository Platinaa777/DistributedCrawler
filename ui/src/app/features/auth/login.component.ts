import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, ActivatedRoute, RouterLink } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule
  ],
  template: `
    <div class="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-950 flex items-center justify-center p-6">
      <div class="max-w-md w-full space-y-6">
        <div class="text-center text-white space-y-2">
          <div class="inline-flex items-center gap-3 px-4 py-2 rounded-full bg-white/10 backdrop-blur-md border border-white/10">
            <mat-icon class="text-amber-300">travel_explore</mat-icon>
            <span class="tracking-wide text-sm uppercase">Distributed Crawler</span>
          </div>
          <h1 class="text-3xl font-bold">Welcome back</h1>
          <p class="text-slate-200">Sign in to orchestrate and monitor your crawling pipeline.</p>
        </div>

        <mat-card class="bg-white/95 backdrop-blur-lg shadow-2xl border border-slate-200/60 p-8">
          <form [formGroup]="form" (ngSubmit)="onSubmit()" class="space-y-6">
            <div class="space-y-2">
              <h2 class="text-xl font-semibold text-slate-900 tracking-tight">Login</h2>
              <p class="text-sm text-slate-600">Use your account credentials to continue.</p>
            </div>

            <mat-form-field appearance="fill" class="w-full">
              <mat-label>Email</mat-label>
              <input matInput type="email" formControlName="email" autocomplete="email" />
              <mat-error *ngIf="email?.invalid && email?.touched">Enter a valid email</mat-error>
            </mat-form-field>

            <mat-form-field appearance="fill" class="w-full">
              <mat-label>Password</mat-label>
              <input matInput [type]="showPassword ? 'text' : 'password'" formControlName="password" autocomplete="current-password" />
              <button mat-icon-button matSuffix type="button" (click)="togglePasswordVisibility()" [attr.aria-label]="showPassword ? 'Hide password' : 'Show password'">
                <mat-icon>{{ showPassword ? 'visibility_off' : 'visibility' }}</mat-icon>
              </button>
              <mat-error *ngIf="password?.invalid && password?.touched">Password is required</mat-error>
            </mat-form-field>

            <div *ngIf="error" class="rounded-lg border border-red-200 bg-red-50 text-red-700 px-3 py-2 text-sm">
              {{ error }}
            </div>

            <button
              mat-raised-button
              color="primary"
              class="w-full h-12 flex items-center justify-center gap-3 text-base font-semibold"
              [disabled]="form.invalid || loading"
              type="submit">
              <mat-progress-spinner
                *ngIf="loading"
                mode="indeterminate"
                diameter="16"
                strokeWidth="3"
                color="primary"
                class="!w-4 !h-4">
              </mat-progress-spinner>
              <span>{{ loading ? 'Signing in...' : 'Sign in' }}</span>
            </button>

            <div class="text-center text-sm text-slate-600">
              Don't have an account?
              <a routerLink="/auth/register" class="text-indigo-600 hover:text-indigo-700 font-semibold">Create one</a>
            </div>
          </form>
        </mat-card>
      </div>
    </div>
  `
})
export class LoginComponent implements OnInit {
  form: FormGroup;
  loading = false;
  error: string | null = null;
  returnUrl = '/jobs';
  showPassword = false;

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private router: Router,
    private route: ActivatedRoute
  ) {
    this.form = this.fb.group({
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required]]
    });
  }

  ngOnInit(): void {
    const fromQuery = this.route.snapshot.queryParamMap.get('returnUrl');
    if (fromQuery) {
      this.returnUrl = fromQuery;
    }
  }

  get email() {
    return this.form.get('email');
  }

  get password() {
    return this.form.get('password');
  }

  togglePasswordVisibility(): void {
    this.showPassword = !this.showPassword;
  }

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.loading = true;
    this.error = null;

    const { email, password } = this.form.value;

    this.authService.login(email, password).subscribe({
      next: () => {
        this.loading = false;
        this.router.navigateByUrl(this.returnUrl);
      },
      error: (err) => {
        this.loading = false;
        this.error = err?.error?.message || 'Unable to sign in. Please check your credentials.';
      }
    });
  }
}
