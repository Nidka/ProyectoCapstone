import { describe, expect, it, vi } from 'vitest';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { GastoDetalleComponent, consultaDeOrigen } from './gasto-detalle.component';
import { AdjuntarComprobanteComponent } from '../adjuntar-comprobante/adjuntar-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { CrudService } from '../../../core/services/crud.service';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { comprobantesDe } from '../../../core/state/gastos.logic';

function gasto(id: number, estado: Gasto['estado']): Gasto {
  return { id, estado, monto: 120.5, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-01T15:00:00Z' };
}

function storeFalso(gastos: Gasto[], comprobantes: Comprobante[], cargado = true) {
  const g = signal(gastos);
  const c = signal(comprobantes);
  return {
    gastos: g, comprobantes: c, cargando: signal(!cargado), cargado: signal(cargado), error: signal(false),
    gasto: (id: number) => g().find(x => x.id === id),
    comprobantesDe: (id: number) => comprobantesDe(c(), id),
    asegurarCargado: vi.fn(), cargar: vi.fn(), adjuntar: vi.fn(), aprobar: vi.fn(), rechazar: vi.fn(),
    cargarGasto: vi.fn((_id: number) => of<Gasto | null>(null)),
  };
}

function montar(store: ReturnType<typeof storeFalso>, id: string) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: GastosStore, useValue: store },
      { provide: CrudService, useValue: { etiqueta: () => 'Agro Norte SAC', asegurarCargado: vi.fn() } },
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => u } },
    ],
  });
  const f = TestBed.createComponent(GastoDetalleComponent);
  f.componentRef.setInput('id', id);
  f.detectChanges();
  return f;
}

describe('GastoDetalleComponent', () => {
  it('mientras carga muestra el esqueleto y luego avisa si el gasto no existe', () => {
    const store = storeFalso([], [], false);
    const f = montar(store, '999');
    expect(f.nativeElement.querySelector('.detalle__esqueleto')).not.toBeNull();
    store.cargando.set(false);
    store.cargado.set(true);
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('No encontramos el gasto N° 999.');
  });

  it('si el servicio de gastos no responde, muestra el error con Reintentar en vez de un esqueleto eterno', () => {
    const store = storeFalso([], [], false);
    store.cargando.set(false);
    store.error.set(true);
    const f = montar(store, '31');
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.detalle__esqueleto')).toBeNull();
    expect(el.textContent).toContain('No se pudieron cargar los gastos.');
    [...el.querySelectorAll('button')].find(b => b.textContent?.trim() === 'Reintentar')?.click();
    expect(store.cargar).toHaveBeenCalled();
  });

  it('un gasto que no está en la lista se busca por su número antes de decir que no existe', () => {
    const store = storeFalso([], []);
    store.cargarGasto.mockImplementation((id: number) => {
      store.gastos.update(l => [...l, gasto(id, 'Registrado')]);
      return of(gasto(id, 'Registrado'));
    });
    const f = montar(store, '999');
    f.detectChanges();
    expect(store.cargarGasto).toHaveBeenCalledWith(999);
    expect(f.nativeElement.querySelector('h1')?.textContent).toContain('N° 999');
  });

  it('al pasar de un gasto a otro, el paso de adjuntar empieza de cero', () => {
    const f = montar(storeFalso([gasto(31, 'Registrado'), gasto(33, 'Registrado')], []), '31');
    const adjuntar = f.debugElement.query(By.directive(AdjuntarComprobanteComponent)).componentInstance as AdjuntarComprobanteComponent;
    adjuntar.elegir(new File(['x'], 'boleta-del-31.pdf', { type: 'application/pdf' }));
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('boleta-del-31.pdf');
    f.componentRef.setInput('id', '33');
    f.detectChanges();
    expect(f.nativeElement.textContent).not.toContain('boleta-del-31.pdf');
  });

  it('un gasto registrado muestra el paso de adjuntar', () => {
    const f = montar(storeFalso([gasto(31, 'Registrado')], []), '31');
    const paso: HTMLElement = f.nativeElement.querySelector('#paso');
    expect(paso.textContent).toContain('Adjuntar comprobante de pago');
    expect(paso.classList).not.toContain('detalle__paso--senal');
    expect(paso.querySelector('app-adjuntar-comprobante')).not.toBeNull();
    expect(f.nativeElement.textContent).toContain('S/ 120.50');
  });

  it('un gasto pagado muestra la revisión del comprobante pendiente', () => {
    const pendiente: Comprobante = {
      id: 9, gasto_id: 32, archivo_url: '/u/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z',
    };
    const f = montar(storeFalso([gasto(32, 'Pagado')], [pendiente]), '32');
    expect(f.nativeElement.querySelector('#paso app-revision-comprobante')).not.toBeNull();
    // Pagado es azul informativo: el paso se anuncia con la placa informativa
    expect(f.nativeElement.querySelector('#paso h2.cartel--informativa')?.textContent?.trim()).toBe('Revisar comprobante');
    // El cartel es la banda de cabecera de la tarjeta del paso (no una pastilla que parezca botón)
    expect(f.nativeElement.querySelector('#paso').classList).toContain('detalle__paso--senal');
  });

  it('el monto va pintado en el hito del estado actual', () => {
    const f = montar(storeFalso([gasto(31, 'Registrado')], []), '31');
    const hito: HTMLElement = f.nativeElement.querySelector('.detalle__cabecera app-hito');
    expect(hito.getAttribute('data-estado')).toBe('Registrado');
    expect(hito.querySelector('.hito__cabezal')?.textContent?.trim()).toBe('Registrado');
    expect(hito.querySelector('.hito__cifra')?.textContent?.trim()).toBe('S/ 120.50');
  });

  it('un gasto sustentado llega a la señal de guía verde', () => {
    const aprobado: Comprobante = {
      id: 8, gasto_id: 29, archivo_url: '/u/8.jpg', tipo_archivo: 'imagen', estado: 'Aprobado', creado_en: '2026-09-29T17:00:00Z',
      aprobado_por: 'Logística', aprobado_en: '2026-09-30T15:00:00Z',
    };
    const f = montar(storeFalso([gasto(29, 'Sustentado')], [aprobado]), '29');
    expect(f.nativeElement.querySelector('#paso h2.cartel--guia')?.textContent?.trim()).toBe('Gasto sustentado');
    expect(f.nativeElement.querySelector('#paso').classList).toContain('detalle__paso--senal');
  });

  it('Cliente y Viaje llevan a su ficha en Operación', () => {
    const f = montar(storeFalso([gasto(31, 'Registrado')], []), '31');
    const enlaces = [...(f.nativeElement as HTMLElement).querySelectorAll<HTMLAnchorElement>('dd a')].map(a => a.getAttribute('href'));
    expect(enlaces).toContain('/app/operacion/clientes?ver=1');
    expect(enlaces).toContain('/app/operacion/viajes?ver=1');
  });

  it('el historial muestra el checklist guardado y quién lo revisó', () => {
    const rechazado: Comprobante = {
      id: 7, gasto_id: 31, archivo_url: '/u/7.jpg', tipo_archivo: 'imagen', estado: 'Rechazado', creado_en: '2026-10-01T17:00:00Z',
      motivo_rechazo: 'No cumple: monto coincide.', aprobado_por: 'Ana Quispe', aprobado_en: '2026-09-30T15:00:00Z',
      checklist: [
        { criterio: 'legible', cumple: true }, { criterio: 'tipo_valido', cumple: true }, { criterio: 'a_nombre_empresa', cumple: true },
        { criterio: 'monto_coincide', cumple: false }, { criterio: 'fecha_coincide', cumple: true },
      ],
    };
    const f = montar(storeFalso([gasto(31, 'Registrado')], [rechazado]), '31');
    const el: HTMLElement = f.nativeElement;
    const items = [...el.querySelectorAll('.historial__checklist li')];
    expect(items).toHaveLength(5);
    expect(items[3].textContent).toContain('Monto coincide');
    expect(items[3].classList).toContain('historial__criterio--no');
    expect(items[0].classList).not.toContain('historial__criterio--no');
    expect(el.querySelector('.historial__revisor')?.textContent?.trim()).toBe('Revisado por Ana Quispe el 30/09/2026 10:00');
  });

  it('un comprobante revisado antes del checklist no muestra lista ni revisor', () => {
    const antiguo: Comprobante = {
      id: 6, gasto_id: 29, archivo_url: '/u/6.jpg', tipo_archivo: 'imagen', estado: 'Aprobado', creado_en: '2026-09-29T17:00:00Z',
      aprobado_por: 'Logística', aprobado_en: '2026-09-30T15:00:00Z', checklist: [],
    };
    const f = montar(storeFalso([gasto(29, 'Sustentado')], [antiguo]), '29');
    expect(f.nativeElement.querySelector('.historial__checklist')).toBeNull();
    expect(f.nativeElement.querySelector('.historial__revisor')).toBeNull();
  });

  it('«Gastos» vuelve a la pestaña de origen', () => {
    expect(consultaDeOrigen({ origen: 'por-aprobar' })).toEqual({ estado: 'por-aprobar' });
    expect(consultaDeOrigen({ origen: 'todos' })).toEqual({});
    expect(consultaDeOrigen(null)).toEqual({});
    expect(consultaDeOrigen({ origen: 'otra' })).toEqual({});
  });
});
