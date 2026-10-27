import { ChangeDetectionStrategy, Component, OnInit, inject, viewChild } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ApiService } from '../core/services/api.service';
import { AuthService } from '../core/services/auth.service';
import { GastosStore } from '../core/state/gastos.store';
import { BarraMovilComponent } from './barra-movil/barra-movil.component';
import { BarraSuperiorComponent } from './barra-superior/barra-superior.component';
import { PaletaComandosComponent } from './paleta-comandos/paleta-comandos.component';

@Component({
  selector: 'app-shell',
  imports: [RouterOutlet, BarraSuperiorComponent, BarraMovilComponent, PaletaComandosComponent],
  templateUrl: './shell.component.html',
  host: { '(document:keydown)': 'alTeclear($event)' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ShellComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);
  private readonly store = inject(GastosStore);
  private readonly paleta = viewChild.required(PaletaComandosComponent);

  ngOnInit(): void {
    this.api.checkHealth();
    this.auth.refrescar();
    this.store.asegurarCargado();
  }

  protected abrirPaleta(): void {
    this.paleta().abrir();
  }

  protected alTeclear(evento: KeyboardEvent): void {
    if ((evento.metaKey || evento.ctrlKey) && evento.key.toLowerCase() === 'k') {
      evento.preventDefault();
      this.paleta().abrir();
    }
  }

  protected saltarAlContenido(): void {
    document.getElementById('contenido')?.focus();
  }
}
