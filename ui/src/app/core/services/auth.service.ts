import { Injectable } from '@angular/core';
import { Router, UrlTree } from '@angular/router';
import { Observable, of, throwError } from 'rxjs';
import { catchError, finalize, map, shareReplay } from 'rxjs/operators';
import { AuthApiService } from './api/auth-api.service';
import { AuthResponse, AuthTokens, UserRole } from '../models';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly storageKey = 'crawler_auth_tokens';
  private tokens?: AuthTokens;
  private refreshInProgress$?: Observable<AuthTokens>;

  constructor(
    private authApi: AuthApiService,
    private router: Router
  ) {
    this.tokens = this.loadTokens();
  }

  get accessToken(): string | undefined {
    return this.tokens?.accessToken;
  }

  get refreshToken(): string | undefined {
    return this.tokens?.refreshToken;
  }

  get email(): string | undefined {
    return this.tokens?.email;
  }

  get role(): UserRole | undefined {
    const token = this.tokens?.accessToken;
    if (!token) {
      return undefined;
    }
    return this.extractRoleFromToken(token);
  }

  get userId(): string | undefined {
    const token = this.tokens?.accessToken;
    if (!token) {
      return undefined;
    }
    return this.extractUserIdFromToken(token);
  }

  hasValidAccessToken(): boolean {
    if (!this.tokens) {
      return false;
    }

    const buffer = 5_000; // refresh slightly before expiry
    return this.tokens.expiresAt - buffer > Date.now();
  }

  login(email: string, password: string): Observable<AuthTokens> {
    return this.authApi.login(email, password).pipe(
      map((response) => this.persistTokens(response, email))
    );
  }

  register(email: string, password: string): Observable<AuthTokens> {
    return this.authApi.register(email, password).pipe(
      map((response) => this.persistTokens(response, email))
    );
  }

  refreshTokens(): Observable<AuthTokens> {
    if (!this.refreshToken) {
      return throwError(() => new Error('No refresh token available'));
    }

    if (!this.refreshInProgress$) {
      this.refreshInProgress$ = this.authApi.refresh(this.refreshToken).pipe(
        map((response) => this.persistTokens(response)),
        shareReplay(1),
        finalize(() => {
          this.refreshInProgress$ = undefined;
        })
      );
    }

    return this.refreshInProgress$;
  }

  logout(redirectToLogin = true): Observable<void> {
    const refreshToken = this.refreshToken;
    this.clearTokens();

    if (redirectToLogin) {
      this.router.navigate(['/auth/login']);
    }

    return refreshToken
      ? this.authApi.logout(refreshToken).pipe(catchError(() => of(void 0)))
      : of(void 0);
  }

  clearTokens(): void {
    this.tokens = undefined;
    localStorage.removeItem(this.storageKey);
  }

  hasRole(allowed: UserRole[]): boolean {
    const role = this.role;
    if (!role) {
      return false;
    }
    return allowed.includes(role);
  }

  hasMinimumRole(minRole: UserRole): boolean {
    const role = this.role;
    if (!role) {
      return false;
    }
    return this.roleLevel(role) >= this.roleLevel(minRole);
  }

  private persistTokens(response: AuthResponse, email?: string): AuthTokens {
    const tokens: AuthTokens = {
      accessToken: response.access_token,
      refreshToken: response.refresh_token,
      expiresAt: Date.now() + response.expires_in * 1000,
      email: email ?? this.tokens?.email
    };

    this.tokens = tokens;
    localStorage.setItem(this.storageKey, JSON.stringify(tokens));
    return tokens;
  }

  private loadTokens(): AuthTokens | undefined {
    try {
      const raw = localStorage.getItem(this.storageKey);
      if (!raw) {
        return undefined;
      }

      const parsed = JSON.parse(raw) as AuthTokens;
      if (!parsed.accessToken || !parsed.refreshToken || !parsed.expiresAt) {
        return undefined;
      }

      return parsed;
    } catch {
      return undefined;
    }
  }

  private extractRoleFromToken(token: string): UserRole | undefined {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) {
        return undefined;
      }

      const payload = JSON.parse(this.decodeBase64Url(parts[1])) as { role?: string };
      if (!payload.role) {
        return undefined;
      }

      if (payload.role === 'READ' || payload.role === 'READ_WRITE' || payload.role === 'ADMINISTRATOR') {
        return payload.role;
      }

      return undefined;
    } catch {
      return undefined;
    }
  }

  private extractUserIdFromToken(token: string): string | undefined {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) {
        return undefined;
      }

      const payload = JSON.parse(this.decodeBase64Url(parts[1])) as { sub?: string };
      return payload.sub;
    } catch {
      return undefined;
    }
  }

  private decodeBase64Url(value: string): string {
    const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
    const padded = normalized.padEnd(normalized.length + (4 - (normalized.length % 4)) % 4, '=');
    return atob(padded);
  }

  private roleLevel(role: UserRole): number {
    switch (role) {
      case 'READ':
        return 1;
      case 'READ_WRITE':
        return 2;
      case 'ADMINISTRATOR':
        return 3;
      default:
        return 0;
    }
  }

  createLoginRedirect(returnUrl: string): UrlTree {
    return this.router.createUrlTree(['/auth/login'], { queryParams: { returnUrl } });
  }
}
