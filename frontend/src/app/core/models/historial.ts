import { formatFecha, formatMonto } from '../utils/format';

/** Un movimiento de la bitácora de auditoría, tal como lo devuelve GET /gastos/{id}/historial */
export interface RegistroHistorial {
  /** RFC 3339 con zona */
  fecha: string;
  /** Nombre de quien hizo el cambio */
  usuario: string;
  /** Código corto: «gasto.registrar», «comprobante.aprobar»… */
  accion: string;
  entidad: string;
  id_entidad: number | null;
  /** Valores antes y después del cambio; su forma depende de la acción */
  detalle: Record<string, unknown>;
}

const ACCIONES: Record<string, string> = {
  'gasto.registrar': 'Registró el gasto',
  'gasto.corregir': 'Corrigió el gasto',
  'comprobante.adjuntar': 'Adjuntó un comprobante',
  'comprobante.aprobar': 'Aprobó el comprobante',
  'comprobante.rechazar': 'Rechazó el comprobante',
  'checklist.configurar': 'Configuró el checklist del cliente',
  'checklist.restablecer': 'Volvió al checklist estándar del cliente',
  'usuario.crear': 'Creó un usuario',
  'usuario.editar': 'Editó un usuario',
  'usuario.desactivar': 'Desactivó un usuario',
  'usuario.activar': 'Activó un usuario',
  'usuario.cambiar_clave': 'Cambió una contraseña',
  'sesion.inicio': 'Inició sesión',
  'sesion.cierre': 'Cerró sesión',
  'maestro.crear': 'Creó un registro',
  'maestro.editar': 'Editó un registro',
  'maestro.eliminar': 'Eliminó un registro',
};

const ACCION_DESCONOCIDA = 'Hizo un cambio';

/** La acción en español sencillo; un código que esta pantalla no conoce sale como «Hizo un cambio». */
export function textoAccion(accion: string): string {
  return Object.hasOwn(ACCIONES, accion) ? ACCIONES[accion] : ACCION_DESCONOCIDA;
}

function objeto(valor: unknown): Record<string, unknown> | null {
  return valor !== null && typeof valor === 'object' && !Array.isArray(valor) ? (valor as Record<string, unknown>) : null;
}

/** Una línea extra para lo que importa al leer el historial: el motivo de un rechazo, lo que cambió una corrección o el avance del checklist. */
export function detalleDeRegistro(r: RegistroHistorial): string {
  const detalle = objeto(r.detalle);
  if (!detalle) return '';
  if (r.accion === 'comprobante.rechazar') {
    const motivo = objeto(detalle['despues'])?.['motivo_rechazo'];
    return typeof motivo === 'string' && motivo.trim() ? `Motivo: ${motivo.trim()}` : '';
  }
  if (r.accion === 'gasto.corregir') {
    const antes = objeto(detalle['antes']);
    const despues = objeto(detalle['despues']);
    if (!antes || !despues) return '';
    const partes: string[] = [];
    if (antes['monto'] !== despues['monto']) partes.push(`Monto: ${formatMonto(antes['monto'] as number)} → ${formatMonto(despues['monto'] as number)}`);
    if (antes['fecha'] !== despues['fecha']) partes.push(`Fecha: ${formatFecha(antes['fecha'] as string)} → ${formatFecha(despues['fecha'] as string)}`);
    const motivo = detalle['motivo'];
    if (typeof motivo === 'string' && motivo.trim()) partes.push(`Motivo: ${motivo.trim()}`);
    return partes.join(' · ');
  }
  if (r.accion === 'comprobante.aprobar') {
    const puntos = detalle['checklist'];
    if (!Array.isArray(puntos) || puntos.length === 0) return '';
    const cumplen = puntos.filter(p => objeto(p)?.['cumple'] === true).length;
    return `Checklist: ${cumplen} de ${puntos.length} puntos`;
  }
  return '';
}

/** Del más antiguo al más reciente. En un empate se respeta el orden en que llegaron (el servidor desempata por número). */
export function ordenarCronologico(registros: readonly RegistroHistorial[]): RegistroHistorial[] {
  const instante = (r: RegistroHistorial) => {
    const t = new Date(r.fecha).getTime();
    return Number.isNaN(t) ? 0 : t;
  };
  return [...registros].sort((a, b) => instante(a) - instante(b));
}
