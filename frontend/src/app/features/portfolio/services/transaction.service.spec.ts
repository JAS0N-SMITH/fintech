import { TestBed } from '@angular/core/testing';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { vi } from 'vitest';
import { signal } from '@angular/core';
import { TransactionService, deriveHoldings } from './transaction.service';
import type { Transaction } from '../models/transaction.model';
import { environment } from '../../../../environments/environment';
import { TickerStateService } from '../../../core/ticker-state.service';

// Minimal stub for TickerStateService — only the methods called by TransactionService.
function makeTickerStateStub() {
  return {
    tickers: signal<Record<string, { currentPrice: number | null }>>({}).asReadonly(),
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  };
}

function txBase(portfolioId: string): string {
  return `${environment.apiBaseUrl}/portfolios/${portfolioId}/transactions`;
}

function makeTx(overrides: Partial<Transaction> = {}): Transaction {
  return {
    id: 'tx1',
    portfolio_id: 'p1',
    transaction_type: 'buy',
    symbol: 'AAPL',
    transaction_date: '2026-01-15',
    quantity: '10',
    price_per_share: '150.00',
    total_amount: '1500.00',
    created_at: '2026-01-15T12:00:00Z',
    updated_at: '2026-01-15T12:00:00Z',
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// deriveHoldings — pure function tests (no TestBed needed)
// ---------------------------------------------------------------------------

describe('deriveHoldings()', () => {
  it('returns empty array for no transactions', () => {
    expect(deriveHoldings([])).toEqual([]);
  });

  it('100-share buy → holding with quantity 100', () => {
    const holdings = deriveHoldings([
      makeTx({ quantity: '100', price_per_share: '50.00', total_amount: '5000.00' }),
    ]);
    expect(holdings.length).toBe(1);
    expect(holdings[0].symbol).toBe('AAPL');
    expect(holdings[0].quantity).toBe('100');
    expect(holdings[0].avgCostBasis).toBe('50.00');
    expect(holdings[0].totalCost).toBe('5000.00');
  });

  it('100 buy + 50 sell → quantity 50', () => {
    const holdings = deriveHoldings([
      makeTx({ id: 'tx1', quantity: '100', total_amount: '5000.00' }),
      makeTx({ id: 'tx2', transaction_type: 'sell', quantity: '50', total_amount: '2600.00' }),
    ]);
    expect(holdings[0].quantity).toBe('50');
    // Cost basis is based on buy total only (sells do not change cost basis)
    expect(holdings[0].totalCost).toBe('5000.00');
    expect(holdings[0].avgCostBasis).toBe('100.00'); // 5000 / 50
  });

  it('full sell → holding disappears', () => {
    const holdings = deriveHoldings([
      makeTx({ id: 'tx1', quantity: '10', total_amount: '1500.00' }),
      makeTx({ id: 'tx2', transaction_type: 'sell', quantity: '10', total_amount: '1800.00' }),
    ]);
    expect(holdings).toEqual([]);
  });

  it('dividend-only → no holding created (dividends do not grant shares)', () => {
    const holdings = deriveHoldings([
      makeTx({
        transaction_type: 'dividend',
        quantity: undefined,
        price_per_share: undefined,
        dividend_per_share: '0.25',
        total_amount: '25.00',
      }),
    ]);
    expect(holdings).toEqual([]);
  });

  it('reinvested_dividend adds to quantity and cost basis', () => {
    const holdings = deriveHoldings([
      makeTx({ id: 'tx1', quantity: '100', price_per_share: '50.00', total_amount: '5000.00' }),
      makeTx({
        id: 'tx2',
        transaction_type: 'reinvested_dividend',
        quantity: '2',
        price_per_share: '52.00',
        dividend_per_share: '0.26',
        total_amount: '104.00',
      }),
    ]);
    expect(holdings[0].quantity).toBe('102');
    expect(holdings[0].totalCost).toBe('5104.00');
  });

  it('multiple symbols produce separate holdings sorted by symbol', () => {
    const holdings = deriveHoldings([
      makeTx({ symbol: 'MSFT', quantity: '5', total_amount: '1000.00' }),
      makeTx({ symbol: 'AAPL', quantity: '10', total_amount: '1500.00' }),
    ]);
    expect(holdings[0].symbol).toBe('AAPL');
    expect(holdings[1].symbol).toBe('MSFT');
  });
});

// ---------------------------------------------------------------------------
// TransactionService — HTTP + signal tests
// ---------------------------------------------------------------------------

describe('TransactionService', () => {
  let service: TransactionService;
  let httpMock: HttpTestingController;
  let tickerStub: ReturnType<typeof makeTickerStateStub>;

  beforeEach(() => {
    tickerStub = makeTickerStateStub();
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: TickerStateService, useValue: tickerStub },
      ],
    });
    service = TestBed.inject(TransactionService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  describe('loadByPortfolio()', () => {
    it('fires GET and populates the signal', () => {
      const data = [makeTx(), makeTx({ id: 'tx2', symbol: 'MSFT' })];
      service.loadByPortfolio('p1').subscribe();

      const req = httpMock.expectOne(txBase('p1'));
      expect(req.request.method).toBe('GET');
      req.flush(data);

      expect(service.transactions()).toEqual(data);
    });

    it('resets loading to false after success', () => {
      service.loadByPortfolio('p1').subscribe();
      httpMock.expectOne(txBase('p1')).flush([]);
      expect(service.loading()).toBe(false);
    });
  });

  describe('create()', () => {
    it('fires POST and prepends the new transaction', () => {
      service.loadByPortfolio('p1').subscribe();
      httpMock.expectOne(txBase('p1')).flush([makeTx({ id: 'tx0' })]);

      const newTx = makeTx({ id: 'tx1' });
      service
        .create('p1', {
          transaction_type: 'buy',
          symbol: 'AAPL',
          transaction_date: '2026-02-01',
          quantity: '10',
          price_per_share: '150.00',
          total_amount: '1500.00',
        })
        .subscribe();

      const req = httpMock.expectOne(txBase('p1'));
      expect(req.request.method).toBe('POST');
      req.flush(newTx);

      expect(service.transactions()[0].id).toBe('tx1');
      expect(service.transactions().length).toBe(2);
    });
  });

  describe('delete()', () => {
    it('fires DELETE and removes from signal', () => {
      const tx1 = makeTx({ id: 'tx1' });
      const tx2 = makeTx({ id: 'tx2', symbol: 'MSFT' });
      service.loadByPortfolio('p1').subscribe();
      httpMock.expectOne(txBase('p1')).flush([tx1, tx2]);

      service.delete('p1', 'tx1').subscribe();
      httpMock.expectOne(`${txBase('p1')}/tx1`).flush(null);

      expect(service.transactions().length).toBe(1);
      expect(service.transactions()[0].id).toBe('tx2');
    });
  });

  describe('holdings computed signal', () => {
    it('re-derives when transactions change', () => {
      service.loadByPortfolio('p1').subscribe();
      httpMock.expectOne(txBase('p1')).flush([
        makeTx({ quantity: '10', total_amount: '1500.00' }),
      ]);

      expect(service.holdings().length).toBe(1);
      expect(service.holdings()[0].symbol).toBe('AAPL');
    });
  });

  describe('clear()', () => {
    it('empties the transaction signal', () => {
      service.loadByPortfolio('p1').subscribe();
      httpMock.expectOne(txBase('p1')).flush([makeTx()]);

      service.clear();
      expect(service.transactions()).toEqual([]);
    });
  });
});

// ---------------------------------------------------------------------------
// deriveHoldings() — market data field initialisation
// ---------------------------------------------------------------------------

describe('deriveHoldings() market data fields', () => {
  it('initialises currentPrice, currentValue, gainLoss, gainLossPercent to null', () => {
    const holdings = deriveHoldings([
      makeTx({ quantity: '10', price_per_share: '150.00', total_amount: '1500.00' }),
    ]);
    expect(holdings[0].currentPrice).toBeNull();
    expect(holdings[0].currentValue).toBeNull();
    expect(holdings[0].gainLoss).toBeNull();
    expect(holdings[0].gainLossPercent).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// enrichHoldingsWithPrices() — gain/loss computation pure function tests
// ---------------------------------------------------------------------------

import { enrichHoldingsWithPrices } from './transaction.service';

describe('enrichHoldingsWithPrices()', () => {
  function makeHolding(
    symbol: string,
    quantity: string,
    totalCost: string,
  ) {
    return deriveHoldings([
      makeTx({ symbol, quantity, total_amount: totalCost }),
    ])[0];
  }

  it('returns holding with null market fields when no price is available', () => {
    const holding = makeHolding('AAPL', '10', '1500.00');
    const result = enrichHoldingsWithPrices([holding], {})[0];
    expect(result.currentPrice).toBeNull();
    expect(result.currentValue).toBeNull();
    expect(result.gainLoss).toBeNull();
    expect(result.gainLossPercent).toBeNull();
  });

  it('computes currentValue = quantity × currentPrice', () => {
    const holding = makeHolding('AAPL', '10', '1500.00');
    const result = enrichHoldingsWithPrices([holding], { AAPL: 175.0 })[0];
    expect(result.currentPrice).toBe(175.0);
    expect(result.currentValue).toBe('1750.00');
  });

  it('computes gainLoss = currentValue - totalCost', () => {
    const holding = makeHolding('AAPL', '10', '1500.00');
    const result = enrichHoldingsWithPrices([holding], { AAPL: 175.0 })[0];
    // 1750 - 1500 = 250
    expect(result.gainLoss).toBe('250.00');
  });

  it('computes gainLossPercent = gainLoss / totalCost × 100', () => {
    const holding = makeHolding('AAPL', '10', '1500.00');
    const result = enrichHoldingsWithPrices([holding], { AAPL: 175.0 })[0];
    // 250 / 1500 * 100 = 16.666...%
    expect(result.gainLossPercent).toBeCloseTo(16.67, 2);
  });

  it('handles a loss correctly (negative gainLoss and gainLossPercent)', () => {
    const holding = makeHolding('AAPL', '10', '1500.00');
    const result = enrichHoldingsWithPrices([holding], { AAPL: 120.0 })[0];
    // 1200 - 1500 = -300
    expect(result.gainLoss).toBe('-300.00');
    expect(result.gainLossPercent).toBeCloseTo(-20.0, 2);
  });

  it('handles zero totalCost gracefully (gainLossPercent is null)', () => {
    // Edge case: reinvested dividend resulted in zero cost basis scenario
    const holding = makeHolding('AAPL', '10', '0.00');
    const result = enrichHoldingsWithPrices([holding], { AAPL: 50.0 })[0];
    expect(result.gainLossPercent).toBeNull();
  });

  it('enriches multiple holdings independently', () => {
    const holdings = deriveHoldings([
      makeTx({ symbol: 'AAPL', quantity: '10', total_amount: '1500.00' }),
      makeTx({ symbol: 'MSFT', quantity: '5', total_amount: '1000.00' }),
    ]);
    const prices = { AAPL: 160.0, MSFT: 220.0 };
    const results = enrichHoldingsWithPrices(holdings, prices);

    const aapl = results.find((h) => h.symbol === 'AAPL')!;
    const msft = results.find((h) => h.symbol === 'MSFT')!;

    expect(aapl.currentValue).toBe('1600.00');
    expect(msft.currentValue).toBe('1100.00');
  });
});
