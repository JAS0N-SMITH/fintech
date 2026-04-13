import { InjectionToken } from '@angular/core';
import { createClient, SupabaseClient } from '@supabase/supabase-js';
import { environment } from '../../environments/environment';

/**
 * Injection token for the Supabase client. Inject this instead of calling createClient directly.
 *
 * Session persistence: Supabase uses its default localStorage adapter so that the refresh token
 * survives page reloads. The access token is short-lived (15 min) and refreshed automatically.
 *
 * TODO(Phase 3): Replace with an HTTP-only cookie via a Go /auth/refresh proxy endpoint
 * to eliminate localStorage entirely and meet the stricter token storage rule.
 */
export const SUPABASE_CLIENT = new InjectionToken<SupabaseClient>('SupabaseClient', {
  providedIn: 'root',
  factory: () =>
    createClient(environment.supabaseUrl, environment.supabaseAnonKey, {
      auth: {
        autoRefreshToken: true,
        persistSession: true,
        detectSessionInUrl: true,
      },
    }),
});
