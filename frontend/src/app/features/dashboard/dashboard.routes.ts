import { Routes } from '@angular/router';
import { AppShellComponent } from '../../shared/layout/app-shell/app-shell.component';

export const DASHBOARD_ROUTES: Routes = [
  {
    path: '',
    component: AppShellComponent,
    children: [
      {
        path: '',
        loadComponent: () =>
          import('./pages/dashboard/dashboard.component').then(m => m.DashboardComponent),
      },
      {
        path: 'portfolios',
        loadChildren: () =>
          import('../portfolio/portfolio.routes').then((m) => m.PORTFOLIO_ROUTES),
      },
      {
        path: 'watchlists',
        loadChildren: () =>
          import('../watchlist/watchlist.routes').then((m) => m.WATCHLIST_ROUTES),
      },
      {
        path: 'tickers/:symbol',
        loadChildren: () =>
          import('../tickers/ticker.routes').then((m) => m.TICKER_ROUTES),
      },
    ],
  },
];
