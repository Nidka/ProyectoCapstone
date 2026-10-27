import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree, provideRouter } from '@angular/router';
import { cambioClaveGuard } from './cambio-clave.guard';
import { AuthService } from '../services/auth.service';

function ejecutar(debeCambiar: boolean, url: string) {
  TestBed.configureTestingModule({
    providers: [provideRouter([]), { provide: AuthService, useValue: { debeCambiarClave: () => debeCambiar } }],
  });
  return TestBed.runInInjectionContext(() => cambioClaveGuard({} as ActivatedRouteSnapshot, { url } as RouterStateSnapshot));
}

describe('cambioClaveGuard', () => {
  it('con la contraseña al día deja pasar', () => {
    expect(ejecutar(false, '/app/gastos')).toBe(true);
  });

  it('con la contraseña pendiente bloquea la pantalla y lleva a cambiarla, recordando a dónde iba', () => {
    const destino = ejecutar(true, '/app/gastos?estado=por-aprobar') as UrlTree;
    expect(TestBed.inject(Router).serializeUrl(destino)).toBe('/cambiar-clave?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
  });

  it('bloquea también la raíz de la app', () => {
    const destino = ejecutar(true, '/app/inicio') as UrlTree;
    expect(TestBed.inject(Router).serializeUrl(destino)).toBe('/cambiar-clave?volver=%2Fapp%2Finicio');
  });
});
