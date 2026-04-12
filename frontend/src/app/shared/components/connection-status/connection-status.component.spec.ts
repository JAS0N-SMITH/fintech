import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { ConnectionStatusComponent } from './connection-status.component';
import { TickerStateService } from '../../../core/ticker-state.service';
import type { ConnectionState, TickerState } from '../../../features/portfolio/models/market-data.model';

describe('ConnectionStatusComponent', () => {
  let component: ConnectionStatusComponent;
  let fixture: ComponentFixture<ConnectionStatusComponent>;
  let tickerStateService: Partial<TickerStateService>;

  beforeEach(async () => {
    // Mock TickerStateService
    const mockConnectionState = signal<ConnectionState>('disconnected');
    const mockTickers = signal<Record<string, TickerState>>({});

    tickerStateService = {
      connectionState: () => mockConnectionState(),
      tickers: () => mockTickers(),
    };

    await TestBed.configureTestingModule({
      imports: [ConnectionStatusComponent],
      providers: [
        { provide: TickerStateService, useValue: tickerStateService },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ConnectionStatusComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  describe('label computation', () => {
    it('should display "Live" when connected', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('connected');

      fixture.detectChanges();
      expect(component.label()).toBe('Live');
    });

    it('should display "Reconnecting…" when reconnecting', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('reconnecting');

      fixture.detectChanges();
      expect(component.label()).toBe('Reconnecting…');
    });

    it('should display "Offline" when disconnected', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('disconnected');

      fixture.detectChanges();
      expect(component.label()).toBe('Offline');
    });
  });

  describe('severity computation', () => {
    it('should have "success" severity when connected', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('connected');

      fixture.detectChanges();
      expect(component.severity()).toBe('success');
    });

    it('should have "warn" severity when reconnecting', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('reconnecting');

      fixture.detectChanges();
      expect(component.severity()).toBe('warn');
    });

    it('should have "danger" severity when disconnected', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('disconnected');

      fixture.detectChanges();
      expect(component.severity()).toBe('danger');
    });
  });

  describe('last updated display', () => {
    it('should not show stale info when connected', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('connected');

      fixture.detectChanges();
      expect(component.showStaleInfo()).toBe(false);
    });

    it('should show stale info when not connected', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('disconnected');

      fixture.detectChanges();
      expect(component.showStaleInfo()).toBe(true);
    });

    it('should return null for lastUpdated when no symbol provided', () => {
      expect(component.lastUpdated()).toBeNull();
    });

    it('should return lastUpdated for provided symbol', () => {
      const now = new Date();
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._tickers = signal<Record<string, TickerState>>({
        AAPL: {
          symbol: 'AAPL',
          quote: null,
          currentPrice: 150,
          dayHigh: 155,
          dayLow: 148,
          previousClose: 149,
          lastUpdated: now,
        },
      });

      TestBed.runInInjectionContext(() => {
        const localFixture = TestBed.createComponent(ConnectionStatusComponent);
        localFixture.componentRef.setInput('symbol', 'AAPL');
        localFixture.detectChanges();

        expect(localFixture.componentInstance.lastUpdated()).toBe(now);
      });
    });
  });

  describe('accessibility', () => {
    it('should have accessible aria-label without symbol', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('connected');

      fixture.detectChanges();
      const ariaLabel = component.ariaLabel();
      expect(ariaLabel).toContain('Global connection status');
    });

    it('should have accessible aria-label with symbol', () => {
      const mockService = TestBed.inject(TickerStateService) as any;
      mockService._connectionState = signal<ConnectionState>('disconnected');

      TestBed.runInInjectionContext(() => {
        const localFixture = TestBed.createComponent(ConnectionStatusComponent);
        localFixture.componentRef.setInput('symbol', 'AAPL');
        localFixture.detectChanges();

        const ariaLabel = localFixture.componentInstance.ariaLabel();
        expect(ariaLabel).toContain('AAPL');
        expect(ariaLabel).toContain('disconnected');
      });
    });
  });
});
