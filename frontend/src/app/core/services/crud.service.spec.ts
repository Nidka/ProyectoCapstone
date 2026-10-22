import { beforeEach, describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { CrudService } from './crud.service';
import { ToastService } from './toast.service';

describe('CrudService', () => {
  let crud: CrudService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    crud = TestBed.inject(CrudService);
    http = TestBed.inject(HttpTestingController);
  });

  it('refsDe encuentra las relaciones de forma transitiva', () => {
    expect(crud.refsDe('cliente')).toEqual(['rubro']);
    expect(crud.refsDe('rubro')).toEqual([]);
    expect(new Set(crud.refsDe('viaje'))).toEqual(new Set([
      'unidad', 'cond_uni', 'cliente', 'direccion', 'tipo_unidad', 'subtipo_unidad', 'marca', 'conductor', 'rubro', 'ubigeo',
    ]));
  });

  it('etiqueta muestra #id hasta que llega la lista y luego el nombre', () => {
    expect(crud.etiqueta('cliente', 5)).toBe('#5');
    crud.data.set({ cliente: [{ id_cliente: 5, razon_social: 'Agro Norte SAC' }] });
    expect(crud.etiqueta('cliente', 5)).toBe('Agro Norte SAC');
    expect(crud.etiqueta('cliente', null)).toBe('—');
  });

  it('las etiquetas compuestas usan otras relaciones', () => {
    crud.data.set({
      conductor: [{ id_conductor: 4, nombre1: 'Juan', apellido1: 'Pérez' }],
      unidad: [{ id_unidad: 2, placa: 'ABC-123', modelo: 'Actros' }],
      cond_uni: [{ id_conductor_unidad: 9, id_conductor: 4, id_unidad: 2 }],
    });
    expect(crud.etiqueta('cond_uni', 9)).toBe('Juan Pérez · ABC-123 · Actros');
  });

  it('opciones arma valor y etiqueta para los selectores', () => {
    crud.data.set({ rubro: [{ id_rubro: 3, nombre: 'Agroindustria' }] });
    expect(crud.opciones('rubro')).toEqual([{ valor: '3', etiqueta: 'Agroindustria' }]);
  });

  it('loadList marca la carga y guarda los registros', () => {
    crud.loadList('rubro');
    expect(crud.estaCargando('rubro')).toBe(true);
    http.expectOne(r => r.url.endsWith('/rubros?limit=200&offset=0')).flush([{ id_rubro: 1, nombre: 'Minería' }]);
    expect(crud.estaCargando('rubro')).toBe(false);
    expect(crud.estaCargado('rubro')).toBe(true);
    expect(crud.getItems('rubro')).toHaveLength(1);
  });

  it('asegurarCargado no repite la petición', () => {
    crud.asegurarCargado('marca');
    crud.asegurarCargado('marca');
    http.expectOne(r => r.url.endsWith('/marcas?limit=200&offset=0')).flush([]);
    http.verify();
  });

  it('guardar emite el registro creado y avisa con el texto del módulo', () => {
    let resultado: unknown = 'sin respuesta';
    crud.guardar('marca', { nombre: 'Volvo' }).subscribe(r => (resultado = r));
    http.expectOne(r => r.method === 'POST' && r.url.endsWith('/marcas')).flush({ id_marca: 7, nombre: 'Volvo' });
    expect(resultado).toEqual({ id_marca: 7, nombre: 'Volvo' });
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Marca guardada');
    http.expectOne(r => r.method === 'GET' && r.url.endsWith('/marcas?limit=200&offset=0')).flush([]);
  });

  it('guardar falla con el detalle de la API para mostrarlo dentro de la hoja', () => {
    let error = '';
    crud.guardar('marca', {}).subscribe({ error: (e: Error) => (error = e.message) });
    http.expectOne(r => r.method === 'POST').flush({ detail: 'Ya existe una marca con ese nombre' }, { status: 409, statusText: 'Conflict' });
    expect(error).toBe('Ya existe una marca con ese nombre');
  });

  it('loadList trae la lista completa, de 200 en 200, hasta una página corta', () => {
    const pagina = (desde: number, n: number) => Array.from({ length: n }, (_, i) => ({ codigo_ubigeo: desde + i }));
    crud.loadList('ubigeo');
    http.expectOne(r => r.url.endsWith('/ubigeos?limit=200&offset=0')).flush(pagina(0, 200));
    http.expectOne(r => r.url.endsWith('/ubigeos?limit=200&offset=200')).flush(pagina(200, 3));
    http.verify();
    expect(crud.getItems('ubigeo')).toHaveLength(203);
    expect(crud.estaCargado('ubigeo')).toBe(true);
  });

  it('si la lista no carga, el módulo queda con error (no como vacío) hasta que una recarga funcione', () => {
    crud.loadList('rubro');
    http.expectOne(r => r.url.includes('/rubros?')).flush('caído', { status: 503, statusText: 'Service Unavailable' });
    expect(crud.estaConError('rubro')).toBe(true);
    expect(crud.estaCargado('rubro')).toBe(false);
    crud.loadList('rubro');
    http.expectOne(r => r.url.includes('/rubros?')).flush([{ id_rubro: 1, nombre: 'Minería' }]);
    expect(crud.estaConError('rubro')).toBe(false);
  });
});
