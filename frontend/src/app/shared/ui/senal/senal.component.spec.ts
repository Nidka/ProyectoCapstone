import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { SenalComponent } from './senal.component';

describe('SenalComponent', () => {
  it('dibuja la señal de su clase con el icono y queda oculta para lectores', () => {
    const f = TestBed.createComponent(SenalComponent);
    f.componentRef.setInput('tipo', 'preventiva');
    f.componentRef.setInput('icono', 'exclamation-mark');
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.classList).toContain('senal--preventiva');
    expect(el.getAttribute('aria-hidden')).toBe('true');
    expect(el.querySelectorAll('svg path').length).toBeGreaterThan(0);
  });

  it('el tamaño llega como variable de estilo', () => {
    const f = TestBed.createComponent(SenalComponent);
    f.componentRef.setInput('tipo', 'guia');
    f.componentRef.setInput('icono', 'check');
    f.componentRef.setInput('tamano', 56);
    f.detectChanges();
    expect((f.nativeElement as HTMLElement).style.getPropertyValue('--tam')).toBe('56px');
  });
});
