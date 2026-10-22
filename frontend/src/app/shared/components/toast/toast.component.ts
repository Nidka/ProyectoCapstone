import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ToastService } from '../../../core/services/toast.service';
import { ToastType } from '../../../core/models/crud.model';
import { IconoComponent } from '../../ui/icono/icono.component';
import { IconoNombre } from '../../ui/icono/iconos';
import { SenalComponent, TipoSenal } from '../../ui/senal/senal.component';

/** Avisos transitorios. Entran con @starting-style y salen con animate.leave (más rápido que la entrada). */
@Component({
  selector: 'app-toast',
  imports: [IconoComponent, SenalComponent],
  template: `
    <section class="avisos" aria-live="polite" aria-label="Avisos">
      @for (t of toast.toasts(); track t.id) {
        <div [class]="'aviso aviso--' + t.type" role="status" animate.leave="aviso--saliendo">
          @if (senales[t.type]; as tipo) {
            <app-senal class="aviso__icono" [tipo]="tipo" [icono]="iconos[t.type]" [tamano]="24" />
          } @else {
            <app-icono class="aviso__icono" [nombre]="iconos[t.type]" [tamano]="18" />
          }
          <div class="aviso__texto">
            <p class="aviso__titulo">{{ t.title }}</p>
            @if (t.msg) {
              <p class="aviso__mensaje">{{ t.msg }}</p>
            }
          </div>
          <button type="button" class="aviso__cerrar" (click)="toast.remove(t.id)" aria-label="Cerrar aviso">
            <app-icono nombre="x" [tamano]="16" />
          </button>
        </div>
      }
    </section>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ToastComponent {
  protected readonly toast = inject(ToastService);
  protected readonly iconos: Record<ToastType, IconoNombre> = {
    ok: 'check',
    err: 'alert-triangle',
    info: 'info-small',
    warn: 'exclamation-mark',
  };
  /** Cada aviso lleva la señal vial de su clase; el error conserva su icono rojo */
  protected readonly senales: Partial<Record<ToastType, TipoSenal>> = { ok: 'guia', warn: 'preventiva', info: 'informativa' };
}
