import { TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { provideRouter } from '@angular/router';
import type { AuthChangeEvent, Session, User } from '@supabase/supabase-js';
import { AuthService } from './auth.service';
import { SUPABASE_CLIENT } from './supabase.token';

// ---------------------------------------------------------------------------
// Minimal Supabase auth mock
// ---------------------------------------------------------------------------

type AuthStateCallback = (event: AuthChangeEvent, session: Session | null) => void;

const mockUser: User = {
  id: 'user-123',
  email: 'test@example.com',
  app_metadata: { role: 'user' },
  user_metadata: {},
  aud: 'authenticated',
  created_at: new Date().toISOString(),
} as User;

const mockSession: Session = {
  access_token: 'mock-access-token',
  refresh_token: 'mock-refresh-token',
  token_type: 'bearer',
  expires_in: 900,
  user: mockUser,
} as Session;

function createMockSupabase(initialSession: Session | null = null) {
  let authCallback: AuthStateCallback | null = null;

  return {
    auth: {
      getSession: vi.fn().mockResolvedValue({ data: { session: initialSession } }),
      signInWithPassword: vi.fn().mockResolvedValue({ error: null }),
      signUp: vi.fn().mockResolvedValue({ error: null }),
      signOut: vi.fn().mockResolvedValue({ error: null }),
      onAuthStateChange: vi.fn().mockImplementation((cb: AuthStateCallback) => {
        authCallback = cb;
        return { data: { subscription: { unsubscribe: vi.fn() } } };
      }),
      /** Simulates Supabase firing an auth state event. */
      _emit: (event: AuthChangeEvent, session: Session | null) => {
        authCallback?.(event, session);
      },
    },
  };
}

/** Flush the microtask queue so resolved promises propagate to signal updates. */
const flush = () => Promise.resolve();

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('AuthService', () => {
  let service: AuthService;
  let mockSupabase: ReturnType<typeof createMockSupabase>;
  let router: Router;

  function setup(initialSession: Session | null = null) {
    mockSupabase = createMockSupabase(initialSession);

    TestBed.configureTestingModule({
      providers: [
        provideRouter([{ path: '**', redirectTo: '' }]),
        { provide: SUPABASE_CLIENT, useValue: mockSupabase },
      ],
    });

    service = TestBed.inject(AuthService);
    router = TestBed.inject(Router);
  }

  afterEach(() => TestBed.resetTestingModule());

  describe('initial state (before getSession resolves)', () => {
    it('isAuthenticated is false', () => {
      setup(null);
      expect(service.isAuthenticated()).toBe(false);
    });

    it('isLoading is true', () => {
      setup(null);
      expect(service.isLoading()).toBe(true);
    });

    it('user is null', () => {
      setup(null);
      expect(service.user()).toBeNull();
    });

    it('accessToken is null', () => {
      setup(null);
      expect(service.accessToken()).toBeNull();
    });
  });

  describe('after getSession resolves with no session', () => {
    it('isAuthenticated is false', async () => {
      setup(null);
      await flush();
      expect(service.isAuthenticated()).toBe(false);
    });

    it('isLoading becomes false', async () => {
      setup(null);
      await flush();
      expect(service.isLoading()).toBe(false);
    });

    it('user remains null', async () => {
      setup(null);
      await flush();
      expect(service.user()).toBeNull();
    });
  });

  describe('after getSession resolves with an active session', () => {
    it('isAuthenticated is true', async () => {
      setup(mockSession);
      await flush();
      expect(service.isAuthenticated()).toBe(true);
    });

    it('user reflects the session user', async () => {
      setup(mockSession);
      await flush();
      expect(service.user()?.id).toBe('user-123');
    });

    it('accessToken reflects the session token', async () => {
      setup(mockSession);
      await flush();
      expect(service.accessToken()).toBe('mock-access-token');
    });

    it('isLoading becomes false', async () => {
      setup(mockSession);
      await flush();
      expect(service.isLoading()).toBe(false);
    });
  });

  describe('auth state changes', () => {
    it('updates signals when Supabase fires SIGNED_IN', async () => {
      setup(null);
      await flush();
      expect(service.isAuthenticated()).toBe(false);

      mockSupabase.auth._emit('SIGNED_IN', mockSession);

      expect(service.isAuthenticated()).toBe(true);
      expect(service.user()?.id).toBe('user-123');
      expect(service.accessToken()).toBe('mock-access-token');
    });

    it('clears signals when Supabase fires SIGNED_OUT', async () => {
      setup(mockSession);
      await flush();
      expect(service.isAuthenticated()).toBe(true);

      mockSupabase.auth._emit('SIGNED_OUT', null);

      expect(service.isAuthenticated()).toBe(false);
      expect(service.user()).toBeNull();
      expect(service.accessToken()).toBeNull();
    });
  });

  describe('signIn', () => {
    it('calls supabase signInWithPassword with credentials', async () => {
      setup(null);
      await service.signIn('test@example.com', 'password123');
      expect(mockSupabase.auth.signInWithPassword).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });

    it('returns null error on success', async () => {
      setup(null);
      const { error } = await service.signIn('test@example.com', 'password123');
      expect(error).toBeNull();
    });

    it('returns error when Supabase rejects', async () => {
      setup(null);
      const authError = { message: 'Invalid credentials', status: 400 };
      mockSupabase.auth.signInWithPassword.mockResolvedValue({ error: authError });
      const { error } = await service.signIn('bad@example.com', 'wrongpass');
      expect(error).toBe(authError);
    });
  });

  describe('signUp', () => {
    it('calls supabase signUp with credentials', async () => {
      setup(null);
      await service.signUp('new@example.com', 'password123');
      expect(mockSupabase.auth.signUp).toHaveBeenCalledWith({
        email: 'new@example.com',
        password: 'password123',
      });
    });

    it('returns null error on success', async () => {
      setup(null);
      const { error } = await service.signUp('new@example.com', 'password123');
      expect(error).toBeNull();
    });

    it('returns error when Supabase rejects', async () => {
      setup(null);
      const authError = { message: 'User already registered', status: 422 };
      mockSupabase.auth.signUp.mockResolvedValue({ error: authError });
      const { error } = await service.signUp('existing@example.com', 'password123');
      expect(error).toBe(authError);
    });
  });

  describe('signOut', () => {
    it('calls supabase signOut', async () => {
      setup(mockSession);
      await service.signOut();
      expect(mockSupabase.auth.signOut).toHaveBeenCalled();
    });

    it('navigates to /auth/login after sign out', async () => {
      setup(mockSession);
      const spy = vi.spyOn(router, 'navigate');
      await service.signOut();
      expect(spy).toHaveBeenCalledWith(['/auth/login']);
    });
  });

  describe('ngOnDestroy', () => {
    it('unsubscribes from auth state changes', () => {
      setup(null);
      const unsubSpy =
        mockSupabase.auth.onAuthStateChange.mock.results[0].value.data.subscription.unsubscribe;
      service.ngOnDestroy();
      expect(unsubSpy).toHaveBeenCalled();
    });
  });
});
