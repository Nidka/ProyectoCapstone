import { Gasto } from '../../core/models/gasto.model';
import { ACCIONES_PALETA, EntradaPaleta, destinosPaleta } from '../../core/navigation/navegacion';
import { buscarGastos, ordenarRecientes } from '../../core/state/gastos.logic';
import { formatMonto } from '../../core/utils/format';
import { normalizar } from '../../core/utils/texto';

export const MAX_GASTOS_PALETA = 6;

/** Acciones, destinos y gastos que coinciden con lo escrito en la paleta (⌘K). */
export function resultadosPaleta(termino: string, gastos: Gasto[], nombreCliente: (clienteId: number) => string): EntradaPaleta[] {
  const fijas = [...ACCIONES_PALETA, ...destinosPaleta()];
  const t = normalizar(termino).trim();
  if (!t) return fijas;
  const coinciden = fijas.filter(e => normalizar(`${e.etiqueta} ${e.detalle ?? ''}`).includes(t));
  const deGastos = buscarGastos(ordenarRecientes(gastos), termino, nombreCliente)
    .slice(0, MAX_GASTOS_PALETA)
    .map<EntradaPaleta>(g => ({
      id: `gasto-${g.id}`,
      grupo: 'Gastos',
      etiqueta: `Gasto N° ${g.id}`,
      detalle: `${nombreCliente(g.cliente_id)} · ${formatMonto(g.monto)}`,
      icono: 'receipt',
      ruta: `/app/gastos/${g.id}`,
    }));
  return [...coinciden, ...deGastos];
}
