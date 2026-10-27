import { describe, expect, it } from 'vitest';
import {
  ACCIONES_PALETA, ENLACES_PRINCIPALES, MODULOS, SECCIONES, accionesPaleta, destinoTrasLogin, destinosPaleta,
  esSeccion, moduloPorClave, moduloPorSlug, rutaDeClave, rutaModulo, seccion, visiblesPara,
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
  it('la barra tiene Inicio, Gastos, Reportes, Operación, Catálogos y Usuarios', () => {
    expect(ENLACES_PRINCIPALES.map(e => e.ruta)).toEqual([
      '/app/inicio', '/app/gastos', '/app/reportes', '/app/operacion', '/app/catalogos', '/app/usuarios',
    ]);
  });

  it('«Usuarios» es solo del administrador, en la barra y en la paleta', () => {
    expect(ENLACES_PRINCIPALES.find(e => e.ruta === '/app/usuarios')?.roles).toEqual(['admin']);
    expect(destinosPaleta().find(d => d.ruta === '/app/usuarios')).toMatchObject({ etiqueta: 'Usuarios', roles: ['admin'] });
  });

  it('la paleta ofrece las acciones del flujo y todos los destinos', () => {
    expect(ACCIONES_PALETA.map(a => a.etiqueta)).toEqual(['Registrar gasto', 'Adjuntar comprobante', 'Revisar comprobantes por aprobar']);
    expect(destinosPaleta()).toHaveLength(4 + MODULOS.length);
    expect(destinosPaleta().find(d => d.etiqueta === 'Reportes')?.ruta).toBe('/app/reportes');
    expect(destinosPaleta().find(d => d.etiqueta === 'Flota')?.detalle).toBe('Operación');
  });
});

describe('visibilidad por rol', () => {
  it('Logística ve todas las acciones del flujo', () => {
    expect(accionesPaleta('logistica').map(a => a.id)).toEqual(['accion-registrar', 'accion-adjuntar', 'accion-revisar']);
  });

  it('el administrador no ve acciones que registran, adjuntan ni revisan', () => {
    expect(accionesPaleta('admin')).toEqual([]);
  });

  it('sin rol conocido no se ofrece ninguna acción', () => {
    expect(accionesPaleta(undefined)).toEqual([]);
    expect(accionesPaleta(null)).toEqual([]);
  });

  it('los enlaces sin roles los ven todos; los que tienen roles, solo esos', () => {
    const items = [{ id: 'a' }, { id: 'b', roles: ['admin'] as const }, { id: 'c', roles: ['logistica'] as const }];
    expect(visiblesPara(items, 'admin').map(i => i.id)).toEqual(['a', 'b']);
    expect(visiblesPara(items, 'logistica').map(i => i.id)).toEqual(['a', 'c']);
    expect(visiblesPara(items, undefined).map(i => i.id)).toEqual(['a']);
  });

  it('la barra principal la ven ambos roles; «Usuarios» solo el administrador', () => {
    expect(visiblesPara(ENLACES_PRINCIPALES, 'admin').map(e => e.etiqueta)).toContain('Usuarios');
    expect(visiblesPara(ENLACES_PRINCIPALES, 'admin')).toHaveLength(6);
    expect(visiblesPara(ENLACES_PRINCIPALES, 'logistica')).toHaveLength(5);
    expect(visiblesPara(ENLACES_PRINCIPALES, 'logistica').map(e => e.etiqueta)).not.toContain('Usuarios');
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
