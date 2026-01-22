import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { MenuModule } from 'primeng/menu';
import { MenuItem } from 'primeng/api';
import { AuthService } from './core/services/auth.service';

@Component({
  selector: 'app-root',
  imports: [CommonModule, RouterOutlet, ButtonModule, MenuModule],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent {
  title = 'ui';
  menuItems: MenuItem[] = [];

  constructor(private authService: AuthService) {
    this.menuItems = [
      {
        label: 'Logout',
        icon: 'pi pi-sign-out',
        command: () => this.logout()
      }
    ];
  }

  get isAuthenticated(): boolean {
    return this.authService.hasValidAccessToken();
  }

  get accountEmail(): string | undefined {
    return this.authService.email;
  }

  logout(): void {
    this.authService.logout().subscribe();
  }
}
