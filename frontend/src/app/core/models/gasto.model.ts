export type EstadoGasto = 'Registrado' | 'Pagado' | 'Sustentado';
export type EstadoComprobante = 'Pendiente' | 'Aprobado' | 'Rechazado';

export interface ItemChecklist {
  /** Clave del criterio (estable aunque el cliente cambie su texto) */
  criterio: string;
  /** El criterio tal como estaba cuando se revisó el comprobante */
  texto: string;
  cumple: boolean;
}

export interface Gasto {
  id: number;
  viaje_id: number;
  cliente_id: number;
  monto: number;
  fecha: string;
  estado: EstadoGasto;
  creado_por: string;
  creado_en: string;
}

export interface Comprobante {
  id: number;
  gasto_id: number;
  archivo_url: string;
  tipo_archivo: 'imagen' | 'pdf';
  estado: EstadoComprobante;
  motivo_rechazo?: string | null;
  aprobado_por?: string | null;
  aprobado_en?: string | null;
  creado_en: string;
  /** Lo que Logística verificó al revisarlo; vacío en comprobantes revisados antes del checklist */
  checklist?: ItemChecklist[] | null;
}
