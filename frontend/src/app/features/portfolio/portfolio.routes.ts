import { Routes } from '@angular/router';

export const PORTFOLIO_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./pages/portfolio-list/portfolio-list.component').then(
        (m) => m.PortfolioListComponent,
      ),
  },
  {
    path: ':id',
    loadComponent: () =>
      import('./pages/portfolio-detail/portfolio-detail.component').then(
        (m) => m.PortfolioDetailComponent,
      ),
  },
];
