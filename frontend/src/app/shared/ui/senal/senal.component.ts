import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { IconoComponent } from '../icono/icono.component';
import { IconoNombre } from '../icono/iconos';

/** Clase de señal vial (MTC): la forma y el color dicen qué tipo de aviso es. */
export type TipoSenal = 'preventiva' | 'guia' | 'informativa';

/**
 * Señal vial dibujada: rombo amarillo preventivo, placa verde de guía o placa azul informativa.
 * Es decorativa (aria-hidden); el texto vecino dice lo que la señal anuncia.
 */
@Component({
  selector: 'app-senal',
  imports: [IconoComponent],
  template: `
    <span class="senal__forma">
      <app-icono [nombre]="icono()" [tamano]="tamanoIcono()" [trazo]="2.5" />
    </span>
  `,
  host: {
    '[class]': "'senal senal--' + tipo()",
    '[style.--tam]': "tamano() + 'px'",
    'aria-hidden': 'true',
  },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SenalComponent {
  readonly tipo = input.required<TipoSenal>();
  readonly icono = input.required<IconoNombre>();
  /** Lado del cuadro que ocupa la señal, en px */
  readonly tamano = input(40);

  /** El rombo es un cuadrado girado: su símbolo va más chico que en las placas */
  protected readonly tamanoIcono = computed(() => Math.round(this.tamano() * (this.tipo() === 'preventiva' ? 0.5 : 0.55)));
}
