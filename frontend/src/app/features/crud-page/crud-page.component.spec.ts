import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Router, provideRouter } from '@angular/router';
import { CrudPageComponent } from './crud-page.component';
import { CrudService } from '../../core/services/crud.service';

function montar(seccion: 'operacion' | 'catalogos', modulo: string, preparar?: (crud: CrudService) => void, ver?: string) {
  TestBed.configureTestingModule({ providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()] });
  const crud = TestBed.inject(CrudService);
  preparar?.(crud);
  const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
  const f = TestBed.createComponent(CrudPageComponent);
  f.componentRef.setInput('seccion', seccion);
  f.componentRef.setInput('modulo', modulo);
  if (ver !== undefined) f.componentRef.setInput('ver', ver);
  f.detectChanges();
  return { el: f.nativeElement as HTMLElement, navegar, f };
}

/** jsdom no implementa showModal/close de <dialog>: se simulan con el atributo open. */
function conDialogos(): void {
  const proto = HTMLDialogElement.prototype as unknown as { showModal: () => void; close: () => void };
  proto.showModal = function (this: HTMLDialogElement) {
    this.setAttribute('open', '');
  };
  proto.close = function (this: HTMLDialogElement) {
    this.removeAttribute('open');
  };
}

function clientesCargados(crud: CrudService): void {
  crud.data.set({
    cliente: [{ id_cliente: 1, razon_social: 'Agro Norte SAC', ruc: '20123456789', id_rubro: 3 }],
    rubro: [{ id_rubro: 3, nombre: 'Agroindustria' }],
  });
  crud.cargado.set({ cliente: true, rubro: true });
}

interface Pagina {
  nuevo(): void;
  guardar(): void;
  campos(): { value: string }[];
}

describe('CrudPageComponent', () => {
  it('un módulo desconocido redirige al primero de la sección', () => {
    const { navegar } = montar('catalogos', 'xyz');
    expect(navegar).toHaveBeenCalledWith(['/app', 'catalogos', 'ubigeo'], { replaceUrl: true });
  });

  it('la tabla muestra nombres relacionados en lugar de IDs', () => {
    const { el } = montar('operacion', 'clientes', crud => {
      crud.data.set({
        cliente: [{ id_cliente: 1, razon_social: 'Agro Norte SAC', ruc: '20123456789', id_rubro: 3 }],
        rubro: [{ id_rubro: 3, nombre: 'Agroindustria' }],
      });
      crud.cargado.set({ cliente: true, rubro: true });
    });
    expect(el.querySelector('h1')?.textContent).toBe('Clientes');
    const fila = el.querySelector('tbody tr');
    expect(fila?.textContent).toContain('Agro Norte SAC');
    expect(fila?.textContent).toContain('Agroindustria');
  });

  it('en Viajes cada fila lleva a registrar un gasto de ese viaje', () => {
    const { el } = montar('operacion', 'viajes', crud => {
      crud.data.set({
        viaje: [{ id_viaje: 14, id_tracto: 2, id_cliente: 1, id_direccion_origen: 5, id_direccion_destino: 6, fecha_programado: '2026-09-29' }],
        unidad: [{ id_unidad: 2, placa: 'T4K-829', modelo: 'FH 460' }],
        cond_uni: [{ id_conductor_unidad: 1, id_conductor: 4, id_unidad: 2 }],
        cliente: [{ id_cliente: 1, razon_social: 'Agro Norte SAC' }],
        direccion: [{ id_direccion: 5, direccion_detallada: 'Av. A 100' }, { id_direccion: 6, direccion_detallada: 'Av. B 200' }],
      });
      crud.cargado.set({ viaje: true, unidad: true, cond_uni: true, cliente: true, direccion: true });
    });
    const accion = [...el.querySelectorAll('tbody a')].find(a => a.textContent?.includes('Registrar gasto'));
    expect(accion?.getAttribute('href')).toBe('/app/gastos/nuevo?viaje=14');
    // «Ver registro» se ve igual que en los demás módulos: con su texto, no solo el icono
    const ver = [...el.querySelectorAll<HTMLButtonElement>('tbody button')].find(b => b.textContent?.includes('Ver registro'));
    expect(ver?.classList).not.toContain('btn--icono');
    expect(ver?.querySelector('.oculto-visual')).toBeNull();
  });

  it('«Ver registro» muestra los campos con su nombre, no el JSON', () => {
    conDialogos();
    const { el, f } = montar('operacion', 'clientes', clientesCargados);
    [...el.querySelectorAll<HTMLButtonElement>('tbody button')].find(b => b.textContent?.includes('Ver registro'))!.click();
    f.detectChanges();
    const dialogo = el.querySelector('dialog.dialogo')!;
    expect(dialogo.querySelector('pre')).toBeNull();
    const campos = [...dialogo.querySelectorAll('.registro__fila')].map(r => [r.querySelector('dt')?.textContent?.trim(), r.querySelector('dd')?.textContent?.trim()]);
    expect(campos).toContainEqual(['Rubro', 'Agroindustria']);
    expect(campos).toContainEqual(['RUC', '20123456789']);
  });

  it('con ?ver= abre la ficha de ese registro (enlaces desde el detalle de un gasto)', () => {
    conDialogos();
    const { el } = montar('operacion', 'clientes', clientesCargados, '1');
    const dialogo = el.querySelector('dialog.dialogo')!;
    expect(dialogo.hasAttribute('open')).toBe(true);
    expect(dialogo.querySelector('h2')?.textContent).toContain('Agro Norte SAC');
  });

  it('la ficha abierta muestra el nombre relacionado apenas llega su lista', () => {
    conDialogos();
    const { el, f } = montar('operacion', 'clientes', crud => {
      crud.data.set({ cliente: [{ id_cliente: 1, razon_social: 'Agro Norte SAC', ruc: '20123456789', id_rubro: 3 }] });
      crud.cargado.set({ cliente: true });
    }, '1');
    const crud = TestBed.inject(CrudService);
    crud.data.update(d => ({ ...d, rubro: [{ id_rubro: 3, nombre: 'Agroindustria' }] }));
    crud.cargado.update(c => ({ ...c, rubro: true }));
    f.detectChanges();
    const rubro = [...el.querySelectorAll('dialog.dialogo .registro__fila')].find(r => r.querySelector('dt')?.textContent?.trim() === 'Rubro');
    expect(rubro?.querySelector('dd')?.textContent?.trim()).toBe('Agroindustria');
  });

  it('el título no repite la sección encima (ya la marcan la barra y las pestañas)', () => {
    const { el } = montar('catalogos', 'rubros', crud => {
      crud.data.set({ rubro: [] });
      crud.cargado.set({ rubro: true });
    });
    expect(el.querySelector('.migas')).toBeNull();
  });

  it('si la lista no carga, muestra el error con Reintentar y no «Aún no hay…»', () => {
    const { el, f } = montar('catalogos', 'rubros');
    TestBed.inject(HttpTestingController)
      .expectOne(r => r.url.includes('/rubros?'))
      .flush('caído', { status: 503, statusText: 'Service Unavailable' });
    f.detectChanges();
    expect(el.textContent).toContain('No se pudo cargar la lista de rubros.');
    expect(el.textContent).not.toContain('Aún no hay rubros.');
    expect(el.textContent).not.toContain('0 registros');
    expect([...el.querySelectorAll('button')].some(b => b.textContent?.trim() === 'Reintentar')).toBe(true);
  });

  it('si guardar falla, el motivo se ve dentro de la hoja (un aviso quedaría tapado por el modal)', () => {
    conDialogos();
    const { el, f } = montar('catalogos', 'rubros', crud => {
      crud.data.set({ rubro: [] });
      crud.cargado.set({ rubro: true });
    });
    const pagina = f.componentInstance as unknown as Pagina;
    pagina.nuevo();
    f.detectChanges();
    for (const c of pagina.campos()) c.value = 'Minería';
    pagina.guardar();
    TestBed.inject(HttpTestingController)
      .expectOne(r => r.method === 'POST')
      .flush({ detail: 'Ya existe un rubro con ese nombre' }, { status: 409, statusText: 'Conflict' });
    f.detectChanges();
    expect(el.querySelector('.hoja [role="alert"]')?.textContent).toContain('Ya existe un rubro con ese nombre');
    expect(el.querySelector('dialog.hoja')?.hasAttribute('open')).toBe(true);
  });

  it('si una lista relacionada no carga, su selector lo dice en vez de «Aún no hay…»', () => {
    conDialogos();
    const { el, f } = montar('operacion', 'clientes', crud => {
      crud.data.set({ cliente: [] });
      crud.cargado.set({ cliente: true });
    });
    TestBed.inject(HttpTestingController)
      .expectOne(r => r.url.includes('/rubros?'))
      .flush('caído', { status: 503, statusText: 'Service Unavailable' });
    (f.componentInstance as unknown as Pagina).nuevo();
    f.detectChanges();
    el.querySelector<HTMLInputElement>('#cl_rb')!.dispatchEvent(new FocusEvent('focus'));
    f.detectChanges();
    expect(el.textContent).toContain('No se pudo cargar la lista.');
    expect(el.textContent).not.toContain('Aún no hay rubros.');
  });

  it('al cambiar de módulo (p. ej. con Atrás) se cierra la hoja abierta', () => {
    conDialogos();
    const { el, f } = montar('catalogos', 'rubros', crud => {
      crud.data.set({ rubro: [], marca: [] });
      crud.cargado.set({ rubro: true, marca: true });
    });
    (f.componentInstance as unknown as Pagina).nuevo();
    f.detectChanges();
    expect(el.querySelector('dialog.hoja')?.hasAttribute('open')).toBe(true);
    f.componentRef.setInput('modulo', 'marcas');
    f.detectChanges();
    expect(el.querySelector('dialog.hoja')?.hasAttribute('open')).toBe(false);
    expect((f.componentInstance as unknown as Pagina).campos()).toEqual([]);
  });

  it('avisa los datos previos que faltan con enlaces para crearlos', () => {
    const { el } = montar('operacion', 'viajes', crud => {
      crud.data.set({ viaje: [], unidad: [], cond_uni: [], cliente: [{ id_cliente: 1, razon_social: 'Agro Norte SAC' }], direccion: [] });
      crud.cargado.set({ viaje: true, unidad: true, cond_uni: true, cliente: true, direccion: true });
    });
    const enlaces = [...el.querySelectorAll('.alerta a')].map(a => a.textContent?.trim());
    expect(enlaces).toEqual(['unidades', 'asignaciones', 'direcciones']);
    expect(el.textContent).toContain('Antes de crear viajes necesitas: unidades, asignaciones, clientes y direcciones.');
  });
});
