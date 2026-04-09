import { inject } from '@angular/core';
import { toObservable } from '@angular/core/rxjs-interop';
import { Router, type CanActivateFn, type CanMatchFn } from '@angular/router';
import { filter, map, take } from 'rxjs';
import { AuthService } from './auth.service';

/** AppMetadata shape stored in Supabase (admin-controlled, not user-editable). */
interface AppMetadata {
  role?: string;
}

/**
 * Blocks navigation to authenticated routes when the user is not signed in.
 * Waits for the initial session check to complete before deciding, preventing
 * a redirect flash while isLoading is true on page load.
 * Redirects to /auth/login with returnUrl preserved for post-login navigation.
 */
export const authGuard: CanActivateFn = (_route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  // Wait until isLoading is false before evaluating auth state.
  return toObservable(auth.isLoading).pipe(
    filter((loading) => !loading),
    take(1),
    map(() => {
      if (auth.isAuthenticated()) return true;
      return router.createUrlTree(['/auth/login'], {
        queryParams: { returnUrl: state.url },
      });
    }),
  );
};

/**
 * Prevents the admin feature bundle from being downloaded for non-admin users.
 * Use with canMatch on the admin route to block both navigation and code download.
 * Waits for initial session load before evaluating role.
 *
 * Role is read from app_metadata (admin-controlled, not user-editable).
 * This is a UX convenience only — the Go middleware enforces the real boundary.
 */
export const adminGuard: CanMatchFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  return toObservable(auth.isLoading).pipe(
    filter((loading) => !loading),
    take(1),
    map(() => {
      const user = auth.user();
      if (!user) return router.createUrlTree(['/auth/login']);

      const meta = user.app_metadata as AppMetadata;
      if (meta?.role === 'admin') return true;

      return router.createUrlTree(['/']);
    }),
  );
};
