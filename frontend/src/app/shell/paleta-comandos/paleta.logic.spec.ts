import { describe, expect, it } from 'vitest';
import { resultadosPaleta } from './paleta.logic';
import { Gasto } from '../../core/models/gasto.model';

const gastos: Gasto[] = [
  { id: 31, estado: 'Registrado', monto: 120, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-01T12:00:00Z' },
  { id: 32, estado: 'Pagado', monto: 85.5, viaje_id: 1, cliente_id: 2, fecha: '2026-10-02T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-02T12:00:00Z' },
];
const nombre = (id: number) => (id === 1 ? 'Agro Norte SAC' : 'Logística Andina');

describe('resultadosPaleta', () => {
  it('sin término ofrece las acciones y todos los destinos, sin gastos', () => {
    const r = resultadosPaleta('', gastos, nombre, 'logistica');
    expect(r.filter(e => e.grupo === 'Acciones')).toHaveLength(3);
    expect(r.filter(e => e.grupo === 'Ir a')).toHaveLength(14);
    expect(r.some(e => e.id === 'ir-usuarios')).toBe(false);
    expect(r.some(e => e.grupo === 'Gastos')).toBe(false);
  });

  it('al administrador no le ofrece registrar, adjuntar ni revisar, pero sí ir a las pantallas de consulta', () => {
    const r = resultadosPaleta('', gastos, nombre, 'admin');
    expect(r.some(e => e.grupo === 'Acciones')).toBe(false);
    expect(r.filter(e => e.grupo === 'Ir a')).toHaveLength(15);
    expect(resultadosPaleta('usuarios', gastos, nombre, 'admin').map(e => e.ruta)).toEqual(['/app/usuarios']);
    expect(resultadosPaleta('usuarios', gastos, nombre, 'logistica')).toEqual([]);
    expect(resultadosPaleta('registrar', gastos, nombre, 'admin')).toEqual([]);
    expect(resultadosPaleta('registrar', gastos, nombre, 'logistica').map(e => e.id)).toEqual(['accion-registrar']);
  });

  it('encuentra módulos sin importar tildes', () => {
    expect(resultadosPaleta('flota', gastos, nombre, 'logistica').map(e => e.ruta)).toEqual(['/app/operacion/flota']);
    expect(resultadosPaleta('direccion', gastos, nombre, 'logistica').map(e => e.etiqueta)).toEqual(['Direcciones']);
  });

  it('encuentra gastos por número y por cliente', () => {
    expect(resultadosPaleta('31', gastos, nombre, 'logistica')).toEqual([
      { id: 'gasto-31', grupo: 'Gastos', etiqueta: 'Gasto N° 31', detalle: 'Agro Norte SAC · S/ 120.00', icono: 'receipt', ruta: '/app/gastos/31' },
    ]);
    expect(resultadosPaleta('andina', gastos, nombre, 'logistica').map(e => e.id)).toEqual(['gasto-32']);
  });
});
