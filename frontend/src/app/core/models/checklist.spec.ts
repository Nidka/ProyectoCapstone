import { describe, expect, it } from 'vitest';
import {
  CRITERIOS_CHECKLIST, Checklist, checklistCompleto, checklistVacio, cortoDe, cumplidos, motivoDesdeChecklist,
} from './checklist';
import { Gasto } from './gasto.model';

const gasto: Gasto = {
  id: 30, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T13:00:00Z',
};

function con(valores: Partial<Checklist>): Checklist {
  return { ...checklistVacio(), ...valores };
}

describe('checklist documental', () => {
  it('las claves coinciden con gastos-service, en el mismo orden', () => {
    expect(CRITERIOS_CHECKLIST.map(c => c.clave)).toEqual(['legible', 'tipo_valido', 'a_nombre_empresa', 'monto_coincide', 'fecha_coincide']);
  });

  it('los criterios de monto y fecha muestran el dato del gasto', () => {
    const textos = CRITERIOS_CHECKLIST.map(c => c.texto(gasto));
    expect(textos).toEqual([
      'El comprobante es legible y está completo.',
      'Es una boleta o factura válida.',
      'Está emitido a nombre de LIZAR SRL.',
      'El monto coincide con el gasto (S/ 85.50).',
      'La fecha corresponde al gasto (30/09/2026).',
    ]);
  });

  it('cuenta lo marcado y solo está completo con los 5', () => {
    expect(cumplidos(checklistVacio())).toBe(0);
    expect(checklistCompleto(checklistVacio())).toBe(false);
    const cuatro = con({ legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true });
    expect(cumplidos(cuatro)).toBe(4);
    expect(checklistCompleto(cuatro)).toBe(false);
    expect(checklistCompleto({ ...cuatro, fecha_coincide: true })).toBe(true);
  });

  it('propone el motivo con los criterios sin marcar, y nada si cumple todos', () => {
    expect(motivoDesdeChecklist(con({ legible: true, tipo_valido: true, a_nombre_empresa: true }))).toBe('No cumple: monto coincide; fecha coincide.');
    expect(motivoDesdeChecklist(con({ monto_coincide: true, fecha_coincide: true, legible: true }))).toBe(
      'No cumple: boleta o factura válida; a nombre de LIZAR SRL.',
    );
    const todo = con({ legible: true, tipo_valido: true, a_nombre_empresa: true, monto_coincide: true, fecha_coincide: true });
    expect(motivoDesdeChecklist(todo)).toBe('');
  });

  it('el texto corto sirve para el historial', () => {
    expect(cortoDe('monto_coincide')).toBe('Monto coincide');
    expect(cortoDe('a_nombre_empresa')).toBe('A nombre de LIZAR SRL');
  });
});
