import { Injectable, computed, inject, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, catchError, forkJoin, of, tap, throwError, timeout } from 'rxjs';
import { GastosApiService } from '../services/gastos-api.service';
import { ToastService } from '../services/toast.service';
import { Checklist } from '../models/checklist';
import { Comprobante, Gasto } from '../models/gasto.model';
import { HTTP_TIMEOUT_MS, mensajeError } from '../utils/error-messages';
import { comprobantesDe, contarPorEstado } from './gastos.logic';

export interface NuevoGasto {
  viaje_id: number;
  cliente_id: number;
  monto: number;
  fecha: string;
}

/** Gastos y comprobantes compartidos por Inicio, la bandeja, el detalle, la barra y la paleta. */
@Injectable({ providedIn: 'root' })
export class GastosStore {
  private readonly api = inject(GastosApiService);
  private readonly toast = inject(ToastService);

  readonly gastos = signal<Gasto[]>([]);
  readonly comprobantes = signal<Comprobante[]>([]);
  readonly cargando = signal(false);
  readonly cargado = signal(false);
  readonly error = signal(false);

  readonly conteo = computed(() => contarPorEstado(this.gastos()));
  readonly pendientes = computed(() => this.comprobantes().filter(c => c.estado === 'Pendiente'));

  gasto(id: number): Gasto | undefined {
    return this.gastos().find(g => g.id === id);
  }

  comprobantesDe(gastoId: number): Comprobante[] {
    return comprobantesDe(this.comprobantes(), gastoId);
  }

  cargar(): void {
    this.cargando.set(true);
    forkJoin({
      gastos: this.api.get<Gasto[]>('/gastos'),
      comprobantes: this.api.get<Comprobante[]>('/comprobantes'),
      // El servicio devuelve solo los 200 más recientes: los pendientes se piden aparte para que ninguno quede sin revisar
      pendientes: this.api.get<Comprobante[]>('/comprobantes?estado=Pendiente'),
    }).pipe(timeout(HTTP_TIMEOUT_MS)).subscribe({
      next: ({ gastos, comprobantes, pendientes }) => {
        const porId = new Map<number, Comprobante>();
        for (const c of [...(comprobantes ?? []), ...(pendientes ?? [])]) if (!porId.has(c.id)) porId.set(c.id, c);
        this.gastos.set(gastos ?? []);
        this.comprobantes.set([...porId.values()]);
        this.error.set(false);
        this.cargado.set(true);
        this.cargando.set(false);
      },
      error: err => {
        this.error.set(true);
        this.cargando.set(false);
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }

  /** Un gasto que no vino en la lista (el servicio devuelve los 200 más recientes) se pide por su número; emite null si no existe. */
  cargarGasto(id: number): Observable<Gasto | null> {
    return this.api.get<Gasto>(`/gastos/${id}`).pipe(
      timeout(HTTP_TIMEOUT_MS),
      tap(g => this.gastos.update(lista => (lista.some(x => x.id === g.id) ? lista : [...lista, g]))),
      catchError(err => (err instanceof HttpErrorResponse && err.status === 404 ? of(null) : throwError(() => err))),
    );
  }

  asegurarCargado(): void {
    if (!this.cargado() && !this.cargando()) this.cargar();
  }

  registrar(datos: NuevoGasto): Observable<Gasto> {
    return this.api.post<Gasto>('/gastos', datos).pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }

  adjuntar(gastoId: number, archivo: File): Observable<Comprobante> {
    return this.api
      .uploadFile<Comprobante>(`/gastos/${gastoId}/comprobantes`, archivo)
      .pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }

  aprobar(comprobanteId: number, checklist: Checklist): Observable<unknown> {
    return this.api
      .post(`/comprobantes/${comprobanteId}/aprobar`, { checklist })
      .pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }

  rechazar(comprobanteId: number, motivo: string, checklist: Checklist): Observable<unknown> {
    return this.api
      .post(`/comprobantes/${comprobanteId}/rechazar`, { motivo_rechazo: motivo, checklist })
      .pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }
}
