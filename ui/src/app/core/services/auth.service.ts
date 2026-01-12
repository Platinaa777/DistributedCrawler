import { Injectable } from '@angular/core';
import { Router, UrlTree } from '@angular/router';
import { Observable, of, throwError } from 'rxjs';
import { catchError, finalize, map, shareReplay, tap } from 'rxjs/operators';
import { AuthApiService } from './api/auth-api.service';
import { AuthResponse, AuthTokens } from '../models';

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

  hasValidAccessToken(): boolean {
    if (!this.tokens) {
      return false;
    }

    const buffer = 5_000; // refresh slightly before expiry
    return this.tokens.expiresAt - buffer > Date.now();
  }

  login(email: string, password: string): Observable<AuthTokens> {
    return this.authApi.login(email, password).pipe(
      map((response) => this.persistTokens(response))
    );
  }

  register(email: string, password: string): Observable<AuthTokens> {
    return this.authApi.register(email, password).pipe(
      map((response) => this.persistTokens(response))
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

    const logout$ = refreshToken
      ? this.authApi.logout(refreshToken).pipe(
          catchError(() => of(void 0)) // ignore logout errors
        )
      : of(void 0);

    if (redirectToLogin) {
      return logout$.pipe(tap(() => this.router.navigate(['/auth/login'])));
    }

    return logout$;
  }

  clearTokens(): void {
    this.tokens = undefined;
    localStorage.removeItem(this.storageKey);
  }

  private persistTokens(response: AuthResponse): AuthTokens {
    const tokens: AuthTokens = {
      accessToken: response.access_token,
      refreshToken: response.refresh_token,
      expiresAt: Date.now() + response.expires_in * 1000
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

  createLoginRedirect(returnUrl: string): UrlTree {
    return this.router.createUrlTree(['/auth/login'], { queryParams: { returnUrl } });
  }
}
