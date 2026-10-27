import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { Observable, Subject, of, throwError } from 'rxjs';
import { NotificacionesStore, SONDEO_MS } from './notificaciones.store';
import { NotificacionesService } from '../services/notificaciones.service';
import { Notificacion, RespuestaNotificaciones } from '../models/notificacion';

function aviso(id: number, extra: Partial<Notificacion> = {}): Notificacion {
  return {
    id, tipo: 'sin_comprobante', id_gasto: 10 + id, mensaje: `Aviso ${id}`,
    creada_en: '2026-10-02T10:00:00Z', leida: false, leida_en: null, ...extra,
  };
}

const RESPUESTA: RespuestaNotificaciones = { no_leidas: 2, notificaciones: [aviso(2), aviso(1), aviso(3, { leida: true, leida_en: '2026-10-02T12:00:00Z' })] };

function montar() {
  const servicio = {
    listar: vi.fn((): Observable<RespuestaNotificaciones> => of(RESPUESTA)),
    marcarLeida: vi.fn((id: number): Observable<Notificacion> => of(aviso(id, { leida: true, leida_en: '2026-10-02T13:00:00Z' }))),
  };
  TestBed.configureTestingModule({ providers: [{ provide: NotificacionesService, useValue: servicio }] });
  return { servicio, store: TestBed.inject(NotificacionesStore) };
}

describe('NotificacionesStore', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    TestBed.inject(NotificacionesStore).detenerSondeo();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it('cargar deja la lista y el número de no leídas que dice el servicio', () => {
    const { store } = montar();
    store.cargar();
    expect(store.lista().map(a => a.id)).toEqual([2, 1, 3]);
    expect(store.noLeidas()).toBe(2);
    expect(store.cargado()).toBe(true);
    expect(store.error()).toBe(false);
    expect(store.cargando()).toBe(false);
  });

  it('mientras llega la primera respuesta está cargando', () => {
    const { store, servicio } = montar();
    servicio.listar.mockReturnValue(new Subject<RespuestaNotificaciones>());
    store.cargar();
    expect(store.cargando()).toBe(true);
    expect(store.cargado()).toBe(false);
  });

  it('si falla, marca el error sin avisar con un toast y conserva lo que ya había', () => {
    const { store, servicio } = montar();
    store.cargar();
    servicio.listar.mockReturnValue(throwError(() => new Error('sin red')));
    store.cargar();
    expect(store.error()).toBe(true);
    expect(store.cargando()).toBe(false);
    expect(store.lista()).toHaveLength(3);
    expect(store.noLeidas()).toBe(2);
    // Una carga que sale bien limpia el error
    servicio.listar.mockReturnValue(of(RESPUESTA));
    store.cargar();
    expect(store.error()).toBe(false);
  });

  it('una respuesta atrasada no pisa a la más nueva', () => {
    const { store, servicio } = montar();
    const lenta = new Subject<RespuestaNotificaciones>();
    servicio.listar.mockReturnValueOnce(lenta);
    store.cargar();
    servicio.listar.mockReturnValueOnce(of({ no_leidas: 1, notificaciones: [aviso(7)] }));
    store.cargar();
    lenta.next(RESPUESTA);
    expect(store.lista().map(a => a.id)).toEqual([7]);
    expect(store.noLeidas()).toBe(1);
  });

  describe('marcarLeida', () => {
    it('pone el aviso como leído y baja el número', () => {
      const { store, servicio } = montar();
      store.cargar();
      store.marcarLeida(2);
      expect(servicio.marcarLeida).toHaveBeenCalledWith(2);
      expect(store.lista().find(a => a.id === 2)?.leida).toBe(true);
      expect(store.noLeidas()).toBe(1);
    });

    it('no vuelve a pedir un aviso que ya estaba leído', () => {
      const { store, servicio } = montar();
      store.cargar();
      store.marcarLeida(3);
      expect(servicio.marcarLeida).not.toHaveBeenCalled();
      expect(store.noLeidas()).toBe(2);
    });

    it('si el servicio falla, el aviso sigue sin leer', () => {
      const { store, servicio } = montar();
      store.cargar();
      servicio.marcarLeida.mockReturnValue(throwError(() => new Error('sin red')));
      store.marcarLeida(2);
      expect(store.lista().find(a => a.id === 2)?.leida).toBe(false);
      expect(store.noLeidas()).toBe(2);
    });

    it('el número nunca baja de cero', () => {
      const { store, servicio } = montar();
      servicio.listar.mockReturnValue(of({ no_leidas: 0, notificaciones: [aviso(1)] }));
      store.cargar();
      store.marcarLeida(1);
      expect(store.noLeidas()).toBe(0);
    });
  });

  describe('sondeo', () => {
    it('pide los avisos al iniciar y luego cada 5 minutos', () => {
      const { store, servicio } = montar();
      store.iniciarSondeo();
      expect(servicio.listar).toHaveBeenCalledTimes(1);
      expect(SONDEO_MS).toBe(5 * 60_000);
      vi.advanceTimersByTime(SONDEO_MS);
      expect(servicio.listar).toHaveBeenCalledTimes(2);
      vi.advanceTimersByTime(SONDEO_MS * 2);
      expect(servicio.listar).toHaveBeenCalledTimes(4);
    });

    it('iniciarlo dos veces no duplica el temporizador', () => {
      const { store, servicio } = montar();
      store.iniciarSondeo();
      store.iniciarSondeo();
      vi.advanceTimersByTime(SONDEO_MS);
      expect(servicio.listar).toHaveBeenCalledTimes(2);
    });

    it('al detenerlo limpia el temporizador y vacía lo que había (otra sesión no lo ve)', () => {
      const { store, servicio } = montar();
      store.iniciarSondeo();
      store.detenerSondeo();
      vi.advanceTimersByTime(SONDEO_MS * 3);
      expect(servicio.listar).toHaveBeenCalledTimes(1);
      expect(vi.getTimerCount()).toBe(0);
      expect(store.lista()).toEqual([]);
      expect(store.noLeidas()).toBe(0);
      expect(store.cargado()).toBe(false);
    });

    it('una respuesta que llega después de detenerlo se descarta', () => {
      const { store, servicio } = montar();
      const lenta = new Subject<RespuestaNotificaciones>();
      servicio.listar.mockReturnValue(lenta);
      store.iniciarSondeo();
      store.detenerSondeo();
      lenta.next(RESPUESTA);
      expect(store.lista()).toEqual([]);
      expect(store.noLeidas()).toBe(0);
    });

    it('con la pestaña oculta no pide nada', () => {
      const { store, servicio } = montar();
      store.iniciarSondeo();
      vi.spyOn(document, 'hidden', 'get').mockReturnValue(true);
      vi.advanceTimersByTime(SONDEO_MS);
      expect(servicio.listar).toHaveBeenCalledTimes(1);
    });

    it('al volver a la pestaña actualiza si pasó más tiempo que el sondeo', () => {
      const { store, servicio } = montar();
      store.iniciarSondeo();
      const oculta = vi.spyOn(document, 'hidden', 'get').mockReturnValue(true);
      vi.advanceTimersByTime(SONDEO_MS + 1000);
      oculta.mockReturnValue(false);
      document.dispatchEvent(new Event('visibilitychange'));
      expect(servicio.listar).toHaveBeenCalledTimes(2);
      // Si acaba de actualizar, volver a la pestaña no repite el pedido
      document.dispatchEvent(new Event('visibilitychange'));
      expect(servicio.listar).toHaveBeenCalledTimes(2);
    });

    it('después de detenerlo, volver a la pestaña no pide nada', () => {
      const { store, servicio } = montar();
      store.iniciarSondeo();
      store.detenerSondeo();
      vi.advanceTimersByTime(SONDEO_MS * 2);
      document.dispatchEvent(new Event('visibilitychange'));
      expect(servicio.listar).toHaveBeenCalledTimes(1);
    });
  });

  describe('actualizarSiViejo', () => {
    it('no repite el pedido si la última carga es reciente', () => {
      const { store, servicio } = montar();
      store.cargar();
      vi.advanceTimersByTime(10_000);
      store.actualizarSiViejo();
      expect(servicio.listar).toHaveBeenCalledTimes(1);
    });

    it('vuelve a pedir si pasaron más de 30 segundos', () => {
      const { store, servicio } = montar();
      store.cargar();
      vi.advanceTimersByTime(31_000);
      store.actualizarSiViejo();
      expect(servicio.listar).toHaveBeenCalledTimes(2);
    });

    it('pide si todavía no cargó nada', () => {
      const { store, servicio } = montar();
      store.actualizarSiViejo();
      expect(servicio.listar).toHaveBeenCalledTimes(1);
    });
  });
});
