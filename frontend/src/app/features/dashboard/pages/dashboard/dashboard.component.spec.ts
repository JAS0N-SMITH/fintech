import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DashboardComponent } from './dashboard.component';
import { signal } from '@angular/core';
import { PortfolioService } from '../../../portfolio/services/portfolio.service';
import { TransactionService } from '../../../portfolio/services/transaction.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import { of } from 'rxjs';

describe('DashboardComponent', () => {
  let component: DashboardComponent;
  let fixture: ComponentFixture<DashboardComponent>;
  let portfolioService: any;
  let transactionService: any;
  let tickerStateService: any;

  beforeEach(async () => {
    const portfolioServiceMock = {
      portfolios: signal([]).asReadonly(),
      loadAll: vi.fn(),
    };
    const transactionServiceMock = {
      transactions: signal([]).asReadonly(),
      holdings: signal([]).asReadonly(),
      loadByPortfolio: vi.fn(),
      clear: vi.fn(),
    };
    const tickerStateServiceMock = {
      tickers: signal<Record<string, unknown>>({}).asReadonly(),
      subscribe: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [DashboardComponent],
      providers: [
        {
          provide: PortfolioService,
          useValue: portfolioServiceMock as unknown as PortfolioService,
        },
        {
          provide: TransactionService,
          useValue: transactionServiceMock as unknown as TransactionService,
        },
        {
          provide: TickerStateService,
          useValue: tickerStateServiceMock as unknown as TickerStateService,
        },
      ],
    }).compileComponents();

    portfolioService = TestBed.inject(PortfolioService) as unknown as typeof portfolioServiceMock;
    transactionService = TestBed.inject(
      TransactionService,
    ) as unknown as typeof transactionServiceMock;
    tickerStateService = TestBed.inject(
      TickerStateService,
    ) as unknown as typeof tickerStateServiceMock;

    portfolioService.loadAll.mockReturnValue(of([]));
    transactionService.loadByPortfolio.mockReturnValue(of([]));

    fixture = TestBed.createComponent(DashboardComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load all portfolios on init', () => {
    fixture.detectChanges();
    expect(portfolioService.loadAll).toHaveBeenCalled();
  });

  it('should clear transactions on destroy', () => {
    component.ngOnDestroy();
    expect(transactionService.clear).toHaveBeenCalled();
  });

  it('should compute total portfolio value', () => {
    expect(component.totalPortfolioValue()).toBe('0.00');
  });

  it('should compute total unrealized gain/loss', () => {
    expect(component.totalUnrealizedGainLoss()).toBe('0.00');
  });

  it('should compute day gain/loss', () => {
    expect(component.dayGainLoss()).toBe('0.00');
  });
});
