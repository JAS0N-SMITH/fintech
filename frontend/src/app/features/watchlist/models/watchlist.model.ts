export interface Watchlist {
  id: string;
  user_id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface WatchlistItem {
  id: string;
  watchlist_id: string;
  symbol: string;
  target_price?: number;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateWatchlistInput {
  name: string;
}

export interface UpdateWatchlistInput {
  name: string;
}

export interface CreateWatchlistItemInput {
  symbol: string;
  target_price?: number;
  notes?: string;
}

export interface UpdateWatchlistItemInput {
  target_price?: number;
  notes?: string;
}
