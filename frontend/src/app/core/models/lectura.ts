import { formatFecha, formatMonto } from '../utils/format';
import { CriterioCliente } from './checklist';
import { Gasto } from './gasto.model';

/** leido = hay sugerencias; los demás estados se muestran como un aviso y la revisión sigue igual. */
export type EstadoLectura = 'leido' | 'sin_texto' | 'no_disponible' | 'error';

export interface DatoLeido<T> {
  valor: T;
  /** La línea del comprobante donde se leyó */
  texto: string;
  confianza: 'alta' | 'media' | 'ninguna';
}

/** El sistema solo sugiere ✓; el detalle dice lo leído, también cuando no coincide con el gasto. */
export interface Sugerencia {
  sugerido: boolean;
  detalle: string;
}

/** Lo que devuelve GET /comprobantes/{id}/lectura. */
export interface LecturaComprobante {
  estado: EstadoLectura;
  datos: { monto: DatoLeido<number>; fecha: DatoLeido<string>; rucs: string[]; tipo: DatoLeido<string> };
  /** Por clave de criterio; vacío si el estado no es leido */
  sugerencias: Record<string, Sugerencia>;
}

export const MSG_LECTURA_FALLIDA = 'No se pudo leer el comprobante automáticamente';
export const MSG_GASTO_NO_CORREGIBLE = 'Este gasto ya no se puede corregir.';

/** Etiqueta junto a la casilla: «✓ Coincide», «⚠ Revisar» (no es un rechazo: pide mirar), o vacía si la lectura no opina. */
export function etiquetaSugerencia(s: Sugerencia | undefined): string {
  if (!s) return '';
  return s.sugerido ? '✓ Coincide' : '⚠ Revisar';
}

/** Claves que la lectura sugiere marcar, solo entre los criterios del cliente y en su orden. */
export function clavesSugeridas(criterios: CriterioCliente[], lectura: LecturaComprobante | undefined): string[] {
  if (lectura?.estado !== 'leido') return [];
  return criterios.filter(c => lectura.sugerencias[c.clave]?.sugerido).map(c => c.clave);
}

/** Una fila del panel «Lectura automática»: lo que dice el gasto frente a lo que leyó el comprobante. */
export interface FilaLectura {
  clave: string;
  dato: string;
  gasto: string;
  leido: string;
  /** false si el comprobante no dio nada para este dato: «leido» es solo el aviso de que falta */
  leyo: boolean;
  coincide: boolean;
  /** El valor leído con el que se puede corregir el gasto (solo monto y fecha); el resto se resuelve con otro comprobante */
  corregible?: { campo: 'monto' | 'fecha'; valor: number | string };
}

/** Los criterios que la lectura puede comparar, en el orden en que se muestran. */
const COMPARABLES: { clave: string; dato: string; sinLectura: string; esperado: (g: Gasto) => string }[] = [
  { clave: 'monto_coincide', dato: 'Monto', sinLectura: 'No se leyó', esperado: g => formatMonto(g.monto) },
  { clave: 'fecha_coincide', dato: 'Fecha', sinLectura: 'No se leyó', esperado: g => formatFecha(g.fecha) },
  { clave: 'a_nombre_empresa', dato: 'Empresa', sinLectura: 'No se encontró', esperado: () => 'LIZAR SRL' },
  { clave: 'tipo_valido', dato: 'Documento', sinLectura: 'No es factura ni boleta', esperado: () => 'Factura o boleta' },
];

/**
 * Filas del panel: solo los criterios que el cliente revisa y que la lectura comenta, y solo con una
 * lectura leída. «leido» sale del detalle de la sugerencia sin su prefijo («Leído S/ 85.50» → «S/ 85.50»).
 */
export function filasLectura(criterios: CriterioCliente[], lectura: LecturaComprobante | undefined, gasto: Gasto): FilaLectura[] {
  if (lectura?.estado !== 'leido') return [];
  const revisados = new Set(criterios.map(c => c.clave));
  return COMPARABLES.filter(c => revisados.has(c.clave) && lectura.sugerencias[c.clave]).map(c => {
    const s = lectura.sugerencias[c.clave];
    const fila: FilaLectura = {
      clave: c.clave,
      dato: c.dato,
      gasto: c.esperado(gasto),
      leido: s.detalle.replace(/^Leíd[oa]\s+/, '') || c.sinLectura,
      leyo: s.detalle !== '',
      coincide: s.sugerido,
    };
    if (!s.sugerido) {
      const { monto, fecha } = lectura.datos;
      if (c.clave === 'monto_coincide' && monto.confianza !== 'ninguna' && monto.valor > 0) fila.corregible = { campo: 'monto', valor: monto.valor };
      if (c.clave === 'fecha_coincide' && fecha.confianza !== 'ninguna' && /^\d{4}-\d{2}-\d{2}$/.test(fecha.valor)) fila.corregible = { campo: 'fecha', valor: fecha.valor };
    }
    return fila;
  });
}

/**
 * El motivo de rechazo que se deduce de la lectura, para que la persona lo confirme o lo edite.
 * Nombra solo lo que el comprobante contradice con un dato leído; un monto o una fecha que no se
 * pudieron leer no prueban nada y no se incluyen. Sin diferencias, vacío.
 */
export function motivoDesdeLectura(filas: FilaLectura[]): string {
  const razones = filas
    .filter(f => !f.coincide)
    .map(f => {
      switch (f.clave) {
        case 'monto_coincide':
          return f.leyo ? `el monto leído (${f.leido}) no coincide con el del gasto (${f.gasto})` : '';
        case 'fecha_coincide':
          return f.leyo ? `la fecha leída (${f.leido}) no coincide con la del gasto (${f.gasto})` : '';
        case 'a_nombre_empresa':
          return 'no aparece a nombre de LIZAR SRL';
        case 'tipo_valido':
          return 'no es una factura ni una boleta';
        default:
          return '';
      }
    })
    .filter(Boolean);
  return razones.length ? `No cumple: ${razones.join('; ')}.` : '';
}
