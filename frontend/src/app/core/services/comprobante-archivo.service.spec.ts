import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ComprobanteArchivoService } from './comprobante-archivo.service';
import { GastosApiService } from './gastos-api.service';
import { ToastService } from './toast.service';

function montar() {
  const toast = { show: vi.fn() };
  TestBed.configureTestingModule({
    providers: [
      provideHttpClient(), provideHttpClientTesting(),
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => `http://api${u}` } },
      { provide: ToastService, useValue: toast },
    ],
  });
  return { servicio: TestBed.inject(ComprobanteArchivoService), http: TestBed.inject(HttpTestingController), toast };
}

describe('ComprobanteArchivoService', () => {
  it('descarga el comprobante como Blob desde la URL de la API (el interceptor agrega el token)', () => {
    const { servicio, http } = montar();
    let recibido: Blob | undefined;
    servicio.descargar('/uploads/9.jpg').subscribe(b => (recibido = b));
    const req = http.expectOne('http://api/uploads/9.jpg');
    expect(req.request.method).toBe('GET');
    expect(req.request.responseType).toBe('blob');
    req.flush(new Blob(['x']));
    expect(recibido).toBeInstanceOf(Blob);
  });

  it('abrir muestra el comprobante en la pestaña abierta al instante', () => {
    const { servicio, http } = montar();
    const ventana = { location: { href: '' }, close: vi.fn() };
    vi.spyOn(window, 'open').mockReturnValue(ventana as unknown as Window);
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:abc');
    servicio.abrir('/uploads/9.jpg');
    http.expectOne('http://api/uploads/9.jpg').flush(new Blob(['x']));
    expect(ventana.location.href).toBe('blob:abc');
  });

  it('si la descarga falla cierra la pestaña y avisa', () => {
    const { servicio, http, toast } = montar();
    const ventana = { location: { href: '' }, close: vi.fn() };
    vi.spyOn(window, 'open').mockReturnValue(ventana as unknown as Window);
    servicio.abrir('/uploads/9.jpg');
    http.expectOne('http://api/uploads/9.jpg').flush(new Blob(['no']), { status: 401, statusText: 'Unauthorized' });
    expect(ventana.close).toHaveBeenCalled();
    expect(toast.show).toHaveBeenCalledWith('No se pudo abrir el comprobante', expect.any(String), 'err');
  });
});
