import {
  ChangeDetectionStrategy, Component, afterRenderEffect, computed, input, model, output, signal,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { OpcionRef } from '../../../core/models/crud.model';
import { normalizar } from '../../../core/utils/texto';
import { IconoComponent } from '../icono/icono.component';

export const MAX_OPCIONES = 50;

export function coincidencias(opciones: OpcionRef[], termino: string): OpcionRef[] {
  const t = normalizar(termino).trim();
  return t ? opciones.filter(o => normalizar(o.etiqueta).includes(t)) : opciones;
}

/**
 * Elige un registro de otro módulo por su nombre. La lista se despliega dentro del flujo
 * (no flota), así no se recorta dentro de hojas con scroll. Teclado: ↑ ↓ Enter Esc.
 */
@Component({
  selector: 'app-selector-referencia',
  imports: [RouterLink, IconoComponent],
  templateUrl: './selector-referencia.component.html',
  host: { class: 'combo', '[class.combo--abierto]': 'abierto()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SelectorReferenciaComponent {
  readonly idControl = input.required<string>();
  readonly opciones = input.required<OpcionRef[]>();
  readonly valor = model('');
  readonly invalido = input(false);
  readonly describedBy = input<string | null>(null);
  readonly cargando = input(false);
  readonly placeholder = input('Escribe para buscar');
  readonly vacioTexto = input('Aún no hay registros.');
  readonly crearRuta = input<string | null>(null);
  readonly crearTexto = input('Crear registro');
  readonly creando = output<void>();

  protected readonly abierto = signal(false);
  protected readonly termino = signal('');
  protected readonly activo = signal(0);
  protected readonly todas = computed(() => coincidencias(this.opciones(), this.termino()));
  protected readonly visibles = computed(() => this.todas().slice(0, MAX_OPCIONES));
  protected readonly seleccionada = computed(() => this.opciones().find(o => o.valor === this.valor()));
  protected readonly listaId = computed(() => `${this.idControl()}-lista`);
  protected readonly texto = computed(() => (this.abierto() ? this.termino() : this.seleccionada()?.etiqueta ?? ''));
  protected readonly activoId = computed(() =>
    this.abierto() && this.visibles().length ? `${this.listaId()}-${this.activo()}` : null,
  );

  constructor() {
    afterRenderEffect(() => {
      const id = this.activoId();
      if (id) document.getElementById(id)?.scrollIntoView?.({ block: 'nearest' });
    });
  }

  protected abrir(): void {
    if (this.abierto()) return;
    this.termino.set('');
    this.activo.set(Math.max(0, this.visibles().findIndex(o => o.valor === this.valor())));
    this.abierto.set(true);
  }

  protected cerrar(): void {
    this.abierto.set(false);
    this.termino.set('');
  }

  protected escribir(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
    this.activo.set(0);
    this.abierto.set(true);
  }

  protected elegir(opcion: OpcionRef): void {
    this.valor.set(opcion.valor);
    this.cerrar();
  }

  protected teclear(evento: KeyboardEvent): void {
    const total = this.visibles().length;
    switch (evento.key) {
      case 'ArrowDown':
        evento.preventDefault();
        if (!this.abierto()) {
          this.abrir();
          return;
        }
        this.activo.set(Math.min(this.activo() + 1, Math.max(total - 1, 0)));
        break;
      case 'ArrowUp':
        evento.preventDefault();
        this.activo.set(Math.max(this.activo() - 1, 0));
        break;
      case 'Enter':
        if (this.abierto() && total) {
          evento.preventDefault();
          this.elegir(this.visibles()[this.activo()]);
        }
        break;
      case 'Escape':
        if (this.abierto()) {
          evento.preventDefault();
          evento.stopPropagation();
          this.cerrar();
        }
        break;
    }
  }
}
