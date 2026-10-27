import { Injectable, signal } from '@angular/core';
import { HttpClient, HttpResponse } from '@angular/common/http';
import { Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class GastosApiService {
  private readonly storageKey = 'lizar_gastos_api_base';
  private readonly defaultBase = 'http://localhost:8081';
  private _base = signal(localStorage.getItem(this.storageKey) ?? this.defaultBase);
  readonly base = this._base.asReadonly();

  constructor(private http: HttpClient) {}

  setBase(url: string): void {
    const normalized = url.trim().replace(/\/$/, '');
    this._base.set(normalized);
    localStorage.setItem(this.storageKey, normalized);
  }

  get<T>(path: string): Observable<T> {
    return this.http.get<T>(`${this._base()}${path}`);
  }

  /** Descarga un archivo con sus cabeceras (para leer el nombre en Content-Disposition). */
  getBlob(path: string): Observable<HttpResponse<Blob>> {
    return this.http.get(`${this._base()}${path}`, { responseType: 'blob', observe: 'response' });
  }

  post<T>(path: string, body: unknown): Observable<T> {
    return this.http.post<T>(`${this._base()}${path}`, body);
  }

  put<T>(path: string, body: unknown): Observable<T> {
    return this.http.put<T>(`${this._base()}${path}`, body);
  }

  delete<T>(path: string): Observable<T> {
    return this.http.delete<T>(`${this._base()}${path}`);
  }

  uploadFile<T>(path: string, file: File): Observable<T> {
    const form = new FormData();
    form.append('archivo', file);
    return this.http.post<T>(`${this._base()}${path}`, form);
  }

  fileUrl(archivoUrl: string): string {
    return archivoUrl.startsWith('http') ? archivoUrl : `${this._base()}${archivoUrl}`;
  }
}
