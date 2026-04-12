import { Routes } from '@angular/router';
import { Component } from '@angular/core';

// Placeholder component for Phase 8
@Component({
  selector: 'app-watchlist-placeholder',
  standalone: true,
  template: `<div class="p-6"><h2>Watchlists</h2><p>Coming in Phase 8</p></div>`,
})
class WatchlistPlaceholderComponent {}

export const WATCHLIST_ROUTES: Routes = [
  {
    path: '',
    component: WatchlistPlaceholderComponent,
  },
];
