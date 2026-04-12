import { ErrorHandler, Injectable, inject, isDevMode } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { MessageService } from 'primeng/api';

/**
 * GlobalErrorHandler catches unhandled errors and displays them to the user
 * via PrimeNG toast notifications. Logs full details in development mode.
 *
 * Provides user-friendly error messages for known error types (401, 429, 503).
 * Generic errors show a fallback message without exposing internal details.
 */
@Injectable()
export class GlobalErrorHandler implements ErrorHandler {
  private readonly messageService = inject(MessageService);

  handleError(error: unknown): void {
    // Log full error in development
    if (isDevMode()) {
      console.error('Global error caught:', error);
    }

    // Extract message and show toast
    const message = this.extractErrorMessage(error);
    const detail = this.extractErrorDetail(error);
    const severity = this.extractErrorSeverity(error);

    this.messageService.add({
      severity,
      summary: message,
      detail: detail || undefined,
      life: 5000, // Auto-dismiss after 5 seconds
    });
  }

  private extractErrorMessage(error: unknown): string {
    if (error instanceof HttpErrorResponse) {
      // Domain-specific messages for known HTTP statuses
      switch (error.status) {
        case 0:
          return 'Network error';
        case 401:
          return 'Session expired';
        case 403:
          return 'Access denied';
        case 404:
          return 'Not found';
        case 429:
          return 'Too many requests';
        case 500:
          return 'Server error';
        case 503:
          return 'Service temporarily unavailable';
        default:
          return 'An error occurred';
      }
    }

    if (error instanceof Error) {
      return error.message || 'An unexpected error occurred';
    }

    return 'An unexpected error occurred';
  }

  private extractErrorDetail(error: unknown): string | null {
    if (error instanceof HttpErrorResponse) {
      // Try to extract detail from RFC 7807 Problem Details response
      if (error.error && typeof error.error === 'object') {
        const problemDetails = error.error as any;
        if (problemDetails.detail) {
          return problemDetails.detail;
        }
      }

      // Fallback to error status text (sanitized)
      if (error.statusText && error.statusText !== 'Unknown Error') {
        return error.statusText;
      }

      // For 429, provide helpful retry guidance
      if (error.status === 429) {
        return 'Please wait a moment before trying again.';
      }

      // For 503, suggest refresh
      if (error.status === 503) {
        return 'Please try again in a few moments.';
      }

      return null;
    }

    return null;
  }

  private extractErrorSeverity(
    error: unknown,
  ): 'error' | 'warn' | 'info' | 'success' {
    if (error instanceof HttpErrorResponse) {
      // 429 and 503 are retryable, so lower severity
      if (error.status === 429 || error.status === 503) {
        return 'warn';
      }

      // 4xx are client errors (except rate limits)
      if (error.status >= 400 && error.status < 500) {
        return 'error';
      }

      // 5xx are server errors
      if (error.status >= 500) {
        return 'error';
      }
    }

    return 'error';
  }
}
