import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HitoComponent } from './hito.component';

describe('HitoComponent', () => {
  it('pinta la cifra en el poste y el estado en el cabezal, con el color de su estado', () => {
    const f = TestBed.createComponent(HitoComponent);
    f.componentRef.setInput('cabezal', 'Pagado');
    f.componentRef.setInput('cifra', 2);
    f.componentRef.setInput('estado', 'Pagado');
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.classList).toContain('hito');
    expect(el.getAttribute('data-estado')).toBe('Pagado');
    expect(el.querySelector('.hito__cabezal')?.textContent?.trim()).toBe('Pagado');
    expect(el.querySelector('.hito__cifra')?.textContent?.trim()).toBe('2');
  });

  it('el hito grande lleva su clase', () => {
    const f = TestBed.createComponent(HitoComponent);
    f.componentRef.setInput('cabezal', 'Registrado');
    f.componentRef.setInput('cifra', 'S/ 120.50');
    f.componentRef.setInput('tamano', 'grande');
    f.detectChanges();
    expect((f.nativeElement as HTMLElement).classList).toContain('hito--grande');
  });
});
