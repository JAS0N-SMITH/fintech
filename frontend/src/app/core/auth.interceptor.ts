import { inject } from '@angular/core';
import { HttpInterceptorFn, HttpRequest } from '@angular/common/http';
import { environment } from '../../environments/environment';
import { AuthService } from './auth.service';

/**
 * Attaches the Supabase JWT access token as a Bearer header on all requests
 * to the Go API. Requests to other origins (e.g. Finnhub) are left untouched.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  if (!isApiRequest(req)) {
    return next(req);
  }

  const token = inject(AuthService).accessToken();
  if (!token) {
    return next(req);
  }

  return next(
    req.clone({ setHeaders: { Authorization: `Bearer ${token}` } }),
  );
};

/** Returns true only for requests targeting the Go API base URL. */
function isApiRequest(req: HttpRequest<unknown>): boolean {
  return req.url.startsWith(environment.apiBaseUrl);
}
