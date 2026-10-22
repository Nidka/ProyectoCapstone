import { ChangeDetectionStrategy, Component } from '@angular/core';
import { MarcaComponent } from '../marca/marca.component';

/** Panel de asfalto del login y el registro: la foto de la carretera, la marca grande y la ruta de los tres hitos. */
@Component({
  selector: 'app-panel-acceso',
  imports: [MarcaComponent],
  template: `
    <figure class="acceso__foto">
      <img
        src="https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=960&q=75&auto=format&fit=crop"
        srcset="
          https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=640&q=75&auto=format&fit=crop 640w,
          https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=960&q=75&auto=format&fit=crop 960w,
          https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=1400&q=75&auto=format&fit=crop 1400w
        "
        sizes="40vw"
        width="1400"
        height="928"
        alt="Camión de carga avanzando por una carretera de montaña con postes y señales de tránsito"
        loading="lazy"
        decoding="async"
      />
    </figure>
    <div class="cinta" aria-hidden="true"></div>
    <div class="acceso__contenido">
      <app-marca class="acceso__marca-grande" />
      <ol class="acceso__via" aria-hidden="true">
        <li class="acceso__hito" data-estado="Registrado">Registrado</li>
        <li class="acceso__hito" data-estado="Pagado">Pagado</li>
        <li class="acceso__hito" data-estado="Sustentado">Sustentado</li>
      </ol>
      <p class="acceso__lema">Gastos de viaje con su comprobante, de la ruta a la oficina.</p>
    </div>
  `,
  host: { class: 'acceso__marca' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PanelAccesoComponent {}
