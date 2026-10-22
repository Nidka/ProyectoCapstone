import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { Observable, Subject, of, throwError } from 'rxjs';
import { ChecklistClienteComponent } from './checklist-cliente.component';
import { ChecklistService } from '../../core/services/checklist.service';
import { CrudService } from '../../core/services/crud.service';
import { ToastService } from '../../core/services/toast.service';
import { ChecklistCliente } from '../../core/models/checklist';
import { mensajeError } from '../../core/utils/error-messages';

const ESTANDAR: ChecklistCliente = {
  cliente_id: 3,
  personalizado: false,
  criterios: [
    { clave: 'legible', texto: 'El comprobante es legible y está completo.' },
    { clave: 'tipo_valido', texto: 'Es una boleta o factura válida.' },
    { clave: 'monto_coincide', texto: 'El monto coincide con el gasto.' },
  ],
};

interface Opciones {
  cliente?: string;
  checklist?: () => Observable<ChecklistCliente>;
  guardar?: (id: number, criterios: { clave?: string; texto: string }[]) => Observable<ChecklistCliente>;
  restablecer?: () => Observable<ChecklistCliente>;
}

async function montar(op: Opciones = {}) {
  const servicio = {
    deCliente: vi.fn(op.checklist ?? (() => of(ESTANDAR))),
    guardar: vi.fn(op.guardar ?? ((id: number, criterios: { clave?: string; texto: string }[]) =>
      of({ cliente_id: id, personalizado: true, criterios: criterios.map((c, i) => ({ clave: c.clave ?? `c-nuevo${i}`, texto: c.texto })) }))),
    restablecer: vi.fn(op.restablecer ?? (() => of(ESTANDAR))),
  };
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: ChecklistService, useValue: servicio },
      { provide: CrudService, useValue: { etiqueta: () => 'Agro Norte SAC', asegurarCargado: vi.fn() } },
    ],
  });
  const f = TestBed.createComponent(ChecklistClienteComponent);
  if (op.cliente !== undefined) f.componentRef.setInput('cliente', op.cliente);
  else f.componentRef.setInput('cliente', '3');
  const refrescar = async () => {
    f.detectChanges();
    await f.whenStable();
    f.detectChanges();
  };
  await refrescar();
  const el = f.nativeElement as HTMLElement;
  const boton = (texto: string) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes(texto) || b.getAttribute('aria-label')?.includes(texto)) as HTMLButtonElement;
  const textos = () => [...el.querySelectorAll<HTMLInputElement>('.criterio__texto')].map(i => i.value);
  const escribir = async (input: HTMLInputElement, valor: string) => {
    input.value = valor;
    input.dispatchEvent(new Event('input'));
    await refrescar();
  };
  return { f, el, servicio, boton, textos, escribir, refrescar };
}

describe('ChecklistClienteComponent', () => {
  it('muestra el cliente, que usa el checklist estándar, y sus criterios editables', async () => {
    const { el, servicio, textos } = await montar();
    expect(servicio.deCliente).toHaveBeenCalledWith(3);
    expect(el.querySelector('h1')?.textContent).toContain('Agro Norte SAC');
    expect(el.textContent).toContain('Usa el checklist estándar');
    expect(textos()).toEqual(ESTANDAR.criterios.map(c => c.texto));
  });

  it('con criterios propios lo dice y ofrece restablecer al estándar', async () => {
    const propio = { ...ESTANDAR, personalizado: true };
    const { el, boton } = await montar({ checklist: () => of(propio) });
    expect(el.textContent).toContain('Tiene un checklist propio');
    expect(boton('Restablecer al estándar')).toBeTruthy();
  });

  it('sin cambios no se puede guardar', async () => {
    const { boton } = await montar();
    expect(boton('Guardar checklist').disabled).toBe(true);
  });

  it('edita, agrega, mueve y quita criterios, y guarda con la clave de los que ya existían', async () => {
    const { el, servicio, boton, textos, escribir, refrescar } = await montar();
    const entradas = () => [...el.querySelectorAll<HTMLInputElement>('.criterio__texto')];
    await escribir(entradas()[0], 'Se lee completo');
    await escribir(el.querySelector<HTMLInputElement>('#nuevo-criterio')!, 'Lleva sello de recepción');
    boton('Agregar criterio').click();
    await refrescar();
    boton('Bajar criterio 1').click();
    await refrescar();
    boton('Quitar criterio 3').click(); // monto_coincide
    await refrescar();
    expect(textos()).toEqual(['Es una boleta o factura válida.', 'Se lee completo', 'Lleva sello de recepción']);
    boton('Guardar checklist').click();
    expect(servicio.guardar).toHaveBeenCalledWith(3, [
      { clave: 'tipo_valido', texto: 'Es una boleta o factura válida.' },
      { clave: 'legible', texto: 'Se lee completo' },
      { texto: 'Lleva sello de recepción' },
    ]);
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Checklist guardado.');
  });

  it('valida el largo y los repetidos antes de enviar', async () => {
    const { el, servicio, boton, escribir } = await montar();
    const entradas = () => [...el.querySelectorAll<HTMLInputElement>('.criterio__texto')];
    await escribir(entradas()[0], 'ab');
    expect(el.textContent).toContain('Escribe entre 3 y 120 caracteres.');
    expect(boton('Guardar checklist').disabled).toBe(true);
    await escribir(entradas()[0], 'es UNA boleta o factura válida.');
    expect(el.textContent).toContain('Este criterio está repetido.');
    expect(boton('Guardar checklist').disabled).toBe(true);
    expect(servicio.guardar).not.toHaveBeenCalled();
  });

  it('no deja quitar el último criterio ni pasar de 15', async () => {
    const uno = { ...ESTANDAR, criterios: [ESTANDAR.criterios[0]] };
    const { boton: botonUno } = await montar({ checklist: () => of(uno) });
    expect(botonUno('Quitar criterio 1').disabled).toBe(true);
    TestBed.resetTestingModule();
    const quince = { ...ESTANDAR, criterios: Array.from({ length: 15 }, (_, i) => ({ clave: `k${i}`, texto: `Criterio número ${i + 1}` })) };
    const { el, boton } = await montar({ checklist: () => of(quince) });
    expect(boton('Agregar criterio').disabled).toBe(true);
    expect(el.textContent).toContain('15 de 15 criterios');
  });

  it('restablecer pide confirmación y deja el checklist estándar', async () => {
    const propio = { ...ESTANDAR, personalizado: true, criterios: [{ clave: 'x', texto: 'Solo este criterio' }] };
    const { el, servicio, boton, refrescar } = await montar({ checklist: () => of(propio) });
    boton('Restablecer al estándar').click();
    await refrescar();
    expect(el.textContent).toContain('se pierden los criterios propios');
    expect(servicio.restablecer).not.toHaveBeenCalled();
    boton('Sí, restablecer').click();
    await refrescar();
    expect(servicio.restablecer).toHaveBeenCalledWith(3);
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Checklist restablecido al estándar.');
    expect(el.textContent).toContain('Usa el checklist estándar');
  });

  it('evita el doble clic mientras guarda', async () => {
    const pendiente = new Subject<ChecklistCliente>();
    const { el, servicio, boton, escribir } = await montar({ guardar: () => pendiente });
    await escribir(el.querySelector<HTMLInputElement>('.criterio__texto')!, 'Se lee completo');
    boton('Guardar checklist').click();
    boton('Guardar checklist').click();
    expect(servicio.guardar).toHaveBeenCalledTimes(1);
  });

  it('si el servidor rechaza el checklist (422) lo explica; otro error usa el mensaje de la app', async () => {
    const { el, boton, escribir } = await montar({ guardar: () => throwError(() => new HttpErrorResponse({ status: 422 })) });
    await escribir(el.querySelector<HTMLInputElement>('.criterio__texto')!, 'Se lee completo');
    boton('Guardar checklist').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('No se pudo guardar el checklist. Revisa los criterios.');
    TestBed.resetTestingModule();
    const err = new HttpErrorResponse({ status: 500 });
    const m = await montar({ guardar: () => throwError(() => err) });
    await m.escribir(m.el.querySelector<HTMLInputElement>('.criterio__texto')!, 'Se lee completo');
    m.boton('Guardar checklist').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe(mensajeError(err));
  });

  it('sin cliente en la URL pide elegir uno', async () => {
    const { el, servicio } = await montar({ cliente: 'abc' });
    expect(el.textContent).toContain('Elige un cliente');
    expect(servicio.deCliente).not.toHaveBeenCalled();
  });

  it('si no carga ofrece reintentar', async () => {
    let fallar = true;
    const { el, servicio, boton, refrescar } = await montar({ checklist: () => (fallar ? throwError(() => new HttpErrorResponse({ status: 0 })) : of(ESTANDAR)) });
    expect(el.textContent).toContain('No se pudo cargar el checklist.');
    fallar = false;
    boton('Reintentar').click();
    await refrescar();
    expect(servicio.deCliente).toHaveBeenCalledTimes(2);
    expect(el.querySelectorAll('.criterio__texto')).toHaveLength(3);
  });
});
