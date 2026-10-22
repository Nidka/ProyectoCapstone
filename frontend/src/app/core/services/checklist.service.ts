import { Injectable, inject } from '@angular/core';
import { Observable, timeout } from 'rxjs';
import { ChecklistCliente } from '../models/checklist';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';
import { GastosApiService } from './gastos-api.service';

/** Un criterio al guardar: sin clave si es nuevo; los que ya existían la conservan. */
export interface EntradaCriterio {
  clave?: string;
  texto: string;
}

/** Checklist documental configurable por cliente (gastos-service). */
@Injectable({ providedIn: 'root' })
export class ChecklistService {
  private readonly api = inject(GastosApiService);

  deCliente(clienteId: number): Observable<ChecklistCliente> {
    return this.api.get<ChecklistCliente>(`/clientes/${clienteId}/checklist`).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  guardar(clienteId: number, criterios: EntradaCriterio[]): Observable<ChecklistCliente> {
    return this.api.put<ChecklistCliente>(`/clientes/${clienteId}/checklist`, { criterios }).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  restablecer(clienteId: number): Observable<ChecklistCliente> {
    return this.api.delete<ChecklistCliente>(`/clientes/${clienteId}/checklist`).pipe(timeout(HTTP_TIMEOUT_MS));
  }
}
