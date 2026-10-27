import { ChangeDetectionStrategy, Component, DestroyRef, ElementRef, Injector, afterNextRender, computed, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Notificacion, textoInsignia, tituloAviso } from '../../core/models/notificacion';
import { NotificacionesStore } from '../../core/state/notificaciones.store';
import { formatFechaHora } from '../../core/utils/format';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';

/**
 * Campana de avisos de la barra superior (solo Logística): insignia con los avisos sin leer y una lista
 * que lleva al gasto de cada uno. Mientras exista, pregunta por avisos cada 5 minutos.
 */
@Component({
  selector: 'app-campana',
  imports: [RouterLink, IconoComponent, EstadoVacioComponent],
  templateUrl: './campana.component.html',
  host: {
    '(document:keydown.escape)': 'alEscape()',
    '(document:click)': 'alClicEnPagina($event)',
    '(focusout)': 'alPerderFoco($event)',
  },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CampanaComponent {
  protected readonly store = inject(NotificacionesStore);
  private readonly inicio = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;
  private readonly injector = inject(Injector);

  protected readonly abierto = signal(false);
  protected readonly noLeidas = this.store.noLeidas;
  protected readonly etiqueta = computed(() => {
    const n = this.noLeidas();
    return n > 0 ? `Avisos, ${n} sin leer` : 'Avisos, ninguno sin leer';
  });

  protected readonly textoInsignia = textoInsignia;
  protected readonly tituloAviso = tituloAviso;
  protected readonly formatFechaHora = formatFechaHora;

  constructor() {
    this.store.iniciarSondeo();
    inject(DestroyRef).onDestroy(() => this.store.detenerSondeo());
  }

  protected alternar(): void {
    if (this.abierto()) {
      this.cerrar(true);
      return;
    }
    this.abierto.set(true);
    this.store.actualizarSiViejo();
    // El diálogo aún no está en la página: el foco se pone cuando ya se dibujó
    afterNextRender(() => (this.elementos()[0] ?? this.panel())?.focus(), { injector: this.injector });
  }

  /** Pulsar un aviso lo marca como leído; el enlace del propio aviso lleva al gasto. */
  protected abrirAviso(aviso: Notificacion): void {
    if (!aviso.leida) this.store.marcarLeida(aviso.id);
    this.cerrar(false);
  }

  protected alEscape(): void {
    if (this.abierto()) this.cerrar(true);
  }

  protected alClicEnPagina(evento: MouseEvent): void {
    if (this.abierto() && !this.inicio.contains(evento.target as Node)) this.cerrar(false);
  }

  /** Salir con Tab hacia otra parte de la página cierra la lista; moverse dentro de ella, no. */
  protected alPerderFoco(evento: FocusEvent): void {
    const destino = evento.relatedTarget as Node | null;
    if (this.abierto() && destino && !this.inicio.contains(destino)) this.cerrar(false);
  }

  protected alTeclear(evento: KeyboardEvent): void {
    const lista = this.elementos();
    if (!lista.length) return;
    const actual = lista.indexOf(document.activeElement as HTMLElement);
    let destino: number;
    switch (evento.key) {
      case 'ArrowDown': destino = Math.min(actual + 1, lista.length - 1); break;
      case 'ArrowUp': destino = Math.max(actual - 1, 0); break;
      case 'Home': destino = 0; break;
      case 'End': destino = lista.length - 1; break;
      default: return;
    }
    evento.preventDefault();
    lista[destino].focus();
  }

  private cerrar(devolverFoco: boolean): void {
    this.abierto.set(false);
    if (devolverFoco) this.inicio.querySelector<HTMLElement>('.avisos__boton')?.focus();
  }

  private panel(): HTMLElement | null {
    return this.inicio.querySelector<HTMLElement>('.avisos__panel');
  }

  private elementos(): HTMLElement[] {
    return [...(this.panel()?.querySelectorAll<HTMLElement>('.avisos__item') ?? [])];
  }
}
