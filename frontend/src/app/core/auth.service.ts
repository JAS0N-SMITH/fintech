import { inject, Injectable, OnDestroy, signal, computed } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import type { AuthError, Session, User } from '@supabase/supabase-js';
import { clearSessionCookie, COOKIE_REFRESH_SENTINEL, SUPABASE_CLIENT } from './supabase.token';
import { environment } from '../../environments/environment';

/** Represents the result of a sign-in or sign-up operation. */
export interface AuthResult {
  error: AuthError | null;
}

/** Shape of the Go GET /auth/session response body. */
export interface GoSessionResponse {
  access_token: string;
  expires_in: number;
  token_type: string;
  user: User;
}

/** Minimum seconds before expiry that the refresh timer will wait. */
const MIN_REFRESH_LEAD_SECONDS = 10;
/** Refresh the access token this many seconds before it expires. */
const REFRESH_LEAD_SECONDS = 60;
/** Seconds before JWT expiry at which the token is considered stale. */
const TOKEN_EXPIRY_BUFFER_SECONDS = 10;

/**
 * Extracts the exp claim from a JWT without verifying the signature.
 * Client-side only — do not use for security decisions; the Go middleware
 * performs real signature verification.
 * Returns null if the token is malformed or lacks an exp claim.
 */
function extractJwtExp(token: string): number | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'))) as unknown;
    if (typeof payload !== 'object' || payload === null) return null;
    const exp = (payload as Record<string, unknown>)['exp'];
    return typeof exp === 'number' ? exp : null;
  } catch {
    return null;
  }
}

/**
 * AuthService manages authentication state using Supabase Auth plus a Go
 * /auth/session proxy that owns the refresh token cookie.
 *
 * On cold start (page reload), the service calls Go GET /auth/session, which
 * exchanges the HTTP-only cookie for a fresh access token. The access token is
 * passed to supabase.auth.setSession() using a sentinel refresh_token — the
 * real refresh token never leaves the cookie.
 *
 * Because autoRefreshToken is disabled on the Supabase client, this service
 * schedules its own refresh timer that calls the Go proxy again shortly before
 * the access token expires.
 */
@Injectable({ providedIn: 'root' })
export class AuthService implements OnDestroy {
  private readonly supabase = inject(SUPABASE_CLIENT);
  private readonly router = inject(Router);
  private readonly http = inject(HttpClient);

  private readonly _session = signal<Session | null>(null);
  private readonly _isLoading = signal(true);

  /** The currently authenticated user, or null if not signed in. */
  readonly user = computed<User | null>(() => this._session()?.user ?? null);

  /**
   * The current JWT access token, or null if not signed in or within expiry buffer.
   * Returns null for tokens within TOKEN_EXPIRY_BUFFER_SECONDS of expiration,
   * triggering a 401 recovery or refresh cycle. This is a defense-in-depth check;
   * the Go middleware validates all token signatures and expiry independently.
   */
  readonly accessToken = computed<string | null>(() => {
    const token = this._session()?.access_token ?? null;
    if (!token) return null;
    const exp = extractJwtExp(token);
    if (exp !== null && Date.now() / 1000 >= exp - TOKEN_EXPIRY_BUFFER_SECONDS) {
      return null;
    }
    return token;
  });

  /** True when a user is signed in and the session is valid. */
  readonly isAuthenticated = computed(() => this._session() !== null);

  /** True while the initial session check is in flight. */
  readonly isLoading = this._isLoading.asReadonly();

  private authSubscription: { unsubscribe: () => void } | null = null;
  private refreshTimer: ReturnType<typeof setTimeout> | null = null;
  private restoreInFlight = true;
  private restorePromise: Promise<void> | null = null;

  private readonly TAB_HIDDEN_KEY = 'tab-hidden-at';
  /** Grace period in ms. Anything below this is a tab switch, not a tab close. */
  private readonly TAB_CLOSE_GRACE_MS = 30_000;
  private visibilityListener: (() => void) | null = null;

  constructor() {
    this.initAuthListener();
  }

  /**
   * Waits for the initial session restoration attempt to complete.
   * Call this in an APP_INITIALIZER to ensure the session is ready before
   * other initializers make authenticated requests.
   * Returns immediately if restore has already completed.
   */
  waitForRestore(): Promise<void> {
    // If restore is already done, return a resolved promise immediately
    if (!this.restoreInFlight) {
      return Promise.resolve();
    }

    // If we haven't started tracking the restore promise yet, create one.
    if (this.restorePromise === null) {
      this.restorePromise = new Promise((resolve) => {
        const checkInterval = setInterval(() => {
          if (!this.restoreInFlight) {
            clearInterval(checkInterval);
            resolve();
          }
        }, 50);
      });
    }
    return this.restorePromise;
  }

  /**
   * Signs in an existing user with email and password.
   * On success, Supabase emits a SIGNED_IN event which updates all signals and
   * the storage adapter POSTs the refresh token to the Go cookie proxy.
   */
  async signIn(email: string, password: string): Promise<AuthResult> {
    const { data, error } = await this.supabase.auth.signInWithPassword({ email, password });
    if (!error && data.session) {
      this.scheduleRefresh(data.session.expires_in ?? 900);
    }
    return { error };
  }

  /**
   * Registers a new user with email and password.
   * Supabase sends a confirmation email; session signals remain null until confirmed.
   */
  async signUp(email: string, password: string): Promise<AuthResult> {
    const { error } = await this.supabase.auth.signUp({ email, password });
    return { error };
  }

  /**
   * Signs out the current user and redirects to the login page.
   * Clears the refresh timer and session signals, and explicitly clears the
   * refresh-token cookie on the Go /auth/session endpoint.
   */
  async signOut(): Promise<void> {
    this.clearRefreshTimer();
    sessionStorage.removeItem(this.TAB_HIDDEN_KEY);
    await clearSessionCookie(this.http);
    await this.supabase.auth.signOut();
    await this.router.navigate(['/auth/login']);
  }

  /**
   * Applies a session obtained by the auth interceptor during 401 recovery.
   * Calls setSession() with the sentinel pattern and re-arms the refresh timer.
   * This is the only external write path for session state other than the
   * Supabase auth state change listener.
   */
  applyRefreshedSession(resp: GoSessionResponse): void {
    void this.supabase.auth.setSession({
      access_token: resp.access_token,
      refresh_token: COOKIE_REFRESH_SENTINEL,
    });
    this.scheduleRefresh(resp.expires_in);
  }

  ngOnDestroy(): void {
    this.authSubscription?.unsubscribe();
    this.clearRefreshTimer();
    if (this.visibilityListener) {
      document.removeEventListener('visibilitychange', this.visibilityListener);
    }
  }

  /**
   * Subscribes to Supabase auth state changes and kicks off cold-start restore.
   * The listener is attached first so the TOKEN_REFRESHED event fired by
   * setSession() during restore is not missed.
   */
  private initAuthListener(): void {
    const { data } = this.supabase.auth.onAuthStateChange((_event, session) => {
      this._session.set(session);
      // Avoid redirect races on reload: keep loading true until our explicit
      // cookie-backed restore attempt has completed.
      if (!this.restoreInFlight) {
        this._isLoading.set(false);
      }
    });
    this.authSubscription = data.subscription;

    this.tryRestoreSession();
    this.initVisibilityGuard();
  }

  /**
   * Detects when a tab becomes hidden and returns visible again after a grace
   * period, indicating the tab was closed and restored or switched away for a
   * long time. Signs out on detection.
   *
   * Uses sessionStorage because it is cleared when the tab closes (unlike
   * localStorage), not on browser exit. The grace period (30s) distinguishes
   * a quick tab switch from a prolonged absence or true tab close.
   */
  private initVisibilityGuard(): void {
    this.visibilityListener = () => {
      if (document.hidden) {
        sessionStorage.setItem(this.TAB_HIDDEN_KEY, String(Date.now()));
      } else {
        const hiddenAt = sessionStorage.getItem(this.TAB_HIDDEN_KEY);
        sessionStorage.removeItem(this.TAB_HIDDEN_KEY);

        if (
          hiddenAt !== null &&
          Date.now() - Number(hiddenAt) > this.TAB_CLOSE_GRACE_MS &&
          this.isAuthenticated()
        ) {
          void this.signOut();
        }
      }
    };
    document.addEventListener('visibilitychange', this.visibilityListener);
  }

  /**
   * Attempts to restore a session by calling GET /auth/session on the Go proxy.
   * Go exchanges the HTTP-only cookie for a fresh access token from Supabase
   * and rotates the cookie. We then hand the new access token to Supabase JS
   * via setSession() with a sentinel refresh_token so the real refresh token
   * stays in the cookie only.
   */
  private async tryRestoreSession(): Promise<void> {
    const url = `${environment.apiBaseUrl}/auth/session`;
    try {
      const goResp = await firstValueFrom(
        this.http.get<GoSessionResponse>(url, { withCredentials: true }),
      );

      const { error } = await this.supabase.auth.setSession({
        access_token: goResp.access_token,
        refresh_token: COOKIE_REFRESH_SENTINEL,
      });

      if (error) {
        this._session.set(null);
        this.restoreInFlight = false;
        this._isLoading.set(false);
        return;
      }

      this.scheduleRefresh(goResp.expires_in);
      this.restoreInFlight = false;
      this._isLoading.set(false);
    } catch (err) {
      if (err instanceof HttpErrorResponse) {
        this._session.set(null);
      }
      this.restoreInFlight = false;
      this._isLoading.set(false);
    }
  }

  /**
   * Schedules the next tryRestoreSession() call to run shortly before the
   * access token expires. Replaces any previously scheduled timer.
   */
  private scheduleRefresh(expiresInSeconds: number): void {
    this.clearRefreshTimer();
    const lead = Math.max(MIN_REFRESH_LEAD_SECONDS, expiresInSeconds - REFRESH_LEAD_SECONDS);
    this.refreshTimer = setTimeout(() => this.tryRestoreSession(), lead * 1000);
  }

  private clearRefreshTimer(): void {
    if (this.refreshTimer !== null) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }
}
