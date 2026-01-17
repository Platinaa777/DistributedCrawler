import { CanActivateFn, Router } from '@angular/router';
import { inject } from '@angular/core';
import { AuthService } from '../services/auth.service';
import { UserRole } from '../models';

export const roleGuard: CanActivateFn = (route) => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const minRole = route.data?.['minRole'] as UserRole | undefined;

  if (!minRole) {
    return true;
  }

  if (authService.hasMinimumRole(minRole)) {
    return true;
  }

  return router.createUrlTree(['/jobs']);
};
