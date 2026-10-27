import { describe, expect, it } from 'vitest';
import { Gasto } from './gasto.model';
import { LecturaComprobante, clavesSugeridas, etiquetaSugerencia, filasLectura, motivoDesdeLectura } from './lectura';

const lectura = (estado: LecturaComprobante['estado'], sugerencias: LecturaComprobante['sugerencias']): LecturaComprobante => ({
  estado,
  datos: {
    monto: { valor: 0, texto: '', confianza: 'ninguna' },
    fecha: { valor: '', texto: '', confianza: 'ninguna' },
    rucs: [],
    tipo: { valor: 'otro', texto: '', confianza: 'ninguna' },
  },
  sugerencias,
});

const gasto: Gasto = {
  id: 7, viaje_id: 14, cliente_id: 1, monto: 100, fecha: '2026-10-24T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-24T13:00:00Z',
};
const estandar = [
  { clave: 'legible', texto: 'Legible' },
  { clave: 'tipo_valido', texto: 'Tipo' },
  { clave: 'a_nombre_empresa', texto: 'Empresa' },
  { clave: 'monto_coincide', texto: 'Monto' },
  { clave: 'fecha_coincide', texto: 'Fecha' },
];

describe('lectura del comprobante', () => {
  it('rotula cada sugerencia: coincide, revisar o nada', () => {
    expect(etiquetaSugerencia({ sugerido: true, detalle: 'Leído S/ 85.50' })).toBe('✓ Coincide');
    expect(etiquetaSugerencia({ sugerido: false, detalle: 'Leída 29/09/2026' })).toBe('⚠ Revisar');
    expect(etiquetaSugerencia({ sugerido: false, detalle: '' })).toBe('⚠ Revisar');
    expect(etiquetaSugerencia(undefined)).toBe('');
  });

  it('sugiere solo criterios del cliente, en su orden, y solo con una lectura leída', () => {
    const criterios = [
      { clave: 'fecha_coincide', texto: 'Fecha' },
      { clave: 'legible', texto: 'Legible' },
      { clave: 'monto_coincide', texto: 'Monto' },
    ];
    const sugerencias = {
      monto_coincide: { sugerido: true, detalle: 'Leído S/ 85.50' },
      fecha_coincide: { sugerido: true, detalle: 'Leída 30/09/2026' },
      tipo_valido: { sugerido: true, detalle: 'Boleta' },
    };
    expect(clavesSugeridas(criterios, lectura('leido', sugerencias))).toEqual(['fecha_coincide', 'monto_coincide']);
    expect(clavesSugeridas(criterios, lectura('sin_texto', sugerencias))).toEqual([]);
    expect(clavesSugeridas(criterios, undefined)).toEqual([]);
  });

  describe('filasLectura: gasto frente a comprobante', () => {
    const sugerencias = {
      tipo_valido: { sugerido: true, detalle: 'Factura' },
      a_nombre_empresa: { sugerido: false, detalle: '' },
      monto_coincide: { sugerido: false, detalle: 'Leído S/ 513.00' },
      fecha_coincide: { sugerido: false, detalle: 'Leída 26/09/2019' },
    };

    it('compara monto, fecha, empresa y documento, en ese orden, con lo leído sin el prefijo', () => {
      expect(filasLectura(estandar, lectura('leido', sugerencias), gasto)).toEqual([
        { clave: 'monto_coincide', dato: 'Monto', gasto: 'S/ 100.00', leido: 'S/ 513.00', leyo: true, coincide: false },
        { clave: 'fecha_coincide', dato: 'Fecha', gasto: '24/10/2026', leido: '26/09/2019', leyo: true, coincide: false },
        { clave: 'a_nombre_empresa', dato: 'Empresa', gasto: 'LIZAR SRL', leido: 'No se encontró', leyo: false, coincide: false },
        { clave: 'tipo_valido', dato: 'Documento', gasto: 'Factura o boleta', leido: 'Factura', leyo: true, coincide: true },
      ]);
    });

    it('lo que no se leyó lo dice, y lo que coincide se marca', () => {
      const filas = filasLectura(
        estandar,
        lectura('leido', {
          monto_coincide: { sugerido: true, detalle: 'Leído S/ 100.00' },
          fecha_coincide: { sugerido: false, detalle: '' },
          tipo_valido: { sugerido: false, detalle: '' },
          a_nombre_empresa: { sugerido: true, detalle: 'RUC 20481608750' },
        }),
        gasto,
      );
      expect(filas.map(f => [f.clave, f.leido, f.coincide])).toEqual([
        ['monto_coincide', 'S/ 100.00', true],
        ['fecha_coincide', 'No se leyó', false],
        ['a_nombre_empresa', 'RUC 20481608750', true],
        ['tipo_valido', 'No es factura ni boleta', false],
      ]);
    });

    it('solo incluye criterios que el cliente revisa y que la lectura comenta; sin lectura leída no hay filas', () => {
      const soloMonto = [{ clave: 'monto_coincide', texto: 'Monto' }, { clave: 'legible', texto: 'Legible' }];
      expect(filasLectura(soloMonto, lectura('leido', sugerencias), gasto).map(f => f.clave)).toEqual(['monto_coincide']);
      expect(filasLectura(estandar, lectura('leido', {}), gasto)).toEqual([]);
      expect(filasLectura(estandar, lectura('error', sugerencias), gasto)).toEqual([]);
      expect(filasLectura(estandar, undefined, gasto)).toEqual([]);
    });
  });

  describe('filasLectura: lo que se puede corregir', () => {
    const conDatos = (monto: Partial<LecturaComprobante['datos']['monto']>, fecha: Partial<LecturaComprobante['datos']['fecha']>) => ({
      ...lectura('leido', {
        monto_coincide: { sugerido: false, detalle: 'Leído S/ 513.00' },
        fecha_coincide: { sugerido: false, detalle: 'Leída 26/09/2019' },
        a_nombre_empresa: { sugerido: false, detalle: '' },
        tipo_valido: { sugerido: false, detalle: '' },
      }),
      datos: {
        monto: { valor: 513, texto: 'TOTAL S/ 513.00', confianza: 'alta' as const, ...monto },
        fecha: { valor: '2019-09-26', texto: 'FECHA 26/09/2019', confianza: 'alta' as const, ...fecha },
        rucs: [],
        tipo: { valor: 'otro', texto: '', confianza: 'ninguna' as const },
      },
    });

    it('el monto y la fecha leídos que no coinciden se pueden corregir, con el valor leído', () => {
      const filas = filasLectura(estandar, conDatos({}, {}), gasto);
      expect(filas.find(f => f.clave === 'monto_coincide')?.corregible).toEqual({ campo: 'monto', valor: 513 });
      expect(filas.find(f => f.clave === 'fecha_coincide')?.corregible).toEqual({ campo: 'fecha', valor: '2019-09-26' });
    });

    it('empresa y documento no se corrigen: eso se resuelve con otro comprobante', () => {
      const filas = filasLectura(estandar, conDatos({}, {}), gasto);
      expect(filas.find(f => f.clave === 'a_nombre_empresa')?.corregible).toBeUndefined();
      expect(filas.find(f => f.clave === 'tipo_valido')?.corregible).toBeUndefined();
    });

    it('lo que coincide, lo que no se leyó y un valor dudoso no ofrecen corregir', () => {
      const coincide = { ...conDatos({}, {}), sugerencias: { ...conDatos({}, {}).sugerencias, monto_coincide: { sugerido: true, detalle: 'Leído S/ 513.00' } } };
      expect(filasLectura(estandar, coincide, gasto).find(f => f.clave === 'monto_coincide')?.corregible).toBeUndefined();
      const sinLeer = filasLectura(estandar, conDatos({ confianza: 'ninguna', valor: 0 }, { confianza: 'ninguna', valor: '' }), gasto);
      expect(sinLeer.find(f => f.clave === 'monto_coincide')?.corregible).toBeUndefined();
      expect(sinLeer.find(f => f.clave === 'fecha_coincide')?.corregible).toBeUndefined();
      // Un monto leído con confianza media sí se ofrece: la persona lo confirma antes de guardar
      expect(filasLectura(estandar, conDatos({ confianza: 'media' }, {}), gasto).find(f => f.clave === 'monto_coincide')?.corregible).toEqual({ campo: 'monto', valor: 513 });
    });
  });

  describe('motivoDesdeLectura: el rechazo redactado con lo que leyó el comprobante', () => {
    const filas = (sugerencias: LecturaComprobante['sugerencias']) => filasLectura(estandar, lectura('leido', sugerencias), gasto);

    it('nombra cada diferencia con lo leído y lo del gasto', () => {
      const f = filas({
        monto_coincide: { sugerido: false, detalle: 'Leído S/ 513.00' },
        fecha_coincide: { sugerido: false, detalle: 'Leída 26/09/2019' },
        a_nombre_empresa: { sugerido: false, detalle: '' },
        tipo_valido: { sugerido: false, detalle: '' },
      });
      expect(motivoDesdeLectura(f)).toBe(
        'No cumple: el monto leído (S/ 513.00) no coincide con el del gasto (S/ 100.00); ' +
          'la fecha leída (26/09/2019) no coincide con la del gasto (24/10/2026); ' +
          'no aparece a nombre de LIZAR SRL; no es una factura ni una boleta.',
      );
    });

    it('omite lo que coincide y lo que no se pudo leer (no prueba nada), y sin diferencias no propone nada', () => {
      const f = filas({
        monto_coincide: { sugerido: true, detalle: 'Leído S/ 100.00' },
        fecha_coincide: { sugerido: false, detalle: '' },
        a_nombre_empresa: { sugerido: false, detalle: '' },
        tipo_valido: { sugerido: true, detalle: 'Factura' },
      });
      expect(motivoDesdeLectura(f)).toBe('No cumple: no aparece a nombre de LIZAR SRL.');
      expect(motivoDesdeLectura(filas({ monto_coincide: { sugerido: true, detalle: 'Leído S/ 100.00' } }))).toBe('');
      expect(motivoDesdeLectura([])).toBe('');
    });
  });
});
