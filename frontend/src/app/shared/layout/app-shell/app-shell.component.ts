import { Component, OnInit, signal, computed, ChangeDetectionStrategy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { MenuModule } from 'primeng/menu';
import { ButtonModule } from 'primeng/button';
import { AvatarModule } from 'primeng/avatar';
import { TooltipModule } from 'primeng/tooltip';
import { MenuItem } from 'primeng/api';

import { AuthService } from '../../../core/auth.service';
import { ThemeService } from '../../../core/theme.service';

/**
 * AppShellComponent is the main layout shell with persistent sidebar navigation.
 * Contains the main navigation menu, user profile, theme toggle, and route outlet.
 */
@Component({
  selector: 'app-shell',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    MenuModule,
    ButtonModule,
    AvatarModule,
    TooltipModule,
  ],
  templateUrl: './app-shell.component.html',
  styleUrl: './app-shell.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppShellComponent implements OnInit {
  private readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  readonly themeService = inject(ThemeService);

  readonly isSidebarCollapsed = signal(false);
  readonly currentUrl = signal(this.router.url);

  readonly user = this.authService.user;

  menuItems = signal<MenuItem[]>([]);

  readonly isDark = this.themeService.isDark;

  ngOnInit(): void {
    this.buildMenuItems();
    // Update current URL when navigation changes
    this.router.events.subscribe(() => {
      this.currentUrl.set(this.router.url);
    });
  }

  toggleSidebar(): void {
    this.isSidebarCollapsed.update(v => !v);
  }

  toggleTheme(): void {
    this.themeService.toggle();
  }

  logout(): void {
    this.authService.signOut();
  }

  private buildMenuItems(): void {
    const user = this.user();
    const isAdmin = user?.role === 'admin';

    const items: MenuItem[] = [
      {
        label: 'Dashboard',
        icon: 'pi pi-fw pi-home',
        routerLink: '/dashboard',
      },
      {
        label: 'Portfolios',
        icon: 'pi pi-fw pi-briefcase',
        routerLink: '/dashboard/portfolios',
      },
      {
        label: 'Watchlists',
        icon: 'pi pi-fw pi-star',
        routerLink: '/dashboard/watchlists',
      },
    ];

    if (isAdmin) {
      items.push({
        separator: true,
      });
      items.push({
        label: 'Admin',
        icon: 'pi pi-fw pi-cog',
        routerLink: '/admin',
      });
    }

    this.menuItems.set(items);
  }
}
