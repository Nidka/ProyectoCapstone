import { beforeEach, describe, expect, it, vi } from 'vitest';
import { HttpErrorResponse } from '@angular/common/http';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Observable, Subject, of, throwError } from 'rxjs';
import { UsuariosComponent } from './usuarios.component';
import { UsuarioOut } from '../../core/models/auth.model';
import { AuthService } from '../../core/services/auth.service';
import { ToastService } from '../../core/services/toast.service';
import { UsuariosService } from '../../core/services/usuarios.service';
import { MSG_ERROR_RED } from '../../core/utils/error-messages';

const ADMIN: UsuarioOut = { id_usuario: 1, username: 'admin', email: 'admin@lizar.pe', nombre: 'Admin Prueba', rol: 'admin', estado: true, debe_cambiar_clave: false };
const ANA: UsuarioOut = { id_usuario: 2, username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'logistica', estado: true, debe_cambiar_clave: true };
const LUIS: UsuarioOut = { id_usuario: 3, username: 'lperez', email: 'luis@lizar.pe', nombre: 'Luis Pérez', rol: 'logistica', estado: false, debe_cambiar_clave: false };

const http = (status: number, detail: unknown) => throwError(() => new HttpErrorResponse({ status, error: { detail } }));

/** jsdom no implementa showModal/close de <dialog>: se simulan con el atributo open. */
beforeEach(() => {
  const proto = HTMLDialogElement.prototype as unknown as { showModal: () => void; close: () => void };
  proto.showModal = function (this: HTMLDialogElement) {
    this.setAttribute('open', '');
  };
  proto.close = function (this: HTMLDialogElement) {
    this.removeAttribute('open');
  };
});

interface Opciones {
  /** false: no esperar a que la carga termine (para ver el esqueleto) */
  esperar?: boolean;
  lista?: () => Observable<UsuarioOut[]>;
  crear?: (c: unknown) => Observable<UsuarioOut>;
  editar?: (id: number, c: unknown) => Observable<UsuarioOut>;
  desactivar?: (id: number) => Observable<UsuarioOut>;
  activar?: (id: number) => Observable<UsuarioOut>;
}

async function montar(op: Opciones = {}) {
  const servicio = {
    listar: vi.fn(op.lista ?? (() => of([ADMIN, ANA, LUIS]))),
    crear: vi.fn(op.crear ?? ((c: Record<string, unknown>) => of({ ...ANA, id_usuario: 4, ...c, estado: true, debe_cambiar_clave: true } as UsuarioOut))),
    editar: vi.fn(op.editar ?? ((id: number, c: Record<string, unknown>) => of({ ...ANA, id_usuario: id, ...c } as UsuarioOut))),
    desactivar: vi.fn(op.desactivar ?? ((id: number) => of({ ...ANA, id_usuario: id, estado: false }))),
    activar: vi.fn(op.activar ?? ((id: number) => of({ ...LUIS, id_usuario: id, estado: true }))),
  };
  TestBed.configureTestingModule({
    providers: [
      { provide: UsuariosService, useValue: servicio },
      { provide: AuthService, useValue: { usuario: signal(ADMIN) } },
    ],
  });
  const f = TestBed.createComponent(UsuariosComponent);
  const refrescar = async () => {
    f.detectChanges();
    await f.whenStable();
    f.detectChanges();
  };
  if (op.esperar === false) f.detectChanges();
  else await refrescar();
  const el = f.nativeElement as HTMLElement;
  const boton = (texto: string, dentro: ParentNode = el) =>
    [...dentro.querySelectorAll('button')].find(b => b.textContent?.trim().includes(texto) || b.getAttribute('aria-label')?.includes(texto)) as HTMLButtonElement;
  const filas = () => [...el.querySelectorAll<HTMLTableRowElement>('tbody tr:not(.usuarios__confirmacion)')];
  const fila = (nombre: string) => filas().find(r => r.textContent?.includes(nombre))!;
  const campo = (id: string) => el.querySelector(`#${id}`) as HTMLInputElement & HTMLSelectElement;
  const escribir = async (id: string, valor: string) => {
    const c = campo(id);
    c.value = valor;
    c.dispatchEvent(new Event(c.tagName === 'SELECT' ? 'change' : 'input'));
    await refrescar();
  };
  const enviar = async () => {
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    await refrescar();
  };
  const errores = () => [...el.querySelectorAll('.hoja .campo__error')].map(e => e.textContent?.trim());
  const hoja = () => el.querySelector('dialog.hoja') as HTMLDialogElement;
  const clic = async (b: HTMLButtonElement) => {
    b.click();
    await refrescar();
  };
  return { f, el, clic, servicio, boton, filas, fila, campo, escribir, enviar, errores, hoja, refrescar, toast: TestBed.inject(ToastService) };
}

describe('UsuariosComponent: lista', () => {
  it('muestra a todos los usuarios, activos e inactivos, con el rol en palabras y su estado', async () => {
    const { el, filas, fila } = await montar();
    expect(el.querySelector('h1')?.textContent).toContain('Usuarios');
    expect(filas()).toHaveLength(3);
    expect(fila('Ana Quispe').textContent).toContain('aquispe');
    expect(fila('Ana Quispe').textContent).toContain('ana@lizar.pe');
    expect(fila('Ana Quispe').textContent).toContain('Encargado de Logística');
    expect(fila('Admin Prueba').textContent).toContain('Administrador del Sistema');
    expect(fila('Ana Quispe').textContent).toContain('Activo');
    expect(fila('Luis Pérez').textContent).toContain('Inactivo');
    expect(el.textContent).toContain('3 usuarios');
  });

  it('señala al usuario propio y a quien debe cambiar su contraseña', async () => {
    const { fila } = await montar();
    expect(fila('Admin Prueba').textContent).toContain('Tú');
    expect(fila('Ana Quispe').textContent).toContain('Debe cambiar su contraseña');
    expect(fila('Luis Pérez').textContent).not.toContain('Debe cambiar su contraseña');
  });

  it('mientras carga muestra un esqueleto', async () => {
    const { el } = await montar({ lista: () => new Subject<UsuarioOut[]>(), esperar: false });
    expect(el.querySelector('[aria-busy="true"]')).toBeTruthy();
    expect(el.querySelector('table')).toBeNull();
  });

  it('si no carga, lo dice y permite reintentar', async () => {
    let intento = 0;
    const { clic, el, servicio, boton, refrescar, filas } = await montar({ lista: () => (++intento === 1 ? http(0, null) : of([ADMIN])) });
    expect(el.textContent).toContain('No se pudo cargar la lista de usuarios.');
    expect(el.querySelector('table')).toBeNull();
    await clic(boton('Reintentar'));
    await refrescar();
    expect(servicio.listar).toHaveBeenCalledTimes(2);
    expect(filas()).toHaveLength(1);
  });

  it('busca por nombre, usuario o correo, sin importar tildes', async () => {
    const { el, filas, escribir } = await montar();
    await escribir('usuarios-buscar', 'perez');
    expect(filas().map(r => r.textContent)).toEqual([expect.stringContaining('Luis Pérez')]);
    await escribir('usuarios-buscar', 'ana@lizar');
    expect(filas()).toHaveLength(1);
    await escribir('usuarios-buscar', 'zzz');
    expect(el.textContent).toContain('Ningún usuario coincide con la búsqueda.');
  });
});

describe('UsuariosComponent: crear', () => {
  it('«Nuevo usuario» abre el formulario vacío con el rol de Logística', async () => {
    const { clic, boton, hoja, campo } = await montar();
    await clic(boton('Nuevo usuario'));
    expect(hoja().hasAttribute('open')).toBe(true);
    expect(campo('usuario-username').value).toBe('');
    expect(campo('usuario-rol').value).toBe('logistica');
    expect(campo('usuario-username').disabled).toBe(false);
  });

  it('no envía y marca todo lo que falta o está mal, igual que el servidor', async () => {
    const { clic, boton, enviar, errores, servicio, escribir } = await montar();
    await clic(boton('Nuevo usuario'));
    await enviar();
    expect(servicio.crear).not.toHaveBeenCalled();
    expect(errores()).toEqual(['Escribe el nombre.', 'Escribe el usuario.', 'Escribe el correo.', 'Escribe la contraseña.']);
    await escribir('usuario-username', 'ana quispe');
    await escribir('usuario-email', 'sin-arroba');
    await escribir('usuario-nombre', 'Ana');
    await escribir('usuario-password', '1234567');
    await enviar();
    expect(errores()).toEqual(['El usuario no puede tener espacios.', 'Escribe un correo válido.', 'Usa al menos 8 caracteres.']);
  });

  it('crea el usuario, lo agrega a la lista, cierra el formulario y avisa', async () => {
    const { clic, boton, enviar, escribir, servicio, hoja, filas, fila, toast } = await montar();
    await clic(boton('Nuevo usuario'));
    await escribir('usuario-username', ' mtorres ');
    await escribir('usuario-email', 'maria@lizar.pe');
    await escribir('usuario-nombre', 'María Torres');
    await escribir('usuario-rol', 'admin');
    await escribir('usuario-password', 'clave-segura-1');
    await enviar();
    expect(servicio.crear).toHaveBeenCalledWith({
      username: 'mtorres', email: 'maria@lizar.pe', nombre: 'María Torres', rol: 'admin', password: 'clave-segura-1',
    });
    expect(hoja().hasAttribute('open')).toBe(false);
    expect(filas()).toHaveLength(4);
    expect(fila('María Torres').textContent).toContain('Debe cambiar su contraseña');
    expect(toast.toasts().map(t => t.title)).toEqual(['Usuario creado.']);
  });

  it('un usuario repetido (409) se marca en su campo y deja el formulario abierto con lo escrito', async () => {
    const { clic, boton, enviar, escribir, errores, hoja, campo } = await montar({ crear: () => http(409, "El usuario 'mtorres' ya existe") });
    await clic(boton('Nuevo usuario'));
    await escribir('usuario-username', 'mtorres');
    await escribir('usuario-email', 'maria@lizar.pe');
    await escribir('usuario-nombre', 'María Torres');
    await escribir('usuario-password', 'clave-segura-1');
    await enviar();
    expect(errores()).toEqual(["El usuario 'mtorres' ya existe"]);
    expect(hoja().hasAttribute('open')).toBe(true);
    expect(campo('usuario-nombre').value).toBe('María Torres');
    expect(campo('usuario-username').getAttribute('aria-invalid')).toBe('true');
  });

  it('un fallo de red se muestra dentro del formulario, no queda tapado por él', async () => {
    const { clic, boton, enviar, escribir, el } = await montar({ crear: () => http(0, null) });
    await clic(boton('Nuevo usuario'));
    await escribir('usuario-username', 'mtorres');
    await escribir('usuario-email', 'maria@lizar.pe');
    await escribir('usuario-nombre', 'María Torres');
    await escribir('usuario-password', 'clave-segura-1');
    await enviar();
    expect(el.querySelector('.hoja .alerta--error')?.textContent).toContain(MSG_ERROR_RED);
  });

  it('al corregir un campo con error se le quita el error', async () => {
    const { clic, boton, enviar, escribir, errores } = await montar();
    await clic(boton('Nuevo usuario'));
    await enviar();
    await escribir('usuario-username', 'mtorres');
    expect(errores()).toEqual(['Escribe el nombre.', 'Escribe el correo.', 'Escribe la contraseña.']);
  });

  it('el ojo muestra u oculta la contraseña', async () => {
    const { clic, boton, campo, refrescar } = await montar();
    await clic(boton('Nuevo usuario'));
    expect(campo('usuario-password').type).toBe('password');
    await clic(boton('Mostrar contraseña'));
    await refrescar();
    expect(campo('usuario-password').type).toBe('text');
  });
});

describe('UsuariosComponent: editar', () => {
  it('abre con los datos del usuario; el usuario no se puede cambiar', async () => {
    const { clic, fila, boton, campo, hoja } = await montar();
    await clic(boton('Editar', fila('Ana Quispe')));
    expect(hoja().hasAttribute('open')).toBe(true);
    expect(campo('usuario-nombre').value).toBe('Ana Quispe');
    expect(campo('usuario-email').value).toBe('ana@lizar.pe');
    expect(campo('usuario-rol').value).toBe('logistica');
    expect(campo('usuario-username').value).toBe('aquispe');
    expect(campo('usuario-username').disabled).toBe(true);
    expect(campo('usuario-password').value).toBe('');
  });

  it('sin cambios no se puede guardar', async () => {
    const { clic, fila, boton, hoja } = await montar();
    await clic(boton('Editar', fila('Ana Quispe')));
    expect((hoja().querySelector('button[type="submit"]') as HTMLButtonElement).disabled).toBe(true);
  });

  it('envía solo lo que cambió y actualiza la fila', async () => {
    const { clic, fila, boton, escribir, enviar, servicio, hoja, toast } = await montar();
    await clic(boton('Editar', fila('Ana Quispe')));
    await escribir('usuario-nombre', 'Ana María Quispe');
    await enviar();
    expect(servicio.editar).toHaveBeenCalledWith(2, { nombre: 'Ana María Quispe' });
    expect(hoja().hasAttribute('open')).toBe(false);
    expect(fila('Ana María Quispe')).toBeTruthy();
    expect(toast.toasts().map(t => t.title)).toEqual(['Cambios guardados.']);
  });

  it('restablecer la contraseña avisa que deberá cambiarla y la envía', async () => {
    const { clic, fila, boton, escribir, enviar, servicio, el } = await montar();
    await clic(boton('Editar', fila('Luis Pérez')));
    expect(el.querySelector('.hoja')?.textContent).toContain('deberá cambiarla al ingresar');
    await escribir('usuario-password', 'otra-clave-123');
    await enviar();
    expect(servicio.editar).toHaveBeenCalledWith(3, { password: 'otra-clave-123' });
  });

  it('una contraseña corta al editar se rechaza', async () => {
    const { clic, fila, boton, escribir, enviar, errores, servicio } = await montar();
    await clic(boton('Editar', fila('Ana Quispe')));
    await escribir('usuario-password', 'corta');
    await enviar();
    expect(errores()).toEqual(['Usa al menos 8 caracteres.']);
    expect(servicio.editar).not.toHaveBeenCalled();
  });

  it('degradar al último administrador (409) se explica dentro del formulario', async () => {
    const { clic, fila, boton, escribir, enviar, el, hoja } = await montar({ editar: () => http(409, 'Debe quedar al menos un administrador activo') });
    await clic(boton('Editar', fila('Ana Quispe')));
    await escribir('usuario-rol', 'admin');
    await enviar();
    expect(el.querySelector('.hoja .alerta--error')?.textContent).toContain('Debe quedar al menos un administrador activo');
    expect(hoja().hasAttribute('open')).toBe(true);
  });

  it('al editarse a sí mismo no puede cambiar su rol ni su contraseña desde aquí', async () => {
    const { clic, fila, boton, campo, el } = await montar();
    await clic(boton('Editar', fila('Admin Prueba')));
    expect(campo('usuario-rol').disabled).toBe(true);
    expect(el.querySelector('#usuario-password')).toBeNull();
    expect(el.querySelector('.hoja')?.textContent).toContain('Cambiar contraseña');
  });
});

describe('UsuariosComponent: desactivar y activar', () => {
  it('a uno mismo no se ofrece desactivar', async () => {
    const { fila, boton } = await montar();
    expect(boton('Desactivar', fila('Admin Prueba'))).toBeUndefined();
    expect(boton('Editar', fila('Admin Prueba'))).toBeTruthy();
  });

  it('pide confirmación en la misma fila y no hace nada hasta confirmar', async () => {
    const { clic, el, fila, boton, servicio, refrescar } = await montar();
    await clic(boton('Desactivar', fila('Ana Quispe')));
    await refrescar();
    const confirmacion = el.querySelector('.usuarios__confirmacion')!;
    expect(confirmacion.textContent).toContain('Ana Quispe');
    expect(confirmacion.textContent).toContain('Ya no podrá ingresar');
    expect(servicio.desactivar).not.toHaveBeenCalled();
  });

  it('cancelar cierra la confirmación sin cambiar nada', async () => {
    const { clic, el, fila, boton, servicio, refrescar } = await montar();
    await clic(boton('Desactivar', fila('Ana Quispe')));
    await refrescar();
    await clic(boton('Cancelar', el.querySelector('.usuarios__confirmacion')!));
    await refrescar();
    expect(el.querySelector('.usuarios__confirmacion')).toBeNull();
    expect(servicio.desactivar).not.toHaveBeenCalled();
  });

  it('confirmar desactiva, cambia el estado de la fila y ofrece activar', async () => {
    const { clic, el, fila, boton, servicio, refrescar, toast } = await montar();
    await clic(boton('Desactivar', fila('Ana Quispe')));
    await refrescar();
    await clic(boton('Sí, desactivar', el.querySelector('.usuarios__confirmacion')!));
    await refrescar();
    expect(servicio.desactivar).toHaveBeenCalledWith(2);
    expect(el.querySelector('.usuarios__confirmacion')).toBeNull();
    expect(fila('Ana Quispe').textContent).toContain('Inactivo');
    expect(boton('Activar', fila('Ana Quispe'))).toBeTruthy();
    expect(toast.toasts().map(t => t.title)).toEqual(['Usuario desactivado.']);
  });

  it('activar también pide confirmación', async () => {
    const { clic, el, fila, boton, servicio, refrescar, toast } = await montar();
    await clic(boton('Activar', fila('Luis Pérez')));
    await refrescar();
    expect(servicio.activar).not.toHaveBeenCalled();
    expect(el.querySelector('.usuarios__confirmacion')?.textContent).toContain('Podrá ingresar de nuevo');
    await clic(boton('Sí, activar', el.querySelector('.usuarios__confirmacion')!));
    await refrescar();
    expect(servicio.activar).toHaveBeenCalledWith(3);
    expect(fila('Luis Pérez').textContent).toContain('Activo');
    expect(toast.toasts().map(t => t.title)).toEqual(['Usuario activado.']);
  });

  it('si el servidor lo rechaza (último administrador) explica el motivo en la misma fila', async () => {
    const { clic, el, fila, boton, refrescar } = await montar({ desactivar: () => http(409, 'Debe quedar al menos un administrador activo') });
    await clic(boton('Desactivar', fila('Ana Quispe')));
    await refrescar();
    await clic(boton('Sí, desactivar', el.querySelector('.usuarios__confirmacion')!));
    await refrescar();
    expect(el.querySelector('.usuarios__confirmacion')?.textContent).toContain('Debe quedar al menos un administrador activo');
    expect(fila('Ana Quispe').textContent).toContain('Activo');
  });

  it('un fallo de red se dice con el texto de siempre y permite volver a intentar', async () => {
    const { clic, el, fila, boton, refrescar } = await montar({ desactivar: () => http(0, null) });
    await clic(boton('Desactivar', fila('Ana Quispe')));
    await refrescar();
    await clic(boton('Sí, desactivar', el.querySelector('.usuarios__confirmacion')!));
    await refrescar();
    expect(el.querySelector('.usuarios__confirmacion')?.textContent).toContain(MSG_ERROR_RED);
    expect(boton('Sí, desactivar', el.querySelector('.usuarios__confirmacion')!).disabled).toBe(false);
  });

  it('solo una confirmación a la vez: abrir otra cierra la anterior', async () => {
    const { clic, el, fila, boton, refrescar } = await montar();
    await clic(boton('Desactivar', fila('Ana Quispe')));
    await refrescar();
    await clic(boton('Activar', fila('Luis Pérez')));
    await refrescar();
    expect(el.querySelectorAll('.usuarios__confirmacion')).toHaveLength(1);
    expect(el.querySelector('.usuarios__confirmacion')?.textContent).toContain('Luis Pérez');
  });
});
