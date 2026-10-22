import { ChangeDetectionStrategy, Component, afterNextRender, computed, input, signal } from '@angular/core';
import { Comprobante } from '../../../core/models/gasto.model';
import { ESTADO_GASTO_LABEL } from '../../../core/models/gasto-estados';
import { PasoRuta, etiquetaRuta } from '../../../core/state/gastos.logic';
import { formatFechaHora } from '../../../core/utils/format';
import { CamionComponent } from '../marca/camion.component';

/**
 * La ruta del gasto: tres hitos (Registrado, Pagado, Sustentado) sobre una línea de carretera,
 * con el camión del logo en el hito actual. Firma visual de «Hitos de carretera».
 */
@Component({
  selector: 'app-ruta-gasto',
  imports: [CamionComponent],
  templateUrl: './ruta-gasto.component.html',
  host: {
    class: 'ruta',
    '[class.ruta--compacta]': "variante() === 'compacta'",
    '[class.ruta--completa]': "variante() === 'completa'",
    '[class.ruta--lista]': 'lista()',
    '[style.--actual]': 'indiceActual()',
    '[attr.role]': "variante() === 'compacta' ? 'img' : null",
    '[attr.aria-label]': "variante() === 'compacta' ? etiqueta() : null",
  },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RutaGastoComponent {
  readonly pasos = input.required<PasoRuta[]>();
  readonly variante = input<'compacta' | 'completa'>('compacta');
  readonly rechazo = input<Comprobante | undefined>(undefined);

  protected readonly etiqueta = computed(() => etiquetaRuta(this.pasos()));
  protected readonly indiceActual = computed(() => Math.max(0, this.pasos().findIndex(p => p.actual)));
  /** Se activa tras el primer render: el camión no se anima al aparecer, solo cuando el gasto avanza. */
  protected readonly lista = signal(false);
  protected readonly nombres = ESTADO_GASTO_LABEL;
  protected readonly formatFechaHora = formatFechaHora;

  constructor() {
    afterNextRender(() => this.lista.set(true));
  }
}
