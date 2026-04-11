import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { DatePipe } from '@angular/common';
import { Router } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { Button } from 'primeng/button';
import { TableModule } from 'primeng/table';
import { Dialog } from 'primeng/dialog';
import { ConfirmDialog } from 'primeng/confirmdialog';
import { PortfolioService } from '../../services/portfolio.service';
import { PortfolioFormComponent } from '../../components/portfolio-form/portfolio-form.component';
import type { Portfolio } from '../../models/portfolio.model';

/**
 * PortfolioListComponent displays all portfolios for the authenticated user.
 *
 * Provides create, edit, and delete actions.
 * Clicking a portfolio name navigates to the portfolio detail view.
 */
@Component({
  selector: 'app-portfolio-list',
  standalone: true,
  imports: [DatePipe, Button, TableModule, Dialog, ConfirmDialog, PortfolioFormComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [ConfirmationService],
  templateUrl: './portfolio-list.component.html',
})
export class PortfolioListComponent implements OnInit {
  protected readonly portfolioService = inject(PortfolioService);
  private readonly router = inject(Router);
  private readonly messages = inject(MessageService);
  private readonly confirmation = inject(ConfirmationService);

  /** Controls create/edit dialog visibility. */
  protected readonly dialogVisible = signal(false);

  /** The portfolio being edited; null when creating. */
  protected readonly editTarget = signal<Portfolio | null>(null);

  ngOnInit(): void {
    // Load portfolios silently; empty state template handles the no-portfolio case.
    // Errors are logged to console for debugging, but don't disrupt the user experience.
    this.portfolioService.loadAll().subscribe();
  }

  protected openCreateDialog(): void {
    this.editTarget.set(null);
    this.dialogVisible.set(true);
  }

  protected openEditDialog(portfolio: Portfolio): void {
    this.editTarget.set(portfolio);
    this.dialogVisible.set(true);
  }

  protected onFormSaved(): void {
    this.dialogVisible.set(false);
  }

  protected onFormCancelled(): void {
    this.dialogVisible.set(false);
  }

  protected confirmDelete(portfolio: Portfolio): void {
    this.confirmation.confirm({
      message: `Delete "${portfolio.name}"? This cannot be undone.`,
      header: 'Confirm deletion',
      icon: 'pi pi-exclamation-triangle',
      acceptButtonStyleClass: 'p-button-danger',
      accept: () => this.deletePortfolio(portfolio),
    });
  }

  private deletePortfolio(portfolio: Portfolio): void {
    this.portfolioService.delete(portfolio.id).subscribe({
      next: () =>
        this.messages.add({
          severity: 'success',
          summary: 'Deleted',
          detail: `"${portfolio.name}" was removed.`,
        }),
      error: () =>
        this.messages.add({
          severity: 'error',
          summary: 'Delete failed',
          detail: 'Could not delete the portfolio.',
        }),
    });
  }

  protected navigateToDetail(portfolioId: string): void {
    this.router.navigate(['/portfolios', portfolioId]);
  }
}
