import { TestBed, ComponentFixture } from '@angular/core/testing';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { TransactionFormComponent } from './transaction-form.component';
import { provideHttpClient } from '@angular/common/http';

// Helper: create a fresh fixture with TestBed
function setup(): ComponentFixture<TransactionFormComponent> {
  TestBed.configureTestingModule({
    imports: [TransactionFormComponent],
    providers: [provideHttpClient(), provideAnimationsAsync()],
  });
  const fixture = TestBed.createComponent(TransactionFormComponent);
  fixture.detectChanges(); // triggers ngOnInit → applyValidators('buy')
  return fixture;
}

describe('TransactionFormComponent — field validation by type', () => {
  it('buy: quantity and price_per_share are required; dividend_per_share is not', () => {
    const fixture = setup();
    const ctrl = (fixture.componentInstance as unknown as { form: { controls: Record<string, import('@angular/forms').AbstractControl> } }).form.controls;

    // Default type is 'buy'
    expect(ctrl['quantity'].invalid).toBe(true); // no value yet → required
    expect(ctrl['price_per_share'].invalid).toBe(true);
    expect(ctrl['dividend_per_share'].invalid).toBe(false); // not required for buy
  });

  it('sell: quantity and price_per_share required', () => {
    const fixture = setup();
    const ctrl = (fixture.componentInstance as unknown as { form: { controls: Record<string, import('@angular/forms').AbstractControl> } }).form.controls;

    ctrl['transaction_type'].setValue('sell');
    fixture.detectChanges();

    expect(ctrl['quantity'].invalid).toBe(true);
    expect(ctrl['price_per_share'].invalid).toBe(true);
    expect(ctrl['dividend_per_share'].invalid).toBe(false);
  });

  it('dividend: dividend_per_share required; quantity and price_per_share not required', () => {
    const fixture = setup();
    const ctrl = (fixture.componentInstance as unknown as { form: { controls: Record<string, import('@angular/forms').AbstractControl> } }).form.controls;

    ctrl['transaction_type'].setValue('dividend');
    fixture.detectChanges();

    expect(ctrl['quantity'].invalid).toBe(false);       // cleared for dividend
    expect(ctrl['price_per_share'].invalid).toBe(false); // cleared for dividend
    expect(ctrl['dividend_per_share'].invalid).toBe(true); // required for dividend
  });

  it('reinvested_dividend: quantity, price_per_share, and dividend_per_share all required', () => {
    const fixture = setup();
    const ctrl = (fixture.componentInstance as unknown as { form: { controls: Record<string, import('@angular/forms').AbstractControl> } }).form.controls;

    ctrl['transaction_type'].setValue('reinvested_dividend');
    fixture.detectChanges();

    expect(ctrl['quantity'].invalid).toBe(true);
    expect(ctrl['price_per_share'].invalid).toBe(true);
    expect(ctrl['dividend_per_share'].invalid).toBe(true);
  });

  it('buy with all required fields filled → form is valid', () => {
    const fixture = setup();
    const { form } = fixture.componentInstance as unknown as { form: import('@angular/forms').FormGroup };

    form.patchValue({
      transaction_type: 'buy',
      symbol: 'AAPL',
      transaction_date: new Date('2026-01-15'),
      quantity: 10,
      price_per_share: 150,
      total_amount: 1500,
    });
    fixture.detectChanges();

    expect(form.valid).toBe(true);
  });

  it('dividend with dividend_per_share and total_amount filled → form is valid', () => {
    const fixture = setup();
    const { form } = fixture.componentInstance as unknown as { form: import('@angular/forms').FormGroup };

    form.patchValue({
      transaction_type: 'dividend',
      symbol: 'AAPL',
      transaction_date: new Date('2026-01-15'),
      dividend_per_share: 0.25,
      total_amount: 25,
    });
    fixture.detectChanges();

    expect(form.valid).toBe(true);
  });

  it('emits submitted with formatted output on valid buy submit', () => {
    const fixture = setup();
    const component = fixture.componentInstance;
    const { form } = component as unknown as { form: import('@angular/forms').FormGroup };

    const emitted: unknown[] = [];
    component.submitted.subscribe((v: unknown) => emitted.push(v));

    form.patchValue({
      transaction_type: 'buy',
      symbol: 'AAPL',
      transaction_date: new Date('2026-01-15T00:00:00'),
      quantity: 10,
      price_per_share: 150,
      total_amount: 1500,
    });
    fixture.detectChanges();

    component.onSubmit();

    expect(emitted.length).toBe(1);
    const result = emitted[0] as Record<string, unknown>;
    expect(result['transaction_type']).toBe('buy');
    expect(result['total_amount']).toBe('1500.00');
    expect(result['quantity']).toBe('10');
    expect(result['price_per_share']).toBe('150.00');
  });

  it('does not emit if form is invalid', () => {
    const fixture = setup();
    const component = fixture.componentInstance;

    const emitted: unknown[] = [];
    component.submitted.subscribe((v: unknown) => emitted.push(v));

    // Form is invalid by default (required fields empty)
    component.onSubmit();

    expect(emitted.length).toBe(0);
  });

  it('emits cancelled when onCancel is called', () => {
    const fixture = setup();
    const component = fixture.componentInstance;
    let cancelled = false;
    component.cancelled.subscribe(() => (cancelled = true));

    component.onCancel();
    expect(cancelled).toBe(true);
  });
});
