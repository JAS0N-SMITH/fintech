import { TestBed } from '@angular/core/testing';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { PortfolioService } from './portfolio.service';
import type { Portfolio } from '../models/portfolio.model';
import { environment } from '../../../../environments/environment';

const BASE = `${environment.apiBaseUrl}/portfolios`;

function makePortfolio(overrides: Partial<Portfolio> = {}): Portfolio {
  return {
    id: 'p1',
    user_id: 'u1',
    name: 'Main',
    description: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

describe('PortfolioService', () => {
  let service: PortfolioService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    service = TestBed.inject(PortfolioService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => httpMock.verify());

  describe('loadAll()', () => {
    it('fires GET /portfolios and populates the signal', () => {
      const data = [makePortfolio({ id: 'p1' }), makePortfolio({ id: 'p2', name: 'Roth' })];

      service.loadAll().subscribe();

      const req = httpMock.expectOne(BASE);
      expect(req.request.method).toBe('GET');
      req.flush(data);

      expect(service.portfolios()).toEqual(data);
    });

    it('sets loading false after success', () => {
      service.loadAll().subscribe();
      httpMock.expectOne(BASE).flush([]);
      expect(service.loading()).toBe(false);
    });

    it('sets loading false after error', () => {
      service.loadAll().subscribe({ error: () => { /* intentionally ignored for test */ } });
      httpMock.expectOne(BASE).flush('error', { status: 500, statusText: 'Server Error' });
      expect(service.loading()).toBe(false);
    });
  });

  describe('create()', () => {
    it('fires POST /portfolios with correct body and prepends to signal', () => {
      const existing = makePortfolio({ id: 'p0' });
      // seed initial state via loadAll
      service.loadAll().subscribe();
      httpMock.expectOne(BASE).flush([existing]);

      const newPortfolio = makePortfolio({ id: 'p1', name: 'Brokerage' });
      service.create({ name: 'Brokerage' }).subscribe();

      const req = httpMock.expectOne(BASE);
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ name: 'Brokerage' });
      req.flush(newPortfolio);

      expect(service.portfolios()[0].id).toBe('p1');
      expect(service.portfolios().length).toBe(2);
    });
  });

  describe('update()', () => {
    it('fires PUT /portfolios/:id and replaces the entry in the signal', () => {
      const original = makePortfolio({ id: 'p1', name: 'Old' });
      service.loadAll().subscribe();
      httpMock.expectOne(BASE).flush([original]);

      const updated = { ...original, name: 'New' };
      service.update('p1', { name: 'New' }).subscribe();

      const req = httpMock.expectOne(`${BASE}/p1`);
      expect(req.request.method).toBe('PUT');
      req.flush(updated);

      expect(service.portfolios()[0].name).toBe('New');
    });
  });

  describe('delete()', () => {
    it('fires DELETE /portfolios/:id and removes from signal', () => {
      const p1 = makePortfolio({ id: 'p1' });
      const p2 = makePortfolio({ id: 'p2', name: 'Roth' });
      service.loadAll().subscribe();
      httpMock.expectOne(BASE).flush([p1, p2]);

      service.delete('p1').subscribe();

      const req = httpMock.expectOne(`${BASE}/p1`);
      expect(req.request.method).toBe('DELETE');
      req.flush(null);

      expect(service.portfolios().length).toBe(1);
      expect(service.portfolios()[0].id).toBe('p2');
    });
  });

  describe('loadById()', () => {
    it('fires GET /portfolios/:id and sets selectedPortfolio', () => {
      const p = makePortfolio({ id: 'p1' });
      service.loadById('p1').subscribe();

      const req = httpMock.expectOne(`${BASE}/p1`);
      expect(req.request.method).toBe('GET');
      req.flush(p);

      expect(service.selectedPortfolio()).toEqual(p);
    });
  });
});
