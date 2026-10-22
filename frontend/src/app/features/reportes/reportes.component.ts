import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, input, signal } from '@angular/core';
import { rxResource } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router } from '@angular/router';
import { FiltroReporte, FilaRanking, TipoPdf } from '../../core/models/reporte.model';
import { ReportesService } from '../../core/services/reportes.service';
import { ToastService } from '../../core/services/toast.service';
import { guardarArchivo } from '../../core/utils/descarga';
import { mensajeError } from '../../core/utils/error-messages';
import { formatFecha, formatMonto, hoyIso } from '../../core/utils/format';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import {
  ATAJOS, Atajo, atajosActivos, errorDeFiltro, filtroDesdeConsulta, mesCompleto, nombreMes, rangoDeAtajo, rangoDelMes,
} from './filtro-reporte';
import { GraficoComponent } from './grafico/grafico.component';
import {
  COLOR_ESTADO, ConfigGrafico, configPorEstado, configPorMes, configRanking, descripcionPorEstado, descripcionPorMes,
  descripcionRanking, porcentaje,
} from './graficos';

/** Chrome emite «change» en cada tecla de un campo de fecha: la edición se aplica tras esta pausa */
const ESPERA_EDICION_MS = 500;

interface Ranking {
  id: string;
  titulo: string;
  columna: string;
  filas: FilaRanking[];
  config: ConfigGrafico;
  descripcion: string;
}

/**
 * Dashboard gerencial: estadísticas de gastos de un rango de fechas, con gráficos
 * interactivos y descarga de los dos PDF. El rango vive en la URL (?desde=&hasta=).
 */
@Component({
  selector: 'app-reportes',
  imports: [EstadoVacioComponent, GraficoComponent, IconoComponent],
  templateUrl: './reportes.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ReportesComponent {
  /** Parámetros de la URL, enlazados por withComponentInputBinding */
  readonly desde = input<string>();
  readonly hasta = input<string>();

  private readonly reportes = inject(ReportesService);
  private readonly router = inject(Router);
  private readonly ruta = inject(ActivatedRoute);
  private readonly toast = inject(ToastService);

  protected readonly hoy = hoyIso();
  protected readonly atajos = ATAJOS;
  protected readonly filtro = computed(() => filtroDesdeConsulta(this.desde(), this.hasta(), this.hoy));
  protected readonly errorFiltro = computed(() => errorDeFiltro(this.filtro()));
  protected readonly atajosActivos = computed(() => atajosActivos(this.filtro(), this.hoy));
  /** Rango que había antes de elegir un mes en el gráfico; «Quitar» vuelve a él */
  private readonly rangoPrevio = signal<FiltroReporte | null>(null);
  protected readonly mesVisto = computed(() => (this.rangoPrevio() ? mesCompleto(this.filtro()) : null));
  protected readonly descargando = signal<TipoPdf | 'excel' | null>(null);
  private edicionPendiente: ReturnType<typeof setTimeout> | undefined;

  protected readonly resumen = rxResource({
    params: () => (this.errorFiltro() ? undefined : this.filtro()),
    stream: ({ params }) => this.reportes.resumen(params),
  });
  protected readonly datos = computed(() => (this.resumen.hasValue() ? this.resumen.value() : null));
  protected readonly configMes = computed(() => {
    const d = this.datos();
    return d ? configPorMes(d.por_mes, mes => this.elegirMes(mes)) : null;
  });
  protected readonly configEstado = computed(() => {
    const d = this.datos();
    return d ? configPorEstado(d.por_estado) : null;
  });
  protected readonly rankings = computed<Ranking[]>(() => {
    const d = this.datos();
    if (!d) return [];
    const ranking = (id: string, titulo: string, columna: string, filas: FilaRanking[]): Ranking => ({
      id, titulo, columna, filas, config: configRanking(filas), descripcion: descripcionRanking(titulo, filas),
    });
    return [
      ranking('cliente', 'Gasto por cliente', 'Cliente', d.por_cliente),
      ranking('unidad', 'Gasto por unidad', 'Unidad (placa)', d.por_unidad),
      ranking('conductor', 'Gasto por conductor', 'Conductor', d.por_conductor),
    ];
  });

  protected readonly formatMonto = formatMonto;
  protected readonly formatFecha = formatFecha;
  protected readonly porcentaje = porcentaje;
  protected readonly nombreMes = nombreMes;
  protected readonly colorEstado = COLOR_ESTADO;
  protected readonly descripcionPorMes = descripcionPorMes;
  protected readonly descripcionPorEstado = descripcionPorEstado;

  constructor() {
    inject(DestroyRef).onDestroy(() => clearTimeout(this.edicionPendiente));
  }

  protected elegirAtajo(atajo: Atajo): void {
    this.rangoPrevio.set(null);
    this.irA(rangoDeAtajo(atajo, this.hoy));
  }

  /**
   * Usa las dos fechas visibles (si se cambian seguidas, la URL todavía no tiene la primera),
   * ignora un campo a medio borrar y aplica tras una pausa, reemplazando la entrada del
   * historial en vez de sumar una por cada tecla.
   */
  protected cambiarRango(desde: string, hasta: string): void {
    clearTimeout(this.edicionPendiente);
    if (!desde || !hasta) return;
    this.edicionPendiente = setTimeout(() => {
      this.rangoPrevio.set(null);
      this.irA({ desde, hasta }, true);
    }, ESPERA_EDICION_MS);
  }

  /**
   * Clic en una columna del gráfico (o «Ver mes» en su tabla): todo el dashboard pasa a ese mes.
   * Recuerda el rango actual salvo que ya se esté viendo un mes, así «Quitar» vuelve a él
   * aunque entre medio se haya usado «Atrás» del navegador.
   */
  protected elegirMes(mes: string): void {
    if (!this.mesVisto()) this.rangoPrevio.set(this.filtro());
    this.irA(rangoDelMes(mes));
  }

  protected quitarMes(): void {
    const previo = this.rangoPrevio();
    this.rangoPrevio.set(null);
    if (previo) this.irA(previo);
  }

  protected descargar(tipo: TipoPdf | 'excel'): void {
    if (this.descargando() || this.errorFiltro()) return;
    this.descargando.set(tipo);
    const archivo = tipo === 'excel' ? this.reportes.descargarExcel(this.filtro()) : this.reportes.descargar(tipo, this.filtro());
    archivo.subscribe({
      next: ({ blob, nombre }) => {
        this.descargando.set(null);
        guardarArchivo(blob, nombre);
      },
      error: err => {
        this.descargando.set(null);
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }

  private irA(filtro: FiltroReporte, reemplazar = false): void {
    void this.router.navigate([], {
      relativeTo: this.ruta,
      queryParams: { desde: filtro.desde, hasta: filtro.hasta },
      replaceUrl: reemplazar,
    });
  }
}
