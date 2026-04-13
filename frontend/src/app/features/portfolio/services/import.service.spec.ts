import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ImportService } from './import.service';
import type {
  ImportPreview,
  ImportConfirmRequest,
  ImportResult,
} from '../models/import.model';
import type { CreateTransactionInput } from '../models/transaction.model';

describe('ImportService', () => {
  let service: ImportService;
  let httpMock: HttpTestingController;
  const portfolioId = 'port-123';
  const baseUrl = `/api/v1/portfolios/${portfolioId}/import`;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [ImportService],
    });

    service = TestBed.inject(ImportService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  describe('preview', () => {
    it('should POST CSV file and return preview', (done) => {
      const file = new File(['Symbol,Date\nAAPL,2024-01-15'], 'test.csv', {
        type: 'text/csv',
      });

      const mockPreview: ImportPreview = {
        parsed: 1,
        valid: 1,
        errors: [],
        transactions: [
          {
            transaction_type: 'buy',
            symbol: 'AAPL',
            transaction_date: '2024-01-15',
            total_amount: '1500.00',
          },
        ],
      };

      service.preview(portfolioId, file).subscribe((result) => {
        expect(result.parsed).toBe(1);
        expect(result.valid).toBe(1);
        expect(result.transactions.length).toBe(1);
        done();
      });

      const req = httpMock.expectOne(baseUrl);
      expect(req.request.method).toBe('POST');
      expect(req.request.body instanceof FormData).toBeTruthy();
      req.flush(mockPreview);
    });

    it('should include brokerage param when provided', (done) => {
      const file = new File(['test'], 'test.csv', { type: 'text/csv' });
      const mockPreview: ImportPreview = {
        parsed: 0,
        valid: 0,
        errors: [],
        transactions: [],
      };

      service.preview(portfolioId, file, 'fidelity').subscribe(() => {
        done();
      });

      const req = httpMock.expectOne(`${baseUrl}?brokerage=fidelity`);
      expect(req.request.method).toBe('POST');
      req.flush(mockPreview);
    });

    it('should return preview with errors', (done) => {
      const file = new File(['test'], 'test.csv', { type: 'text/csv' });
      const mockPreview: ImportPreview = {
        parsed: 2,
        valid: 1,
        errors: [{ row: 2, message: 'invalid symbol format' }],
        transactions: [
          {
            transaction_type: 'buy',
            symbol: 'AAPL',
            transaction_date: '2024-01-15',
            total_amount: '1500.00',
          },
        ],
      };

      service.preview(portfolioId, file).subscribe((result) => {
        expect(result.parsed).toBe(2);
        expect(result.valid).toBe(1);
        expect(result.errors.length).toBe(1);
        expect(result.errors[0].message).toBe('invalid symbol format');
        done();
      });

      const req = httpMock.expectOne(baseUrl);
      req.flush(mockPreview);
    });

    it('should URL-encode brokerage param', (done) => {
      const file = new File(['test'], 'test.csv', { type: 'text/csv' });
      const mockPreview: ImportPreview = {
        parsed: 0,
        valid: 0,
        errors: [],
        transactions: [],
      };

      service.preview(portfolioId, file, 'test brokerage').subscribe(() => {
        done();
      });

      const req = httpMock.expectOne(`${baseUrl}?brokerage=test%20brokerage`);
      req.flush(mockPreview);
    });
  });

  describe('confirm', () => {
    it('should POST transactions and return result', (done) => {
      const transactions: CreateTransactionInput[] = [
        {
          transaction_type: 'buy',
          symbol: 'AAPL',
          transaction_date: '2024-01-15',
          quantity: '10',
          price_per_share: '150.00',
          total_amount: '1500.00',
        },
      ];

      const mockResult: ImportResult = {
        created: 1,
        failed: 0,
        errors: [],
        messages: ['Successfully imported 1 transactions'],
      };

      service.confirm(portfolioId, transactions).subscribe((result) => {
        expect(result.created).toBe(1);
        expect(result.failed).toBe(0);
        expect(result.messages.length).toBe(1);
        done();
      });

      const req = httpMock.expectOne(`${baseUrl}/confirm`);
      expect(req.request.method).toBe('POST');

      // Verify request body
      const body = req.request.body as ImportConfirmRequest;
      expect(body.transactions.length).toBe(1);
      expect(body.transactions[0].symbol).toBe('AAPL');

      req.flush(mockResult);
    });

    it('should handle partial import failures', (done) => {
      const transactions: CreateTransactionInput[] = [
        {
          transaction_type: 'buy',
          symbol: 'AAPL',
          transaction_date: '2024-01-15',
          quantity: '10',
          price_per_share: '150.00',
          total_amount: '1500.00',
        },
        {
          transaction_type: 'sell',
          symbol: 'AAPL',
          transaction_date: '2024-02-01',
          quantity: '100',
          price_per_share: '160.00',
          total_amount: '16000.00',
        },
      ];

      const mockResult: ImportResult = {
        created: 1,
        failed: 1,
        errors: [{ row: 2, message: 'insufficient holdings' }],
        messages: [
          'Successfully imported 1 transactions',
          'Failed to import 1 transactions (see errors for details)',
        ],
      };

      service.confirm(portfolioId, transactions).subscribe((result) => {
        expect(result.created).toBe(1);
        expect(result.failed).toBe(1);
        expect(result.errors.length).toBe(1);
        expect(result.messages.length).toBe(2);
        done();
      });

      const req = httpMock.expectOne(`${baseUrl}/confirm`);
      req.flush(mockResult);
    });

    it('should handle multiple transactions', (done) => {
      const transactions: CreateTransactionInput[] = [
        {
          transaction_type: 'buy',
          symbol: 'AAPL',
          transaction_date: '2024-01-15',
          quantity: '10',
          price_per_share: '150.00',
          total_amount: '1500.00',
        },
        {
          transaction_type: 'buy',
          symbol: 'TSLA',
          transaction_date: '2024-01-20',
          quantity: '5',
          price_per_share: '200.00',
          total_amount: '1000.00',
        },
      ];

      const mockResult: ImportResult = {
        created: 2,
        failed: 0,
        errors: [],
        messages: ['Successfully imported 2 transactions'],
      };

      service.confirm(portfolioId, transactions).subscribe((result) => {
        expect(result.created).toBe(2);
        done();
      });

      const req = httpMock.expectOne(`${baseUrl}/confirm`);
      const body = req.request.body as ImportConfirmRequest;
      expect(body.transactions.length).toBe(2);
      req.flush(mockResult);
    });

    it('should serialize decimal fields as strings', (done) => {
      const transactions: CreateTransactionInput[] = [
        {
          transaction_type: 'buy',
          symbol: 'AAPL',
          transaction_date: '2024-01-15',
          quantity: '10.5',
          price_per_share: '150.25',
          total_amount: '1577.625',
        },
      ];

      const mockResult: ImportResult = {
        created: 1,
        failed: 0,
        errors: [],
        messages: [],
      };

      service.confirm(portfolioId, transactions).subscribe(() => {
        done();
      });

      const req = httpMock.expectOne(`${baseUrl}/confirm`);
      const body = req.request.body as ImportConfirmRequest;
      const tx = body.transactions[0];
      expect(tx.quantity).toBe('10.5');
      expect(tx.price_per_share).toBe('150.25');
      expect(tx.total_amount).toBe('1577.625');
      req.flush(mockResult);
    });
  });

  describe('error handling', () => {
    it('should propagate preview errors', (done) => {
      const file = new File(['test'], 'test.csv', { type: 'text/csv' });

      service.preview(portfolioId, file).subscribe({
        next: () => {
          fail('should have errored');
        },
        error: (err) => {
          expect(err.status).toBe(413);
          done();
        },
      });

      const req = httpMock.expectOne(baseUrl);
      req.flush('File too large', { status: 413, statusText: 'Payload Too Large' });
    });

    it('should propagate confirm errors', (done) => {
      const transactions: CreateTransactionInput[] = [];

      service.confirm(portfolioId, transactions).subscribe({
        next: () => {
          fail('should have errored');
        },
        error: (err) => {
          expect(err.status).toBe(400);
          done();
        },
      });

      const req = httpMock.expectOne(`${baseUrl}/confirm`);
      req.flush('Bad request', { status: 400, statusText: 'Bad Request' });
    });
  });
});
