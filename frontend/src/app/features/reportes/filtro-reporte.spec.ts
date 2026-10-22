import { describe, expect, it } from 'vitest';
import {
  MSG_FECHA_INVALIDA, MSG_RANGO_INVERTIDO, MSG_RANGO_LARGO, atajosActivos, errorDeFiltro, fechaValida, filtroDesdeConsulta,
  mesCompleto, nombreMes, rangoDeAtajo, rangoDelMes,
} from './filtro-reporte';

const HOY = '2026-10-17';

describe('filtro de reportes', () => {
  it('los atajos calculan su rango desde hoy', () => {
    expect(rangoDeAtajo('ultimos-12', HOY)).toEqual({ desde: '2025-11-01', hasta: HOY });
    expect(rangoDeAtajo('este-anio', HOY)).toEqual({ desde: '2026-01-01', hasta: HOY });
    expect(rangoDeAtajo('este-mes', HOY)).toEqual({ desde: '2026-10-01', hasta: HOY });
    expect(rangoDeAtajo('ultimos-12', '2026-01-05')).toEqual({ desde: '2025-02-01', hasta: '2026-01-05' });
  });

  it('reconoce los atajos activos, también cuando dos coinciden', () => {
    expect(atajosActivos({ desde: '2026-10-01', hasta: HOY }, HOY)).toEqual(['este-mes']);
    expect(atajosActivos({ desde: '2026-03-01', hasta: '2026-03-31' }, HOY)).toEqual([]);
    // En enero, «Este año» y «Este mes» son el mismo rango
    expect(atajosActivos({ desde: '2026-01-01', hasta: '2026-01-05' }, '2026-01-05')).toEqual(['este-anio', 'este-mes']);
  });

  it('sin parámetros usa los mismos valores por defecto que el servidor', () => {
    expect(filtroDesdeConsulta(null, null, HOY)).toEqual({ desde: '2025-11-01', hasta: HOY });
    expect(filtroDesdeConsulta(undefined, '2026-03-15', HOY)).toEqual({ desde: '2025-04-01', hasta: '2026-03-15' });
    expect(filtroDesdeConsulta('2026-01-01', null, HOY)).toEqual({ desde: '2026-01-01', hasta: HOY });
  });

  it('valida fechas reales', () => {
    expect(fechaValida('2026-02-28')).toBe(true);
    expect(fechaValida('2026-02-30')).toBe(false);
    expect(fechaValida('17/10/2026')).toBe(false);
  });

  it('los errores coinciden con gastos-service, incluido el límite de 731 días', () => {
    expect(errorDeFiltro({ desde: '2026-13-01', hasta: HOY })).toBe(MSG_FECHA_INVALIDA);
    expect(errorDeFiltro({ desde: '2026-05-02', hasta: '2026-05-01' })).toBe(MSG_RANGO_INVERTIDO);
    expect(errorDeFiltro({ desde: '2024-01-01', hasta: '2025-12-31' })).toBeNull();
    expect(errorDeFiltro({ desde: '2024-01-01', hasta: '2026-01-01' })).toBe(MSG_RANGO_LARGO);
  });

  it('el rango de un mes va del día 1 al último día', () => {
    expect(rangoDelMes('2026-02')).toEqual({ desde: '2026-02-01', hasta: '2026-02-28' });
    expect(rangoDelMes('2025-12')).toEqual({ desde: '2025-12-01', hasta: '2025-12-31' });
    expect(mesCompleto({ desde: '2026-02-01', hasta: '2026-02-28' })).toBe('2026-02');
    expect(mesCompleto({ desde: '2026-02-01', hasta: '2026-02-27' })).toBeNull();
    expect(nombreMes('2026-10')).toBe('octubre 2026');
  });
});
