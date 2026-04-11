import { TestBed } from '@angular/core/testing';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { MarketDataService } from './market-data.service';
import type { Bar, Quote } from '../features/portfolio/models/market-data.model';
import { environment } from '../../environments/environment';

const BASE = `${environment.apiBaseUrl}`;

function makeQuote(overrides: Partial<Quote> = {}): Quote {
  return {
    symbol: 'AAPL',
    price: 150.25,
    day_high: 152.0,
    day_low: 148.5,
    open: 149.0,
    previous_close: 148.0,
    volume: 1000000,
    timestamp: '2024-01-15T16:00:00Z',
    ...overrides,
  };
}

function makeBar(overrides: Partial<Bar> = {}): Bar {
  return {
    symbol: 'AAPL',
    open: 149.0,
    high: 152.0,
    low: 148.5,
    close: 150.25,
    volume: 1000000,
    timestamp: '2024-01-15T16:00:00Z',
    ...overrides,
  };
}

describe('MarketDataService', () => {
  let service: MarketDataService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    service = TestBed.inject(MarketDataService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  describe('getQuote()', () => {
    it('fires GET /quotes/:symbol and returns a Quote', () => {
      const expected = makeQuote();
      let result: Quote | undefined;

      service.getQuote('AAPL').subscribe((q) => (result = q));

      const req = httpMock.expectOne(`${BASE}/quotes/AAPL`);
      expect(req.request.method).toBe('GET');
      req.flush(expected);

      expect(result).toEqual(expected);
    });

    it('URL-encodes the symbol', () => {
      service.getQuote('BRK.B').subscribe();
      const req = httpMock.expectOne(`${BASE}/quotes/BRK.B`);
      req.flush(makeQuote({ symbol: 'BRK.B' }));
    });
  });

  describe('getQuotesBatch()', () => {
    it('fires GET /quotes?symbols=... and returns a map', () => {
      const expected: Record<string, Quote> = {
        AAPL: makeQuote({ symbol: 'AAPL' }),
        MSFT: makeQuote({ symbol: 'MSFT', price: 300.0 }),
      };
      let result: Record<string, Quote> | undefined;

      service.getQuotesBatch(['AAPL', 'MSFT']).subscribe((r) => (result = r));

      const req = httpMock.expectOne((r) =>
        r.url === `${BASE}/quotes` && r.params.get('symbols') === 'AAPL,MSFT',
      );
      expect(req.request.method).toBe('GET');
      req.flush(expected);

      expect(result?.['AAPL']?.price).toBe(150.25);
      expect(result?.['MSFT']?.price).toBe(300.0);
    });
  });

  describe('getHistoricalBars()', () => {
    it('fires GET /bars/:symbol with default timeframe 1M', () => {
      const expected = [makeBar()];
      let result: Bar[] | undefined;

      service.getHistoricalBars('AAPL').subscribe((b) => (result = b));

      const req = httpMock.expectOne((r) =>
        r.url === `${BASE}/bars/AAPL` && r.params.get('timeframe') === '1M',
      );
      expect(req.request.method).toBe('GET');
      req.flush(expected);

      expect(result).toHaveLength(1);
      expect(result?.[0]?.close).toBe(150.25);
    });

    it('passes custom timeframe and date range as query params', () => {
      service
        .getHistoricalBars('AAPL', '1Y', '2023-01-01T00:00:00Z', '2024-01-01T00:00:00Z')
        .subscribe();

      const req = httpMock.expectOne((r) => {
        return (
          r.url === `${BASE}/bars/AAPL` &&
          r.params.get('timeframe') === '1Y' &&
          r.params.get('start') === '2023-01-01T00:00:00Z' &&
          r.params.get('end') === '2024-01-01T00:00:00Z'
        );
      });
      req.flush([]);
    });
  });
});
