import {
  ChangeDetectionStrategy,
  Component,
  input,
  inject,
  signal,
  output,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Dialog } from 'primeng/dialog';
import { Button } from 'primeng/button';
import { FileUpload } from 'primeng/fileupload';
import { Select } from 'primeng/select';
import { TableModule } from 'primeng/table';
import { Toast } from 'primeng/toast';
import { ProgressSpinner } from 'primeng/progressspinner';
import { MessageService } from 'primeng/api';
import { ImportService } from '../../services/import.service';
import type {
  ImportPreview,
  ImportResult,
} from '../../models/import.model';
import type { CreateTransactionInput } from '../../models/transaction.model';

/**
 * ImportDialogComponent manages a multi-step CSV import workflow:
 * 1. Upload: select CSV file and brokerage
 * 2. Preview: review parsed transactions, select/deselect rows
 * 3. Confirm: persist selected transactions
 *
 * Emits `imported` with the result after successful confirmation.
 */
@Component({
  selector: 'app-import-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    Dialog,
    Button,
    FileUpload,
    Select,
    TableModule,
    Toast,
    ProgressSpinner,
  ],
  providers: [MessageService],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './import-dialog.component.html',
  styleUrls: ['./import-dialog.component.css'],
})
export class ImportDialogComponent {
  private readonly importService = inject(ImportService);
  private readonly messageService = inject(MessageService);

  /** Portfolio ID to import to. */
  readonly portfolioId = input.required<string>();

  /** Emitted with the import result after successful confirmation. */
  readonly imported = output<ImportResult>();

  /** True when the dialog is visible. */
  readonly visible = signal(false);

  // Step 1: Upload
  protected readonly selectedFile = signal<File | null>(null);
  protected readonly selectedBrokerage = signal<string>('');
  protected readonly brokerageOptions = [
    { label: 'Auto-detect', value: '' },
    { label: 'Fidelity', value: 'fidelity' },
    { label: 'SoFi', value: 'sofi' },
    { label: 'Generic', value: 'generic' },
  ];

  // Step 2: Preview
  protected readonly currentStep = signal<'upload' | 'preview' | 'confirm'>('upload');
  protected readonly preview = signal<ImportPreview | null>(null);
  protected readonly selectedRows = signal<Set<number>>(new Set());
  protected readonly previewLoading = signal(false);
  protected readonly previewError = signal<string | null>(null);

  // Step 3: Confirm
  protected readonly confirming = signal(false);
  protected readonly confirmError = signal<string | null>(null);

  /** Opens the import dialog. */
  open(): void {
    this.reset();
    this.visible.set(true);
  }

  /** Closes the dialog. */
  close(): void {
    this.visible.set(false);
  }

  /** Handles file selection. */
  onFileSelect(event: any): void {
    const files = event.files as File[];
    if (files && files.length > 0) {
      this.selectedFile.set(files[0]);
    }
  }

  /** Initiates CSV preview (dry-run). */
  async doPreview(): Promise<void> {
    const file = this.selectedFile();
    if (!file) {
      this.messageService.add({
        severity: 'error',
        summary: 'Error',
        detail: 'Please select a CSV file',
      });
      return;
    }

    this.previewLoading.set(true);
    this.previewError.set(null);

    try {
      const preview = await this.importService
        .preview(this.portfolioId(), file, this.selectedBrokerage())
        .toPromise();

      if (!preview) {
        throw new Error('No preview data returned');
      }

      this.preview.set(preview);

      // Auto-select all valid rows initially
      const selected = new Set<number>();
      for (let i = 0; i < preview.transactions.length; i++) {
        selected.add(i);
      }
      this.selectedRows.set(selected);

      this.currentStep.set('preview');

      if (preview.errors.length > 0) {
        this.messageService.add({
          severity: 'warn',
          summary: 'Import Issues',
          detail: `${preview.errors.length} row(s) have errors and will be skipped`,
          sticky: true,
        });
      }
    } catch (err: any) {
      const message = err?.error?.detail || err?.message || 'Preview failed';
      this.previewError.set(message);
      this.messageService.add({
        severity: 'error',
        summary: 'Preview Failed',
        detail: message,
      });
    } finally {
      this.previewLoading.set(false);
    }
  }

  /** Checks if a row is selected. */
  isRowSelected(index: number): boolean {
    return this.selectedRows().has(index);
  }

  /** Toggles row selection. */
  toggleRow(index: number): void {
    const selected = new Set(this.selectedRows());
    if (selected.has(index)) {
      selected.delete(index);
    } else {
      selected.add(index);
    }
    this.selectedRows.set(selected);
  }

  /** Proceeds to confirm step with selected rows. */
  doConfirm(): void {
    const p = this.preview();
    if (!p) return;

    const selected = this.selectedRows();
    if (selected.size === 0) {
      this.messageService.add({
        severity: 'error',
        summary: 'Error',
        detail: 'Please select at least one transaction to import',
      });
      return;
    }

    const transactions: CreateTransactionInput[] = [];
    for (const idx of selected) {
      if (idx < p.transactions.length) {
        transactions.push(p.transactions[idx]);
      }
    }

    this.currentStep.set('confirm');

    // Auto-confirm with the transactions
    this.executeConfirm(transactions);
  }

  /** Executes the confirmation request. */
  private async executeConfirm(
    transactions: CreateTransactionInput[],
  ): Promise<void> {
    this.confirming.set(true);
    this.confirmError.set(null);

    try {
      const result = await this.importService
        .confirm(this.portfolioId(), transactions)
        .toPromise();

      if (!result) {
        throw new Error('No result returned');
      }

      // Show summary
      if (result.created > 0) {
        this.messageService.add({
          severity: 'success',
          summary: 'Import Successful',
          detail: result.messages.join(' • '),
        });
      }

      if (result.failed > 0) {
        this.messageService.add({
          severity: 'warn',
          summary: 'Partial Import',
          detail: `${result.failed} transaction(s) failed to import`,
          sticky: true,
        });
      }

      this.imported.emit(result);
      this.close();
    } catch (err: any) {
      const message = err?.error?.detail || err?.message || 'Import failed';
      this.confirmError.set(message);
      this.messageService.add({
        severity: 'error',
        summary: 'Import Failed',
        detail: message,
      });
    } finally {
      this.confirming.set(false);
    }
  }

  /** Returns to the preview step. */
  backToPreview(): void {
    this.currentStep.set('preview');
    this.confirmError.set(null);
  }

  /** Cancels the import process. */
  cancel(): void {
    this.close();
  }

  private reset(): void {
    this.selectedFile.set(null);
    this.selectedBrokerage.set('');
    this.currentStep.set('upload');
    this.preview.set(null);
    this.selectedRows.set(new Set());
    this.previewError.set(null);
    this.confirmError.set(null);
  }
}
