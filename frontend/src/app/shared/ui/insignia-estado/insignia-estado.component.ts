import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { EstadoComprobante, EstadoGasto } from '../../../core/models/gasto.model';
import {
  ESTADO_COMPROBANTE_CLASS, ESTADO_COMPROBANTE_LABEL, ESTADO_GASTO_CLASS, ESTADO_GASTO_LABEL,
} from '../../../core/models/gasto-estados';

/** Estado de un gasto o comprobante con el copy y el color exactos de gasto-estados.ts. */
@Component({
  selector: 'app-insignia-estado',
  template: `<span [class]="clase()"><span class="lbl__poste" aria-hidden="true"></span>{{ etiqueta() }}</span>`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class InsigniaEstadoComponent {
  readonly estado = input.required<EstadoGasto | EstadoComprobante>();
  readonly tipo = input<'gasto' | 'comprobante'>('gasto');

  protected readonly etiqueta = computed(() => {
    const e = this.estado();
    return this.tipo() === 'gasto'
      ? ESTADO_GASTO_LABEL[e as EstadoGasto] ?? e
      : ESTADO_COMPROBANTE_LABEL[e as EstadoComprobante] ?? e;
  });

  protected readonly clase = computed(() => {
    const e = this.estado();
    const c = this.tipo() === 'gasto' ? ESTADO_GASTO_CLASS[e as EstadoGasto] : ESTADO_COMPROBANTE_CLASS[e as EstadoComprobante];
    return `lbl ${c ?? 'lbl-n'}`;
  });
}
