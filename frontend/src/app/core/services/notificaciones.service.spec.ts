import { afterEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { NEVER, TimeoutError, of } from 'rxjs';
import { NotificacionesService } from './notificaciones.service';
import { GastosApiService } from './gastos-api.service';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';

function montar(respuesta: unknown) {
  const api = { get: vi.fn(() => respuesta), post: vi.fn(() => respuesta) };
  TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
  return { servicio: TestBed.inject(NotificacionesService), api };
}

describe('NotificacionesService', () => {
  afterEach(() => vi.useRealTimers());

  it('pide la lista de avisos y el número de no leídos', () => {
    const cuerpo = { no_leidas: 0, notificaciones: [] };
    const { servicio, api } = montar(of(cuerpo));
    let recibido: unknown;
    servicio.listar().subscribe(r => (recibido = r));
    expect(api.get).toHaveBeenCalledWith('/notificaciones');
    expect(recibido).toEqual(cuerpo);
  });

  it('marca un aviso como leído con POST /notificaciones/{id}/leida', () => {
    const aviso = { id: 4, tipo: 'sin_comprobante', id_gasto: 12, mensaje: 'x', creada_en: '2026-10-02T10:00:00Z', leida: true, leida_en: '2026-10-02T11:00:00Z' };
    const { servicio, api } = montar(of(aviso));
    let recibido: unknown;
    servicio.marcarLeida(4).subscribe(r => (recibido = r));
    expect(api.post).toHaveBeenCalledWith('/notificaciones/4/leida', {});
    expect(recibido).toEqual(aviso);
  });

  it('se rinde con el tiempo de espera de las demás consultas', () => {
    vi.useFakeTimers();
    const { servicio } = montar(NEVER);
    const errores: unknown[] = [];
    servicio.listar().subscribe({ error: e => errores.push(e) });
    servicio.marcarLeida(1).subscribe({ error: e => errores.push(e) });
    vi.advanceTimersByTime(HTTP_TIMEOUT_MS);
    expect(errores).toHaveLength(2);
    expect(errores.every(e => e instanceof TimeoutError)).toBe(true);
  });
});
