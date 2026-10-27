import { beforeEach, describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { GastosApiService } from './gastos-api.service';

function montar() {
  TestBed.configureTestingModule({ providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()] });
  return { api: TestBed.inject(GastosApiService), http: TestBed.inject(HttpTestingController) };
}

describe('GastosApiService', () => {
  beforeEach(() => localStorage.clear());

  it('no envía X-User-Role ni X-User-Name: la identidad viaja en el Bearer del interceptor', () => {
    const { api, http } = montar();
    api.get('/gastos').subscribe();
    api.post('/gastos', {}).subscribe();
    api.put('/clientes/1/checklist', {}).subscribe();
    api.delete('/clientes/1/checklist').subscribe();
    api.uploadFile('/gastos/1/comprobantes', new File(['x'], 'a.png')).subscribe();
    api.getBlob('/gastos/exportar.xlsx').subscribe();
    const pedidas = http.match(() => true);
    expect(pedidas).toHaveLength(6);
    for (const p of pedidas) {
      expect(p.request.headers.has('X-User-Role')).toBe(false);
      expect(p.request.headers.has('X-User-Name')).toBe(false);
    }
  });

  it('apunta al servicio de gastos y arma el enlace de los comprobantes', () => {
    const { api } = montar();
    expect(api.base()).toBe('http://localhost:8081');
    expect(api.fileUrl('/uploads/a.png')).toBe('http://localhost:8081/uploads/a.png');
    expect(api.fileUrl('https://x.test/a.png')).toBe('https://x.test/a.png');
  });
});
