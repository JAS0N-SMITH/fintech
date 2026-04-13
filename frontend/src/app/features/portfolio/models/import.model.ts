import type { CreateTransactionInput } from './transaction.model';

/** Error for a single row during CSV import. */
export interface ImportError {
  row: number; // 1-indexed row number
  message: string; // Human-readable error description
}

/** Preview response from the import endpoint (dry-run, no DB writes). */
export interface ImportPreview {
  parsed: number; // Total rows parsed (excluding header)
  valid: number; // Rows that passed validation
  errors: ImportError[]; // Per-row errors
  transactions: CreateTransactionInput[]; // Valid transactions ready to create
}

/** Request to confirm and create transactions after preview approval. */
export interface ImportConfirmRequest {
  transactions: CreateTransactionInput[]; // Selected transactions to create
}

/** Result response after confirming transactions. */
export interface ImportResult {
  created: number; // Number successfully created
  failed: number; // Number that failed
  errors: ImportError[]; // Per-row failure details
  messages: string[]; // Summary messages
}
