/** Busca el nombre legible de un registro de otro módulo: ref('cliente', 5) → «Agro Norte SAC». */
export type RefLookup = (clave: string, id: unknown) => string;

/** Definición de un campo de formulario CRUD */
export interface CrudField {
  id: string;
  l: string;               // label
  tp: 'text' | 'number' | 'textarea' | 'email' | 'date' | 'ref';
  req?: boolean;
  half?: boolean;          // ocupa la mitad del grid
  placeholder?: string;
  rule?: (v: string) => boolean;
  msg?: string;            // mensaje de error de regla
  ref?: string;            // tp 'ref': clave CRUD del módulo relacionado
}

/** Textos con género de cada módulo: «Nuevo viaje», «Dirección guardada». */
export interface CrudTextos {
  nuevo: string;
  guardado: string;
  singular: string;
  plural: string;
}

/** Definición completa de un módulo CRUD */
export interface CrudDef {
  key: string;
  title: string;
  route: string;           // ruta API: /conductores
  nk: string;              // id histórico del módulo
  group: 'Maestros' | 'Operaciones' | 'Logística' | 'Finanzas';
  desc: string;
  fields: CrudField[];
  pay: (form: Record<string, { value: string }>) => Record<string, unknown>;
  cols: string[];
  row: (item: Record<string, unknown>, ref: RefLookup) => string[];
  idKey?: string;          // campo identificador en la API
  nombre?: (item: Record<string, unknown>, ref: RefLookup) => string;
  textos?: CrudTextos;
  /** Acción de cada fila que continúa en otro módulo (p. ej. registrar un gasto del viaje) */
  accionFila?: { texto: string; ruta: string; parametro: string };
}

/** Opción de un selector de registros relacionados */
export interface OpcionRef {
  valor: string;
  etiqueta: string;
}

/** Toast */
export type ToastType = 'ok' | 'err' | 'info' | 'warn';
export interface Toast {
  id: number;
  title: string;
  msg: string;
  type: ToastType;
}
