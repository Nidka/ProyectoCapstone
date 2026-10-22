import {
  ChangeDetectionStrategy, Component, ElementRef, afterRenderEffect, computed, inject, signal, viewChild,
} from '@angular/core';
import { Router } from '@angular/router';
import { EntradaPaleta, GrupoPaleta } from '../../core/navigation/navegacion';
import { CrudService } from '../../core/services/crud.service';
import { GastosStore } from '../../core/state/gastos.store';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { resultadosPaleta } from './paleta.logic';

const GRUPOS: readonly GrupoPaleta[] = ['Acciones', 'Ir a', 'Gastos'];

/** Buscar o ir a… (⌘K / Ctrl+K). Sin animación: se usa muchas veces al día. */
@Component({
  selector: 'app-paleta-comandos',
  imports: [IconoComponent],
  templateUrl: './paleta-comandos.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PaletaComandosComponent {
  private readonly router = inject(Router);
  private readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);
  private readonly dialogo = viewChild.required<ElementRef<HTMLDialogElement>>('dialogo');
  private readonly campo = viewChild.required<ElementRef<HTMLInputElement>>('campo');

  protected readonly termino = signal('');
  protected readonly activo = signal(0);
  protected readonly resultados = computed(() =>
    resultadosPaleta(this.termino(), this.store.gastos(), id => this.crud.etiqueta('cliente', id)),
  );
  protected readonly grupos = computed(() => {
    const conIndice = this.resultados().map((entrada, indice) => ({ entrada, indice }));
    return GRUPOS.map(nombre => ({ nombre, entradas: conIndice.filter(x => x.entrada.grupo === nombre) })).filter(
      g => g.entradas.length,
    );
  });
  protected readonly activoId = computed(() => this.resultados()[this.activo()]?.id ?? null);

  constructor() {
    afterRenderEffect(() => {
      const id = this.activoId();
      if (id) document.getElementById(id)?.scrollIntoView?.({ block: 'nearest' });
    });
  }

  abrir(): void {
    this.termino.set('');
    this.activo.set(0);
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    const dialogo = this.dialogo().nativeElement;
    if (!dialogo.open) dialogo.showModal();
    const campo = this.campo().nativeElement;
    campo.value = '';
    campo.focus();
  }

  protected cerrar(): void {
    this.dialogo().nativeElement.close();
  }

  protected cerrarSiFondo(evento: MouseEvent): void {
    if (evento.target === this.dialogo().nativeElement) this.cerrar();
  }

  protected escribir(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
    this.activo.set(0);
  }

  protected ir(entrada: EntradaPaleta): void {
    this.cerrar();
    this.router.navigate([entrada.ruta], { queryParams: entrada.queryParams });
  }

  protected teclear(evento: KeyboardEvent): void {
    const total = this.resultados().length;
    if (evento.key === 'ArrowDown') {
      evento.preventDefault();
      this.activo.set(total ? (this.activo() + 1) % total : 0);
    } else if (evento.key === 'ArrowUp') {
      evento.preventDefault();
      this.activo.set(total ? (this.activo() - 1 + total) % total : 0);
    } else if (evento.key === 'Enter') {
      const entrada = this.resultados()[this.activo()];
      if (entrada) {
        evento.preventDefault();
        this.ir(entrada);
      }
    }
  }
}
