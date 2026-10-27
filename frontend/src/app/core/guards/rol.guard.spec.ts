import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree, provideRouter } from '@angular/router';
import { rolGuard } from './rol.guard';
import { AuthService } from '../services/auth.service';
import { ToastService } from '../services/toast.service';

function ejecutar(sesion: { rol: string } | null, roles: string[] | undefined, url = '/app/gastos/nuevo') {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: AuthService, useValue: { isAuth: () => !!sesion, usuario: () => sesion } },
    ],
  });
  const resultado = TestBed.runInInjectionContext(() =>
    rolGuard({ data: roles ? { roles } : {} } as unknown as ActivatedRouteSnapshot, { url } as RouterStateSnapshot),
  );
  return { resultado, router: TestBed.inject(Router), toast: TestBed.inject(ToastService) };
}

describe('rolGuard', () => {
  it('deja pasar al rol que la ruta permite', () => {
    expect(ejecutar({ rol: 'logistica' }, ['logistica']).resultado).toBe(true);
  });

  it('una ruta sin roles pide solo estar dentro', () => {
    expect(ejecutar({ rol: 'admin' }, undefined).resultado).toBe(true);
  });

  it('sin sesión manda al login recordando a dónde iba', () => {
    const { resultado, router } = ejecutar(null, ['logistica'], '/app/gastos/nuevo?viaje=3');
    expect(router.serializeUrl(resultado as UrlTree)).toBe('/login?volver=%2Fapp%2Fgastos%2Fnuevo%3Fviaje%3D3');
  });

  it('otro rol vuelve a Inicio con un aviso, sin cerrar la sesión', () => {
    const { resultado, router, toast } = ejecutar({ rol: 'admin' }, ['logistica']);
    expect(router.serializeUrl(resultado as UrlTree)).toBe('/app/inicio');
    expect(toast.toasts().map(t => t.title)).toEqual(['Tu usuario no tiene permiso para entrar a esa pantalla.']);
  });

  it('un rol que ya no existe se trata como sin permiso', () => {
    const { resultado, router } = ejecutar({ rol: 'operador' }, ['admin', 'logistica']);
    expect(router.serializeUrl(resultado as UrlTree)).toBe('/app/inicio');
  });
});
