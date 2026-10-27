import { beforeEach, describe, expect, it } from 'vitest';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Router, provideRouter } from '@angular/router';
import { AuthService } from './auth.service';

@Component({ template: '' })
class Vacio {}

const ANA = { id_usuario: 2, username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'logistica', estado: true };

function montar(guardado?: object) {
  localStorage.clear();
  if (guardado) {
    localStorage.setItem('lizar_token', 'tok');
    localStorage.setItem('lizar_user', JSON.stringify(guardado));
  }
  TestBed.configureTestingModule({
    providers: [provideRouter([{ path: '**', component: Vacio }]), provideHttpClient(), provideHttpClientTesting()],
  });
  return { auth: TestBed.inject(AuthService), http: TestBed.inject(HttpTestingController), router: TestBed.inject(Router) };
}

describe('AuthService', () => {
  beforeEach(() => localStorage.clear());

  it('distingue al Encargado de Logística del administrador', () => {
    expect(montar(ANA).auth.esLogistica()).toBe(true);
    TestBed.resetTestingModule();
    const { auth } = montar({ ...ANA, rol: 'admin' });
    expect(auth.esLogistica()).toBe(false);
    expect(auth.isAdmin()).toBe(true);
  });

  it('una sesión guardada con el rol viejo «operador» pasa a «logistica»', () => {
    const { auth } = montar({ ...ANA, rol: 'operador' });
    expect(auth.usuario()?.rol).toBe('logistica');
    expect(auth.esLogistica()).toBe(true);
  });

  it('al iniciar sesión guarda el token y el usuario', () => {
    const { auth, http } = montar();
    auth.login({ username: 'aquispe', password: 'x' }).subscribe();
    http.expectOne('http://localhost:8081/api/v1/auth/login').flush({ access_token: 'abc', token_type: 'bearer', usuario: ANA });
    expect(auth.token()).toBe('abc');
    expect(auth.isAuth()).toBe(true);
    expect(localStorage.getItem('lizar_token')).toBe('abc');
  });

  it('sesionVencida cierra la sesión y lleva al login recordando la pantalla', async () => {
    const { auth, router } = montar(ANA);
    await router.navigateByUrl('/app/gastos?estado=por-aprobar');
    auth.sesionVencida();
    await new Promise(r => setTimeout(r));
    expect(auth.isAuth()).toBe(false);
    expect(localStorage.getItem('lizar_token')).toBeNull();
    expect(router.url).toBe('/login?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
  });

  it('sesionVencida desde una pantalla pública no agrega «volver»', async () => {
    const { auth, router } = montar(ANA);
    await router.navigateByUrl('/');
    auth.sesionVencida();
    await new Promise(r => setTimeout(r));
    expect(router.url).toBe('/login');
  });

  it('refrescar trae el usuario real (con su rol actual) desde /auth/me', () => {
    const { auth, http } = montar({ ...ANA, rol: 'admin' });
    auth.refrescar();
    http.expectOne('http://localhost:8081/api/v1/auth/me').flush(ANA);
    expect(auth.usuario()?.rol).toBe('logistica');
    expect(JSON.parse(localStorage.getItem('lizar_user')!).rol).toBe('logistica');
  });

  it('refrescar sin sesión no pide nada', () => {
    const { auth, http } = montar();
    auth.refrescar();
    http.expectNone('http://localhost:8081/api/v1/auth/me');
  });

  it('al iniciar sesión guarda si debe cambiar la contraseña, venga en el usuario o en la raíz', () => {
    const { auth, http } = montar();
    auth.login({ username: 'aquispe', password: 'x' }).subscribe();
    http.expectOne('http://localhost:8081/api/v1/auth/login').flush({
      access_token: 'abc', token_type: 'bearer', usuario: { ...ANA, debe_cambiar_clave: true }, debe_cambiar_clave: true,
    });
    expect(auth.debeCambiarClave()).toBe(true);
    expect(JSON.parse(localStorage.getItem('lizar_user')!).debe_cambiar_clave).toBe(true);

    TestBed.resetTestingModule();
    const otro = montar();
    otro.auth.login({ username: 'aquispe', password: 'x' }).subscribe();
    otro.http.expectOne('http://localhost:8081/api/v1/auth/login').flush({ access_token: 'abc', token_type: 'bearer', usuario: ANA, debe_cambiar_clave: true });
    expect(otro.auth.debeCambiarClave()).toBe(true);
  });

  it('una sesión sin la marca (guardada antes) no debe cambiar la contraseña', () => {
    expect(montar(ANA).auth.debeCambiarClave()).toBe(false);
  });

  it('cambiarClave envía las dos contraseñas y guarda al usuario ya sin la marca', () => {
    const { auth, http } = montar({ ...ANA, debe_cambiar_clave: true });
    expect(auth.debeCambiarClave()).toBe(true);
    let salida: unknown;
    auth.cambiarClave({ clave_actual: 'vieja-clave', clave_nueva: 'nueva-clave-1' }).subscribe(u => (salida = u));
    const req = http.expectOne('http://localhost:8081/api/v1/auth/cambiar-clave');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ clave_actual: 'vieja-clave', clave_nueva: 'nueva-clave-1' });
    req.flush({ ...ANA, debe_cambiar_clave: false });
    expect(salida).toMatchObject({ username: 'aquispe' });
    expect(auth.debeCambiarClave()).toBe(false);
    expect(JSON.parse(localStorage.getItem('lizar_user')!).debe_cambiar_clave).toBe(false);
  });

  it('cambiarClave con la contraseña actual equivocada no toca la sesión', () => {
    const { auth, http } = montar({ ...ANA, debe_cambiar_clave: true });
    let fallo: { status?: number } | undefined;
    auth.cambiarClave({ clave_actual: 'mal', clave_nueva: 'nueva-clave-1' }).subscribe({ error: e => (fallo = e) });
    http.expectOne('http://localhost:8081/api/v1/auth/cambiar-clave').flush({ detail: 'La contraseña actual no es correcta' }, { status: 400, statusText: 'Bad Request' });
    expect(fallo?.status).toBe(400);
    expect(auth.isAuth()).toBe(true);
    expect(auth.debeCambiarClave()).toBe(true);
  });

  it('refrescar lleva a cambiar la contraseña si el servidor dice que está pendiente', async () => {
    const { auth, http, router } = montar(ANA);
    await router.navigateByUrl('/app/gastos');
    auth.refrescar();
    http.expectOne('http://localhost:8081/api/v1/auth/me').flush({ ...ANA, debe_cambiar_clave: true });
    await new Promise(r => setTimeout(r));
    expect(auth.debeCambiarClave()).toBe(true);
    expect(router.url).toBe('/cambiar-clave?volver=%2Fapp%2Fgastos');
  });

  it('refrescar con la contraseña al día no cambia de pantalla', async () => {
    const { auth, http, router } = montar({ ...ANA, debe_cambiar_clave: true });
    await router.navigateByUrl('/app/gastos');
    auth.refrescar();
    http.expectOne('http://localhost:8081/api/v1/auth/me').flush({ ...ANA, debe_cambiar_clave: false });
    await new Promise(r => setTimeout(r));
    expect(auth.debeCambiarClave()).toBe(false);
    expect(router.url).toBe('/app/gastos');
  });

  it('exigirCambioClave marca la sesión y no repite la navegación si ya está en esa pantalla', async () => {
    const { auth, router } = montar(ANA);
    await router.navigateByUrl('/app/inicio');
    auth.exigirCambioClave();
    await new Promise(r => setTimeout(r));
    expect(auth.debeCambiarClave()).toBe(true);
    expect(router.url).toBe('/cambiar-clave?volver=%2Fapp%2Finicio');
    auth.exigirCambioClave();
    await new Promise(r => setTimeout(r));
    expect(router.url).toBe('/cambiar-clave?volver=%2Fapp%2Finicio');
  });

  it('logout avisa al servidor con el token de la sesión y cierra la local sin esperar la respuesta', async () => {
    const { auth, http, router } = montar(ANA);
    await router.navigateByUrl('/app/inicio');
    auth.logout();
    const req = http.expectOne('http://localhost:8081/api/v1/auth/logout');
    expect(req.request.method).toBe('POST');
    expect(req.request.headers.get('Authorization')).toBe('Bearer tok');
    // El servidor todavía no contestó y la sesión ya está cerrada
    expect(auth.isAuth()).toBe(false);
    expect(localStorage.getItem('lizar_token')).toBeNull();
    expect(localStorage.getItem('lizar_user')).toBeNull();
    await new Promise(r => setTimeout(r));
    expect(router.url).toBe('/login');
    req.flush({ detail: 'Sesión cerrada' });
  });

  it('logout cierra la sesión aunque el servidor no responda o devuelva error', async () => {
    for (const falla of [
      (r: { error: (e: ProgressEvent) => void }) => r.error(new ProgressEvent('error')),
      (r: { flush: (b: unknown, o: object) => void }) => r.flush({ error: 'bitácora' }, { status: 500, statusText: 'Server Error' }),
      (r: { flush: (b: unknown, o: object) => void }) => r.flush({ detail: 'Token inválido' }, { status: 401, statusText: 'Unauthorized' }),
    ]) {
      TestBed.resetTestingModule();
      const { auth, http, router } = montar(ANA);
      await router.navigateByUrl('/app/inicio');
      auth.logout();
      falla(http.expectOne('http://localhost:8081/api/v1/auth/logout') as never);
      await new Promise(r => setTimeout(r));
      expect(auth.isAuth()).toBe(false);
      expect(router.url).toBe('/login');
    }
  });

  it('logout sin sesión no llama al servidor', async () => {
    const { auth, http, router } = montar();
    auth.logout();
    http.expectNone('http://localhost:8081/api/v1/auth/logout');
    await new Promise(r => setTimeout(r));
    expect(router.url).toBe('/login');
  });
});
