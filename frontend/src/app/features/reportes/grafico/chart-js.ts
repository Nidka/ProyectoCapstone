import { BarController, BarElement, CategoryScale, Chart, Legend, LinearScale, Tooltip } from 'chart.js';
import type { ConfigGrafico } from '../graficos';

// Solo lo que usan los gráficos de barras: Chart.js queda en el chunk de Reportes.
Chart.register(BarController, BarElement, CategoryScale, LinearScale, Tooltip, Legend);
Chart.defaults.font.family = "'Overpass Variable', 'Overpass', system-ui, sans-serif";
Chart.defaults.color = '#4a5056'; // --tinta-2

/** Lo que el componente necesita de un gráfico; en las pruebas se reemplaza (jsdom no tiene canvas). */
export interface InstanciaGrafico {
  actualizar(config: ConfigGrafico): void;
  destruir(): void;
}

export type FabricaGrafico = (lienzo: HTMLCanvasElement, config: ConfigGrafico) => InstanciaGrafico;

function conMovimiento(config: ConfigGrafico): ConfigGrafico {
  const reducido = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches;
  return { ...config, options: { ...config.options, animation: reducido ? false : { duration: 400 } } };
}

export function crearGraficoChartJs(lienzo: HTMLCanvasElement, config: ConfigGrafico): InstanciaGrafico {
  const grafico = new Chart(lienzo, conMovimiento(config));
  return {
    actualizar(nueva) {
      const conAnimacion = conMovimiento(nueva);
      grafico.data = conAnimacion.data;
      grafico.options = conAnimacion.options ?? {};
      grafico.update();
    },
    destruir() {
      grafico.destroy();
    },
  };
}
