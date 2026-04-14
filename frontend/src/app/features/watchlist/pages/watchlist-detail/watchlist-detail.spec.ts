import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { of } from 'rxjs';
import { WatchlistDetailComponent } from './watchlist-detail.component';
import { WatchlistService } from '../../services/watchlist.service';
import { TickerStateService } from '../../../../core/ticker-state.service';
import { MessageService, ConfirmationService } from 'primeng/api';
import type { Watchlist, WatchlistItem } from '../../models/watchlist.model';

describe('WatchlistDetailComponent', () => {
  let component: WatchlistDetailComponent;
  let fixture: ComponentFixture<WatchlistDetailComponent>;
  let watchlistServiceMock: any;
  let tickerStateServiceMock: any;
  let routerMock: any;
  let activatedRouteMock: any;

  const mockWatchlist: Watchlist = {
    id: 'wl-1',
    user_id: 'user-1',
    name: 'Tech Stocks',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  };

  const mockItems: WatchlistItem[] = [
    {
      id: 'item-1',
      watchlist_id: 'wl-1',
      symbol: 'AAPL',
      target_price: 150,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ];

  beforeEach(async () => {
    const watchlistSpy = {
      loadById: vi.fn(),
      removeItem: vi.fn(),
      cleanup: vi.fn(),
      selectedWatchlist: vi.fn().mockReturnValue(mockWatchlist),
      items: vi.fn().mockReturnValue(mockItems),
      loading: vi.fn().mockReturnValue(false),
    };

    const tickerSpy = {
      subscribe: vi.fn(),
      unsubscribe: vi.fn(),
      tickers: vi.fn().mockReturnValue({
        AAPL: {
          symbol: 'AAPL',
          currentPrice: 155,
          dayHigh: 156,
          dayLow: 150,
          previousClose: 154,
          quote: { previous_close: 154, price: 155 },
        },
      }),
    };

    const routerSpy = {
      navigate: vi.fn(),
    };

    activatedRouteMock = {
      snapshot: {
        paramMap: {
          get: vi.fn().mockReturnValue('wl-1'),
        },
      },
    };

    await TestBed.configureTestingModule({
      imports: [WatchlistDetailComponent],
      providers: [
        { provide: WatchlistService, useValue: watchlistSpy },
        { provide: TickerStateService, useValue: tickerSpy },
        { provide: Router, useValue: routerSpy },
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        MessageService,
        ConfirmationService,
      ],
    }).compileComponents();

    watchlistServiceMock = TestBed.inject(WatchlistService) as any;
    tickerStateServiceMock = TestBed.inject(TickerStateService) as any;
    routerMock = TestBed.inject(Router) as any;

    watchlistServiceMock.loadById.mockReturnValue(of(mockWatchlist));
    watchlistServiceMock.removeItem.mockReturnValue(of(void 0));

    fixture = TestBed.createComponent(WatchlistDetailComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should load watchlist on init', () => {
    fixture.detectChanges();

    expect(watchlistServiceMock.loadById).toHaveBeenCalledWith('wl-1');
  });

  it('should navigate to watchlists if no ID in route', () => {
    activatedRouteMock.snapshot.paramMap.get.mockReturnValue(null);

    fixture.detectChanges();

    expect(routerMock.navigate).toHaveBeenCalledWith(['/watchlists']);
  });

  it('should display watchlist items', () => {
    fixture.detectChanges();

    const items = component['items']();
    expect(items.length).toBe(1);
    expect(items[0].symbol).toBe('AAPL');
  });

  it('should get target price status above when price exceeds target', () => {
    const item = mockItems[0];
    const status = component['getTargetPriceStatus'](item);

    expect(status).toBe('above');
  });

  it('should get target price status below when price is below target', () => {
    const item: WatchlistItem = {
      ...mockItems[0],
      target_price: 160,
    };
    const status = component['getTargetPriceStatus'](item);

    expect(status).toBe('below');
  });

  it('should return null status when no target price', () => {
    const item: WatchlistItem = {
      ...mockItems[0],
      target_price: undefined,
    };
    const status = component['getTargetPriceStatus'](item);

    expect(status).toBeNull();
  });

  it('should open add item dialog', () => {
    fixture.detectChanges();
    component['openAddItemDialog']();

    expect(component['addItemDialogVisible']()).toBe(true);
  });

  it('should cleanup on destroy', () => {
    fixture.detectChanges();
    fixture.destroy();

    expect(watchlistServiceMock.cleanup).toHaveBeenCalled();
  });
});
