import { inject } from '@angular/core';
import { Router, type CanActivateFn, type CanMatchFn } from '@angular/router';
import { AuthService } from './auth.service';

/**
 * Blocks navigation to authenticated routes when the user is not signed in.
 * Redirects to /auth/login, preserving the intended URL for post-login redirect.
 */
export const authGuard: CanActivateFn = (_route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (auth.isAuthenticated()) {
    return true;
  }

  return router.createUrlTree(['/auth/login'], {
    queryParams: { returnUrl: state.url },
  });
};

/**
 * Prevents the admin feature bundle from being downloaded for non-admin users.
 * Use with canMatch on the admin route to block both navigation and code download.
 *
 * Role is read from app_metadata (admin-controlled, not user-editable).
 * This is a UX convenience only — the Go middleware enforces the real boundary.
 */
export const adminGuard: CanMatchFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const user = auth.user();
  if (!user) {
    return router.createUrlTree(['/auth/login']);
  }

  const role = (user.app_metadata as Record<string, unknown>)?.['role'];
  if (role === 'admin') {
    return true;
  }

  return router.createUrlTree(['/']);
};
