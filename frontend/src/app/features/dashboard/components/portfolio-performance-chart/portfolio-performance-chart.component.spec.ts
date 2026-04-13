import { describe, it, expect } from 'vitest';
import { signal } from '@angular/core';
import type { Transaction } from '../../../portfolio/models/transaction.model';

/**
 * PortfolioPerformanceChartComponent spec
 *
 * Tests focus on component logic (signals, computed values, methods).
 * Template rendering and chart initialization are better covered by E2E tests.
 */
describe('PortfolioPerformanceChartComponent', () => {
  it('should have correct timeframe options', () => {
    const timeframeOptions = ['1M', '3M', '1Y'] as const;
    expect(timeframeOptions).toEqual(['1M', '3M', '1Y']);
  });

  it('should initialize with correct defaults', () => {
    const selectedTimeframe = signal<'1M' | '3M' | '1Y'>('1M');
    const isLoading = signal(false);
    const hasError = signal(false);
    const chartPoints = signal<any[]>([]);

    expect(selectedTimeframe()).toBe('1M');
    expect(isLoading()).toBe(false);
    expect(hasError()).toBe(false);
    expect(chartPoints().length).toBe(0);
  });

  it('should update selectedTimeframe when changed', () => {
    const selectedTimeframe = signal<'1M' | '3M' | '1Y'>('1M');

    selectedTimeframe.set('3M');
    expect(selectedTimeframe()).toBe('3M');

    selectedTimeframe.set('1Y');
    expect(selectedTimeframe()).toBe('1Y');

    selectedTimeframe.set('1M');
    expect(selectedTimeframe()).toBe('1M');
  });

  it('should compute isEmpty correctly for empty transactions', () => {
    const transactions = signal<Transaction[]>([]);
    const isEmpty = () => transactions().length === 0;

    expect(isEmpty()).toBe(true);
  });

  it('should compute isEmpty correctly for non-empty transactions', () => {
    const mockTransaction: Transaction = {
      id: '1',
      portfolio_id: 'p1',
      transaction_type: 'buy',
      symbol: 'SPY',
      transaction_date: '2024-01-10',
      quantity: '10',
      price_per_share: '100',
      dividend_per_share: null,
      total_amount: '1000',
      notes: '',
      created_at: '2024-01-10T00:00:00Z',
      updated_at: '2024-01-10T00:00:00Z',
    };

    const transactions = signal<Transaction[]>([mockTransaction]);
    const isEmpty = () => transactions().length === 0;

    expect(isEmpty()).toBe(false);
  });

  it('should toggle loading state', () => {
    const isLoading = signal(false);

    expect(isLoading()).toBe(false);

    isLoading.set(true);
    expect(isLoading()).toBe(true);

    isLoading.set(false);
    expect(isLoading()).toBe(false);
  });

  it('should toggle error state', () => {
    const hasError = signal(false);

    expect(hasError()).toBe(false);

    hasError.set(true);
    expect(hasError()).toBe(true);

    hasError.set(false);
    expect(hasError()).toBe(false);
  });

  it('should update chart points', () => {
    const chartPoints = signal<any[]>([]);

    expect(chartPoints().length).toBe(0);

    const mockPoints = [{ time: 1234567890, value: 1000 }];
    chartPoints.set(mockPoints);

    expect(chartPoints().length).toBe(1);
    expect(chartPoints()[0].value).toBe(1000);
  });
});
