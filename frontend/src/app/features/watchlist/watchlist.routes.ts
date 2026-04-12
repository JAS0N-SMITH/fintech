import { Routes } from '@angular/router';
import { WatchlistListComponent } from './pages/watchlist-list/watchlist-list.component';
import { WatchlistDetailComponent } from './pages/watchlist-detail/watchlist-detail.component';

export const WATCHLIST_ROUTES: Routes = [
  {
    path: '',
    component: WatchlistListComponent,
  },
  {
    path: ':id',
    component: WatchlistDetailComponent,
  },
];
