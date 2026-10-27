import { describe, expect, it } from 'vitest';
import { HttpErrorResponse } from '@angular/common/http';
import { TimeoutError } from 'rxjs';
import {
  ValoresUsuario, cambiosDeUsuario, erroresDeCambioClave, erroresDeUsuario, validarCambioClave, validarUsuario,
} from './usuario';
import { MSG_ERROR_RED, MSG_TIMEOUT } from '../utils/error-messages';

const VALIDO: ValoresUsuario = { username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'logistica', password: 'clave-larga' };
const http = (status: number, detail: unknown) => new HttpErrorResponse({ status, error: { detail } });

describe('validarUsuario (espejo del servidor)', () => {
  it('un usuario completo no tiene errores', () => {
    expect(validarUsuario(VALIDO, 'crear')).toEqual({});
  });

  it('el usuario es obligatorio, sin espacios y de hasta 50 caracteres', () => {
    expect(validarUsuario({ ...VALIDO, username: '   ' }, 'crear').username).toBe('Escribe el usuario.');
    expect(validarUsuario({ ...VALIDO, username: 'ana quispe' }, 'crear').username).toBe('El usuario no puede tener espacios.');
    expect(validarUsuario({ ...VALIDO, username: 'a'.repeat(51) }, 'crear').username).toBe('Usa hasta 50 caracteres.');
    expect(validarUsuario({ ...VALIDO, username: 'a'.repeat(50) }, 'crear').username).toBeUndefined();
  });

  it('el correo debe ser válido y de hasta 100 caracteres', () => {
    expect(validarUsuario({ ...VALIDO, email: '' }, 'crear').email).toBe('Escribe el correo.');
    expect(validarUsuario({ ...VALIDO, email: 'ana-lizar.pe' }, 'crear').email).toBe('Escribe un correo válido.');
    expect(validarUsuario({ ...VALIDO, email: 'ana@lizar' }, 'crear').email).toBe('Escribe un correo válido.');
    expect(validarUsuario({ ...VALIDO, email: `${'a'.repeat(96)}@l.pe` }, 'crear').email).toBe('Usa hasta 100 caracteres.');
  });

  it('el nombre es obligatorio y de hasta 100 caracteres', () => {
    expect(validarUsuario({ ...VALIDO, nombre: ' ' }, 'crear').nombre).toBe('Escribe el nombre.');
    expect(validarUsuario({ ...VALIDO, nombre: 'n'.repeat(101) }, 'crear').nombre).toBe('Usa hasta 100 caracteres.');
  });

  it('la contraseña pide al menos 8 caracteres al crear', () => {
    expect(validarUsuario({ ...VALIDO, password: '' }, 'crear').password).toBe('Escribe la contraseña.');
    expect(validarUsuario({ ...VALIDO, password: '1234567' }, 'crear').password).toBe('Usa al menos 8 caracteres.');
    expect(validarUsuario({ ...VALIDO, password: '12345678' }, 'crear').password).toBeUndefined();
  });

  it('al editar el usuario no cuenta y la contraseña es opcional, pero si se escribe pide 8', () => {
    expect(validarUsuario({ ...VALIDO, username: '', password: '' }, 'editar')).toEqual({});
    expect(validarUsuario({ ...VALIDO, password: 'corta' }, 'editar').password).toBe('Usa al menos 8 caracteres.');
  });

  it('solo existen los roles admin y logistica', () => {
    expect(validarUsuario({ ...VALIDO, rol: 'operador' as never }, 'crear').rol).toBe('Elige un rol.');
  });
});

describe('cambiosDeUsuario', () => {
  const ORIGINAL = { id_usuario: 2, username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'logistica' as const, estado: true };

  it('envía solo lo que cambió, sin espacios sobrantes', () => {
    expect(cambiosDeUsuario(ORIGINAL, { ...VALIDO, nombre: '  Ana María Quispe ', password: '' })).toEqual({ nombre: 'Ana María Quispe' });
    expect(cambiosDeUsuario(ORIGINAL, { ...VALIDO, rol: 'admin', email: 'nueva@lizar.pe', password: '' })).toEqual({ rol: 'admin', email: 'nueva@lizar.pe' });
  });

  it('incluye la contraseña solo si se escribió una', () => {
    expect(cambiosDeUsuario(ORIGINAL, { ...VALIDO, password: 'otra-clave-1' })).toEqual({ password: 'otra-clave-1' });
  });

  it('sin cambios queda vacío', () => {
    expect(cambiosDeUsuario(ORIGINAL, { ...VALIDO, password: '' })).toEqual({});
  });
});

describe('erroresDeUsuario (lo que responde el servidor)', () => {
  it('un 422 marca los campos que el servidor señala, con texto en español', () => {
    const err = new HttpErrorResponse({
      status: 422,
      error: { detail: [{ loc: ['body', 'email'], msg: 'value is not a valid email address', type: 'value_error' }, { loc: ['body', 'password'], msg: 'x', input: null }] },
    });
    expect(erroresDeUsuario(err, { ...VALIDO, email: 'mal', password: '123' })).toEqual({
      campos: { email: 'Escribe un correo válido.', password: 'Usa al menos 8 caracteres.' },
      general: '',
    });
  });

  it('un 422 de un campo que aquí parecía bien usa un texto genérico', () => {
    const err = new HttpErrorResponse({ status: 422, error: { detail: [{ loc: ['body', 'nombre'], msg: 'x' }] } });
    expect(erroresDeUsuario(err, VALIDO).campos).toEqual({ nombre: 'Revisa este dato.' });
  });

  it('un 422 que no señala ningún campo conocido da un aviso general', () => {
    expect(erroresDeUsuario(http(422, [{ loc: ['path', 'id'], msg: 'x' }]), VALIDO)).toEqual({
      campos: {},
      general: 'Revisa los datos e inténtalo de nuevo.',
    });
  });

  it('un 409 por usuario o correo repetido se muestra en su campo', () => {
    expect(erroresDeUsuario(http(409, "El usuario 'aquispe' ya existe"), VALIDO)).toEqual({
      campos: { username: "El usuario 'aquispe' ya existe" },
      general: '',
    });
    expect(erroresDeUsuario(http(409, "El email 'ana@lizar.pe' ya está registrado"), VALIDO).campos).toEqual({
      email: "El email 'ana@lizar.pe' ya está registrado",
    });
  });

  it('otro 409 (último administrador) es un aviso general con el texto del servidor', () => {
    expect(erroresDeUsuario(http(409, 'Debe quedar al menos un administrador activo'), VALIDO)).toEqual({
      campos: {},
      general: 'Debe quedar al menos un administrador activo',
    });
  });

  it('un fallo de red o de tiempo usa los textos de siempre', () => {
    expect(erroresDeUsuario(new HttpErrorResponse({ status: 0 }), VALIDO).general).toBe(MSG_ERROR_RED);
    expect(erroresDeUsuario(new TimeoutError(), VALIDO).general).toBe(MSG_TIMEOUT);
  });
});

describe('validarCambioClave', () => {
  const ok = { actual: 'clave-vieja', nueva: 'clave-nueva-1', confirmacion: 'clave-nueva-1' };

  it('con los tres datos bien no hay errores', () => {
    expect(validarCambioClave(ok)).toEqual({});
  });

  it('pide la contraseña actual', () => {
    expect(validarCambioClave({ ...ok, actual: '' }).actual).toBe('Escribe tu contraseña actual.');
  });

  it('la nueva lleva al menos 8 caracteres y debe ser distinta de la actual', () => {
    expect(validarCambioClave({ ...ok, nueva: '1234567', confirmacion: '1234567' }).nueva).toBe('Usa al menos 8 caracteres.');
    expect(validarCambioClave({ actual: 'misma-clave-1', nueva: 'misma-clave-1', confirmacion: 'misma-clave-1' }).nueva).toBe(
      'La contraseña nueva debe ser distinta de la actual.',
    );
  });

  it('la confirmación debe coincidir', () => {
    expect(validarCambioClave({ ...ok, confirmacion: '' }).confirmacion).toBe('Repite la contraseña nueva.');
    expect(validarCambioClave({ ...ok, confirmacion: 'otra-distinta' }).confirmacion).toBe('Las contraseñas no coinciden.');
  });
});

describe('erroresDeCambioClave', () => {
  it('la contraseña actual incorrecta (400) se marca en su campo', () => {
    expect(erroresDeCambioClave(http(400, 'La contraseña actual no es correcta'))).toEqual({
      campos: { actual: 'La contraseña actual no es correcta' },
      general: '',
    });
  });

  it('la nueva igual a la actual (400) se marca en el campo de la nueva', () => {
    expect(erroresDeCambioClave(http(400, 'La contraseña nueva debe ser distinta de la actual')).campos).toEqual({
      nueva: 'La contraseña nueva debe ser distinta de la actual',
    });
  });

  it('un 422 pide revisar la contraseña nueva', () => {
    expect(erroresDeCambioClave(http(422, [{ loc: ['body', 'clave_nueva'], msg: 'x', input: null }])).campos).toEqual({
      nueva: 'Usa al menos 8 caracteres.',
    });
  });

  it('un fallo de red usa el texto de siempre', () => {
    expect(erroresDeCambioClave(new HttpErrorResponse({ status: 0 })).general).toBe(MSG_ERROR_RED);
  });
});
