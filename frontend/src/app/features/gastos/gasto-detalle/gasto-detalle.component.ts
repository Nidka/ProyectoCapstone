import { Location } from '@angular/common';
import {
  ChangeDetectionStrategy, Component, OnInit, afterRenderEffect, computed, effect, inject, input, signal, untracked,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { Comprobante } from '../../../core/models/gasto.model';
import { ESTADO_GASTO_LABEL } from '../../../core/models/gasto-estados';
import { CrudService } from '../../../core/services/crud.service';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { esPestana, pasosRuta, ultimoRechazo } from '../../../core/state/gastos.logic';
import { GastosStore } from '../../../core/state/gastos.store';
import { formatFecha, formatFechaHora, formatMonto } from '../../../core/utils/format';
import { EstadoVacioComponent } from '../../../shared/ui/estado-vacio/estado-vacio.component';
import { HitoComponent } from '../../../shared/ui/hito/hito.component';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';
import { InsigniaEstadoComponent } from '../../../shared/ui/insignia-estado/insignia-estado.component';
import { RutaGastoComponent } from '../../../shared/ui/ruta-gasto/ruta-gasto.component';
import { AdjuntarComprobanteComponent } from '../adjuntar-comprobante/adjuntar-comprobante.component';
import { RevisionComprobanteComponent } from '../revision-comprobante/revision-comprobante.component';

/** La bandeja pasa { origen } en el state de la navegación; «← Gastos» vuelve a esa pestaña. */
export function consultaDeOrigen(estado: unknown): Record<string, string> {
  const origen = (estado as { origen?: unknown } | null)?.origen;
  return esPestana(origen) && origen !== 'todos' ? { estado: origen } : {};
}

@Component({
  selector: 'app-gasto-detalle',
  imports: [
    RouterLink, IconoComponent, InsigniaEstadoComponent, RutaGastoComponent, EstadoVacioComponent, HitoComponent,
    AdjuntarComprobanteComponent, RevisionComprobanteComponent,
  ],
  templateUrl: './gasto-detalle.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GastoDetalleComponent implements OnInit {
  /** :id de la ruta */
  readonly id = input.required<string>();

  protected readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);
  private readonly gastosApi = inject(GastosApiService);
  private readonly ruta = inject(ActivatedRoute);
  private readonly location = inject(Location);
  /** Gasto al que ya se llevó la vista con #paso (el componente se reutiliza al pasar de un gasto a otro) */
  private desplazadoA: number | null = null;
  /** Gasto que no vino en la lista y se pidió por su número */
  private readonly busqueda = signal<{ id: number; estado: 'buscando' | 'encontrado' | 'no-existe' | 'error' } | null>(null);

  protected readonly volverQuery = computed(() => {
    this.id();
    return consultaDeOrigen(this.location.getState());
  });
  protected readonly gasto = computed(() => this.store.gasto(Number(this.id())));
  protected readonly comprobantes = computed(() => this.store.comprobantesDe(Number(this.id())));
  protected readonly pasos = computed(() => {
    const g = this.gasto();
    return g ? pasosRuta(g, this.store.comprobantes()) : [];
  });
  protected readonly pendiente = computed(() => this.comprobantes().find(c => c.estado === 'Pendiente'));
  protected readonly aprobado = computed(() => this.comprobantes().find(c => c.estado === 'Aprobado'));
  protected readonly rechazoReciente = computed(() => this.comprobantes().find(c => c.estado === 'Rechazado'));
  protected readonly rechazoPrevio = computed(() => {
    const g = this.gasto();
    return g?.estado === 'Registrado' ? ultimoRechazo(this.store.comprobantes(), g.id) : undefined;
  });
  protected readonly cliente = computed(() => {
    const g = this.gasto();
    return g ? this.crud.etiqueta('cliente', g.cliente_id) : '';
  });
  protected readonly viaje = computed(() => {
    const g = this.gasto();
    return g ? this.crud.etiqueta('viaje', g.viaje_id) : '';
  });
  private readonly estadoBusqueda = computed(() => {
    const b = this.busqueda();
    return b && b.id === Number(this.id()) ? b.estado : null;
  });
  protected readonly buscando = computed(() => this.estadoBusqueda() === 'buscando');
  /** El servicio no respondió: se dice así, no con un esqueleto eterno ni con «No encontramos…» */
  protected readonly sinServicio = computed(
    () => (this.store.error() && !this.store.cargado()) || this.estadoBusqueda() === 'error',
  );
  protected readonly etiquetaEstado = ESTADO_GASTO_LABEL;
  protected readonly formatMonto = formatMonto;
  protected readonly formatFecha = formatFecha;
  protected readonly formatFechaHora = formatFechaHora;

  constructor() {
    // Al llegar con #paso (desde la bandeja o tras registrar), lleva la vista al paso que toca.
    afterRenderEffect(() => {
      const g = this.gasto();
      if (!g || this.desplazadoA === g.id || this.ruta.snapshot.fragment !== 'paso') return;
      this.desplazadoA = g.id;
      document.getElementById('paso')?.scrollIntoView?.({ block: 'start' });
    });
    // Un gasto que no está entre los 200 más recientes se pide por su número antes de decir que no existe.
    effect(() => {
      const id = Number(this.id());
      if (!this.store.cargado() || this.gasto() || this.busqueda()?.id === id) return;
      untracked(() => this.buscar(id));
    });
  }

  protected reintentar(): void {
    if (!this.store.cargado()) this.store.cargar();
    else this.buscar(Number(this.id()));
  }

  private buscar(id: number): void {
    this.busqueda.set({ id, estado: 'buscando' });
    this.store.cargarGasto(id).subscribe({
      next: g => this.busqueda.set({ id, estado: g ? 'encontrado' : 'no-existe' }),
      error: () => this.busqueda.set({ id, estado: 'error' }),
    });
  }

  ngOnInit(): void {
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    this.crud.asegurarCargado('viaje');
  }

  protected archivo(c: Comprobante): string {
    return this.gastosApi.fileUrl(c.archivo_url);
  }
}
