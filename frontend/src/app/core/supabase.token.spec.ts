import { TestBed } from '@angular/core/testing';
import {
  HttpClient,
  HttpErrorResponse,
  provideHttpClient,
} from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import {
  COOKIE_REFRESH_SENTINEL,
  __resetSessionCacheForTests,
  clearSessionCookie,
  makeCookieStorage,
} from './supabase.token';
import { environment } from '../../environments/environment';

const SESSION_URL = `${environment.apiBaseUrl}/auth/session`;
const AUTH_KEY = 'sb-test-auth-token';

describe('cookieStorage (Supabase adapter)', () => {
  let http: HttpClient;
  let httpMock: HttpTestingController;
  let storage: ReturnType<typeof makeCookieStorage>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpClient);
    httpMock = TestBed.inject(HttpTestingController);
    storage = makeCookieStorage(http);
    __resetSessionCacheForTests();
  });

  afterEach(() => {
    httpMock.verify();
  });

  describe('getItem', () => {
    it('returns null when key is absent', () => {
      expect(storage.getItem('missing')).toBeNull();
    });

    it('returns cached value after setItem (non-auth key skips HTTP)', () => {
      storage.setItem('sb-code-verifier', 'abc');
      expect(storage.getItem('sb-code-verifier')).toBe('abc');
      httpMock.expectNone(SESSION_URL);
    });
  });

  describe('setItem', () => {
    it('posts refresh_token to Go when auth-token key has a real refresh_token', () => {
      const value = JSON.stringify({
        access_token: 'at',
        refresh_token: 'rt-real',
      });
      storage.setItem(AUTH_KEY, value);

      const req = httpMock.expectOne(SESSION_URL);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ refresh_token: 'rt-real' });
      expect(req.request.withCredentials).toBe(true);
      req.flush(null);

      expect(storage.getItem(AUTH_KEY)).toBe(value);
    });

    it('does not POST when refresh_token is the cookie sentinel', () => {
      const value = JSON.stringify({
        access_token: 'at',
        refresh_token: COOKIE_REFRESH_SENTINEL,
      });
      storage.setItem(AUTH_KEY, value);
      httpMock.expectNone(SESSION_URL);
      expect(storage.getItem(AUTH_KEY)).toBe(value);
    });

    it('does not POST when refresh_token field is missing', () => {
      const value = JSON.stringify({ access_token: 'at' });
      storage.setItem(AUTH_KEY, value);
      httpMock.expectNone(SESSION_URL);
      expect(storage.getItem(AUTH_KEY)).toBe(value);
    });

    it('does not throw or POST when value is malformed JSON', () => {
      expect(() => storage.setItem(AUTH_KEY, '{')).not.toThrow();
      httpMock.expectNone(SESSION_URL);
      expect(storage.getItem(AUTH_KEY)).toBe('{');
    });

    it('does not POST for non-auth-token keys', () => {
      storage.setItem('sb-code-verifier', 'pkce-value');
      httpMock.expectNone(SESSION_URL);
      expect(storage.getItem('sb-code-verifier')).toBe('pkce-value');
    });

    it('swallows POST failures without throwing', async () => {
      const value = JSON.stringify({ access_token: 'at', refresh_token: 'rt' });
      expect(() => storage.setItem(AUTH_KEY, value)).not.toThrow();

      const req = httpMock.expectOne(SESSION_URL);
      req.flush('fail', { status: 500, statusText: 'Server Error' });

      // Value still cached despite POST failure.
      expect(storage.getItem(AUTH_KEY)).toBe(value);
    });
  });

  describe('removeItem', () => {
    it('deletes from cache and does not call Go DELETE implicitly', () => {
      storage.setItem(AUTH_KEY, JSON.stringify({ refresh_token: 'rt' }));
      httpMock.expectOne(SESSION_URL).flush(null); // consume the POST

      storage.removeItem(AUTH_KEY);
      httpMock.expectNone(SESSION_URL);

      expect(storage.getItem(AUTH_KEY)).toBeNull();
    });

    it('does not call DELETE for non-auth-token keys', () => {
      storage.setItem('sb-code-verifier', 'v');
      storage.removeItem('sb-code-verifier');
      httpMock.expectNone(SESSION_URL);
      expect(storage.getItem('sb-code-verifier')).toBeNull();
    });

    it('does not throw when removing auth-token key', () => {
      expect(() => storage.removeItem(AUTH_KEY)).not.toThrow();
    });
  });

  describe('clearSessionCookie', () => {
    it('calls Go DELETE with credentials', async () => {
      const pending = clearSessionCookie(http);
      const req = httpMock.expectOne(SESSION_URL);
      expect(req.request.method).toBe('DELETE');
      expect(req.request.withCredentials).toBe(true);
      req.flush(null);
      await pending;
    });

    it('swallows DELETE failures without throwing', async () => {
      const pending = clearSessionCookie(http);
      const req = httpMock.expectOne(SESSION_URL);
      req.flush('fail', { status: 500, statusText: 'Server Error' });
      await pending;
    });
  });
});
