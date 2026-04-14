import { InjectionToken, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { createClient, SupabaseClient, SupportedStorage } from '@supabase/supabase-js';
import { firstValueFrom } from 'rxjs';
import { environment } from '../../environments/environment';

/**
 * Sentinel placeholder written into Supabase's session state during cold-start
 * restore. The real refresh token lives only in the HTTP-only cookie — we never
 * expose it to JS after a page reload. The storage adapter skips POSTing this
 * value back to the Go proxy (see setItem below).
 */
export const COOKIE_REFRESH_SENTINEL = '__cookie__';

/**
 * In-memory cache backing the Supabase storage adapter. Supabase JS calls
 * storage methods synchronously, so we return/write to this Map directly and
 * fire any Go proxy calls as async side effects. Nothing in this cache persists
 * across page reloads — reload-time session restore is handled by AuthService.
 */
const sessionCache = new Map<string, string>();

/** Test-only — resets the module-level cache between specs. */
export function __resetSessionCacheForTests(): void {
  sessionCache.clear();
}

/**
 * makeCookieStorage returns a Supabase-compatible storage adapter that routes
 * refresh-token persistence through the Go /api/v1/auth/session proxy. The
 * access token and Supabase user object stay in the in-memory cache; the
 * refresh token is mirrored into an HTTP-only cookie managed by Go.
 *
 * The adapter is intentionally fire-and-forget: Go calls happen as background
 * promises, with failures logged but not surfaced. Supabase's storage contract
 * is synchronous, so we cannot await them.
 */
export function makeCookieStorage(http: HttpClient): SupportedStorage {
  const sessionUrl = `${environment.apiBaseUrl}/auth/session`;

  return {
    getItem(key: string): string | null {
      return sessionCache.get(key) ?? null;
    },

    setItem(key: string, value: string): void {
      sessionCache.set(key, value);

      // Only persist the session token key — skip PKCE verifiers and other
      // Supabase bookkeeping keys that should never leave the browser.
      if (!key.includes('-auth-token')) return;

      try {
        const parsed = JSON.parse(value) as { refresh_token?: string };
        if (!parsed.refresh_token || parsed.refresh_token === COOKIE_REFRESH_SENTINEL) return;

        firstValueFrom(
          http.post(sessionUrl, { refresh_token: parsed.refresh_token }, { withCredentials: true }),
        ).catch(() => {
          console.warn('[auth] failed to persist session cookie');
        });
      } catch {
        // Malformed JSON — ignore, the cache is still authoritative in-memory.
      }
    },

    removeItem(key: string): void {
      sessionCache.delete(key);

      // Intentionally do not clear the server cookie from storage.removeItem.
      // Supabase may call removeItem during internal session housekeeping on
      // startup; clearing the cookie here can cause unexpected logout on the
      // next page refresh. Cookie deletion is handled explicitly by sign-out.
    },
  };
}

/**
 * Clears the refresh-token cookie held by the Go auth proxy.
 * This should only be called from explicit user sign-out flows.
 */
export async function clearSessionCookie(http: HttpClient): Promise<void> {
  const sessionUrl = `${environment.apiBaseUrl}/auth/session`;
  try {
    await firstValueFrom(http.delete(sessionUrl, { withCredentials: true }));
  } catch {
    console.warn('[auth] failed to clear session cookie');
  }
}

/**
 * Injection token for the Supabase client.
 *
 * Refresh tokens live in an HTTP-only cookie owned by the Go /auth/session
 * proxy. The access token lives in memory only (Supabase client internal state
 * + AuthService signal). Nothing session-related is written to localStorage.
 *
 * autoRefreshToken is disabled because Supabase JS cannot refresh a session
 * when the real refresh token is only in the cookie — AuthService handles
 * refresh manually via a timer that calls GET /auth/session.
 */
export const SUPABASE_CLIENT = new InjectionToken<SupabaseClient>('SupabaseClient', {
  providedIn: 'root',
  factory: () => {
    const http = inject(HttpClient);
    return createClient(environment.supabaseUrl, environment.supabaseAnonKey, {
      auth: {
        autoRefreshToken: false,
        persistSession: true,
        detectSessionInUrl: true,
        storage: makeCookieStorage(http),
      },
    });
  },
});
