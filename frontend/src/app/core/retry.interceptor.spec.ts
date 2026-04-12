import { TestBed } from '@angular/core/testing';
import {
  HttpClientTestingModule,
  HttpTestingController,
} from '@angular/common/http/testing';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { retryInterceptor } from './retry.interceptor';
import { provideHttpClient, withInterceptors } from '@angular/common/http';

describe('retryInterceptor', () => {
  let httpMock: HttpTestingController;
  let httpClient: HttpClient;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [
        provideHttpClient(withInterceptors([retryInterceptor])),
      ],
    });

    httpMock = TestBed.inject(HttpTestingController);
    httpClient = TestBed.inject(HttpClient);
  });

  afterEach(() => {
    httpMock.verify();
  });

  describe('503 Service Unavailable', () => {
    it('should retry on 503 response', (done) => {
      httpClient.get('/test').subscribe({
        next: (response) => {
          expect(response).toEqual({ success: true });
          done();
        },
        error: () => {
          fail('Should have succeeded after retry');
        },
      });

      // First attempt: 503
      let req = httpMock.expectOne('/test');
      req.flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      // Retry after delay
      setTimeout(() => {
        req = httpMock.expectOne('/test');
        req.flush({ success: true });
      }, 1100); // After 1s backoff
    });

    it('should stop retrying after max retries exceeded', (done) => {
      let requestCount = 0;

      httpClient.get('/test').subscribe({
        next: () => {
          fail('Should not succeed');
        },
        error: (error: HttpErrorResponse) => {
          expect(error.status).toBe(503);
          expect(requestCount).toBe(4); // 1 initial + 3 retries
          done();
        },
      });

      // Respond to initial + 3 retries with 503
      for (let i = 0; i < 4; i++) {
        setTimeout(() => {
          requestCount++;
          const req = httpMock.expectOne('/test');
          req.flush('Service Unavailable', {
            status: 503,
            statusText: 'Service Unavailable',
          });
        }, i * 2000); // Account for exponential backoff
      }
    });
  });

  describe('429 Rate Limited', () => {
    it('should retry on 429 response', (done) => {
      httpClient.get('/test').subscribe({
        next: (response) => {
          expect(response).toEqual({ success: true });
          done();
        },
        error: () => {
          fail('Should have succeeded after retry');
        },
      });

      // First attempt: 429
      let req = httpMock.expectOne('/test');
      req.flush('Too Many Requests', {
        status: 429,
        statusText: 'Too Many Requests',
      });

      // Retry after delay
      setTimeout(() => {
        req = httpMock.expectOne('/test');
        req.flush({ success: true });
      }, 1100); // After 1s backoff
    });

    it('should respect Retry-After header (seconds)', (done) => {
      const startTime = Date.now();

      httpClient.get('/test').subscribe({
        next: (response) => {
          const elapsed = Date.now() - startTime;
          // Should wait ~3 seconds as specified in Retry-After
          expect(elapsed).toBeGreaterThan(2900);
          expect(response).toEqual({ success: true });
          done();
        },
        error: () => {
          fail('Should have succeeded after retry');
        },
      });

      // First attempt: 429 with Retry-After header
      let req = httpMock.expectOne('/test');
      req.flush('Too Many Requests', {
        status: 429,
        statusText: 'Too Many Requests',
        headers: { 'Retry-After': '3' },
      });

      // Retry after 3 seconds (as specified in header)
      setTimeout(() => {
        req = httpMock.expectOne('/test');
        req.flush({ success: true });
      }, 3100);
    });
  });

  describe('non-retryable errors', () => {
    it('should not retry on 400 Bad Request', (done) => {
      httpClient.get('/test').subscribe({
        next: () => {
          fail('Should not succeed');
        },
        error: (error: HttpErrorResponse) => {
          expect(error.status).toBe(400);
          done();
        },
      });

      const req = httpMock.expectOne('/test');
      req.flush('Bad Request', {
        status: 400,
        statusText: 'Bad Request',
      });

      // Should not attempt any retries
      httpMock.expectNone('/test');
    });

    it('should not retry on 401 Unauthorized', (done) => {
      httpClient.get('/test').subscribe({
        next: () => {
          fail('Should not succeed');
        },
        error: (error: HttpErrorResponse) => {
          expect(error.status).toBe(401);
          done();
        },
      });

      const req = httpMock.expectOne('/test');
      req.flush('Unauthorized', {
        status: 401,
        statusText: 'Unauthorized',
      });

      httpMock.expectNone('/test');
    });

    it('should not retry on 500 Internal Server Error', (done) => {
      httpClient.get('/test').subscribe({
        next: () => {
          fail('Should not succeed');
        },
        error: (error: HttpErrorResponse) => {
          expect(error.status).toBe(500);
          done();
        },
      });

      const req = httpMock.expectOne('/test');
      req.flush('Internal Server Error', {
        status: 500,
        statusText: 'Internal Server Error',
      });

      httpMock.expectNone('/test');
    });
  });

  describe('successful responses', () => {
    it('should pass through 200 OK without retry', (done) => {
      httpClient.get('/test').subscribe({
        next: (response) => {
          expect(response).toEqual({ data: 'success' });
          done();
        },
        error: () => {
          fail('Should not error');
        },
      });

      const req = httpMock.expectOne('/test');
      req.flush({ data: 'success' });

      // Should not attempt any additional requests
      httpMock.expectNone('/test');
    });
  });

  describe('exponential backoff timing', () => {
    it('should use correct backoff delays', (done) => {
      const timings: number[] = [];
      const startTime = Date.now();

      httpClient.get('/test').subscribe({
        next: () => {
          const duration = Date.now() - startTime;
          // 1000 + 2000 + 4000 = 7000ms total backoff
          expect(duration).toBeGreaterThan(6900);
          expect(duration).toBeLessThan(8000);
          done();
        },
        error: () => {
          fail('Should succeed after retries');
        },
      });

      // Initial request: 503
      let req = httpMock.expectOne('/test');
      timings.push(Date.now() - startTime);
      req.flush('Service Unavailable', {
        status: 503,
        statusText: 'Service Unavailable',
      });

      // Retry 1 (after 1s): 503
      setTimeout(() => {
        req = httpMock.expectOne('/test');
        timings.push(Date.now() - startTime);
        req.flush('Service Unavailable', {
          status: 503,
          statusText: 'Service Unavailable',
        });
      }, 1100);

      // Retry 2 (after 2s): 503
      setTimeout(() => {
        req = httpMock.expectOne('/test');
        timings.push(Date.now() - startTime);
        req.flush('Service Unavailable', {
          status: 503,
          statusText: 'Service Unavailable',
        });
      }, 3200);

      // Retry 3 (after 4s): success
      setTimeout(() => {
        req = httpMock.expectOne('/test');
        req.flush({ success: true });
      }, 7300);
    });
  });
});
