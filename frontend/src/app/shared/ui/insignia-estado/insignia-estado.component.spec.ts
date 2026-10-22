import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { InsigniaEstadoComponent } from './insignia-estado.component';

function insignia(estado: string, tipo?: 'gasto' | 'comprobante'): HTMLElement {
  const f = TestBed.createComponent(InsigniaEstadoComponent);
  f.componentRef.setInput('estado', estado);
  if (tipo) f.componentRef.setInput('tipo', tipo);
  f.detectChanges();
  return f.nativeElement.querySelector('.lbl');
}

describe('InsigniaEstadoComponent', () => {
  it('usa la etiqueta y el color exactos de un gasto', () => {
    const el = insignia('Pagado');
    expect(el.textContent?.trim()).toBe('Pagado');
    expect(el.classList).toContain('lbl-info');
  });

  it('usa la etiqueta exacta de un comprobante pendiente', () => {
    const el = insignia('Pendiente', 'comprobante');
    expect(el.textContent?.trim()).toBe('Pendiente de aprobación');
    expect(el.classList).toContain('lbl-warn');
  });
});
