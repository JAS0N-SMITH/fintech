import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DashboardComponent } from './dashboard.component';
import { PortfolioService } from '../../services/portfolio.service';
import { TransactionService } from '../../services/transaction.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import { of } from 'rxjs';

describe('DashboardComponent', () => {
  let component: DashboardComponent;
  let fixture: ComponentFixture<DashboardComponent>;
  let portfolioService: jasmine.SpyObj<PortfolioService>;
  let transactionService: jasmine.SpyObj<TransactionService>;
  let tickerStateService: jasmine.SpyObj<TickerStateService>;

  beforeEach(async () => {
    const portfolioServiceMock = jasmine.createSpyObj('PortfolioService', ['loadAll'], {
      portfolios: jasmine.createSpy().and.returnValue([]),
    });
    const transactionServiceMock = jasmine.createSpyObj(
      'TransactionService',
      ['loadByPortfolio', 'clear'],
      {
        transactions: jasmine.createSpy().and.returnValue([]),
        holdings: jasmine.createSpy().and.returnValue([]),
      }
    );
    const tickerStateServiceMock = jasmine.createSpyObj('TickerStateService', ['subscribe'], {
      tickers: jasmine.createSpy().and.returnValue({}),
    });

    await TestBed.configureTestingModule({
      imports: [DashboardComponent],
      providers: [
        { provide: PortfolioService, useValue: portfolioServiceMock },
        { provide: TransactionService, useValue: transactionServiceMock },
        { provide: TickerStateService, useValue: tickerStateServiceMock },
      ],
    }).compileComponents();

    portfolioService = TestBed.inject(PortfolioService) as jasmine.SpyObj<PortfolioService>;
    transactionService = TestBed.inject(TransactionService) as jasmine.SpyObj<TransactionService>;
    tickerStateService = TestBed.inject(TickerStateService) as jasmine.SpyObj<TickerStateService>;

    portfolioService.loadAll.and.returnValue(of([]));
    transactionService.loadByPortfolio.and.returnValue(of([]));

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
