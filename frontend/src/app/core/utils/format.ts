/** Formato de moneda y fecha en un lenguaje natural para el usuario final. */

const MONTO = new Intl.NumberFormat('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export function formatMonto(valor: number | string | undefined | null): string {
  const n = Number(valor ?? 0);
  return `S/ ${MONTO.format(Number.isFinite(n) ? n : 0)}`;
}

/** Fechas sin hora (fecha del gasto, fecha programada): se leen en UTC para no correr el día. */
export function formatFecha(fecha: string | undefined | null): string {
  if (!fecha) return '—';
  const d = new Date(fecha);
  if (isNaN(d.getTime())) return fecha.slice(0, 10);
  const dd = String(d.getUTCDate()).padStart(2, '0');
  const mm = String(d.getUTCMonth() + 1).padStart(2, '0');
  const yyyy = d.getUTCFullYear();
  return `${dd}/${mm}/${yyyy}`;
}

/** LIZAR opera en Perú: los momentos con hora se muestran en hora de Lima. */
const ZONA = 'America/Lima';

const FECHA_HORA = new Intl.DateTimeFormat('es-PE', {
  timeZone: ZONA,
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
});

const DIA_LIMA = new Intl.DateTimeFormat('en-CA', { timeZone: ZONA });

/** Momentos con hora (registro, comprobante, aprobación) en hora de Lima: 02/10/2026 14:05. */
export function formatFechaHora(fecha: string | undefined | null): string {
  if (!fecha) return '—';
  const d = new Date(fecha);
  if (isNaN(d.getTime())) return fecha;
  const p = Object.fromEntries(FECHA_HORA.formatToParts(d).map(x => [x.type, x.value]));
  return `${p['day']}/${p['month']}/${p['year']} ${p['hour']}:${p['minute']}`;
}

/** Fecha de hoy en Lima (AAAA-MM-DD) para los campos de tipo fecha. */
export function hoyIso(ahora: Date = new Date()): string {
  return DIA_LIMA.format(ahora);
}
