import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { MessageService } from 'primeng/api';
import { PriceAlertService } from './price-alert.service';
import { WatchlistService } from '../../features/watchlist/services/watchlist.service';
import { TickerStateService } from '../ticker-state.service';
import type { WatchlistItem } from '../../features/watchlist/models/watchlist.model';
import type { TickerState } from '../../features/portfolio/models/market-data.model';

// --- test helpers ---

function makeWatchlistItem(
  symbol: string,
  targetPrice: number,
  id = `item-${symbol}`,
): Omit<WatchlistItem, 'id' | 'watchlist_id' | 'created_at' | 'updated_at'> & {
  id: string;
  watchlist_id: string;
  created_at: string;
  updated_at: string;
  target_price: number;
} {
  return {
    id,
    watchlist_id: 'wl-1',
    symbol,
    target_price: targetPrice,
    notes: '',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  };
}

function makeTickerState(symbol: string, currentPrice: number, previousClose = currentPrice - 1): TickerState {
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

// --- mock factories ---

function makeWatchlistMock() {
  return {
    items: signal<WatchlistItem[]>([]),
  };
}

function makeTickerMock() {
  return {
    tickers: signal<Record<string, TickerState>>({}),
  };
}

function makeMessageMock() {
  return {
    add: vi.fn(),
  };
}

// --- tests ---

describe('PriceAlertService', () => {
  let service: PriceAlertService;
  let watchlistMock: ReturnType<typeof makeWatchlistMock>;
  let tickerMock: ReturnType<typeof makeTickerMock>;
  let messageMock: ReturnType<typeof makeMessageMock>;

  function setup() {
    watchlistMock = makeWatchlistMock();
    tickerMock = makeTickerMock();
    messageMock = makeMessageMock();

    TestBed.configureTestingModule({
      providers: [
        PriceAlertService,
        { provide: WatchlistService, useValue: watchlistMock },
        { provide: TickerStateService, useValue: tickerMock },
        { provide: MessageService, useValue: messageMock },
      ],
    });

    service = TestBed.inject(PriceAlertService);
  }

  beforeEach(() => {
    vi.stubGlobal('Notification', vi.fn());
    Object.defineProperty(document, 'visibilityState', {
      value: 'visible',
      configurable: true,
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    TestBed.resetTestingModule();
  });

  describe('threshold crossing detection', () => {
    it('fires when price rises to or above target (above direction)', () => {
      setup();

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();
      expect(messageMock.add).not.toHaveBeenCalled();

      // Price crosses above target
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledOnce();
      const call = messageMock.add.mock.calls[0][0];
      expect(call.severity).toBe('warn');
      expect(call.summary).toContain('AAPL');
      expect(call.detail).toContain('151');
      expect(call.detail).toContain('150');
    });

    it('does NOT re-fire on subsequent ticks already above target', () => {
      setup();

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      // Cross above
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();
      expect(messageMock.add).toHaveBeenCalledOnce();

      // Another tick above target
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 152) });
      TestBed.flushEffects();

      // Still only called once
      expect(messageMock.add).toHaveBeenCalledOnce();
    });

    it('resets fired after price falls back below target', () => {
      setup();

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      // Cross above
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();
      expect(messageMock.add).toHaveBeenCalledOnce();

      // Price falls back below target
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      // fired state should reset (but no new alert yet)
      expect(messageMock.add).toHaveBeenCalledOnce();
    });

    it('fires again after reset when price recrosses', () => {
      setup();

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      // First cross above
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();
      expect(messageMock.add).toHaveBeenCalledOnce();

      // Price falls back below target
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      // Price rises above target again
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 152) });
      TestBed.flushEffects();

      // Should fire again
      expect(messageMock.add).toHaveBeenCalledTimes(2);
    });

    it('fires when price falls to or below target (below direction)', () => {
      setup();

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();
      expect(messageMock.add).not.toHaveBeenCalled();

      // Price falls below target
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledOnce();
      const call = messageMock.add.mock.calls[0][0];
      expect(call.summary).toContain('AAPL');
      expect(call.detail).toContain('149');
    });

    it('does not fire when currentPrice is null', () => {
      setup();

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({});
      TestBed.flushEffects();

      expect(messageMock.add).not.toHaveBeenCalled();
    });

    it('handles multiple symbols independently', () => {
      setup();

      const item1 = makeWatchlistItem('AAPL', 150, 'item-1');
      const item2 = makeWatchlistItem('MSFT', 300, 'item-2');
      watchlistMock.items.set([item1, item2]);
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 149),
        MSFT: makeTickerState('MSFT', 299),
      });
      TestBed.flushEffects();
      expect(messageMock.add).not.toHaveBeenCalled();

      // Only AAPL crosses above
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 151),
        MSFT: makeTickerState('MSFT', 299),
      });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledOnce();
      expect(messageMock.add.mock.calls[0][0].detail).toContain('AAPL');

      // Then MSFT crosses
      tickerMock.tickers.set({
        AAPL: makeTickerState('AAPL', 151),
        MSFT: makeTickerState('MSFT', 301),
      });
      TestBed.flushEffects();

      expect(messageMock.add).toHaveBeenCalledTimes(2);
      expect(messageMock.add.mock.calls[1][0].detail).toContain('MSFT');
    });

    it('ignores items without target_price set', () => {
      setup();

      const itemWithoutTarget: WatchlistItem = {
        id: 'item-1',
        watchlist_id: 'wl-1',
        symbol: 'AAPL',
        target_price: undefined,
        notes: '',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      watchlistMock.items.set([itemWithoutTarget]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 100) });
      TestBed.flushEffects();

      expect(messageMock.add).not.toHaveBeenCalled();
    });
  });

  describe('browser notification delivery', () => {
    it('fires browser Notification when tab is hidden and price crosses', () => {
      setup();
      Object.defineProperty(document, 'visibilityState', {
        value: 'hidden',
        configurable: true,
      });

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      // Manually grant permission (in real flow, user clicks a button)
      service.setNotificationPermission('granted');

      // Cross above
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();

      // Browser Notification constructor should have been called
      const NotificationConstructor = window.Notification as any;
      expect(NotificationConstructor).toHaveBeenCalled();
    });

    it('does not fire browser Notification when tab is visible', () => {
      setup();
      // visibilityState is 'visible' by default in beforeEach

      const item = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      service.setNotificationPermission('granted');

      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();

      // Browser Notification should NOT have been called
      const NotificationConstructor = window.Notification as any;
      expect(NotificationConstructor).not.toHaveBeenCalled();
    });
  });

  describe('alert rule lifecycle', () => {
    it('survives watchlist reload and preserves fired state', () => {
      setup();

      const item1 = makeWatchlistItem('AAPL', 150, 'item-1');
      watchlistMock.items.set([item1]);
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 149) });
      TestBed.flushEffects();

      // Cross above
      tickerMock.tickers.set({ AAPL: makeTickerState('AAPL', 151) });
      TestBed.flushEffects();
      expect(messageMock.add).toHaveBeenCalledOnce();

      // Simulate watchlist reload with same item
      watchlistMock.items.set([item1]);
      TestBed.flushEffects();

      // Price still above, should NOT re-fire
      expect(messageMock.add).toHaveBeenCalledOnce();
    });
  });
});
