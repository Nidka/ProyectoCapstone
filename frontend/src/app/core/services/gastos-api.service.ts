import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpHeaders, HttpResponse } from '@angular/common/http';
import { Observable } from 'rxjs';
import { AuthService } from './auth.service';

@Injectable({ providedIn: 'root' })
export class GastosApiService {
  private readonly storageKey = 'lizar_gastos_api_base';
  private readonly defaultBase = 'http://localhost:8081';
  private _base = signal(localStorage.getItem(this.storageKey) ?? this.defaultBase);
  readonly base = this._base.asReadonly();

  private auth = inject(AuthService);

  constructor(private http: HttpClient) {}

  setBase(url: string): void {
    const normalized = url.trim().replace(/\/$/, '');
    this._base.set(normalized);
    localStorage.setItem(this.storageKey, normalized);
  }

  // Mientras no exista Gestión de Usuarios (RF-19 a RF-21), la persona con
  // sesión iniciada actúa como Encargado de Logística para aprobar o
  // rechazar comprobantes; es un usuario fijo, sin selector visible.
  private headers(): HttpHeaders {
    const nombre = this.auth.usuario()?.nombre ?? 'Encargado de Logística';
    return new HttpHeaders({ 'X-User-Name': nombre, 'X-User-Role': 'logistica' });
  }

  get<T>(path: string): Observable<T> {
    return this.http.get<T>(`${this._base()}${path}`, { headers: this.headers() });
  }

  /** Descarga un archivo con sus cabeceras (para leer el nombre en Content-Disposition). */
  getBlob(path: string): Observable<HttpResponse<Blob>> {
    return this.http.get(`${this._base()}${path}`, { headers: this.headers(), responseType: 'blob', observe: 'response' });
  }

  post<T>(path: string, body: unknown): Observable<T> {
    return this.http.post<T>(`${this._base()}${path}`, body, { headers: this.headers() });
  }

  put<T>(path: string, body: unknown): Observable<T> {
    return this.http.put<T>(`${this._base()}${path}`, body, { headers: this.headers() });
  }

  delete<T>(path: string): Observable<T> {
    return this.http.delete<T>(`${this._base()}${path}`, { headers: this.headers() });
  }

  uploadFile<T>(path: string, file: File): Observable<T> {
    const form = new FormData();
    form.append('archivo', file);
    return this.http.post<T>(`${this._base()}${path}`, form, { headers: this.headers() });
  }

  fileUrl(archivoUrl: string): string {
    return archivoUrl.startsWith('http') ? archivoUrl : `${this._base()}${archivoUrl}`;
  }
}
