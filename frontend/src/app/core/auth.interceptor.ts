import { inject } from '@angular/core';
import { HttpClient, HttpErrorResponse, HttpInterceptorFn, HttpRequest } from '@angular/common/http';
import { catchError, switchMap, throwError } from 'rxjs';
import { environment } from '../../environments/environment';
import { AuthService, GoSessionResponse } from './auth.service';

/**
 * Module-level flag to prevent a storm of concurrent 401s from each triggering
 * a refresh call. Reset to false in both success and error paths.
 */
let isRefreshing = false;

/**
 * Attaches the Supabase JWT access token as a Bearer header on all requests
 * to the Go API. Handles 401 responses by attempting one token refresh via the
 * Go /auth/session endpoint, then retrying the original request with the new
 * token. If refresh fails, calls signOut() and lets the error propagate.
 *
 * Requests to other origins (e.g. Finnhub) are left untouched.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  if (!isApiRequest(req)) {
    return next(req);
  }

  const auth = inject(AuthService);
  const token = auth.accessToken();
  const authedReq = token
    ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })
    : req;

  return next(authedReq).pipe(
    catchError((error: unknown) => {
      // Only attempt 401 recovery if: error is 401, we're not already refreshing,
      // and it's not the refresh endpoint itself (infinite loop guard).
      if (
        error instanceof HttpErrorResponse &&
        error.status === 401 &&
        !isRefreshing &&
        !req.url.includes('/auth/session')
      ) {
        isRefreshing = true;
        const http = inject(HttpClient);
        const refreshUrl = `${environment.apiBaseUrl}/auth/session`;

        return http
          .get<GoSessionResponse>(refreshUrl, { withCredentials: true })
          .pipe(
            switchMap((resp) => {
              isRefreshing = false;
              // Apply the new session and reschedule the refresh timer.
              auth.applyRefreshedSession(resp);
              // Retry the original request with the new token.
              const retryReq = req.clone({
                setHeaders: { Authorization: `Bearer ${resp.access_token}` },
              });
              return next(retryReq);
            }),
            catchError((refreshErr) => {
              isRefreshing = false;
              // Refresh failed — force logout and propagate the error.
              void auth.signOut();
              return throwError(() => refreshErr);
            }),
          );
      }
      return throwError(() => error);
    }),
  );
};

/**
 * Returns true only for requests targeting the Go API base URL.
 * Excludes /auth/session — those endpoints are cookie-based and handled specially.
 */
function isApiRequest(req: HttpRequest<unknown>): boolean {
  return (
    req.url.startsWith(environment.apiBaseUrl) && !req.url.includes('/auth/session')
  );
}
