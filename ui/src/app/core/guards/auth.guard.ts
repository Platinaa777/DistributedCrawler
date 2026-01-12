import { CanActivateFn, Router } from '@angular/router';
import { inject } from '@angular/core';
import { catchError, map, of } from 'rxjs';
import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = (_, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (authService.hasValidAccessToken()) {
    return true;
  }

  if (authService.refreshToken) {
    return authService.refreshTokens().pipe(
      map(() => true),
      catchError(() => of(authService.createLoginRedirect(state.url)))
    );
  }

  return router.createUrlTree(['/auth/login'], { queryParams: { returnUrl: state.url } });
};
