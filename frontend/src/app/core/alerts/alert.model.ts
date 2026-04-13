/**
 * Direction of a price alert threshold crossing.
 * 'above' fires when price rises to or above target.
 * 'below' fires when price falls to or below target.
 */
export type AlertDirection = 'above' | 'below';

/**
 * A single evaluatable price alert derived from a watchlist item's target_price.
 * AlertRules are computed from WatchlistService state — not stored separately.
 */
export interface AlertRule {
  readonly watchlistItemId: string;
  readonly symbol: string;
  targetPrice: number;
  direction: AlertDirection;
  /** True after the crossing fires once. Reset only when price recrosses back. */
  fired: boolean;
  /** The price at the last evaluation — used to detect crossing direction. */
  lastKnownPrice: number | null;
}

/**
 * A portfolio-level alert threshold configured by the user.
 * Stored in profiles.preferences JSONB under key "alert_thresholds".
 */
export interface PortfolioAlertThreshold {
  /** Unique stable identifier (can be a slug like "portfolio-daily-loss"). */
  id: string;
  type: 'portfolio_daily_change' | 'position_gain_loss';
  /** Symbol applies only when type === 'position_gain_loss'. */
  symbol?: string;
  /** Threshold percentage. Negative means loss (e.g. -5 = down 5%). */
  thresholdPercent: number;
  /** Direction: 'above' for gain alert, 'below' for loss alert. */
  direction: AlertDirection;
  fired: boolean;
}

/**
 * The shape stored under profiles.preferences.alert_thresholds.
 * This is what gets serialised to/from JSONB.
 */
export interface AlertPreferences {
  thresholds: PortfolioAlertThreshold[];
}

/** A fired alert event — passed to delivery layer. */
export interface AlertEvent {
  type: 'price' | 'portfolio';
  symbol?: string;
  title: string;
  detail: string;
}
