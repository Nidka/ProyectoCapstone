import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';
import { Observable, of } from 'rxjs';
import { LoginComponent } from './login.component';
import { AuthService } from '../../core/services/auth.service';

async function abrir(url: string, login: () => Observable<unknown> = vi.fn(() => of({}))) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([{ path: 'login', component: LoginComponent }]),
      { provide: AuthService, useValue: { login } },
    ],
  });
  const harness = await RouterTestingHarness.create();
  await harness.navigateByUrl(url, LoginComponent);
  return { harness, el: harness.routeNativeElement as HTMLElement, login };
}

function escribir(el: HTMLElement, id: string, valor: string): void {
  const campo = el.querySelector(`#${id}`) as HTMLInputElement;
  campo.value = valor;
  campo.dispatchEvent(new Event('input'));
}

describe('LoginComponent', () => {
  it('pide usuario y contraseña', async () => {
    const { harness, el } = await abrir('/login');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    harness.detectChanges();
    expect(el.querySelector('.alerta--error')?.textContent?.trim()).toBe('Ingresa usuario y contraseña');
  });

  it('tras iniciar sesión vuelve a la pantalla que se pidió', async () => {
    const { el, login } = await abrir('/login?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
    const navegar = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
    escribir(el, 'username', 'admin');
    escribir(el, 'password', 'admin123');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    expect(login).toHaveBeenCalledWith({ username: 'admin', password: 'admin123' });
    expect(navegar).toHaveBeenCalledWith('/app/gastos?estado=por-aprobar');
  });

  it('si la contraseña está pendiente de cambio lleva a cambiarla, recordando a dónde iba', async () => {
    const pendiente = vi.fn(() => of({ access_token: 't', token_type: 'bearer', usuario: { debe_cambiar_clave: true } }));
    const { el } = await abrir('/login?volver=%2Fapp%2Fgastos', pendiente);
    const router = TestBed.inject(Router);
    const navegar = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    const navegarUrl = vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    escribir(el, 'username', 'aquispe');
    escribir(el, 'password', 'clave-inicial');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    expect(navegar).toHaveBeenCalledWith(['/cambiar-clave'], { queryParams: { volver: '/app/gastos' } });
    expect(navegarUrl).not.toHaveBeenCalled();
  });

  it('la marca también se reconoce si llega solo en la raíz de la respuesta', async () => {
    const pendiente = vi.fn(() => of({ access_token: 't', token_type: 'bearer', usuario: {}, debe_cambiar_clave: true }));
    const { el } = await abrir('/login', pendiente);
    const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    escribir(el, 'username', 'aquispe');
    escribir(el, 'password', 'clave-inicial');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    expect(navegar).toHaveBeenCalledWith(['/cambiar-clave'], {});
  });

  it('no ofrece crear usuarios: los crea el administrador desde dentro', async () => {
    const { el } = await abrir('/login');
    expect(el.querySelector('a[href="/register"]')).toBeNull();
    expect(el.textContent).not.toContain('Crear usuario');
  });

  it('el botón de la contraseña es accesible', async () => {
    const { harness, el } = await abrir('/login');
    const ojo = el.querySelector('.control-con-icono__boton') as HTMLButtonElement;
    expect(ojo.getAttribute('aria-label')).toBe('Mostrar contraseña');
    ojo.click();
    harness.detectChanges();
    expect((el.querySelector('#password') as HTMLInputElement).type).toBe('text');
    expect(ojo.getAttribute('aria-label')).toBe('Ocultar contraseña');
  });
});
