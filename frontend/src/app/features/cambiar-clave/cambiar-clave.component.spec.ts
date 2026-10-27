import { describe, expect, it, vi } from 'vitest';
import { HttpErrorResponse } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';
import { Observable, Subject, of, throwError } from 'rxjs';
import { CambiarClaveComponent } from './cambiar-clave.component';
import { AuthService } from '../../core/services/auth.service';
import { ToastService } from '../../core/services/toast.service';
import { MSG_ERROR_RED } from '../../core/utils/error-messages';

interface Opciones {
  forzado?: boolean;
  cambiar?: (body: { clave_actual: string; clave_nueva: string }) => Observable<unknown>;
}

async function abrir(url = '/cambiar-clave', op: Opciones = {}) {
  const auth = {
    debeCambiarClave: () => op.forzado ?? true,
    cambiarClave: vi.fn(op.cambiar ?? (() => of({ debe_cambiar_clave: false }))),
    logout: vi.fn(),
  };
  TestBed.configureTestingModule({
    providers: [provideRouter([{ path: 'cambiar-clave', component: CambiarClaveComponent }]), { provide: AuthService, useValue: auth }],
  });
  const harness = await RouterTestingHarness.create();
  await harness.navigateByUrl(url, CambiarClaveComponent);
  const el = harness.routeNativeElement as HTMLElement;
  const escribir = (id: string, valor: string) => {
    const campo = el.querySelector(`#${id}`) as HTMLInputElement;
    campo.value = valor;
    campo.dispatchEvent(new Event('input'));
  };
  const enviar = async () => {
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    harness.detectChanges();
    await harness.fixture.whenStable();
    harness.detectChanges();
  };
  const llenar = (actual = 'clave-vieja', nueva = 'clave-nueva-1', confirmacion = nueva) => {
    escribir('clave-actual', actual);
    escribir('clave-nueva', nueva);
    escribir('clave-confirmacion', confirmacion);
  };
  const errores = () => [...el.querySelectorAll('.campo__error')].map(e => e.textContent?.trim());
  return { harness, el, auth, escribir, enviar, llenar, errores, router: TestBed.inject(Router), toast: TestBed.inject(ToastService) };
}

describe('CambiarClaveComponent', () => {
  it('cuando es obligatorio lo dice y ofrece cerrar sesión en vez de cancelar', async () => {
    const { el, auth } = await abrir('/cambiar-clave');
    expect(el.querySelector('h1')?.textContent).toContain('Cambiar contraseña');
    expect(el.textContent).toContain('antes de continuar');
    expect(el.querySelector('a[href="/app/inicio"]')).toBeNull();
    const salir = [...el.querySelectorAll('button')].find(b => b.textContent?.includes('Cerrar sesión'))!;
    salir.click();
    expect(auth.logout).toHaveBeenCalled();
  });

  it('cuando se cambia por gusto ofrece cancelar y volver a Inicio', async () => {
    const { el } = await abrir('/cambiar-clave', { forzado: false });
    expect(el.textContent).not.toContain('antes de continuar');
    expect(el.querySelector('a[href="/app/inicio"]')?.textContent).toContain('Cancelar');
    expect([...el.querySelectorAll('button')].some(b => b.textContent?.includes('Cerrar sesión'))).toBe(false);
  });

  it('pide los tres datos y no envía nada si faltan', async () => {
    const { auth, enviar, errores } = await abrir();
    await enviar();
    expect(auth.cambiarClave).not.toHaveBeenCalled();
    expect(errores()).toEqual(['Escribe tu contraseña actual.', 'Usa al menos 8 caracteres.', 'Repite la contraseña nueva.']);
  });

  it('exige mínimo 8 caracteres, distinta de la actual y bien repetida', async () => {
    const { auth, enviar, errores, llenar } = await abrir();
    llenar('clave-vieja', 'corta', 'corta');
    await enviar();
    expect(errores()).toEqual(['Usa al menos 8 caracteres.']);
    llenar('clave-vieja', 'clave-vieja', 'clave-vieja');
    await enviar();
    expect(errores()).toEqual(['La contraseña nueva debe ser distinta de la actual.']);
    llenar('clave-vieja', 'clave-nueva-1', 'clave-nueva-2');
    await enviar();
    expect(errores()).toEqual(['Las contraseñas no coinciden.']);
    expect(auth.cambiarClave).not.toHaveBeenCalled();
  });

  it('marca los campos con error para lectores de pantalla', async () => {
    const { el, enviar } = await abrir();
    await enviar();
    const actual = el.querySelector('#clave-actual')!;
    expect(actual.getAttribute('aria-invalid')).toBe('true');
    expect(el.querySelector(`#${actual.getAttribute('aria-describedby')}`)?.textContent).toContain('contraseña actual');
  });

  it('envía la contraseña actual y la nueva; al terminar avisa y vuelve a la pantalla que se pidió', async () => {
    const { auth, enviar, llenar, router, toast } = await abrir('/cambiar-clave?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
    const navegar = vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    llenar('clave-vieja', 'clave-nueva-1');
    await enviar();
    expect(auth.cambiarClave).toHaveBeenCalledWith({ clave_actual: 'clave-vieja', clave_nueva: 'clave-nueva-1' });
    expect(navegar).toHaveBeenCalledWith('/app/gastos?estado=por-aprobar');
    expect(toast.toasts().map(t => t.title)).toEqual(['Contraseña cambiada.']);
  });

  it('sin «volver» entra a Inicio', async () => {
    const { enviar, llenar, router } = await abrir();
    const navegar = vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    llenar();
    await enviar();
    expect(navegar).toHaveBeenCalledWith('/app/inicio');
  });

  it('mientras guarda deshabilita el botón y no envía dos veces', async () => {
    const respuesta = new Subject<unknown>();
    const { el, auth, enviar, llenar, harness } = await abrir('/cambiar-clave', { cambiar: () => respuesta });
    llenar();
    await enviar();
    await enviar();
    expect(auth.cambiarClave).toHaveBeenCalledTimes(1);
    const boton = el.querySelector('button[type="submit"]') as HTMLButtonElement;
    expect(boton.disabled).toBe(true);
    expect(boton.textContent).toContain('Guardando');
    respuesta.complete();
    harness.detectChanges();
  });

  it('la contraseña actual incorrecta (400) se marca en su campo y deja reintentar', async () => {
    const { el, enviar, llenar, errores, harness } = await abrir('/cambiar-clave', {
      cambiar: () => throwError(() => new HttpErrorResponse({ status: 400, error: { detail: 'La contraseña actual no es correcta' } })),
    });
    llenar();
    await enviar();
    expect(errores()).toEqual(['La contraseña actual no es correcta']);
    expect((el.querySelector('button[type="submit"]') as HTMLButtonElement).disabled).toBe(false);
    expect(harness.routeNativeElement?.querySelector('#clave-actual')?.getAttribute('aria-invalid')).toBe('true');
  });

  it('un fallo de red se dice en un aviso, sin perder lo escrito', async () => {
    const { el, enviar, llenar } = await abrir('/cambiar-clave', {
      cambiar: () => throwError(() => new HttpErrorResponse({ status: 0 })),
    });
    llenar();
    await enviar();
    expect(el.querySelector('.alerta--error')?.textContent).toContain(MSG_ERROR_RED);
    expect((el.querySelector('#clave-nueva') as HTMLInputElement).value).toBe('clave-nueva-1');
  });

  it('el botón del ojo muestra u oculta las tres contraseñas', async () => {
    const { el, harness } = await abrir();
    const casilla = el.querySelector('input[type="checkbox"]') as HTMLInputElement;
    expect((el.querySelector('#clave-nueva') as HTMLInputElement).type).toBe('password');
    casilla.click();
    harness.detectChanges();
    for (const id of ['clave-actual', 'clave-nueva', 'clave-confirmacion']) {
      expect((el.querySelector(`#${id}`) as HTMLInputElement).type).toBe('text');
    }
  });
});
