import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../../environments/environment';
import type {
  ImportPreview,
  ImportConfirmRequest,
  ImportResult,
} from '../models/import.model';
import type { CreateTransactionInput } from '../models/transaction.model';

const importBase = (portfolioId: string) =>
  `${environment.apiBaseUrl}/portfolios/${portfolioId}/import`;

/**
 * ImportService handles CSV brokerage import operations.
 *
 * Two-step workflow:
 * 1. preview() — parses and validates CSV without persisting
 * 2. confirm() — persists selected transactions to the database
 */
@Injectable({ providedIn: 'root' })
export class ImportService {
  private readonly http = inject(HttpClient);

  /**
   * Preview parses and validates a CSV file without persisting transactions.
   *
   * @param portfolioId Portfolio ID to import to
   * @param file CSV file from user selection
   * @param brokerage Optional brokerage name (fidelity, sofi, generic). Auto-detected if omitted.
   * @returns Observable of ImportPreview with parsed/valid counts and any errors
   */
  preview(
    portfolioId: string,
    file: File,
    brokerage: string = '',
  ): Observable<ImportPreview> {
    const formData = new FormData();
    formData.append('file', file);

    let url = importBase(portfolioId);
    if (brokerage) {
      url += `?brokerage=${encodeURIComponent(brokerage)}`;
    }

    return this.http.post<ImportPreview>(url, formData);
  }

  /**
   * Confirm persists a set of validated transactions to the database.
   *
   * Called after user reviews the preview and selects rows to import.
   *
   * @param portfolioId Portfolio ID to import to
   * @param transactions Selected transactions from the preview
   * @returns Observable of ImportResult with created/failed counts
   */
  confirm(
    portfolioId: string,
    transactions: CreateTransactionInput[],
  ): Observable<ImportResult> {
    const request: ImportConfirmRequest = { transactions };
    return this.http.post<ImportResult>(
      `${importBase(portfolioId)}/confirm`,
      request,
    );
  }
}
