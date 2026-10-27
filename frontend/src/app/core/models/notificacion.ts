export type TipoNotificacion = 'sin_comprobante' | 'cobro_proximo';

/** Un aviso del servicio (GET /notificaciones): lo genera cada hora y es solo de Logística */
export interface Notificacion {
  id: number;
  tipo: TipoNotificacion;
  id_gasto: number;
  /** Texto ya redactado por el servicio, en español */
  mensaje: string;
  creada_en: string;
  leida: boolean;
  leida_en: string | null;
}

export interface RespuestaNotificaciones {
  /** Todas las pendientes, aunque la lista traiga solo las 100 más recientes */
  no_leidas: number;
  notificaciones: Notificacion[];
}

const TITULOS: Record<TipoNotificacion, string> = {
  sin_comprobante: 'Falta el comprobante',
  cobro_proximo: 'Cobro cerca',
};

/** Título corto del aviso; un tipo que esta pantalla no conoce sale como «Aviso». */
export function tituloAviso(tipo: string): string {
  return Object.hasOwn(TITULOS, tipo) ? TITULOS[tipo as TipoNotificacion] : 'Aviso';
}

/** Número de la insignia de la campana: se corta en 99+ para que no ensanche la barra. */
export function textoInsignia(noLeidas: number): string {
  return noLeidas > 99 ? '99+' : String(noLeidas);
}
