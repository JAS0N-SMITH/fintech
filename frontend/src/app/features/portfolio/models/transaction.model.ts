/** Financial event types. Mirrors backend TransactionType enum. */
export type TransactionType = 'buy' | 'sell' | 'dividend' | 'reinvested_dividend';

/**
 * A recorded financial event.
 *
 * Decimal fields are serialised as strings by the Go backend (shopspring/decimal).
 * Nullable fields are absent from the JSON when not applicable to the transaction type:
 *   - buy/sell: quantity + price_per_share present
 *   - dividend: dividend_per_share present
 *   - reinvested_dividend: quantity + price_per_share + dividend_per_share all present
 */
export interface Transaction {
  id: string;
  portfolio_id: string;
  transaction_type: TransactionType;
  symbol: string;
  transaction_date: string; // ISO 8601 date string
  quantity?: string;
  price_per_share?: string;
  dividend_per_share?: string;
  total_amount: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

/** Input for recording a new transaction. */
export interface CreateTransactionInput {
  transaction_type: TransactionType;
  symbol: string;
  transaction_date: string; // YYYY-MM-DD
  quantity?: string;
  price_per_share?: string;
  dividend_per_share?: string;
  total_amount: string;
  notes?: string;
}

/**
 * A current holding derived from transaction history.
 *
 * Never stored — always computed from transactions via deriveHoldings().
 * All numeric fields are string-encoded decimals for display precision.
 */
export interface Holding {
  symbol: string;
  quantity: string;
  avgCostBasis: string;
  totalCost: string;
}
