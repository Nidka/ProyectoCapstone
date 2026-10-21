import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpHeaders, HttpResponse } from '@angular/common/http';
import { of } from 'rxjs';
import { ReportesService, nombreDeArchivo } from './reportes.service';
import { GastosApiService } from './gastos-api.service';

function montar(cabeceras: Record<string, string> = {}) {
  const pdf = new Blob(['%PDF-1.7'], { type: 'application/pdf' });
  const api = {
    get: vi.fn(() => of({ desde: '2026-01-01' })),
    getBlob: vi.fn(() => of(new HttpResponse({ body: pdf, headers: new HttpHeaders(cabeceras) }))),
  };
  TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
  return { servicio: TestBed.inject(ReportesService), api, pdf };
}

const filtro = { desde: '2025-11-01', hasta: '2026-10-17' };

describe('ReportesService', () => {
  it('exporta a Excel conservando simultáneamente viaje, cliente, estado y fechas', () => {
    const { servicio, api } = montar();
    servicio.descargarExcel({
      viaje_id: 14, cliente_id: 5, estado: 'Sustentado', desde: '2026-10-01', hasta: '2026-10-21',
    }).subscribe();
    expect(api.getBlob).toHaveBeenCalledWith(
      '/gastos/exportar.xlsx?viaje_id=14&cliente_id=5&estado=Sustentado&desde=2026-10-01&hasta=2026-10-21',
    );
    expect(api.getBlob).toHaveBeenCalledTimes(1);
  });

  it('pide el resumen del rango', () => {
    const { servicio, api } = montar();
    servicio.resumen(filtro).subscribe();
    expect(api.get).toHaveBeenCalledWith('/reportes/resumen?desde=2025-11-01&hasta=2026-10-17');
  });

  it('descarga el PDF con el nombre que manda el servidor', () => {
    const { servicio, api, pdf } = montar({ 'Content-Disposition': 'attachment; filename="reporte-estadistico-2025-11-01_2026-10-17.pdf"' });
    let archivo: { blob: Blob; nombre: string } | undefined;
    servicio.descargar('estadistico', filtro).subscribe(a => (archivo = a));
    expect(api.getBlob).toHaveBeenCalledWith('/reportes/estadistico.pdf?desde=2025-11-01&hasta=2026-10-17');
    expect(archivo).toEqual({ blob: pdf, nombre: 'reporte-estadistico-2025-11-01_2026-10-17.pdf' });
  });

  it('sin Content-Disposition, arma el nombre con el rango', () => {
    const { servicio } = montar();
    let nombre = '';
    servicio.descargar('detalle', filtro).subscribe(a => (nombre = a.nombre));
    expect(nombre).toBe('detalle-gastos-2025-11-01_2026-10-17.pdf');
  });

  it('nombreDeArchivo lee la cabecera', () => {
    expect(nombreDeArchivo('attachment; filename="a.pdf"', 'b.pdf')).toBe('a.pdf');
    expect(nombreDeArchivo(null, 'b.pdf')).toBe('b.pdf');
  });

  it('descarga el Excel de gastos con los filtros que vienen, y omite los vacíos', () => {
    const { servicio, api } = montar({ 'Content-Disposition': 'attachment; filename="gastos-2026-10-22.xlsx"' });
    let nombre = '';
    servicio.descargarExcel({ viaje_id: 14, cliente_id: null, estado: 'Pagado', desde: '', hasta: undefined }).subscribe(a => (nombre = a.nombre));
    expect(api.getBlob).toHaveBeenCalledWith('/gastos/exportar.xlsx?viaje_id=14&estado=Pagado');
    expect(nombre).toBe('gastos-2026-10-22.xlsx');
  });

  it('sin filtros pide el Excel sin parámetros y el nombre de respaldo lleva la fecha', () => {
    const { servicio, api } = montar();
    let nombre = '';
    servicio.descargarExcel({}).subscribe(a => (nombre = a.nombre));
    expect(api.getBlob).toHaveBeenCalledWith('/gastos/exportar.xlsx');
    expect(nombre).toMatch(/^gastos-\d{4}-\d{2}-\d{2}\.xlsx$/);
  });
});
