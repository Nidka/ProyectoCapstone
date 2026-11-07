import { describe, expect, it } from 'vitest';
import { API_LOCAL, API_PRODUCCION, origenApi } from './api-origen';

describe('origenApi', () => {
  it('en localhost y 127.0.0.1 usa la API local del 8081', () => {
    expect(origenApi('localhost')).toBe(API_LOCAL);
    expect(origenApi('127.0.0.1')).toBe(API_LOCAL);
  });

  it('en cualquier otro dominio usa la API desplegada, por https', () => {
    expect(origenApi('lizar.vercel.app')).toBe(API_PRODUCCION);
    expect(API_PRODUCCION.startsWith('https://')).toBe(true);
    expect(API_PRODUCCION.endsWith('/')).toBe(false);
  });
});
