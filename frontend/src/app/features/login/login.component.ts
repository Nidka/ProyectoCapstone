import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { exigeCambioDeClave } from '../../core/models/auth.model';
import { destinoTrasLogin } from '../../core/navigation/navegacion';
import { AuthService } from '../../core/services/auth.service';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { PanelAccesoComponent } from '../../shared/ui/panel-acceso/panel-acceso.component';

@Component({
  selector: 'app-login',
  imports: [FormsModule, MarcaComponent, IconoComponent, PanelAccesoComponent],
  templateUrl: './login.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly ruta = inject(ActivatedRoute);

  protected username = '';
  protected password = '';
  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly showPass = signal(false);

  protected submit(): void {
    if (!this.username.trim() || !this.password) {
      this.error.set('Ingresa usuario y contraseña');
      return;
    }
    this.error.set('');
    this.loading.set(true);
    this.auth.login({ username: this.username, password: this.password }).subscribe({
      next: res => {
        const volver = destinoTrasLogin(this.ruta.snapshot.queryParamMap.get('volver'));
        // Cuenta nueva o con la clave restablecida: antes de entrar debe cambiarla
        if (exigeCambioDeClave(res)) {
          this.router.navigate(['/cambiar-clave'], volver === '/app/inicio' ? {} : { queryParams: { volver } });
        } else {
          this.router.navigateByUrl(volver);
        }
      },
      error: e => {
        this.error.set(e?.error?.detail ?? 'Credenciales incorrectas');
        this.loading.set(false);
      },
    });
  }
}
