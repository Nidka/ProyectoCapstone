import {
  ChangeDetectionStrategy, Component, DestroyRef, ElementRef, InjectionToken, afterRenderEffect, inject, input, viewChild,
} from '@angular/core';
import type { ConfigGrafico } from '../graficos';
import { FabricaGrafico, InstanciaGrafico, crearGraficoChartJs } from './chart-js';

export const FABRICA_GRAFICO = new InjectionToken<FabricaGrafico>('FABRICA_GRAFICO', {
  providedIn: 'root',
  factory: () => crearGraficoChartJs,
});

/** Lienzo de Chart.js: lo crea con la primera configuración, lo actualiza y lo destruye con el componente. */
@Component({
  selector: 'app-grafico',
  template: `
    <div class="grafico" [style.height.px]="alto()">
      <canvas #lienzo role="img" [attr.aria-label]="descripcion()"></canvas>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GraficoComponent {
  readonly config = input.required<ConfigGrafico>();
  /** Resumen del gráfico para lectores de pantalla */
  readonly descripcion = input.required<string>();
  readonly alto = input(280);

  private readonly lienzo = viewChild.required<ElementRef<HTMLCanvasElement>>('lienzo');
  private readonly fabrica = inject(FABRICA_GRAFICO);
  private instancia: InstanciaGrafico | null = null;

  constructor() {
    afterRenderEffect(() => {
      const config = this.config();
      if (this.instancia) {
        this.instancia.actualizar(config);
      } else {
        this.instancia = this.fabrica(this.lienzo().nativeElement, config);
      }
    });
    inject(DestroyRef).onDestroy(() => this.instancia?.destruir());
  }
}
