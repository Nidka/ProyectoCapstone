import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { CrudService } from '../../../core/services/crud.service';
import { ReportesService } from '../../../core/services/reportes.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import {
  Pestana, buscarGastos, contarPorEstado, esPestana, estadoDePestana, filtrarPorPestana, filtrarPorViajeYCliente, idDeConsulta,
  ordenarRecientes, totalMonto,
} from '../../../core/state/gastos.logic';
import { guardarArchivo } from '../../../core/utils/descarga';
import { mensajeError } from '../../../core/utils/error-messages';
import { formatMonto } from '../../../core/utils/format';
import { EstadoVacioComponent } from '../../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';
import { FilaGastoComponent } from '../fila-gasto/fila-gasto.component';

const VACIOS: Record<Pestana, { titulo: string; texto: string }> = {
  todos: { titulo: 'Todavía no hay gastos registrados.', texto: 'Presiona "Registrar gasto" para crear el primero.' },
  'sin-comprobante': { titulo: 'No hay gastos sin comprobante.', texto: 'Todos los gastos registrados ya tienen su comprobante.' },
  'por-aprobar': { titulo: 'No hay comprobantes pendientes de aprobación.', texto: 'Cuando se adjunte un comprobante aparecerá aquí para revisarlo.' },
  sustentados: { titulo: 'Todavía no hay gastos sustentados.', texto: 'Un gasto queda sustentado cuando se aprueba su comprobante.' },
};

@Component({
  selector: 'app-gastos-lista',
  imports: [RouterLink, IconoComponent, EstadoVacioComponent, FilaGastoComponent],
  templateUrl: './gastos-lista.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GastosListaComponent implements OnInit {
  /** ?estado= de la URL */
  readonly estado = input<string | undefined>(undefined);
  /** ?viaje= y ?cliente= de la URL: filtros por viaje y por cliente (HU-12) */
  readonly viaje = input<string | undefined>(undefined);
  readonly cliente = input<string | undefined>(undefined);

  protected readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);
  private readonly router = inject(Router);
  private readonly reportes = inject(ReportesService);
  private readonly toast = inject(ToastService);

  protected readonly termino = signal('');
  protected readonly pestanas: { id: Pestana; etiqueta: string }[] = [
    { id: 'todos', etiqueta: 'Todos' },
    { id: 'sin-comprobante', etiqueta: 'Sin comprobante' },
    { id: 'por-aprobar', etiqueta: 'Por aprobar' },
    { id: 'sustentados', etiqueta: 'Sustentados' },
  ];
  protected readonly pestana = computed<Pestana>(() => {
    const e = this.estado();
    return esPestana(e) ? e : 'todos';
  });
  protected readonly viajeId = computed(() => idDeConsulta(this.viaje()));
  protected readonly clienteId = computed(() => idDeConsulta(this.cliente()));
  protected readonly hayFiltros = computed(() => this.viajeId() !== null || this.clienteId() !== null);
  /** Los gastos del viaje y del cliente elegidos; las pestañas y el buscador trabajan sobre ellos */
  private readonly delFiltro = computed(() =>
    filtrarPorViajeYCliente(this.store.gastos(), { viaje: this.viajeId(), cliente: this.clienteId() }),
  );
  protected readonly opcionesViaje = computed(() => this.opciones('viaje_id', 'viaje'));
  protected readonly opcionesCliente = computed(() => this.opciones('cliente_id', 'cliente'));
  protected readonly conteos = computed<Record<Pestana, number>>(() => {
    const c = contarPorEstado(this.delFiltro());
    return { todos: this.delFiltro().length, 'sin-comprobante': c.Registrado, 'por-aprobar': c.Pagado, sustentados: c.Sustentado };
  });
  protected readonly visibles = computed(() =>
    buscarGastos(
      filtrarPorPestana(ordenarRecientes(this.delFiltro()), this.pestana()),
      this.termino(),
      id => this.crud.etiqueta('cliente', id),
    ),
  );
  protected readonly exportando = signal(false);
  protected readonly total = computed(() => totalMonto(this.visibles()));
  protected readonly vacio = computed(() => VACIOS[this.pestana()]);
  protected readonly formatMonto = formatMonto;

  ngOnInit(): void {
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    this.crud.asegurarCargado('viaje');
  }

  /** Viajes o clientes que aparecen en los gastos, con su nombre, para los selectores */
  private opciones(campo: 'viaje_id' | 'cliente_id', clave: 'viaje' | 'cliente'): { valor: number; etiqueta: string }[] {
    const ids = [...new Set(this.store.gastos().map(g => g[campo]))].sort((a, b) => a - b);
    return ids.map(id => ({ valor: id, etiqueta: this.crud.etiqueta(clave, id) }));
  }

  /** Las pestañas cambian el estado y conservan el viaje y el cliente elegidos */
  protected queryDe(pestana: Pestana): Record<string, string | null> {
    return {
      estado: pestana === 'todos' ? null : pestana,
      viaje: this.viajeId() === null ? null : String(this.viajeId()),
      cliente: this.clienteId() === null ? null : String(this.clienteId()),
    };
  }

  protected elegir(filtro: 'viaje' | 'cliente', evento: Event): void {
    const valor = (evento.target as HTMLSelectElement).value;
    void this.router.navigate([], { queryParams: { [filtro]: valor || null }, queryParamsHandling: 'merge' });
  }

  protected exportar(): void {
    if (this.exportando()) return;
    this.exportando.set(true);
    this.reportes
      .descargarExcel({ viaje_id: this.viajeId(), cliente_id: this.clienteId(), estado: estadoDePestana(this.pestana()) })
      .subscribe({
        next: ({ blob, nombre }) => {
          this.exportando.set(false);
          guardarArchivo(blob, nombre);
        },
        error: err => {
          this.exportando.set(false);
          this.toast.show(mensajeError(err), '', 'err');
        },
      });
  }

  protected buscar(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
  }
}
