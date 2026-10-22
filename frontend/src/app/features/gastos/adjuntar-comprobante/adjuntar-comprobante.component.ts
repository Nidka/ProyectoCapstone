import { ChangeDetectionStrategy, Component, DestroyRef, inject, input, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';

const TIPOS_PERMITIDOS = ['image/png', 'image/jpeg', 'image/jpg', 'application/pdf'];
const TAMANO_MAXIMO = 5 * 1024 * 1024; // 5 MB

/** Sube el comprobante de un gasto: arrastrar en escritorio, cámara o archivos en el celular. */
@Component({
  selector: 'app-adjuntar-comprobante',
  imports: [IconoComponent],
  templateUrl: './adjuntar-comprobante.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdjuntarComprobanteComponent {
  readonly gasto = input.required<Gasto>();
  readonly rechazoPrevio = input<Comprobante | undefined>(undefined);

  private readonly store = inject(GastosStore);
  private readonly toast = inject(ToastService);

  protected readonly archivo = signal<File | null>(null);
  protected readonly previa = signal<string | null>(null);
  protected readonly error = signal('');
  protected readonly arrastrando = signal(false);
  protected readonly subiendo = signal(false);

  constructor() {
    inject(DestroyRef).onDestroy(() => this.limpiarPrevia());
  }

  elegir(archivo: File): void {
    if (!TIPOS_PERMITIDOS.includes(archivo.type)) {
      this.error.set('Solo se aceptan archivos JPG, PNG o PDF.');
      this.quitar();
      return;
    }
    if (archivo.size > TAMANO_MAXIMO) {
      this.error.set('El archivo supera el tamaño máximo permitido (5 MB).');
      this.quitar();
      return;
    }
    this.error.set('');
    this.limpiarPrevia();
    this.archivo.set(archivo);
    const puedePrevisualizar = archivo.type.startsWith('image/') && typeof URL.createObjectURL === 'function';
    this.previa.set(puedePrevisualizar ? URL.createObjectURL(archivo) : null);
  }

  protected alArrastrar(evento: DragEvent): void {
    evento.preventDefault();
    this.arrastrando.set(true);
  }

  protected alSalir(evento: DragEvent): void {
    evento.preventDefault();
    this.arrastrando.set(false);
  }

  protected alSoltar(evento: DragEvent): void {
    evento.preventDefault();
    this.arrastrando.set(false);
    const archivo = evento.dataTransfer?.files?.[0];
    if (archivo) this.elegir(archivo);
  }

  protected alSeleccionar(evento: Event): void {
    const campo = evento.target as HTMLInputElement;
    const archivo = campo.files?.[0];
    if (archivo) this.elegir(archivo);
    campo.value = '';
  }

  protected subir(): void {
    const archivo = this.archivo();
    if (!archivo) {
      if (!this.error()) this.error.set('Selecciona al menos un comprobante antes de continuar.');
      return;
    }
    this.subiendo.set(true);
    this.store.adjuntar(this.gasto().id, archivo).subscribe({
      next: () => {
        this.subiendo.set(false);
        this.toast.show('Comprobante adjuntado. El gasto cambió a estado "Pagado".', '', 'ok');
        this.quitar();
      },
      error: err => {
        this.subiendo.set(false);
        if (err instanceof HttpErrorResponse && err.status === 422) {
          this.error.set('Solo se aceptan archivos JPG, PNG o PDF.');
          return;
        }
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }

  private quitar(): void {
    this.limpiarPrevia();
    this.archivo.set(null);
  }

  private limpiarPrevia(): void {
    const url = this.previa();
    if (url && typeof URL.revokeObjectURL === 'function') URL.revokeObjectURL(url);
    this.previa.set(null);
  }
}
