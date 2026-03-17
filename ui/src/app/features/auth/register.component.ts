import { CommonModule } from '@angular/common';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, ActivatedRoute, RouterLink } from '@angular/router';
import { CardModule } from 'primeng/card';
import { InputTextModule } from 'primeng/inputtext';
import { PasswordModule } from 'primeng/password';
import { ButtonModule } from 'primeng/button';
import { ProgressSpinnerModule } from 'primeng/progressspinner';
import { FloatLabelModule } from 'primeng/floatlabel';
import { AuthService } from '../../core/services/auth.service';
import { ThemeService } from '../../core/services/theme.service';

@Component({
  selector: 'app-register',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    CardModule,
    InputTextModule,
    PasswordModule,
    ButtonModule,
    ProgressSpinnerModule,
    FloatLabelModule
  ],
  template: `
    <div class="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-950 flex items-center justify-center p-6">
      <div class="max-w-md w-full space-y-6">
        <div class="text-center text-white space-y-2">
          <div class="inline-flex items-center gap-3 px-4 py-2 rounded-full bg-white/10 backdrop-blur-md border border-white/10">
            <i class="pi pi-globe text-emerald-300 text-xl"></i>
            <span class="tracking-wide text-sm uppercase">Distributed Crawler</span>
          </div>
          <h1 class="text-3xl font-bold">Create your account</h1>
          <p class="text-slate-200">Register to start scheduling and monitoring crawl jobs.</p>
        </div>

        <p-card styleClass="bg-white/95 backdrop-blur-lg shadow-2xl border border-slate-200/60">
          <form [formGroup]="form" (ngSubmit)="onSubmit()" class="space-y-6 p-2">
            <div class="space-y-2">
              <h2 class="text-xl font-semibold text-slate-900 tracking-tight">Register</h2>
              <p class="text-sm text-slate-600">Use a strong password (8+ characters).</p>
            </div>

            <div class="space-y-4">
              <p-floatlabel>
                <input
                  pInputText
                  id="email"
                  type="email"
                  formControlName="email"
                  autocomplete="email"
                  class="w-full" />
                <label for="email">Email</label>
              </p-floatlabel>
              <small *ngIf="email?.invalid && email?.touched" class="text-red-500">
                Enter a valid email
              </small>

              <div class="mt-6" style="margin-top: 1.5rem;">
                <p-floatlabel class="block">
                  <p-password
                    id="password"
                    formControlName="password"
                    [feedback]="true"
                    [toggleMask]="true"
                    autocomplete="new-password"
                    styleClass="w-full"
                    inputStyleClass="w-full" />
                  <label for="password">Password</label>
                </p-floatlabel>
              </div>
              <small *ngIf="password?.hasError('required') && password?.touched" class="text-red-500">
                Password is required
              </small>
              <small *ngIf="password?.hasError('minlength') && password?.touched" class="text-red-500">
                Password must be at least 8 characters
              </small>

              <div class="mt-6" style="margin-top: 1.5rem;">
              <p-floatlabel>
                <p-password
                  id="confirmPassword"
                  formControlName="confirmPassword"
                  [feedback]="false"
                  [toggleMask]="true"
                  autocomplete="new-password"
                  styleClass="w-full"
                  inputStyleClass="w-full" />
                <label for="confirmPassword">Confirm password</label>
              </p-floatlabel>
              </div>
              <small *ngIf="confirmPassword?.hasError('required') && confirmPassword?.touched" class="text-red-500">
                Confirm your password
              </small>
              <small *ngIf="form.errors?.['passwordMismatch'] && confirmPassword?.touched" class="text-red-500">
                Passwords do not match
              </small>
            </div>

            <div *ngIf="error" class="rounded-lg border border-red-200 bg-red-50 text-red-700 px-3 py-2 text-sm">
              {{ error }}
            </div>

            <p-button
              type="submit"
              [label]="loading ? 'Creating account...' : 'Create account'"
              [disabled]="form.invalid || loading"
              [loading]="loading"
              styleClass="w-full mt-2" />

            <div class="text-center text-sm text-slate-600">
              Already have an account?
              <a routerLink="/auth/login" class="text-indigo-600 hover:text-indigo-700 font-semibold">Sign in</a>
            </div>
          </form>
        </p-card>
      </div>
    </div>
  `
})
export class RegisterComponent implements OnInit, OnDestroy {
  form: FormGroup;
  loading = false;
  error: string | null = null;
  returnUrl = '/jobs';
  private releaseDarkMode: (() => void) | null = null;

  constructor(
    private fb: FormBuilder,
    private authService: AuthService,
    private router: Router,
    private route: ActivatedRoute,
    private themeService: ThemeService
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
    this.releaseDarkMode = this.themeService.suppressDarkMode();
    const fromQuery = this.route.snapshot.queryParamMap.get('returnUrl');
    if (fromQuery) {
      this.returnUrl = fromQuery;
    }
  }

  ngOnDestroy(): void {
    this.releaseDarkMode?.();
    this.releaseDarkMode = null;
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
