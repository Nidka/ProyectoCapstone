import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Router, provideRouter } from '@angular/router';
import { Observable, of, throwError } from 'rxjs';
import { GastoFormComponent } from './gasto-form.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { CrudService } from '../../../core/services/crud.service';
import { ToastService } from '../../../core/services/toast.service';

interface Formulario {
  viajeId: { set(v: string): void };
  clienteId: { set(v: string): void };
  monto: string;
  fecha: string;
}

function montar(registrar: () => Observable<unknown> = vi.fn(() => of({ id: 31 }))) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: GastosStore, useValue: { registrar } },
      {
        provide: CrudService,
        useValue: { asegurarCargado: vi.fn(), opciones: () => [], estaCargado: () => true, estaCargando: () => false, estaConError: () => false, getItems: () => [] },
      },
    ],
  });
  const f = TestBed.createComponent(GastoFormComponent);
  f.detectChanges();
  return { f, registrar, form: f.componentInstance as unknown as Formulario };
}

function enviar(el: HTMLElement): void {
  el.querySelector('form')!.dispatchEvent(new Event('submit'));
}

describe('GastoFormComponent', () => {
  it('si la lista de viajes no carga, el selector lo dice en vez de «Aún no hay viajes.»', () => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: GastosStore, useValue: { registrar: vi.fn() } },
        {
          provide: CrudService,
          useValue: {
            asegurarCargado: vi.fn(), opciones: () => [], getItems: () => [],
            estaCargado: (clave: string) => clave !== 'viaje', estaCargando: () => false, estaConError: (clave: string) => clave === 'viaje',
          },
        },
      ],
    });
    const f = TestBed.createComponent(GastoFormComponent);
    f.detectChanges();
    f.nativeElement.querySelector('#gf-viaje').dispatchEvent(new FocusEvent('focus'));
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('No se pudo cargar la lista.');
    expect(f.nativeElement.textContent).not.toContain('Aún no hay viajes.');
  });

  it('si llega desde un viaje (?viaje=15) deja ese viaje elegido', () => {
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: GastosStore, useValue: { registrar: vi.fn() } },
        {
          provide: CrudService,
          useValue: {
            asegurarCargado: vi.fn(),
            opciones: (clave: string) => (clave === 'viaje' ? [{ valor: '15', etiqueta: 'Viaje N° 15 · 01/10/2026' }] : []),
            estaCargado: () => true,
            estaCargando: () => false,
            estaConError: () => false,
            getItems: () => [{}],
          },
        },
      ],
    });
    const f = TestBed.createComponent(GastoFormComponent);
    f.componentRef.setInput('viaje', '15');
    f.detectChanges();
    expect((f.nativeElement.querySelector('#gf-viaje') as HTMLInputElement).value).toBe('Viaje N° 15 · 01/10/2026');
  });

  it('sin datos muestra los cuatro mensajes exactos', () => {
    const { f, form } = montar();
    form.fecha = '';
    enviar(f.nativeElement);
    f.detectChanges();
    const errores = [...f.nativeElement.querySelectorAll('.campo__error')].map((e: Element) => e.textContent?.trim());
    expect(errores).toEqual([
      'Selecciona el viaje al que corresponde este gasto.',
      'Selecciona el cliente responsable de pago.',
      'Ingresa un monto mayor a S/ 0.00.',
      'Selecciona la fecha del gasto.',
    ]);
  });

  it('al registrar abre el gasto nuevo justo en el paso de adjuntar', () => {
    const { f, form, registrar } = montar();
    const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    form.viajeId.set('12');
    form.clienteId.set('5');
    form.monto = '120.5';
    form.fecha = '2026-10-02';
    enviar(f.nativeElement);
    expect(registrar).toHaveBeenCalledWith({ viaje_id: 12, cliente_id: 5, monto: 120.5, fecha: '2026-10-02' });
    expect(navegar).toHaveBeenCalledWith(['/app/gastos', 31], { fragment: 'paso' });
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe(
      'Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.',
    );
  });

  it('si la API responde 422 marca el viaje con el mensaje exacto', () => {
    const { f, form } = montar(vi.fn(() => throwError(() => new HttpErrorResponse({ status: 422 }))));
    form.viajeId.set('99');
    form.clienteId.set('5');
    form.monto = '10';
    form.fecha = '2026-10-02';
    enviar(f.nativeElement);
    f.detectChanges();
    expect(f.nativeElement.querySelector('#gf-viaje-error')?.textContent?.trim()).toBe(
      'Este viaje no existe o no está disponible. Verifica el número de viaje.',
    );
  });
});
