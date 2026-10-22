import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';

@Component({
  selector: 'app-no-encontrada',
  imports: [RouterLink, MarcaComponent],
  template: `
    <main class="no-encontrada">
      <app-marca [sobreAsfalto]="false" />
      <p class="no-encontrada__codigo">Error 404</p>
      <h1>Página no encontrada</h1>
      <p>La dirección que abriste no existe.</p>
      <a class="btn btn--primario" routerLink="/">Ir al inicio</a>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NoEncontradaComponent {}
