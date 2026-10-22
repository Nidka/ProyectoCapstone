import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { CamionComponent } from './camion.component';

@Component({
  selector: 'app-marca',
  imports: [CamionComponent],
  template: `
    <span class="marca__insignia" aria-hidden="true"><app-camion /></span>
    @if (conTexto()) {
      <span class="marca__texto">
        <span class="marca__nombre">LIZAR</span>
        <span class="marca__sub">Fleet Manager Pro</span>
      </span>
    }
  `,
  host: { class: 'marca', '[class.marca--clara]': '!sobreAsfalto()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MarcaComponent {
  readonly conTexto = input(true);
  readonly sobreAsfalto = input(true);
}
