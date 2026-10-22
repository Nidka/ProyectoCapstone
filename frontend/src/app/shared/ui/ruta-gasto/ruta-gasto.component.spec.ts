import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { RutaGastoComponent } from './ruta-gasto.component';
import { PasoRuta } from '../../../core/state/gastos.logic';
import { Comprobante } from '../../../core/models/gasto.model';

const pasos: PasoRuta[] = [
  { estado: 'Registrado', alcanzado: true, actual: false, fecha: '2026-10-01T15:00:00Z', actor: 'Ana Quispe' },
  { estado: 'Pagado', alcanzado: true, actual: true, fecha: '2026-10-01T17:00:00Z' },
  { estado: 'Sustentado', alcanzado: false, actual: false },
];

function crear(variante: 'compacta' | 'completa', rechazo?: Comprobante): HTMLElement {
  const f = TestBed.createComponent(RutaGastoComponent);
  f.componentRef.setInput('pasos', pasos);
  f.componentRef.setInput('variante', variante);
  if (rechazo) f.componentRef.setInput('rechazo', rechazo);
  f.detectChanges();
  return f.nativeElement;
}

describe('RutaGastoComponent', () => {
  it('la versión compacta es una imagen descrita para lectores de pantalla', () => {
    const el = crear('compacta');
    expect(el.getAttribute('role')).toBe('img');
    expect(el.getAttribute('aria-label')).toBe('Recorrido: Registrado, Pagado (actual), Sustentado pendiente');
    expect(el.querySelectorAll('.ruta__hito')).toHaveLength(3);
    expect(el.querySelector('.ruta__nombre')).toBeNull();
  });

  it('la versión completa muestra estado, hora de Lima y quién', () => {
    const el = crear('completa');
    const hitos = el.querySelectorAll('.ruta__hito');
    expect(el.getAttribute('role')).toBeNull();
    expect(hitos[1].classList).toContain('ruta__hito--actual');
    expect(hitos[1].getAttribute('aria-current')).toBe('step');
    expect(hitos[0].textContent).toContain('01/10/2026 10:00');
    expect(hitos[0].textContent).toContain('Ana Quispe');
    expect(hitos[2].textContent).toContain('Pendiente');
  });

  it('dibuja el desvío con su fecha; el motivo lo muestran el paso de adjuntar y el historial', () => {
    const el = crear('completa', {
      id: 5, gasto_id: 1, archivo_url: '/uploads/5.jpg', tipo_archivo: 'imagen', estado: 'Rechazado',
      motivo_rechazo: 'Monto no coincide', aprobado_en: '2026-10-01T16:30:00Z', creado_en: '2026-10-01T16:00:00Z',
    });
    const desvio = el.querySelector('.ruta__desvio')?.textContent?.trim();
    expect(desvio).toBe('Comprobante rechazado el 01/10/2026 11:30.');
  });
});
