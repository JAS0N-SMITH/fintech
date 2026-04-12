import { TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { vi } from 'vitest';
import { signal } from '@angular/core';
import { TickerDetailComponent } from './ticker-detail.component';
import { MarketDataService } from '../../../../core/market-data.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import { ThemeService } from '../../../../core/theme.service';
import { TransactionService } from '../../../portfolio/services/transaction.service';
import { PortfolioService } from '../../../portfolio/services/portfolio.service';
import { computeHoldingPeriod } from './ticker-detail.utils';
import type { Transaction, Holding } from '../../../portfolio/models/transaction.model';
import type { Quote, Bar, Timeframe } from '../../../portfolio/models/market-data.model';
import { of } from 'rxjs';

// Helper: create test transaction
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

// Helper: create test quote
function makeQuote(overrides: Partial<Quote> = {}): Quote {
  return {
    symbol: 'AAPL',
    price: 175.0,
    day_high: 178.5,
    day_low: 172.0,
    open: 173.0,
    previous_close: 172.5,
    volume: 52000000,
    timestamp: '2026-04-11T16:00:00Z',
    ...overrides,
  };
}

// Helper: create test bar
function makeBar(overrides: Partial<Bar> = {}): Bar {
  return {
    symbol: 'AAPL',
    open: 150.0,
    high: 155.0,
    low: 148.5,
    close: 154.0,
    volume: 5000000,
    timestamp: '2026-04-11T16:00:00Z',
    ...overrides,
  };
}

// Helper: create stub services
function makeMarketDataServiceStub() {
  return {
    getQuote: vi.fn().mockReturnValue(of(makeQuote())),
    getHistoricalBars: vi.fn().mockReturnValue(of([makeBar()])),
    getQuotesBatch: vi.fn().mockReturnValue(of([])),
  };
}

function makeTickerStateServiceStub() {
  return {
    tickers: signal<Record<string, any>>({}).asReadonly(),
    connectionState: signal<'connected' | 'reconnecting' | 'disconnected'>('disconnected').asReadonly(),
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
  };
}

function makeThemeServiceStub() {
  return {
    isDark: signal(false).asReadonly(),
    toggle: vi.fn(),
  };
}

function makeTransactionServiceStub() {
  return {
    transactions: signal<Transaction[]>([]).asReadonly(),
    loading: signal(false).asReadonly(),
    holdings: signal<Holding[]>([]).asReadonly(),
    loadByPortfolio: vi.fn().mockReturnValue(of([])),
    create: vi.fn(),
    delete: vi.fn(),
    clear: vi.fn(),
  };
}

function makePortfolioServiceStub() {
  return {
    portfolios: signal<any[]>([]).asReadonly(),
    loading: signal(false).asReadonly(),
    selectedPortfolio: signal<any | null>(null).asReadonly(),
    loadAll: vi.fn().mockReturnValue(of([])),
    loadById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  };
}

// ===========================================================================
// Pure function tests — computeHoldingPeriod()
// ===========================================================================

describe('computeHoldingPeriod()', () => {
  it('returns null for empty transaction list', () => {
    const result = computeHoldingPeriod([]);
    expect(result).toBeNull();
  });

  it('returns the earliest transaction_date when given one transaction', () => {
    const txs = [makeTx({ transaction_date: '2026-01-15' })];
    const result = computeHoldingPeriod(txs);
    expect(result).toBe('2026-01-15');
  });

  it('returns the earliest transaction_date among multiple transactions', () => {
    const txs = [
      makeTx({ transaction_date: '2026-01-15', id: 'tx1' }),
      makeTx({ transaction_date: '2025-06-01', id: 'tx2' }),
      makeTx({ transaction_date: '2026-12-31', id: 'tx3' }),
    ];
    const result = computeHoldingPeriod(txs);
    expect(result).toBe('2025-06-01');
  });

  it('handles ISO date comparison correctly (string sorting)', () => {
    const txs = [
      makeTx({ transaction_date: '2026-02-15', id: 'tx1' }),
      makeTx({ transaction_date: '2026-01-01', id: 'tx2' }),
      makeTx({ transaction_date: '2026-01-20', id: 'tx3' }),
    ];
    const result = computeHoldingPeriod(txs);
    expect(result).toBe('2026-01-01');
  });
});

// ===========================================================================
// Timeframe → API parameter mapping tests
// ===========================================================================

describe('TickerDetailComponent — Timeframe API parameter mapping', () => {
  let component: TickerDetailComponent;
  let marketDataStub: ReturnType<typeof makeMarketDataServiceStub>;

  beforeEach(() => {
    marketDataStub = makeMarketDataServiceStub();
    TestBed.configureTestingModule({
      imports: [TickerDetailComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: { get: () => 'AAPL' } } } },
        { provide: MarketDataService, useValue: marketDataStub },
        { provide: TickerStateService, useValue: makeTickerStateServiceStub() },
        { provide: ThemeService, useValue: makeThemeServiceStub() },
        { provide: TransactionService, useValue: makeTransactionServiceStub() },
        { provide: PortfolioService, useValue: makePortfolioServiceStub() },
      ],
    });
    component = TestBed.createComponent(TickerDetailComponent).componentInstance;
  });

  it('default timeframe 1M is passed to getHistoricalBars', () => {
    expect(component.selectedTimeframe()).toBe('1M');
    component.ngAfterViewInit();
    // Trigger the effect
    component.selectedTimeframe.set('1M');

    // Small delay to allow the effect to run
    vi.runAllTimersAsync();
    expect(marketDataStub.getHistoricalBars).toHaveBeenCalledWith('AAPL', '1M');
  });

  it.each([
    ['1D'],
    ['1W'],
    ['1M'],
    ['3M'],
    ['1Y'],
    ['ALL'],
  ] as [Timeframe][])('selectTimeframe("%s") passes it to getHistoricalBars', (tf) => {
    component.ngAfterViewInit();
    component.selectTimeframe(tf);

    // Allow effect to run
    vi.runAllTimersAsync();

    // Check that the call was made with the exact timeframe
    const calls = (marketDataStub.getHistoricalBars as any).mock.calls;
    const lastCall = calls[calls.length - 1];
    expect(lastCall[1]).toBe(tf);
  });
});

// ===========================================================================
// Position summary derivation tests
// ===========================================================================

describe('TickerDetailComponent — Position summary derivation', () => {
  let component: TickerDetailComponent;
  let transactionServiceStub: ReturnType<typeof makeTransactionServiceStub>;

  beforeEach(() => {
    transactionServiceStub = makeTransactionServiceStub();
    TestBed.configureTestingModule({
      imports: [TickerDetailComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: { get: () => 'AAPL' } } } },
        { provide: MarketDataService, useValue: makeMarketDataServiceStub() },
        { provide: TickerStateService, useValue: makeTickerStateServiceStub() },
        { provide: ThemeService, useValue: makeThemeServiceStub() },
        { provide: TransactionService, useValue: transactionServiceStub },
        { provide: PortfolioService, useValue: makePortfolioServiceStub() },
      ],
    });
    component = TestBed.createComponent(TickerDetailComponent).componentInstance;
  });

  it('symbolTransactions() filters to symbol only', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([
      makeTx({ symbol: 'AAPL', id: 'tx1' }),
      makeTx({ symbol: 'MSFT', id: 'tx2' }),
      makeTx({ symbol: 'AAPL', id: 'tx3' }),
    ]);

    const filtered = component.symbolTransactions();
    expect(filtered).toHaveLength(2);
    expect(filtered.every((tx) => tx.symbol === 'AAPL')).toBe(true);
  });

  it('symbolHolding derives from filtered transactions using deriveHoldings', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([
      makeTx({ symbol: 'AAPL', quantity: '10', price_per_share: '100.00', total_amount: '1000.00' }),
    ]);

    const holding = component.symbolHolding();
    expect(holding).not.toBeNull();
    expect(holding!.symbol).toBe('AAPL');
    expect(holding!.quantity).toBe('10');
    expect(holding!.avgCostBasis).toBe('100.00');
    expect(holding!.totalCost).toBe('1000.00');
  });

  it('symbolHolding returns null for empty transactions', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([]);

    const holding = component.symbolHolding();
    expect(holding).toBeNull();
  });

  it('buy + sell quantity netting works correctly', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([
      makeTx({ id: 'tx1', quantity: '100', price_per_share: '50.00', total_amount: '5000.00' }),
      makeTx({ id: 'tx2', transaction_type: 'sell', quantity: '30', total_amount: '1560.00' }),
    ]);

    const holding = component.symbolHolding();
    expect(holding!.quantity).toBe('70');
  });
});

// ===========================================================================
// Gain/loss calculation tests
// ===========================================================================

describe('TickerDetailComponent — Gain/loss calculation', () => {
  let component: TickerDetailComponent;
  let tickersSignal: ReturnType<typeof signal>;

  beforeEach(() => {
    // Create a writable signal for testing
    tickersSignal = signal<Record<string, any>>({});
    const tickerStateStub = {
      ...makeTickerStateServiceStub(),
      tickers: tickersSignal.asReadonly(),
    };

    TestBed.configureTestingModule({
      imports: [TickerDetailComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: { get: () => 'AAPL' } } } },
        { provide: MarketDataService, useValue: makeMarketDataServiceStub() },
        { provide: TickerStateService, useValue: tickerStateStub },
        { provide: ThemeService, useValue: makeThemeServiceStub() },
        { provide: TransactionService, useValue: makeTransactionServiceStub() },
        { provide: PortfolioService, useValue: makePortfolioServiceStub() },
      ],
    });
    component = TestBed.createComponent(TickerDetailComponent).componentInstance;
  });

  it('10 shares @ $100 cost, $120 price = $200 gain, 20%', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([
      makeTx({ quantity: '10', price_per_share: '100.00', total_amount: '1000.00' }),
    ]);

    // Set live price using the writable signal
    tickersSignal.set({
      AAPL: { currentPrice: 120, quote: null, dayHigh: 120, dayLow: 100, previousClose: 115 },
    });

    const holding = component.symbolHolding();
    expect(holding).not.toBeNull();
    expect(holding!.gainLoss).toBe('200.00');
    expect(holding!.gainLossPercent).toBeCloseTo(20.0, 1);
  });

  it('10 shares @ $100 cost, $80 price = -$200 loss, -20%', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([
      makeTx({ quantity: '10', price_per_share: '100.00', total_amount: '1000.00' }),
    ]);

    tickersSignal.set({
      AAPL: { currentPrice: 80, quote: null, dayHigh: 100, dayLow: 80, previousClose: 90 },
    });

    const holding = component.symbolHolding();
    expect(holding).not.toBeNull();
    expect(holding!.gainLoss).toBe('-200.00');
    expect(holding!.gainLossPercent).toBeCloseTo(-20.0, 1);
  });

  it('no price available = null gain/loss', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([
      makeTx({ quantity: '10', price_per_share: '100.00', total_amount: '1000.00' }),
    ]);

    // No tickers set
    tickersSignal.set({});

    const holding = component.symbolHolding();
    expect(holding).not.toBeNull();
    expect(holding!.gainLoss).toBeNull();
    expect(holding!.gainLossPercent).toBeNull();
  });
});

// ===========================================================================
// Holding period calculation tests
// ===========================================================================

describe('TickerDetailComponent — Holding period', () => {
  let component: TickerDetailComponent;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [TickerDetailComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: ActivatedRoute, useValue: { snapshot: { paramMap: { get: () => 'AAPL' } } } },
        { provide: MarketDataService, useValue: makeMarketDataServiceStub() },
        { provide: TickerStateService, useValue: makeTickerStateServiceStub() },
        { provide: ThemeService, useValue: makeThemeServiceStub() },
        { provide: TransactionService, useValue: makeTransactionServiceStub() },
        { provide: PortfolioService, useValue: makePortfolioServiceStub() },
      ],
    });
    component = TestBed.createComponent(TickerDetailComponent).componentInstance;
  });

  it('returns null when no transactions', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([]);

    const period = component.holdingPeriod();
    expect(period).toBeNull();
  });

  it('returns earliest transaction date', () => {
    component.symbol.set('AAPL');
    component.allTransactions.set([
      makeTx({ symbol: 'AAPL', transaction_date: '2026-01-15', id: 'tx1' }),
      makeTx({ symbol: 'AAPL', transaction_date: '2025-06-01', id: 'tx2' }),
    ]);

    const period = component.holdingPeriod();
    expect(period).toBe('2025-06-01');
  });
});
