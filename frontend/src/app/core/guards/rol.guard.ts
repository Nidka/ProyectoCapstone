import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { Rol } from '../models/auth.model';
import { AuthService } from '../services/auth.service';
import { ToastService } from '../services/toast.service';

/**
 * Deja entrar solo a los roles de `data.roles` de la ruta (`data: { roles: ['logistica'] }`).
 * Sin `roles` basta con tener sesión. Va después de `authGuard` en la ruta padre.
 */
export const rolGuard: CanActivateFn = (ruta, estado) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (!auth.isAuth()) return router.createUrlTree(['/login'], { queryParams: { volver: estado.url } });
  const roles = ruta.data['roles'] as readonly Rol[] | undefined;
  const rol = auth.usuario()?.rol;
  if (!roles || (rol && roles.includes(rol))) return true;
  inject(ToastService).show('Tu usuario no tiene permiso para entrar a esa pantalla.', '', 'warn');
  return router.createUrlTree(['/app/inicio']);
};
