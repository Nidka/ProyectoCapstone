import { CriterioChecklist, Gasto } from './gasto.model';
import { formatFecha, formatMonto } from '../utils/format';

/** Checklist documental en curso: un valor por criterio. */
export type Checklist = Record<CriterioChecklist, boolean>;

interface DefinicionCriterio {
  clave: CriterioChecklist;
  /** Texto corto del historial */
  corto: string;
  /** Texto de la revisión; el monto y la fecha salen del gasto para comparar */
  texto: (gasto: Gasto) => string;
}

/** Criterios del checklist documental, en el orden en que se revisan. */
export const CRITERIOS_CHECKLIST: readonly DefinicionCriterio[] = [
  { clave: 'legible', corto: 'Legible y completo', texto: () => 'El comprobante es legible y está completo.' },
  { clave: 'tipo_valido', corto: 'Boleta o factura válida', texto: () => 'Es una boleta o factura válida.' },
  { clave: 'a_nombre_empresa', corto: 'A nombre de LIZAR SRL', texto: () => 'Está emitido a nombre de LIZAR SRL.' },
  { clave: 'monto_coincide', corto: 'Monto coincide', texto: g => `El monto coincide con el gasto (${formatMonto(g.monto)}).` },
  { clave: 'fecha_coincide', corto: 'Fecha coincide', texto: g => `La fecha corresponde al gasto (${formatFecha(g.fecha)}).` },
];

export const TOTAL_CRITERIOS = CRITERIOS_CHECKLIST.length;

export const MSG_COMPROBANTE_YA_REVISADO = 'Este comprobante ya fue revisado. Actualizamos la vista.';
export const MSG_CHECKLIST_INCOMPLETO = 'Para aprobar, el comprobante debe cumplir los 5 criterios del checklist.';

export function checklistVacio(): Checklist {
  return { legible: false, tipo_valido: false, a_nombre_empresa: false, monto_coincide: false, fecha_coincide: false };
}

export function cumplidos(checklist: Checklist): number {
  return CRITERIOS_CHECKLIST.filter(c => checklist[c.clave]).length;
}

export function checklistCompleto(checklist: Checklist): boolean {
  return cumplidos(checklist) === TOTAL_CRITERIOS;
}

/** «No cumple: monto coincide; fecha coincide.» con lo que quedó sin marcar; vacío si cumple todo. */
export function motivoDesdeChecklist(checklist: Checklist): string {
  const faltan = CRITERIOS_CHECKLIST.filter(c => !checklist[c.clave]).map(c => c.corto.charAt(0).toLowerCase() + c.corto.slice(1));
  return faltan.length ? `No cumple: ${faltan.join('; ')}.` : '';
}

export function cortoDe(clave: CriterioChecklist): string {
  return CRITERIOS_CHECKLIST.find(c => c.clave === clave)?.corto ?? clave;
}
