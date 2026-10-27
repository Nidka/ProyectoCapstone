import { describe, expect, it, vi } from 'vitest';
import { signal } from '@angular/core';
import { Router, provideRouter } from '@angular/router';
import { TestBed } from '@angular/core/testing';
import { MenuUsuarioComponent, iniciales } from './menu-usuario.component';
import { AuthService } from '../../core/services/auth.service';

describe('iniciales', () => {
  it('usa las iniciales de nombre y apellido, o las dos primeras letras', () => {
    expect(iniciales('Ana Quispe')).toBe('AQ');
    expect(iniciales('admin')).toBe('AD');
    expect(iniciales('   ')).toBe('US');
  });
});

describe('MenuUsuarioComponent', () => {
  function rolMostrado(rol: string): string | undefined {
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [{ provide: AuthService, useValue: { usuario: signal({ nombre: 'Ana Quispe', rol }), logout: () => {} } }],
    });
    const f = TestBed.createComponent(MenuUsuarioComponent);
    f.detectChanges();
    return (f.nativeElement as HTMLElement).querySelector('.menu__rol')?.textContent?.trim();
  }

  it('dice el cargo en palabras, no el código del rol', () => {
    expect(rolMostrado('logistica')).toBe('Encargado de Logística');
    expect(rolMostrado('admin')).toBe('Administrador del Sistema');
  });
});

describe('MenuUsuarioComponent: cambiar contraseña', () => {
  it('ofrece cambiar la contraseña y lleva a esa pantalla', () => {
    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [provideRouter([]), { provide: AuthService, useValue: { usuario: signal({ nombre: 'Ana Quispe', rol: 'logistica' }), logout: () => {} } }],
    });
    const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    const f = TestBed.createComponent(MenuUsuarioComponent);
    f.detectChanges();
    const el = f.nativeElement as HTMLElement;
    const item = [...el.querySelectorAll<HTMLButtonElement>('.menu__item')].find(b => b.textContent?.includes('Cambiar contraseña'))!;
    expect(item).toBeTruthy();
    item.click();
    expect(navegar).toHaveBeenCalledWith(['/cambiar-clave']);
  });
});
