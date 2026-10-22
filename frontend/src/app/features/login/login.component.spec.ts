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
      provideRouter([{ path: 'login', component: LoginComponent }, { path: 'register', children: [] }]),
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

  it('avisa que el usuario se creó y el botón de la contraseña es accesible', async () => {
    const { harness, el } = await abrir('/login?registered=1');
    expect(el.querySelector('.alerta--exito')?.textContent?.trim()).toBe('Usuario creado. Ya puedes iniciar sesión.');
    const ojo = el.querySelector('.control-con-icono__boton') as HTMLButtonElement;
    expect(ojo.getAttribute('aria-label')).toBe('Mostrar contraseña');
    ojo.click();
    harness.detectChanges();
    expect((el.querySelector('#password') as HTMLInputElement).type).toBe('text');
    expect(ojo.getAttribute('aria-label')).toBe('Ocultar contraseña');
  });
});
