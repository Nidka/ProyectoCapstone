import { describe, expect, it } from 'vitest';
import { textoInsignia, tituloAviso } from './notificacion';

describe('notificacion', () => {
  it('cada tipo de aviso tiene su título en español sencillo', () => {
    expect(tituloAviso('sin_comprobante')).toBe('Falta el comprobante');
    expect(tituloAviso('cobro_proximo')).toBe('Cobro cerca');
  });

  it('un tipo que esta pantalla no conoce sale como «Aviso», no crudo', () => {
    expect(tituloAviso('otro_tipo')).toBe('Aviso');
    expect(tituloAviso('constructor')).toBe('Aviso');
  });

  it('la insignia muestra el número y se corta en 99+', () => {
    expect(textoInsignia(3)).toBe('3');
    expect(textoInsignia(99)).toBe('99');
    expect(textoInsignia(100)).toBe('99+');
  });
});
