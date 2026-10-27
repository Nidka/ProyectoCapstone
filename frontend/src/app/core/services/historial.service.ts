import { Injectable, inject } from '@angular/core';
import { Observable, timeout } from 'rxjs';
import { RegistroHistorial } from '../models/historial';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';
import { GastosApiService } from './gastos-api.service';

/** Historial de cambios de un gasto y sus comprobantes (bitácora de auditoría, gastos-service). */
@Injectable({ providedIn: 'root' })
export class HistorialService {
  private readonly api = inject(GastosApiService);

  deGasto(gastoId: number): Observable<RegistroHistorial[]> {
    return this.api.get<RegistroHistorial[]>(`/gastos/${gastoId}/historial`).pipe(timeout(HTTP_TIMEOUT_MS));
  }
}
