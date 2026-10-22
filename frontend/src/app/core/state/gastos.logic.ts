import { Comprobante, EstadoGasto, Gasto } from '../models/gasto.model';
import { normalizar } from '../utils/texto';

export type Pestana = 'todos' | 'sin-comprobante' | 'por-aprobar' | 'sustentados';

export const PESTANAS: readonly Pestana[] = ['todos', 'sin-comprobante', 'por-aprobar', 'sustentados'];

/** Orden del recorrido de un gasto. */
export const ORDEN_ESTADOS: readonly EstadoGasto[] = ['Registrado', 'Pagado', 'Sustentado'];

export const ESTADO_DE_PESTANA: Record<Exclude<Pestana, 'todos'>, EstadoGasto> = {
  'sin-comprobante': 'Registrado',
  'por-aprobar': 'Pagado',
  sustentados: 'Sustentado',
};

export const PESTANA_DE_ESTADO: Record<EstadoGasto, Exclude<Pestana, 'todos'>> = {
  Registrado: 'sin-comprobante',
  Pagado: 'por-aprobar',
  Sustentado: 'sustentados',
};

export type SiguientePaso = 'adjuntar' | 'revisar' | 'ninguno';

export interface PasoRuta {
  estado: EstadoGasto;
  alcanzado: boolean;
  actual: boolean;
  fecha?: string;
  actor?: string | null;
}

export function esPestana(valor: unknown): valor is Pestana {
  return typeof valor === 'string' && (PESTANAS as readonly string[]).includes(valor);
}

export function filtrarPorPestana(gastos: Gasto[], pestana: Pestana): Gasto[] {
  if (pestana === 'todos') return gastos;
  const estado = ESTADO_DE_PESTANA[pestana];
  return gastos.filter(g => g.estado === estado);
}

/** Estado de gasto que corresponde a una pestaña; «todos» no filtra. */
export function estadoDePestana(pestana: Pestana): EstadoGasto | null {
  return pestana === 'todos' ? null : ESTADO_DE_PESTANA[pestana];
}

export interface FiltroViajeCliente {
  viaje?: number | null;
  cliente?: number | null;
}

/** Deja los gastos del viaje y del cliente elegidos; sin filtro devuelve la misma lista. */
export function filtrarPorViajeYCliente(gastos: Gasto[], filtro: FiltroViajeCliente): Gasto[] {
  return gastos.filter(g => (!filtro.viaje || g.viaje_id === filtro.viaje) && (!filtro.cliente || g.cliente_id === filtro.cliente));
}

/** Lee un id de la URL: solo un entero positivo vale; cualquier otra cosa es «sin filtro». */
export function idDeConsulta(valor: unknown): number | null {
  if (typeof valor !== 'string' || !/^[1-9]\d*$/.test(valor)) return null;
  return Number(valor);
}

export function contarPorEstado(gastos: Gasto[]): Record<EstadoGasto, number> {
  const conteo: Record<EstadoGasto, number> = { Registrado: 0, Pagado: 0, Sustentado: 0 };
  for (const g of gastos) if (g.estado in conteo) conteo[g.estado]++;
  return conteo;
}

export function sumaPorEstado(gastos: Gasto[]): Record<EstadoGasto, number> {
  const suma: Record<EstadoGasto, number> = { Registrado: 0, Pagado: 0, Sustentado: 0 };
  for (const g of gastos) if (g.estado in suma) suma[g.estado] += Number(g.monto) || 0;
  return suma;
}

export function totalMonto(gastos: Gasto[]): number {
  return gastos.reduce((acc, g) => acc + (Number(g.monto) || 0), 0);
}

export function siguientePaso(gasto: Pick<Gasto, 'estado'>): SiguientePaso {
  switch (gasto.estado) {
    case 'Registrado':
      return 'adjuntar';
    case 'Pagado':
      return 'revisar';
    default:
      return 'ninguno';
  }
}

function momento(fecha: string | null | undefined): number {
  const t = fecha ? Date.parse(fecha) : NaN;
  return isNaN(t) ? 0 : t;
}

/** Comprobantes de un gasto, del más reciente al más antiguo. */
export function comprobantesDe(comprobantes: Comprobante[], gastoId: number): Comprobante[] {
  return comprobantes
    .filter(c => c.gasto_id === gastoId)
    .sort((a, b) => momento(b.creado_en) - momento(a.creado_en));
}

/** Si el último comprobante del gasto fue rechazado, lo devuelve para mostrar el motivo. */
export function ultimoRechazo(comprobantes: Comprobante[], gastoId: number): Comprobante | undefined {
  const ultimo = comprobantesDe(comprobantes, gastoId)[0];
  return ultimo?.estado === 'Rechazado' ? ultimo : undefined;
}

/** Los tres hitos del gasto con la fecha y quién en cada uno. */
export function pasosRuta(gasto: Gasto, comprobantes: Comprobante[]): PasoRuta[] {
  const indice = ORDEN_ESTADOS.indexOf(gasto.estado);
  const lista = comprobantesDe(comprobantes, gasto.id);
  const aprobado = lista.find(c => c.estado === 'Aprobado');
  const pendiente = lista.find(c => c.estado === 'Pendiente');
  return ORDEN_ESTADOS.map((estado, i) => {
    const paso: PasoRuta = { estado, alcanzado: i <= indice, actual: i === indice };
    if (estado === 'Registrado') {
      paso.fecha = gasto.creado_en;
      paso.actor = gasto.creado_por;
    } else if (estado === 'Pagado' && paso.alcanzado) {
      paso.fecha = (aprobado ?? pendiente)?.creado_en;
    } else if (estado === 'Sustentado' && paso.alcanzado && aprobado) {
      paso.fecha = aprobado.aprobado_en ?? undefined;
      paso.actor = aprobado.aprobado_por ?? null;
    }
    return paso;
  });
}

export function etiquetaRuta(pasos: PasoRuta[]): string {
  const partes = pasos.map(p => (p.actual ? `${p.estado} (actual)` : p.alcanzado ? p.estado : `${p.estado} pendiente`));
  return `Recorrido: ${partes.join(', ')}`;
}

export function ordenarRecientes(gastos: Gasto[]): Gasto[] {
  return [...gastos].sort((a, b) => momento(b.creado_en) - momento(a.creado_en) || b.id - a.id);
}

export function recientes(gastos: Gasto[], cantidad: number): Gasto[] {
  return ordenarRecientes(gastos).slice(0, cantidad);
}

/** Busca por N° («31», «N° 31», «#31») o por nombre del cliente, sin distinguir tildes. */
export function buscarGastos(gastos: Gasto[], termino: string, nombreCliente: (clienteId: number) => string): Gasto[] {
  const t = normalizar(termino).trim();
  if (!t) return gastos;
  const numero = t.match(/^(?:n[°º.]?\s*|#\s*)?(\d+)$/)?.[1];
  if (numero) return gastos.filter(g => String(g.id) === numero);
  return gastos.filter(g => normalizar(nombreCliente(g.cliente_id)).includes(t));
}
