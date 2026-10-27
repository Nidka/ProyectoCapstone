import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, timeout } from 'rxjs';
import { UsuarioOut } from '../models/auth.model';
import { CambiosUsuario, NuevoUsuario } from '../models/usuario';
import { HTTP_TIMEOUT_MS } from '../utils/error-messages';
import { ApiService } from './api.service';

/** Gestión de usuarios (solo administrador). Nunca se borra un usuario: se desactiva. */
@Injectable({ providedIn: 'root' })
export class UsuariosService {
  private readonly http = inject(HttpClient);
  private readonly api = inject(ApiService);

  private url(ruta = ''): string {
    return `${this.api.base()}/usuarios${ruta}`;
  }

  /** Activos e inactivos, por número de usuario */
  listar(): Observable<UsuarioOut[]> {
    return this.http.get<UsuarioOut[]>(this.url()).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  crear(cuerpo: NuevoUsuario): Observable<UsuarioOut> {
    return this.http.post<UsuarioOut>(this.url(), cuerpo).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  editar(id: number, cambios: CambiosUsuario): Observable<UsuarioOut> {
    return this.http.put<UsuarioOut>(this.url(`/${id}`), cambios).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  desactivar(id: number): Observable<UsuarioOut> {
    return this.http.post<UsuarioOut>(this.url(`/${id}/desactivar`), null).pipe(timeout(HTTP_TIMEOUT_MS));
  }

  activar(id: number): Observable<UsuarioOut> {
    return this.http.post<UsuarioOut>(this.url(`/${id}/activar`), null).pipe(timeout(HTTP_TIMEOUT_MS));
  }
}
