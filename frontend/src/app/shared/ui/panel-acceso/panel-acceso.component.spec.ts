import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { PanelAccesoComponent } from './panel-acceso.component';

describe('PanelAccesoComponent', () => {
  it('abre con la foto de la carretera, la marca grande y la ruta de los tres hitos', () => {
    const f = TestBed.createComponent(PanelAccesoComponent);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.acceso__foto img')?.getAttribute('alt')).toContain('Camión de carga');
    expect(el.querySelector('app-marca.acceso__marca-grande')).not.toBeNull();
    expect([...el.querySelectorAll('.acceso__hito')].map(h => h.textContent?.trim())).toEqual(['Registrado', 'Pagado', 'Sustentado']);
  });
});
