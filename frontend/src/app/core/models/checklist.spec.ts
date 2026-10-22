import { describe, expect, it } from 'vitest';
import {
  CriterioCliente, checklistCompleto, checklistVacio, cumplidos, motivoDesdeChecklist, textoConDato,
} from './checklist';
import { Gasto } from './gasto.model';

const gasto: Gasto = {
  id: 30, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T13:00:00Z',
};

const criterios: CriterioCliente[] = [
  { clave: 'legible', texto: 'El comprobante es legible y está completo.' },
  { clave: 'monto_coincide', texto: 'El monto coincide con el gasto.' },
  { clave: 'fecha_coincide', texto: 'La fecha corresponde al gasto.' },
  { clave: 'c-a1b2c3', texto: 'Lleva sello de recepción.' },
];

describe('checklist documental por cliente', () => {
  it('parte vacío, con una entrada por criterio del cliente', () => {
    expect(checklistVacio(criterios)).toEqual({ legible: false, monto_coincide: false, fecha_coincide: false, 'c-a1b2c3': false });
    expect(checklistVacio([])).toEqual({});
  });

  it('cuenta lo marcado y solo está completo con todos los criterios (y al menos uno)', () => {
    const vacio = checklistVacio(criterios);
    expect(cumplidos(criterios, vacio)).toBe(0);
    expect(checklistCompleto(criterios, vacio)).toBe(false);
    const casi = { ...vacio, legible: true, monto_coincide: true, fecha_coincide: true };
    expect(cumplidos(criterios, casi)).toBe(3);
    expect(checklistCompleto(criterios, casi)).toBe(false);
    expect(checklistCompleto(criterios, { ...casi, 'c-a1b2c3': true })).toBe(true);
    // Sin criterios no hay nada que cumplir: no se considera completo (no se puede aprobar sin checklist)
    expect(checklistCompleto([], {})).toBe(false);
  });

  it('propone el motivo con los criterios sin marcar, y nada si cumple todos', () => {
    const c = { ...checklistVacio(criterios), legible: true, 'c-a1b2c3': true };
    expect(motivoDesdeChecklist(criterios, c)).toBe('No cumple: el monto coincide con el gasto; la fecha corresponde al gasto.');
    const todo = Object.fromEntries(criterios.map(x => [x.clave, true]));
    expect(motivoDesdeChecklist(criterios, todo)).toBe('');
  });

  it('el monto y la fecha del gasto se muestran junto a sus criterios, para comparar', () => {
    expect(textoConDato(criterios[0], gasto)).toBe('El comprobante es legible y está completo.');
    expect(textoConDato(criterios[1], gasto)).toBe('El monto coincide con el gasto (S/ 85.50).');
    expect(textoConDato(criterios[2], gasto)).toBe('La fecha corresponde al gasto (30/09/2026).');
    expect(textoConDato(criterios[3], gasto)).toBe('Lleva sello de recepción.');
  });
});
