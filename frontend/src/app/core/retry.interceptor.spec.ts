import { TestBed } from '@angular/core/testing';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { retryInterceptor } from './retry.interceptor';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';

describe('retryInterceptor', () => {
  let httpMock: HttpTestingController;
  let httpClient: HttpClient;
  const testRequestMatcher = (req: { url: string }) => req.url.includes('/test');

  beforeEach(() => {
    vi.useFakeTimers();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([retryInterceptor])),
        provideHttpClientTesting(),
      ],
    });

    httpMock = TestBed.inject(HttpTestingController);
    httpClient = TestBed.inject(HttpClient);
  });

  afterEach(() => {
    httpMock.verify();
    vi.useRealTimers();
  });

  describe('503 Service Unavailable', () => {
    it('should retry on 503 response', async () => {
      const responsePromise = firstValueFrom(httpClient.get('/test'));

      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      await vi.advanceTimersByTimeAsync(1000);
      httpMock.expectOne(testRequestMatcher).flush({ success: true });

      await expect(responsePromise).resolves.toEqual({ success: true });
    });

    it('should stop retrying after max retries exceeded', async () => {
      const resultPromise = firstValueFrom(httpClient.get('/test')).catch(
        (error: HttpErrorResponse) => error,
      );

      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      await vi.advanceTimersByTimeAsync(1000);
      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      await vi.advanceTimersByTimeAsync(2000);
      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      await vi.advanceTimersByTimeAsync(4000);
      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      const error = (await resultPromise) as HttpErrorResponse;
      expect(error.status).toBe(503);
    });
  });

  describe('429 Rate Limited', () => {
    it('should retry on 429 response', async () => {
      const responsePromise = firstValueFrom(httpClient.get('/test'));

      httpMock.expectOne(testRequestMatcher).flush('Too Many Requests', {
        status: 429,
        statusText: 'Too Many Requests',
      });

      await vi.advanceTimersByTimeAsync(1000);
      httpMock.expectOne(testRequestMatcher).flush({ success: true });

      await expect(responsePromise).resolves.toEqual({ success: true });
    });

    it('should respect Retry-After header (seconds)', async () => {
      const responsePromise = firstValueFrom(httpClient.get('/test'));

      httpMock.expectOne(testRequestMatcher).flush('Too Many Requests', {
        status: 429,
        statusText: 'Too Many Requests',
        headers: { 'Retry-After': '3' },
      });

      await vi.advanceTimersByTimeAsync(2999);
      httpMock.expectNone(testRequestMatcher);

      await vi.advanceTimersByTimeAsync(1);
      httpMock.expectOne(testRequestMatcher).flush({ success: true });

      await expect(responsePromise).resolves.toEqual({ success: true });
    });
  });

  describe('non-retryable errors', () => {
    it('should not retry on 400 Bad Request', async () => {
      const resultPromise = firstValueFrom(httpClient.get('/test')).catch(
        (error: HttpErrorResponse) => error,
      );

      httpMock.expectOne(testRequestMatcher).flush('Bad Request', {
        status: 400,
        statusText: 'Bad Request',
      });

      const error = (await resultPromise) as HttpErrorResponse;
      expect(error.status).toBe(400);
      httpMock.expectNone(testRequestMatcher);
    });

    it('should not retry on 401 Unauthorized', async () => {
      const resultPromise = firstValueFrom(httpClient.get('/test')).catch(
        (error: HttpErrorResponse) => error,
      );

      httpMock.expectOne(testRequestMatcher).flush('Unauthorized', {
        status: 401,
        statusText: 'Unauthorized',
      });

      const error = (await resultPromise) as HttpErrorResponse;
      expect(error.status).toBe(401);
      httpMock.expectNone(testRequestMatcher);
    });

    it('should not retry on 500 Internal Server Error', async () => {
      const resultPromise = firstValueFrom(httpClient.get('/test')).catch(
        (error: HttpErrorResponse) => error,
      );

      httpMock.expectOne(testRequestMatcher).flush('Internal Server Error', {
        status: 500,
        statusText: 'Internal Server Error',
      });

      const error = (await resultPromise) as HttpErrorResponse;
      expect(error.status).toBe(500);
      httpMock.expectNone(testRequestMatcher);
    });
  });

  describe('successful responses', () => {
    it('should pass through 200 OK without retry', async () => {
      const responsePromise = firstValueFrom(httpClient.get('/test'));
      httpMock.expectOne(testRequestMatcher).flush({ data: 'success' });
      await expect(responsePromise).resolves.toEqual({ data: 'success' });
      httpMock.expectNone(testRequestMatcher);
    });
  });

  describe('exponential backoff timing', () => {
    it('should use correct backoff delays', async () => {
      const responsePromise = firstValueFrom(httpClient.get('/test'));

      // Initial request: 503, next after 1s
      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      await vi.advanceTimersByTimeAsync(999);
      httpMock.expectNone(testRequestMatcher);
      await vi.advanceTimersByTimeAsync(1);
      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      // Next retry after 2s
      await vi.advanceTimersByTimeAsync(1999);
      httpMock.expectNone(testRequestMatcher);
      await vi.advanceTimersByTimeAsync(1);
      httpMock.expectOne(testRequestMatcher).flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      // Next retry after 4s
      await vi.advanceTimersByTimeAsync(3999);
      httpMock.expectNone(testRequestMatcher);
      await vi.advanceTimersByTimeAsync(1);
      httpMock.expectOne(testRequestMatcher).flush({ success: true });

      await expect(responsePromise).resolves.toEqual({ success: true });
    });
  });
});
