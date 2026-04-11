import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnDestroy,
  OnInit,
  output,
  signal,
} from '@angular/core';
import {
  ReactiveFormsModule,
  FormBuilder,
  Validators,
  AbstractControl,
} from '@angular/forms';
import { Subscription } from 'rxjs';
import { Button } from 'primeng/button';
import { InputText } from 'primeng/inputtext';
import { InputNumber } from 'primeng/inputnumber';
import { Select } from 'primeng/select';
import { DatePicker } from 'primeng/datepicker';
import { Textarea } from 'primeng/textarea';
import type { CreateTransactionInput, TransactionType } from '../../models/transaction.model';

const TRANSACTION_TYPE_OPTIONS: { label: string; value: TransactionType }[] = [
  { label: 'Buy', value: 'buy' },
  { label: 'Sell', value: 'sell' },
  { label: 'Dividend', value: 'dividend' },
  { label: 'Reinvested dividend', value: 'reinvested_dividend' },
];

/**
 * TransactionFormComponent renders a form for recording a financial event.
 *
 * Field visibility and validators change based on the selected transaction type:
 *   - buy/sell:               symbol, date, quantity*, price_per_share*, total_amount*, notes
 *   - dividend:               symbol, date, dividend_per_share*, total_amount*, notes
 *   - reinvested_dividend:    symbol, date, quantity*, price_per_share*, dividend_per_share*, total_amount*, notes
 *
 * (* = required for that type)
 *
 * Emits `submitted` with CreateTransactionInput on valid submit.
 * Emits `cancelled` when the user dismisses.
 */
@Component({
  selector: 'app-transaction-form',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    Button,
    InputText,
    InputNumber,
    Select,
    DatePicker,
    Textarea,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './transaction-form.component.html',
})
export class TransactionFormComponent implements OnInit, OnDestroy {
  private readonly fb = inject(FormBuilder);

  /** Emitted with the validated input data on successful submit. */
  readonly submitted = output<CreateTransactionInput>();

  /** Emitted when the user cancels. */
  readonly cancelled = output<void>();

  protected readonly isSubmitting = signal(false);
  protected readonly typeOptions = TRANSACTION_TYPE_OPTIONS;
  protected readonly today = new Date();

  protected readonly form = this.fb.nonNullable.group({
    transaction_type: ['buy' as TransactionType, Validators.required],
    symbol: ['', [Validators.required, Validators.pattern(/^[A-Z0-9.-]{1,20}$/)]],
    transaction_date: [null as Date | null, Validators.required],
    quantity: [null as number | null],
    price_per_share: [null as number | null],
    dividend_per_share: [null as number | null],
    total_amount: [null as number | null, Validators.required],
    notes: ['', Validators.maxLength(1000)],
  });

  private typeSubscription?: Subscription;

  /** Whether the current type requires the quantity field. */
  protected get showQuantity(): boolean {
    const t = this.form.controls.transaction_type.value;
    return t === 'buy' || t === 'sell' || t === 'reinvested_dividend';
  }

  /** Whether the current type requires price_per_share. */
  protected get showPricePerShare(): boolean {
    const t = this.form.controls.transaction_type.value;
    return t === 'buy' || t === 'sell' || t === 'reinvested_dividend';
  }

  /** Whether the current type requires dividend_per_share. */
  protected get showDividendPerShare(): boolean {
    const t = this.form.controls.transaction_type.value;
    return t === 'dividend' || t === 'reinvested_dividend';
  }

  ngOnInit(): void {
    this.applyValidators(this.form.controls.transaction_type.value);
    this.typeSubscription = this.form.controls.transaction_type.valueChanges.subscribe(
      (type) => this.applyValidators(type),
    );
  }

  ngOnDestroy(): void {
    this.typeSubscription?.unsubscribe();
  }

  /**
   * Applies the correct required validators for the selected transaction type.
   * Clearing validators on hidden fields prevents invalid state for fields
   * that are not applicable to the current type.
   */
  private applyValidators(type: TransactionType): void {
    const { quantity, price_per_share, dividend_per_share } = this.form.controls;

    const setRequired = (ctrl: AbstractControl, required: boolean) => {
      if (required) {
        ctrl.setValidators([Validators.required, Validators.min(0.000001)]);
      } else {
        ctrl.clearValidators();
        ctrl.setValue(null);
      }
      ctrl.updateValueAndValidity({ emitEvent: false });
    };

    setRequired(quantity, type === 'buy' || type === 'sell' || type === 'reinvested_dividend');
    setRequired(price_per_share, type === 'buy' || type === 'sell' || type === 'reinvested_dividend');
    setRequired(dividend_per_share, type === 'dividend' || type === 'reinvested_dividend');
  }

  /** Uppercases the symbol input on blur to match backend validation. */
  protected onSymbolBlur(): void {
    const ctrl = this.form.controls.symbol;
    ctrl.setValue(ctrl.value.toUpperCase(), { emitEvent: false });
  }

  onSubmit(): void {
    if (this.form.invalid || this.isSubmitting()) return;

    this.isSubmitting.set(true);
    const raw = this.form.getRawValue();

    // Format the date as YYYY-MM-DD for the backend.
    const d = raw.transaction_date as Date;
    const dateStr = d.toISOString().split('T')[0];

    const input: CreateTransactionInput = {
      transaction_type: raw.transaction_type,
      symbol: raw.symbol,
      transaction_date: dateStr,
      total_amount: raw.total_amount!.toFixed(2),
      notes: raw.notes || undefined,
    };

    if (raw.quantity != null) input.quantity = raw.quantity.toString();
    if (raw.price_per_share != null) input.price_per_share = raw.price_per_share.toFixed(2);
    if (raw.dividend_per_share != null) input.dividend_per_share = raw.dividend_per_share.toFixed(4);

    this.isSubmitting.set(false);
    this.submitted.emit(input);
  }

  onCancel(): void {
    this.cancelled.emit();
  }
}
