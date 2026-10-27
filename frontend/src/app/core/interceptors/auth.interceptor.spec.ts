import { beforeEach, describe, expect, it } from 'vitest';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Router, provideRouter } from '@angular/router';
import { authInterceptor } from './auth.interceptor';
import { AuthService } from '../services/auth.service';
import { ToastService } from '../services/toast.service';

@Component({ template: '' })
class Vacio {}

const ANA = { id_usuario: 2, username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'logistica', estado: true };
const GASTOS = 'http://localhost:8081/gastos';
const MAESTROS = 'http://localhost:8081/api/v1/clientes';

function montar(conSesion = true) {
  localStorage.clear();
  if (conSesion) {
    localStorage.setItem('lizar_token', 'tok-123');
    localStorage.setItem('lizar_user', JSON.stringify(ANA));
  }
  TestBed.configureTestingModule({
    providers: [
      provideRouter([{ path: '**', component: Vacio }]),
      provideHttpClient(withInterceptors([authInterceptor])),
      provideHttpClientTesting(),
    ],
  });
  return {
    http: TestBed.inject(HttpClient),
    ctrl: TestBed.inject(HttpTestingController),
    auth: TestBed.inject(AuthService),
    toast: TestBed.inject(ToastService),
    router: TestBed.inject(Router),
  };
}

const esperar = () => new Promise(r => setTimeout(r));

describe('authInterceptor', () => {
  beforeEach(() => localStorage.clear());

  it('añade el Bearer a las peticiones del servicio de gastos y a las de /api/v1', () => {
    const { http, ctrl } = montar();
    http.get(GASTOS).subscribe();
    http.get(MAESTROS).subscribe();
    expect(ctrl.expectOne(GASTOS).request.headers.get('Authorization')).toBe('Bearer tok-123');
    expect(ctrl.expectOne(MAESTROS).request.headers.get('Authorization')).toBe('Bearer tok-123');
  });

  it('no envía el token a otro sitio ni sin sesión', () => {
    const { http, ctrl } = montar();
    http.get('https://otro.sitio/datos').subscribe();
    expect(ctrl.expectOne('https://otro.sitio/datos').request.headers.has('Authorization')).toBe(false);
    TestBed.resetTestingModule();
    const sinSesion = montar(false);
    sinSesion.http.get(GASTOS).subscribe({ error: () => {} });
    expect(sinSesion.ctrl.expectOne(GASTOS).request.headers.has('Authorization')).toBe(false);
  });

  it('la salud y el login siguen funcionando: el login no lleva Bearer porque aún no hay sesión', () => {
    const { http, ctrl } = montar(false);
    http.get('http://localhost:8081/health').subscribe();
    http.post('http://localhost:8081/api/v1/auth/login', {}).subscribe();
    expect(ctrl.expectOne('http://localhost:8081/health').request.headers.has('Authorization')).toBe(false);
    expect(ctrl.expectOne('http://localhost:8081/api/v1/auth/login').request.headers.has('Authorization')).toBe(false);
  });

  it('un 401 cierra la sesión, avisa y lleva al login conservando la pantalla', async () => {
    const { http, ctrl, auth, toast, router } = montar();
    await router.navigateByUrl('/app/gastos?estado=por-aprobar');
    let fallo: unknown;
    http.get(GASTOS).subscribe({ error: e => (fallo = e) });
    ctrl.expectOne(GASTOS).flush({ detail: 'Token inválido' }, { status: 401, statusText: 'Unauthorized' });
    await esperar();
    expect(fallo).toBeTruthy();
    expect(auth.isAuth()).toBe(false);
    expect(router.url).toBe('/login?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
    expect(toast.toasts().map(t => t.title)).toEqual(['Tu sesión terminó. Ingresa de nuevo.']);
  });

  it('varios 401 seguidos avisan una sola vez', async () => {
    const { http, ctrl, toast, router } = montar();
    await router.navigateByUrl('/app/gastos');
    http.get(GASTOS).subscribe({ error: () => {} });
    http.get(`${GASTOS}/1`).subscribe({ error: () => {} });
    ctrl.expectOne(GASTOS).flush({}, { status: 401, statusText: 'Unauthorized' });
    ctrl.expectOne(`${GASTOS}/1`).flush({}, { status: 401, statusText: 'Unauthorized' });
    expect(toast.toasts()).toHaveLength(1);
  });

  it('un 401 del login (clave equivocada) no cierra nada: lo muestra la pantalla de login', async () => {
    const { http, ctrl, toast, router, auth } = montar(false);
    await router.navigateByUrl('/login');
    let fallo: { status?: number } | undefined;
    http.post('http://localhost:8081/api/v1/auth/login', {}).subscribe({ error: e => (fallo = e) });
    ctrl.expectOne('http://localhost:8081/api/v1/auth/login').flush({ detail: 'Credenciales incorrectas' }, { status: 401, statusText: 'Unauthorized' });
    expect(fallo?.status).toBe(401);
    expect(toast.toasts()).toHaveLength(0);
    expect(router.url).toBe('/login');
    expect(auth.isAuth()).toBe(false);
  });

  it('el aviso de salida al servidor no muestra avisos si falla (token vencido o sin permiso): la sesión ya se cerró', async () => {
    for (const status of [401, 403, 500]) {
      TestBed.resetTestingModule();
      const { ctrl, auth, toast, router } = montar();
      await router.navigateByUrl('/app/gastos');
      auth.logout();
      const req = ctrl.expectOne('http://localhost:8081/api/v1/auth/logout');
      expect(req.request.headers.get('Authorization')).toBe('Bearer tok-123');
      req.flush({ detail: 'x' }, { status, statusText: 'Error' });
      await esperar();
      expect(toast.toasts()).toHaveLength(0);
      expect(router.url).toBe('/login');
    }
  });

  it('un 403 avisa en palabras sencillas y mantiene la sesión', async () => {
    const { http, ctrl, toast, auth, router } = montar();
    await router.navigateByUrl('/app/gastos');
    let fallo: { status?: number } | undefined;
    http.post(GASTOS, {}).subscribe({ error: e => (fallo = e) });
    ctrl.expectOne(GASTOS).flush({ error: 'sin permiso' }, { status: 403, statusText: 'Forbidden' });
    expect(fallo?.status).toBe(403);
    expect(auth.isAuth()).toBe(true);
    expect(router.url).toBe('/app/gastos');
    expect(toast.toasts()).toHaveLength(1);
    expect(toast.toasts()[0]).toMatchObject({ title: 'Tu usuario no tiene permiso para hacer esto.', type: 'err' });
  });

  it('un 403 «Debes cambiar tu contraseña» lleva a cambiarla, sin el aviso de falta de permiso', async () => {
    const { http, ctrl, toast, auth, router } = montar();
    await router.navigateByUrl('/app/gastos?estado=por-aprobar');
    let fallo: { status?: number } | undefined;
    http.get(GASTOS).subscribe({ error: e => (fallo = e) });
    ctrl.expectOne(GASTOS).flush({ detail: 'Debes cambiar tu contraseña' }, { status: 403, statusText: 'Forbidden' });
    await esperar();
    expect(fallo?.status).toBe(403);
    expect(toast.toasts()).toHaveLength(0);
    expect(auth.isAuth()).toBe(true);
    expect(auth.debeCambiarClave()).toBe(true);
    expect(router.url).toBe('/cambiar-clave?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
  });

  it('varios 403 de clave pendiente seguidos llevan una sola vez a cambiarla', async () => {
    const { http, ctrl, router } = montar();
    await router.navigateByUrl('/app/gastos');
    http.get(GASTOS).subscribe({ error: () => {} });
    http.get(`${GASTOS}/1`).subscribe({ error: () => {} });
    ctrl.expectOne(GASTOS).flush({ detail: 'Debes cambiar tu contraseña' }, { status: 403, statusText: 'Forbidden' });
    await esperar();
    ctrl.expectOne(`${GASTOS}/1`).flush({ detail: 'Debes cambiar tu contraseña' }, { status: 403, statusText: 'Forbidden' });
    await esperar();
    expect(router.url).toBe('/cambiar-clave?volver=%2Fapp%2Fgastos');
  });

  it('otros errores pasan sin tocar la sesión ni mostrar avisos', () => {
    const { http, ctrl, toast, auth } = montar();
    http.get(GASTOS).subscribe({ error: () => {} });
    ctrl.expectOne(GASTOS).flush({}, { status: 500, statusText: 'Server Error' });
    expect(toast.toasts()).toHaveLength(0);
    expect(auth.isAuth()).toBe(true);
  });

  it('un 401 de otro sitio no cierra la sesión', () => {
    const { http, ctrl, auth } = montar();
    http.get('https://otro.sitio/datos').subscribe({ error: () => {} });
    ctrl.expectOne('https://otro.sitio/datos').flush({}, { status: 401, statusText: 'Unauthorized' });
    expect(auth.isAuth()).toBe(true);
  });
});
