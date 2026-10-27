import { beforeEach, describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ApiService } from './api.service';

const CLAVE = 'lizar_api_base';

function servicio(): { api: ApiService; http: HttpTestingController } {
  TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
  return { api: TestBed.inject(ApiService), http: TestBed.inject(HttpTestingController) };
}

describe('ApiService', () => {
  beforeEach(() => localStorage.clear());

  it('en desarrollo local usa la API única en el puerto 8081', () => {
    const { api } = servicio();
    expect(api.base()).toBe('http://localhost:8081/api/v1');
    expect(localStorage.getItem(CLAVE)).toBe('http://localhost:8081/api/v1');
  });

  it('una base guardada que apunta al antiguo servicio del 8000 se reemplaza', () => {
    localStorage.setItem(CLAVE, 'http://localhost:8000/api/v1');
    const { api } = servicio();
    expect(api.base()).toBe('http://localhost:8081/api/v1');
  });

  it('la salud se consulta en la raíz del servicio, sin /api/v1', () => {
    const { api, http } = servicio();
    api.checkHealth();
    http.expectOne('http://localhost:8081/health').flush({ status: 'ok', env: 'dev' });
    expect(api.health()).toEqual({ connected: true, env: 'dev' });
  });
});
