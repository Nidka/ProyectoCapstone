import { describe, expect, it } from 'vitest';
import {
  ACCIONES_PALETA, ENLACES_PRINCIPALES, MODULOS, SECCIONES, destinoTrasLogin, destinosPaleta,
  esSeccion, moduloPorClave, moduloPorSlug, rutaDeClave, rutaModulo, seccion,
} from './navegacion';

describe('módulos visibles', () => {
  it('cubre exactamente las 11 claves CRUD de esta iteración', () => {
    expect(MODULOS.map(m => m.clave).sort()).toEqual(
      ['cliente', 'cond_uni', 'conductor', 'direccion', 'marca', 'rubro', 'subtipo_unidad', 'tipo_unidad', 'ubigeo', 'unidad', 'viaje'],
    );
  });

  it('no repite slugs dentro de una sección', () => {
    for (const s of SECCIONES) {
      const slugs = s.modulos.map(m => m.slug);
      expect(new Set(slugs).size).toBe(slugs.length);
    }
  });

  it('resuelve módulos por slug y por clave', () => {
    expect(moduloPorSlug('operacion', 'flota')?.clave).toBe('unidad');
    expect(moduloPorSlug('catalogos', 'flota')).toBeUndefined();
    expect(moduloPorClave('cond_uni')?.slug).toBe('asignaciones');
    expect(seccion('catalogos').modulos[0].slug).toBe('ubigeo');
    expect(esSeccion('operacion')).toBe(true);
    expect(esSeccion('gastos')).toBe(false);
  });

  it('arma las rutas de cada módulo', () => {
    expect(rutaModulo(moduloPorClave('viaje')!)).toBe('/app/operacion/viajes');
    expect(rutaDeClave('tipo_unidad')).toBe('/app/catalogos/tipos-unidad');
    expect(rutaDeClave('factura')).toBeNull();
  });
});

describe('enlaces y paleta', () => {
  it('la barra tiene Inicio, Gastos, Reportes, Operación y Catálogos', () => {
    expect(ENLACES_PRINCIPALES.map(e => e.ruta)).toEqual(['/app/inicio', '/app/gastos', '/app/reportes', '/app/operacion', '/app/catalogos']);
  });

  it('la paleta ofrece las acciones del flujo y todos los destinos', () => {
    expect(ACCIONES_PALETA.map(a => a.etiqueta)).toEqual(['Registrar gasto', 'Adjuntar comprobante', 'Revisar comprobantes por aprobar']);
    expect(destinosPaleta()).toHaveLength(3 + MODULOS.length);
    expect(destinosPaleta().find(d => d.etiqueta === 'Reportes')?.ruta).toBe('/app/reportes');
    expect(destinosPaleta().find(d => d.etiqueta === 'Flota')?.detalle).toBe('Operación');
  });
});

describe('destinoTrasLogin', () => {
  it('vuelve a la pantalla pedida solo si es de la app', () => {
    expect(destinoTrasLogin('/app/gastos?estado=por-aprobar')).toBe('/app/gastos?estado=por-aprobar');
    expect(destinoTrasLogin('https://otro.sitio')).toBe('/app/inicio');
    expect(destinoTrasLogin('/app')).toBe('/app/inicio');
    expect(destinoTrasLogin(null)).toBe('/app/inicio');
  });
});
