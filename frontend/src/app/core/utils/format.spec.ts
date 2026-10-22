import { describe, expect, it } from 'vitest';
import { formatFecha, formatFechaHora, formatMonto, hoyIso } from './format';

describe('formatMonto', () => {
  it('usa separador de miles y dos decimales de es-PE', () => {
    expect(formatMonto(1240.5)).toBe('S/ 1,240.50');
    expect(formatMonto(1234567.891)).toBe('S/ 1,234,567.89');
  });

  it('acepta texto, nulos y valores inválidos', () => {
    expect(formatMonto('85.5')).toBe('S/ 85.50');
    expect(formatMonto(null)).toBe('S/ 0.00');
    expect(formatMonto('abc')).toBe('S/ 0.00');
  });
});

describe('formatFecha', () => {
  it('lee las fechas sin hora en UTC', () => {
    expect(formatFecha('2026-10-02T00:00:00Z')).toBe('02/10/2026');
  });

  it('muestra el vacío existente cuando no hay fecha', () => {
    expect(formatFecha(null)).toBe('—');
  });
});

describe('formatFechaHora', () => {
  it('muestra la hora de Lima', () => {
    expect(formatFechaHora('2026-10-02T19:05:00Z')).toBe('02/10/2026 14:05');
  });

  it('muestra el vacío existente cuando no hay fecha', () => {
    expect(formatFechaHora(undefined)).toBe('—');
  });
});

describe('hoyIso', () => {
  it('usa el día de Lima, no el de UTC', () => {
    expect(hoyIso(new Date('2026-10-03T02:30:00Z'))).toBe('2026-10-02');
  });
});
