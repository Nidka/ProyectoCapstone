import { Injectable, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, of } from 'rxjs';

export interface HealthStatus {
  connected: boolean;
  env: string;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly storageKey = 'lizar_api_base';
  private readonly defaultBase = 'http://13.222.175.52:8000/api/v1';
  private _base = signal(this.resolveInitialBase());

  readonly base = this._base.asReadonly();
  readonly health = signal<HealthStatus>({ connected: false, env: '…' });

  constructor(private http: HttpClient) {}

  private resolveInitialBase(): string {
    const isLocalDev = ['localhost', '127.0.0.1'].includes(window.location.hostname);
    if (isLocalDev) {
      const localBase = 'http://localhost:8000/api/v1';
      localStorage.setItem(this.storageKey, localBase);
      return localBase;
    }
    const saved = localStorage.getItem(this.storageKey);
    if (!saved || saved.includes('127.0.0.1:8000') || saved.includes('localhost:8000')) {
      localStorage.setItem(this.storageKey, this.defaultBase);
      return this.defaultBase;
    }
    return saved;
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
