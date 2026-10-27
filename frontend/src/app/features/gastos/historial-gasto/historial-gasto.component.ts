import { ChangeDetectionStrategy, Component, DestroyRef, effect, inject, input, signal, untracked } from '@angular/core';
import { Subscription } from 'rxjs';
import { RegistroHistorial, detalleDeRegistro, ordenarCronologico, textoAccion } from '../../../core/models/historial';
import { HistorialService } from '../../../core/services/historial.service';
import { formatFechaHora } from '../../../core/utils/format';
import { EstadoVacioComponent } from '../../../shared/ui/estado-vacio/estado-vacio.component';

type Estado = 'cargando' | 'listo' | 'error';

/** «Historial de cambios» de un gasto: quién hizo qué y cuándo, del más antiguo al más reciente (bitácora de auditoría). */
@Component({
  selector: 'app-historial-gasto',
  imports: [EstadoVacioComponent],
  templateUrl: './historial-gasto.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HistorialGastoComponent {
  readonly gastoId = input.required<number>();
  /** Cambia cuando el gasto tuvo movimientos (se adjuntó o revisó un comprobante): el historial se vuelve a pedir */
  readonly marca = input<unknown>();

  private readonly historiales = inject(HistorialService);
  private pedido?: Subscription;
  /** Gasto del que se muestran los movimientos; al cambiar, la lista anterior ya no vale */
  private deGasto: number | null = null;

  protected readonly estado = signal<Estado>('cargando');
  protected readonly registros = signal<RegistroHistorial[]>([]);

  protected readonly textoAccion = textoAccion;
  protected readonly detalleDeRegistro = detalleDeRegistro;
  protected readonly formatFechaHora = formatFechaHora;

  constructor() {
    effect(() => {
      const id = this.gastoId();
      this.marca();
      untracked(() => this.cargar(id));
    });
    inject(DestroyRef).onDestroy(() => this.pedido?.unsubscribe());
  }

  protected reintentar(): void {
    this.cargar(this.gastoId());
  }

  private cargar(id: number): void {
    // Una respuesta atrasada de un pedido anterior no debe pisar a la nueva
    this.pedido?.unsubscribe();
    if (id !== this.deGasto) {
      this.deGasto = id;
      this.registros.set([]);
      this.estado.set('cargando');
    } else if (this.estado() === 'error') {
      this.estado.set('cargando');
    } // Si solo se refresca, la lista que ya se ve se queda hasta que llegue la nueva
    this.pedido = this.historiales.deGasto(id).subscribe({
      next: lista => {
        this.registros.set(ordenarCronologico(lista));
        this.estado.set('listo');
      },
      error: () => {
        this.registros.set([]);
        this.estado.set('error');
      },
    });
  }
}
