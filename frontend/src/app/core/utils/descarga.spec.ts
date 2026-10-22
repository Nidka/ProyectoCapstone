import { afterEach, describe, expect, it, vi } from 'vitest';
import { guardarArchivo } from './descarga';

describe('guardarArchivo', () => {
  afterEach(() => vi.restoreAllMocks());

  it('crea un enlace temporal con el nombre del archivo y lo pulsa', () => {
    // jsdom no implementa URL.createObjectURL
    const crear = vi.fn(() => 'blob:prueba');
    const revocar = vi.fn();
    Object.assign(URL, { createObjectURL: crear, revokeObjectURL: revocar });
    vi.useFakeTimers();
    let pulsado: HTMLAnchorElement | undefined;
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      pulsado = this;
    });
    const blob = new Blob(['%PDF'], { type: 'application/pdf' });

    guardarArchivo(blob, 'reporte.pdf');

    expect(crear).toHaveBeenCalledWith(blob);
    expect(pulsado?.download).toBe('reporte.pdf');
    expect(pulsado?.href).toBe('blob:prueba');
    expect(document.querySelector('a[download]')).toBeNull();
    // Safari en iOS lee el blob después de confirmar la descarga: la URL debe seguir viva un rato
    vi.advanceTimersByTime(1000);
    expect(revocar).not.toHaveBeenCalled();
    vi.advanceTimersByTime(60_000);
    expect(revocar).toHaveBeenCalledWith('blob:prueba');
    vi.useRealTimers();
  });
});
