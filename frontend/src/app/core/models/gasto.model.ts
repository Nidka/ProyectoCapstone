export type EstadoGasto = 'Registrado' | 'Pagado' | 'Sustentado';
export type EstadoComprobante = 'Pendiente' | 'Aprobado' | 'Rechazado';

/** Criterios del checklist documental (las claves coinciden con gastos-service). */
export type CriterioChecklist = 'legible' | 'tipo_valido' | 'a_nombre_empresa' | 'monto_coincide' | 'fecha_coincide';

export interface ItemChecklist {
  criterio: CriterioChecklist;
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
