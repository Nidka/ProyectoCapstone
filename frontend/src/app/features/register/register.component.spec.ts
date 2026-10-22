import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';
import { of } from 'rxjs';
import { RegisterComponent } from './register.component';
import { AuthService } from '../../core/services/auth.service';

describe('RegisterComponent', () => {
  it('valida que las contraseñas coincidan y al crear vuelve al login con aviso', async () => {
    const register = vi.fn(() => of({}));
    TestBed.configureTestingModule({
      providers: [
        provideRouter([{ path: 'register', component: RegisterComponent }, { path: 'login', children: [] }]),
        { provide: AuthService, useValue: { register } },
      ],
    });
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/register', RegisterComponent);
    const el = harness.routeNativeElement as HTMLElement;
    const escribir = (id: string, valor: string) => {
      const campo = el.querySelector(`#${id}`) as HTMLInputElement;
      campo.value = valor;
      campo.dispatchEvent(new Event('input'));
    };
    escribir('nombre', 'Ana Quispe');
    escribir('username', 'aquispe');
    escribir('email', 'ana@lizar.pe');
    escribir('password', 'secreto1');
    escribir('confirmPassword', 'otro');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    harness.detectChanges();
    expect(el.querySelector('.alerta--error')?.textContent?.trim()).toBe('Las contraseñas no coinciden');

    const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    escribir('confirmPassword', 'secreto1');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    expect(register).toHaveBeenCalledWith({ username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'operador', password: 'secreto1' });
    expect(navegar).toHaveBeenCalledWith(['/login'], { queryParams: { registered: '1' } });
  });
});
