import { EstadoGasto } from './gasto.model';

/** Rango de fechas del gasto (AAAA-MM-DD), con ambos extremos incluidos. */
export interface FiltroReporte {
  desde: string;
  hasta: string;
}

export type TipoPdf = 'estadistico' | 'detalle';

export interface IndicadoresReporte {
  total: number;
  cantidad: number;
  promedio: number;
  sustentado: number;
  pendiente: number;
  porcentaje_sustentado: number;
  /** ISC (HU-15): gastos sustentados ÷ gastos totales × 100, por cantidad */
  gastos_sustentados: number;
  isc: number;
}

/** Gasto de un mes (AAAA-MM) separado por estado. */
export interface MesReporte {
  mes: string;
  registrado: number;
  pagado: number;
  sustentado: number;
  total: number;
}

export interface EstadoReporte {
  estado: EstadoGasto;
  cantidad: number;
  monto: number;
  porcentaje: number;
}

export interface FilaRanking {
  puesto: number;
  etiqueta: string;
  cantidad: number;
  monto: number;
  porcentaje: number;
}

/** Respuesta de GET /reportes/resumen (gastos-service). */
export interface ResumenReporte {
  desde: string;
  hasta: string;
  indicadores: IndicadoresReporte;
  por_mes: MesReporte[];
  por_estado: EstadoReporte[];
  por_cliente: FilaRanking[];
  por_unidad: FilaRanking[];
  por_conductor: FilaRanking[];
}

/** Un archivo recibido del servicio, listo para guardarse. */
export interface ArchivoDescargable {
  blob: Blob;
  nombre: string;
}

export type ArchivoPdf = ArchivoDescargable;

/** Filtros de GET /gastos/exportar.xlsx; los vacíos no se envían. */
export interface ParamsExcel {
  viaje_id?: number | null;
  cliente_id?: number | null;
  estado?: string | null;
  desde?: string;
  hasta?: string;
}
