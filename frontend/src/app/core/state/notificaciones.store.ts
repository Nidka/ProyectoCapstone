import { Injectable, inject, signal } from '@angular/core';
import { Subscription } from 'rxjs';
import { Notificacion } from '../models/notificacion';
import { NotificacionesService } from '../services/notificaciones.service';

/** El servicio genera avisos cada hora: preguntar cada 5 minutos basta y no lo satura (RNF-8 pide 24 h). */
export const SONDEO_MS = 5 * 60_000;
/** Abrir la campana vuelve a pedir los avisos, salvo que se hayan pedido hace menos de esto. */
const FRESCO_MS = 30_000;

/**
 * Avisos de Logística para la campana de la barra. Quien la muestra inicia el sondeo al aparecer
 * y lo detiene al desaparecer; detenerlo vacía todo para que otra sesión no vea avisos ajenos.
 */
@Injectable({ providedIn: 'root' })
export class NotificacionesStore {
  private readonly servicio = inject(NotificacionesService);

  readonly lista = signal<Notificacion[]>([]);
  readonly noLeidas = signal(0);
  /** Hay un pedido en curso */
  readonly cargando = signal(false);
  /** Ya llegó al menos una respuesta */
  readonly cargado = signal(false);
  /** El último pedido falló; la lista que ya había se conserva */
  readonly error = signal(false);

  private pedido?: Subscription;
  private temporizador?: ReturnType<typeof setInterval>;
  private ultimaCarga = 0;
  private readonly alVolver = () => {
    if (!document.hidden) this.actualizarSiViejo(SONDEO_MS);
  };

  cargar(): void {
    // Una respuesta atrasada de un pedido anterior no debe pisar a la nueva
    this.pedido?.unsubscribe();
    this.cargando.set(true);
    this.pedido = this.servicio.listar().subscribe({
      next: r => {
        this.lista.set(r.notificaciones ?? []);
        this.noLeidas.set(r.no_leidas ?? 0);
        this.error.set(false);
        this.cargado.set(true);
        this.cargando.set(false);
        this.ultimaCarga = Date.now();
      },
      // Es un sondeo de fondo: un fallo no interrumpe con un aviso, la campana lo dice al abrirla
      error: () => {
        this.error.set(true);
        this.cargando.set(false);
      },
    });
  }

  /** Para cuando se abre la campana: actualiza solo si lo que se ve tiene más de 30 segundos. */
  actualizarSiViejo(edadMaximaMs = FRESCO_MS): void {
    if (!this.cargado() || Date.now() - this.ultimaCarga > edadMaximaMs) this.cargar();
  }

  marcarLeida(id: number): void {
    const aviso = this.lista().find(a => a.id === id);
    if (!aviso || aviso.leida) return;
    this.servicio.marcarLeida(id).subscribe({
      next: actualizado => {
        this.lista.update(lista => lista.map(a => (a.id === id ? { ...a, leida: true, leida_en: actualizado.leida_en } : a)));
        this.noLeidas.update(n => Math.max(0, n - 1));
      },
      error: () => {}, // sigue sin leer; la próxima vez que se pulse se reintenta
    });
  }

  iniciarSondeo(): void {
    if (this.temporizador !== undefined) return;
    this.cargar();
    // Con la pestaña oculta no se pide nada; al volver, alVolver actualiza si ya pasó el intervalo
    this.temporizador = setInterval(() => {
      if (!document.hidden) this.cargar();
    }, SONDEO_MS);
    document.addEventListener('visibilitychange', this.alVolver);
  }

  detenerSondeo(): void {
    if (this.temporizador !== undefined) clearInterval(this.temporizador);
    this.temporizador = undefined;
    document.removeEventListener('visibilitychange', this.alVolver);
    this.pedido?.unsubscribe();
    this.lista.set([]);
    this.noLeidas.set(0);
    this.cargando.set(false);
    this.cargado.set(false);
    this.error.set(false);
    this.ultimaCarga = 0;
  }
}
