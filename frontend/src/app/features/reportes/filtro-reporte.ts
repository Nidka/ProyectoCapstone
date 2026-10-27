import { FiltroReporte } from '../../core/models/reporte.model';

/** Atajos de rango del dashboard; el primero es el rango por defecto. */
export type Atajo = 'ultimos-12' | 'este-anio' | 'este-mes';

export const ATAJOS: readonly { id: Atajo; etiqueta: string }[] = [
  { id: 'ultimos-12', etiqueta: 'Últimos 12 meses' },
  { id: 'este-anio', etiqueta: 'Este año' },
  { id: 'este-mes', etiqueta: 'Este mes' },
];

/** Los mismos mensajes que devuelve gastos-service (sección 3 de la especificación). */
export const MSG_FECHA_INVALIDA = 'Fecha inválida: use AAAA-MM-DD.';
export const MSG_RANGO_INVERTIDO = 'La fecha inicial no puede ser posterior a la final.';
export const MSG_RANGO_LARGO = 'El rango no puede superar 24 meses.';

export const MSG_FECHA_FUERA_DEL_RANGO =
  'Esta fecha es anterior a los últimos 12 meses: el gasto no saldrá en el dashboard con su rango por defecto. Revisa que sea la correcta.';
export const MSG_FECHA_FUTURA = 'Esta fecha es posterior a hoy: el gasto no saldrá en el dashboard hasta ese día.';

const RANGO_MAXIMO_DIAS = 731;
const ISO = /^(\d{4})-(\d{2})-(\d{2})$/;
const MESES = ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'];

/** AAAA-MM-DD de un día; Date.UTC corrige meses fuera de rango (mes 0 = diciembre del año anterior). */
function iso(anio: number, mes: number, dia: number): string {
  return new Date(Date.UTC(anio, mes - 1, dia)).toISOString().slice(0, 10);
}

function anioMes(fecha: string): [number, number] {
  const [anio, mes] = fecha.split('-').map(Number);
  return [anio, mes];
}

export function rangoDeAtajo(atajo: Atajo, hoy: string): FiltroReporte {
  const [anio, mes] = anioMes(hoy);
  switch (atajo) {
    case 'ultimos-12':
      return { desde: iso(anio, mes - 11, 1), hasta: hoy };
    case 'este-anio':
      return { desde: iso(anio, 1, 1), hasta: hoy };
    case 'este-mes':
      return { desde: iso(anio, mes, 1), hasta: hoy };
  }
}

/** Atajos cuyo rango es el actual; en enero y en diciembre pueden coincidir dos. */
export function atajosActivos(filtro: FiltroReporte, hoy: string): Atajo[] {
  return ATAJOS.filter(a => {
    const r = rangoDeAtajo(a.id, hoy);
    return r.desde === filtro.desde && r.hasta === filtro.hasta;
  }).map(a => a.id);
}

/** El rango de la URL, con los mismos valores por defecto que gastos-service. */
export function filtroDesdeConsulta(desde: string | null | undefined, hasta: string | null | undefined, hoy: string): FiltroReporte {
  const fin = hasta || hoy;
  const [anio, mes] = anioMes(fechaValida(fin) ? fin : hoy);
  return { desde: desde || iso(anio, mes - 11, 1), hasta: fin };
}

export function fechaValida(fecha: string): boolean {
  const m = ISO.exec(fecha);
  return !!m && iso(+m[1], +m[2], +m[3]) === fecha;
}

/** Mensaje de error del rango, o null si es válido. */
export function errorDeFiltro(filtro: FiltroReporte): string | null {
  if (!fechaValida(filtro.desde) || !fechaValida(filtro.hasta)) return MSG_FECHA_INVALIDA;
  if (filtro.desde > filtro.hasta) return MSG_RANGO_INVERTIDO;
  const dias = (Date.parse(filtro.hasta) - Date.parse(filtro.desde)) / 86_400_000 + 1;
  return dias > RANGO_MAXIMO_DIAS ? MSG_RANGO_LARGO : null;
}

/** El mes completo AAAA-MM: del día 1 al último día. */
export function rangoDelMes(mes: string): FiltroReporte {
  const [anio, m] = anioMes(mes);
  return { desde: iso(anio, m, 1), hasta: iso(anio, m + 1, 0) };
}

/** AAAA-MM si el rango es exactamente un mes completo; si no, null. */
export function mesCompleto(filtro: FiltroReporte): string | null {
  const mes = filtro.desde.slice(0, 7);
  const r = rangoDelMes(mes);
  return r.desde === filtro.desde && r.hasta === filtro.hasta ? mes : null;
}

/** «octubre 2026» */
export function nombreMes(mes: string): string {
  const [anio, m] = anioMes(mes);
  return `${MESES[m - 1]} ${anio}`;
}

/**
 * Aviso para quien registra un gasto con una fecha que el dashboard no mostraría con su rango por
 * defecto (los últimos 12 meses hasta hoy): el reporte cuenta por la fecha del gasto, no por el día en
 * que se registró. Vacío si la fecha cae dentro o no es una fecha real.
 */
export function avisoFechaGasto(fecha: string, hoy: string): string {
  if (!fecha || !fechaValida(fecha)) return '';
  if (fecha > hoy) return MSG_FECHA_FUTURA;
  return fecha < rangoDeAtajo('ultimos-12', hoy).desde ? MSG_FECHA_FUERA_DEL_RANGO : '';
}
