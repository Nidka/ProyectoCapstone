import { inject } from '@angular/core';
import { CanActivateChildFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * Mientras la contraseña esté pendiente de cambio, ninguna pantalla de /app se abre:
 * se lleva a «Cambiar contraseña». Va en `canActivateChild` para valer en cada navegación interna.
 */
export const cambioClaveGuard: CanActivateChildFn = (_ruta, estado) => {
  const auth = inject(AuthService);
  if (!auth.debeCambiarClave()) return true;
  return inject(Router).createUrlTree(['/cambiar-clave'], { queryParams: { volver: estado.url } });
};
