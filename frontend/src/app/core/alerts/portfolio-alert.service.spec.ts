import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { MessageService } from 'primeng/api';
import { PortfolioAlertService } from './portfolio-alert.service';
import { UserPreferencesService } from '../user-preferences.service';
import { TickerStateService } from '../ticker-state.service';
import { TransactionService } from '../../features/portfolio/services/transaction.service';
import type { PortfolioAlertThreshold, AlertPreferences } from './alert.model';
import type { Holding } from '../../features/portfolio/models/transaction.model';
import type { TickerState } from '../../features/portfolio/models/market-data.model';

// --- test helpers ---

function makeHolding(
  symbol: string,
  quantity = '100',
  avgCostBasis = '50',
  currentPrice: number | null = 50,
): Holding {
  const qty = parseFloat(quantity);
  const cost = parseFloat(avgCostBasis);
  const totalCost = (qty * cost).toFixed(2);
  const currentValue = currentPrice !== null ? (qty * currentPrice).toFixed(2) : null;
  const gainLossNum = currentValue !== null ? parseFloat(currentValue) - parseFloat(totalCost) : null;
  const gainLoss = gainLossNum !== null ? gainLossNum.toFixed(2) : null;
  const gainLossPercent =
    gainLossNum !== null && parseFloat(totalCost) !== 0 ? (gainLossNum / parseFloat(totalCost)) * 100 : null;

  return {
    symbol,
    quantity,
    avgCostBasis,
    totalCost,
    currentPrice,
    currentValue,
    gainLoss,
    gainLossPercent,
  };
}

function makeTickerState(symbol: string, currentPrice: number, previousClose: number): TickerState {
  return {
    symbol,
    currentPrice,
    dayHigh: currentPrice + 5,
    dayLow: currentPrice - 5,
    previousClose,
    lastUpdated: new Date(),
    quote: {
      symbol,
      price: currentPrice,
      day_high: currentPrice + 5,
      day_low: currentPrice - 5,
      open: currentPrice,
      previous_close: previousClose,
      volume: 1_000_000,
      timestamp: new Date().toISOString(),
    },
  };
}

function makeThreshold(
  id: string,
  type: 'portfolio_daily_change' | 'position_gain_loss',
  thresholdPercent: number,
  direction: 'above' | 'below' = 'below',
  symbol?: string,
): PortfolioAlertThreshold {
  return { id, type, thresholdPercent, direction, symbol, fired: false };
}

// --- mock factories ---

function makePreferencesMock() {
  return {
    preferences: signal<AlertPreferences>({ thresholds: [] }),
    load: vi.fn(),
    saveThresholds: vi.fn(),
  };
}

function makeTickerMock() {
  return {
    tickers: signal<Record<string, TickerState>>({}),
  };
}

function makeTransactionMock() {
  return {
    holdings: signal<Holding[]>([]),
  };
}

function makeMessageMock() {
  return {
    add: vi.fn(),
  };
}

// --- tests ---

describe('PortfolioAlertService', () => {
  let service: PortfolioAlertService;
  let prefsMock: ReturnType<typeof makePreferencesMock>;
  let tickerMock: ReturnType<typeof makeTickerMock>;
  let transactionMock: ReturnType<typeof makeTransactionMock>;
  let messageMock: ReturnType<typeof makeMessageMock>;

  function setup() {
    prefsMock = makePreferencesMock();
    tickerMock = makeTickerMock();
    transactionMock = makeTransactionMock();
    messageMock = makeMessageMock();

    TestBed.configureTestingModule({
      providers: [
        PortfolioAlertService,
        { provide: UserPreferencesService, useValue: prefsMock },
        { provide: TickerStateService, useValue: tickerMock },
        { provide: TransactionService, useValue: transactionMock },
        { provide: MessageService, useValue: messageMock },
      ],
    });

    service = TestBed.inject(PortfolioAlertService);
  }

  beforeEach(() => {
    vi.stubGlobal('Notification', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    TestBed.resetTestingModule();
  });

  describe('portfolio daily change alerts', () => {
    it('fires portfolio_daily_change alert when daily loss exceeds threshold', () => {
      setup();

      // Two holdings: AAPL and MSFT
      const holdings: Holding[] = [
        makeHolding('AAPL', '100', '150', 150), // break-even on daily basis
        makeHolding('MSFT', '50', '300', 280), // down to 280 from 300 (previous close)
      ];
      transactionMock.holdings.set(holdings);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 150, 150),
        MSFT: makeTickerState('MSFT', 280, 300), // previous close was 300
      });

      // Configure a threshold: alert if daily loss > 5%
      const threshold = makeThreshold('daily-5pct-loss', 'portfolio_daily_change', -5, 'below');
      prefsMock.preferences.set({ thresholds: [threshold] });
      TestBed.flushEffects();

      // Daily change: (AAPL 150*100 + MSFT 280*50) - (AAPL 150*100 + MSFT 300*50)
      //            = (15000 + 14000) - (15000 + 15000) = 29000 - 30000 = -1000
      // Percent: -1000 / 30000 * 100 = -3.33%
      // -3.33% > -5% so threshold NOT crossed yet (loss is less than 5%)

      expect(messageMock.add).not.toHaveBeenCalled();

      // Now make MSFT drop more: 280 -> 260 (from 300 previous)
      // Daily change: (15000 + 13000) - 30000 = -2000 / 30000 = -6.67%
      // -6.67% <= -5% so threshold IS crossed
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 150, 150),
        MSFT: makeTickerState('MSFT', 260, 300),
      });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledOnce();
      const call = messageMock.add.mock.calls[0][0];
      expect(call.severity).toBe('warn');
      expect(call.summary).toContain('Portfolio');
      expect(call.summary).toContain('Daily change');
    });

    it('does not fire when holdings are empty', () => {
      setup();

      transactionMock.holdings.set([]);
      tickerMock.tickers.set({});

      const threshold = makeThreshold('daily-5pct-loss', 'portfolio_daily_change', -5, 'below');
      prefsMock.preferences.set({ thresholds: [threshold] });
      TestBed.flushEffects();

      expect(messageMock.add).not.toHaveBeenCalled();
    });
  });

  describe('position gain/loss alerts', () => {
    it('fires position_gain_loss alert when holding exceeds threshold', () => {
      setup();

      // AAPL: 100 shares @ $50 cost basis, currently $52.50 = +5% gain
      const holdings: Holding[] = [makeHolding('AAPL', '100', '50', 52.5)];
      transactionMock.holdings.set(holdings);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 52.5, 52),
      });

      // Alert when position is up more than 3%
      const threshold = makeThreshold('aapl-gain-3pct', 'position_gain_loss', 3, 'above', 'AAPL');
      prefsMock.preferences.set({ thresholds: [threshold] });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledOnce();
      const call = messageMock.add.mock.calls[0][0];
      expect(call.summary).toContain('AAPL');
      expect(call.summary).toContain('gain');
    });

    it('fires position_gain_loss alert when holding loses more than threshold', () => {
      setup();

      // AAPL: 100 shares @ $50 cost basis, currently $47 = -6% loss
      const holdings: Holding[] = [makeHolding('AAPL', '100', '50', 47)];
      transactionMock.holdings.set(holdings);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 47, 47),
      });

      // Alert when position is down more than 5%
      const threshold = makeThreshold('aapl-loss-5pct', 'position_gain_loss', -5, 'below', 'AAPL');
      prefsMock.preferences.set({ thresholds: [threshold] });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledOnce();
    });

    it('does not fire position alert for non-matching symbols', () => {
      setup();

      const holdings: Holding[] = [makeHolding('AAPL', '100', '50', 52.5)];
      transactionMock.holdings.set(holdings);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 52.5, 52),
      });

      // Alert for MSFT (not held)
      const threshold = makeThreshold('msft-gain-3pct', 'position_gain_loss', 3, 'above', 'MSFT');
      prefsMock.preferences.set({ thresholds: [threshold] });
      TestBed.flushEffects();

      expect(messageMock.add).not.toHaveBeenCalled();
    });
  });

  describe('alert rule lifecycle', () => {
    it('resets fired after position recovers', () => {
      setup();

      // AAPL: 100 @ $50, currently $47 (-6%)
      const holdings: Holding[] = [makeHolding('AAPL', '100', '50', 47)];
      transactionMock.holdings.set(holdings);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 47, 47),
      });

      const threshold = makeThreshold('aapl-loss-5pct', 'position_gain_loss', -5, 'below', 'AAPL');
      prefsMock.preferences.set({ thresholds: [threshold] });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledOnce();

      // Position recovers to $52.50 (+5% gain)
      transactionMock.holdings.set([makeHolding('AAPL', '100', '50', 52.5)]);
      TestBed.flushEffects();

      // Should reset, but no new alert yet (not below -5% anymore)
      expect(messageMock.add).toHaveBeenCalledOnce();

      // Price drops again to $47
      transactionMock.holdings.set([makeHolding('AAPL', '100', '50', 47)]);
      TestBed.flushEffects();

      // Should fire again
      expect(messageMock.add).toHaveBeenCalledTimes(2);
    });

    it('ignores thresholds with no symbols set for position_gain_loss type', () => {
      setup();

      const holdings: Holding[] = [makeHolding('AAPL', '100', '50', 52.5)];
      transactionMock.holdings.set(holdings);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 52.5, 52),
      });

      // Threshold without symbol (invalid for position_gain_loss)
      const threshold = makeThreshold('invalid-threshold', 'position_gain_loss', 3, 'above');
      prefsMock.preferences.set({ thresholds: [threshold] });
      TestBed.flushEffects();

      expect(messageMock.add).not.toHaveBeenCalled();
    });
  });

  describe('multiple thresholds', () => {
    it('evaluates all configured thresholds independently', () => {
      setup();

      // AAPL: +5%, MSFT: -6%
      const holdings: Holding[] = [
        makeHolding('AAPL', '100', '50', 52.5),
        makeHolding('MSFT', '50', '300', 282),
      ];
      transactionMock.holdings.set(holdings);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 52.5, 52),
        MSFT: makeTickerState('MSFT', 282, 300),
      });

      const thresholds: PortfolioAlertThreshold[] = [
        makeThreshold('aapl-gain-3pct', 'position_gain_loss', 3, 'above', 'AAPL'),
        makeThreshold('msft-loss-5pct', 'position_gain_loss', -5, 'below', 'MSFT'),
      ];
      prefsMock.preferences.set({ thresholds });
      TestBed.flushEffects();

      // Both thresholds should fire
      expect(messageMock.add).toHaveBeenCalledTimes(2);
    });
  });
});
