import { Routes } from '@angular/router';

export const TICKER_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./pages/ticker-detail/ticker-detail.component').then(
        (m) => m.TickerDetailComponent
      ),
  },
];
