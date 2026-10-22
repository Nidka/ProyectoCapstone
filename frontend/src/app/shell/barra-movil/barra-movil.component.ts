import { ChangeDetectionStrategy, Component, ElementRef, computed, inject, viewChild } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { rutaModulo, seccion } from '../../core/navigation/navegacion';
import { urlActual } from '../../core/navigation/url-actual';
import { AuthService } from '../../core/services/auth.service';
import { GastosStore } from '../../core/state/gastos.store';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { iniciales } from '../menu-usuario/menu-usuario.component';

/** Barra inferior del celular: Inicio, Gastos, Operación y Más (Reportes, Catálogos y cuenta). */
@Component({
  selector: 'app-barra-movil',
  imports: [RouterLink, RouterLinkActive, IconoComponent],
  templateUrl: './barra-movil.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BarraMovilComponent {
  protected readonly auth = inject(AuthService);
  private readonly store = inject(GastosStore);
  private readonly url = urlActual();
  private readonly hoja = viewChild.required<ElementRef<HTMLDialogElement>>('hojaMas');

  protected readonly pendientes = computed(() => this.store.pendientes().length);
  protected readonly catalogos = seccion('catalogos').modulos;
  /** «Más» se marca activo en las secciones que viven dentro de la hoja */
  protected readonly enMas = computed(() => ['/app/catalogos', '/app/reportes'].some(r => this.url().startsWith(r)));
  protected readonly nombre = computed(() => this.auth.usuario()?.nombre ?? 'Usuario');
  protected readonly siglas = computed(() => iniciales(this.nombre()));
  protected readonly rutaModulo = rutaModulo;

  protected abrirMas(): void {
    this.hoja().nativeElement.showModal();
  }

  protected cerrarMas(): void {
    this.hoja().nativeElement.close();
  }

  protected cerrarSiFondo(evento: MouseEvent): void {
    if (evento.target === this.hoja().nativeElement) this.cerrarMas();
  }

  protected salir(): void {
    this.cerrarMas();
    this.auth.logout();
  }
}
