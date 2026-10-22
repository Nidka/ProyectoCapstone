import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Observable, of, throwError } from 'rxjs';
import { RevisionComprobanteComponent } from './revision-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { mensajeError } from '../../../core/utils/error-messages';

const comprobante: Comprobante = {
  id: 9, gasto_id: 7, archivo_url: '/uploads/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z',
};
const gasto: Gasto = {
  id: 7, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T13:00:00Z',
};
const TODO = { legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true };

function montar(respuesta: () => Observable<unknown> = () => of({})) {
  const store = { aprobar: vi.fn(respuesta), rechazar: vi.fn(respuesta), cargar: vi.fn() };
  TestBed.configureTestingModule({
    providers: [
      { provide: GastosStore, useValue: store },
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => `http://localhost:8081${u}` } },
    ],
  });
  const f = TestBed.createComponent(RevisionComprobanteComponent);
  f.componentRef.setInput('comprobante', comprobante);
  f.componentRef.setInput('gasto', gasto);
  f.detectChanges();
  const el = f.nativeElement as HTMLElement;
  const boton = (texto: string) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes(texto)) as HTMLButtonElement;
  const casillas = () => [...el.querySelectorAll<HTMLInputElement>('.checklist input[type="checkbox"]')];
  const marcar = (cuantas: number) => {
    casillas().slice(0, cuantas).forEach(c => c.click());
    f.detectChanges();
  };
  return { f, el, store, boton, casillas, marcar };
}

describe('RevisionComprobanteComponent', () => {
  it('muestra el comprobante para revisarlo', () => {
    const { el } = montar();
    expect(el.querySelector('img')?.getAttribute('src')).toBe('http://localhost:8081/uploads/9.jpg');
  });

  it('muestra los 5 criterios con el monto y la fecha del gasto', () => {
    const { el, casillas } = montar();
    expect(el.querySelector('.checklist legend')?.textContent?.trim()).toBe('Checklist documental');
    expect(casillas()).toHaveLength(5);
    const textos = [...el.querySelectorAll('.checklist__criterio')].map(c => c.textContent?.trim());
    expect(textos).toContain('El monto coincide con el gasto (S/ 85.50).');
    expect(textos).toContain('La fecha corresponde al gasto (30/09/2026).');
  });

  it('Aprobar queda deshabilitado hasta marcar los 5 criterios', () => {
    const { el, boton, marcar } = montar();
    expect(boton('Aprobar').disabled).toBe(true);
    expect(el.querySelector('#checklist-aviso')?.textContent?.trim()).toBe('Marca los 5 criterios para aprobar.');
    expect(boton('Aprobar').getAttribute('aria-describedby')).toBe('checklist-aviso');
    marcar(4);
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('4 de 5 criterios');
    expect(boton('Aprobar').disabled).toBe(true);
  });

  it('con los 5 criterios, aprobar envía el checklist y avisa con el copy exacto', () => {
    const { el, store, boton, marcar } = montar();
    marcar(5);
    expect(boton('Aprobar').disabled).toBe(false);
    expect(el.querySelector('#checklist-aviso')).toBeNull();
    boton('Aprobar').click();
    expect(store.aprobar).toHaveBeenCalledWith(9, TODO);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Comprobante aprobado. El gasto quedó como "Sustentado".');
  });

  it('Aprobar va en verde de guía, Rechazar es neutro y el rojo queda para confirmar el rechazo', () => {
    const { f, boton } = montar();
    expect(boton('Aprobar').classList).toContain('btn--guia');
    expect(boton('Rechazar').classList).toContain('btn--secundario');
    boton('Rechazar').click();
    f.detectChanges();
    expect(boton('Confirmar rechazo').classList).toContain('btn--peligro');
  });

  it('rechazar propone el motivo con los criterios sin marcar y envía el checklist', async () => {
    const { f, el, store, boton, marcar } = montar();
    marcar(3);
    boton('Rechazar').click();
    f.detectChanges();
    // ngModel escribe el valor en el textarea en una microtarea
    await f.whenStable();
    f.detectChanges();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    expect(area.value).toBe('No cumple: monto coincide; fecha coincide.');
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'No cumple: monto coincide; fecha coincide.', {
      ...TODO, monto_coincide: false, fecha_coincide: false,
    });
  });

  it('rechazar exige un motivo (con los 5 criterios marcados no se propone ninguno) y lo envía sin espacios sobrantes', () => {
    const { f, el, store, boton, marcar } = montar();
    marcar(5);
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
    // Un rechazo no lleva la señal verde de guía: es un aviso informativo
    expect(TestBed.inject(ToastService).toasts().at(-1)?.type).toBe('info');
  });

  it('si el comprobante ya fue revisado (409), lo avisa y recarga', () => {
    const { store, boton, marcar } = montar(() => throwError(() => new HttpErrorResponse({ status: 409 })));
    marcar(5);
    boton('Aprobar').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Este comprobante ya fue revisado. Actualizamos la vista.');
    expect(store.cargar).toHaveBeenCalled();
  });

  it('si el servidor rechaza el checklist (422), lo explica', () => {
    const { boton, marcar } = montar(() => throwError(() => new HttpErrorResponse({ status: 422 })));
    marcar(5);
    boton('Aprobar').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Para aprobar, el comprobante debe cumplir los 5 criterios del checklist.');
  });

  it('con el rechazo abierto, el motivo propuesto sigue a las casillas mientras no se edite', async () => {
    const { f, el, boton, marcar, casillas } = montar();
    const refrescar = async () => {
      f.detectChanges();
      await f.whenStable();
      f.detectChanges();
    };
    marcar(3);
    boton('Rechazar').click();
    await refrescar();
    const area = el.querySelector<HTMLTextAreaElement>('#motivo-rechazo')!;
    expect(area.value).toBe('No cumple: monto coincide; fecha coincide.');
    casillas()[3].click();
    await refrescar();
    expect(area.value).toBe('No cumple: fecha coincide.');
    area.value = 'La boleta está a nombre de otra empresa';
    area.dispatchEvent(new Event('input'));
    casillas()[4].click();
    await refrescar();
    expect(area.value).toBe('La boleta está a nombre de otra empresa');
  });

  it('si el servidor responde 422 al rechazar, usa el mensaje general de la app', () => {
    const err = new HttpErrorResponse({ status: 422 });
    const { f, boton, marcar } = montar(() => throwError(() => err));
    marcar(2);
    boton('Rechazar').click();
    f.detectChanges();
    boton('Confirmar rechazo').click();
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe(mensajeError(err));
  });

  it('el checklist vuelve a vacío solo si cambia el comprobante', () => {
    const { f, el, marcar } = montar();
    marcar(2);
    f.componentRef.setInput('comprobante', { ...comprobante });
    f.detectChanges();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('2 de 5 criterios');
    f.componentRef.setInput('comprobante', { ...comprobante, id: 10 });
    f.detectChanges();
    expect(el.querySelector('.checklist__conteo')?.textContent?.trim()).toBe('0 de 5 criterios');
  });
});
