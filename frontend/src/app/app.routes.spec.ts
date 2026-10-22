import { describe, expect, it } from 'vitest';
import { routes } from './app.routes';
import { authGuard } from './core/guards/auth.guard';

describe('rutas', () => {
  it('cada pantalla tiene su URL y /app pide sesión', () => {
    const app = routes.find(r => r.path === 'app')!;
    expect(app.canActivate).toEqual([authGuard]);
    expect(app.children?.map(c => c.path)).toEqual([
      '', 'inicio', 'gastos', 'gastos/nuevo', 'gastos/:id', 'reportes', 'checklist', 'operacion', 'operacion/:modulo', 'catalogos', 'catalogos/:modulo', '**',
    ]);
    expect(app.children?.find(c => c.path === 'catalogos/:modulo')?.data).toEqual({ seccion: 'catalogos' });
  });

  it('los enlaces viejos siguen funcionando y lo desconocido cae en «no encontrada»', () => {
    expect(routes.find(r => r.path === 'dashboard')?.redirectTo).toBe('app/inicio');
    expect(routes.at(-1)?.path).toBe('**');
    expect(routes.at(-1)?.loadComponent).toBeTypeOf('function');
  });
});
