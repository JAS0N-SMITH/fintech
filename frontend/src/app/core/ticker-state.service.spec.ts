import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { signal } from '@angular/core';
import { vi } from 'vitest';
import { of } from 'rxjs';
import { TickerStateService } from './ticker-state.service';
import { MarketDataService } from './market-data.service';
import { AuthService } from './auth.service';
import type { Quote, PriceTick } from '../features/portfolio/models/market-data.model';

// --- helpers ---

function makeQuote(symbol: string, price = 100, high = 110, low = 90): Quote {
  return {
    symbol,
    price,
    day_high: high,
    day_low: low,
    open: price - 1,
    previous_close: price - 2,
    volume: 1_000_000,
    timestamp: new Date().toISOString(),
  };
}

function makeTick(symbol: string, price: number): PriceTick {
  return { symbol, price, volume: 500, timestamp: new Date().toISOString() };
}

// --- mock factories ---

function makeMarketDataMock(defaultQuote = makeQuote('AAPL', 150, 155, 145)) {
  return { getQuote: vi.fn().mockReturnValue(of(defaultQuote)) };
}

function makeAuthMock(token: string | null = 'fake-token') {
  return { accessToken: signal<string | null>(token) };
}

describe('TickerStateService', () => {
  let service: TickerStateService;
  let marketMock: ReturnType<typeof makeMarketDataMock>;

  function setup(defaultQuote?: Quote) {
    marketMock = makeMarketDataMock(defaultQuote);
    const authMock = makeAuthMock();

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: MarketDataService, useValue: marketMock },
        { provide: AuthService, useValue: authMock },
      ],
    });
    service = TestBed.inject(TickerStateService);
  }

  afterEach(() => {
    service.destroy();
    TestBed.resetTestingModule();
  });

  // --- snapshot initialisation ---
  // of() is synchronous so no fakeAsync/tick needed

  it('initialises ticker state from quote snapshot on subscribe', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);

    const state = service.tickers()['AAPL'];
    expect(state).toBeDefined();
    expect(state.currentPrice).toBe(150);
    expect(state.dayHigh).toBe(155);
    expect(state.dayLow).toBe(145);
    expect(state.quote?.price).toBe(150);
  });

  it('sets state for multiple symbols independently', () => {
    setup();
    marketMock.getQuote.mockImplementation((sym: string) =>
      of(sym === 'AAPL' ? makeQuote('AAPL', 150, 155, 145) : makeQuote('MSFT', 300, 310, 290)),
    );

    service.subscribe(['AAPL', 'MSFT']);

    expect(service.tickers()['AAPL'].currentPrice).toBe(150);
    expect(service.tickers()['MSFT'].currentPrice).toBe(300);
  });

  // --- tick merging ---

  it('tick update replaces currentPrice without clobbering quote snapshot', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);

    service.applyTick(makeTick('AAPL', 152));

    const state = service.tickers()['AAPL'];
    expect(state.currentPrice).toBe(152);
    expect(state.quote?.price).toBe(150); // snapshot unchanged
    expect(state.dayHigh).toBe(155);      // 152 < 155, no new high
  });

  it('tick updates dayHigh when price exceeds snapshot day_high', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);

    service.applyTick(makeTick('AAPL', 160));

    expect(service.tickers()['AAPL'].dayHigh).toBe(160);
  });

  it('tick updates dayLow when price falls below snapshot day_low', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);

    service.applyTick(makeTick('AAPL', 140));

    expect(service.tickers()['AAPL'].dayLow).toBe(140);
  });

  it('tick for unknown symbol is ignored gracefully', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);

    expect(() => service.applyTick(makeTick('GOOG', 2800))).not.toThrow();
    expect(service.tickers()['GOOG']).toBeUndefined();
  });

  it('multiple ticks update price sequentially; high/low track correctly', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);

    service.applyTick(makeTick('AAPL', 151));
    service.applyTick(makeTick('AAPL', 153));
    service.applyTick(makeTick('AAPL', 148));

    const state = service.tickers()['AAPL'];
    expect(state.currentPrice).toBe(148); // last tick wins
    expect(state.dayHigh).toBe(155);      // 153 < 155, no new high
    expect(state.dayLow).toBe(145);       // 148 > 145, no new low
  });

  // --- connection state ---

  it('connection state starts as disconnected', () => {
    setup();
    expect(service.connectionState()).toBe('disconnected');
  });

  it('setConnectionState updates the connection state signal', () => {
    setup();
    service.setConnectionState('connected');
    expect(service.connectionState()).toBe('connected');

    service.setConnectionState('reconnecting');
    expect(service.connectionState()).toBe('reconnecting');
  });

  // --- unsubscribe ---

  it('unsubscribe removes symbols from tracked tickers', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);
    expect(service.tickers()['AAPL']).toBeDefined();

    service.unsubscribe(['AAPL']);

    expect(service.tickers()['AAPL']).toBeUndefined();
  });

  // --- resync on reconnect ---

  it('resync re-fetches snapshots for all tracked symbols and updates state', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);

    // Simulate reconnect with updated prices
    marketMock.getQuote.mockReturnValue(of(makeQuote('AAPL', 160, 165, 155)));
    service.resync();

    const state = service.tickers()['AAPL'];
    expect(state.currentPrice).toBe(160);
    expect(state.dayHigh).toBe(165);
    expect(state.dayLow).toBe(155);
  });

  // --- lastUpdated timestamp tracking ---

  it('lastUpdated is null before any snapshot or tick', () => {
    setup();
    // Create state without going through subscribe
    service['_tickers'].set({
      AAPL: {
        symbol: 'AAPL',
        quote: null,
        currentPrice: null,
        dayHigh: null,
        dayLow: null,
        previousClose: null,
        lastUpdated: null,
      },
    });

    expect(service.tickers()['AAPL'].lastUpdated).toBeNull();
  });

  it('lastUpdated is set when snapshot is applied via subscribe', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    const beforeTime = new Date();
    service.subscribe(['AAPL']);
    const afterTime = new Date();

    const state = service.tickers()['AAPL'];
    expect(state.lastUpdated).not.toBeNull();
    expect(state.lastUpdated!.getTime()).toBeGreaterThanOrEqual(beforeTime.getTime());
    expect(state.lastUpdated!.getTime()).toBeLessThanOrEqual(afterTime.getTime());
  });

  it('lastUpdated is updated when tick is applied', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);
    const initialLastUpdated = service.tickers()['AAPL'].lastUpdated;

    // Wait a small amount and apply tick
    vi.useFakeTimers();
    vi.setSystemTime(Date.now() + 100);

    const beforeTickTime = new Date();
    service.applyTick(makeTick('AAPL', 152));
    const afterTickTime = new Date();

    const state = service.tickers()['AAPL'];
    expect(state.lastUpdated).not.toBe(initialLastUpdated);
    expect(state.lastUpdated!.getTime()).toBeGreaterThanOrEqual(beforeTickTime.getTime());
    expect(state.lastUpdated!.getTime()).toBeLessThanOrEqual(afterTickTime.getTime());

    vi.useRealTimers();
  });

  it('lastUpdated is updated on resync', () => {
    setup(makeQuote('AAPL', 150, 155, 145));
    service.subscribe(['AAPL']);
    const initialLastUpdated = service.tickers()['AAPL'].lastUpdated;

    vi.useFakeTimers();
    vi.setSystemTime(Date.now() + 100);

    const beforeResyncTime = new Date();
    marketMock.getQuote.mockReturnValue(of(makeQuote('AAPL', 160, 165, 155)));
    service.resync();
    const afterResyncTime = new Date();

    const state = service.tickers()['AAPL'];
    expect(state.lastUpdated).not.toBe(initialLastUpdated);
    expect(state.lastUpdated!.getTime()).toBeGreaterThanOrEqual(beforeResyncTime.getTime());
    expect(state.lastUpdated!.getTime()).toBeLessThanOrEqual(afterResyncTime.getTime());

    vi.useRealTimers();
  });

  // --- connection state transitions ---

  it('connection state transitions: disconnected -> connected -> reconnecting -> connected', () => {
    setup();
    expect(service.connectionState()).toBe('disconnected');

    service.setConnectionState('connected');
    expect(service.connectionState()).toBe('connected');

    service.setConnectionState('reconnecting');
    expect(service.connectionState()).toBe('reconnecting');

    service.setConnectionState('connected');
    expect(service.connectionState()).toBe('connected');
  });

  it('destroy sets connection state to disconnected', () => {
    setup();
    service.setConnectionState('connected');
    expect(service.connectionState()).toBe('connected');

    service.destroy();

    expect(service.connectionState()).toBe('disconnected');
  });
});
