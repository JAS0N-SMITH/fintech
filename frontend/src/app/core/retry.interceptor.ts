import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { throwError, timer } from 'rxjs';
import { retryWhen, mergeMap, finalize } from 'rxjs/operators';

/** Configuration for retry backoff strategy. */
const RETRY_CONFIG = {
  maxRetries: 3,
  initialDelayMs: 1000,
  backoffMultiplier: 2,
};

/**
 * RetryInterceptor implements exponential backoff retry logic for transient failures.
 *
 * Retries on:
 *   - 503 Service Unavailable (always retried)
 *   - 429 Rate Limited (always retried, respects Retry-After header)
 *
 * Does not retry:
 *   - 4xx client errors (except 429)
 *   - 5xx errors other than 503
 *
 * Backoff strategy: 1s, 2s, 4s (exponential with 2x multiplier, max 3 retries)
 * Retry-After header respected for 429 responses.
 */
export const retryInterceptor: HttpInterceptorFn = (req, next) => {
  let retryCount = 0;

  return next(req).pipe(
    retryWhen((errors) =>
      errors.pipe(
        mergeMap((error, index) => {
          retryCount = index + 1;

          // Only retry on 503 or 429
          if (!(error instanceof HttpErrorResponse)) {
            return throwError(() => error);
          }

          if (error.status !== 503 && error.status !== 429) {
            return throwError(() => error);
          }

          // Max retries exceeded
          if (retryCount > RETRY_CONFIG.maxRetries) {
            return throwError(() => error);
          }

          // Calculate delay
          let delayMs = calculateDelay(retryCount);

          // Respect Retry-After header for 429
          if (error.status === 429) {
            const retryAfter = error.headers.get('Retry-After');
            if (retryAfter) {
              // Retry-After can be in seconds or an HTTP-date
              const retryAfterSeconds = parseInt(retryAfter, 10);
              if (!isNaN(retryAfterSeconds)) {
                delayMs = retryAfterSeconds * 1000;
              }
            }
          }

          return timer(delayMs);
        }),
      ),
    ),
  );
};

/**
 * Calculates exponential backoff delay.
 * Formula: initialDelayMs * (backoffMultiplier ^ (retryCount - 1))
 *
 * Examples:
 *   retryCount=1: 1000ms
 *   retryCount=2: 2000ms
 *   retryCount=3: 4000ms
 */
function calculateDelay(retryCount: number): number {
  return (
    RETRY_CONFIG.initialDelayMs *
    Math.pow(RETRY_CONFIG.backoffMultiplier, retryCount - 1)
  );
}
