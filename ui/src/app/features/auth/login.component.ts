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
  selector: 'app-login',
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
            <i class="pi pi-globe text-amber-300 text-xl"></i>
            <span class="tracking-wide text-sm uppercase">Distributed Crawler</span>
          </div>
          <h1 class="text-3xl font-bold">Welcome back</h1>
          <p class="text-slate-200">Sign in to orchestrate and monitor your crawling pipeline.</p>
        </div>

        <p-card styleClass="bg-white/95 backdrop-blur-lg shadow-2xl border border-slate-200/60">
          <form [formGroup]="form" (ngSubmit)="onSubmit()" class="space-y-6 p-2">
            <div class="space-y-2">
              <h2 class="text-xl font-semibold text-slate-900 tracking-tight">Login</h2>
              <p class="text-sm text-slate-600">Use your account credentials to continue.</p>
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
                    [feedback]="false"
                    [toggleMask]="true"
                    autocomplete="current-password"
                    styleClass="w-full"
                    inputStyleClass="w-full" />
                  <label for="password">Password</label>
                </p-floatlabel>
              </div>
              <small *ngIf="password?.invalid && password?.touched" class="text-red-500">
                Password is required
              </small>
            </div>

            <div *ngIf="error" class="rounded-lg border border-red-200 bg-red-50 text-red-700 px-3 py-2 text-sm">
              {{ error }}
            </div>

            <p-button
              type="submit"
              [label]="loading ? 'Signing in...' : 'Sign in'"
              [disabled]="form.invalid || loading"
              [loading]="loading"
              styleClass="w-full mt-2" />

            <div class="text-center text-sm text-slate-600">
              Don't have an account?
              <a routerLink="/auth/register" class="text-indigo-600 hover:text-indigo-700 font-semibold">Create one</a>
            </div>
          </form>
        </p-card>
      </div>
    </div>
  `
})
export class LoginComponent implements OnInit, OnDestroy {
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
    this.form = this.fb.group({
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required]]
    });
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
