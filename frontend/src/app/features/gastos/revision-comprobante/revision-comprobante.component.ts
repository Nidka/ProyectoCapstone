import { HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy, Component, Injector, afterNextRender, computed, inject, input, linkedSignal, signal,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  CRITERIOS_CHECKLIST, Checklist, MSG_CHECKLIST_INCOMPLETO, MSG_COMPROBANTE_YA_REVISADO, TOTAL_CRITERIOS,
  checklistCompleto, checklistVacio, cumplidos, motivoDesdeChecklist,
} from '../../../core/models/checklist';
import { Comprobante, CriterioChecklist, Gasto } from '../../../core/models/gasto.model';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { formatFechaHora } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';

/**
 * Revisión del comprobante pendiente: el checklist documental y, con él, aprobar o rechazar
 * con motivo, en línea y sin ventana modal.
 */
@Component({
  selector: 'app-revision-comprobante',
  imports: [FormsModule, IconoComponent],
  templateUrl: './revision-comprobante.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RevisionComprobanteComponent {
  readonly comprobante = input.required<Comprobante>();
  /** El gasto del comprobante: su monto y su fecha aparecen en los criterios para comparar */
  readonly gasto = input.required<Gasto>();

  private readonly store = inject(GastosStore);
  private readonly toast = inject(ToastService);
  private readonly gastosApi = inject(GastosApiService);
  private readonly injector = inject(Injector);

  protected readonly criterios = CRITERIOS_CHECKLIST;
  protected readonly total = TOTAL_CRITERIOS;
  /** Checklist en curso; vuelve a vacío solo cuando llega otro comprobante (no con una recarga del mismo) */
  protected readonly checklist = linkedSignal({
    source: () => this.comprobante().id,
    // linkedSignal recalcula con cualquier cambio del input; con el mismo id se conserva lo marcado
    computation: (id: number, previo?: { source: number; value: Checklist }) =>
      previo && previo.source === id ? previo.value : checklistVacio(),
  });
  protected readonly marcados = computed(() => cumplidos(this.checklist()));
  protected readonly completo = computed(() => checklistCompleto(this.checklist()));

  protected readonly rechazando = signal(false);
  protected readonly enviando = signal(false);
  protected readonly motivoError = signal('');
  protected motivo = '';
  /** Último motivo propuesto: si la persona no lo editó, sigue a las casillas */
  private motivoPropuesto = '';
  protected readonly url = computed(() => this.gastosApi.fileUrl(this.comprobante().archivo_url));
  protected readonly esPdf = computed(() => this.comprobante().tipo_archivo === 'pdf');
  protected readonly formatFechaHora = formatFechaHora;

  protected marcar(clave: CriterioChecklist, cumple: boolean): void {
    this.checklist.update(c => ({ ...c, [clave]: cumple }));
    if (this.rechazando() && this.motivo === this.motivoPropuesto) {
      this.motivo = this.motivoPropuesto = motivoDesdeChecklist(this.checklist());
    }
  }

  protected aprobar(): void {
    if (!this.completo()) return;
    this.enviando.set(true);
    this.store.aprobar(this.comprobante().id, this.checklist()).subscribe({
      next: () => {
        this.enviando.set(false);
        this.toast.show('Comprobante aprobado. El gasto quedó como "Sustentado".', '', 'ok');
      },
      error: err => {
        this.enviando.set(false);
        this.avisarError(err, 'aprobar');
      },
    });
  }

  protected abrirRechazo(): void {
    this.rechazando.set(true);
    // Lo que quedó sin marcar es el motivo más probable; se puede editar
    this.motivo = this.motivoPropuesto = motivoDesdeChecklist(this.checklist());
    this.motivoError.set('');
    afterNextRender(() => document.getElementById('motivo-rechazo')?.focus(), { injector: this.injector });
  }

  protected cancelarRechazo(): void {
    this.rechazando.set(false);
    this.motivo = this.motivoPropuesto = '';
    this.motivoError.set('');
  }

  protected confirmarRechazo(): void {
    const motivo = this.motivo.trim();
    if (!motivo) {
      this.motivoError.set('Debes indicar un motivo para rechazar el comprobante.');
      return;
    }
    this.enviando.set(true);
    this.store.rechazar(this.comprobante().id, motivo, this.checklist()).subscribe({
      next: () => {
        this.enviando.set(false);
        this.toast.show('Comprobante rechazado. El gasto volvió a estado "Registrado".', '', 'info');
        this.cancelarRechazo();
      },
      error: err => {
        this.enviando.set(false);
        this.avisarError(err, 'rechazar');
      },
    });
  }

  /** 409 y el 422 de aprobar tienen su propio mensaje; el resto usa el copy exacto de error-messages.ts */
  private avisarError(err: unknown, accion: 'aprobar' | 'rechazar'): void {
    if (err instanceof HttpErrorResponse && err.status === 409) {
      this.toast.show(MSG_COMPROBANTE_YA_REVISADO, '', 'warn');
      this.store.cargar();
      return;
    }
    if (accion === 'aprobar' && err instanceof HttpErrorResponse && err.status === 422) {
      this.toast.show(MSG_CHECKLIST_INCOMPLETO, '', 'err');
      return;
    }
    this.toast.show(mensajeError(err), '', 'err');
  }
}
