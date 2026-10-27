/** Administrador del Sistema (`admin`) o Encargado de Logística (`logistica`). */
export type Rol = 'admin' | 'logistica';

export const ROL_ETIQUETA: Record<Rol, string> = {
  admin: 'Administrador del Sistema',
  logistica: 'Encargado de Logística',
};

export interface LoginRequest {
  username: string;
  password: string;
}

export interface UsuarioOut {
  id_usuario: number;
  username: string;
  email: string;
  nombre: string;
  rol: Rol;
  estado: boolean;
  /** El servidor la pone en true para cuentas nuevas o con la clave restablecida; mientras sea true solo deja cambiar la contraseña. */
  debe_cambiar_clave?: boolean;
}

export interface TokenResponse {
  access_token: string;
  token_type: string;
  usuario: UsuarioOut;
  /** Lo mismo que `usuario.debe_cambiar_clave`, también en la raíz de la respuesta */
  debe_cambiar_clave?: boolean;
}

/** Texto con que el servidor responde 403 a todo lo que no sea cambiar la contraseña. */
export const DETALLE_DEBE_CAMBIAR_CLAVE = 'Debes cambiar tu contraseña';

/** ¿El inicio de sesión exige cambiar la contraseña antes de seguir? */
export function exigeCambioDeClave(res: Partial<TokenResponse>): boolean {
  return (res.usuario?.debe_cambiar_clave ?? res.debe_cambiar_clave) === true;
}
