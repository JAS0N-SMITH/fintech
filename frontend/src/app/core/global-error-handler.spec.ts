import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { MessageService } from 'primeng/api';
import { GlobalErrorHandler } from './global-error-handler';

describe('GlobalErrorHandler', () => {
  let handler: GlobalErrorHandler;
  let messageService: MessageService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [GlobalErrorHandler, MessageService],
    });

    handler = TestBed.inject(GlobalErrorHandler);
    messageService = TestBed.inject(MessageService);
    spyOn(messageService, 'add');
  });

  describe('generic errors', () => {
    it('should show error message for generic Error', () => {
      const error = new Error('Something went wrong');
      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Something went wrong',
        }),
      );
    });

    it('should show fallback message for non-Error objects', () => {
      handler.handleError('string error');

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'An unexpected error occurred',
        }),
      );
    });
  });

  describe('HTTP errors', () => {
    it('should show 401 session expired message', () => {
      const error = new HttpErrorResponse({
        status: 401,
        statusText: 'Unauthorized',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Session expired',
        }),
      );
    });

    it('should show 429 rate limit message with helpful detail', () => {
      const error = new HttpErrorResponse({
        status: 429,
        statusText: 'Too Many Requests',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'warn',
          summary: 'Too many requests',
          detail: 'Please wait a moment before trying again.',
        }),
      );
    });

    it('should show 503 service unavailable message', () => {
      const error = new HttpErrorResponse({
        status: 503,
        statusText: 'Service Unavailable',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'warn',
          summary: 'Service temporarily unavailable',
          detail: 'Please try again in a few moments.',
        }),
      );
    });

    it('should extract detail from RFC 7807 Problem Details response', () => {
      const error = new HttpErrorResponse({
        status: 500,
        statusText: 'Internal Server Error',
        error: {
          type: 'https://api.example.com/errors/server-error',
          title: 'Server Error',
          detail: 'The server encountered an unexpected condition.',
          status: 500,
        },
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Server error',
          detail: 'The server encountered an unexpected condition.',
        }),
      );
    });

    it('should show 403 access denied message', () => {
      const error = new HttpErrorResponse({
        status: 403,
        statusText: 'Forbidden',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Access denied',
        }),
      );
    });

    it('should show 404 not found message', () => {
      const error = new HttpErrorResponse({
        status: 404,
        statusText: 'Not Found',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Not found',
        }),
      );
    });

    it('should show 0 status as network error', () => {
      const error = new HttpErrorResponse({
        status: 0,
        statusText: 'Unknown Error',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Network error',
        }),
      );
    });
  });

  describe('severity mapping', () => {
    it('should use warn severity for 429', () => {
      const error = new HttpErrorResponse({
        status: 429,
        statusText: 'Too Many Requests',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'warn',
        }),
      );
    });

    it('should use warn severity for 503', () => {
      const error = new HttpErrorResponse({
        status: 503,
        statusText: 'Service Unavailable',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'warn',
        }),
      );
    });

    it('should use error severity for 4xx (non-retryable)', () => {
      const error = new HttpErrorResponse({
        status: 400,
        statusText: 'Bad Request',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
        }),
      );
    });

    it('should use error severity for 5xx errors', () => {
      const error = new HttpErrorResponse({
        status: 500,
        statusText: 'Internal Server Error',
      });

      handler.handleError(error);

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
        }),
      );
    });
  });
});
