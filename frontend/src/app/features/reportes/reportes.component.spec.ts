import { afterEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Router, provideRouter } from '@angular/router';
import { Observable, Subject, of, throwError } from 'rxjs';
import type { ActiveElement, ChartEvent } from 'chart.js';
import { ReportesComponent } from './reportes.component';
import { FABRICA_GRAFICO } from './grafico/grafico.component';
import type { ConfigGrafico } from './graficos';
import { rangoDeAtajo } from './filtro-reporte';
import { ReportesService } from '../../core/services/reportes.service';
import { ToastService } from '../../core/services/toast.service';
import { ArchivoPdf, ResumenReporte } from '../../core/models/reporte.model';
import { mensajeError } from '../../core/utils/error-messages';
import { hoyIso } from '../../core/utils/format';

const fila = (puesto: number, etiqueta: string) => ({ puesto, etiqueta, cantidad: 3, monto: 300 - puesto * 10, porcentaje: 30 });
const RESUMEN: ResumenReporte = {
  desde: '2025-11-01',
  hasta: '2026-01-31',
  indicadores: { total: 1600, cantidad: 12, promedio: 133.33, sustentado: 1249.5, pendiente: 350.5, porcentaje_sustentado: 78.1, gastos_sustentados: 9, isc: 75 },
  por_mes: [
    { mes: '2025-11', registrado: 100, pagado: 200, sustentado: 300, total: 600 },
    { mes: '2025-12', registrado: 0, pagado: 0, sustentado: 0, total: 0 },
    { mes: '2026-01', registrado: 50.5, pagado: 0, sustentado: 949.5, total: 1000 },
  ],
  por_estado: [
    { estado: 'Registrado', cantidad: 2, monto: 150.5, porcentaje: 9.4 },
    { estado: 'Pagado', cantidad: 1, monto: 200, porcentaje: 12.5 },
    { estado: 'Sustentado', cantidad: 9, monto: 1249.5, porcentaje: 78.1 },
  ],
  por_cliente: [fila(1, 'Agro Norte SAC'), fila(2, 'Minera Sur SA')],
  por_unidad: [fila(1, 'T1A-123')],
  por_conductor: [],
};
const VACIO: ResumenReporte = {
  ...RESUMEN,
  indicadores: { total: 0, cantidad: 0, promedio: 0, sustentado: 0, pendiente: 0, porcentaje_sustentado: 0, gastos_sustentados: 0, isc: 0 },
  por_cliente: [], por_unidad: [], por_conductor: [],
};

interface Opciones {
  resumen?: () => Observable<ResumenReporte>;
  descargar?: () => Observable<ArchivoPdf>;
  descargarExcel?: () => Observable<ArchivoPdf>;
  desde?: string;
  hasta?: string;
}

async function montar(op: Opciones = {}) {
  const configs: ConfigGrafico[] = [];
  const servicio = {
    resumen: vi.fn(op.resumen ?? (() => of(RESUMEN))),
    descargar: vi.fn(op.descargar ?? (() => of({ blob: new Blob(['%PDF']), nombre: 'reporte.pdf' }))),
    descargarExcel: vi.fn(op.descargarExcel ?? (() => of({ blob: new Blob(['PK']), nombre: 'gastos-2026-10-22.xlsx' }))),
  };
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: ReportesService, useValue: servicio },
      { provide: FABRICA_GRAFICO, useValue: (_l: HTMLCanvasElement, c: ConfigGrafico) => (configs.push(c), { actualizar: vi.fn(), destruir: vi.fn() }) },
    ],
  });
  const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
  const f = TestBed.createComponent(ReportesComponent);
  if (op.desde !== undefined) f.componentRef.setInput('desde', op.desde);
  if (op.hasta !== undefined) f.componentRef.setInput('hasta', op.hasta);
  f.detectChanges();
  await f.whenStable();
  f.detectChanges();
  const el = f.nativeElement as HTMLElement;
  const boton = (texto: string) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes(texto)) as HTMLButtonElement;
  const toasts = () => TestBed.inject(ToastService).toasts();
  return { f, el, servicio, navegar, configs, boton, toasts };
}

// El gráfico por mes es el único con una serie por estado
const clicEnMes = (configs: ConfigGrafico[], indice: number) => {
  const porMes = configs.find(c => c.data.datasets.length === 3)!;
  (porMes.options!.onClick as (e: ChartEvent, el: ActiveElement[]) => void)({} as ChartEvent, [{ index: indice } as ActiveElement]);
};

const rangoNavegado = (navegar: ReturnType<typeof vi.fn>) => (navegar.mock.lastCall?.[1] as { queryParams: object }).queryParams;

describe('ReportesComponent', () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('sin rango en la URL consulta los últimos 12 meses', async () => {
    const { servicio, boton } = await montar();
    expect(servicio.resumen).toHaveBeenCalledWith(rangoDeAtajo('ultimos-12', hoyIso()));
    expect(boton('Últimos 12 meses').getAttribute('aria-pressed')).toBe('true');
  });

  it('muestra los indicadores, los gráficos y la tabla de datos del rango de la URL', async () => {
    const { el, servicio, configs } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    expect(servicio.resumen).toHaveBeenCalledWith({ desde: '2025-11-01', hasta: '2026-01-31' });
    const valores = [...el.querySelectorAll('.indicador__valor')].map(v => v.textContent?.trim());
    expect(valores).toEqual(['S/ 1,600.00', '12', 'S/ 133.33', '75.0 %', '78.1 %']);
    // El ISC va destacado, con el conteo en que se basa
    const isc = el.querySelector('.indicador--destacado')!;
    expect(isc.querySelector('.indicador__etiqueta')?.textContent?.trim()).toBe('ISC · gastos sustentados');
    expect(isc.textContent).toContain('9 de 12 gastos');
    expect([...el.querySelectorAll('.indicador__nota')].map(n => n.textContent?.trim())).toContain('Pendiente: S/ 350.50');
    // por mes, por estado, cliente y unidad; el conductor no tiene datos
    expect(configs).toHaveLength(4);
    expect(el.textContent).toContain('Sin datos en este período.');
    expect(el.querySelector('canvas')?.getAttribute('aria-label')).toContain('El mes con más gasto fue enero 2026');
    expect([...el.querySelectorAll('.reportes__datos th[scope="row"]')].map(t => t.textContent?.trim())).toContain('diciembre 2025');
  });

  it('un atajo cambia el rango de la URL', async () => {
    const { boton, navegar } = await montar();
    boton('Este año').click();
    expect(rangoNavegado(navegar)).toEqual(rangoDeAtajo('este-anio', hoyIso()));
  });

  it('un rango inválido se explica junto a las fechas, no se consulta y no se descarga', async () => {
    const { el, servicio, boton } = await montar({ desde: '2026-05-02', hasta: '2026-05-01' });
    expect(el.querySelector('#reporte-error')?.textContent?.trim()).toBe('La fecha inicial no puede ser posterior a la final.');
    expect(el.querySelector('#reporte-desde')?.getAttribute('aria-invalid')).toBe('true');
    expect(servicio.resumen).not.toHaveBeenCalled();
    expect(boton('Descargar reporte estadístico').disabled).toBe(true);
  });

  it('un clic en un mes filtra a ese mes, avisa y «Quitar» vuelve al rango anterior', async () => {
    const { f, el, navegar, configs, boton } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    clicEnMes(configs, 1);
    expect(rangoNavegado(navegar)).toEqual({ desde: '2025-12-01', hasta: '2025-12-31' });
    f.componentRef.setInput('desde', '2025-12-01');
    f.componentRef.setInput('hasta', '2025-12-31');
    f.detectChanges();
    expect(el.querySelector('.reportes__aviso')?.textContent).toContain('Viendo: diciembre 2025');
    boton('Quitar').click();
    expect(rangoNavegado(navegar)).toEqual({ desde: '2025-11-01', hasta: '2026-01-31' });
  });

  it('«Quitar» vuelve al rango desde el que se eligió el mes, aunque antes se haya usado «Atrás»', async () => {
    const { f, navegar, configs, boton } = await montar({ desde: '2026-01-01', hasta: '2026-10-17' });
    const irA = async (desde: string, hasta: string) => {
      f.componentRef.setInput('desde', desde);
      f.componentRef.setInput('hasta', hasta);
      f.detectChanges();
      await f.whenStable();
      f.detectChanges();
    };
    clicEnMes(configs, 1);
    await irA('2025-12-01', '2025-12-31');
    // «Atrás» dos veces: la URL vuelve a otro rango sin pasar por los botones de la página
    await irA('2025-11-01', '2026-01-31');
    clicEnMes(configs, 0);
    await irA('2025-11-01', '2025-11-30');
    boton('Quitar').click();
    expect(rangoNavegado(navegar)).toEqual({ desde: '2025-11-01', hasta: '2026-01-31' });
  });

  it('un mes también se elige con el teclado desde la tabla, y el aviso se anuncia', async () => {
    const { f, el, navegar } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    const verMes = [...el.querySelectorAll<HTMLButtonElement>('.reportes__datos button')].filter(b => b.textContent?.includes('Ver mes'));
    expect(verMes).toHaveLength(3);
    verMes[2].click();
    expect(rangoNavegado(navegar)).toEqual({ desde: '2026-01-01', hasta: '2026-01-31' });
    f.componentRef.setInput('desde', '2026-01-01');
    f.componentRef.setInput('hasta', '2026-01-31');
    f.detectChanges();
    expect(el.querySelector('[role="status"] .reportes__aviso')?.textContent).toContain('Viendo: enero 2026');
  });

  it('cambiar una fecha a mano después de elegir un mes quita el aviso', async () => {
    const { f, el, configs } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    clicEnMes(configs, 1);
    f.componentRef.setInput('desde', '2025-12-01');
    f.componentRef.setInput('hasta', '2025-12-31');
    f.detectChanges();
    vi.useFakeTimers();
    const desde = el.querySelector<HTMLInputElement>('#reporte-desde')!;
    desde.value = '2025-12-05';
    desde.dispatchEvent(new Event('change'));
    vi.advanceTimersByTime(500);
    f.detectChanges();
    expect(el.querySelector('.reportes__aviso')).toBeNull();
  });

  it('cambiar las dos fechas seguidas usa ambas, aunque la URL no se haya actualizado', async () => {
    const { el, navegar } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    vi.useFakeTimers();
    const desde = el.querySelector<HTMLInputElement>('#reporte-desde')!;
    const hasta = el.querySelector<HTMLInputElement>('#reporte-hasta')!;
    desde.value = '2026-05-02';
    desde.dispatchEvent(new Event('change'));
    hasta.value = '2026-05-01';
    hasta.dispatchEvent(new Event('change'));
    vi.advanceTimersByTime(500);
    expect(rangoNavegado(navegar)).toEqual({ desde: '2026-05-02', hasta: '2026-05-01' });
  });

  it('escribir una fecha con el teclado aplica solo el valor final, sin una entrada de historial por tecla', async () => {
    const { el, navegar } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    vi.useFakeTimers();
    const desde = el.querySelector<HTMLInputElement>('#reporte-desde')!;
    // Chrome emite «change» en cada tecla de un segmento; Retroceso deja el campo vacío
    for (const valor of ['0002-01-15', '0020-01-15', '', '2025-01-15']) {
      desde.value = valor;
      desde.dispatchEvent(new Event('change'));
    }
    expect(navegar).not.toHaveBeenCalled();
    vi.advanceTimersByTime(500);
    expect(navegar).toHaveBeenCalledTimes(1);
    expect(navegar.mock.lastCall?.[1]).toMatchObject({ queryParams: { desde: '2025-01-15', hasta: '2026-01-31' }, replaceUrl: true });
  });

  it('un campo de fecha vaciado no cambia el rango ni el valor del campo', async () => {
    const { f, el, navegar } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    vi.useFakeTimers();
    const desde = el.querySelector<HTMLInputElement>('#reporte-desde')!;
    desde.value = '';
    desde.dispatchEvent(new Event('change'));
    vi.advanceTimersByTime(1000);
    f.detectChanges();
    expect(navegar).not.toHaveBeenCalled();
    expect(desde.value).toBe('');
  });

  it('sin gastos en el rango muestra el estado vacío y deja descargar', async () => {
    const { el, boton } = await montar({ resumen: () => of(VACIO) });
    expect(el.textContent).toContain('No hay gastos en este período.');
    expect(el.querySelector('.indicadores')).toBeNull();
    expect(boton('Descargar detalle de gastos').disabled).toBe(false);
  });

  it('si falla la consulta ofrece reintentar', async () => {
    let fallar = true;
    const { f, el, servicio, boton } = await montar({ resumen: () => (fallar ? throwError(() => new HttpErrorResponse({ status: 0 })) : of(RESUMEN)) });
    expect(el.textContent).toContain('No se pudo cargar el reporte.');
    fallar = false;
    boton('Reintentar').click();
    await f.whenStable();
    f.detectChanges();
    expect(servicio.resumen).toHaveBeenCalledTimes(2);
    expect(el.querySelector('.indicadores')).not.toBeNull();
  });

  it('descargar muestra «Generando…», evita el doble clic y guarda el archivo', async () => {
    const pendiente = new Subject<ArchivoPdf>();
    const { f, servicio, boton } = await montar({ descargar: () => pendiente });
    Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() });
    const clic = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    // Dos clics antes de que la vista deshabilite el botón: la guarda evita la segunda petición
    const descargar = boton('Descargar reporte estadístico');
    descargar.click();
    descargar.click();
    expect(servicio.descargar).toHaveBeenCalledTimes(1);
    f.detectChanges();
    expect(boton('Generando…').disabled).toBe(true);
    expect(boton('Descargar detalle de gastos').disabled).toBe(true);
    pendiente.next({ blob: new Blob(['%PDF']), nombre: 'reporte-estadistico.pdf' });
    pendiente.complete();
    f.detectChanges();
    expect(clic).toHaveBeenCalledTimes(1);
    expect(boton('Descargar reporte estadístico').disabled).toBe(false);
  });

  it('si la descarga falla, avisa con el mensaje de error de la app', async () => {
    const err = new HttpErrorResponse({ status: 500 });
    const { f, toasts, boton } = await montar({ descargar: () => throwError(() => err) });
    boton('Descargar detalle de gastos').click();
    f.detectChanges();
    expect(toasts().at(-1)?.title).toBe(mensajeError(err));
    expect(toasts().at(-1)?.type).toBe('err');
    expect(boton('Descargar detalle de gastos').disabled).toBe(false);
  });

  it('el detalle en Excel se descarga con el rango elegido', async () => {
    const { f, servicio, boton } = await montar({ desde: '2025-11-01', hasta: '2026-01-31' });
    Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() });
    const clic = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    boton('Descargar detalle (Excel)').click();
    f.detectChanges();
    expect(servicio.descargarExcel).toHaveBeenCalledWith({ desde: '2025-11-01', hasta: '2026-01-31' });
    expect(clic).toHaveBeenCalledTimes(1);
  });

  it('con el rango inválido no se descarga el Excel', async () => {
    const { boton } = await montar({ desde: '2026-05-02', hasta: '2026-05-01' });
    expect(boton('Descargar detalle (Excel)').disabled).toBe(true);
  });
});
