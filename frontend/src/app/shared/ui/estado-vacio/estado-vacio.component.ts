import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { IconoComponent } from '../icono/icono.component';
import { IconoNombre } from '../icono/iconos';

/** Estado vacío que enseña qué hacer: ícono, título, una frase y las acciones proyectadas. */
@Component({
  selector: 'app-estado-vacio',
  imports: [IconoComponent],
  template: `
    <div class="vacio" [class.vacio--compacto]="compacto()">
      <span class="vacio__icono"><app-icono [nombre]="icono()" [tamano]="22" /></span>
      <p class="vacio__titulo">{{ titulo() }}</p>
      @if (texto()) {
        <p class="vacio__texto">{{ texto() }}</p>
      }
      <div class="vacio__acciones"><ng-content /></div>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EstadoVacioComponent {
  readonly icono = input<IconoNombre>('inbox');
  readonly titulo = input.required<string>();
  readonly texto = input('');
  readonly compacto = input(false);
}
