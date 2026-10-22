import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = (_ruta, estado) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (auth.isAuth()) return true;
  return router.createUrlTree(['/login'], { queryParams: { volver: estado.url } });
};
