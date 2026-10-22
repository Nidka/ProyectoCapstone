import { Gasto } from './gasto.model';
import { formatFecha, formatMonto } from '../utils/format';

/** Un punto del checklist de un cliente. La clave no cambia al renombrar el texto. */
export interface CriterioCliente {
  clave: string;
  texto: string;
}

/** Lo que devuelve GET /clientes/{id}/checklist: los criterios que se revisan en sus comprobantes. */
export interface ChecklistCliente {
  cliente_id: number;
  /** false = el cliente usa los criterios estándar */
  personalizado: boolean;
  criterios: CriterioCliente[];
}

/** Checklist en curso: un valor por clave de criterio. */
export type Checklist = Record<string, boolean>;

export const MSG_COMPROBANTE_YA_REVISADO = 'Este comprobante ya fue revisado. Actualizamos la vista.';
export const MSG_CHECKLIST_INCOMPLETO = 'Para aprobar, el comprobante debe cumplir todos los criterios del checklist.';

export const MAX_CRITERIOS = 15;
export const MIN_TEXTO = 3;
export const MAX_TEXTO = 120;

export function checklistVacio(criterios: CriterioCliente[]): Checklist {
  return Object.fromEntries(criterios.map(c => [c.clave, false]));
}

export function cumplidos(criterios: CriterioCliente[], checklist: Checklist): number {
  return criterios.filter(c => checklist[c.clave]).length;
}

/** Completo = todos los criterios marcados; sin criterios no hay checklist que dar por cumplido. */
export function checklistCompleto(criterios: CriterioCliente[], checklist: Checklist): boolean {
  return criterios.length > 0 && cumplidos(criterios, checklist) === criterios.length;
}

function enMinuscula(texto: string): string {
  const sinPunto = texto.replace(/\.$/, '');
  return sinPunto.charAt(0).toLowerCase() + sinPunto.slice(1);
}

/** «No cumple: el monto coincide con el gasto; …» con lo que quedó sin marcar; vacío si cumple todo. */
export function motivoDesdeChecklist(criterios: CriterioCliente[], checklist: Checklist): string {
  const faltan = criterios.filter(c => !checklist[c.clave]).map(c => enMinuscula(c.texto));
  return faltan.length ? `No cumple: ${faltan.join('; ')}.` : '';
}

/**
 * El texto del criterio para revisar. Los criterios de monto y de fecha muestran el dato del gasto
 * para que quien revisa compare sin salir de la pantalla.
 */
export function textoConDato(criterio: CriterioCliente, gasto: Gasto): string {
  const dato =
    criterio.clave === 'monto_coincide' ? formatMonto(gasto.monto) : criterio.clave === 'fecha_coincide' ? formatFecha(gasto.fecha) : '';
  return dato ? `${criterio.texto.replace(/\.$/, '')} (${dato}).` : criterio.texto;
}
