import { ChangeDetectionStrategy, Component, OnInit, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ESTADO_GASTO_LABEL } from '../../core/models/gasto-estados';
import { AuthService } from '../../core/services/auth.service';
import { CrudService } from '../../core/services/crud.service';
import { ORDEN_ESTADOS, PESTANA_DE_ESTADO, recientes, sumaPorEstado } from '../../core/state/gastos.logic';
import { GastosStore } from '../../core/state/gastos.store';
import { formatMonto } from '../../core/utils/format';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { HitoComponent } from '../../shared/ui/hito/hito.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { SenalComponent } from '../../shared/ui/senal/senal.component';
import { FilaGastoComponent } from '../gastos/fila-gasto/fila-gasto.component';

@Component({
  selector: 'app-inicio',
  imports: [RouterLink, IconoComponent, EstadoVacioComponent, FilaGastoComponent, HitoComponent, SenalComponent],
  templateUrl: './inicio.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class InicioComponent implements OnInit {
  protected readonly store = inject(GastosStore);
  protected readonly auth = inject(AuthService);
  private readonly crud = inject(CrudService);

  protected readonly orden = ORDEN_ESTADOS;
  protected readonly etiqueta = ESTADO_GASTO_LABEL;
  protected readonly pestanaDe = PESTANA_DE_ESTADO;
  protected readonly pendientes = computed(() => this.store.pendientes().length);
  protected readonly sumas = computed(() => sumaPorEstado(this.store.gastos()));
  protected readonly ultimos = computed(() => recientes(this.store.gastos(), 5));
  protected readonly formatMonto = formatMonto;

  ngOnInit(): void {
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    this.crud.asegurarCargado('viaje');
  }
}
