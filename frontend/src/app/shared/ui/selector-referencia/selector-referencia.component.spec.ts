import { describe, expect, it } from 'vitest';
import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { MAX_OPCIONES, SelectorReferenciaComponent, coincidencias } from './selector-referencia.component';
import { OpcionRef } from '../../../core/models/crud.model';

const opciones: OpcionRef[] = [
  { valor: '1', etiqueta: 'Agroindustria' },
  { valor: '2', etiqueta: 'Minería' },
  { valor: '3', etiqueta: 'Logística' },
];

describe('coincidencias', () => {
  it('filtra sin distinguir tildes ni mayúsculas', () => {
    expect(coincidencias(opciones, 'MINERIA').map(o => o.valor)).toEqual(['2']);
    expect(coincidencias(opciones, '  ')).toHaveLength(3);
  });
});

@Component({
  imports: [SelectorReferenciaComponent],
  template: `<app-selector-referencia idControl="rubro" [opciones]="opciones()" [(valor)]="valor" />`,
})
class Anfitrion {
  opciones = signal<OpcionRef[]>(opciones);
  valor = signal('');
}

function montar() {
  TestBed.configureTestingModule({ providers: [provideRouter([])] });
  const f = TestBed.createComponent(Anfitrion);
  f.detectChanges();
  const input: HTMLInputElement = f.nativeElement.querySelector('input');
  return { f, input };
}

describe('SelectorReferenciaComponent', () => {
  it('se elige con teclado: escribir y Enter', () => {
    const { f, input } = montar();
    input.dispatchEvent(new FocusEvent('focus'));
    input.value = 'log';
    input.dispatchEvent(new Event('input'));
    f.detectChanges();
    expect(f.nativeElement.querySelectorAll('[role="option"]')).toHaveLength(1);
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    f.detectChanges();
    expect(f.componentInstance.valor()).toBe('3');
    expect(input.value).toBe('Logística');
    expect(input.getAttribute('aria-expanded')).toBe('false');
  });

  it('con muchas opciones muestra 50 y avisa cuántas hay', () => {
    const { f, input } = montar();
    f.componentInstance.opciones.set(Array.from({ length: 80 }, (_, i) => ({ valor: String(i), etiqueta: `Distrito ${i}` })));
    f.detectChanges();
    input.dispatchEvent(new FocusEvent('focus'));
    f.detectChanges();
    expect(f.nativeElement.querySelectorAll('[role="option"]')).toHaveLength(MAX_OPCIONES);
    expect(f.nativeElement.textContent).toContain('Mostrando 50 de 80');
  });

  it('sin registros ofrece el enlace para crearlos', () => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
    const f = TestBed.createComponent(SelectorReferenciaComponent);
    f.componentRef.setInput('idControl', 'rubro');
    f.componentRef.setInput('opciones', []);
    f.componentRef.setInput('vacioTexto', 'Aún no hay rubros.');
    f.componentRef.setInput('crearRuta', '/app/catalogos/rubros');
    f.componentRef.setInput('crearTexto', 'Nuevo rubro');
    f.detectChanges();
    f.nativeElement.querySelector('input').dispatchEvent(new FocusEvent('focus'));
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('Aún no hay rubros.');
    expect(f.nativeElement.querySelector('a')?.getAttribute('href')).toBe('/app/catalogos/rubros');
  });
});
