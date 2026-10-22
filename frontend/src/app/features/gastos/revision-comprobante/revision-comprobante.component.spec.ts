import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, Subject, of, throwError } from 'rxjs';
import { RevisionComprobanteComponent } from './revision-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ChecklistService } from '../../../core/services/checklist.service';
import { ToastService } from '../../../core/services/toast.service';
import { ChecklistCliente } from '../../../core/models/checklist';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { mensajeError } from '../../../core/utils/error-messages';

const comprobante: Comprobante = {
  id: 9, gasto_id: 7, archivo_url: '/uploads/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z',
};
const gasto: Gasto = {
  id: 7, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T13:00:00Z',
};
const ESTANDAR: ChecklistCliente = {
  cliente_id: 1,
  personalizado: false,
  criterios: [
    { clave: 'legible', texto: 'El comprobante es legible y está completo.' },
    { clave: 'tipo_valido', texto: 'Es una boleta o factura válida.' },
    { clave: 'a_nombre_empresa', texto: 'Está emitido a nombre de LIZAR SRL.' },
    { clave: 'monto_coincide', texto: 'El monto coincide con el gasto.' },
    { clave: 'fecha_coincide', texto: 'La fecha corresponde al gasto.' },
  ],
};
const TODO = { legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true };

interface Opciones {
  respuesta?: () => Observable<unknown>;
  checklist?: () => Observable<ChecklistCliente>;
  /** false = no esperar a que la carga termine (para probar el estado «cargando») */
  esperar?: boolean;
}

async function montar(op: Opciones = {}) {
  const store = { aprobar: vi.fn(op.respuesta ?? (() => of({}))), rechazar: vi.fn(op.respuesta ?? (() => of({}))), cargar: vi.fn() };
  const checklists = { deCliente: vi.fn(op.checklist ?? (() => of(ESTANDAR))) };
  TestBed.configureTestingModule({
    providers: [
      { provide: GastosStore, useValue: store },
      { provide: ChecklistService, useValue: checklists },
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => `http://localhost:8081${u}` } },
    ],
  });
  const f = TestBed.createComponent(RevisionComprobanteComponent);
  f.componentRef.setInput('comprobante', comprobante);
  f.componentRef.setInput('gasto', gasto);
  const refrescar = async () => {
    f.detectChanges();
    if (op.esperar !== false) await f.whenStable();
    f.detectChanges();
  };
  await refrescar();
  const el = f.nativeElement as HTMLElement;
  const boton = (texto: string) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes(texto)) as HTMLButtonElement;
  const casillas = () => [...el.querySelectorAll<HTMLInputElement>('.checklist input[type="checkbox"]')];
  const marcar = async (cuantas: number) => {
    casillas().slice(0, cuantas).forEach(c => c.click());
    await refrescar();
  };
  return { f, el, store, checklists, boton, casillas, marcar, refrescar };
}

describe('RevisionComprobanteComponent', () => {
  it('muestra el comprobante para revisarlo', async () => {
    const { el } = await montar();
    expect(el.querySelector('img')?.getAttribute('src')).toBe('http://localhost:8081/uploads/9.jpg');
  });

  it('pide los criterios del cliente del gasto y los muestra con el monto y la fecha', async () => {
    const { el, checklists, casillas } = await montar();
    expect(checklists.deCliente).toHaveBeenCalledWith(1);
    expect(el.querySelector('.checklist legend')?.textContent?.trim()).toBe('Checklist documental');
    expect(casillas()).toHaveLength(5);
    const textos = [...el.querySelectorAll('.checklist__criterio')].map(c => c.textContent?.trim());
    expect(textos).toContain('El monto coincide con el gasto (S/ 85.50).');
    expect(textos).toContain('La fecha corresponde al gasto (30/09/2026).');
  });

  it('muestra exactamente los criterios personalizados del cliente', async () => {
    const propio: ChecklistCliente = {
      cliente_id: 1, personalizado: true,
      criterios: [{ clave: 'legible', texto: 'Se lee completo' }, { clave: 'c-9f2a11', texto: 'Lleva sello de recepción' }],
    };
    const { el, store, boton, marcar, casillas } = await montar({ checklist: () => of(propio) });
    expect(casillas()).toHaveLength(2);
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('0 de 2 criterios');
    await marcar(2);
    expect(boton('Aprobar').disabled).toBe(false);
    boton('Aprobar').click();
    expect(store.aprobar).toHaveBeenCalledWith(9, { legible: true, 'c-9f2a11': true });
  });

  it('Aprobar queda deshabilitado hasta marcar todos los criterios', async () => {
    const { el, boton, marcar } = await montar();
    expect(boton('Aprobar').disabled).toBe(true);
    expect(el.querySelector('#checklist-aviso')?.textContent?.trim()).toBe('Marca todos los criterios para aprobar.');
    expect(boton('Aprobar').getAttribute('aria-describedby')).toBe('checklist-aviso');
    await marcar(4);
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('4 de 5 criterios');
    expect(boton('Aprobar').disabled).toBe(true);
  });

  it('con todos los criterios, aprobar envía el checklist y avisa con el copy exacto', async () => {
    const { el, store, boton, marcar } = await montar();
    await marcar(5);
    expect(boton('Aprobar').disabled).toBe(false);
    expect(el.querySelector('#checklist-aviso')).toBeNull();
    boton('Aprobar').click();
    expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Comprobante aprobado. El gasto quedó como "Sustentado".');
  });

  it('Aprobar va en verde de guía, Rechazar es neutro y el rojo queda para confirmar el rechazo', async () => {
    const { f, boton } = await montar();
    expect(boton('Aprobar').classList).toContain('btn--guia');
    expect(boton('Rechazar').classList).toContain('btn--secundario');
    boton('Rechazar').click();
    f.detectChanges();
    expect(boton('Confirmar rechazo').classList).toContain('btn--peligro');
  });

  it('rechazar propone el motivo con los criterios sin marcar y envía el checklist', async () => {
    const { el, store, boton, marcar, refrescar } = await montar();
    await marcar(3);
    boton('Rechazar').click();
    await refrescar();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    expect(area.value).toBe('No cumple: el monto coincide con el gasto; la fecha corresponde al gasto.');
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'No cumple: el monto coincide con el gasto; la fecha corresponde al gasto.', {
      ...TODO, monto_coincide: false, fecha_coincide: false,
    });
  });

  it('rechazar exige un motivo (con todo marcado no se propone ninguno) y lo envía sin espacios sobrantes', async () => {
    const { f, el, store, boton, marcar } = await montar();
    await marcar(5);
    boton('Rechazar').click();
    f.detectChanges();
    boton('Confirmar rechazo').click();
    f.detectChanges();
    expect(el.querySelector('#motivo-error')?.textContent?.trim()).toBe('Debes indicar un motivo para rechazar el comprobante.');
    expect(store.rechazar).not.toHaveBeenCalled();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    area.value = '  Comprobante ilegible  ';
    area.dispatchEvent(new Event('input'));
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'Comprobante ilegible', TODO);
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Comprobante rechazado. El gasto volvió a estado "Registrado".');
    expect(TestBed.inject(ToastService).toasts().at(-1)?.type).toBe('info');
  });

  it('con el rechazo abierto, el motivo propuesto sigue a las casillas mientras no se edite', async () => {
    const { el, boton, marcar, casillas, refrescar } = await montar();
    await marcar(3);
    boton('Rechazar').click();
    await refrescar();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    expect(area.value).toBe('No cumple: el monto coincide con el gasto; la fecha corresponde al gasto.');
    casillas()[3].click();
    await refrescar();
    expect(area.value).toBe('No cumple: la fecha corresponde al gasto.');
    area.value = 'La boleta está a nombre de otra empresa';
    area.dispatchEvent(new Event('input'));
    casillas()[4].click();
    await refrescar();
    expect(area.value).toBe('La boleta está a nombre de otra empresa');
  });

  it('si el comprobante ya fue revisado (409), lo avisa y recarga', async () => {
    const { store, boton, marcar } = await montar({ respuesta: () => throwError(() => new HttpErrorResponse({ status: 409 })) });
    await marcar(5);
    boton('Aprobar').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Este comprobante ya fue revisado. Actualizamos la vista.');
    expect(store.cargar).toHaveBeenCalled();
  });

  it('si el servidor rechaza el checklist al aprobar (422), lo explica y vuelve a pedir los criterios', async () => {
    const { checklists, boton, marcar, refrescar } = await montar({ respuesta: () => throwError(() => new HttpErrorResponse({ status: 422 })) });
    await marcar(5);
    boton('Aprobar').click();
    await refrescar();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Para aprobar, el comprobante debe cumplir todos los criterios del checklist.');
    // El checklist del cliente pudo cambiar mientras se revisaba
    expect(checklists.deCliente).toHaveBeenCalledTimes(2);
  });

  it('si el servidor responde 422 al rechazar, usa el mensaje general de la app', async () => {
    const err = new HttpErrorResponse({ status: 422 });
    const { f, boton, marcar } = await montar({ respuesta: () => throwError(() => err) });
    await marcar(2);
    boton('Rechazar').click();
    f.detectChanges();
    boton('Confirmar rechazo').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe(mensajeError(err));
  });

  it('lo marcado se conserva con una recarga del mismo comprobante y se reinicia con otro', async () => {
    const { f, el, marcar, refrescar } = await montar();
    await marcar(2);
    f.componentRef.setInput('comprobante', { ...comprobante });
    await refrescar();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('2 de 5 criterios');
    f.componentRef.setInput('comprobante', { ...comprobante, id: 10 });
    await refrescar();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('0 de 5 criterios');
  });

  it('mientras carga los criterios no se puede aprobar, pero sí rechazar (sin checklist)', async () => {
    const pendiente = new Subject<ChecklistCliente>();
    const { f, el, store, boton } = await montar({ checklist: () => pendiente, esperar: false });
    expect(el.textContent).toContain('Cargando checklist…');
    expect(boton('Aprobar').disabled).toBe(true);
    boton('Rechazar').click();
    f.detectChanges();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    area.value = 'Ilegible';
    area.dispatchEvent(new Event('input'));
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'Ilegible', undefined);
  });

  it('si no se pueden cargar los criterios lo dice, ofrece reintentar y no deja aprobar', async () => {
    let fallar = true;
    const { el, checklists, boton, refrescar } = await montar({
      checklist: () => (fallar ? throwError(() => new HttpErrorResponse({ status: 0 })) : of(ESTANDAR)),
    });
    expect(el.textContent).toContain('No se pudo cargar el checklist de este cliente.');
    expect(boton('Aprobar').disabled).toBe(true);
    fallar = false;
    boton('Reintentar').click();
    await refrescar();
    expect(checklists.deCliente).toHaveBeenCalledTimes(2);
    expect(el.querySelectorAll('.checklist input[type="checkbox"]')).toHaveLength(5);
  });
});
