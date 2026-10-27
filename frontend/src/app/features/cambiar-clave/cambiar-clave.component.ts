import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { CampoCambioClave, ErroresCambioClave, MIN_CLAVE, erroresDeCambioClave, validarCambioClave } from '../../core/models/usuario';
import { destinoTrasLogin } from '../../core/navigation/navegacion';
import { AuthService } from '../../core/services/auth.service';
import { ToastService } from '../../core/services/toast.service';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { PanelAccesoComponent } from '../../shared/ui/panel-acceso/panel-acceso.component';

/**
 * Cambio de contraseña. Es obligatorio (y lo único que se puede hacer) cuando el servidor marca la cuenta
 * con la clave pendiente: el login lleva aquí y `cambioClaveGuard` bloquea el resto de la app.
 */
@Component({
  selector: 'app-cambiar-clave',
  imports: [FormsModule, RouterLink, MarcaComponent, IconoComponent, PanelAccesoComponent],
  templateUrl: './cambiar-clave.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CambiarClaveComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly ruta = inject(ActivatedRoute);
  private readonly toast = inject(ToastService);

  /** Obligatorio o por gusto: cambia el texto y si se puede cancelar. Se fija al abrir la pantalla. */
  protected readonly obligatorio = this.auth.debeCambiarClave();
  protected readonly minimo = MIN_CLAVE;

  protected readonly actual = signal('');
  protected readonly nueva = signal('');
  protected readonly confirmacion = signal('');
  protected readonly mostrar = signal(false);
  protected readonly errores = signal<ErroresCambioClave>({});
  protected readonly general = signal('');
  protected readonly guardando = signal(false);

  /** Al corregir un campo se le quita su error; el resto se vuelve a revisar al enviar. */
  protected escribir(campo: CampoCambioClave, valor: string): void {
    ({ actual: this.actual, nueva: this.nueva, confirmacion: this.confirmacion })[campo].set(valor);
    if (this.errores()[campo]) this.errores.update(e => ({ ...e, [campo]: undefined }));
  }

  protected enviar(): void {
    if (this.guardando()) return;
    const valores = { actual: this.actual(), nueva: this.nueva(), confirmacion: this.confirmacion() };
    const errores = validarCambioClave(valores);
    this.errores.set(errores);
    this.general.set('');
    if (Object.keys(errores).length) return;

    this.guardando.set(true);
    this.auth.cambiarClave({ clave_actual: valores.actual, clave_nueva: valores.nueva }).subscribe({
      next: () => {
        this.toast.show('Contraseña cambiada.', '', 'ok');
        this.router.navigateByUrl(destinoTrasLogin(this.ruta.snapshot.queryParamMap.get('volver')));
      },
      error: err => {
        const { campos, general } = erroresDeCambioClave(err);
        this.errores.set(campos);
        this.general.set(general);
        this.guardando.set(false);
      },
    });
  }

  protected salir(): void {
    this.auth.logout();
  }
}
