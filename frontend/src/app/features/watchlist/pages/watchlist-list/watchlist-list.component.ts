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
import { InputText } from 'primeng/inputtext';
import { TooltipModule } from 'primeng/tooltip';
import { FormsModule } from '@angular/forms';
import { WatchlistService } from '../../services/watchlist.service';
import type { Watchlist } from '../../models/watchlist.model';

/**
 * WatchlistListComponent displays all watchlists for the authenticated user.
 *
 * Provides create, rename, and delete actions.
 * Clicking a watchlist name navigates to the watchlist detail view.
 */
@Component({
  selector: 'app-watchlist-list',
  standalone: true,
  imports: [DatePipe, Button, TableModule, Dialog, ConfirmDialog, InputText, TooltipModule, FormsModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [ConfirmationService],
  templateUrl: './watchlist-list.component.html',
})
export class WatchlistListComponent implements OnInit {
  protected readonly watchlistService = inject(WatchlistService);
  private readonly router = inject(Router);
  private readonly messages = inject(MessageService);
  private readonly confirmation = inject(ConfirmationService);

  /** Controls create/edit dialog visibility. */
  protected readonly dialogVisible = signal(false);

  /** The watchlist being edited; null when creating. */
  protected readonly editTarget = signal<Watchlist | null>(null);

  /** New watchlist name input. */
  protected readonly newName = signal('');

  ngOnInit(): void {
    // Load watchlists silently; empty state template handles the no-watchlist case.
    this.watchlistService.loadAll().subscribe();
  }

  protected openCreateDialog(): void {
    this.editTarget.set(null);
    this.newName.set('');
    this.dialogVisible.set(true);
  }

  protected openEditDialog(watchlist: Watchlist): void {
    this.editTarget.set(watchlist);
    this.newName.set(watchlist.name);
    this.dialogVisible.set(true);
  }

  protected onSave(): void {
    const target = this.editTarget();
    const name = this.newName().trim();

    if (!name) {
      this.messages.add({
        severity: 'warn',
        summary: 'Invalid input',
        detail: 'Please enter a watchlist name.',
      });
      return;
    }

    if (target) {
      // Update existing watchlist
      this.watchlistService.update(target.id, { name }).subscribe({
        next: () => {
          this.messages.add({
            severity: 'success',
            summary: 'Updated',
            detail: `"${name}" was renamed.`,
          });
          this.dialogVisible.set(false);
        },
        error: () => {
          this.messages.add({
            severity: 'error',
            summary: 'Update failed',
            detail: 'Could not update the watchlist.',
          });
        },
      });
    } else {
      // Create new watchlist
      this.watchlistService.create({ name }).subscribe({
        next: () => {
          this.messages.add({
            severity: 'success',
            summary: 'Created',
            detail: `Watchlist "${name}" was created.`,
          });
          this.dialogVisible.set(false);
        },
        error: () => {
          this.messages.add({
            severity: 'error',
            summary: 'Create failed',
            detail: 'Could not create the watchlist.',
          });
        },
      });
    }
  }

  protected onCancel(): void {
    this.dialogVisible.set(false);
  }

  protected confirmDelete(watchlist: Watchlist): void {
    this.confirmation.confirm({
      message: `Delete "${watchlist.name}"? This cannot be undone.`,
      header: 'Confirm deletion',
      icon: 'pi pi-exclamation-triangle',
      acceptButtonStyleClass: 'p-button-danger',
      accept: () => this.deleteWatchlist(watchlist),
    });
  }

  private deleteWatchlist(watchlist: Watchlist): void {
    this.watchlistService.delete(watchlist.id).subscribe({
      next: () =>
        this.messages.add({
          severity: 'success',
          summary: 'Deleted',
          detail: `"${watchlist.name}" was removed.`,
        }),
      error: () =>
        this.messages.add({
          severity: 'error',
          summary: 'Delete failed',
          detail: 'Could not delete the watchlist.',
        }),
    });
  }

  protected navigateToDetail(watchlistId: string): void {
    this.router.navigate(['/watchlists', watchlistId]);
  }
}
