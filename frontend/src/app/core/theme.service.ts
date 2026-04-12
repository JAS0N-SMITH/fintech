import { Injectable, effect, signal } from '@angular/core';

/**
 * ThemeService manages application theme state (light/dark mode).
 * Theme preference is persisted to localStorage and reflected on the document element.
 */
@Injectable({
  providedIn: 'root',
})
export class ThemeService {
  private readonly THEME_STORAGE_KEY = 'app-theme';

  readonly isDark = signal(this.loadThemePreference());

  constructor() {
    // Side effect: update DOM class and localStorage when theme changes
    effect(() => {
      const dark = this.isDark();
      if (dark) {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
      localStorage.setItem(this.THEME_STORAGE_KEY, dark ? 'dark' : 'light');
    });
  }

  /**
   * Toggle between light and dark theme.
   */
  toggle(): void {
    this.isDark.update(v => !v);
  }

  /**
   * Set theme explicitly.
   */
  setDark(dark: boolean): void {
    this.isDark.set(dark);
  }

  private loadThemePreference(): boolean {
    const stored = localStorage.getItem(this.THEME_STORAGE_KEY);
    if (stored) {
      return stored === 'dark';
    }
    // Default to light mode
    return false;
  }
}
