import { ChangeDetectionStrategy, Component } from '@angular/core';

/** Pictograma del camión de LIZAR (logo). Compromiso de marca: se conserva su geometría original. */
@Component({
  selector: 'app-camion',
  template: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
      <rect x="1" y="4" width="14" height="13" rx="2" />
      <path d="M15 9h4.5L22 13v4h-7V9z" />
      <circle cx="5.5" cy="19.5" r="2.5" />
      <circle cx="18.5" cy="19.5" r="2.5" />
    </svg>
  `,
  host: { class: 'camion' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CamionComponent {}
