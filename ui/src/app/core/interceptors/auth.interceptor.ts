import { Injectable } from '@angular/core';
import {
  HttpErrorResponse,
  HttpEvent,
  HttpHandler,
  HttpInterceptor,
  HttpRequest
} from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, throwError } from 'rxjs';
import { catchError, switchMap } from 'rxjs/operators';
import { AuthService } from '../services/auth.service';

@Injectable()
export class AuthInterceptor implements HttpInterceptor {
  private readonly retryHeader = 'X-Auth-Retry';

  constructor(
    private authService: AuthService,
    private router: Router
  ) {}

  intercept(req: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {
    if (this.isAuthRequest(req.url)) {
      return next.handle(req);
    }

    if (this.authService.hasValidAccessToken()) {
      req = this.addAuthHeader(req, this.authService.accessToken!);
      return next.handle(req).pipe(
        catchError((error) => this.handleAuthError(error, req, next))
      );
    }

    if (this.authService.refreshToken) {
      return this.authService.refreshTokens().pipe(
        switchMap((tokens) => {
          const authorizedRequest = this.addAuthHeader(req, tokens.accessToken);
          return next.handle(authorizedRequest);
        }),
        catchError((error) => this.handleRefreshFailure(error))
      );
    }

    return next.handle(req).pipe(
      catchError((error) => this.handleAuthError(error, req, next))
    );
  }

  private handleAuthError(
    error: unknown,
    req: HttpRequest<unknown>,
    next: HttpHandler
  ): Observable<HttpEvent<unknown>> {
    if (error instanceof HttpErrorResponse && error.status === 403) {
      return throwError(() => error);
    }

    if (error instanceof HttpErrorResponse && error.status === 401) {
      if (req.headers.has(this.retryHeader) || !this.authService.refreshToken || this.isAuthRequest(req.url)) {
        this.forceLoginRedirect();
        return throwError(() => error);
      }

      return this.authService.refreshTokens().pipe(
        switchMap((tokens) => {
          const retryRequest = this.addAuthHeader(
            req.clone({ headers: req.headers.set(this.retryHeader, '1') }),
            tokens.accessToken
          );
          return next.handle(retryRequest);
        }),
        catchError((refreshError) => this.handleRefreshFailure(refreshError))
      );
    }

    return throwError(() => error);
  }

  private handleRefreshFailure(error: unknown): Observable<never> {
    this.forceLoginRedirect();
    return throwError(() => error);
  }

  private addAuthHeader(req: HttpRequest<unknown>, accessToken: string): HttpRequest<unknown> {
    return req.clone({
      setHeaders: {
        Authorization: `Bearer ${accessToken}`
      }
    });
  }

  private isAuthRequest(url: string): boolean {
    return url.includes('/auth/');
  }

  private forceLoginRedirect(): void {
    this.authService.clearTokens();
    this.router.navigate(['/auth/login'], {
      queryParams: { returnUrl: this.router.url }
    });
  }
}
