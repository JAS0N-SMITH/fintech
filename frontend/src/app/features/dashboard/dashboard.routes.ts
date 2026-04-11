import { Routes } from '@angular/router';

export const DASHBOARD_ROUTES: Routes = [
  { path: '', redirectTo: 'portfolios', pathMatch: 'full' },
  {
    path: 'portfolios',
    loadChildren: () =>
      import('../portfolio/portfolio.routes').then((m) => m.PORTFOLIO_ROUTES),
  },
];
