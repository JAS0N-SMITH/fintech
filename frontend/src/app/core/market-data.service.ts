import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import type { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Bar, Quote, Timeframe, StockSymbol } from '../features/portfolio/models/market-data.model';

const BASE = `${environment.apiBaseUrl}`;

/**
 * MarketDataService fetches market snapshots and historical bar data
 * from the Go API REST endpoints.
 *
 * Methods return Observables (RxJS for HTTP, per angular.md rules).
 * Real-time price updates are handled by TickerStateService via WebSocket.
 */
@Injectable({ providedIn: 'root' })
export class MarketDataService {
  private readonly http = inject(HttpClient);

  /**
   * Fetches a full quote snapshot for a single ticker symbol.
   * The Go API serves this from a short-lived cache (10s TTL).
   */
  getQuote(symbol: string): Observable<Quote> {
    return this.http.get<Quote>(`${BASE}/quotes/${encodeURIComponent(symbol)}`);
  }

  /**
   * Fetches quotes for multiple ticker symbols in a single request.
   * Returns a map of symbol → Quote.
   */
  getQuotesBatch(symbols: string[]): Observable<Record<string, Quote>> {
    const params = new HttpParams().set('symbols', symbols.join(','));
    return this.http.get<Record<string, Quote>>(`${BASE}/quotes`, { params });
  }

  /**
   * Fetches historical OHLCV bar data for a symbol.
   *
   * @param symbol   Ticker symbol (e.g. 'AAPL')
   * @param timeframe Candle resolution (default '1M')
   * @param start    Start of date range (ISO 8601)
   * @param end      End of date range (ISO 8601)
   */
  getHistoricalBars(
    symbol: string,
    timeframe: Timeframe = '1M',
    start?: string,
    end?: string,
  ): Observable<Bar[]> {
    let params = new HttpParams().set('timeframe', timeframe);
    if (start) params = params.set('start', start);
    if (end) params = params.set('end', end);
    return this.http.get<Bar[]>(`${BASE}/bars/${encodeURIComponent(symbol)}`, { params });
  }

  /**
   * Searches for stock symbols by query string.
   * Filters by case-insensitive prefix match on symbol and substring match on description.
   *
   * @param query Search term (optional)
   * @param limit Maximum number of results (1-50, default 20)
   */
  searchSymbols(query: string = '', limit: number = 20): Observable<StockSymbol[]> {
    let params = new HttpParams();
    if (query) params = params.set('q', query);
    if (limit && limit !== 20) params = params.set('limit', limit.toString());
    return this.http.get<StockSymbol[]>(`${BASE}/symbols`, { params });
  }
}
