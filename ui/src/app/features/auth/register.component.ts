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
  selector: 'app-register',
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
            <mat-icon class="text-emerald-300">travel_explore</mat-icon>
            <span class="tracking-wide text-sm uppercase">Distributed Crawler</span>
          </div>
          <h1 class="text-3xl font-bold">Create your account</h1>
          <p class="text-slate-200">Register to start scheduling and monitoring crawl jobs.</p>
        </div>

        <mat-card class="bg-white/95 backdrop-blur-lg shadow-2xl border border-slate-200/60 p-8">
          <form [formGroup]="form" (ngSubmit)="onSubmit()" class="space-y-6">
            <div class="space-y-2">
              <h2 class="text-xl font-semibold text-slate-900 tracking-tight">Register</h2>
              <p class="text-sm text-slate-600">Use a strong password (8+ characters).</p>
            </div>

            <mat-form-field appearance="fill" class="w-full">
              <mat-label>Email</mat-label>
              <input matInput type="email" formControlName="email" autocomplete="email" />
              <mat-error *ngIf="email?.invalid && email?.touched">Enter a valid email</mat-error>
            </mat-form-field>

            <mat-form-field appearance="fill" class="w-full">
              <mat-label>Password</mat-label>
              <input matInput [type]="showPassword ? 'text' : 'password'" formControlName="password" autocomplete="new-password" />
              <button mat-icon-button matSuffix type="button" (click)="togglePasswordVisibility()" [attr.aria-label]="showPassword ? 'Hide password' : 'Show password'">
                <mat-icon>{{ showPassword ? 'visibility_off' : 'visibility' }}</mat-icon>
              </button>
              <mat-error *ngIf="password?.hasError('required') && password?.touched">Password is required</mat-error>
              <mat-error *ngIf="password?.hasError('minlength') && password?.touched">Password must be at least 8 characters</mat-error>
            </mat-form-field>

            <mat-form-field appearance="fill" class="w-full">
              <mat-label>Confirm password</mat-label>
              <input matInput [type]="showPassword ? 'text' : 'password'" formControlName="confirmPassword" autocomplete="new-password" />
              <mat-error *ngIf="confirmPassword?.hasError('required') && confirmPassword?.touched">Confirm your password</mat-error>
              <mat-error *ngIf="form.errors?.['passwordMismatch'] && confirmPassword?.touched">Passwords do not match</mat-error>
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
              <span>{{ loading ? 'Creating account...' : 'Create account' }}</span>
            </button>

            <div class="text-center text-sm text-slate-600">
              Already have an account?
              <a routerLink="/auth/login" class="text-indigo-600 hover:text-indigo-700 font-semibold">Sign in</a>
            </div>
          </form>
        </mat-card>
      </div>
    </div>
  `
})
export class RegisterComponent implements OnInit {
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
    this.form = this.fb.group(
      {
        email: ['', [Validators.required, Validators.email]],
        password: ['', [Validators.required, Validators.minLength(8)]],
        confirmPassword: ['', [Validators.required]]
      },
      { validators: this.passwordsMatchValidator }
    );
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

  get confirmPassword() {
    return this.form.get('confirmPassword');
  }

  togglePasswordVisibility(): void {
    this.showPassword = !this.showPassword;
  }

  private passwordsMatchValidator(group: FormGroup) {
    const password = group.get('password')?.value;
    const confirm = group.get('confirmPassword')?.value;
    return password === confirm ? null : { passwordMismatch: true };
  }

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.loading = true;
    this.error = null;

    const { email, password } = this.form.value;

    this.authService.register(email, password).subscribe({
      next: () => {
        this.loading = false;
        this.router.navigateByUrl(this.returnUrl);
      },
      error: (err) => {
        this.loading = false;
        this.error = err?.error?.message || 'Unable to create your account right now.';
      }
    });
  }
}
