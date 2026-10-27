import { describe, expect, it, vi } from 'vitest';
import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { BarraMovilComponent } from './barra-movil.component';
import { AuthService } from '../../core/services/auth.service';
import { GastosStore } from '../../core/state/gastos.store';

@Component({ template: '' })
class Vacio {}

async function montar(url: string, rol: 'admin' | 'logistica' = 'logistica') {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([{ path: '**', component: Vacio }]),
      { provide: AuthService, useValue: { usuario: signal({ nombre: 'Ana Quispe', rol }), logout: vi.fn() } },
      { provide: GastosStore, useValue: { pendientes: signal([]) } },
    ],
  });
  await TestBed.inject(Router).navigateByUrl(url);
  const f = TestBed.createComponent(BarraMovilComponent);
  f.detectChanges();
  return f.nativeElement as HTMLElement;
}

const botonMas = (el: HTMLElement) => [...el.querySelectorAll('button')].find(b => b.textContent?.includes('Más'))!;

describe('BarraMovilComponent', () => {
  it('la hoja «Más» lleva a Reportes', async () => {
    const el = await montar('/app/inicio');
    const enlace = [...el.querySelectorAll<HTMLAnchorElement>('dialog a')].find(a => a.textContent?.includes('Reportes'));
    expect(enlace?.getAttribute('href')).toBe('/app/reportes');
    expect(botonMas(el).classList).not.toContain('tabbar__item--activo');
  });

  it('en Reportes y en Catálogos se marca «Más» como activo', async () => {
    expect(botonMas(await montar('/app/reportes?desde=2026-01-01')).classList).toContain('tabbar__item--activo');
    TestBed.resetTestingModule();
    expect(botonMas(await montar('/app/catalogos/ubigeo')).classList).toContain('tabbar__item--activo');
  });

  it('«Usuarios» aparece en la hoja «Más» solo para el administrador', async () => {
    const enlaces = (el: HTMLElement) => [...el.querySelectorAll<HTMLAnchorElement>('dialog a')].map(a => a.getAttribute('href'));
    expect(enlaces(await montar('/app/inicio', 'admin'))).toContain('/app/usuarios');
    TestBed.resetTestingModule();
    expect(enlaces(await montar('/app/inicio', 'logistica'))).not.toContain('/app/usuarios');
  });

  it('en Usuarios se marca «Más» como activo', async () => {
    expect(botonMas(await montar('/app/usuarios', 'admin')).classList).toContain('tabbar__item--activo');
  });

  it('la hoja «Más» ofrece cambiar la contraseña', async () => {
    const el = await montar('/app/inicio');
    expect([...el.querySelectorAll<HTMLAnchorElement>('dialog a')].find(a => a.textContent?.includes('Cambiar contraseña'))?.getAttribute('href')).toBe('/cambiar-clave');
  });
});
