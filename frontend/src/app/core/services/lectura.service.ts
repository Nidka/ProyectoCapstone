import { Injectable, inject } from '@angular/core';
import { Observable, timeout } from 'rxjs';
import { LecturaComprobante } from '../models/lectura';
import { GastosApiService } from './gastos-api.service';

/**
 * Leer un comprobante puede tardar (OCR): el servidor corta a los 30 s y aquí se espera 5 s más,
 * para que en una lectura lenta llegue su respuesta y no un error de conexión.
 */
export const LECTURA_TIMEOUT_MS = 35_000;

/** Lectura automática (OCR) del comprobante pendiente, para sugerir el checklist. */
@Injectable({ providedIn: 'root' })
export class LecturaService {
  private readonly api = inject(GastosApiService);

  lectura(comprobanteId: number): Observable<LecturaComprobante> {
    return this.api.get<LecturaComprobante>(`/comprobantes/${comprobanteId}/lectura`).pipe(timeout(LECTURA_TIMEOUT_MS));
  }
}
