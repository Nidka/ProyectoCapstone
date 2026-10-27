import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';
import { DETALLE_DEBE_CAMBIAR_CLAVE } from '../models/auth.model';
import { ApiService } from '../services/api.service';
import { AuthService } from '../services/auth.service';
import { GastosApiService } from '../services/gastos-api.service';
import { ToastService } from '../services/toast.service';

export const MSG_SESION_VENCIDA = 'Tu sesión terminó. Ingresa de nuevo.';
export const MSG_SIN_PERMISO = 'Tu usuario no tiene permiso para hacer esto.';

function origen(url: string): string | null {
  try {
    return new URL(url).origin;
  } catch {
    return null;
  }
}

/**
 * Añade el Bearer a lo que va al servicio (gastos y /api/v1) y atiende sus respuestas:
 * 401 cierra la sesión y manda al login; 403 avisa sin cerrarla, salvo «Debes cambiar tu contraseña»,
 * que lleva a la pantalla para cambiarla. El login y el aviso de salida (/auth/logout) no disparan nada de eso.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const toast = inject(ToastService);
  const propios = [inject(GastosApiService).base(), inject(ApiService).base()].map(origen);
  const esDelServicio = propios.includes(origen(req.url));
  if (!esDelServicio) return next(req);

  const token = auth.token();
  if (token) {
    req = req.clone({ setHeaders: { Authorization: `Bearer ${token}` } });
  }
  // El login lo atiende su pantalla; el aviso de salida ocurre con la sesión ya cerrada y no debe molestar
  const esAcceso = req.url.endsWith('/auth/login') || req.url.endsWith('/auth/logout');
  return next(req).pipe(
    catchError((err: unknown) => {
      if (err instanceof HttpErrorResponse && !esAcceso) {
        // Si ya no hay sesión, otra petición simultánea ya la cerró y avisó: no se repite.
        if (err.status === 401 && auth.isAuth()) {
          auth.sesionVencida();
          toast.show(MSG_SESION_VENCIDA, '', 'warn');
        } else if (err.status === 403 && err.error?.detail === DETALLE_DEBE_CAMBIAR_CLAVE) {
          // No es falta de permiso: la clave está pendiente y solo se puede cambiarla
          auth.exigirCambioClave();
        } else if (err.status === 403) {
          toast.show(MSG_SIN_PERMISO, '', 'err');
        }
      }
      return throwError(() => err);
    }),
  );
};
