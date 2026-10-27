import { ChangeDetectionStrategy, Component, computed, inject, output } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { ENLACES_PRINCIPALES, visiblesPara } from '../../core/navigation/navegacion';
import { urlActual } from '../../core/navigation/url-actual';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { GastosStore } from '../../core/state/gastos.store';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { CampanaComponent } from '../campana/campana.component';
import { MenuUsuarioComponent } from '../menu-usuario/menu-usuario.component';

@Component({
  selector: 'app-barra-superior',
  imports: [RouterLink, RouterLinkActive, MarcaComponent, IconoComponent, CampanaComponent, MenuUsuarioComponent],
  templateUrl: './barra-superior.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BarraSuperiorComponent {
  readonly buscar = output<void>();

  protected readonly api = inject(ApiService);
  protected readonly auth = inject(AuthService);
  private readonly store = inject(GastosStore);
  private readonly url = urlActual();

  protected readonly enlaces = computed(() => visiblesPara(ENLACES_PRINCIPALES, this.auth.usuario()?.rol));
  protected readonly pendientes = computed(() => this.store.pendientes().length);
  protected readonly atajo = /Mac|iPhone|iPad/.test(navigator.userAgent) ? '⌘K' : 'Ctrl K';
  protected readonly titulo = computed(() => ENLACES_PRINCIPALES.find(e => this.url().startsWith(e.ruta))?.etiqueta ?? '');
  /** Solo se muestra cuando la verificación terminó y el servicio no responde. */
  protected readonly servicioCaido = computed(() => {
    const h = this.api.health();
    return !h.connected && h.env !== '…';
  });
}
