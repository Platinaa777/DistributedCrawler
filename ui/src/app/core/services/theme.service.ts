import { Injectable, signal, effect, PLATFORM_ID, inject } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';

export type Theme = 'light' | 'dark';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  private readonly STORAGE_KEY = 'theme-preference';
  private readonly platformId = inject(PLATFORM_ID);
  private darkModeSuppressCount = 0;

  readonly theme = signal<Theme>(this.getInitialTheme());

  constructor() {
    effect(() => {
      this.applyTheme(this.theme());
    });
  }

  private getInitialTheme(): Theme {
    if (!isPlatformBrowser(this.platformId)) {
      return 'light';
    }

    const stored = localStorage.getItem(this.STORAGE_KEY) as Theme | null;
    if (stored === 'light' || stored === 'dark') {
      return stored;
    }

    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }

  private applyTheme(theme: Theme): void {
    if (!isPlatformBrowser(this.platformId)) {
      return;
    }

    const root = document.documentElement;
    if (theme === 'dark' && this.darkModeSuppressCount === 0) {
      root.classList.add('dark-mode');
    } else {
      root.classList.remove('dark-mode');
    }
    localStorage.setItem(this.STORAGE_KEY, theme);
  }

  toggle(): void {
    this.theme.update(current => current === 'dark' ? 'light' : 'dark');
  }

  setTheme(theme: Theme): void {
    this.theme.set(theme);
  }

  suppressDarkMode(): () => void {
    if (!isPlatformBrowser(this.platformId)) {
      return () => undefined;
    }

    this.darkModeSuppressCount += 1;
    if (this.darkModeSuppressCount === 1) {
      document.documentElement.classList.remove('dark-mode');
    }

    return () => {
      if (!isPlatformBrowser(this.platformId)) {
        return;
      }

      if (this.darkModeSuppressCount > 0) {
        this.darkModeSuppressCount -= 1;
      }

      if (this.darkModeSuppressCount === 0) {
        this.applyTheme(this.theme());
      }
    };
  }

  get isDark(): boolean {
    return this.theme() === 'dark';
  }
}
