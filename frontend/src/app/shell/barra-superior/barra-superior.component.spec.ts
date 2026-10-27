import { describe, expect, it, vi } from 'vitest';
import { computed, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { BarraSuperiorComponent } from './barra-superior.component';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { GastosStore } from '../../core/state/gastos.store';
import { NotificacionesStore } from '../../core/state/notificaciones.store';

const avisos = {
  lista: signal([]), noLeidas: signal(0), cargando: signal(false), cargado: signal(true), error: signal(false),
  iniciarSondeo: vi.fn(), detenerSondeo: vi.fn(), actualizarSiViejo: vi.fn(), cargar: vi.fn(), marcarLeida: vi.fn(),
};

function montar(rol: 'admin' | 'logistica') {
  const usuario = signal({ nombre: 'Ana Quispe', rol });
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: ApiService, useValue: { health: signal({ connected: true, env: 'dev' }), checkHealth: vi.fn() } },
      { provide: AuthService, useValue: { usuario, esLogistica: computed(() => usuario().rol === 'logistica'), logout: vi.fn() } },
      { provide: GastosStore, useValue: { pendientes: signal([]) } },
      { provide: NotificacionesStore, useValue: avisos },
    ],
  });
  const f = TestBed.createComponent(BarraSuperiorComponent);
  f.detectChanges();
  return f.nativeElement as HTMLElement;
}

describe('BarraSuperiorComponent', () => {
  const CONSULTA = ['/app/inicio', '/app/gastos', '/app/reportes', '/app/operacion', '/app/catalogos'];
  const enlacesDe = (rol: 'admin' | 'logistica') => [...montar(rol).querySelectorAll('.barra__enlace')].map(a => a.getAttribute('href'));

  it('Logística ve las cinco secciones de consulta, sin «Usuarios»', () => {
    expect(enlacesDe('logistica')).toEqual(CONSULTA);
  });

  it('el administrador ve además «Usuarios»', () => {
    expect(enlacesDe('admin')).toEqual([...CONSULTA, '/app/usuarios']);
  });

  it('Logística ve la campana de avisos y empieza a preguntar por ellos', () => {
    avisos.iniciarSondeo.mockClear();
    const el = montar('logistica');
    expect(el.querySelector('app-campana .avisos__boton')).not.toBeNull();
    expect(avisos.iniciarSondeo).toHaveBeenCalled();
  });

  it('el administrador no ve la campana (los avisos son de Logística) ni consulta el servicio', () => {
    avisos.iniciarSondeo.mockClear();
    const el = montar('admin');
    expect(el.querySelector('app-campana')).toBeNull();
    expect(avisos.iniciarSondeo).not.toHaveBeenCalled();
  });
});
