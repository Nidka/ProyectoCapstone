import { describe, expect, it, vi } from 'vitest';
import type { ActiveElement, ChartEvent, TooltipItem } from 'chart.js';
import { EstadoReporte, FilaRanking, MesReporte } from '../../core/models/reporte.model';
import {
  COLOR_ESTADO, configPorEstado, configPorMes, configRanking, descripcionPorEstado, descripcionPorMes, descripcionRanking,
  mesCorto, montoCompacto, montoEje, recortar,
} from './graficos';

const meses: MesReporte[] = [
  { mes: '2025-11', registrado: 100, pagado: 200, sustentado: 300, total: 600 },
  { mes: '2025-12', registrado: 0, pagado: 0, sustentado: 0, total: 0 },
  { mes: '2026-01', registrado: 50.5, pagado: 0, sustentado: 949.5, total: 1000 },
];
const estados: EstadoReporte[] = [
  { estado: 'Registrado', cantidad: 2, monto: 150.5, porcentaje: 9.4 },
  { estado: 'Pagado', cantidad: 1, monto: 200, porcentaje: 12.5 },
  { estado: 'Sustentado', cantidad: 9, monto: 1249.5, porcentaje: 78.1 },
];
const filas: FilaRanking[] = [
  { puesto: 1, etiqueta: 'Agroexportadora Norte del Perú SAC', cantidad: 7, monto: 900, porcentaje: 56.3 },
  { puesto: 2, etiqueta: 'Minera Sur SA', cantidad: 1, monto: 700, porcentaje: 43.7 },
];

// Los callbacks de Chart.js solo leen dataIndex, parsed y dataset de cada elemento
const item = (dataIndex: number, extra: object = {}) => ({ dataIndex, ...extra }) as unknown as TooltipItem<'bar'>;

describe('configuración de los gráficos', () => {
  it('gasto por mes: una serie por estado, en orden y con su color', () => {
    const c = configPorMes(meses, () => {});
    expect(c.data.labels).toEqual(['nov 25', 'dic 25', 'ene 26']);
    expect(c.data.datasets.map(d => d.label)).toEqual(['Registrado', 'Pagado', 'Sustentado']);
    expect(c.data.datasets.map(d => d.backgroundColor)).toEqual([COLOR_ESTADO.Registrado, COLOR_ESTADO.Pagado, COLOR_ESTADO.Sustentado]);
    expect(c.data.datasets[2].data).toEqual([300, 0, 949.5]);
    expect(c.options?.scales?.['x']?.stacked).toBe(true);
  });

  it('gasto por mes: el tooltip da el monto por estado y el total del mes', () => {
    const tooltip = configPorMes(meses, () => {}).options!.plugins!.tooltip!.callbacks!;
    const items = [item(2, { dataset: { label: 'Sustentado' }, parsed: { y: 949.5 } })];
    expect((tooltip.title as (i: TooltipItem<'bar'>[]) => string)(items)).toBe('enero 2026');
    expect((tooltip.label as (i: TooltipItem<'bar'>) => string)(items[0])).toBe('Sustentado: S/ 949.50');
    expect((tooltip.footer as (i: TooltipItem<'bar'>[]) => string)(items)).toBe('Total: S/ 1,000.00');
  });

  it('gasto por mes: un clic en una columna elige ese mes; en el vacío no hace nada', () => {
    const elegir = vi.fn();
    const onClick = configPorMes(meses, elegir).options!.onClick as (e: ChartEvent, el: ActiveElement[]) => void;
    onClick({} as ChartEvent, [{ index: 1 } as ActiveElement]);
    onClick({} as ChartEvent, []);
    expect(elegir).toHaveBeenCalledTimes(1);
    expect(elegir).toHaveBeenCalledWith('2025-12');
  });

  it('distribución por estado: barras horizontales con el color de cada estado', () => {
    const c = configPorEstado(estados);
    expect(c.options?.indexAxis).toBe('y');
    expect(c.data.datasets[0].backgroundColor).toEqual([COLOR_ESTADO.Registrado, COLOR_ESTADO.Pagado, COLOR_ESTADO.Sustentado]);
    const label = c.options!.plugins!.tooltip!.callbacks!.label as (i: TooltipItem<'bar'>) => string;
    expect(label(item(0, { parsed: { x: 150.5 } }))).toBe('S/ 150.50 · 2 gastos · 9.4 %');
  });

  it('ranking: recorta nombres largos en el eje y los muestra completos en el tooltip', () => {
    const c = configRanking(filas);
    expect(c.data.labels).toEqual(['Agroexportadora Nor…', 'Minera Sur SA']);
    const title = c.options!.plugins!.tooltip!.callbacks!.title as (i: TooltipItem<'bar'>[]) => string;
    const label = c.options!.plugins!.tooltip!.callbacks!.label as (i: TooltipItem<'bar'>) => string;
    expect(title([item(0)])).toBe('1. Agroexportadora Norte del Perú SAC');
    expect(label(item(1, { parsed: { x: 700 } }))).toBe('S/ 700.00 · 1 gasto · 43.7 %');
    expect(c.options?.plugins?.legend?.display).toBe(false);
  });

  it('24 meses siguen teniendo una etiqueta por mes', () => {
    const largos = Array.from({ length: 24 }, (_, i) => ({ mes: `${2025 + Math.floor(i / 12)}-${String((i % 12) + 1).padStart(2, '0')}`, registrado: 1, pagado: 1, sustentado: 1, total: 3 }));
    expect(configPorMes(largos, () => {}).data.labels).toHaveLength(24);
  });

  it('formatos del eje y etiquetas', () => {
    expect(mesCorto('2026-10')).toBe('oct 26');
    expect(montoEje(2000)).toBe('S/ 2,000');
    expect(montoEje(2500.5)).toBe('S/ 2,500.50');
    expect(recortar('corto')).toBe('corto');
    expect(montoCompacto(40000)).toBe('S/ 40 mil');
    expect(montoCompacto(2500)).toBe('S/ 2.5 mil');
    expect(montoCompacto(800)).toBe('S/ 800');
  });

  it('describe cada gráfico para los lectores de pantalla', () => {
    expect(descripcionPorMes(meses)).toBe('Gasto por mes de noviembre 2025 a enero 2026. El mes con más gasto fue enero 2026, con S/ 1,000.00.');
    expect(descripcionPorEstado(estados)).toBe('Distribución por estado: Registrado 9.4 %, Pagado 12.5 %, Sustentado 78.1 %.');
    expect(descripcionRanking('Gasto por cliente', filas)).toBe('Gasto por cliente: primero Agroexportadora Norte del Perú SAC, con S/ 900.00 (56.3 % del total).');
    expect(descripcionRanking('Gasto por unidad', [])).toBe('Gasto por unidad: sin datos.');
  });
});
