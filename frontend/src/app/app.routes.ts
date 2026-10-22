import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

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
    path: 'register',
    title: 'Crear usuario · LIZAR',
    loadComponent: () => import('./features/register/register.component').then(m => m.RegisterComponent),
  },
  { path: 'dashboard', redirectTo: 'app/inicio' },
  {
    path: 'app',
    canActivate: [authGuard],
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
