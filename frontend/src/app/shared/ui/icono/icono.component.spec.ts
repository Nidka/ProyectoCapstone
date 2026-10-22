import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { IconoComponent } from './icono.component';
import { ICONOS } from './iconos';

describe('ICONOS', () => {
  it('cada icono tiene al menos un trazo', () => {
    for (const [nombre, trazos] of Object.entries(ICONOS)) {
      expect(trazos.length, nombre).toBeGreaterThan(0);
    }
  });
});

describe('IconoComponent', () => {
  it('dibuja los trazos del icono con el tamaño pedido y oculto para lectores', () => {
    const fixture = TestBed.createComponent(IconoComponent);
    fixture.componentRef.setInput('nombre', 'home');
    fixture.componentRef.setInput('tamano', 22);
    fixture.detectChanges();
    const svg: SVGElement = fixture.nativeElement.querySelector('svg');
    expect(svg.getAttribute('width')).toBe('22');
    expect(svg.getAttribute('aria-hidden')).toBe('true');
    expect(svg.querySelectorAll('path').length).toBe(ICONOS.home.length);
  });
});
