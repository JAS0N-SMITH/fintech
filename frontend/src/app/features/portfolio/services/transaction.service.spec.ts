import { TestBed } from '@angular/core/testing';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { TransactionService, deriveHoldings } from './transaction.service';
import type { Transaction } from '../models/transaction.model';
import { environment } from '../../../../environments/environment';

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

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
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
