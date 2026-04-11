import {
  ChangeDetectionStrategy,
  Component,
  inject,
  input,
  OnInit,
  output,
  signal,
} from '@angular/core';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MessageService } from 'primeng/api';
import { Button } from 'primeng/button';
import { InputText } from 'primeng/inputtext';
import { Textarea } from 'primeng/textarea';
import { PortfolioService } from '../../services/portfolio.service';
import type { Portfolio } from '../../models/portfolio.model';

/**
 * PortfolioFormComponent handles both create and edit modes.
 *
 * When `portfolio` input is provided the form is pre-filled and submitting
 * calls update. When absent it calls create.
 *
 * Emit `saved` with the resulting Portfolio on success so the parent can
 * close the dialog. Emit `cancelled` when the user dismisses the form.
 */
@Component({
  selector: 'app-portfolio-form',
  standalone: true,
  imports: [ReactiveFormsModule, Button, InputText, Textarea],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './portfolio-form.component.html',
})
export class PortfolioFormComponent implements OnInit {
  private readonly portfolioService = inject(PortfolioService);
  private readonly messages = inject(MessageService);
  private readonly fb = inject(FormBuilder);

  /** When provided the form operates in edit mode. */
  readonly portfolio = input<Portfolio | null>(null);

  /** Emitted with the created or updated Portfolio on success. */
  readonly saved = output<Portfolio>();

  /** Emitted when the user cancels. */
  readonly cancelled = output<void>();

  protected readonly isSubmitting = signal(false);

  protected readonly form = this.fb.nonNullable.group({
    name: ['', [Validators.required, Validators.minLength(1), Validators.maxLength(100)]],
    description: ['', [Validators.maxLength(500)]],
  });

  ngOnInit(): void {
    const p = this.portfolio();
    if (p) {
      this.form.patchValue({ name: p.name, description: p.description });
    }
  }

  protected get isEditMode(): boolean {
    return this.portfolio() !== null;
  }

  /** Submits the form, calling create or update as appropriate. */
  onSubmit(): void {
    if (this.form.invalid || this.isSubmitting()) return;

    this.isSubmitting.set(true);
    const { name, description } = this.form.getRawValue();
    const input = { name, description: description || undefined };
    const p = this.portfolio();

    const request$ = p
      ? this.portfolioService.update(p.id, input)
      : this.portfolioService.create(input);

    request$.subscribe({
      next: (result) => {
        this.isSubmitting.set(false);
        this.messages.add({
          severity: 'success',
          summary: p ? 'Portfolio updated' : 'Portfolio created',
          detail: result.name,
        });
        this.saved.emit(result);
      },
      error: (err) => {
        this.isSubmitting.set(false);
        this.messages.add({
          severity: 'error',
          summary: p ? 'Update failed' : 'Create failed',
          detail: err?.error?.detail ?? 'An unexpected error occurred.',
        });
      },
    });
  }

  onCancel(): void {
    this.cancelled.emit();
  }
}
