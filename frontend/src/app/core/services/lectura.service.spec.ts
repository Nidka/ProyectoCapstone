import { afterEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { NEVER, TimeoutError, of } from 'rxjs';
import { LecturaService } from './lectura.service';
import { GastosApiService } from './gastos-api.service';

function montar(respuesta: unknown) {
  const api = { get: vi.fn(() => respuesta) };
  TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
  return { servicio: TestBed.inject(LecturaService), api };
}

describe('LecturaService', () => {
  afterEach(() => vi.useRealTimers());

  it('pide la lectura del comprobante', () => {
    const lectura = { estado: 'no_disponible', sugerencias: {} };
    const { servicio, api } = montar(of(lectura));
    let recibido: unknown;
    servicio.lectura(9).subscribe(r => (recibido = r));
    expect(api.get).toHaveBeenCalledWith('/comprobantes/9/lectura');
    expect(recibido).toEqual(lectura);
  });

  it('espera hasta 35 s (5 s más que el servidor), no los 8 s de las demás operaciones', () => {
    vi.useFakeTimers();
    const { servicio } = montar(NEVER);
    let error: unknown;
    servicio.lectura(9).subscribe({ error: e => (error = e) });
    vi.advanceTimersByTime(34_999);
    expect(error).toBeUndefined();
    vi.advanceTimersByTime(1);
    expect(error).toBeInstanceOf(TimeoutError);
  });
});
