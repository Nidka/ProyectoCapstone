import { HttpErrorResponse } from '@angular/common/http';
import { mensajeError } from '../utils/error-messages';
import { Rol, UsuarioOut } from './auth.model';

/** Límites que también aplica el servidor (gestión de usuarios). */
export const MAX_USERNAME = 50;
export const MAX_EMAIL = 100;
export const MAX_NOMBRE = 100;
export const MIN_CLAVE = 8;

/** Cuerpo de POST /usuarios. */
export interface NuevoUsuario {
  username: string;
  email: string;
  nombre: string;
  rol: Rol;
  password: string;
}

/** Cuerpo de PUT /usuarios/{id}: solo lo que cambia. El usuario (username) nunca cambia. */
export type CambiosUsuario = Partial<Pick<NuevoUsuario, 'email' | 'nombre' | 'rol' | 'password'>>;

/** Cuerpo de POST /auth/cambiar-clave. */
export interface CambiarClaveRequest {
  clave_actual: string;
  clave_nueva: string;
}

/** Lo que se escribe en el formulario de un usuario. */
export type ValoresUsuario = NuevoUsuario;
export type CampoUsuario = keyof ValoresUsuario;
export type ErroresUsuario = Partial<Record<CampoUsuario, string>>;

export interface ErroresDelServidor<C extends string> {
  /** Mensaje por campo; los que el servidor rechazó */
  campos: Partial<Record<C, string>>;
  /** Aviso que no pertenece a un campo; vacío si no hay */
  general: string;
}

const CAMPOS_USUARIO: readonly CampoUsuario[] = ['username', 'email', 'nombre', 'rol', 'password'];
const MSG_LARGO = (max: number) => `Usa hasta ${max} caracteres.`;
const MSG_CLAVE_CORTA = `Usa al menos ${MIN_CLAVE} caracteres.`;
const MSG_REVISAR = 'Revisa este dato.';
const MSG_REVISAR_DATOS = 'Revisa los datos e inténtalo de nuevo.';
const CORREO = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Las mismas reglas que el servidor, para avisar antes de enviar. En «editar» el usuario no cuenta y la contraseña es opcional. */
export function validarUsuario(v: ValoresUsuario, modo: 'crear' | 'editar'): ErroresUsuario {
  const errores: ErroresUsuario = {};

  if (modo === 'crear') {
    const username = v.username.trim();
    if (!username) errores.username = 'Escribe el usuario.';
    else if (/\s/.test(username)) errores.username = 'El usuario no puede tener espacios.';
    else if (username.length > MAX_USERNAME) errores.username = MSG_LARGO(MAX_USERNAME);
  }

  const email = v.email.trim();
  if (!email) errores.email = 'Escribe el correo.';
  else if (email.length > MAX_EMAIL) errores.email = MSG_LARGO(MAX_EMAIL);
  else if (!CORREO.test(email)) errores.email = 'Escribe un correo válido.';

  const nombre = v.nombre.trim();
  if (!nombre) errores.nombre = 'Escribe el nombre.';
  else if (nombre.length > MAX_NOMBRE) errores.nombre = MSG_LARGO(MAX_NOMBRE);

  if (v.rol !== 'admin' && v.rol !== 'logistica') errores.rol = 'Elige un rol.';

  if (modo === 'crear' && !v.password) errores.password = 'Escribe la contraseña.';
  else if (v.password && v.password.length < MIN_CLAVE) errores.password = MSG_CLAVE_CORTA;

  return errores;
}

/** Solo lo que cambió respecto al usuario guardado: el servidor actualiza únicamente esos campos. */
export function cambiosDeUsuario(original: UsuarioOut, v: ValoresUsuario): CambiosUsuario {
  const cambios: CambiosUsuario = {};
  const nombre = v.nombre.trim();
  const email = v.email.trim();
  if (nombre !== original.nombre) cambios.nombre = nombre;
  if (email !== original.email) cambios.email = email;
  if (v.rol !== original.rol) cambios.rol = v.rol;
  if (v.password) cambios.password = v.password;
  return cambios;
}

function detalleTexto(err: unknown): string {
  const detalle = err instanceof HttpErrorResponse ? (err.error as { detail?: unknown } | null)?.detail : undefined;
  return typeof detalle === 'string' ? detalle : '';
}

/** Campos que un 422 señala: el último tramo de `loc` en cada error. */
function camposRechazados(err: HttpErrorResponse): string[] {
  const detalle = (err.error as { detail?: unknown } | null)?.detail;
  if (!Array.isArray(detalle)) return [];
  return detalle.map(e => (Array.isArray(e?.loc) ? String(e.loc[e.loc.length - 1]) : '')).filter(Boolean);
}

/**
 * Traduce el rechazo del servidor al guardar un usuario. Los textos de campo salen de las reglas
 * locales (en español); el servidor solo dice cuál campo falló.
 */
export function erroresDeUsuario(err: unknown, v: ValoresUsuario): ErroresDelServidor<CampoUsuario> {
  const campos: ErroresUsuario = {};
  if (err instanceof HttpErrorResponse && err.status === 422) {
    const locales = validarUsuario(v, 'crear');
    for (const campo of camposRechazados(err)) {
      if ((CAMPOS_USUARIO as readonly string[]).includes(campo)) {
        const c = campo as CampoUsuario;
        campos[c] = locales[c] ?? MSG_REVISAR;
      }
    }
    return { campos, general: Object.keys(campos).length ? '' : MSG_REVISAR_DATOS };
  }
  if (err instanceof HttpErrorResponse && err.status === 409) {
    const detalle = detalleTexto(err);
    if (/^El usuario .* ya existe/.test(detalle)) return { campos: { username: detalle }, general: '' };
    if (/^El email /.test(detalle)) return { campos: { email: detalle }, general: '' };
    if (detalle) return { campos, general: detalle };
  }
  return { campos, general: mensajeError(err) };
}

/** Motivo en español con que el servidor rechazó una acción (p. ej. 409 «Debe quedar al menos un administrador activo»). */
export function motivoDelServidor(err: unknown): string {
  return err instanceof HttpErrorResponse && err.status === 409 && detalleTexto(err) ? detalleTexto(err) : mensajeError(err);
}

export interface ValoresCambioClave {
  actual: string;
  nueva: string;
  confirmacion: string;
}
export type CampoCambioClave = keyof ValoresCambioClave;
export type ErroresCambioClave = Partial<Record<CampoCambioClave, string>>;

export function validarCambioClave(v: ValoresCambioClave): ErroresCambioClave {
  const errores: ErroresCambioClave = {};
  if (!v.actual) errores.actual = 'Escribe tu contraseña actual.';
  if (v.nueva.length < MIN_CLAVE) errores.nueva = MSG_CLAVE_CORTA;
  else if (v.nueva === v.actual) errores.nueva = 'La contraseña nueva debe ser distinta de la actual.';
  if (!v.confirmacion) errores.confirmacion = 'Repite la contraseña nueva.';
  else if (v.confirmacion !== v.nueva) errores.confirmacion = 'Las contraseñas no coinciden.';
  return errores;
}

/** Rechazo del servidor a POST /auth/cambiar-clave: 400 (actual incorrecta o nueva repetida) o 422 (formato). */
export function erroresDeCambioClave(err: unknown): ErroresDelServidor<CampoCambioClave> {
  if (err instanceof HttpErrorResponse && err.status === 400) {
    const detalle = detalleTexto(err);
    if (/actual no es correcta/i.test(detalle)) return { campos: { actual: detalle }, general: '' };
    if (/distinta/i.test(detalle)) return { campos: { nueva: detalle }, general: '' };
    if (detalle) return { campos: {}, general: detalle };
  }
  if (err instanceof HttpErrorResponse && err.status === 422) {
    return { campos: { nueva: MSG_CLAVE_CORTA }, general: '' };
  }
  return { campos: {}, general: mensajeError(err) };
}
