import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Gasto } from '../../../core/models/gasto.model';
import { CrudService } from '../../../core/services/crud.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { Pestana, pasosRuta, siguientePaso, ultimoRechazo } from '../../../core/state/gastos.logic';
import { formatFecha, formatMonto } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';
import { InsigniaEstadoComponent } from '../../../shared/ui/insignia-estado/insignia-estado.component';
import { RutaGastoComponent } from '../../../shared/ui/ruta-gasto/ruta-gasto.component';

/** Una fila de la bandeja: toda la fila abre el gasto y la acción lleva directo al paso que toca. */
@Component({
  selector: 'li[app-fila-gasto]',
  imports: [RouterLink, IconoComponent, InsigniaEstadoComponent, RutaGastoComponent],
  templateUrl: './fila-gasto.component.html',
  host: { class: 'fila-gasto', '[class.fila-gasto--cerrada]': "atenuar() && gasto().estado === 'Sustentado'" },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FilaGastoComponent {
  readonly gasto = input.required<Gasto>();
  readonly origen = input<Pestana>('todos');
  readonly atenuar = input(false);

  private readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);

  protected readonly pasos = computed(() => pasosRuta(this.gasto(), this.store.comprobantes()));
  protected readonly paso = computed(() => siguientePaso(this.gasto()));
  protected readonly rechazo = computed(() =>
    this.gasto().estado === 'Registrado' ? ultimoRechazo(this.store.comprobantes(), this.gasto().id) : undefined,
  );
  protected readonly cliente = computed(() => this.crud.etiqueta('cliente', this.gasto().cliente_id));
  protected readonly viaje = computed(() => this.crud.etiqueta('viaje', this.gasto().viaje_id));
  protected readonly enlace = computed(() => ['/app/gastos', this.gasto().id]);
  protected readonly estadoNavegacion = computed(() => ({ origen: this.origen() }));
  protected readonly formatMonto = formatMonto;
  protected readonly formatFecha = formatFecha;
}
