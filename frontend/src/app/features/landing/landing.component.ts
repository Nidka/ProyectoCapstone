import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { forkJoin, timeout } from 'rxjs';
import { Comprobante, EstadoGasto, Gasto } from '../../core/models/gasto.model';
import { ESTADO_GASTO_LABEL } from '../../core/models/gasto-estados';
import { AuthService } from '../../core/services/auth.service';
import { GastosApiService } from '../../core/services/gastos-api.service';
import { ORDEN_ESTADOS, contarPorEstado } from '../../core/state/gastos.logic';
import { HTTP_TIMEOUT_MS } from '../../core/utils/error-messages';
import { HitoComponent } from '../../shared/ui/hito/hito.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { InsigniaEstadoComponent } from '../../shared/ui/insignia-estado/insignia-estado.component';
import { CamionComponent } from '../../shared/ui/marca/camion.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { SenalComponent } from '../../shared/ui/senal/senal.component';

@Component({
  selector: 'app-landing',
  imports: [RouterLink, MarcaComponent, CamionComponent, IconoComponent, InsigniaEstadoComponent, HitoComponent, SenalComponent],
  templateUrl: './landing.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LandingComponent implements OnInit {
  protected readonly auth = inject(AuthService);
  private readonly gastosApi = inject(GastosApiService);

  /** Solo indicadores reales del servicio de gastos; mientras cargan se muestra «…». */
  protected readonly conteo = signal<Record<EstadoGasto, number> | null>(null);
  protected readonly pendientes = signal<number | null>(null);
  protected readonly sinDatos = signal(false);
  /** Los hitos de la carretera de la portada: cada uno con su estado en el cabezal */
  protected readonly hitos: readonly EstadoGasto[] = ORDEN_ESTADOS;
  protected readonly etiqueta = ESTADO_GASTO_LABEL;

  ngOnInit(): void {
    forkJoin({
      gastos: this.gastosApi.get<Gasto[]>('/gastos'),
      pendientes: this.gastosApi.get<Comprobante[]>('/comprobantes?estado=Pendiente'),
    })
      .pipe(timeout(HTTP_TIMEOUT_MS))
      .subscribe({
        next: ({ gastos, pendientes }) => {
          this.conteo.set(contarPorEstado(gastos ?? []));
          this.pendientes.set((pendientes ?? []).length);
        },
        error: () => this.sinDatos.set(true),
      });
  }
}
