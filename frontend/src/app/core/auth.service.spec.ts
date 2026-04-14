import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import type { AuthChangeEvent, Session, User } from '@supabase/supabase-js';
import { AuthService } from './auth.service';
import { COOKIE_REFRESH_SENTINEL, SUPABASE_CLIENT } from './supabase.token';
import { environment } from '../../environments/environment';

// ---------------------------------------------------------------------------
// Mock Supabase client
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

function buildSession(accessToken = 'mock-access-token', expiresIn = 900): Session {
  return {
    access_token: accessToken,
    refresh_token: 'mock-refresh-token',
    token_type: 'bearer',
    expires_in: expiresIn,
    user: mockUser,
  } as Session;
}

function createMockSupabase() {
  let authCallback: AuthStateCallback | null = null;

  return {
    auth: {
      signInWithPassword: vi.fn().mockResolvedValue({ data: { session: null }, error: null }),
      signUp: vi.fn().mockResolvedValue({ error: null }),
      signOut: vi.fn().mockResolvedValue({ error: null }),
      setSession: vi.fn().mockImplementation(async ({ access_token }: { access_token: string }) => {
        const session = { ...buildSession(access_token) };
        authCallback?.('TOKEN_REFRESHED', session);
        return { data: { session, user: mockUser }, error: null };
      }),
      onAuthStateChange: vi.fn().mockImplementation((cb: AuthStateCallback) => {
        authCallback = cb;
        return { data: { subscription: { unsubscribe: vi.fn() } } };
      }),
      _emit: (event: AuthChangeEvent, session: Session | null) => {
        authCallback?.(event, session);
      },
    },
  };
}

const SESSION_URL = `${environment.apiBaseUrl}/auth/session`;

/** Flush the microtask queue so resolved promises propagate. */
const flush = () => Promise.resolve().then(() => Promise.resolve());

// ---------------------------------------------------------------------------

describe('AuthService', () => {
  let service: AuthService;
  let mockSupabase: ReturnType<typeof createMockSupabase>;
  let router: Router;
  let httpMock: HttpTestingController;

  function setup(): void {
    mockSupabase = createMockSupabase();
    TestBed.configureTestingModule({
      providers: [
        provideRouter([{ path: '**', redirectTo: '' }]),
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: SUPABASE_CLIENT, useValue: mockSupabase },
      ],
    });
    service = TestBed.inject(AuthService);
    router = TestBed.inject(Router);
    httpMock = TestBed.inject(HttpTestingController);
  }

  afterEach(() => {
    vi.useRealTimers();
    TestBed.resetTestingModule();
  });

  // -------------------------------------------------------------------------
  // Cold start (page reload) flow
  // -------------------------------------------------------------------------

  describe('cold-start restore', () => {
    it('keeps loading true until restore attempt completes', () => {
      setup();

      // Simulate the early null event Supabase can emit on startup.
      mockSupabase.auth._emit('SIGNED_OUT', null);
      expect(service.isLoading()).toBe(true);

      httpMock
        .expectOne(SESSION_URL)
        .flush('no session', { status: 401, statusText: 'Unauthorized' });

      expect(service.isLoading()).toBe(false);
    });

    it('calls GET /auth/session with credentials on init', () => {
      setup();
      const req = httpMock.expectOne(SESSION_URL);
      expect(req.request.method).toBe('GET');
      expect(req.request.withCredentials).toBe(true);
      req.flush({ access_token: 'at', expires_in: 900, token_type: 'bearer', user: mockUser });
    });

    it('restores the session on happy path', async () => {
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({
          access_token: 'at-restored',
          expires_in: 900,
          token_type: 'bearer',
          user: mockUser,
        });
      await flush();

      expect(mockSupabase.auth.setSession).toHaveBeenCalledWith({
        access_token: 'at-restored',
        refresh_token: COOKIE_REFRESH_SENTINEL,
      });
      expect(service.isAuthenticated()).toBe(true);
      expect(service.accessToken()).toBe('at-restored');
      expect(service.isLoading()).toBe(false);
    });

    it('clears state on 401 from Go', async () => {
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush('no session', { status: 401, statusText: 'Unauthorized' });
      await flush();

      expect(mockSupabase.auth.setSession).not.toHaveBeenCalled();
      expect(service.isAuthenticated()).toBe(false);
      expect(service.isLoading()).toBe(false);
    });

    it('clears state on network error', async () => {
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .error(new ProgressEvent('error'), { status: 0, statusText: 'Network' });
      await flush();

      expect(service.isAuthenticated()).toBe(false);
      expect(service.isLoading()).toBe(false);
    });

    it('clears state when setSession returns an error', async () => {
      setup();
      mockSupabase.auth.setSession.mockResolvedValueOnce({
        data: { session: null, user: null },
        error: { message: 'bad token', status: 400 },
      });
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();

      expect(service.isAuthenticated()).toBe(false);
      expect(service.isLoading()).toBe(false);
    });
  });

  // -------------------------------------------------------------------------
  // Refresh timer
  // -------------------------------------------------------------------------

  describe('refresh timer', () => {
    it('schedules refresh at expires_in minus 60 seconds', async () => {
      vi.useFakeTimers();
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at1', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();

      // (900 - 60) * 1000 = 840000ms. Advance to just before that.
      vi.advanceTimersByTime(839_999);
      httpMock.expectNone(SESSION_URL);

      vi.advanceTimersByTime(1);
      const second = httpMock.expectOne(SESSION_URL);
      expect(second.request.method).toBe('GET');
      second.flush({ access_token: 'at2', expires_in: 900, token_type: 'bearer', user: mockUser });
    });

    it('enforces minimum lead time of 10 seconds', async () => {
      vi.useFakeTimers();
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at1', expires_in: 5, token_type: 'bearer', user: mockUser });
      await flush();

      vi.advanceTimersByTime(9_999);
      httpMock.expectNone(SESSION_URL);
      vi.advanceTimersByTime(1);
      httpMock.expectOne(SESSION_URL).flush({
        access_token: 'at2',
        expires_in: 900,
        token_type: 'bearer',
        user: mockUser,
      });
    });

    it('stops refreshing after a failed refresh', async () => {
      vi.useFakeTimers();
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at1', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();

      vi.advanceTimersByTime(840_000);
      httpMock.expectOne(SESSION_URL).flush('gone', { status: 401, statusText: 'Unauthorized' });
      await flush();

      expect(service.isAuthenticated()).toBe(false);

      vi.advanceTimersByTime(60 * 60 * 1000);
      httpMock.expectNone(SESSION_URL);
    });

    it('is cancelled on signOut', async () => {
      vi.useFakeTimers();
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at1', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();

      await service.signOut();
      vi.advanceTimersByTime(60 * 60 * 1000);
      httpMock.expectNone(SESSION_URL);
    });

    it('is cancelled on destroy', async () => {
      vi.useFakeTimers();
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at1', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();

      service.ngOnDestroy();
      vi.advanceTimersByTime(60 * 60 * 1000);
      httpMock.expectNone(SESSION_URL);
    });
  });

  // -------------------------------------------------------------------------
  // Auth state changes
  // -------------------------------------------------------------------------

  describe('auth state changes', () => {
    it('updates signals when Supabase emits SIGNED_IN', async () => {
      setup();
      httpMock.expectOne(SESSION_URL).flush('', { status: 401, statusText: 'Unauthorized' });
      await flush();
      expect(service.isAuthenticated()).toBe(false);

      mockSupabase.auth._emit('SIGNED_IN', buildSession());

      expect(service.isAuthenticated()).toBe(true);
      expect(service.user()?.id).toBe('user-123');
    });

    it('clears signals when Supabase emits SIGNED_OUT', async () => {
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();
      expect(service.isAuthenticated()).toBe(true);

      mockSupabase.auth._emit('SIGNED_OUT', null);
      expect(service.isAuthenticated()).toBe(false);
    });
  });

  // -------------------------------------------------------------------------
  // signIn / signUp / signOut
  // -------------------------------------------------------------------------

  describe('signIn', () => {
    it('calls supabase signInWithPassword with credentials', async () => {
      setup();
      httpMock.expectOne(SESSION_URL).flush('', { status: 401, statusText: 'Unauthorized' });
      await service.signIn('test@example.com', 'password123');
      expect(mockSupabase.auth.signInWithPassword).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
      });
    });

    it('returns null error on success', async () => {
      setup();
      httpMock.expectOne(SESSION_URL).flush('', { status: 401, statusText: 'Unauthorized' });
      const { error } = await service.signIn('test@example.com', 'password123');
      expect(error).toBeNull();
    });

    it('returns error when Supabase rejects', async () => {
      setup();
      httpMock.expectOne(SESSION_URL).flush('', { status: 401, statusText: 'Unauthorized' });
      const authError = { message: 'Invalid credentials', status: 400 };
      mockSupabase.auth.signInWithPassword.mockResolvedValueOnce({
        data: { session: null },
        error: authError,
      });
      const { error } = await service.signIn('bad@example.com', 'wrongpass');
      expect(error).toBe(authError);
    });
  });

  describe('signUp', () => {
    it('calls supabase signUp with credentials', async () => {
      setup();
      httpMock.expectOne(SESSION_URL).flush('', { status: 401, statusText: 'Unauthorized' });
      await service.signUp('new@example.com', 'password123');
      expect(mockSupabase.auth.signUp).toHaveBeenCalledWith({
        email: 'new@example.com',
        password: 'password123',
      });
    });
  });

  describe('signOut', () => {
    it('calls supabase signOut and navigates to login', async () => {
      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();

      const spy = vi.spyOn(router, 'navigate');
      await service.signOut();

      expect(mockSupabase.auth.signOut).toHaveBeenCalled();
      expect(spy).toHaveBeenCalledWith(['/auth/login']);
    });
  });

  // -------------------------------------------------------------------------
  // localStorage regression guard
  // -------------------------------------------------------------------------

  describe('localStorage isolation', () => {
    it('never reads or writes a session key in localStorage', async () => {
      const setSpy = vi.spyOn(Storage.prototype, 'setItem');
      const getSpy = vi.spyOn(Storage.prototype, 'getItem');

      setup();
      httpMock
        .expectOne(SESSION_URL)
        .flush({ access_token: 'at', expires_in: 900, token_type: 'bearer', user: mockUser });
      await flush();
      mockSupabase.auth._emit('SIGNED_IN', buildSession());
      await service.signOut();

      const authKeyRegex = /sb-.*-auth-token/;
      const sessionWrites = setSpy.mock.calls.filter(([k]) => authKeyRegex.test(String(k)));
      const sessionReads = getSpy.mock.calls.filter(([k]) => authKeyRegex.test(String(k)));
      expect(sessionWrites).toEqual([]);
      expect(sessionReads).toEqual([]);
    });
  });

  describe('ngOnDestroy', () => {
    it('unsubscribes from auth state changes', async () => {
      setup();
      httpMock.expectOne(SESSION_URL).flush('', { status: 401, statusText: 'Unauthorized' });
      await flush();
      const unsubSpy =
        mockSupabase.auth.onAuthStateChange.mock.results[0].value.data.subscription.unsubscribe;
      service.ngOnDestroy();
      expect(unsubSpy).toHaveBeenCalled();
    });
  });
};);
