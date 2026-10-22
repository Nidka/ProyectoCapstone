import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { FABRICA_GRAFICO, GraficoComponent } from './grafico.component';
import type { ConfigGrafico } from '../graficos';

const config = (n: number): ConfigGrafico => ({ type: 'bar', data: { labels: ['a'], datasets: [{ data: [n] }] } });

function montar() {
  const instancia = { actualizar: vi.fn(), destruir: vi.fn() };
  const fabrica = vi.fn(() => instancia);
  TestBed.configureTestingModule({ providers: [{ provide: FABRICA_GRAFICO, useValue: fabrica }] });
  const f = TestBed.createComponent(GraficoComponent);
  f.componentRef.setInput('config', config(1));
  f.componentRef.setInput('descripcion', 'Gasto por mes de enero a marzo.');
  f.componentRef.setInput('alto', 300);
  f.detectChanges();
  return { f, fabrica, instancia };
}

describe('GraficoComponent', () => {
  it('crea el gráfico sobre su lienzo, con su descripción accesible y su alto', () => {
    const { f, fabrica } = montar();
    const lienzo: HTMLCanvasElement = f.nativeElement.querySelector('canvas');
    expect(fabrica).toHaveBeenCalledWith(lienzo, config(1));
    expect(lienzo.getAttribute('role')).toBe('img');
    expect(lienzo.getAttribute('aria-label')).toBe('Gasto por mes de enero a marzo.');
    expect((f.nativeElement.querySelector('.grafico') as HTMLElement).style.height).toBe('300px');
  });

  it('con otra configuración actualiza el mismo gráfico en vez de crear otro', () => {
    const { f, fabrica, instancia } = montar();
    f.componentRef.setInput('config', config(2));
    f.detectChanges();
    expect(fabrica).toHaveBeenCalledTimes(1);
    expect(instancia.actualizar).toHaveBeenCalledWith(config(2));
  });

  it('destruye el gráfico al salir', () => {
    const { f, instancia } = montar();
    f.destroy();
    expect(instancia.destruir).toHaveBeenCalled();
  });
});
