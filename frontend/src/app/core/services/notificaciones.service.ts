import { Injectable, inject } from '@angular/core';
import { Observable, timeout } from 'rxjs';
import { Notificacion, RespuestaNotificaciones } from '../models/notificacion';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';
import { GastosApiService } from './gastos-api.service';

/** Avisos de gastos sin comprobante o con cobro cerca (gastos-service). Solo Logística: al administrador el servicio le responde 403. */
@Injectable({ providedIn: 'root' })
export class NotificacionesService {
  private readonly api = inject(GastosApiService);

  listar(): Observable<RespuestaNotificaciones> {
    return this.api.get<RespuestaNotificaciones>('/notificaciones').pipe(timeout(HTTP_TIMEOUT_MS));
  }

  marcarLeida(id: number): Observable<Notificacion> {
    return this.api.post<Notificacion>(`/notificaciones/${id}/leida`, {}).pipe(timeout(HTTP_TIMEOUT_MS));
  }
}
