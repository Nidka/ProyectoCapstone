import { describe, expect, it, vi } from 'vitest';
import { computed, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { InicioComponent } from './inicio.component';
import { GastosStore } from '../../core/state/gastos.store';
import { CrudService } from '../../core/services/crud.service';
import { AuthService } from '../../core/services/auth.service';
import { Comprobante, Gasto } from '../../core/models/gasto.model';
import { contarPorEstado } from '../../core/state/gastos.logic';

function gasto(id: number, estado: Gasto['estado'], monto: number): Gasto {
  return { id, estado, monto, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: `2026-10-0${id}T12:00:00Z` };
}

function montar(gastos: Gasto[], pendientes: Comprobante[], estado = { cargado: true, error: false }) {
  const lista = signal(gastos);
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      {
        provide: GastosStore,
        useValue: {
          gastos: lista, comprobantes: signal(pendientes), cargando: signal(false), cargado: signal(estado.cargado), error: signal(estado.error),
          conteo: computed(() => contarPorEstado(lista())), pendientes: signal(pendientes), asegurarCargado: vi.fn(), cargar: vi.fn(),
        },
      },
      { provide: CrudService, useValue: { asegurarCargado: vi.fn(), etiqueta: () => 'Agro Norte SAC' } },
      { provide: AuthService, useValue: { usuario: signal({ nombre: 'Ana Quispe' }) } },
    ],
  });
  const f = TestBed.createComponent(InicioComponent);
  f.detectChanges();
  return f.nativeElement as HTMLElement;
}

describe('InicioComponent', () => {
  it('sin gastos enseña por dónde empezar', () => {
    const el = montar([], []);
    expect(el.textContent).toContain('Todavía no hay gastos registrados.');
    expect(el.querySelector('a[href="/app/operacion/viajes"]')).not.toBeNull();
  });

  it('si el servicio de gastos no responde, no muestra ceros: avisa y ofrece Reintentar', () => {
    const el = montar([], [], { cargado: false, error: true });
    expect(el.textContent).toContain('No se pudieron cargar los gastos.');
    expect(el.textContent).not.toContain('comprobantes por aprobar');
    expect([...el.querySelectorAll('button')].some(b => b.textContent?.trim() === 'Reintentar')).toBe(true);
  });

  it('muestra lo pendiente, el recorrido con montos y los últimos gastos', () => {
    const pendiente: Comprobante = {
      id: 9, gasto_id: 3, archivo_url: '/u/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-03T12:00:00Z',
    };
    const el = montar([gasto(1, 'Registrado', 120), gasto(2, 'Registrado', 30.5), gasto(3, 'Pagado', 85.5)], [pendiente]);
    const porAprobar = el.querySelector('.atender__item--via');
    expect(porAprobar?.textContent).toContain('1');
    expect(porAprobar?.textContent).toContain('comprobante por aprobar');
    expect(porAprobar?.querySelector('app-senal.senal--preventiva')).not.toBeNull();
    const hitos = [...el.querySelectorAll('.recorrido__hito')].map(h => [
      h.querySelector('.hito__cabezal')?.textContent?.trim(),
      h.querySelector('.hito__cifra')?.textContent?.trim(),
      h.querySelector('.recorrido__monto')?.textContent,
    ]);
    expect(hitos).toEqual([
      ['Registrado', '2', 'S/ 150.50'],
      ['Pagado', '1', 'S/ 85.50'],
      ['Sustentado', '0', 'S/ 0.00'],
    ]);
    expect(el.querySelectorAll('li.fila-gasto')).toHaveLength(3);
    expect(el.textContent).toContain('Hola, Ana Quispe.');
  });
});
