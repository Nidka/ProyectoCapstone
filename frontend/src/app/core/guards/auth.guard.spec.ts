import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree, provideRouter } from '@angular/router';
import { authGuard } from './auth.guard';
import { AuthService } from '../services/auth.service';

function ejecutar(autenticado: boolean, url: string) {
  TestBed.configureTestingModule({
    providers: [provideRouter([]), { provide: AuthService, useValue: { isAuth: () => autenticado } }],
  });
  return TestBed.runInInjectionContext(() => authGuard({} as ActivatedRouteSnapshot, { url } as RouterStateSnapshot));
}

describe('authGuard', () => {
  it('deja pasar con sesión iniciada', () => {
    expect(ejecutar(true, '/app/gastos')).toBe(true);
  });

  it('sin sesión manda al login recordando a dónde iba', () => {
    const destino = ejecutar(false, '/app/gastos?estado=por-aprobar') as UrlTree;
    expect(TestBed.inject(Router).serializeUrl(destino)).toBe('/login?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
  });
});
