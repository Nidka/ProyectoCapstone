import { describe, expect, it } from 'vitest';
import { iniciales } from './menu-usuario.component';

describe('iniciales', () => {
  it('usa las iniciales de nombre y apellido, o las dos primeras letras', () => {
    expect(iniciales('Ana Quispe')).toBe('AQ');
    expect(iniciales('admin')).toBe('AD');
    expect(iniciales('   ')).toBe('US');
  });
});
