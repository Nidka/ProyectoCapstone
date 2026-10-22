import { describe, expect, it } from 'vitest';
import { normalizar } from './texto';

describe('normalizar', () => {
  it('quita tildes y pasa a minúsculas', () => {
    expect(normalizar('Logística ÁÉÍ Ñandú')).toBe('logistica aei nandu');
  });

  it('trata nulos como texto vacío', () => {
    expect(normalizar(null)).toBe('');
    expect(normalizar(undefined)).toBe('');
  });
});
