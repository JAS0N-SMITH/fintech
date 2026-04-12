/**
 * A full market data snapshot for a single ticker symbol.
 * Fetched via REST on component mount (snapshot-plus-deltas pattern, ADR 008).
 */
export interface Quote {
  symbol: string;
  price: number;
  day_high: number;
  day_low: number;
  open: number;
  previous_close: number;
  volume: number;
  timestamp: string; // ISO 8601
}

/**
 * A lightweight real-time price update pushed over WebSocket.
 * Merged into existing TickerState — does not replace the full snapshot.
 */
export interface PriceTick {
  symbol: string;
  price: number;
  volume: number;
  timestamp: string; // ISO 8601
}

/**
 * A single OHLCV candlestick for historical chart data.
 */
export interface Bar {
  symbol: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  timestamp: string; // ISO 8601
}

/** Candle resolution for historical bar requests. */
export type Timeframe = '1D' | '1W' | '1M' | '3M' | '1Y' | 'ALL';

/** WebSocket connection lifecycle state. */
export type ConnectionState = 'connected' | 'reconnecting' | 'disconnected';

/**
 * The merged state for a single ticker symbol.
 * Starts from a Quote snapshot; each PriceTick updates price and tracks high/low.
 */
export interface TickerState {
  symbol: string;
  /** Full snapshot loaded on subscribe. Null until the first REST fetch completes. */
  quote: Quote | null;
  /** Most recent price from a tick (or quote.price if no ticks yet). */
  currentPrice: number | null;
  /** Running day high — updated if a tick price exceeds the snapshot day_high. */
  dayHigh: number | null;
  /** Running day low — updated if a tick price falls below the snapshot day_low. */
  dayLow: number | null;
  /** Previous close from the initial quote snapshot — used for day gain/loss calculation. */
  previousClose: number | null;
  /** Timestamp of the most recent data update (snapshot or tick). Used to show staleness when disconnected. */
  lastUpdated: Date | null;
}
