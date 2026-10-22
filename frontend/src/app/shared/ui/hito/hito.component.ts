import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { EstadoGasto } from '../../../core/models/gasto.model';

/**
 * Hito kilométrico: la cifra va pintada en el cuerpo del poste y el cabezal lleva el color del estado,
 * como los hitos de concreto de la Panamericana. Es un objeto del camino, no una tarjeta de métrica.
 */
@Component({
  selector: 'app-hito',
  template: `
    <span class="hito__cabezal">{{ cabezal() }}</span>
    <span class="hito__cifra">{{ cifra() }}</span>
  `,
  host: {
    class: 'hito',
    '[attr.data-estado]': 'estado() ?? null',
    '[class.hito--grande]': "tamano() === 'grande'",
  },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HitoComponent {
  readonly cabezal = input.required<string>();
  readonly cifra = input.required<string | number>();
  readonly estado = input<EstadoGasto>();
  readonly tamano = input<'normal' | 'grande'>('normal');
}
