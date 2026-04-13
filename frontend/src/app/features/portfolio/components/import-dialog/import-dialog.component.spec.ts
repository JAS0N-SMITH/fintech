import { ComponentFixture, TestBed } from '@angular/core/testing';
import { HttpClientTestingModule } from '@angular/common/http/testing';
import { MessageService } from 'primeng/api';
import { ImportDialogComponent } from './import-dialog.component';
import { ImportService } from '../../services/import.service';
import type { ImportPreview, ImportResult } from '../../models/import.model';
import type { CreateTransactionInput } from '../../models/transaction.model';
import { of, throwError } from 'rxjs';

describe('ImportDialogComponent', () => {
  let component: ImportDialogComponent;
  let fixture: ComponentFixture<ImportDialogComponent>;
  let importService: jasmine.SpyObj<ImportService>;
  let messageService: jasmine.SpyObj<MessageService>;

  beforeEach(async () => {
    const importServiceSpy = jasmine.createSpyObj('ImportService', [
      'preview',
      'confirm',
    ]);
    const messageServiceSpy = jasmine.createSpyObj('MessageService', ['add']);

    await TestBed.configureTestingModule({
      imports: [ImportDialogComponent, HttpClientTestingModule],
      providers: [
        { provide: ImportService, useValue: importServiceSpy },
        { provide: MessageService, useValue: messageServiceSpy },
      ],
    }).compileComponents();

    importService = TestBed.inject(ImportService) as jasmine.SpyObj<ImportService>;
    messageService = TestBed.inject(MessageService) as jasmine.SpyObj<MessageService>;

    fixture = TestBed.createComponent(ImportDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('Dialog lifecycle', () => {
    it('should open dialog with reset state', () => {
      component.open();
      expect(component.visible()).toBeTruthy();
      expect(component.currentStep()).toBe('upload');
      expect(component.selectedFile()).toBeNull();
      expect(component.selectedBrokerage()).toBe('');
      expect(component.preview()).toBeNull();
    });

    it('should close dialog', () => {
      component.visible.set(true);
      component.close();
      expect(component.visible()).toBeFalsy();
    });

    it('should cancel and close', () => {
      component.visible.set(true);
      component.cancel();
      expect(component.visible()).toBeFalsy();
    });
  });

  describe('File selection (Step 1: Upload)', () => {
    it('should set selected file on file select', () => {
      const file = new File(['test'], 'test.csv', { type: 'text/csv' });
      const event = { files: [file] };

      component.onFileSelect(event);
      expect(component.selectedFile()).toBe(file);
    });

    it('should set brokerage on selection', () => {
      component.selectedBrokerage.set('fidelity');
      expect(component.selectedBrokerage()).toBe('fidelity');
    });

    it('should show error if no file selected for preview', async () => {
      component.selectedFile.set(null);
      await component.doPreview();
      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          detail: jasmine.stringContaining('select a CSV file'),
        })
      );
    });
  });

  describe('Preview (Step 2)', () => {
    beforeEach(() => {
      const file = new File(['test'], 'test.csv', { type: 'text/csv' });
      component.selectedFile.set(file);
    });

    it('should transition to preview step on successful preview', async () => {
      const mockPreview: ImportPreview = {
        parsed: 2,
        valid: 2,
        errors: [],
        transactions: [
          {
            transaction_type: 'buy',
            symbol: 'AAPL',
            transaction_date: '2024-01-15',
            total_amount: '1500.00',
          },
          {
            transaction_type: 'buy',
            symbol: 'TSLA',
            transaction_date: '2024-01-16',
            total_amount: '1000.00',
          },
        ],
      };

      importService.preview.and.returnValue(of(mockPreview));

      await component.doPreview();
      fixture.detectChanges();

      expect(component.currentStep()).toBe('preview');
      expect(component.preview()).toEqual(mockPreview);
      expect(importService.preview).toHaveBeenCalled();
    });

    it('should auto-select all valid rows in preview', async () => {
      const mockPreview: ImportPreview = {
        parsed: 2,
        valid: 2,
        errors: [],
        transactions: [
          {
            transaction_type: 'buy',
            symbol: 'AAPL',
            transaction_date: '2024-01-15',
            total_amount: '1500.00',
          },
          {
            transaction_type: 'buy',
            symbol: 'TSLA',
            transaction_date: '2024-01-16',
            total_amount: '1000.00',
          },
        ],
      };

      importService.preview.and.returnValue(of(mockPreview));

      await component.doPreview();
      fixture.detectChanges();

      expect(component.selectedRows().size).toBe(2);
      expect(component.isRowSelected(0)).toBeTruthy();
      expect(component.isRowSelected(1)).toBeTruthy();
    });

    it('should show warning if preview has errors', async () => {
      const mockPreview: ImportPreview = {
        parsed: 3,
        valid: 2,
        errors: [{ row: 3, message: 'invalid date format' }],
        transactions: [
          {
            transaction_type: 'buy',
            symbol: 'AAPL',
            transaction_date: '2024-01-15',
            total_amount: '1500.00',
          },
          {
            transaction_type: 'buy',
            symbol: 'TSLA',
            transaction_date: '2024-01-16',
            total_amount: '1000.00',
          },
        ],
      };

      importService.preview.and.returnValue(of(mockPreview));

      await component.doPreview();
      fixture.detectChanges();

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'warn',
          detail: jasmine.stringContaining('1 row(s) have errors'),
        })
      );
    });

    it('should handle preview errors', async () => {
      importService.preview.and.returnValue(
        throwError(() => new Error('Preview failed'))
      );

      await component.doPreview();
      fixture.detectChanges();

      expect(component.previewError()).toContain('Preview failed');
      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Preview Failed',
        })
      );
    });

    it('should toggle row selection', () => {
      const mockPreview: ImportPreview = {
        parsed: 2,
        valid: 2,
        errors: [],
        transactions: [
          {
            transaction_type: 'buy',
            symbol: 'AAPL',
            transaction_date: '2024-01-15',
            total_amount: '1500.00',
          },
          {
            transaction_type: 'buy',
            symbol: 'TSLA',
            transaction_date: '2024-01-16',
            total_amount: '1000.00',
          },
        ],
      };

      component.preview.set(mockPreview);
      component.selectedRows.set(new Set([0, 1]));

      component.toggleRow(0);
      expect(component.isRowSelected(0)).toBeFalsy();
      expect(component.isRowSelected(1)).toBeTruthy();

      component.toggleRow(1);
      expect(component.isRowSelected(0)).toBeFalsy();
      expect(component.isRowSelected(1)).toBeFalsy();
    });
  });

  describe('Confirm (Step 3)', () => {
    beforeEach(() => {
      const mockPreview: ImportPreview = {
        parsed: 2,
        valid: 2,
        errors: [],
        transactions: [
          {
            transaction_type: 'buy',
            symbol: 'AAPL',
            transaction_date: '2024-01-15',
            total_amount: '1500.00',
          },
          {
            transaction_type: 'buy',
            symbol: 'TSLA',
            transaction_date: '2024-01-16',
            total_amount: '1000.00',
          },
        ],
      };
      component.preview.set(mockPreview);
      component.currentStep.set('preview');
      component.selectedRows.set(new Set([0, 1]));
    });

    it('should not allow confirm with no rows selected', () => {
      component.selectedRows.set(new Set());
      component.doConfirm();

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          detail: jasmine.stringContaining('select at least one'),
        })
      );
    });

    it('should transition to confirm step and execute import', () => {
      const mockResult: ImportResult = {
        created: 2,
        failed: 0,
        errors: [],
        messages: ['Successfully imported 2 transactions'],
      };

      importService.confirm.and.returnValue(of(mockResult));

      component.doConfirm();
      fixture.detectChanges();

      expect(component.currentStep()).toBe('confirm');
      expect(importService.confirm).toHaveBeenCalled();
    });

    it('should emit imported result on successful confirm', (done) => {
      const mockResult: ImportResult = {
        created: 2,
        failed: 0,
        errors: [],
        messages: ['Successfully imported 2 transactions'],
      };

      importService.confirm.and.returnValue(of(mockResult));

      component.imported.subscribe((result) => {
        expect(result).toEqual(mockResult);
        done();
      });

      component.doConfirm();
      fixture.detectChanges();
    });

    it('should close dialog after successful confirm', () => {
      const mockResult: ImportResult = {
        created: 2,
        failed: 0,
        errors: [],
        messages: ['Successfully imported 2 transactions'],
      };

      importService.confirm.and.returnValue(of(mockResult));
      component.visible.set(true);

      component.doConfirm();
      fixture.detectChanges();

      expect(component.visible()).toBeFalsy();
    });

    it('should show success message on successful confirm', () => {
      const mockResult: ImportResult = {
        created: 2,
        failed: 0,
        errors: [],
        messages: ['Successfully imported 2 transactions'],
      };

      importService.confirm.and.returnValue(of(mockResult));

      component.doConfirm();
      fixture.detectChanges();

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'success',
          summary: 'Import Successful',
        })
      );
    });

    it('should handle partial import failures', () => {
      const mockResult: ImportResult = {
        created: 1,
        failed: 1,
        errors: [{ row: 2, message: 'insufficient holdings' }],
        messages: [
          'Successfully imported 1 transactions',
          'Failed to import 1 transactions (see errors for details)',
        ],
      };

      importService.confirm.and.returnValue(of(mockResult));

      component.doConfirm();
      fixture.detectChanges();

      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'warn',
          detail: jasmine.stringContaining('1 transaction(s) failed'),
        })
      );
    });

    it('should handle confirm errors', () => {
      importService.confirm.and.returnValue(
        throwError(() => new Error('Confirm failed'))
      );

      component.doConfirm();
      fixture.detectChanges();

      expect(component.confirmError()).toContain('Confirm failed');
      expect(messageService.add).toHaveBeenCalledWith(
        jasmine.objectContaining({
          severity: 'error',
          summary: 'Import Failed',
        })
      );
    });

    it('should allow returning to preview on error', () => {
      component.currentStep.set('confirm');
      component.confirmError.set('Some error');

      component.backToPreview();

      expect(component.currentStep()).toBe('preview');
      expect(component.confirmError()).toBeNull();
    });
  });

  describe('Brokerage selection', () => {
    it('should have correct brokerage options', () => {
      const options = component.brokerageOptions;
      expect(options.length).toBe(4);
      expect(options[0].label).toBe('Auto-detect');
      expect(options[1].label).toBe('Fidelity');
      expect(options[2].label).toBe('SoFi');
      expect(options[3].label).toBe('Generic');
    });

    it('should pass selected brokerage to preview', () => {
      const file = new File(['test'], 'test.csv', { type: 'text/csv' });
      component.selectedFile.set(file);
      component.selectedBrokerage.set('fidelity');

      const mockPreview: ImportPreview = {
        parsed: 0,
        valid: 0,
        errors: [],
        transactions: [],
      };

      importService.preview.and.returnValue(of(mockPreview));

      component.doPreview();

      expect(importService.preview).toHaveBeenCalledWith(
        component.portfolioId(),
        file,
        'fidelity'
      );
    });
  });
});
