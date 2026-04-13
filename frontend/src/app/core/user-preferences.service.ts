import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { tap, map } from 'rxjs/operators';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { AlertPreferences, PortfolioAlertThreshold } from './alerts/alert.model';

const preferencesBase = `${environment.apiBaseUrl}/me`;

/**
 * Type guard to check if a value is a valid AlertPreferences object.
 */
function isAlertPreferences(value: unknown): value is AlertPreferences {
  if (typeof value !== 'object' || value === null) return false;
  const obj = value as Record<string, unknown>;
  return Array.isArray(obj['thresholds']);
}

/**
 * Shape of the API response from GET /api/v1/me.
 */
interface UserProfile {
  preferences: Record<string, unknown>;
}

/**
 * Shape of the PATCH request body for updating preferences.
 */
interface UpdatePreferencesPayload {
  alert_thresholds: PortfolioAlertThreshold[];
}

/**
 * UserPreferencesService manages user preferences stored in the profiles.preferences JSONB column.
 *
 * Preferences are lazily loaded via load() — not fetched on service construction.
 * The service abstracts the backend API shape so alert services only deal with AlertPreferences.
 */
@Injectable({ providedIn: 'root' })
export class UserPreferencesService {
  private readonly http = inject(HttpClient);

  private readonly _preferences = signal<AlertPreferences>({ thresholds: [] });

  /** Read-only access to current preferences signal. */
  readonly preferences = this._preferences.asReadonly();

  /**
   * Load preferences from the backend via GET /api/v1/me.
   * Updates the preferences signal and returns the loaded data as an Observable.
   *
   * @returns Observable<AlertPreferences>
   */
  load(): Observable<AlertPreferences> {
    return this.http.get<UserProfile>(preferencesBase).pipe(
      tap((profile) => {
        // Extract alert_thresholds from the preferences JSONB,
        // with a type guard to ensure safety.
        const alertThresholds = profile.preferences['alert_thresholds'];
        if (isAlertPreferences({ thresholds: alertThresholds })) {
          this._preferences.set({ thresholds: alertThresholds as PortfolioAlertThreshold[] });
        }
      }),
      map(() => this._preferences()),
    );
  }

  /**
   * Save alert thresholds back to the backend via PATCH /api/v1/me/preferences.
   * Also updates the local preferences signal.
   *
   * @param thresholds The new list of alert thresholds to save
   * @returns Observable<void>
   */
  saveThresholds(thresholds: PortfolioAlertThreshold[]): Observable<void> {
    const payload: UpdatePreferencesPayload = { alert_thresholds: thresholds };
    return this.http.patch<void>(`${preferencesBase}/preferences`, payload).pipe(
      tap(() => {
        this._preferences.set({ thresholds });
      }),
    );
  }
}
