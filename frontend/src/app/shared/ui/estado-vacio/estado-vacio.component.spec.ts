import { describe, expect, it } from 'vitest';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { EstadoVacioComponent } from './estado-vacio.component';

@Component({
  imports: [EstadoVacioComponent],
  template: `
    <app-estado-vacio icono="inbox" titulo="Aún no hay rubros." texto="Agrega el primero.">
      <button type="button">Nuevo rubro</button>
    </app-estado-vacio>
  `,
})
class Anfitrion {}

describe('EstadoVacioComponent', () => {
  it('muestra título, texto y la acción proyectada', () => {
    const f = TestBed.createComponent(Anfitrion);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.vacio__titulo')?.textContent).toBe('Aún no hay rubros.');
    expect(el.querySelector('.vacio__texto')?.textContent).toBe('Agrega el primero.');
    expect(el.querySelector('.vacio__acciones button')?.textContent).toBe('Nuevo rubro');
  });
});
