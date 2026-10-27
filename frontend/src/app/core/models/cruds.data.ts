import { CrudDef, RefLookup } from './crud.model';
import { formatFecha } from '../utils/format';

/* ── Helpers de render de celdas ── */
const esc = (v: unknown) =>
  String(v ?? '').replace(/[&<>"']/g, m => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m] ?? m));
const VACIO = '<span class="celda-vacia">—</span>';
const p = (v: unknown) => (v == null || v === '' ? VACIO : esc(v));
const mkId = (n: unknown) => `<span class="id-tag">N° ${p(n)}</span>`;
const mkMono = (s: unknown) => `<span class="mono">${p(s)}</span>`;
const mkFuerte = (s: unknown) => `<strong>${p(s)}</strong>`;
const mkSec = (s: unknown) => `<span class="celda-secundaria">${p(s)}</span>`;
const mkDoble = (principal: unknown, secundario: unknown) =>
  `<span class="celda-doble"><strong>${p(principal)}</strong>${secundario ? `<span class="celda-secundaria">${esc(secundario)}</span>` : ''}</span>`;
const mkMoney = (n: unknown) => {
  if (n == null || n === '') return VACIO;
  return `<span class="money"><span class="sy">S/</span>${Number(n).toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>`;
};
const mkPlate = (s: unknown) =>
  s ? `<span class="plate"><span class="plate-pe">PE</span><span class="plate-num">${esc(s)}</span></span>` : VACIO;
const FLECHA =
  '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l14 0"/><path d="M13 18l6 -6"/><path d="M13 6l6 6"/></svg>';
const mkRoute = (a: string, b: string) => `<span class="route"><span>${esc(a)}</span>${FLECHA}<span>${esc(b)}</span></span>`;
const mkSt = (v: unknown, tl = 'Activo', fl = 'Inactivo') => {
  const ok = v ?? true;
  return `<span class="lbl ${ok ? 'lbl-ok' : 'lbl-err'}"><span class="lbl__poste"></span>${ok ? tl : fl}</span>`;
};
const mkRef = (ref: RefLookup, clave: string, id: unknown) => (id == null || id === '' ? VACIO : esc(ref(clave, id)));
/** Unidad (placa · modelo) en mono y sin partirse, como las placas */
const mkUnidad = (ref: RefLookup, id: unknown) =>
  id == null || id === '' ? VACIO : `<span class="mono sin-corte">${esc(ref('unidad', id))}</span>`;
const mkFecha = (v: unknown) => (v ? esc(formatFecha(String(v))) : VACIO);
const texto = (v: unknown) => String(v ?? '');

/* ════════════════════════════════════
   MÓDULOS CRUD
   Visibles en esta iteración: los 11 primeros (Maestros, Operaciones y Logística).
   Los del final siguen ocultos (no están en los requerimientos actuales).
════════════════════════════════════ */
export const CRUDS: CrudDef[] = [

  /* ── MAESTROS ── */
  {
    key: 'ubigeo', title: 'Ubigeo', route: '/ubigeos', nk: 'ubigeo', group: 'Maestros',
    desc: 'Distritos, provincias y departamentos del Perú',
    idKey: 'codigo_ubigeo',
    nombre: i => `${texto(i['distrito'])}, ${texto(i['provincia'])} (${texto(i['codigo_ubigeo']).padStart(6, '0')})`,
    textos: { nuevo: 'Nuevo ubigeo', guardado: 'Ubigeo guardado', singular: 'ubigeo', plural: 'ubigeos' },
    fields: [
      { id: 'ub_c', l: 'Código (6 dígitos)', tp: 'number', req: true, half: true, rule: v => /^\d{6}$/.test(v), msg: 'Debe tener 6 dígitos' },
      { id: 'ub_dp', l: 'Departamento', tp: 'text', req: true, half: true },
      { id: 'ub_p', l: 'Provincia', tp: 'text', req: true, half: true },
      { id: 'ub_d', l: 'Distrito', tp: 'text', req: true, half: true },
    ],
    pay: f => ({ codigo_ubigeo: Number(f['ub_c'].value), departamento: f['ub_dp'].value, provincia: f['ub_p'].value, distrito: f['ub_d'].value }),
    cols: ['Código', 'Departamento', 'Provincia', 'Distrito'],
    row: i => [mkMono(texto(i['codigo_ubigeo']).padStart(6, '0')), mkFuerte(i['departamento']), p(i['provincia']), p(i['distrito'])],
  },
  {
    key: 'rubro', title: 'Rubro', route: '/rubros', nk: 'rubro', group: 'Maestros',
    desc: 'Clasificación de actividades económicas de los clientes',
    idKey: 'id_rubro',
    nombre: i => texto(i['nombre']),
    textos: { nuevo: 'Nuevo rubro', guardado: 'Rubro guardado', singular: 'rubro', plural: 'rubros' },
    fields: [
      { id: 'ru_n', l: 'Nombre', tp: 'text', req: true },
      { id: 'ru_d', l: 'Descripción', tp: 'text', req: false },
    ],
    pay: f => ({ nombre: f['ru_n'].value, descripcion: f['ru_d'].value || null }),
    cols: ['N°', 'Nombre', 'Descripción'],
    row: i => [mkId(i['id_rubro'] ?? i['id']), mkFuerte(i['nombre']), mkSec(i['descripcion'])],
  },
  {
    key: 'marca', title: 'Marca', route: '/marcas', nk: 'marca', group: 'Maestros',
    desc: 'Fabricantes de vehículos de la flota',
    idKey: 'id_marca',
    nombre: i => texto(i['nombre']),
    textos: { nuevo: 'Nueva marca', guardado: 'Marca guardada', singular: 'marca', plural: 'marcas' },
    fields: [
      { id: 'ma_n', l: 'Nombre', tp: 'text', req: true, half: true },
      { id: 'ma_p', l: 'País de origen', tp: 'text', req: false, half: true, placeholder: 'Ej: Alemania' },
    ],
    pay: f => ({ nombre: f['ma_n'].value, pais_origen: f['ma_p'].value || null }),
    cols: ['N°', 'Marca', 'País de origen'],
    row: i => [mkId(i['id_marca'] ?? i['id']), mkFuerte(i['nombre']), p(i['pais_origen'])],
  },
  {
    key: 'tipo_unidad', title: 'Tipo de unidad', route: '/tipos-unidad', nk: 'tipo_unidad', group: 'Maestros',
    desc: 'Tracto, camión, semirremolque y otros tipos de vehículo',
    idKey: 'id_tipo_unidad',
    nombre: i => texto(i['nombre']),
    textos: { nuevo: 'Nuevo tipo de unidad', guardado: 'Tipo de unidad guardado', singular: 'tipo de unidad', plural: 'tipos de unidad' },
    fields: [
      { id: 'tu_n', l: 'Nombre', tp: 'text', req: true, placeholder: 'Ej: Tracto' },
      { id: 'tu_d', l: 'Descripción', tp: 'text', req: false },
    ],
    pay: f => ({ nombre: f['tu_n'].value, descripcion: f['tu_d'].value || null }),
    cols: ['N°', 'Tipo', 'Descripción'],
    row: i => [mkId(i['id_tipo_unidad'] ?? i['id']), mkFuerte(i['nombre']), mkSec(i['descripcion'])],
  },
  {
    key: 'subtipo_unidad', title: 'Subtipo de unidad', route: '/subtipos-unidad', nk: 'subtipo_unidad', group: 'Maestros',
    desc: 'Subclasificación dentro de cada tipo de vehículo',
    idKey: 'id_subtipo_unidad',
    nombre: (i, ref) => `${texto(i['nombre'])} · ${ref('tipo_unidad', i['id_tipo_unidad'])}`,
    textos: { nuevo: 'Nuevo subtipo', guardado: 'Subtipo guardado', singular: 'subtipo de unidad', plural: 'subtipos de unidad' },
    fields: [
      { id: 'su_t', l: 'Tipo de unidad', tp: 'ref', ref: 'tipo_unidad', req: true, half: true },
      { id: 'su_n', l: 'Nombre', tp: 'text', req: true, half: true, placeholder: 'Ej: Plataforma' },
      { id: 'su_d', l: 'Descripción', tp: 'text', req: false },
    ],
    pay: f => ({ id_tipo_unidad: Number(f['su_t'].value), nombre: f['su_n'].value, descripcion: f['su_d'].value || null }),
    cols: ['N°', 'Subtipo', 'Tipo de unidad', 'Descripción'],
    row: (i, ref) => [mkId(i['id_subtipo_unidad'] ?? i['id']), mkFuerte(i['nombre']), mkRef(ref, 'tipo_unidad', i['id_tipo_unidad']), mkSec(i['descripcion'])],
  },

  /* ── OPERACIONES ── */
  {
    key: 'direccion', title: 'Dirección', route: '/direcciones', nk: 'direccion', group: 'Operaciones',
    desc: 'Puntos de origen, destino y domicilios',
    idKey: 'id_direccion',
    nombre: i => texto(i['direccion_detallada']),
    textos: { nuevo: 'Nueva dirección', guardado: 'Dirección guardada', singular: 'dirección', plural: 'direcciones' },
    fields: [
      { id: 'di_u', l: 'Ubigeo', tp: 'ref', ref: 'ubigeo', req: true },
      { id: 'di_d', l: 'Dirección detallada', tp: 'text', req: true, placeholder: 'Av. o Jr. y número' },
      { id: 'di_r', l: 'Referencia', tp: 'text', req: false, placeholder: 'Ej: frente al parque' },
    ],
    pay: f => ({ codigo_ubigeo: Number(f['di_u'].value), direccion_detallada: f['di_d'].value, referencia: f['di_r'].value || null }),
    cols: ['N°', 'Dirección', 'Ubigeo', 'Referencia'],
    row: (i, ref) => [mkId(i['id_direccion'] ?? i['id']), mkFuerte(i['direccion_detallada']), mkRef(ref, 'ubigeo', i['codigo_ubigeo']), mkSec(i['referencia'])],
  },
  {
    key: 'conductor', title: 'Conductor', route: '/conductores', nk: 'conductor', group: 'Operaciones',
    desc: 'Personal conductor habilitado con licencia vigente',
    idKey: 'id_conductor',
    nombre: i => `${texto(i['nombre1'])} ${texto(i['apellido1'])}`.trim(),
    textos: { nuevo: 'Nuevo conductor', guardado: 'Conductor guardado', singular: 'conductor', plural: 'conductores' },
    fields: [
      { id: 'co_dni', l: 'DNI', tp: 'text', req: true, half: true, rule: v => /^\d{8}$/.test(v), msg: '8 dígitos' },
      { id: 'co_lic', l: 'N° Licencia', tp: 'text', req: true, half: true },
      { id: 'co_n1', l: 'Primer nombre', tp: 'text', req: true, half: true },
      { id: 'co_a1', l: 'Apellido paterno', tp: 'text', req: true, half: true },
      { id: 'co_n2', l: 'Segundo nombre', tp: 'text', req: false, half: true },
      { id: 'co_a2', l: 'Apellido materno', tp: 'text', req: false, half: true },
      { id: 'co_tel', l: 'Teléfono', tp: 'text', req: false, half: true, placeholder: '+51 9' },
      { id: 'co_cor', l: 'Correo', tp: 'email', req: false, half: true },
    ],
    pay: f => ({
      dni: f['co_dni'].value, nombre1: f['co_n1'].value, nombre2: f['co_n2'].value || null, apellido1: f['co_a1'].value,
      apellido2: f['co_a2'].value || null, licencia_conducir: f['co_lic'].value, telefono: f['co_tel'].value || null, correo: f['co_cor'].value || null,
    }),
    cols: ['Conductor', 'DNI', 'Licencia', 'Teléfono', 'Estado'],
    row: i => [
      mkDoble(`${texto(i['nombre1'])} ${texto(i['apellido1'])}`.trim(), [i['nombre2'], i['apellido2']].filter(Boolean).join(' ')),
      mkMono(i['dni']), mkMono(i['licencia_conducir']), p(i['telefono']), mkSt(i['estado_conductor'] ?? true),
    ],
  },
  {
    key: 'cliente', title: 'Cliente', route: '/clientes', nk: 'cliente', group: 'Operaciones',
    desc: 'Empresas y personas que contratan el servicio',
    idKey: 'id_cliente',
    nombre: i => texto(i['razon_social']),
    textos: { nuevo: 'Nuevo cliente', guardado: 'Cliente guardado', singular: 'cliente', plural: 'clientes' },
    accionFila: { texto: 'Checklist', ruta: '/app/checklist', parametro: 'cliente', icono: 'clipboard-check', roles: ['logistica'] },
    fields: [
      { id: 'cl_r', l: 'Razón social', tp: 'text', req: true },
      { id: 'cl_ruc', l: 'RUC', tp: 'text', req: true, half: true, rule: v => /^\d{11}$/.test(v), msg: '11 dígitos' },
      { id: 'cl_rb', l: 'Rubro', tp: 'ref', ref: 'rubro', req: true, half: true },
      { id: 'cl_t', l: 'Teléfono', tp: 'text', req: false, half: true, placeholder: '+51' },
      { id: 'cl_e', l: 'Correo', tp: 'email', req: false, half: true },
      { id: 'cl_c', l: 'Persona de contacto', tp: 'text', req: false },
    ],
    pay: f => ({
      razon_social: f['cl_r'].value, ruc: f['cl_ruc'].value, telefono: f['cl_t'].value || null, correo: f['cl_e'].value || null,
      contacto: f['cl_c'].value || null, id_rubro: Number(f['cl_rb'].value),
    }),
    cols: ['Empresa', 'RUC', 'Rubro', 'Teléfono', 'Estado'],
    row: (i, ref) => [mkDoble(i['razon_social'], i['correo']), mkMono(i['ruc']), mkRef(ref, 'rubro', i['id_rubro']), p(i['telefono']), mkSt(i['estado_cliente'] ?? true)],
  },
  {
    key: 'unidad', title: 'Flota', route: '/unidades', nk: 'unidad', group: 'Operaciones',
    desc: 'Tractos, camiones y remolques registrados',
    idKey: 'id_unidad',
    nombre: i => [i['placa'], i['modelo']].filter(Boolean).map(texto).join(' · '),
    textos: { nuevo: 'Nueva unidad', guardado: 'Unidad guardada', singular: 'unidad', plural: 'unidades' },
    fields: [
      { id: 'un_p', l: 'Placa', tp: 'text', req: true, half: true, placeholder: 'ABC-123' },
      { id: 'un_a', l: 'Año', tp: 'number', req: false, half: true, rule: v => !v || Number(v) >= 1900, msg: 'Inválido' },
      { id: 'un_ti', l: 'Tipo de unidad', tp: 'ref', ref: 'tipo_unidad', req: true, half: true },
      { id: 'un_s', l: 'Subtipo', tp: 'ref', ref: 'subtipo_unidad', req: false, half: true },
      { id: 'un_m', l: 'Marca', tp: 'ref', ref: 'marca', req: true, half: true },
      { id: 'un_mo', l: 'Modelo', tp: 'text', req: false, half: true, placeholder: 'Ej: Actros 2646' },
      { id: 'un_pt', l: 'Peso tara (ton)', tp: 'number', req: false, half: true },
      { id: 'un_pm', l: 'Peso máx. (ton)', tp: 'number', req: false, half: true },
    ],
    pay: f => ({
      placa: f['un_p'].value.toUpperCase(), id_tipo_unidad: Number(f['un_ti'].value),
      id_subtipo_unidad: f['un_s'].value ? Number(f['un_s'].value) : null, id_marca: Number(f['un_m'].value),
      modelo: f['un_mo'].value || null, anio: f['un_a'].value ? Number(f['un_a'].value) : null,
      peso_tara: f['un_pt'].value ? Number(f['un_pt'].value) : null, peso_maximo: f['un_pm'].value ? Number(f['un_pm'].value) : null,
    }),
    cols: ['Placa', 'Modelo', 'Marca', 'Tipo', 'Año', 'Estado'],
    row: (i, ref) => [
      mkPlate(i['placa']), p(i['modelo']), mkRef(ref, 'marca', i['id_marca']), mkRef(ref, 'tipo_unidad', i['id_tipo_unidad']),
      p(i['anio']), mkSt(i['estado_unidad'] ?? true, 'Operativa', 'Inactiva'),
    ],
  },
  {
    key: 'cond_uni', title: 'Asignación', route: '/asignaciones-conductor-unidad', nk: 'cond_uni', group: 'Operaciones',
    desc: 'Qué conductor maneja cada unidad',
    idKey: 'id_conductor_unidad',
    nombre: (i, ref) => `${ref('conductor', i['id_conductor'])} · ${ref('unidad', i['id_unidad'])}`,
    textos: { nuevo: 'Nueva asignación', guardado: 'Asignación guardada', singular: 'asignación', plural: 'asignaciones' },
    fields: [
      { id: 'cu_c', l: 'Conductor', tp: 'ref', ref: 'conductor', req: true, half: true },
      { id: 'cu_u', l: 'Unidad', tp: 'ref', ref: 'unidad', req: true, half: true },
      { id: 'cu_f', l: 'Fecha de asignación', tp: 'date', req: true },
    ],
    pay: f => ({ id_conductor: Number(f['cu_c'].value), id_unidad: Number(f['cu_u'].value), fecha_asignacion: f['cu_f'].value }),
    cols: ['N°', 'Conductor', 'Unidad', 'Desde', 'Estado'],
    row: (i, ref) => [
      mkId(i['id_conductor_unidad'] ?? i['id']), `<strong>${mkRef(ref, 'conductor', i['id_conductor'])}</strong>`,
      mkUnidad(ref, i['id_unidad']), mkFecha(i['fecha_asignacion']), mkSt(i['estado_asignacion'] ?? true, 'Activa', 'Finalizada'),
    ],
  },

  /* ── LOGÍSTICA ── */
  {
    key: 'viaje', title: 'Viaje', route: '/viajes', nk: 'viaje', group: 'Logística',
    desc: 'Servicios de transporte de carga programados',
    idKey: 'id_viaje',
    nombre: i => `Viaje N° ${texto(i['id_viaje'])} · ${formatFecha(i['fecha_programado'] as string | null)}`,
    textos: { nuevo: 'Nuevo viaje', guardado: 'Viaje guardado', singular: 'viaje', plural: 'viajes' },
    accionFila: { texto: 'Registrar gasto', ruta: '/app/gastos/nuevo', parametro: 'viaje', roles: ['logistica'] },
    fields: [
      { id: 'vi_tr', l: 'Tracto', tp: 'ref', ref: 'unidad', req: true, half: true },
      { id: 'vi_ca', l: 'Carreta', tp: 'ref', ref: 'unidad', req: false, half: true },
      { id: 'vi_cu', l: 'Asignación conductor-unidad', tp: 'ref', ref: 'cond_uni', req: true },
      { id: 'vi_cl', l: 'Cliente', tp: 'ref', ref: 'cliente', req: true },
      { id: 'vi_o', l: 'Dirección de origen', tp: 'ref', ref: 'direccion', req: true, half: true },
      { id: 'vi_d', l: 'Dirección de destino', tp: 'ref', ref: 'direccion', req: true, half: true },
      { id: 'vi_f', l: 'Fecha programada', tp: 'date', req: true },
    ],
    pay: f => ({
      id_tracto: Number(f['vi_tr'].value), id_carreta: f['vi_ca'].value ? Number(f['vi_ca'].value) : null,
      id_conductor_unidad: Number(f['vi_cu'].value), id_cliente: Number(f['vi_cl'].value),
      id_direccion_origen: Number(f['vi_o'].value), id_direccion_destino: Number(f['vi_d'].value), fecha_programado: f['vi_f'].value,
    }),
    cols: ['N°', 'Ruta', 'Cliente', 'Tracto', 'Fecha', 'Estado'],
    row: (i, ref) => [
      mkId(i['id_viaje'] ?? i['id']),
      mkRoute(ref('direccion', i['id_direccion_origen']), ref('direccion', i['id_direccion_destino'])),
      mkRef(ref, 'cliente', i['id_cliente']), mkUnidad(ref, i['id_tracto']), mkFecha(i['fecha_programado']),
      mkSt(i['estado_viaje'] ?? true, 'Programado', 'Cancelado'),
    ],
  },

  /* ── OCULTOS EN ESTA ITERACIÓN ── */
  {
    key: 'tipo_gasto', title: 'Tipo de gasto', route: '/tipos-gasto', nk: 'tipo_gasto', group: 'Maestros',
    desc: 'Combustible, peajes, mantenimiento',
    fields: [{ id: 'tg_n', l: 'Nombre', tp: 'text', req: true, placeholder: 'Ej: Combustible' }, { id: 'tg_d', l: 'Descripción', tp: 'text', req: false }],
    pay: f => ({ nombre: f['tg_n'].value, descripcion: f['tg_d'].value || null }),
    cols: ['N°', 'Tipo de gasto', 'Descripción'],
    row: i => [mkId(i['id_tipo_gasto'] ?? i['id']), mkFuerte(i['nombre']), mkSec(i['descripcion'])],
  },
  {
    key: 'cat_prod', title: 'Categoría de producto', route: '/categorias-producto', nk: 'cat_prod', group: 'Maestros',
    desc: 'Clasificación de mercancías transportadas',
    fields: [{ id: 'cp_n', l: 'Nombre', tp: 'text', req: true, placeholder: 'Ej: Alimentos' }, { id: 'cp_d', l: 'Descripción', tp: 'text', req: false }],
    pay: f => ({ nombre: f['cp_n'].value, descripcion: f['cp_d'].value || null }),
    cols: ['N°', 'Categoría', 'Descripción'],
    row: i => [mkId(i['id_categoria'] ?? i['id']), `<span class="lbl lbl-info">${p(i['nombre'])}</span>`, mkSec(i['descripcion'])],
  },
  {
    key: 'guia', title: 'Guía de transportista', route: '/guias-transportista', nk: 'guia', group: 'Logística',
    desc: 'Guías de remisión del transportista (GRT)',
    fields: [
      { id: 'gu_n', l: 'N° de guía GRT', tp: 'text', req: true, placeholder: 'GRT-001-0001' },
      { id: 'gu_v', l: 'ID Viaje', tp: 'number', req: true, half: true },
      { id: 'gu_p', l: 'Peso total (kg)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
      { id: 'gu_r', l: 'Guías remitente', tp: 'text', req: false },
    ],
    pay: f => ({ numero_guia: f['gu_n'].value, id_viaje: Number(f['gu_v'].value), peso_total: Number(f['gu_p'].value), guias_remitente: f['gu_r'].value || null }),
    cols: ['N° GRT', 'Viaje', 'Peso', 'Estado'],
    row: i => [
      mkMono(i['numero_guia']), `<span class="lbl lbl-n">Viaje #${p(i['id_viaje'])}</span>`,
      `<strong>${p(i['peso_total'])} <small class="celda-secundaria">kg</small></strong>`, mkSt(i['estado_guia'] ?? true, 'Vigente', 'Anulada'),
    ],
  },
  {
    key: 'viaje_r', title: 'Viaje realizado', route: '/viajes-realizados', nk: 'viaje_r', group: 'Logística',
    desc: 'Servicios de transporte completados con flete',
    fields: [
      { id: 'vr_g', l: 'ID Guía GRT', tp: 'number', req: true, half: true },
      { id: 'vr_f', l: 'Flete (S/)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
      { id: 'vr_o', l: 'Observaciones', tp: 'textarea', req: false, placeholder: 'Incidencias del viaje' },
    ],
    pay: f => ({ id_guia_transportista: Number(f['vr_g'].value), flete: Number(f['vr_f'].value), observaciones: f['vr_o'].value || null }),
    cols: ['N°', 'GRT', 'Flete', 'Estado'],
    row: i => [
      mkId(i['id_viaje_realizado'] ?? i['id']), `<span class="mono">GRT #${p(i['id_guia_transportista'])}</span>`,
      mkMoney(i['flete']), mkSt(i['estado_viaje_realizado'] ?? true, 'Completado', 'Pendiente'),
    ],
  },
  {
    key: 'prod_t', title: 'Productos transportados', route: '/productos-transportados', nk: 'prod_t', group: 'Logística',
    desc: 'Detalle de mercancías por guía de remisión',
    fields: [
      { id: 'pt_g', l: 'ID Guía GRT', tp: 'number', req: true, half: true },
      { id: 'pt_c', l: 'ID Categoría', tp: 'number', req: true, half: true },
      { id: 'pt_d', l: 'Descripción', tp: 'text', req: true, placeholder: 'Ej: Harina de trigo' },
      { id: 'pt_ca', l: 'Cantidad', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
      { id: 'pt_p', l: 'Peso (kg)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({
      id_guia_transportista: Number(f['pt_g'].value), id_categoria: Number(f['pt_c'].value), descripcion_producto: f['pt_d'].value,
      cantidad: Number(f['pt_ca'].value), peso: Number(f['pt_p'].value),
    }),
    cols: ['N°', 'Producto', 'Cant.', 'Peso', 'GRT'],
    row: i => [
      mkId(i['id_producto_transportado'] ?? i['id']), mkFuerte(i['descripcion_producto']), mkFuerte(i['cantidad']),
      `<strong>${p(i['peso'])} <small class="celda-secundaria">kg</small></strong>`, `<span class="mono">GRT #${p(i['id_guia_transportista'])}</span>`,
    ],
  },
  {
    key: 'factura', title: 'Factura', route: '/facturas', nk: 'factura', group: 'Finanzas',
    desc: 'Documentos de cobro por servicios de transporte',
    fields: [
      { id: 'fa_n', l: 'N° de factura', tp: 'text', req: true, placeholder: 'F001-00001' },
      { id: 'fa_v', l: 'ID Viaje realizado', tp: 'number', req: true, half: true },
      { id: 'fa_m', l: 'Monto total (S/)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({ numero_factura: f['fa_n'].value, id_viaje_realizado: Number(f['fa_v'].value), monto_total: Number(f['fa_m'].value) }),
    cols: ['N° Factura', 'Viaje realizado', 'Monto', 'Estado'],
    row: i => [
      mkMono(i['numero_factura']), `<span class="lbl lbl-n">VR #${p(i['id_viaje_realizado'])}</span>`,
      mkMoney(i['monto_total']), mkSt(i['estado_factura'] ?? true, 'Activa', 'Anulada'),
    ],
  },
  {
    key: 'presupuesto', title: 'Presupuesto de viaje', route: '/presupuestos-viaje', nk: 'presupuesto', group: 'Finanzas',
    desc: 'Estimación de costos operativos por viaje',
    fields: [
      { id: 'pe_v', l: 'ID Viaje', tp: 'number', req: true, half: true },
      { id: 'pe_m', l: 'Monto total (S/)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({ id_viaje: Number(f['pe_v'].value), monto_total: Number(f['pe_m'].value) }),
    cols: ['N°', 'Viaje', 'Monto', 'Estado'],
    row: i => [
      mkId(i['id_presupuesto'] ?? i['id']), `<span class="lbl lbl-n">Viaje #${p(i['id_viaje'])}</span>`,
      mkMoney(i['monto_total']), mkSt(i['estado_presupuesto'] ?? true, 'Aprobado', 'Pendiente'),
    ],
  },
  {
    key: 'det_pres', title: 'Detalle de presupuesto', route: '/detalles-presupuesto', nk: 'det_pres', group: 'Finanzas',
    desc: 'Desglose de gastos por tipo de cada presupuesto',
    fields: [
      { id: 'dp_p', l: 'ID Presupuesto', tp: 'number', req: true, half: true },
      { id: 'dp_t', l: 'ID Tipo de gasto', tp: 'number', req: true, half: true },
      { id: 'dp_m', l: 'Monto estimado (S/)', tp: 'number', req: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({ id_presupuesto_viaje: Number(f['dp_p'].value), id_tipo_gasto: Number(f['dp_t'].value), monto_estimado: Number(f['dp_m'].value) }),
    cols: ['N°', 'Presupuesto', 'Tipo de gasto', 'Monto est.'],
    row: i => [
      mkId(i['id_detalle_presupuesto'] ?? i['id']), `<span class="lbl lbl-n">Pres. #${p(i['id_presupuesto'] ?? i['id_presupuesto_viaje'])}</span>`,
      `<span class="lbl lbl-warn">TG #${p(i['id_tipo_gasto'])}</span>`, mkMoney(i['monto_estimado']),
    ],
  },
];
