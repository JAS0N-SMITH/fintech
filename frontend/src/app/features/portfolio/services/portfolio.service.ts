import { inject, Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap } from 'rxjs/operators';
import { Observable } from 'rxjs';
import { environment } from '../../../../environments/environment';
import type {
  Portfolio,
  CreatePortfolioInput,
  UpdatePortfolioInput,
} from '../models/portfolio.model';

const BASE = `${environment.apiBaseUrl}/portfolios`;

/**
 * PortfolioService manages portfolio CRUD operations against the Go API.
 *
 * State is held in signals so components react automatically without
 * subscribing manually. All mutations update the in-memory signal so the
 * UI stays consistent without requiring a full reload.
 */
@Injectable({ providedIn: 'root' })
export class PortfolioService {
  private readonly http = inject(HttpClient);

  private readonly _portfolios = signal<Portfolio[]>([]);
  private readonly _loading = signal(false);
  private readonly _selectedPortfolio = signal<Portfolio | null>(null);

  /** Current list of the authenticated user's portfolios. */
  readonly portfolios = this._portfolios.asReadonly();

  /** True while any HTTP request is in flight. */
  readonly loading = this._loading.asReadonly();

  /** The portfolio currently being viewed (set by loadById). */
  readonly selectedPortfolio = this._selectedPortfolio.asReadonly();

  /** Fetches all portfolios belonging to the authenticated user. */
  loadAll(): Observable<Portfolio[]> {
    this._loading.set(true);
    return this.http.get<Portfolio[]>(BASE).pipe(
      tap({
        next: (data) => {
          this._portfolios.set(data);
          this._loading.set(false);
        },
        error: () => this._loading.set(false),
      }),
    );
  }

  /** Fetches a single portfolio by ID and stores it in selectedPortfolio. */
  loadById(id: string): Observable<Portfolio> {
    this._loading.set(true);
    return this.http.get<Portfolio>(`${BASE}/${id}`).pipe(
      tap({
        next: (p) => {
          this._selectedPortfolio.set(p);
          this._loading.set(false);
        },
        error: () => this._loading.set(false),
      }),
    );
  }

  /** Creates a new portfolio and prepends it to the signal list. */
  create(input: CreatePortfolioInput): Observable<Portfolio> {
    return this.http.post<Portfolio>(BASE, input).pipe(
      tap((p) => this._portfolios.update((ps) => [p, ...ps])),
    );
  }

  /** Updates a portfolio in place within the signal list. */
  update(id: string, input: UpdatePortfolioInput): Observable<Portfolio> {
    return this.http.put<Portfolio>(`${BASE}/${id}`, input).pipe(
      tap((updated) => {
        this._portfolios.update((ps) =>
          ps.map((p) => (p.id === id ? updated : p)),
        );
        if (this._selectedPortfolio()?.id === id) {
          this._selectedPortfolio.set(updated);
        }
      }),
    );
  }

  /** Deletes a portfolio and removes it from the signal list. */
  delete(id: string): Observable<void> {
    return this.http.delete<void>(`${BASE}/${id}`).pipe(
      tap(() =>
        this._portfolios.update((ps) => ps.filter((p) => p.id !== id)),
      ),
    );
  }
}
