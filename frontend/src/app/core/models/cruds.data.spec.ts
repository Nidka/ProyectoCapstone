import { describe, expect, it } from 'vitest';
import { CRUDS } from './cruds.data';

const def = (clave: string) => CRUDS.find(c => c.key === clave)!;
const ref = (clave: string, id: unknown) => `${clave}:${id}`;
const VISIBLES = ['ubigeo', 'rubro', 'marca', 'tipo_unidad', 'subtipo_unidad', 'direccion', 'conductor', 'cliente', 'unidad', 'cond_uni', 'viaje'];

describe('cruds.data', () => {
  it('los módulos visibles tienen id, nombre y textos con género', () => {
    for (const clave of VISIBLES) {
      const d = def(clave);
      expect(d.idKey, clave).toBeTruthy();
      expect(d.nombre, clave).toBeTypeOf('function');
      expect(d.textos?.nuevo, clave).toBeTruthy();
    }
    expect(def('direccion').textos?.nuevo).toBe('Nueva dirección');
  });

  it('los campos relacionados se eligen por nombre, no por ID', () => {
    const etiquetas = VISIBLES.flatMap(clave => def(clave).fields.map(f => f.l));
    expect(etiquetas.filter(l => /^ID /.test(l))).toEqual([]);
    expect(def('viaje').fields.filter(f => f.tp === 'ref').map(f => f.ref))
      .toEqual(['unidad', 'unidad', 'cond_uni', 'cliente', 'direccion', 'direccion']);
  });

  it('el número de cada registro se escribe «N°», como en Gastos', () => {
    expect(CRUDS.filter(c => c.cols[0] === '#').map(c => c.key)).toEqual([]);
    expect(def('rubro').row({ id_rubro: 3, nombre: 'Minería' }, ref)[0]).toContain('N° 3');
  });

  it('la unidad de viajes y asignaciones sale en mono y sin cortarse', () => {
    const viaje = def('viaje').row({ id_viaje: 14, id_tracto: 2, id_cliente: 1, fecha_programado: '2026-09-29' }, ref);
    expect(viaje[3]).toBe('<span class="mono sin-corte">unidad:2</span>');
    const asignacion = def('cond_uni').row({ id_conductor_unidad: 1, id_conductor: 4, id_unidad: 2 }, ref);
    expect(asignacion[2]).toBe('<span class="mono sin-corte">unidad:2</span>');
  });

  it('las celdas muestran nombres relacionados y escapan HTML', () => {
    const celdas = def('unidad').row({ placa: 'ABC-123', modelo: '<b>x</b>', id_marca: 2, id_tipo_unidad: 1, anio: 2020 }, ref);
    expect(celdas[1]).toContain('&lt;b&gt;x&lt;/b&gt;');
    expect(celdas[2]).toContain('marca:2');
    expect(celdas[3]).toContain('tipo_unidad:1');
  });

  it('los textos visibles de los módulos no usan raya larga ni emojis', () => {
    for (const clave of VISIBLES) {
      const d = def(clave);
      const visible = `${d.title} ${d.desc} ${d.cols.join(' ')} ${d.fields.map(f => f.l).join(' ')}`;
      expect(visible, clave).not.toMatch(/[—–]|\p{Extended_Pictographic}/u);
    }
  });

  it('el ubigeo muestra el código con seis dígitos', () => {
    expect(def('ubigeo').nombre!({ codigo_ubigeo: 10101, distrito: 'Chachapoyas', provincia: 'Chachapoyas' }, ref))
      .toBe('Chachapoyas, Chachapoyas (010101)');
  });
});
