import { describe, expect, it } from 'vitest';
import { Comprobante, Gasto } from '../models/gasto.model';
import {
  buscarGastos, comprobantesDe, contarPorEstado, esPestana, estadoDePestana, etiquetaRuta, filtrarPorPestana,
  filtrarPorViajeYCliente, idDeConsulta, ordenarRecientes, pasosRuta, recientes, siguientePaso, sumaPorEstado, totalMonto, ultimoRechazo,
} from './gastos.logic';

function gasto(p: Partial<Gasto> & Pick<Gasto, 'id' | 'estado'>): Gasto {
  return {
    viaje_id: 1, cliente_id: 1, monto: 100, fecha: '2026-10-01T00:00:00Z',
    creado_por: 'Ana Quispe', creado_en: '2026-10-01T15:00:00Z', ...p,
  };
}

function comprobante(p: Partial<Comprobante> & Pick<Comprobante, 'id' | 'gasto_id' | 'estado'>): Comprobante {
  return { archivo_url: `/uploads/${p.id}.jpg`, tipo_archivo: 'imagen', creado_en: '2026-10-01T16:00:00Z', ...p };
}

const gastos = [
  gasto({ id: 1, estado: 'Registrado', monto: 120 }),
  gasto({ id: 2, estado: 'Pagado', monto: 85.5 }),
  gasto({ id: 3, estado: 'Sustentado', monto: 310 }),
  gasto({ id: 4, estado: 'Registrado', monto: 40.25, cliente_id: 2 }),
];

describe('pestañas', () => {
  it('reconoce solo las pestañas válidas', () => {
    expect(esPestana('por-aprobar')).toBe(true);
    expect(esPestana('otra')).toBe(false);
    expect(esPestana(undefined)).toBe(false);
  });

  it('filtra por el estado de cada pestaña', () => {
    expect(filtrarPorPestana(gastos, 'todos').map(g => g.id)).toEqual([1, 2, 3, 4]);
    expect(filtrarPorPestana(gastos, 'sin-comprobante').map(g => g.id)).toEqual([1, 4]);
    expect(filtrarPorPestana(gastos, 'por-aprobar').map(g => g.id)).toEqual([2]);
    expect(filtrarPorPestana(gastos, 'sustentados').map(g => g.id)).toEqual([3]);
  });
});

describe('conteos y montos', () => {
  it('sin gastos mantiene los tres estados con conteos y montos en cero', () => {
    expect(contarPorEstado([])).toEqual({ Registrado: 0, Pagado: 0, Sustentado: 0 });
    expect(sumaPorEstado([])).toEqual({ Registrado: 0, Pagado: 0, Sustentado: 0 });
    expect(totalMonto([])).toBe(0);
    for (const pestana of ['sin-comprobante', 'por-aprobar', 'sustentados'] as const) {
      expect(filtrarPorPestana([], pestana)).toEqual([]);
    }
  });

  it('cuenta y suma por estado', () => {
    expect(contarPorEstado(gastos)).toEqual({ Registrado: 2, Pagado: 1, Sustentado: 1 });
    expect(sumaPorEstado(gastos)).toEqual({ Registrado: 160.25, Pagado: 85.5, Sustentado: 310 });
    expect(totalMonto(gastos)).toBeCloseTo(555.75);
  });
});

describe('siguientePaso', () => {
  it('señala la acción que toca', () => {
    expect(siguientePaso({ estado: 'Registrado' })).toBe('adjuntar');
    expect(siguientePaso({ estado: 'Pagado' })).toBe('revisar');
    expect(siguientePaso({ estado: 'Sustentado' })).toBe('ninguno');
  });
});

describe('comprobantes de un gasto', () => {
  const lista = [
    comprobante({ id: 10, gasto_id: 1, estado: 'Rechazado', creado_en: '2026-10-01T16:00:00Z', motivo_rechazo: 'Ilegible' }),
    comprobante({ id: 11, gasto_id: 1, estado: 'Pendiente', creado_en: '2026-10-02T09:00:00-05:00' }),
    comprobante({ id: 12, gasto_id: 2, estado: 'Pendiente' }),
  ];

  it('ordena del más reciente al más antiguo', () => {
    expect(comprobantesDe(lista, 1).map(c => c.id)).toEqual([11, 10]);
  });

  it('solo devuelve el rechazo si es el último comprobante', () => {
    expect(ultimoRechazo(lista, 1)).toBeUndefined();
    expect(ultimoRechazo(lista.slice(0, 1), 1)?.motivo_rechazo).toBe('Ilegible');
  });
});

describe('pasosRuta', () => {
  it('un gasto registrado está en el primer hito con fecha y autor', () => {
    const pasos = pasosRuta(gastos[0], []);
    expect(pasos.map(p => [p.estado, p.alcanzado, p.actual])).toEqual([
      ['Registrado', true, true], ['Pagado', false, false], ['Sustentado', false, false],
    ]);
    expect(pasos[0].fecha).toBe('2026-10-01T15:00:00Z');
    expect(pasos[0].actor).toBe('Ana Quispe');
  });

  it('un gasto sustentado toma las fechas del comprobante aprobado', () => {
    const aprobado = comprobante({
      id: 20, gasto_id: 3, estado: 'Aprobado', creado_en: '2026-10-01T17:00:00Z',
      aprobado_en: '2026-10-02T10:00:00Z', aprobado_por: 'Encargado de Logística',
    });
    const pasos = pasosRuta(gastos[2], [aprobado]);
    expect(pasos.map(p => p.actual)).toEqual([false, false, true]);
    expect(pasos[1].fecha).toBe('2026-10-01T17:00:00Z');
    expect(pasos[2].fecha).toBe('2026-10-02T10:00:00Z');
    expect(pasos[2].actor).toBe('Encargado de Logística');
  });

  it('describe el recorrido para lectores de pantalla', () => {
    const pasos = pasosRuta(gastos[1], [comprobante({ id: 30, gasto_id: 2, estado: 'Pendiente' })]);
    expect(etiquetaRuta(pasos)).toBe('Recorrido: Registrado, Pagado (actual), Sustentado pendiente');
  });
});

describe('orden y búsqueda', () => {
  const conFechas = [
    gasto({ id: 5, estado: 'Registrado', creado_en: '2026-10-01T10:00:00Z' }),
    gasto({ id: 6, estado: 'Registrado', creado_en: '2026-10-03T10:00:00Z' }),
    gasto({ id: 7, estado: 'Registrado', creado_en: '2026-10-03T10:00:00Z' }),
  ];

  it('ordena por registro más reciente y luego por número', () => {
    expect(ordenarRecientes(conFechas).map(g => g.id)).toEqual([7, 6, 5]);
    expect(recientes(conFechas, 2).map(g => g.id)).toEqual([7, 6]);
  });

  const nombres: Record<number, string> = { 1: 'Agro Norte SAC', 2: 'Logística Andina' };
  const cliente = (id: number) => nombres[id] ?? '';

  it('busca por número con o sin prefijo', () => {
    expect(buscarGastos(gastos, '2', cliente).map(g => g.id)).toEqual([2]);
    expect(buscarGastos(gastos, 'N° 3', cliente).map(g => g.id)).toEqual([3]);
    expect(buscarGastos(gastos, '#4', cliente).map(g => g.id)).toEqual([4]);
  });

  it('busca por cliente sin importar tildes', () => {
    expect(buscarGastos(gastos, 'agro', cliente).map(g => g.id)).toEqual([1, 2, 3]);
    expect(buscarGastos(gastos, 'logistica', cliente).map(g => g.id)).toEqual([4]);
    expect(buscarGastos(gastos, '   ', cliente)).toHaveLength(4);
  });
});

describe('filtros por viaje y cliente', () => {
  const g = (id: number, viaje: number, cliente: number): Gasto => ({
    id, viaje_id: viaje, cliente_id: cliente, monto: 10, fecha: '2026-10-01', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T10:00:00Z',
  });
  const lista = [g(1, 1, 1), g(2, 1, 2), g(3, 2, 1), g(4, 2, 2)];

  it('filtra por viaje, por cliente o por ambos, y sin filtro devuelve todo', () => {
    expect(filtrarPorViajeYCliente(lista, {}).map(x => x.id)).toEqual([1, 2, 3, 4]);
    expect(filtrarPorViajeYCliente(lista, { viaje: 1 }).map(x => x.id)).toEqual([1, 2]);
    expect(filtrarPorViajeYCliente(lista, { cliente: 1 }).map(x => x.id)).toEqual([1, 3]);
    expect(filtrarPorViajeYCliente(lista, { viaje: 2, cliente: 2 }).map(x => x.id)).toEqual([4]);
    expect(filtrarPorViajeYCliente(lista, { viaje: 9 })).toEqual([]);
  });

  it('lee el id de la URL solo si es un entero positivo', () => {
    expect(idDeConsulta('14')).toBe(14);
    for (const malo of [undefined, null, '', '0', '-3', 'abc', '1.5', '12x']) expect(idDeConsulta(malo)).toBeNull();
  });

  it('traduce la pestaña a su estado, y «todos» a ninguno', () => {
    expect(estadoDePestana('todos')).toBeNull();
    expect(estadoDePestana('sin-comprobante')).toBe('Registrado');
    expect(estadoDePestana('por-aprobar')).toBe('Pagado');
    expect(estadoDePestana('sustentados')).toBe('Sustentado');
  });
});
