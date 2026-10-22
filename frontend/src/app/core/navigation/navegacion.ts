import type { IconoNombre } from '../../shared/ui/icono/iconos';

export type SeccionId = 'operacion' | 'catalogos';

export interface ModuloNav {
  slug: string;
  clave: string;
  etiqueta: string;
  icono: IconoNombre;
  seccion: SeccionId;
}

export interface SeccionNav {
  id: SeccionId;
  etiqueta: string;
  icono: IconoNombre;
  modulos: ModuloNav[];
}

export interface EnlacePrincipal {
  etiqueta: string;
  ruta: string;
  icono: IconoNombre;
}

export type GrupoPaleta = 'Acciones' | 'Ir a' | 'Gastos';

export interface EntradaPaleta {
  id: string;
  grupo: GrupoPaleta;
  etiqueta: string;
  detalle?: string;
  icono: IconoNombre;
  ruta: string;
  queryParams?: Record<string, string>;
}

const OPERACION: ModuloNav[] = [
  { slug: 'viajes', clave: 'viaje', etiqueta: 'Viajes', icono: 'road', seccion: 'operacion' },
  { slug: 'clientes', clave: 'cliente', etiqueta: 'Clientes', icono: 'building', seccion: 'operacion' },
  { slug: 'conductores', clave: 'conductor', etiqueta: 'Conductores', icono: 'id-badge-2', seccion: 'operacion' },
  { slug: 'flota', clave: 'unidad', etiqueta: 'Flota', icono: 'truck', seccion: 'operacion' },
  { slug: 'asignaciones', clave: 'cond_uni', etiqueta: 'Asignaciones', icono: 'link', seccion: 'operacion' },
  { slug: 'direcciones', clave: 'direccion', etiqueta: 'Direcciones', icono: 'map-pin', seccion: 'operacion' },
];

const CATALOGOS: ModuloNav[] = [
  { slug: 'ubigeo', clave: 'ubigeo', etiqueta: 'Ubigeo', icono: 'map-2', seccion: 'catalogos' },
  { slug: 'rubros', clave: 'rubro', etiqueta: 'Rubros', icono: 'category', seccion: 'catalogos' },
  { slug: 'marcas', clave: 'marca', etiqueta: 'Marcas', icono: 'award', seccion: 'catalogos' },
  { slug: 'tipos-unidad', clave: 'tipo_unidad', etiqueta: 'Tipos de unidad', icono: 'list-details', seccion: 'catalogos' },
  { slug: 'subtipos-unidad', clave: 'subtipo_unidad', etiqueta: 'Subtipos de unidad', icono: 'list-tree', seccion: 'catalogos' },
];

export const SECCIONES: readonly SeccionNav[] = [
  { id: 'operacion', etiqueta: 'Operación', icono: 'route', modulos: OPERACION },
  { id: 'catalogos', etiqueta: 'Catálogos', icono: 'folders', modulos: CATALOGOS },
];

export const MODULOS: readonly ModuloNav[] = [...OPERACION, ...CATALOGOS];

export const ENLACES_PRINCIPALES: readonly EnlacePrincipal[] = [
  { etiqueta: 'Inicio', ruta: '/app/inicio', icono: 'home' },
  { etiqueta: 'Gastos', ruta: '/app/gastos', icono: 'receipt' },
  { etiqueta: 'Reportes', ruta: '/app/reportes', icono: 'chart-bar' },
  { etiqueta: 'Operación', ruta: '/app/operacion', icono: 'route' },
  { etiqueta: 'Catálogos', ruta: '/app/catalogos', icono: 'folders' },
];

export const ACCIONES_PALETA: readonly EntradaPaleta[] = [
  { id: 'accion-registrar', grupo: 'Acciones', etiqueta: 'Registrar gasto', icono: 'plus', ruta: '/app/gastos/nuevo' },
  {
    id: 'accion-adjuntar', grupo: 'Acciones', etiqueta: 'Adjuntar comprobante', icono: 'paperclip',
    ruta: '/app/gastos', queryParams: { estado: 'sin-comprobante' },
  },
  {
    id: 'accion-revisar', grupo: 'Acciones', etiqueta: 'Revisar comprobantes por aprobar', icono: 'clipboard-check',
    ruta: '/app/gastos', queryParams: { estado: 'por-aprobar' },
  },
];

export function seccion(id: SeccionId): SeccionNav {
  return SECCIONES.find(s => s.id === id) ?? SECCIONES[0];
}

export function esSeccion(valor: unknown): valor is SeccionId {
  return valor === 'operacion' || valor === 'catalogos';
}

export function moduloPorSlug(seccionId: SeccionId, slug: string | null | undefined): ModuloNav | undefined {
  return seccion(seccionId).modulos.find(m => m.slug === slug);
}

export function moduloPorClave(clave: string): ModuloNav | undefined {
  return MODULOS.find(m => m.clave === clave);
}

export function rutaModulo(modulo: ModuloNav): string {
  return `/app/${modulo.seccion}/${modulo.slug}`;
}

export function rutaDeClave(clave: string): string | null {
  const modulo = moduloPorClave(clave);
  return modulo ? rutaModulo(modulo) : null;
}

export function destinosPaleta(): EntradaPaleta[] {
  return [
    { id: 'ir-inicio', grupo: 'Ir a', etiqueta: 'Inicio', icono: 'home', ruta: '/app/inicio' },
    { id: 'ir-gastos', grupo: 'Ir a', etiqueta: 'Gastos', icono: 'receipt', ruta: '/app/gastos' },
    { id: 'ir-reportes', grupo: 'Ir a', etiqueta: 'Reportes', icono: 'chart-bar', ruta: '/app/reportes' },
    ...MODULOS.map<EntradaPaleta>(m => ({
      id: `ir-${m.seccion}-${m.slug}`,
      grupo: 'Ir a',
      etiqueta: m.etiqueta,
      detalle: seccion(m.seccion).etiqueta,
      icono: m.icono,
      ruta: rutaModulo(m),
    })),
  ];
}

/** Tras iniciar sesión se vuelve a la pantalla pedida solo si es parte de la app. */
export function destinoTrasLogin(volver: string | null | undefined): string {
  return volver && volver.startsWith('/app/') ? volver : '/app/inicio';
}
