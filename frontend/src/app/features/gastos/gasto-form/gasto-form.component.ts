import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { Router, RouterLink } from '@angular/router';
import { CrudService } from '../../../core/services/crud.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { hoyIso } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';
import { SelectorReferenciaComponent } from '../../../shared/ui/selector-referencia/selector-referencia.component';

@Component({
  selector: 'app-gasto-form',
  imports: [FormsModule, RouterLink, IconoComponent, SelectorReferenciaComponent],
  templateUrl: './gasto-form.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GastoFormComponent implements OnInit {
  private readonly crud = inject(CrudService);
  private readonly store = inject(GastosStore);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);

  /** `?viaje=<id>` al llegar desde la fila de un viaje en Operación */
  readonly viaje = input<string>();

  protected readonly viajeId = signal('');
  protected readonly clienteId = signal('');
  protected monto: string | number | null = '';
  protected fecha = hoyIso();
  protected readonly saving = signal(false);

  protected readonly viajeError = signal('');
  protected readonly clienteError = signal('');
  protected readonly montoError = signal('');
  protected readonly fechaError = signal('');

  protected readonly opcionesViaje = computed(() => this.crud.opciones('viaje'));
  protected readonly opcionesCliente = computed(() => this.crud.opciones('cliente'));
  protected readonly cargandoViajes = computed(() => this.crud.estaCargando('viaje'));
  protected readonly cargandoClientes = computed(() => this.crud.estaCargando('cliente'));
  protected readonly sinViajes = computed(() => this.crud.estaCargado('viaje') && this.crud.getItems('viaje').length === 0);
  protected readonly sinClientes = computed(() => this.crud.estaCargado('cliente') && this.crud.getItems('cliente').length === 0);
  /** Si la lista no cargó, el selector lo dice en vez de «Aún no hay…» */
  protected readonly errorViajes = computed(() => this.crud.estaConError('viaje'));
  protected readonly errorClientes = computed(() => this.crud.estaConError('cliente'));
  protected readonly textoSinLista = 'No se pudo cargar la lista. Recarga la página para intentarlo de nuevo.';

  ngOnInit(): void {
    // RF-1/RF-2/RF-3 necesitan viajes y clientes existentes para el formulario.
    this.crud.asegurarCargado('viaje');
    this.crud.asegurarCargado('cliente');
    const viaje = this.viaje();
    if (viaje) this.viajeId.set(viaje);
  }

  private validar(): boolean {
    this.viajeError.set(this.viajeId() ? '' : 'Selecciona el viaje al que corresponde este gasto.');
    this.clienteError.set(this.clienteId() ? '' : 'Selecciona el cliente responsable de pago.');
    const montoNum = Number(this.monto);
    this.montoError.set(this.monto !== '' && this.monto !== null && montoNum > 0 ? '' : 'Ingresa un monto mayor a S/ 0.00.');
    this.fechaError.set(this.fecha ? '' : 'Selecciona la fecha del gasto.');
    return !this.viajeError() && !this.clienteError() && !this.montoError() && !this.fechaError();
  }

  protected save(): void {
    if (!this.validar()) return;
    this.saving.set(true);
    this.store
      .registrar({ viaje_id: Number(this.viajeId()), cliente_id: Number(this.clienteId()), monto: Number(this.monto), fecha: this.fecha })
      .subscribe({
        next: gasto => {
          this.saving.set(false);
          this.toast.show('Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.', '', 'ok');
          this.router.navigate(['/app/gastos', gasto.id], { fragment: 'paso' });
        },
        error: err => {
          this.saving.set(false);
          if (err instanceof HttpErrorResponse && err.status === 422) {
            this.viajeError.set('Este viaje no existe o no está disponible. Verifica el número de viaje.');
            return;
          }
          this.toast.show(mensajeError(err), '', 'err');
        },
      });
  }
}
