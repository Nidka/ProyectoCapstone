import { afterEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { NEVER, TimeoutError, of } from 'rxjs';
import { HistorialService } from './historial.service';
import { GastosApiService } from './gastos-api.service';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';

function montar(respuesta: unknown) {
  const api = { get: vi.fn(() => respuesta) };
  TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
  return { servicio: TestBed.inject(HistorialService), api };
}

describe('HistorialService', () => {
  afterEach(() => vi.useRealTimers());

  it('pide el historial del gasto', () => {
    const registros = [{ fecha: '2026-10-02T14:05:00-05:00', usuario: 'Ana', accion: 'gasto.registrar', entidad: 'gasto', id_entidad: 31, detalle: {} }];
    const { servicio, api } = montar(of(registros));
    let recibido: unknown;
    servicio.deGasto(31).subscribe(r => (recibido = r));
    expect(api.get).toHaveBeenCalledWith('/gastos/31/historial');
    expect(recibido).toEqual(registros);
  });

  it('se rinde con el tiempo de espera de las demás consultas', () => {
    vi.useFakeTimers();
    const { servicio } = montar(NEVER);
    let error: unknown;
    servicio.deGasto(31).subscribe({ error: e => (error = e) });
    vi.advanceTimersByTime(HTTP_TIMEOUT_MS);
    expect(error).toBeInstanceOf(TimeoutError);
  });
});
