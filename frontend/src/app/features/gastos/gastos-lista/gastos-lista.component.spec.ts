import { describe, expect, it, vi } from 'vitest';
import { computed, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, Subject, of, throwError } from 'rxjs';
import { mensajeError } from '../../../core/utils/error-messages';
import { ReportesService } from '../../../core/services/reportes.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosListaComponent } from './gastos-lista.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { CrudService } from '../../../core/services/crud.service';
import { AuthService } from '../../../core/services/auth.service';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { contarPorEstado } from '../../../core/state/gastos.logic';

function gasto(id: number, estado: Gasto['estado'], monto: number): Gasto {
  return {
    id, estado, monto, viaje_id: 1, cliente_id: id, fecha: '2026-10-01T00:00:00Z',
    creado_por: 'Ana', creado_en: `2026-10-0${id}T12:00:00Z`,
  };
}

function storeFalso(gastos: Gasto[], estado: { error?: boolean; cargado?: boolean } = {}) {
  const lista = signal(gastos);
  return {
    gastos: lista,
    comprobantes: signal<Comprobante[]>([]),
    cargando: signal(false),
    cargado: signal(estado.cargado ?? true),
    error: signal(estado.error ?? false),
    conteo: computed(() => contarPorEstado(lista())),
    pendientes: signal<Comprobante[]>([]),
    asegurarCargado: vi.fn(),
    cargar: vi.fn(),
  };
}

const crudFalso = {
  etiqueta: (clave: string, id: unknown) => (clave === 'cliente' ? `Cliente ${id}` : `Viaje N° ${id}`),
  asegurarCargado: vi.fn(),
};

function montar(store: ReturnType<typeof storeFalso>, estado?: string, filtros: { viaje?: string; cliente?: string } = {}, excel?: ReturnType<typeof excelFalso>, rol: 'admin' | 'logistica' = 'logistica') {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: GastosStore, useValue: store },
      { provide: CrudService, useValue: crudFalso },
      { provide: ReportesService, useValue: excel ?? excelFalso() },
      { provide: AuthService, useValue: { esLogistica: signal(rol === 'logistica'), usuario: signal({ rol }) } },
    ],
  });
  const f = TestBed.createComponent(GastosListaComponent);
  if (estado) f.componentRef.setInput('estado', estado);
  if (filtros.viaje) f.componentRef.setInput('viaje', filtros.viaje);
  if (filtros.cliente) f.componentRef.setInput('cliente', filtros.cliente);
  f.detectChanges();
  return f;
}

function excelFalso(respuesta: () => Observable<{ blob: Blob; nombre: string }> = () => of({ blob: new Blob(['x']), nombre: 'gastos-2026-10-22.xlsx' })) {
  return { descargarExcel: vi.fn(respuesta) };
}

describe('GastosListaComponent', () => {
  const gastos = [gasto(1, 'Registrado', 120), gasto(2, 'Pagado', 85.5), gasto(3, 'Sustentado', 310)];

  it('la pestaña Por aprobar muestra solo gastos pagados, con Revisar y el total', () => {
    const el: HTMLElement = montar(storeFalso(gastos), 'por-aprobar').nativeElement;
    const filas = el.querySelectorAll('li.fila-gasto');
    expect(filas).toHaveLength(1);
    expect(filas[0].textContent).toContain('N° 2');
    expect(filas[0].querySelector('.fila-gasto__accion')?.textContent).toContain('Revisar');
    expect(el.querySelector('.lista-gastos__total')?.textContent).toContain('S/ 85.50');
  });

  it('Logística ve Registrar gasto arriba y en el vacío', () => {
    const el: HTMLElement = montar(storeFalso(gastos)).nativeElement;
    expect(el.querySelector('.pagina__acciones a[href="/app/gastos/nuevo"]')?.textContent).toContain('Registrar gasto');
    TestBed.resetTestingModule();
    const vacio: HTMLElement = montar(storeFalso([])).nativeElement;
    expect(vacio.querySelector('.vacio a[href="/app/gastos/nuevo"]')).not.toBeNull();
  });

  it('el administrador consulta: sin Registrar gasto, y las filas dicen Ver en vez de Adjuntar o Revisar', () => {
    TestBed.resetTestingModule();
    const el: HTMLElement = montar(storeFalso(gastos), undefined, {}, undefined, 'admin').nativeElement;
    expect(el.querySelector('a[href="/app/gastos/nuevo"]')).toBeNull();
    const acciones = [...el.querySelectorAll('.fila-gasto__accion')].map(a => a.textContent?.trim());
    expect(acciones).toEqual(['Ver', 'Ver', 'Ver']);
    expect(el.textContent).not.toContain('Adjuntar comprobante');
    expect(el.textContent).not.toContain('Revisar');
  });

  it('al administrador, sin gastos, no se le pide registrar el primero', () => {
    TestBed.resetTestingModule();
    const el: HTMLElement = montar(storeFalso([]), undefined, {}, undefined, 'admin').nativeElement;
    expect(el.querySelector('a[href="/app/gastos/nuevo"]')).toBeNull();
    expect(el.textContent).not.toContain('Presiona "Registrar gasto"');
    expect(el.textContent).toContain('Todavía no hay gastos registrados.');
  });

  it('cada pestaña muestra su conteo', () => {
    const el: HTMLElement = montar(storeFalso(gastos)).nativeElement;
    const textos = [...el.querySelectorAll('.pestanas__item')].map(a => a.textContent?.replace(/\s+/g, ' ').trim());
    expect(textos).toEqual(['Todos 3', 'Sin comprobante 1', 'Por aprobar 1', 'Sustentados 1']);
  });

  it('busca por número de gasto', () => {
    const f = montar(storeFalso(gastos));
    const input: HTMLInputElement = f.nativeElement.querySelector('input[type="search"]');
    input.value = '3';
    input.dispatchEvent(new Event('input'));
    f.detectChanges();
    const filas = f.nativeElement.querySelectorAll('li.fila-gasto');
    expect(filas).toHaveLength(1);
    expect(filas[0].textContent).toContain('N° 3');
  });

  it('si el servicio de gastos falla muestra el error con Reintentar, no el vacío', () => {
    const store = storeFalso([], { error: true, cargado: false });
    const el: HTMLElement = montar(store).nativeElement;
    expect(el.textContent).toContain('No se pudieron cargar los gastos.');
    expect(el.textContent).not.toContain('Todavía no hay gastos registrados.');
    (el.querySelector('.vacio button') as HTMLButtonElement).click();
    expect(store.cargar).toHaveBeenCalled();
  });

  describe('filtros por viaje y cliente', () => {
    // viaje 1: gastos 1 y 2; viaje 2: gastos 3 y 4. Clientes: cliente_id = id del gasto
    const variados = [
      { ...gasto(1, 'Registrado', 120), viaje_id: 1, cliente_id: 1 },
      { ...gasto(2, 'Pagado', 85.5), viaje_id: 1, cliente_id: 2 },
      { ...gasto(3, 'Sustentado', 310), viaje_id: 2, cliente_id: 1 },
      { ...gasto(4, 'Pagado', 40), viaje_id: 2, cliente_id: 2 },
    ];
    const numeros = (el: HTMLElement) => [...el.querySelectorAll('li.fila-gasto')].map(li => li.textContent?.match(/N° (\d+)/)?.[1]);

    it('ofrece solo los viajes y clientes que tienen gastos', () => {
      const el: HTMLElement = montar(storeFalso(variados)).nativeElement;
      const opciones = (id: string) => [...el.querySelectorAll(`#${id} option`)].map(o => o.textContent?.trim());
      expect(opciones('filtro-viaje')).toEqual(['Todos los viajes', 'Viaje N° 1', 'Viaje N° 2']);
      expect(opciones('filtro-cliente')).toEqual(['Todos los clientes', 'Cliente 1', 'Cliente 2']);
    });

    it('filtra por viaje y por cliente (de la URL), y el total y los conteos reflejan lo filtrado', () => {
      const el: HTMLElement = montar(storeFalso(variados), undefined, { viaje: '2', cliente: '2' }).nativeElement;
      expect(numeros(el)).toEqual(['4']);
      expect(el.querySelector('.lista-gastos__total')?.textContent).toContain('S/ 40.00');
      expect([...el.querySelectorAll('.pestanas__item')].map(a => a.textContent?.replace(/\s+/g, ' ').trim())).toEqual(['Todos 1', 'Sin comprobante 0', 'Por aprobar 1', 'Sustentados 0']);
    });

    it('un filtro que no existe o es inválido se ignora', () => {
      const el: HTMLElement = montar(storeFalso(variados), undefined, { viaje: 'abc', cliente: '-1' }).nativeElement;
      expect(numeros(el)).toEqual(['4', '3', '2', '1']);
    });

    it('elegir un viaje en el selector actualiza la URL sin perder lo demás', () => {
      const f = montar(storeFalso(variados), 'por-aprobar', { cliente: '2' });
      const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
      const selector: HTMLSelectElement = f.nativeElement.querySelector('#filtro-viaje');
      selector.value = '1';
      selector.dispatchEvent(new Event('change'));
      expect(navegar).toHaveBeenCalledWith([], expect.objectContaining({ queryParams: { viaje: '1' }, queryParamsHandling: 'merge' }));
      selector.value = '';
      selector.dispatchEvent(new Event('change'));
      expect(navegar).toHaveBeenLastCalledWith([], expect.objectContaining({ queryParams: { viaje: null } }));
    });

    it('las pestañas conservan el viaje y el cliente elegidos', () => {
      const el: HTMLElement = montar(storeFalso(variados), undefined, { viaje: '1', cliente: '2' }).nativeElement;
      const enlace = [...el.querySelectorAll<HTMLAnchorElement>('.pestanas__item')].find(a => a.textContent?.includes('Por aprobar'));
      expect(enlace?.getAttribute('href')).toBe('/app/gastos?estado=por-aprobar&viaje=1&cliente=2');
    });

    it('«Quitar filtros» aparece solo con un filtro activo y los limpia', () => {
      expect(montar(storeFalso(variados)).nativeElement.querySelector('.filtros__quitar')).toBeNull();
      TestBed.resetTestingModule();
      const f = montar(storeFalso(variados), undefined, { viaje: '1' });
      const quitar: HTMLAnchorElement = f.nativeElement.querySelector('.filtros__quitar');
      expect(quitar.getAttribute('href')).toBe('/app/gastos');
    });

    it('sin resultados explica que los filtros no encontraron gastos y ofrece quitarlos', () => {
      const el: HTMLElement = montar(storeFalso(variados), undefined, { viaje: '99' }).nativeElement;
      expect(el.textContent).toContain('Ningún gasto coincide con los filtros.');
    });
  });

  describe('exportar a Excel', () => {
    const lista = [gasto(1, 'Registrado', 120), gasto(2, 'Pagado', 85.5)];
    const boton = (el: HTMLElement) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes('Exportar a Excel')) as HTMLButtonElement;

    it('descarga con los filtros activos de la pantalla', () => {
      const excel = excelFalso();
      Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() });
      const clic = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
      const f = montar(storeFalso(lista), 'por-aprobar', { viaje: '1', cliente: '3' }, excel);
      boton(f.nativeElement).click();
      expect(excel.descargarExcel).toHaveBeenCalledWith({ viaje_id: 1, cliente_id: 3, estado: 'Pagado' });
      expect(clic).toHaveBeenCalledTimes(1);
    });

    it('sin filtros pide todo, y evita el doble clic mientras genera', () => {
      const excel = excelFalso(() => new Subject<never>());
      const f = montar(storeFalso(lista), undefined, {}, excel);
      boton(f.nativeElement).click();
      boton(f.nativeElement).click();
      expect(excel.descargarExcel).toHaveBeenCalledTimes(1);
      expect(excel.descargarExcel).toHaveBeenCalledWith({ viaje_id: null, cliente_id: null, estado: null });
      f.detectChanges();
      const generando = [...f.nativeElement.querySelectorAll('button')].find((b: HTMLButtonElement) => b.textContent?.includes('Generando…'));
      expect(generando?.disabled).toBe(true);
    });

    it('si falla, avisa con el mensaje de error de la app', () => {
      const err = new HttpErrorResponse({ status: 500 });
      const f = montar(storeFalso(lista), undefined, {}, excelFalso(() => throwError(() => err)));
      boton(f.nativeElement).click();
      f.detectChanges();
      expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe(mensajeError(err));
      expect(boton(f.nativeElement).disabled).toBe(false);
    });
  });
});
