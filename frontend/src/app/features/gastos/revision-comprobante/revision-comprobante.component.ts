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
import {
  FilaLectura, MSG_GASTO_NO_CORREGIBLE, MSG_LECTURA_FALLIDA, clavesSugeridas, etiquetaSugerencia, filasLectura, motivoDesdeLectura,
} from '../../../core/models/lectura';
import { ChecklistService } from '../../../core/services/checklist.service';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { LecturaService } from '../../../core/services/lectura.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { formatFecha, formatFechaHora, formatMonto } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';

/**
 * Revisión del comprobante pendiente: el checklist documental del cliente y, con él, aprobar o
 * rechazar con motivo, en línea y sin ventana modal. La lectura automática del comprobante solo
 * sugiere casillas: no bloquea nada y siempre decide una persona.
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
  private readonly lecturas = inject(LecturaService);
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
  /** Lectura automática del comprobante (OCR); con otro comprobante se vuelve a pedir */
  protected readonly lectura = rxResource({
    params: () => this.comprobante().id,
    stream: ({ params }) => this.lecturas.lectura(params),
  });
  private readonly lecturaLista = computed(() => (this.lectura.hasValue() ? this.lectura.value() : undefined));
  protected readonly leyendo = computed(() => this.lectura.isLoading());
  /** Claves que la lectura sugiere marcar entre los criterios del cliente */
  protected readonly sugeridas = computed(() => clavesSugeridas(this.criterios(), this.lecturaLista()));
  /** Panel «Lectura automática»: el gasto frente a lo leído; sin filas no hay panel */
  protected readonly filas = computed(() => filasLectura(this.criterios(), this.lecturaLista(), this.gasto()));
  /** Lo que el comprobante no respalda: el veredicto del panel dice cuántas son y cuáles */
  protected readonly diferencias = computed(() => this.filas().filter(f => !f.coincide));
  protected readonly veredicto = computed(() => {
    const n = this.diferencias().length;
    const nombres = this.diferencias().map(f => f.dato.toLowerCase()).join(', ');
    return n === 1 ? `1 diferencia con el gasto: ${nombres}.` : `${n} diferencias con el gasto: ${nombres}.`;
  });
  /** Todo lo sugerido ya está marcado: el botón da paso a la confirmación */
  protected readonly aplicadas = computed(() => this.sugeridas().length > 0 && this.sugeridas().every(c => this.checklist()[c]));
  /** Aviso discreto cuando no hay lectura útil; un fallo de red usa el mensaje general de la app */
  protected readonly avisoLectura = computed(() => {
    if (this.lectura.isLoading()) return '';
    if (this.lectura.error()) return mensajeError(this.lectura.error());
    const l = this.lecturaLista();
    return l && l.estado !== 'leido' ? MSG_LECTURA_FALLIDA : '';
  });

  protected readonly marcados = computed(() => cumplidos(this.criterios(), this.checklist()));
  protected readonly completo = computed(() => checklistCompleto(this.criterios(), this.checklist()));
  /** Cuánto falta para poder aprobar, dicho en lugar de pedir «todos» */
  protected readonly faltan = computed(() => {
    const n = this.criterios().length - this.marcados();
    if (!this.criterios().length) return 'Marca todos los criterios para aprobar.';
    return n === 1 ? 'Falta 1 criterio para aprobar.' : `Faltan ${n} criterios para aprobar.`;
  });
  protected readonly avance = computed(() => (this.criterios().length ? Math.round((100 * this.marcados()) / this.criterios().length) : 0));

  /** La fila del panel cuyo gasto se está corrigiendo con lo leído; null = sin formulario abierto */
  protected readonly corrigiendo = signal<FilaLectura | null>(null);
  protected readonly errorCorreccion = signal('');
  protected valorCorreccion: string | number = '';
  protected motivoCorreccion = '';

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

  /** «✓ Coincide» o «⚠ Revisar» junto a la casilla; vacío si la lectura no dice nada de ese criterio */
  protected sugerenciaDe(clave: string): string {
    const l = this.lecturaLista();
    return l?.estado === 'leido' ? etiquetaSugerencia(l.sugerencias[clave]) : '';
  }

  protected marcar(clave: string, cumple: boolean): void {
    this.checklist.update(c => ({ ...c, [clave]: cumple }));
    this.seguirMotivo();
  }

  /** Marca las casillas sugeridas, sin tocar las demás; el conteo y el motivo siguen como con un clic */
  protected aplicarSugerencias(): void {
    const claves = this.sugeridas();
    this.checklist.update(c => ({ ...c, ...Object.fromEntries(claves.map(clave => [clave, true])) }));
    this.seguirMotivo();
  }

  /** Con el rechazo abierto, el motivo propuesto sigue a las casillas mientras la persona no lo edite */
  private seguirMotivo(): void {
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

  /** «Corregir gasto a S/ 513.00» o «… a 26/09/2019»: lo que se aplicaría, dicho en el botón */
  protected textoCorregir(fila: FilaLectura): string {
    const c = fila.corregible;
    if (!c) return '';
    return `Corregir gasto a ${c.campo === 'monto' ? formatMonto(c.valor) : formatFecha(String(c.valor))}`;
  }

  protected abrirCorreccion(fila: FilaLectura): void {
    if (!fila.corregible) return;
    this.corrigiendo.set(fila);
    this.valorCorreccion = fila.corregible.valor;
    this.motivoCorreccion = '';
    this.errorCorreccion.set('');
    afterNextRender(() => document.getElementById('correccion-motivo')?.focus(), { injector: this.injector });
  }

  protected cancelarCorreccion(): void {
    this.corrigiendo.set(null);
    this.motivoCorreccion = '';
    this.errorCorreccion.set('');
  }

  /**
   * Corrige el monto o la fecha del gasto con lo leído (o con lo que la persona ajuste) y deja el motivo
   * en el historial. El servidor lo permite mientras el gasto no esté sustentado; luego se vuelve a
   * leer el comprobante para que las sugerencias se calculen contra el gasto nuevo.
   */
  protected confirmarCorreccion(): void {
    const campo = this.corrigiendo()?.corregible?.campo;
    if (!campo) return;
    const motivo = this.motivoCorreccion.trim();
    if (campo === 'monto' && !(Number(this.valorCorreccion) > 0)) {
      this.errorCorreccion.set('Indica un monto mayor a 0.');
      return;
    }
    if (campo === 'fecha' && !this.valorCorreccion) {
      this.errorCorreccion.set('Indica la fecha correcta.');
      return;
    }
    if (!motivo) {
      this.errorCorreccion.set('Indica el motivo de la corrección.');
      return;
    }
    this.errorCorreccion.set('');
    this.enviando.set(true);
    const datos = campo === 'monto' ? { monto: Number(this.valorCorreccion), motivo } : { fecha: String(this.valorCorreccion), motivo };
    this.store.corregir(this.gasto().id, datos).subscribe({
      next: () => {
        this.enviando.set(false);
        this.toast.show('Gasto corregido. Volvimos a leer el comprobante.', '', 'ok');
        this.cancelarCorreccion();
        this.lectura.reload();
      },
      error: err => {
        this.enviando.set(false);
        if (err instanceof HttpErrorResponse && err.status === 409) {
          this.toast.show(MSG_GASTO_NO_CORREGIBLE, '', 'warn');
          this.cancelarCorreccion();
          this.store.cargar();
          return;
        }
        // El servidor explica lo que no admite (por ejemplo, una fecha posterior a la de cobro)
        const detalle = err instanceof HttpErrorResponse && (err.status === 400 || err.status === 422) ? err.error?.detail : null;
        this.toast.show(typeof detalle === 'string' ? detalle : mensajeError(err), '', 'err');
      },
    });
  }

  /**
   * Abre el rechazo con el motivo que se deduce de la lectura. Queda como un motivo ya editado: marcar
   * casillas no lo pisa, y la persona lo corrige o lo confirma. Sin nada que decir, cae al de las casillas.
   */
  protected rechazarConLectura(): void {
    this.rechazando.set(true);
    const deLectura = motivoDesdeLectura(this.filas());
    this.motivo = deLectura || motivoDesdeChecklist(this.criterios(), this.checklist());
    this.motivoPropuesto = deLectura ? '' : this.motivo;
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
