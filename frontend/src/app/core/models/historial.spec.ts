import { describe, expect, it } from 'vitest';
import { RegistroHistorial, detalleDeRegistro, ordenarCronologico, textoAccion } from './historial';

function registro(extra: Partial<RegistroHistorial>): RegistroHistorial {
  return { fecha: '2026-10-02T14:05:00-05:00', usuario: 'Ana Quispe', accion: 'gasto.registrar', entidad: 'gasto', id_entidad: 31, detalle: {}, ...extra };
}

describe('textoAccion', () => {
  it('cuenta en español sencillo lo que pasó con el gasto y sus comprobantes', () => {
    expect(textoAccion('gasto.registrar')).toBe('Registró el gasto');
    expect(textoAccion('comprobante.adjuntar')).toBe('Adjuntó un comprobante');
    expect(textoAccion('comprobante.aprobar')).toBe('Aprobó el comprobante');
    expect(textoAccion('comprobante.rechazar')).toBe('Rechazó el comprobante');
    expect(textoAccion('gasto.corregir')).toBe('Corrigió el gasto');
  });

  it('conoce también el resto de las acciones que guarda la bitácora', () => {
    for (const accion of [
      'checklist.configurar', 'checklist.restablecer', 'usuario.crear', 'usuario.editar', 'usuario.desactivar',
      'usuario.activar', 'usuario.cambiar_clave', 'sesion.inicio', 'sesion.cierre', 'maestro.crear', 'maestro.editar', 'maestro.eliminar',
    ]) {
      expect(textoAccion(accion)).not.toBe(textoAccion('cosa.rara'));
    }
  });

  it('un código desconocido no se muestra crudo: sale un texto general', () => {
    expect(textoAccion('cosa.rara')).toBe('Hizo un cambio');
    expect(textoAccion('')).toBe('Hizo un cambio');
  });
});

describe('detalleDeRegistro', () => {
  it('en un rechazo muestra el motivo', () => {
    const r = registro({ accion: 'comprobante.rechazar', detalle: { despues: { estado: 'Rechazado', motivo_rechazo: 'Foto borrosa' } } });
    expect(detalleDeRegistro(r)).toBe('Motivo: Foto borrosa');
  });

  it('en una aprobación dice cuántos puntos del checklist se marcaron', () => {
    const r = registro({
      accion: 'comprobante.aprobar',
      detalle: { checklist: [{ criterio: 'a', texto: 'A', cumple: true }, { criterio: 'b', texto: 'B', cumple: true }] },
    });
    expect(detalleDeRegistro(r)).toBe('Checklist: 2 de 2 puntos');
  });

  it('en el resto, y si el detalle viene raro, no inventa nada', () => {
    expect(detalleDeRegistro(registro({}))).toBe('');
    expect(detalleDeRegistro(registro({ accion: 'comprobante.rechazar', detalle: null as never }))).toBe('');
    expect(detalleDeRegistro(registro({ accion: 'comprobante.rechazar', detalle: { despues: { motivo_rechazo: 5 } } }))).toBe('');
  });
});

describe('detalleDeRegistro de una corrección', () => {
  const corregir = (detalle: Record<string, unknown>) => registro({ accion: 'gasto.corregir', detalle });

  it('dice qué cambió, de cuánto a cuánto, y el motivo', () => {
    expect(
      detalleDeRegistro(
        corregir({
          antes: { monto: 100, fecha: '2026-10-24' },
          despues: { monto: 513, fecha: '2026-09-26' },
          motivo: 'Error de digitación',
        }),
      ),
    ).toBe('Monto: S/ 100.00 → S/ 513.00 · Fecha: 24/10/2026 → 26/09/2026 · Motivo: Error de digitación');
  });

  it('nombra solo lo que cambió', () => {
    expect(
      detalleDeRegistro(
        corregir({ antes: { monto: 100, fecha: '2026-10-24' }, despues: { monto: 513, fecha: '2026-10-24' }, motivo: 'El comprobante dice 513' }),
      ),
    ).toBe('Monto: S/ 100.00 → S/ 513.00 · Motivo: El comprobante dice 513');
  });

  it('sin detalle usable no inventa nada', () => {
    expect(detalleDeRegistro(corregir({}))).toBe('');
  });
});

describe('ordenarCronologico', () => {
  it('pone lo más antiguo primero sin tocar la lista original y respeta el orden dado en empates', () => {
    const a = registro({ fecha: '2026-10-02T10:00:00-05:00', accion: 'gasto.registrar' });
    const b = registro({ fecha: '2026-10-02T11:00:00-05:00', accion: 'comprobante.adjuntar' });
    const c = registro({ fecha: '2026-10-02T11:00:00-05:00', accion: 'comprobante.aprobar' });
    const original = [c, b, a];
    expect(ordenarCronologico(original).map(r => r.accion)).toEqual(['gasto.registrar', 'comprobante.aprobar', 'comprobante.adjuntar']);
    expect(original[0]).toBe(c);
  });
});
