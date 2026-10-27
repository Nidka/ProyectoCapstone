import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Observable, of, throwError } from 'rxjs';
import { LandingComponent } from './landing.component';
import { GastosApiService } from '../../core/services/gastos-api.service';
import { AuthService } from '../../core/services/auth.service';
import { Gasto } from '../../core/models/gasto.model';

function gasto(id: number, estado: Gasto['estado']): Gasto {
  return { id, estado, monto: 10, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-01T12:00:00Z' };
}

function montar(get: (ruta: string) => Observable<unknown>, conSesion = false): HTMLElement {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: GastosApiService, useValue: { get } },
      { provide: AuthService, useValue: { isAuth: () => conSesion } },
    ],
  });
  const f = TestBed.createComponent(LandingComponent);
  f.detectChanges();
  return f.nativeElement;
}

describe('LandingComponent', () => {
  it('muestra los conteos reales del servicio de gastos', () => {
    const el = montar(ruta => (ruta === '/gastos' ? of([gasto(1, 'Registrado'), gasto(2, 'Registrado'), gasto(3, 'Pagado')]) : of([{ id: 9 }])));
    expect([...el.querySelectorAll('.via .hito__cifra')].map(n => n.textContent?.trim())).toEqual(['2', '1', '0']);
    expect(el.querySelector('.via__pendientes')?.textContent).toContain('1 comprobante por aprobar');
  });

  it('cada hito de la carretera lleva su estado en el cabezal', () => {
    const el = montar(ruta => (ruta === '/gastos' ? of([gasto(1, 'Registrado'), gasto(2, 'Registrado'), gasto(3, 'Pagado')]) : of([])));
    expect([...el.querySelectorAll('.via .hito__cabezal')].map(n => n.textContent?.trim())).toEqual(['Registrado', 'Pagado', 'Sustentado']);
  });

  it('el titular se parte en dos placas sin perder el espacio entre ellas', () => {
    const el = montar(() => of([]));
    expect(el.querySelector('h1')?.textContent?.replace(/\s+/g, ' ').trim()).toBe('Gastos de viaje con su comprobante, de la ruta a la oficina.');
    expect(el.querySelectorAll('h1 > span')).toHaveLength(2);
  });

  it('la foto de la carretera abre la portada y carga primero', () => {
    const el = montar(() => of([]));
    const foto = el.querySelector<HTMLImageElement>('.portada__foto img');
    expect(foto?.getAttribute('alt')).toContain('Camión de carga');
    expect(foto?.getAttribute('loading')).not.toBe('lazy');
    expect(foto?.getAttribute('fetchpriority')).toBe('high');
  });

  it('la señal preventiva y la placa amarilla solo aparecen cuando algo espera aprobación', () => {
    const sin = montar(() => of([]));
    expect(sin.querySelector('.via__pendientes')?.textContent).toContain('0 comprobantes por aprobar');
    expect(sin.querySelector('.via__pendientes app-senal')).toBeNull();
    expect(sin.querySelector('.via__pendientes--espera')).toBeNull();
    TestBed.resetTestingModule();
    const con = montar(ruta => (ruta === '/gastos' ? of([]) : of([{ id: 9 }])));
    expect(con.querySelector('.via__pendientes--espera app-senal.senal--preventiva')).not.toBeNull();
  });

  it('si el servicio no responde no inventa cifras', () => {
    const el = montar(() => throwError(() => new Error('sin red')));
    expect(el.querySelector('.via__nota')?.textContent?.trim()).toBe('Los indicadores no están disponibles en este momento.');
    expect([...el.querySelectorAll('.via .hito__cifra')].map(n => n.textContent?.trim())).toEqual(['—', '—', '—']);
    expect(el.querySelector('.via__pendientes')).toBeNull();
  });

  it('sin sesión ofrece Ingresar, con sesión Ir a Inicio, y no usa raya larga', () => {
    const sin = montar(() => of([]));
    expect(sin.querySelector('.portada__acciones a')?.getAttribute('href')).toBe('/login');
    expect(sin.querySelectorAll('.portada__acciones a')).toHaveLength(1);
    expect(sin.querySelector('a[href="/register"]')).toBeNull();
    expect(sin.textContent).not.toMatch(/[—–]/);
    TestBed.resetTestingModule();
    const con = montar(() => of([]), true);
    expect(con.querySelector('.portada__acciones a')?.getAttribute('href')).toBe('/app/inicio');
  });
});
