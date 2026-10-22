import { Signal, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router } from '@angular/router';
import { filter, map } from 'rxjs';

/** URL actual del router como signal. Llamar dentro de un contexto de inyección. */
export function urlActual(): Signal<string> {
  const router = inject(Router);
  return toSignal(
    router.events.pipe(
      filter(evento => evento instanceof NavigationEnd),
      map(() => router.url),
    ),
    { initialValue: router.url },
  );
}
