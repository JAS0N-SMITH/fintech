import { InjectionToken } from '@angular/core';
import { createClient, SupabaseClient } from '@supabase/supabase-js';
import { environment } from '../../environments/environment';

/**
 * In-memory storage adapter for the Supabase client.
 *
 * Prevents access and refresh tokens from being written to localStorage.
 * Tokens live only in this map for the lifetime of the page session.
 *
 * TODO(Phase 3): Replace refresh token persistence with an HTTP-only cookie
 * via a Go /auth/refresh proxy endpoint so it survives page reloads securely.
 */
const memoryStorage = new Map<string, string>();

const inMemoryStorageAdapter = {
  getItem: (key: string): string | null => memoryStorage.get(key) ?? null,
  setItem: (key: string, value: string): void => { memoryStorage.set(key, value); },
  removeItem: (key: string): void => { memoryStorage.delete(key); },
};

/** Injection token for the Supabase client. Inject this instead of calling createClient directly. */
export const SUPABASE_CLIENT = new InjectionToken<SupabaseClient>('SupabaseClient', {
  providedIn: 'root',
  factory: () =>
    createClient(environment.supabaseUrl, environment.supabaseAnonKey, {
      auth: {
        storage: inMemoryStorageAdapter,
        autoRefreshToken: true,
        persistSession: true,
        detectSessionInUrl: true,
      },
    }),
});
