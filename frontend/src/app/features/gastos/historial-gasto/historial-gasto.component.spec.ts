import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { Subject, throwError } from 'rxjs';
import { HistorialGastoComponent } from './historial-gasto.component';
import { HistorialService } from '../../../core/services/historial.service';
import { RegistroHistorial } from '../../../core/models/historial';

function registro(extra: Partial<RegistroHistorial>): RegistroHistorial {
  return { fecha: '2026-10-02T14:05:00-05:00', usuario: 'Ana Quispe', accion: 'gasto.registrar', entidad: 'gasto', id_entidad: 31, detalle: {}, ...extra };
}

const REGISTRADO = registro({});
const ADJUNTADO = registro({ fecha: '2026-10-02T15:30:00-05:00', accion: 'comprobante.adjuntar', entidad: 'comprobante', id_entidad: 7 });
const RECHAZADO = registro({
  fecha: '2026-10-03T09:10:00-05:00', usuario: 'Luis Rojas', accion: 'comprobante.rechazar', entidad: 'comprobante', id_entidad: 7,
  detalle: { despues: { estado: 'Rechazado', motivo_rechazo: 'Foto borrosa' } },
});

function montar(deGasto: (id: number) => unknown) {
  const servicio = { deGasto: vi.fn(deGasto) };
  TestBed.configureTestingModule({ providers: [{ provide: HistorialService, useValue: servicio }] });
  const f = TestBed.createComponent(HistorialGastoComponent);
  f.componentRef.setInput('gastoId', 31);
  f.detectChanges();
  return { f, servicio, el: f.nativeElement as HTMLElement };
}

const items = (el: HTMLElement) => [...el.querySelectorAll('.bitacora__item')].map(i => i.textContent?.replace(/\s+/g, ' ').trim());
const boton = (el: HTMLElement, texto: string) => [...el.querySelectorAll('button')].find(b => b.textContent?.trim() === texto);

describe('HistorialGastoComponent', () => {
  it('tiene su título y, mientras carga, un esqueleto marcado como ocupado', () => {
    const { el, servicio } = montar(() => new Subject<RegistroHistorial[]>());
    expect(el.querySelector('h2')?.textContent).toContain('Historial de cambios');
    expect(el.querySelector('[aria-busy="true"] .esqueleto')).not.toBeNull();
    expect(servicio.deGasto).toHaveBeenCalledWith(31);
  });

  it('lista los movimientos en orden cronológico con fecha en hora de Lima, usuario y acción en español', () => {
    const salida = new Subject<RegistroHistorial[]>();
    const { f, el } = montar(() => salida);
    // Llegan desordenados: se muestran del más antiguo al más reciente
    salida.next([RECHAZADO, REGISTRADO, ADJUNTADO]);
    f.detectChanges();
    expect(el.querySelector('[aria-busy="true"]')).toBeNull();
    const filas = items(el);
    expect(filas).toHaveLength(3);
    expect(filas[0]).toContain('02/10/2026 14:05');
    expect(filas[0]).toContain('Registró el gasto');
    expect(filas[0]).toContain('Ana Quispe');
    expect(filas[1]).toContain('Adjuntó un comprobante');
    expect(filas[2]).toContain('Rechazó el comprobante');
    expect(filas[2]).toContain('Luis Rojas');
    expect(filas[2]).toContain('Motivo: Foto borrosa');
    expect(el.querySelector('time')?.getAttribute('datetime')).toBe('2026-10-02T14:05:00-05:00');
  });

  it('un código de acción desconocido sale con un texto general, no crudo', () => {
    const salida = new Subject<RegistroHistorial[]>();
    const { f, el } = montar(() => salida);
    salida.next([registro({ accion: 'gasto.teletransportar' })]);
    f.detectChanges();
    expect(items(el)[0]).toContain('Hizo un cambio');
    expect(el.textContent).not.toContain('teletransportar');
  });

  it('sin movimientos dice que todavía no hay cambios', () => {
    const salida = new Subject<RegistroHistorial[]>();
    const { f, el } = montar(() => salida);
    salida.next([]);
    f.detectChanges();
    expect(el.querySelector('.bitacora')).toBeNull();
    expect(el.textContent).toContain('Todavía no hay cambios registrados para este gasto.');
  });

  it('si falla, avisa sin dejar el esqueleto y Reintentar vuelve a pedir el historial', () => {
    const respuestas = [throwError(() => new Error('red')), new Subject<RegistroHistorial[]>()];
    const { f, el, servicio } = montar(() => respuestas.shift());
    expect(el.querySelector('[aria-busy="true"]')).toBeNull();
    expect(el.textContent).toContain('No se pudo cargar el historial.');
    expect(el.querySelector('[role="alert"]')).toBeNull();
    boton(el, 'Reintentar')!.click();
    f.detectChanges();
    expect(servicio.deGasto).toHaveBeenCalledTimes(2);
    expect(el.querySelector('[aria-busy="true"]')).not.toBeNull();
    expect(el.textContent).not.toContain('No se pudo cargar el historial.');
  });

  it('si el gasto tuvo movimientos nuevos (marca distinta), recarga sin tapar la lista con el esqueleto', () => {
    const salidas = [new Subject<RegistroHistorial[]>(), new Subject<RegistroHistorial[]>()];
    const { f, el, servicio } = montar(() => salidas.shift());
    (servicio.deGasto.mock.results[0].value as Subject<RegistroHistorial[]>).next([REGISTRADO]);
    f.detectChanges();
    expect(items(el)).toHaveLength(1);

    f.componentRef.setInput('marca', 'Pagado');
    f.detectChanges();
    expect(servicio.deGasto).toHaveBeenCalledTimes(2);
    expect(items(el)).toHaveLength(1);
    expect(el.querySelector('[aria-busy="true"]')).toBeNull();

    (servicio.deGasto.mock.results[1].value as Subject<RegistroHistorial[]>).next([REGISTRADO, ADJUNTADO]);
    f.detectChanges();
    expect(items(el)).toHaveLength(2);
  });

  it('al pasar a otro gasto vacía la lista anterior y descarta la respuesta atrasada del primero', () => {
    const { f, el, servicio } = montar(() => new Subject<RegistroHistorial[]>());
    const primero = servicio.deGasto.mock.results[0].value as Subject<RegistroHistorial[]>;
    f.componentRef.setInput('gastoId', 32);
    f.detectChanges();
    expect(servicio.deGasto).toHaveBeenLastCalledWith(32);
    primero.next([REGISTRADO]);
    f.detectChanges();
    expect(items(el)).toHaveLength(0);
    expect(el.querySelector('[aria-busy="true"]')).not.toBeNull();
    (servicio.deGasto.mock.results[1].value as Subject<RegistroHistorial[]>).next([ADJUNTADO]);
    f.detectChanges();
    expect(items(el)).toHaveLength(1);
  });

  it('un fallo al recargar muestra el error, no una lista que ya puede estar desactualizada', () => {
    const respuestas = [new Subject<RegistroHistorial[]>(), throwError(() => new Error('red'))];
    const { f, el, servicio } = montar(() => respuestas.shift());
    (servicio.deGasto.mock.results[0].value as Subject<RegistroHistorial[]>).next([REGISTRADO]);
    f.detectChanges();
    f.componentRef.setInput('marca', 'Pagado');
    f.detectChanges();
    expect(items(el)).toHaveLength(0);
    expect(el.textContent).toContain('No se pudo cargar el historial.');
  });
});
