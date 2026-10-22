import type { ChartConfiguration, TooltipItem } from 'chart.js';
import { EstadoGasto } from '../../core/models/gasto.model';
import { EstadoReporte, FilaRanking, MesReporte } from '../../core/models/reporte.model';
import { formatMonto } from '../../core/utils/format';
import { nombreMes } from './filtro-reporte';

export type ConfigGrafico = ChartConfiguration<'bar'>;

/**
 * Colores de estado de la app (--registrado, --pagado, --sustentado en styles/tokens.css).
 * El gris de Registrado tiene poco croma a propósito: por eso la leyenda, los huecos
 * entre segmentos, los tooltips y la tabla «Ver datos» nombran siempre el estado.
 */
export const COLOR_ESTADO: Record<EstadoGasto, string> = {
  Registrado: '#8a9096',
  Pagado: '#2563eb',
  Sustentado: '#15803d',
};

export const ESTADOS: readonly EstadoGasto[] = ['Registrado', 'Pagado', 'Sustentado'];

const COLOR_RANKING = '#4a5056'; // --tinta-2: una sola serie, un solo tono
const COLOR_LINEA = '#e2e3e0'; // --linea: rejilla discreta
const COLOR_SUPERFICIE = '#fdfdfc'; // --panel: huecos de 2 px entre segmentos
const LARGO_ETIQUETA = 20;
const MESES_CORTOS = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];

/** «oct 26» */
export function mesCorto(mes: string): string {
  const [anio, m] = mes.split('-');
  return `${MESES_CORTOS[Number(m) - 1]} ${anio.slice(2)}`;
}

/** Monto del eje sin centavos: «S/ 2,000». */
export function montoEje(valor: number | string): string {
  return formatMonto(valor).replace(/\.00$/, '');
}

/** Monto corto para ejes angostos: «S/ 40 mil», «S/ 2.5 mil», «S/ 800». */
export function montoCompacto(valor: number | string): string {
  const n = Number(valor);
  if (Math.abs(n) < 1000) return `S/ ${Math.round(n)}`;
  return `S/ ${(n / 1000).toFixed(1).replace(/\.0$/, '')} mil`;
}

export function porcentaje(valor: number): string {
  return `${valor.toFixed(1)} %`;
}

export function gastos(cantidad: number): string {
  return `${cantidad} ${cantidad === 1 ? 'gasto' : 'gastos'}`;
}

/** Recorta nombres largos para el eje; el nombre completo va en el tooltip y en la tabla. */
export function recortar(texto: string, largo = LARGO_ETIQUETA): string {
  return texto.length > largo ? `${texto.slice(0, largo - 1)}…` : texto;
}

function montoDe(mes: MesReporte, estado: EstadoGasto): number {
  return estado === 'Registrado' ? mes.registrado : estado === 'Pagado' ? mes.pagado : mes.sustentado;
}

/** Gasto por mes, apilado por estado. Un clic en cualquier punto de la columna elige ese mes. */
export function configPorMes(meses: MesReporte[], alElegirMes: (mes: string) => void): ConfigGrafico {
  return {
    type: 'bar',
    data: {
      labels: meses.map(m => mesCorto(m.mes)),
      datasets: ESTADOS.map(estado => ({
        label: estado,
        data: meses.map(m => montoDe(m, estado)),
        backgroundColor: COLOR_ESTADO[estado],
        borderColor: COLOR_SUPERFICIE,
        borderWidth: { top: 2 },
        borderRadius: 4,
        maxBarThickness: 40,
      })),
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      scales: {
        x: { stacked: true, grid: { display: false }, ticks: { maxRotation: 0, autoSkip: true } },
        y: { stacked: true, beginAtZero: true, grid: { color: COLOR_LINEA }, border: { display: false }, ticks: { callback: v => montoEje(v) } },
      },
      plugins: {
        legend: { position: 'bottom', labels: { boxWidth: 12, boxHeight: 12, useBorderRadius: true, borderRadius: 2 } },
        tooltip: {
          callbacks: {
            title: items => nombreMes(meses[items[0]?.dataIndex ?? 0]?.mes ?? ''),
            label: (c: TooltipItem<'bar'>) => `${c.dataset.label}: ${formatMonto(c.parsed.y)}`,
            footer: items => `Total: ${formatMonto(meses[items[0]?.dataIndex ?? 0]?.total ?? 0)}`,
          },
        },
      },
      onClick: (_evento, elementos) => {
        const i = elementos[0]?.index;
        if (i !== undefined) alElegirMes(meses[i].mes);
      },
      onHover: (evento, elementos) => {
        const lienzo = evento.native?.target as HTMLElement | undefined;
        if (lienzo) lienzo.style.cursor = elementos.length ? 'pointer' : 'default';
      },
    },
  };
}

/** Barras horizontales comunes a la distribución por estado y a los rankings. */
function barrasHorizontales(etiquetas: string[], montos: number[], colores: string | string[], detalle: (i: number) => string, titulo: (i: number) => string): ConfigGrafico {
  return {
    type: 'bar',
    data: {
      labels: etiquetas,
      datasets: [{ data: montos, backgroundColor: colores, borderRadius: 4, maxBarThickness: 22 }],
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'nearest', axis: 'y', intersect: false },
      scales: {
        x: { beginAtZero: true, grid: { color: COLOR_LINEA }, border: { display: false }, ticks: { callback: v => montoCompacto(v), maxTicksLimit: 5, maxRotation: 0 } },
        y: { grid: { display: false } },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: items => titulo(items[0]?.dataIndex ?? 0),
            label: (c: TooltipItem<'bar'>) => `${formatMonto(c.parsed.x)} · ${detalle(c.dataIndex)}`,
          },
        },
      },
    },
  };
}

export function configPorEstado(estados: EstadoReporte[]): ConfigGrafico {
  return barrasHorizontales(
    estados.map(e => e.estado),
    estados.map(e => e.monto),
    estados.map(e => COLOR_ESTADO[e.estado]),
    i => `${gastos(estados[i].cantidad)} · ${porcentaje(estados[i].porcentaje)}`,
    i => estados[i].estado,
  );
}

export function configRanking(filas: FilaRanking[]): ConfigGrafico {
  return barrasHorizontales(
    filas.map(f => recortar(f.etiqueta)),
    filas.map(f => f.monto),
    COLOR_RANKING,
    i => `${gastos(filas[i].cantidad)} · ${porcentaje(filas[i].porcentaje)}`,
    i => `${filas[i].puesto}. ${filas[i].etiqueta}`,
  );
}

/** Resúmenes para aria-label: lo que una persona con lector de pantalla necesita del gráfico. */
export function descripcionPorMes(meses: MesReporte[]): string {
  if (meses.length === 0) return 'Gasto por mes: sin datos.';
  const mayor = meses.reduce((a, b) => (b.total > a.total ? b : a));
  return `Gasto por mes de ${nombreMes(meses[0].mes)} a ${nombreMes(meses[meses.length - 1].mes)}. ` +
    `El mes con más gasto fue ${nombreMes(mayor.mes)}, con ${formatMonto(mayor.total)}.`;
}

export function descripcionPorEstado(estados: EstadoReporte[]): string {
  return `Distribución por estado: ${estados.map(e => `${e.estado} ${porcentaje(e.porcentaje)}`).join(', ')}.`;
}

export function descripcionRanking(titulo: string, filas: FilaRanking[]): string {
  if (filas.length === 0) return `${titulo}: sin datos.`;
  return `${titulo}: primero ${filas[0].etiqueta}, con ${formatMonto(filas[0].monto)} (${porcentaje(filas[0].porcentaje)} del total).`;
}
