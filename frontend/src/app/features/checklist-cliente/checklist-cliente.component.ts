import { HttpErrorResponse } from '@angular/common/http';
import { ChangeDetectionStrategy, Component, computed, inject, input, linkedSignal, signal } from '@angular/core';
import { rxResource } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ChecklistCliente, MAX_CRITERIOS, MAX_TEXTO, MIN_TEXTO } from '../../core/models/checklist';
import { ChecklistService, EntradaCriterio } from '../../core/services/checklist.service';
import { CrudService } from '../../core/services/crud.service';
import { ToastService } from '../../core/services/toast.service';
import { idDeConsulta } from '../../core/state/gastos.logic';
import { mensajeError } from '../../core/utils/error-messages';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';

/** Un criterio en edición: `clave` solo existe en los que ya estaban guardados. */
interface Fila {
  uid: number;
  clave?: string;
  texto: string;
}

const MSG_LARGO = `Escribe entre ${MIN_TEXTO} y ${MAX_TEXTO} caracteres.`;
const MSG_REPETIDO = 'Este criterio está repetido.';

function normal(texto: string): string {
  return texto.trim().toLowerCase();
}

/** Configuración del checklist documental de un cliente: los criterios que Logística revisa en sus comprobantes. */
@Component({
  selector: 'app-checklist-cliente',
  imports: [FormsModule, RouterLink, IconoComponent, EstadoVacioComponent],
  templateUrl: './checklist-cliente.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ChecklistClienteComponent {
  /** Id del cliente, de ?cliente= en la URL */
  readonly cliente = input<string>();

  private readonly checklists = inject(ChecklistService);
  private readonly crud = inject(CrudService);
  private readonly toast = inject(ToastService);

  protected readonly clienteId = computed(() => idDeConsulta(this.cliente()));
  protected readonly nombre = computed(() => {
    const id = this.clienteId();
    return id ? this.crud.etiqueta('cliente', id) : '';
  });

  protected readonly lista = rxResource({
    params: () => this.clienteId() ?? undefined,
    stream: ({ params }) => this.checklists.deCliente(params),
  });
  protected readonly actual = computed<ChecklistCliente | undefined>(() => (this.lista.hasValue() ? this.lista.value() : undefined));
  protected readonly falloCarga = computed(() => !!this.lista.error() && !this.actual());

  private uid = 0;
  /** Lo que se edita; vuelve a empezar cuando llega otro checklist del servidor (carga, guardado o restablecer) */
  protected readonly filas = linkedSignal<ChecklistCliente | undefined, Fila[]>({
    source: this.actual,
    computation: c => (c ? c.criterios.map(k => ({ uid: ++this.uid, clave: k.clave, texto: k.texto })) : []),
  });
  protected readonly nuevo = signal('');

  protected readonly guardando = signal(false);
  protected readonly confirmandoRestablecer = signal(false);

  protected readonly maxCriterios = MAX_CRITERIOS;
  protected readonly maxTexto = MAX_TEXTO;

  /** Mensaje de error de cada fila (por uid); vacío si está bien */
  protected readonly errores = computed(() => {
    const filas = this.filas();
    const vistos = new Map<string, number>();
    for (const f of filas) vistos.set(normal(f.texto), (vistos.get(normal(f.texto)) ?? 0) + 1);
    const errores = new Map<number, string>();
    for (const f of filas) {
      const largo = f.texto.trim().length;
      if (largo < MIN_TEXTO || largo > MAX_TEXTO) errores.set(f.uid, MSG_LARGO);
      else if ((vistos.get(normal(f.texto)) ?? 0) > 1) errores.set(f.uid, MSG_REPETIDO);
    }
    return errores;
  });
  protected readonly valido = computed(() => this.filas().length >= 1 && this.errores().size === 0);
  protected readonly sucio = computed(() => {
    const original = this.actual()?.criterios ?? [];
    const filas = this.filas();
    return original.length !== filas.length || filas.some((f, i) => f.clave !== original[i].clave || f.texto.trim() !== original[i].texto);
  });
  protected readonly puedeGuardar = computed(() => this.sucio() && this.valido() && !this.guardando());
  protected readonly puedeAgregar = computed(() => this.filas().length < MAX_CRITERIOS && this.nuevo().trim().length >= MIN_TEXTO);

  constructor() {
    this.crud.asegurarCargado('cliente');
  }

  protected editar(uid: number, texto: string): void {
    this.filas.update(l => l.map(f => (f.uid === uid ? { ...f, texto } : f)));
  }

  protected agregar(): void {
    const texto = this.nuevo().trim();
    if (this.filas().length >= MAX_CRITERIOS || texto.length < MIN_TEXTO) return;
    this.filas.update(l => [...l, { uid: ++this.uid, texto }]);
    this.nuevo.set('');
  }

  protected mover(indice: number, hacia: -1 | 1): void {
    this.filas.update(l => {
      const otro = indice + hacia;
      if (otro < 0 || otro >= l.length) return l;
      const copia = [...l];
      [copia[indice], copia[otro]] = [copia[otro], copia[indice]];
      return copia;
    });
  }

  protected quitar(uid: number): void {
    if (this.filas().length <= 1) return;
    this.filas.update(l => l.filter(f => f.uid !== uid));
  }

  protected reintentar(): void {
    this.lista.reload();
  }

  protected guardar(): void {
    const id = this.clienteId();
    if (!id || !this.puedeGuardar()) return;
    this.guardando.set(true);
    const criterios: EntradaCriterio[] = this.filas().map(f => (f.clave ? { clave: f.clave, texto: f.texto.trim() } : { texto: f.texto.trim() }));
    this.checklists.guardar(id, criterios).subscribe({
      next: guardado => {
        this.guardando.set(false);
        this.lista.set(guardado);
        this.toast.show('Checklist guardado.', '', 'ok');
      },
      error: err => {
        this.guardando.set(false);
        this.toast.show(
          err instanceof HttpErrorResponse && err.status === 422 ? 'No se pudo guardar el checklist. Revisa los criterios.' : mensajeError(err),
          '',
          'err',
        );
      },
    });
  }

  protected restablecer(): void {
    const id = this.clienteId();
    if (!id || this.guardando()) return;
    this.guardando.set(true);
    this.checklists.restablecer(id).subscribe({
      next: estandar => {
        this.guardando.set(false);
        this.confirmandoRestablecer.set(false);
        this.lista.set(estandar);
        this.toast.show('Checklist restablecido al estándar.', '', 'ok');
      },
      error: err => {
        this.guardando.set(false);
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }
}
