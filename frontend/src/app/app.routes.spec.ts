import { describe, expect, it } from 'vitest';
import { routes } from './app.routes';
import { authGuard } from './core/guards/auth.guard';
import { cambioClaveGuard } from './core/guards/cambio-clave.guard';
import { rolGuard } from './core/guards/rol.guard';

describe('rutas', () => {
  it('cada pantalla tiene su URL y /app pide sesión', () => {
    const app = routes.find(r => r.path === 'app')!;
    expect(app.canActivate).toEqual([authGuard]);
    expect(app.children?.map(c => c.path)).toEqual([
      '', 'inicio', 'gastos', 'gastos/nuevo', 'gastos/:id', 'reportes', 'checklist', 'usuarios', 'operacion', 'operacion/:modulo', 'catalogos', 'catalogos/:modulo', '**',
    ]);
    expect(app.children?.find(c => c.path === 'catalogos/:modulo')?.data).toEqual({ seccion: 'catalogos' });
  });

  it('registrar un gasto y configurar el checklist son solo de Logística; las pantallas de consulta no piden rol', () => {
    const hijos = routes.find(r => r.path === 'app')!.children!;
    for (const ruta of ['gastos/nuevo', 'checklist']) {
      const r = hijos.find(c => c.path === ruta)!;
      expect(r.canActivate).toEqual([rolGuard]);
      expect(r.data).toEqual({ roles: ['logistica'] });
    }
    for (const ruta of ['inicio', 'gastos', 'gastos/:id', 'reportes', 'operacion/:modulo', 'catalogos/:modulo']) {
      expect(hijos.find(c => c.path === ruta)!.data?.['roles']).toBeUndefined();
    }
  });

  it('crear usuarios ya no es una pantalla pública: «Usuarios» es solo del administrador', () => {
    expect(routes.some(r => r.path === 'register')).toBe(false);
    const usuarios = routes.find(r => r.path === 'app')!.children!.find(c => c.path === 'usuarios')!;
    expect(usuarios.canActivate).toEqual([rolGuard]);
    expect(usuarios.data).toEqual({ roles: ['admin'] });
  });

  it('con la contraseña pendiente ninguna pantalla de /app se abre; «Cambiar contraseña» vive fuera de /app y pide sesión', () => {
    expect(routes.find(r => r.path === 'app')!.canActivateChild).toEqual([cambioClaveGuard]);
    const cambiar = routes.find(r => r.path === 'cambiar-clave')!;
    expect(cambiar.canActivate).toEqual([authGuard]);
    expect(cambiar.loadComponent).toBeTypeOf('function');
  });

  it('los enlaces viejos siguen funcionando y lo desconocido cae en «no encontrada»', () => {
    expect(routes.find(r => r.path === 'dashboard')?.redirectTo).toBe('app/inicio');
    expect(routes.at(-1)?.path).toBe('**');
    expect(routes.at(-1)?.loadComponent).toBeTypeOf('function');
  });
});
