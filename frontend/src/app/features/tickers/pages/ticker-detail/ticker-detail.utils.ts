import type { Transaction } from '../../../portfolio/models/transaction.model';

/**
 * Computes the holding period (earliest transaction date) for a symbol.
 * Returns the ISO date string of the oldest transaction, or null if no transactions exist.
 *
 * @param transactions Array of transactions, typically filtered to a single symbol
 * @returns ISO date string (YYYY-MM-DD) of the earliest transaction, or null
 */
export function computeHoldingPeriod(transactions: Transaction[]): string | null {
  if (!transactions.length) return null;
  return transactions.reduce((min, tx) =>
    tx.transaction_date < min ? tx.transaction_date : min,
    transactions[0].transaction_date
  );
}
