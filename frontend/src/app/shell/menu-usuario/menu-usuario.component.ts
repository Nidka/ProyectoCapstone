import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { Router } from '@angular/router';
import { ROL_ETIQUETA } from '../../core/models/auth.model';
import { AuthService } from '../../core/services/auth.service';
import { IconoComponent } from '../../shared/ui/icono/icono.component';

/** «AQ» para «Ana Quispe»; «AD» para «admin». */
export function iniciales(nombre: string): string {
  const partes = nombre.trim().split(/\s+/).filter(Boolean);
  if (!partes.length) return 'US';
  return (partes.length > 1 ? partes[0][0] + partes[1][0] : partes[0].slice(0, 2)).toUpperCase();
}

/** Menú de cuenta con la API Popover nativa (descarte con clic afuera y Esc). */
@Component({
  selector: 'app-menu-usuario',
  imports: [IconoComponent],
  template: `
    <button type="button" class="usuario" popovertarget="menu-usuario" aria-haspopup="menu" [attr.aria-label]="'Cuenta de ' + nombre()">
      <span class="usuario__iniciales" aria-hidden="true">{{ siglas() }}</span>
      <span class="usuario__nombre">{{ nombre() }}</span>
      <app-icono class="usuario__chevron" nombre="chevron-down" [tamano]="14" />
    </button>
    <div id="menu-usuario" class="menu" popover role="menu">
      <div class="menu__cabecera">
        <span class="menu__nombre">{{ nombre() }}</span>
        @if (rol()) {
          <span class="menu__rol">{{ rol() }}</span>
        }
      </div>
      <button type="button" class="menu__item" role="menuitem" (click)="cambiarClave()">
        <app-icono nombre="lock" [tamano]="16" />Cambiar contraseña
      </button>
      <button type="button" class="menu__item" role="menuitem" (click)="salir()">
        <app-icono nombre="logout" [tamano]="16" />Cerrar sesión
      </button>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MenuUsuarioComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  protected readonly nombre = computed(() => this.auth.usuario()?.nombre ?? 'Usuario');
  protected readonly rol = computed(() => {
    const rol = this.auth.usuario()?.rol;
    return rol ? ROL_ETIQUETA[rol] ?? rol : '';
  });
  protected readonly siglas = computed(() => iniciales(this.nombre()));

  protected cambiarClave(): void {
    this.cerrarMenu();
    this.router.navigate(['/cambiar-clave']);
  }

  protected salir(): void {
    this.cerrarMenu();
    this.auth.logout();
  }

  private cerrarMenu(): void {
    (document.getElementById('menu-usuario') as (HTMLElement & { hidePopover?: () => void }) | null)?.hidePopover?.();
  }
}
