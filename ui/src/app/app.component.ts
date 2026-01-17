import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { AuthService } from './core/services/auth.service';

@Component({
  selector: 'app-root',
  imports: [CommonModule, RouterOutlet, MatButtonModule, MatIconModule, MatMenuModule],
  templateUrl: './app.component.html',
  styleUrl: './app.component.scss'
})
export class AppComponent {
  title = 'ui';

  constructor(private authService: AuthService) {}

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
