import { inject, Injectable, OnDestroy, signal, computed } from '@angular/core';
import { Router } from '@angular/router';
import type { AuthError, Session, User } from '@supabase/supabase-js';
import { SUPABASE_CLIENT } from './supabase.token';

/** Represents the result of a sign-in or sign-up operation. */
export interface AuthResult {
  error: AuthError | null;
}

/**
 * AuthService manages authentication state using Supabase Auth.
 *
 * Access token is held in memory via a signal — never written to localStorage.
 * Auth state changes (login, logout, token refresh) are streamed from Supabase
 * and reflected in signals for reactive consumption across the app.
 */
@Injectable({ providedIn: 'root' })
export class AuthService implements OnDestroy {
  private readonly supabase = inject(SUPABASE_CLIENT);
  private readonly router = inject(Router);

  private readonly _session = signal<Session | null>(null);
  private readonly _isLoading = signal(true);

  /** The currently authenticated user, or null if not signed in. */
  readonly user = computed<User | null>(() => this._session()?.user ?? null);

  /** The current JWT access token, or null if not signed in. */
  readonly accessToken = computed<string | null>(() => this._session()?.access_token ?? null);

  /** True when a user is signed in and the session is valid. */
  readonly isAuthenticated = computed(() => this._session() !== null);

  /** True while the initial session check is in flight. */
  readonly isLoading = this._isLoading.asReadonly();

  private authSubscription: { unsubscribe: () => void } | null = null;

  constructor() {
    this.initAuthListener();
  }

  /**
   * Signs in an existing user with email and password.
   * On success, Supabase emits a SIGNED_IN event which updates all signals.
   */
  async signIn(email: string, password: string): Promise<AuthResult> {
    const { error } = await this.supabase.auth.signInWithPassword({ email, password });
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
   * Clears all session signals and in-memory tokens.
   */
  async signOut(): Promise<void> {
    await this.supabase.auth.signOut();
    await this.router.navigate(['/auth/login']);
  }

  ngOnDestroy(): void {
    this.authSubscription?.unsubscribe();
  }

  /**
   * Subscribes to Supabase auth state changes and keeps signals in sync.
   * Called once at construction time.
   */
  private initAuthListener(): void {
    // Seed with the current session before the listener fires.
    this.supabase.auth.getSession().then(({ data }) => {
      this._session.set(data.session);
      this._isLoading.set(false);
    });

    const { data } = this.supabase.auth.onAuthStateChange((_event, session) => {
      this._session.set(session);
      this._isLoading.set(false);
    });

    this.authSubscription = data.subscription;
  }
}
