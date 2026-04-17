import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { Toast } from 'primeng/toast';
import { IdleTimeoutService } from './core/idle-timeout.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, Toast],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p-toast />
    <router-outlet />
  `,
})
export class App {
  // Eagerly instantiate the idle timeout service to start monitoring user activity.
  private readonly _idleTimeout = inject(IdleTimeoutService);
}
