import { Injectable, inject } from '@angular/core';
import { Observable, map, timeout } from 'rxjs';
import { ArchivoDescargable, ArchivoPdf, FiltroReporte, ParamsExcel, ResumenReporte, TipoPdf } from '../models/reporte.model';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';
import { GastosApiService } from './gastos-api.service';

const PREFIJO_PDF: Record<TipoPdf, string> = { estadistico: 'reporte-estadistico', detalle: 'detalle-gastos' };

/** Lee el nombre de archivo de «attachment; filename="…"»; si no viene, usa el respaldo. */
export function nombreDeArchivo(disposicion: string | null, respaldo: string): string {
  return /filename="([^"]+)"/.exec(disposicion ?? '')?.[1] ?? respaldo;
}

function consulta(filtro: FiltroReporte): string {
  return `desde=${filtro.desde}&hasta=${filtro.hasta}`;
}

/** Estadísticas y PDF de gastos-service para el dashboard gerencial. */
@Injectable({ providedIn: 'root' })
export class ReportesService {
  private readonly api = inject(GastosApiService);

  resumen(filtro: FiltroReporte): Observable<ResumenReporte> {
    return this.api.get<ResumenReporte>(`/reportes/resumen?${consulta(filtro)}`).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  /** Excel con el detalle de los gastos que cumplen los filtros (viaje, cliente, estado, rango). */
  descargarExcel(params: ParamsExcel): Observable<ArchivoDescargable> {
    const consultaExcel = Object.entries(params)
      .filter(([, v]) => v !== null && v !== undefined && v !== '')
      .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
      .join('&');
    const respaldo = `gastos-${new Date().toISOString().slice(0, 10)}.xlsx`;
    return this.api.getBlob(`/gastos/exportar.xlsx${consultaExcel ? `?${consultaExcel}` : ''}`).pipe(
      timeout(HTTP_TIMEOUT_MS),
      map(res => ({
        blob: res.body ?? new Blob([]),
        nombre: nombreDeArchivo(res.headers.get('Content-Disposition'), respaldo),
      })),
    );
  }

  descargar(tipo: TipoPdf, filtro: FiltroReporte): Observable<ArchivoPdf> {
    const respaldo = `${PREFIJO_PDF[tipo]}-${filtro.desde}_${filtro.hasta}.pdf`;
    return this.api.getBlob(`/reportes/${tipo}.pdf?${consulta(filtro)}`).pipe(
      timeout(HTTP_TIMEOUT_MS),
      map(res => ({ blob: res.body ?? new Blob([], { type: 'application/pdf' }), nombre: nombreDeArchivo(res.headers.get('Content-Disposition'), respaldo) })),
    );
  }
}
