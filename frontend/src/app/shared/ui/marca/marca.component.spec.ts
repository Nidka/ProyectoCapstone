import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { MarcaComponent } from './marca.component';

describe('MarcaComponent', () => {
  it('muestra el camión del logo con LIZAR y Fleet Manager Pro', () => {
    const f = TestBed.createComponent(MarcaComponent);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.marca__insignia app-camion svg')).not.toBeNull();
    expect(el.querySelector('.marca__nombre')?.textContent).toBe('LIZAR');
    expect(el.querySelector('.marca__sub')?.textContent).toBe('Fleet Manager Pro');
  });

  it('puede ir sin texto y en versión clara', () => {
    const f = TestBed.createComponent(MarcaComponent);
    f.componentRef.setInput('conTexto', false);
    f.componentRef.setInput('sobreAsfalto', false);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.marca__texto')).toBeNull();
    expect(el.classList).toContain('marca--clara');
  });
});
