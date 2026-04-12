import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { signal } from '@angular/core';

import { environment } from '../../../../environments/environment';
import { Watchlist, WatchlistItem, CreateWatchlistInput, UpdateWatchlistInput, CreateWatchlistItemInput, UpdateWatchlistItemInput } from '../models/watchlist.model';
import { TickerStateService } from '../../../core/ticker-state.service';

@Injectable({ providedIn: 'root' })
export class WatchlistService {
  private http = inject(HttpClient);
  private tickerStateService = inject(TickerStateService);

  private _watchlists = signal<Watchlist[]>([]);
  private _selectedWatchlist = signal<Watchlist | null>(null);
  private _items = signal<WatchlistItem[]>([]);
  private _loading = signal(false);

  readonly watchlists = this._watchlists.asReadonly();
  readonly selectedWatchlist = this._selectedWatchlist.asReadonly();
  readonly items = this._items.asReadonly();
  readonly loading = this._loading.asReadonly();

  private baseUrl = `${environment.apiBaseUrl}/watchlists`;

  /**
   * Load all watchlists for the authenticated user.
   */
  loadAll(): Observable<Watchlist[]> {
    this._loading.set(true);
    return this.http.get<Watchlist[]>(this.baseUrl).pipe(
      tap((ws) => {
        this._watchlists.set(ws || []);
        this._loading.set(false);
      }),
      tap(() => this._loading.set(false))
    );
  }

  /**
   * Load a single watchlist by ID and its items.
   */
  loadById(id: string): Observable<Watchlist> {
    this._loading.set(true);
    return this.http.get<Watchlist>(`${this.baseUrl}/${id}`).pipe(
      tap((w) => {
        this._selectedWatchlist.set(w);
        this._loading.set(false);
        // Load items for this watchlist
        this.loadItems(id).subscribe();
      }),
      tap(() => this._loading.set(false))
    );
  }

  /**
   * Create a new watchlist.
   */
  create(input: CreateWatchlistInput): Observable<Watchlist> {
    return this.http.post<Watchlist>(this.baseUrl, input).pipe(
      tap((w) => {
        this._watchlists.update((ws) => [w, ...ws]);
      })
    );
  }

  /**
   * Update a watchlist's name.
   */
  update(id: string, input: UpdateWatchlistInput): Observable<Watchlist> {
    return this.http.put<Watchlist>(`${this.baseUrl}/${id}`, input).pipe(
      tap((updated) => {
        this._watchlists.update((ws) =>
          ws.map((w) => (w.id === id ? updated : w))
        );
        if (this._selectedWatchlist()?.id === id) {
          this._selectedWatchlist.set(updated);
        }
      })
    );
  }

  /**
   * Delete a watchlist.
   */
  delete(id: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${id}`).pipe(
      tap(() => {
        this._watchlists.update((ws) => ws.filter((w) => w.id !== id));
        if (this._selectedWatchlist()?.id === id) {
          this._selectedWatchlist.set(null);
        }
        // Unsubscribe from WebSocket for all items in this watchlist
        this._items().forEach((item) => {
          this.tickerStateService.unsubscribe([item.symbol]);
        });
        this._items.set([]);
      })
    );
  }

  /**
   * Load all items in a watchlist and subscribe to their prices via WebSocket.
   */
  loadItems(watchlistId: string): Observable<WatchlistItem[]> {
    return this.http.get<WatchlistItem[]>(`${this.baseUrl}/${watchlistId}/items`).pipe(
      tap((items) => {
        this._items.set(items || []);
        // Subscribe to price updates for all symbols in this watchlist
        if (items && items.length > 0) {
          const symbols = items.map((item) => item.symbol);
          this.tickerStateService.subscribe(symbols);
        }
      })
    );
  }

  /**
   * Add a ticker symbol to a watchlist.
   */
  addItem(watchlistId: string, input: CreateWatchlistItemInput): Observable<WatchlistItem> {
    return this.http
      .post<WatchlistItem>(`${this.baseUrl}/${watchlistId}/items`, input)
      .pipe(
        tap((item) => {
          this._items.update((items) => [...items, item]);
          // Subscribe to price updates for the new symbol
          this.tickerStateService.subscribe([item.symbol]);
        })
      );
  }

  /**
   * Update a watchlist item's target price and notes.
   */
  updateItem(watchlistId: string, symbol: string, input: UpdateWatchlistItemInput): Observable<WatchlistItem> {
    return this.http
      .put<WatchlistItem>(`${this.baseUrl}/${watchlistId}/items/${symbol}`, input)
      .pipe(
        tap((updated) => {
          this._items.update((items) =>
            items.map((item) => (item.symbol === symbol ? updated : item))
          );
        })
      );
  }

  /**
   * Remove a ticker symbol from a watchlist.
   */
  removeItem(watchlistId: string, symbol: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/${watchlistId}/items/${symbol}`).pipe(
      tap(() => {
        this._items.update((items) => items.filter((item) => item.symbol !== symbol));
        // Unsubscribe from price updates for this symbol
        this.tickerStateService.unsubscribe([symbol]);
      })
    );
  }

  /**
   * Clean up subscriptions (called on component destroy).
   */
  cleanup(): void {
    // Unsubscribe from all symbols in the current watchlist
    this._items().forEach((item) => {
      this.tickerStateService.unsubscribe([item.symbol]);
    });
  }
}
