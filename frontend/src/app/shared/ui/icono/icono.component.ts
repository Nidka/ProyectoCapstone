import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { ICONOS, IconoNombre } from './iconos';

/** Icono de línea de la familia Tabler; decorativo (aria-hidden): el texto vecino da el significado. */
@Component({
  selector: 'app-icono',
  template: `
    <svg
      [attr.width]="tamano()"
      [attr.height]="tamano()"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      [attr.stroke-width]="trazo()"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      @for (d of trazos(); track $index) {
        <path [attr.d]="d" />
      }
    </svg>
  `,
  host: { class: 'icono' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class IconoComponent {
  readonly nombre = input.required<IconoNombre>();
  readonly tamano = input(18);
  readonly trazo = input(1.75);

  protected readonly trazos = computed(() => ICONOS[this.nombre()]);
}
