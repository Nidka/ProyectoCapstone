import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, of } from 'rxjs';
import { origenApi } from './api-origen';

export interface HealthStatus {
  connected: boolean;
  env: string;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly storageKey = 'lizar_api_base';
  private _base = signal(this.resolveInitialBase());

  readonly base = this._base.asReadonly();
  readonly health = signal<HealthStatus>({ connected: false, env: '…' });

  constructor(private http: HttpClient) {}

  // La base sale siempre de origenApi: una guardada de versiones anteriores (la API de Python
  // del 8000 o su IP pública) dejaría al frontend hablando con un servicio que ya no existe.
  private resolveInitialBase(): string {
    const base = `${origenApi()}/api/v1`;
    localStorage.setItem(this.storageKey, base);
    return base;
  }

  setBase(url: string): void {
    const normalized = url.trim().replace(/\/$/, '');
    this._base.set(normalized);
    localStorage.setItem(this.storageKey, normalized);
  }

  get<T>(path: string): Observable<T> {
    return this.http.get<T>(`${this._base()}${path}`);
  }

  post<T>(path: string, body: unknown): Observable<T> {
    return this.http.post<T>(`${this._base()}${path}`, body);
  }

  checkHealth(): void {
    const base = this._base().replace(/\/api\/v1$/, '');
    this.http.get<{ status: string; env: string }>(`${base}/health`).pipe(
      catchError(() => of(null))
    ).subscribe(d => {
      this.health.set(d ? { connected: true, env: d.env ?? 'ok' } : { connected: false, env: 'sin conexión' });
    });
  }
}
