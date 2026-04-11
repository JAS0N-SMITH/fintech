import {
  inject,
  Injectable,
  OnDestroy,
  signal,
} from '@angular/core';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { Subscription, timer } from 'rxjs';
import { switchMap, retryWhen, tap, delayWhen } from 'rxjs/operators';
import { MarketDataService } from './market-data.service';
import { AuthService } from './auth.service';
import { environment } from '../../environments/environment';
import type {
  ConnectionState,
  PriceTick,
  Quote,
  TickerState,
} from '../features/portfolio/models/market-data.model';

/** Message sent from the Angular client to the Go WebSocket relay. */
interface WsClientMessage {
  action: 'subscribe' | 'unsubscribe';
  symbols: string[];
}

/** Message received from the Go WebSocket relay. */
interface WsServerMessage {
  type: 'tick' | 'error';
  data: PriceTick | string;
}

/** Maximum back-off delay between reconnection attempts (milliseconds). */
const MAX_BACKOFF_MS = 30_000;

/**
 * TickerStateService manages real-time market data state using the
 * snapshot-plus-deltas pattern (ADR 008).
 *
 * On subscribe:
 *   1. REST fetch → Quote snapshot for each symbol (via MarketDataService)
 *   2. WebSocket connection → continuous PriceTick stream
 *   3. Each tick updates currentPrice; tracks running day high/low
 *
 * On WebSocket reconnect:
 *   1. Re-fetch Quote snapshots for all tracked symbols
 *   2. Resume WebSocket subscription
 *
 * State is held in signals — components read via tickers() and connectionState().
 */
@Injectable({ providedIn: 'root' })
export class TickerStateService implements OnDestroy {
  private readonly marketData = inject(MarketDataService);
  private readonly auth = inject(AuthService);

  private readonly _tickers = signal<Record<string, TickerState>>({});
  private readonly _connectionState = signal<ConnectionState>('disconnected');

  /** Read-only map of symbol → TickerState signals. */
  readonly tickers = this._tickers.asReadonly();
  /** Current WebSocket connection lifecycle state. */
  readonly connectionState = this._connectionState.asReadonly();

  private ws: WebSocketSubject<WsServerMessage> | null = null;
  private wsSub: Subscription | null = null;
  private reconnectAttempt = 0;

  /** Set of currently tracked symbols (for resync on reconnect). */
  private readonly trackedSymbols = new Set<string>();

  /**
   * Subscribes to price updates for the given symbols.
   * Fetches Quote snapshots immediately, then connects (or reuses) the WebSocket.
   */
  subscribe(symbols: string[]): void {
    for (const sym of symbols) {
      this.trackedSymbols.add(sym);
    }
    this.fetchSnapshots(symbols);
    this.ensureWebSocket();
    this.sendSubscribe(symbols);
  }

  /**
   * Unsubscribes from price updates for the given symbols.
   * Removes the symbol state from the tickers signal.
   */
  unsubscribe(symbols: string[]): void {
    for (const sym of symbols) {
      this.trackedSymbols.delete(sym);
    }
    this.sendUnsubscribe(symbols);
    this._tickers.update((prev) => {
      const next = { ...prev };
      for (const sym of symbols) {
        delete next[sym];
      }
      return next;
    });
  }

  /**
   * Applies a single PriceTick to the matching TickerState.
   * Updates currentPrice and tracks running day high/low.
   * Ticks for unknown symbols are silently ignored.
   */
  applyTick(tick: PriceTick): void {
    const current = this._tickers()[tick.symbol];
    if (!current) return;

    this._tickers.update((prev) => ({
      ...prev,
      [tick.symbol]: {
        ...current,
        currentPrice: tick.price,
        dayHigh: current.dayHigh !== null && tick.price > current.dayHigh
          ? tick.price
          : current.dayHigh,
        dayLow: current.dayLow !== null && tick.price < current.dayLow
          ? tick.price
          : current.dayLow,
      },
    }));
  }

  /**
   * Re-fetches Quote snapshots for all tracked symbols.
   * Called after a WebSocket reconnection to ensure state is fresh
   * before ticks resume (snapshot-plus-deltas reconnect protocol).
   */
  resync(): void {
    if (this.trackedSymbols.size === 0) return;
    this.fetchSnapshots([...this.trackedSymbols]);
  }

  /** Updates the connection state signal. Exposed for WebSocket lifecycle management. */
  setConnectionState(state: ConnectionState): void {
    this._connectionState.set(state);
  }

  /** Closes the WebSocket and cleans up subscriptions. */
  destroy(): void {
    this.wsSub?.unsubscribe();
    this.ws?.complete();
    this.ws = null;
    this.wsSub = null;
    this._connectionState.set('disconnected');
  }

  ngOnDestroy(): void {
    this.destroy();
  }

  // --- private helpers ---

  private fetchSnapshots(symbols: string[]): void {
    for (const sym of symbols) {
      this.marketData.getQuote(sym).subscribe({
        next: (quote: Quote) => this.applySnapshot(quote),
        error: () => {
          // Snapshot failure is non-fatal — we keep the symbol tracked
          // and attempt resync on reconnect. No state update here.
        },
      });
    }
  }

  private applySnapshot(quote: Quote): void {
    this._tickers.update((prev) => ({
      ...prev,
      [quote.symbol]: {
        symbol: quote.symbol,
        quote,
        currentPrice: quote.price,
        dayHigh: quote.day_high,
        dayLow: quote.day_low,
      },
    }));
  }

  private ensureWebSocket(): void {
    if (this.ws && !this.ws.closed) return;

    const token = this.auth.accessToken();
    const wsUrl = `${environment.wsBaseUrl}/ws/prices${token ? `?token=${token}` : ''}`;

    this.ws = webSocket<WsServerMessage>({
      url: wsUrl,
      openObserver: {
        next: () => {
          this._connectionState.set('connected');
          this.reconnectAttempt = 0;
          // Re-subscribe to all tracked symbols after (re)connect
          if (this.trackedSymbols.size > 0) {
            this.sendSubscribe([...this.trackedSymbols]);
          }
        },
      },
      closeObserver: {
        next: () => {
          this._connectionState.set('reconnecting');
        },
      },
    });

    this.wsSub = this.ws
      .pipe(
        retryWhen((errors) =>
          errors.pipe(
            tap(() => {
              this._connectionState.set('reconnecting');
              this.reconnectAttempt++;
            }),
            delayWhen(() => {
              const delay = Math.min(1000 * 2 ** this.reconnectAttempt, MAX_BACKOFF_MS);
              return timer(delay);
            }),
            tap(() => {
              // Re-fetch snapshots before resuming tick stream on reconnect
              this.resync();
            }),
          ),
        ),
      )
      .subscribe({
        next: (msg: WsServerMessage) => {
          if (msg.type === 'tick') {
            this.applyTick(msg.data as PriceTick);
          }
        },
        error: () => {
          this._connectionState.set('disconnected');
        },
      });
  }

  private sendSubscribe(symbols: string[]): void {
    if (!this.ws || this.ws.closed || symbols.length === 0) return;
    const msg: WsClientMessage = { action: 'subscribe', symbols };
    this.ws.next(msg as unknown as WsServerMessage);
  }

  private sendUnsubscribe(symbols: string[]): void {
    if (!this.ws || this.ws.closed || symbols.length === 0) return;
    const msg: WsClientMessage = { action: 'unsubscribe', symbols };
    this.ws.next(msg as unknown as WsServerMessage);
  }
}
