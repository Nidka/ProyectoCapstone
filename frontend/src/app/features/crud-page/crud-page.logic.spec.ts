import { describe, expect, it } from 'vitest';
import { EstadoCampo, enumerar, validarCampos } from './crud-page.logic';

describe('validarCampos', () => {
  it('marca obligatorios, reglas y números con los mensajes de siempre', () => {
    const campos: EstadoCampo[] = [
      { field: { id: 'a', l: 'Nombre', tp: 'text', req: true }, value: '   ', error: '' },
      { field: { id: 'b', l: 'RUC', tp: 'text', req: true, rule: v => /^\d{11}$/.test(v), msg: '11 dígitos' }, value: '123', error: '' },
      { field: { id: 'c', l: 'Año', tp: 'number' }, value: 'dos mil', error: '' },
      { field: { id: 'd', l: 'Rubro', tp: 'ref', ref: 'rubro', req: true }, value: '3', error: 'viejo' },
    ];
    expect(validarCampos(campos)).toBe(false);
    expect(campos.map(c => c.error)).toEqual(['Campo obligatorio', '11 dígitos', 'Debe ser número', '']);
  });

  it('acepta un formulario completo', () => {
    const campos: EstadoCampo[] = [{ field: { id: 'a', l: 'Nombre', tp: 'text', req: true }, value: 'Volvo', error: '' }];
    expect(validarCampos(campos)).toBe(true);
  });
});

describe('enumerar', () => {
  it('une con comas y una «y» final', () => {
    expect(enumerar([])).toBe('');
    expect(enumerar(['clientes'])).toBe('clientes');
    expect(enumerar(['asignaciones', 'clientes', 'direcciones'])).toBe('asignaciones, clientes y direcciones');
  });
});
