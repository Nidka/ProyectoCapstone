import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { PanelAccesoComponent } from '../../shared/ui/panel-acceso/panel-acceso.component';

@Component({
  selector: 'app-register',
  imports: [FormsModule, RouterLink, MarcaComponent, IconoComponent, PanelAccesoComponent],
  templateUrl: './register.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RegisterComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected username = '';
  protected email = '';
  protected nombre = '';
  protected rol: 'admin' | 'operador' = 'operador';
  protected password = '';
  protected confirmPassword = '';

  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly showPass = signal(false);

  protected submit(): void {
    if (!this.username.trim() || !this.email.trim() || !this.nombre.trim() || !this.password) {
      this.error.set('Completa todos los campos obligatorios');
      return;
    }
    if (this.password.length < 6) {
      this.error.set('La contraseña debe tener al menos 6 caracteres');
      return;
    }
    if (this.password !== this.confirmPassword) {
      this.error.set('Las contraseñas no coinciden');
      return;
    }

    this.error.set('');
    this.loading.set(true);
    this.auth
      .register({ username: this.username.trim(), email: this.email.trim(), nombre: this.nombre.trim(), rol: this.rol, password: this.password })
      .subscribe({
        next: () => this.router.navigate(['/login'], { queryParams: { registered: '1' } }),
        error: e => {
          this.error.set(e?.error?.detail ?? 'No se pudo crear el usuario');
          this.loading.set(false);
        },
      });
  }
}
