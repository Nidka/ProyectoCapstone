import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, of, throwError } from 'rxjs';
import { GastosStore } from './gastos.store';
import { GastosApiService } from '../services/gastos-api.service';
import { ToastService } from '../services/toast.service';
import { MSG_ERROR_RED } from '../utils/error-messages';
import { Comprobante, Gasto } from '../models/gasto.model';

const gastos: Gasto[] = [
  { id: 1, viaje_id: 1, cliente_id: 1, monto: 120, fecha: '2026-10-01T00:00:00Z', estado: 'Registrado', creado_por: 'Ana', creado_en: '2026-10-01T15:00:00Z' },
  { id: 2, viaje_id: 1, cliente_id: 1, monto: 80, fecha: '2026-10-01T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T16:00:00Z' },
];
/** Cada carga pide gastos, comprobantes y, aparte, los comprobantes pendientes */
const GETS_POR_CARGA = 3;

const comprobantes: Comprobante[] = [
  { id: 9, gasto_id: 2, archivo_url: '/uploads/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z' },
];

function apiFalsa() {
  return {
    get: vi.fn((ruta: string): Observable<unknown> => of(ruta === '/gastos' ? gastos : comprobantes)),
    post: vi.fn(() => of({ id: 3 })),
    uploadFile: vi.fn(() => of({ id: 10 })),
  };
}

describe('GastosStore', () => {
  let api: ReturnType<typeof apiFalsa>;
  let store: GastosStore;

  beforeEach(() => {
    api = apiFalsa();
    TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
    store = TestBed.inject(GastosStore);
  });

  it('suma los comprobantes pendientes aunque no estén entre los 200 más recientes', () => {
    const antiguo: Comprobante = { id: 4, gasto_id: 1, archivo_url: '/uploads/4.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-01-01T10:00:00Z' };
    api.get.mockImplementation((ruta: string) =>
      of(ruta === '/gastos' ? gastos : ruta === '/comprobantes?estado=Pendiente' ? [antiguo, comprobantes[0]] : comprobantes),
    );
    store.cargar();
    expect(api.get).toHaveBeenCalledWith('/comprobantes?estado=Pendiente');
    expect(store.pendientes().map(c => c.id).sort()).toEqual([4, 9]);
    expect(store.comprobantes().filter(c => c.id === 9)).toHaveLength(1);
  });

  it('cargarGasto trae por su número un gasto que no está en la lista y lo agrega', () => {
    const viejo: Gasto = { ...gastos[0], id: 5 };
    api.get.mockImplementation((ruta: string) => of(ruta === '/gastos/5' ? viejo : []));
    let recibido: Gasto | null | undefined;
    store.cargarGasto(5).subscribe(g => (recibido = g));
    expect(recibido).toEqual(viejo);
    expect(store.gasto(5)).toEqual(viejo);
  });

  it('cargarGasto emite null si el gasto no existe (404)', () => {
    api.get.mockImplementation(() => throwError(() => new HttpErrorResponse({ status: 404, statusText: 'Not Found' })));
    let recibido: Gasto | null | undefined;
    store.cargarGasto(999).subscribe(g => (recibido = g));
    expect(recibido).toBeNull();
  });

  it('carga gastos y comprobantes y deriva conteos y pendientes', () => {
    store.cargar();
    expect(store.cargado()).toBe(true);
    expect(store.gastos()).toHaveLength(2);
    expect(store.conteo()).toEqual({ Registrado: 1, Pagado: 1, Sustentado: 0 });
    expect(store.pendientes().map(c => c.id)).toEqual([9]);
    expect(store.gasto(2)?.monto).toBe(80);
    expect(store.comprobantesDe(2).map(c => c.id)).toEqual([9]);
  });

  it('asegurarCargado no repite la carga', () => {
    store.asegurarCargado();
    store.asegurarCargado();
    expect(api.get).toHaveBeenCalledTimes(GETS_POR_CARGA);
  });

  it('las mutaciones llaman a la API correcta y recargan', () => {
    const archivo = new File(['x'], 'boleta.jpg', { type: 'image/jpeg' });
    store.registrar({ viaje_id: 1, cliente_id: 1, monto: 50, fecha: '2026-10-02' }).subscribe();
    store.adjuntar(1, archivo).subscribe();
    const todo = { legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true };
    const parcial = { ...todo, monto_coincide: false };
    store.aprobar(9, todo).subscribe();
    store.rechazar(9, 'Ilegible', parcial).subscribe();
    expect(api.post).toHaveBeenCalledWith('/gastos', { viaje_id: 1, cliente_id: 1, monto: 50, fecha: '2026-10-02' });
    expect(api.uploadFile).toHaveBeenCalledWith('/gastos/1/comprobantes', archivo);
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/aprobar', { checklist: todo });
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/rechazar', { motivo_rechazo: 'Ilegible', checklist: parcial });
    expect(api.get).toHaveBeenCalledTimes(4 * GETS_POR_CARGA);
  });

  it('si la carga falla marca el error y avisa con el copy exacto', () => {
    api.get.mockReturnValue(throwError(() => new Error('sin red')));
    store.cargar();
    expect(store.error()).toBe(true);
    expect(store.cargado()).toBe(false);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe(MSG_ERROR_RED);
  });
});
