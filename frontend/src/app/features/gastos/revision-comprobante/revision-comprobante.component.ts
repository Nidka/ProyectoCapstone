import { HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy, Component, Injector, afterNextRender, computed, inject, input, linkedSignal, signal,
} from '@angular/core';
import { rxResource } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import {
  Checklist, CriterioCliente, MSG_CHECKLIST_INCOMPLETO, MSG_COMPROBANTE_YA_REVISADO, checklistCompleto, checklistVacio,
  cumplidos, motivoDesdeChecklist, textoConDato,
} from '../../../core/models/checklist';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { ChecklistService } from '../../../core/services/checklist.service';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { formatFechaHora } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';

/**
 * Revisión del comprobante pendiente: el checklist documental del cliente y, con él, aprobar o
 * rechazar con motivo, en línea y sin ventana modal.
 */
@Component({
  selector: 'app-revision-comprobante',
  imports: [FormsModule, IconoComponent],
  templateUrl: './revision-comprobante.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RevisionComprobanteComponent {
  readonly comprobante = input.required<Comprobante>();
  /** El gasto del comprobante: su cliente define el checklist, y su monto y fecha se muestran para comparar */
  readonly gasto = input.required<Gasto>();

  private readonly store = inject(GastosStore);
  private readonly toast = inject(ToastService);
  private readonly gastosApi = inject(GastosApiService);
  private readonly checklists = inject(ChecklistService);
  private readonly injector = inject(Injector);

  /** Criterios que se revisan: los del cliente del gasto (o los estándar) */
  protected readonly lista = rxResource({
    params: () => this.gasto().cliente_id,
    stream: ({ params }) => this.checklists.deCliente(params),
  });
  protected readonly criterios = computed<CriterioCliente[]>(() => (this.lista.hasValue() ? this.lista.value().criterios : []));
  protected readonly cargaCriterios = computed(() => this.lista.isLoading());
  protected readonly falloCriterios = computed(() => !!this.lista.error() && !this.lista.hasValue());

  /** Checklist en curso; vuelve a vacío solo con otro comprobante o con otra lista de criterios (no con una recarga igual) */
  protected readonly checklist = linkedSignal<{ id: number; claves: string }, Checklist>({
    source: () => ({ id: this.comprobante().id, claves: this.criterios().map(c => c.clave).join('|') }),
    computation: (origen, previo) =>
      previo && previo.source.id === origen.id && previo.source.claves === origen.claves ? previo.value : checklistVacio(this.criterios()),
  });
  protected readonly marcados = computed(() => cumplidos(this.criterios(), this.checklist()));
  protected readonly completo = computed(() => checklistCompleto(this.criterios(), this.checklist()));

  protected readonly rechazando = signal(false);
  protected readonly enviando = signal(false);
  protected readonly motivoError = signal('');
  protected motivo = '';
  /** Último motivo propuesto: si la persona no lo editó, sigue a las casillas */
  private motivoPropuesto = '';
  protected readonly url = computed(() => this.gastosApi.fileUrl(this.comprobante().archivo_url));
  protected readonly esPdf = computed(() => this.comprobante().tipo_archivo === 'pdf');
  protected readonly formatFechaHora = formatFechaHora;
  protected textoDe(criterio: CriterioCliente): string {
    return textoConDato(criterio, this.gasto());
  }

  protected marcar(clave: string, cumple: boolean): void {
    this.checklist.update(c => ({ ...c, [clave]: cumple }));
    if (this.rechazando() && this.motivo === this.motivoPropuesto) {
      this.motivo = this.motivoPropuesto = motivoDesdeChecklist(this.criterios(), this.checklist());
    }
  }

  protected reintentar(): void {
    this.lista.reload();
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
    this.motivo = this.motivoPropuesto = motivoDesdeChecklist(this.criterios(), this.checklist());
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
    // Sin criterios cargados no hay checklist que enviar: el servidor admite rechazar sin él
    const checklist = this.criterios().length ? this.checklist() : undefined;
    this.store.rechazar(this.comprobante().id, motivo, checklist).subscribe({
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
      // El checklist del cliente pudo cambiar mientras se revisaba: se vuelven a pedir los criterios
      this.lista.reload();
      return;
    }
    this.toast.show(mensajeError(err), '', 'err');
  }
}
