import { describe, it, expect } from 'vitest';
import { derivePortfolioValues } from './portfolio-performance-chart.utils';
import type { Transaction } from '../../../portfolio/models/transaction.model';
import type { Bar } from '../../../portfolio/models/market-data.model';

describe('derivePortfolioValues', () => {
  it('returns empty array when transactions are empty', () => {
    const result = derivePortfolioValues([], []);
    expect(result).toEqual([]);
  });

  it('returns empty array when symbol bars are empty', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-15',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-15T00:00:00Z',
        updated_at: '2024-01-15T00:00:00Z',
      },
    ];
    const result = derivePortfolioValues(transactions, []);
    expect(result).toEqual([]);
  });

  it('returns portfolio value for a single symbol with a single buy', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
    ];

    const bars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-01-15T00:00:00Z',
        open: 100,
        high: 105,
        low: 99,
        close: 102,
        volume: 1000000,
      },
    ];

    const result = derivePortfolioValues(transactions, [{ sym: 'SPY', bars }]);

    // 10 shares @ $102 close = $1020
    expect(result).toHaveLength(1);
    expect(result[0]).toEqual({
      time: expect.any(Number),
      value: 1020,
    });
  });

  it('reflects cumulative quantity after buy and sell', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '20',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '2000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
      {
        id: '2',
        portfolio_id: 'p1',
        transaction_type: 'sell',
        symbol: 'SPY',
        transaction_date: '2024-01-12',
        quantity: '5',
        price_per_share: '101',
        dividend_per_share: undefined,
        total_amount: '505',
        notes: '',
        created_at: '2024-01-12T00:00:00Z',
        updated_at: '2024-01-12T00:00:00Z',
      },
    ];

    const bars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-01-10T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-01-12T00:00:00Z',
        open: 101,
        high: 102,
        low: 100,
        close: 101,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-01-15T00:00:00Z',
        open: 102,
        high: 105,
        low: 101,
        close: 104,
        volume: 1000000,
      },
    ];

    const result = derivePortfolioValues(transactions, [{ sym: 'SPY', bars }]);

    // 2024-01-10: 20 shares @ $100 = $2000
    // 2024-01-12: (20 - 5) = 15 shares @ $101 = $1515
    // 2024-01-15: 15 shares @ $104 = $1560
    expect(result).toHaveLength(3);
    expect(result[0].value).toBe(2000);
    expect(result[1].value).toBe(1515);
    expect(result[2].value).toBe(1560);
  });

  it('skips dividend transactions (does not change share count)', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
      {
        id: '2',
        portfolio_id: 'p1',
        transaction_type: 'dividend',
        symbol: 'SPY',
        transaction_date: '2024-01-12',
        quantity: undefined,
        price_per_share: undefined,
        dividend_per_share: '2.5',
        total_amount: '25',
        notes: '',
        created_at: '2024-01-12T00:00:00Z',
        updated_at: '2024-01-12T00:00:00Z',
      },
    ];

    const bars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-01-10T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-01-12T00:00:00Z',
        open: 101,
        high: 102,
        low: 100,
        close: 101,
        volume: 1000000,
      },
    ];

    const result = derivePortfolioValues(transactions, [{ sym: 'SPY', bars }]);

    // Both dates should have 10 shares (dividend does not change quantity)
    expect(result).toHaveLength(2);
    expect(result[0].value).toBe(1000); // 10 @ $100
    expect(result[1].value).toBe(1010); // 10 @ $101
  });

  it('sums values across multiple symbols on the same date', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
      {
        id: '2',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'AAPL',
        transaction_date: '2024-01-10',
        quantity: '5',
        price_per_share: '200',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
    ];

    const bars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-01-10T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
      {
        symbol: 'AAPL',
        timestamp: '2024-01-10T00:00:00Z',
        open: 200,
        high: 200,
        low: 199,
        close: 200,
        volume: 1000000,
      },
    ];

    const result = derivePortfolioValues(transactions, [
      { sym: 'SPY', bars: bars.slice(0, 1) },
      { sym: 'AAPL', bars: bars.slice(1, 2) },
    ]);

    // (10 @ $100) + (5 @ $200) = $1000 + $1000 = $2000
    expect(result).toHaveLength(1);
    expect(result[0].value).toBe(2000);
  });

  it('treats reinvested_dividend like a buy (adds to quantity)', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
      {
        id: '2',
        portfolio_id: 'p1',
        transaction_type: 'reinvested_dividend',
        symbol: 'SPY',
        transaction_date: '2024-01-12',
        quantity: '1',
        price_per_share: '101',
        dividend_per_share: undefined,
        total_amount: '101',
        notes: '',
        created_at: '2024-01-12T00:00:00Z',
        updated_at: '2024-01-12T00:00:00Z',
      },
    ];

    const bars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-01-10T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-01-12T00:00:00Z',
        open: 101,
        high: 102,
        low: 100,
        close: 101,
        volume: 1000000,
      },
    ];

    const result = derivePortfolioValues(transactions, [{ sym: 'SPY', bars }]);

    // 2024-01-10: 10 shares @ $100 = $1000
    // 2024-01-12: (10 + 1) = 11 shares @ $101 = $1111
    expect(result).toHaveLength(2);
    expect(result[0].value).toBe(1000);
    expect(result[1].value).toBe(1111);
  });

  it('does not emit points for dates where portfolio value is zero', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
      {
        id: '2',
        portfolio_id: 'p1',
        transaction_type: 'sell',
        symbol: 'SPY',
        transaction_date: '2024-01-12',
        quantity: '10',
        price_per_share: '101',
        dividend_per_share: undefined,
        total_amount: '1010',
        notes: '',
        created_at: '2024-01-12T00:00:00Z',
        updated_at: '2024-01-12T00:00:00Z',
      },
    ];

    const bars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-01-10T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-01-12T00:00:00Z',
        open: 101,
        high: 102,
        low: 100,
        close: 101,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-01-15T00:00:00Z',
        open: 102,
        high: 105,
        low: 101,
        close: 104,
        volume: 1000000,
      },
    ];

    const result = derivePortfolioValues(transactions, [{ sym: 'SPY', bars }]);

    // 2024-01-10: 10 shares @ $100 = $1000 ✓
    // 2024-01-12: 0 shares @ $101 = $0 ✗ (not emitted)
    // 2024-01-15: 0 shares @ $104 = $0 ✗ (not emitted)
    expect(result).toHaveLength(1);
    expect(result[0].value).toBe(1000);
  });

  it('outputs time values in strictly ascending order', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
    ];

    const bars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-03-20T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-01-10T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
      {
        symbol: 'SPY',
        timestamp: '2024-02-15T00:00:00Z',
        open: 101,
        high: 102,
        low: 100,
        close: 101,
        volume: 1000000,
      },
    ];

    const result = derivePortfolioValues(transactions, [{ sym: 'SPY', bars }]);

    // Verify times are strictly ascending
    for (let i = 1; i < result.length; i++) {
      expect(Number(result[i].time)).toBeGreaterThan(Number(result[i - 1].time));
    }
  });

  it('handles bars with no corresponding transactions gracefully', () => {
    const transactions: Transaction[] = [
      {
        id: '1',
        portfolio_id: 'p1',
        transaction_type: 'buy',
        symbol: 'SPY',
        transaction_date: '2024-01-10',
        quantity: '10',
        price_per_share: '100',
        dividend_per_share: undefined,
        total_amount: '1000',
        notes: '',
        created_at: '2024-01-10T00:00:00Z',
        updated_at: '2024-01-10T00:00:00Z',
      },
    ];

    const spyBars: Bar[] = [
      {
        symbol: 'SPY',
        timestamp: '2024-01-10T00:00:00Z',
        open: 100,
        high: 100,
        low: 99,
        close: 100,
        volume: 1000000,
      },
    ];

    const aaplBars: Bar[] = [
      {
        symbol: 'AAPL',
        timestamp: '2024-01-10T00:00:00Z',
        open: 200,
        high: 200,
        low: 199,
        close: 200,
        volume: 1000000,
      },
    ];

    // AAPL has bars but no transactions — should not crash
    const result = derivePortfolioValues(transactions, [
      { sym: 'SPY', bars: spyBars },
      { sym: 'AAPL', bars: aaplBars },
    ]);

    // Should only have value from SPY (AAPL has 0 quantity)
    expect(result).toHaveLength(1);
    expect(result[0].value).toBe(1000);
  });
});
