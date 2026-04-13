import type { Transaction } from '../../../portfolio/models/transaction.model';
import type { Bar } from '../../../portfolio/models/market-data.model';
import type { Time } from 'lightweight-charts';

export interface LineDataPoint {
  time: Time;
  value: number;
}

export interface SymbolBars {
  sym: string;
  bars: Bar[];
}

/**
 * Derives historical portfolio values from transactions and historical bar data.
 *
 * Computes the total portfolio value for each date by:
 * 1. Tracking cumulative share quantity per symbol (buy/sell/reinvested_dividend events)
 * 2. For each date, multiplying quantity × closing price for all symbols
 * 3. Summing across symbols to get aggregate portfolio value
 *
 * Only emits points where value > 0. Output times are strictly ascending.
 */
export function derivePortfolioValues(
  transactions: Transaction[],
  symbolBars: SymbolBars[]
): LineDataPoint[] {
  // Build symbol -> bars map
  const barMap = new Map(symbolBars.map((sb) => [sb.sym, sb.bars]));

  // Build symbol -> sorted quantity events (date, delta in shares)
  const quantityEvents = buildQuantityEvents(transactions);

  // If no bars or no quantity events, return empty
  if (barMap.size === 0 || quantityEvents.size === 0) {
    return [];
  }

  // Collect all unique YYYY-MM-DD dates from all bars, sorted ascending
  const allDates = [
    ...new Set(
      [...barMap.values()].flatMap((bars) =>
        bars.map((b) => b.timestamp.slice(0, 10))
      )
    ),
  ].sort();

  if (allDates.length === 0) {
    return [];
  }

  // Initialize running quantity per symbol and event cursors
  const runningQty = new Map<string, number>();
  const eventCursors = new Map<string, number>();

  for (const sym of barMap.keys()) {
    runningQty.set(sym, 0);
    eventCursors.set(sym, 0);
  }

  // Build per-symbol date -> Bar map for O(1) lookup
  const barsByDate = new Map<string, Map<string, Bar>>();
  for (const [sym, bars] of barMap) {
    const dateMap = new Map<string, Bar>();
    for (const bar of bars) {
      const dateStr = bar.timestamp.slice(0, 10);
      dateMap.set(dateStr, bar);
    }
    barsByDate.set(sym, dateMap);
  }

  const result: LineDataPoint[] = [];

  // Sweep through all dates
  for (const date of allDates) {
    // Advance quantity cursors for all symbols up to and including this date
    for (const [sym, events] of quantityEvents) {
      let cursor = eventCursors.get(sym) ?? 0;
      while (cursor < events.length && events[cursor].date <= date) {
        const qty = runningQty.get(sym) ?? 0;
        runningQty.set(sym, Math.max(0, qty + events[cursor].delta));
        cursor++;
      }
      eventCursors.set(sym, cursor);
    }

    // Compute portfolio value for this date
    let totalValue = 0;
    for (const [sym, dateMap] of barsByDate) {
      const qty = runningQty.get(sym) ?? 0;
      if (qty <= 0) continue;

      const bar = dateMap.get(date);
      if (bar) {
        totalValue += qty * bar.close;
      }
    }

    // Only emit if portfolio has value on this date
    if (totalValue > 0) {
      const unixSeconds = Math.floor(
        new Date(date + 'T00:00:00Z').getTime() / 1000
      );
      result.push({
        time: unixSeconds as Time,
        value: totalValue,
      });
    }
  }

  return result;
}

/**
 * Builds a map of symbol -> sorted quantity events (date, delta in shares).
 * Events are pre-sorted by date ascending for efficient incremental processing.
 *
 * - buy / reinvested_dividend: +quantity
 * - sell: -quantity
 * - dividend: skipped (cash only, no share delta)
 */
function buildQuantityEvents(
  transactions: Transaction[]
): Map<string, { date: string; delta: number }[]> {
  const eventMap = new Map<string, { date: string; delta: number }[]>();

  for (const tx of transactions) {
    // Skip cash transactions (dividend does not change share count)
    if (tx.transaction_type === 'dividend') {
      continue;
    }

    // Determine delta
    let delta = parseFloat(tx.quantity ?? '0');
    if (tx.transaction_type === 'sell') {
      delta = -delta;
    }
    // buy and reinvested_dividend: positive delta

    const date = tx.transaction_date.slice(0, 10); // YYYY-MM-DD

    if (!eventMap.has(tx.symbol)) {
      eventMap.set(tx.symbol, []);
    }

    eventMap.get(tx.symbol)!.push({ date, delta });
  }

  // Sort each symbol's events by date ascending
  for (const events of eventMap.values()) {
    events.sort((a, b) => a.date.localeCompare(b.date));
  }

  return eventMap;
}
