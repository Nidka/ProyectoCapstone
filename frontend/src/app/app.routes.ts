import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { cambioClaveGuard } from './core/guards/cambio-clave.guard';
import { rolGuard } from './core/guards/rol.guard';

export const routes: Routes = [
  {
    path: '',
    title: 'LIZAR SRL · Fleet Manager Pro',
    loadComponent: () => import('./features/landing/landing.component').then(m => m.LandingComponent),
  },
  {
    path: 'login',
    title: 'Iniciar sesión · LIZAR',
    loadComponent: () => import('./features/login/login.component').then(m => m.LoginComponent),
  },
  {
    path: 'cambiar-clave',
    title: 'Cambiar contraseña · LIZAR',
    canActivate: [authGuard],
    loadComponent: () => import('./features/cambiar-clave/cambiar-clave.component').then(m => m.CambiarClaveComponent),
  },
  { path: 'dashboard', redirectTo: 'app/inicio' },
  {
    path: 'app',
    canActivate: [authGuard],
    canActivateChild: [cambioClaveGuard],
    loadComponent: () => import('./shell/shell.component').then(m => m.ShellComponent),
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'inicio' },
      {
        path: 'inicio',
        title: 'Inicio · LIZAR',
        loadComponent: () => import('./features/inicio/inicio.component').then(m => m.InicioComponent),
      },
      {
        path: 'gastos',
        title: 'Gastos · LIZAR',
        loadComponent: () => import('./features/gastos/gastos-lista/gastos-lista.component').then(m => m.GastosListaComponent),
      },
      {
        path: 'gastos/nuevo',
        title: 'Registrar gasto · LIZAR',
        canActivate: [rolGuard],
        data: { roles: ['logistica'] },
        loadComponent: () => import('./features/gastos/gasto-form/gasto-form.component').then(m => m.GastoFormComponent),
      },
      {
        path: 'gastos/:id',
        title: 'Gasto · LIZAR',
        loadComponent: () => import('./features/gastos/gasto-detalle/gasto-detalle.component').then(m => m.GastoDetalleComponent),
      },
      {
        path: 'reportes',
        title: 'Reportes · LIZAR',
        loadComponent: () => import('./features/reportes/reportes.component').then(m => m.ReportesComponent),
      },
      {
        path: 'checklist',
        title: 'Checklist del cliente · LIZAR',
        canActivate: [rolGuard],
        data: { roles: ['logistica'] },
        loadComponent: () => import('./features/checklist-cliente/checklist-cliente.component').then(m => m.ChecklistClienteComponent),
      },
      {
        path: 'usuarios',
        title: 'Usuarios · LIZAR',
        canActivate: [rolGuard],
        data: { roles: ['admin'] },
        loadComponent: () => import('./features/usuarios/usuarios.component').then(m => m.UsuariosComponent),
      },
      { path: 'operacion', pathMatch: 'full', redirectTo: 'operacion/viajes' },
      {
        path: 'operacion/:modulo',
        data: { seccion: 'operacion' },
        loadComponent: () => import('./features/crud-page/crud-page.component').then(m => m.CrudPageComponent),
      },
      { path: 'catalogos', pathMatch: 'full', redirectTo: 'catalogos/ubigeo' },
      {
        path: 'catalogos/:modulo',
        data: { seccion: 'catalogos' },
        loadComponent: () => import('./features/crud-page/crud-page.component').then(m => m.CrudPageComponent),
      },
      { path: '**', redirectTo: 'inicio' },
    ],
  },
  {
    path: '**',
    title: 'Página no encontrada · LIZAR',
    loadComponent: () => import('./features/no-encontrada/no-encontrada.component').then(m => m.NoEncontradaComponent),
  },
];
