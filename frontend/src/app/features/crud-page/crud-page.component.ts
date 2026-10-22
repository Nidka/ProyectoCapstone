import {
  ChangeDetectionStrategy, Component, ElementRef, Injector, afterNextRender, computed, effect, inject, input,
  signal, untracked, viewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DomSanitizer, SafeHtml, Title } from '@angular/platform-browser';
import { Router, RouterLink } from '@angular/router';
import { OpcionRef } from '../../core/models/crud.model';
import { SeccionId, moduloPorSlug, rutaDeClave, rutaModulo, seccion } from '../../core/navigation/navegacion';
import { CrudService } from '../../core/services/crud.service';
import { ToastService } from '../../core/services/toast.service';
import { normalizar } from '../../core/utils/texto';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { SelectorReferenciaComponent } from '../../shared/ui/selector-referencia/selector-referencia.component';
import { EstadoCampo, enumerar, validarCampos } from './crud-page.logic';

/** Un módulo de Operación o Catálogos: la tabla primero; el formulario en una hoja lateral. */
@Component({
  selector: 'app-crud-page',
  imports: [FormsModule, RouterLink, IconoComponent, EstadoVacioComponent, SelectorReferenciaComponent],
  templateUrl: './crud-page.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CrudPageComponent {
  /** :modulo de la ruta */
  readonly modulo = input('');
  /** data.seccion de la ruta */
  readonly seccion = input<SeccionId>('operacion');
  /** ?ver=<id>: abre la ficha de ese registro (p. ej. desde el detalle de un gasto) */
  readonly ver = input<string>();

  protected readonly crud = inject(CrudService);
  private readonly router = inject(Router);
  private readonly sanitizer = inject(DomSanitizer);
  private readonly titulo = inject(Title);
  private readonly toast = inject(ToastService);
  private readonly injector = inject(Injector);

  private readonly hoja = viewChild<ElementRef<HTMLDialogElement>>('hoja');
  private readonly dialogoRegistro = viewChild<ElementRef<HTMLDialogElement>>('dialogoRegistro');

  protected readonly seccionNav = computed(() => seccion(this.seccion()));
  protected readonly mod = computed(() => moduloPorSlug(this.seccion(), this.modulo()));
  protected readonly def = computed(() => {
    const m = this.mod();
    return m ? this.crud.getCrud(m.clave) : undefined;
  });
  protected readonly termino = signal('');
  protected readonly campos = signal<EstadoCampo[]>([]);
  protected readonly guardando = signal(false);
  /** Motivo con que la API rechazó el guardado; se muestra dentro de la hoja (un aviso quedaría tapado por el modal) */
  protected readonly errorGuardado = signal('');
  /** Registro abierto en la ficha; su contenido se recalcula cuando llegan las listas relacionadas */
  private readonly registroAbierto = signal<Record<string, unknown> | null>(null);
  protected readonly registro = computed<{ titulo: string; campos: { etiqueta: string; valor: SafeHtml }[] } | null>(() => {
    const item = this.registroAbierto();
    const d = this.def();
    const m = this.mod();
    if (!item || !d || !m) return null;
    const ref = this.crud.ref;
    const nombre = d.nombre?.(item, ref);
    // Las mismas columnas de la tabla, con su nombre: nada de JSON
    const celdas = d.row(item, ref);
    const campos = d.cols.map((etiqueta, i) => ({ etiqueta, valor: this.sanitizer.bypassSecurityTrustHtml(celdas[i] ?? '') }));
    return { titulo: nombre ? `${m.etiqueta} · ${nombre}` : m.etiqueta, campos };
  });
  /** Registro ya abierto por ?ver=, para no reabrirlo en cada recarga de la lista */
  private verAbierto = '';

  protected readonly cargando = computed(() => {
    const d = this.def();
    return !!d && this.crud.estaCargando(d.key) && !this.crud.estaCargado(d.key);
  });
  /** La lista no cargó: se dice así, en vez de mostrarla como vacía */
  protected readonly conError = computed(() => {
    const d = this.def();
    return !!d && this.crud.estaConError(d.key) && !this.crud.estaCargado(d.key);
  });

  protected readonly filas = computed(() => {
    const d = this.def();
    if (!d) return [];
    const ref = this.crud.ref;
    const t = normalizar(this.termino()).trim();
    return this.crud
      .getItems(d.key)
      .filter(item => !t || normalizar(`${Object.values(item).join(' ')} ${d.nombre?.(item, ref) ?? ''}`).includes(t))
      .map(item => ({ item, celdas: d.row(item, ref).map(html => this.sanitizer.bypassSecurityTrustHtml(html)) }));
  });

  protected readonly opciones = computed(() => {
    // Partial: un módulo que dejó de mostrarse puede no tener sus opciones (la plantilla usa ?? [])
    const porClave: Partial<Record<string, OpcionRef[]>> = {};
    for (const f of this.def()?.fields ?? []) {
      if (f.tp === 'ref' && f.ref && !porClave[f.ref]) porClave[f.ref] = this.crud.opciones(f.ref);
    }
    return porClave;
  });

  /** Módulos previos obligatorios: «Antes de crear viajes necesitas…». */
  protected readonly requisitos = computed(() => {
    const d = this.def();
    if (!d) return [];
    const claves = [...new Set(d.fields.filter(f => f.tp === 'ref' && f.req && f.ref).map(f => f.ref as string))];
    return claves.map(clave => ({
      clave,
      texto: this.crud.getCrud(clave)?.textos?.plural ?? clave,
      ruta: rutaDeClave(clave),
      vacio: this.crud.estaCargado(clave) && this.crud.getItems(clave).length === 0,
    }));
  });
  protected readonly faltantes = computed(() => this.requisitos().filter(r => r.vacio));
  protected readonly requisitosTexto = computed(() => {
    const d = this.def();
    const r = this.requisitos();
    return d && r.length ? `Antes de crear ${d.textos?.plural ?? 'registros'} necesitas: ${enumerar(r.map(x => x.texto))}.` : '';
  });

  protected readonly rutaModulo = rutaModulo;
  protected readonly rutaDeClave = rutaDeClave;

  constructor() {
    effect(() => {
      const m = this.mod();
      const s = this.seccion();
      untracked(() => {
        if (!m) {
          this.router.navigate(['/app', s, seccion(s).modulos[0].slug], { replaceUrl: true });
          return;
        }
        // Al cambiar de módulo (pestaña o Atrás) no puede quedar abierta la hoja ni el registro del anterior
        this.cerrarDialogos();
        this.crud.asegurarConRelaciones(m.clave);
        this.termino.set('');
        this.titulo.setTitle(`${m.etiqueta} · LIZAR`);
      });
    });

    effect(() => {
      const d = this.def();
      const id = this.ver();
      const idKey = d?.idKey;
      if (!d || !idKey || !id || !this.crud.estaCargado(d.key)) return;
      const item = this.crud.getItems(d.key).find(i => String(i[idKey]) === id);
      const clave = `${d.key}:${id}`;
      if (!item || clave === this.verAbierto) return;
      this.verAbierto = clave;
      // El diálogo existe recién después de dibujar la vista
      untracked(() => afterNextRender(() => this.verRegistro(item), { injector: this.injector }));
    });
  }

  protected buscar(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
  }

  protected actualizar(): void {
    const d = this.def();
    if (d) this.crud.loadList(d.key);
  }

  /** Vuelve a pedir el módulo y las relaciones que no cargaron */
  protected reintentar(): void {
    const d = this.def();
    if (d) this.crud.asegurarConRelaciones(d.key);
  }

  private cerrarDialogos(): void {
    for (const dialogo of [this.hoja()?.nativeElement, this.dialogoRegistro()?.nativeElement]) {
      if (dialogo?.open) dialogo.close();
    }
    this.campos.set([]);
    this.errorGuardado.set('');
  }

  protected nuevo(): void {
    const d = this.def();
    if (!d) return;
    this.campos.set(d.fields.map(field => ({ field, value: '', error: '' })));
    this.errorGuardado.set('');
    this.hoja()?.nativeElement.showModal();
  }

  protected cerrarHoja(): void {
    this.hoja()?.nativeElement.close();
  }

  protected cerrarSiFondo(evento: MouseEvent, dialogo: HTMLDialogElement): void {
    if (evento.target === dialogo) dialogo.close();
  }

  protected textoNuevo(clave: string): string {
    return this.crud.getCrud(clave)?.textos?.nuevo ?? 'Crear registro';
  }

  protected textoVacio(clave: string): string {
    return `Aún no hay ${this.crud.getCrud(clave)?.textos?.plural ?? 'registros'}.`;
  }

  protected guardar(): void {
    const d = this.def();
    if (!d) return;
    const campos = this.campos();
    const valido = validarCampos(campos);
    this.campos.set([...campos]);
    if (!valido) {
      this.toast.show('Error en formulario', 'Corrige los campos marcados', 'err');
      afterNextRender(() => document.querySelector<HTMLElement>('.hoja [aria-invalid="true"]')?.focus(), { injector: this.injector });
      return;
    }
    const formulario = Object.fromEntries(campos.map(c => [c.field.id, { value: String(c.value ?? '') }]));
    this.guardando.set(true);
    this.errorGuardado.set('');
    this.crud.guardar(d.key, d.pay(formulario)).subscribe({
      next: () => {
        this.guardando.set(false);
        this.cerrarHoja();
      },
      error: (e: Error) => {
        this.guardando.set(false);
        this.errorGuardado.set(e.message);
      },
    });
  }

  /** Consulta de la acción de fila: el ID del registro viaja en el parámetro que lee el otro módulo */
  protected consultaAccion(parametro: string, item: Record<string, unknown>): Record<string, string> {
    const id = item[this.def()?.idKey ?? 'id'] ?? item['id'];
    return { [parametro]: String(id) };
  }

  protected verRegistro(item: Record<string, unknown>): void {
    if (!this.def() || !this.mod()) return;
    this.registroAbierto.set(item);
    this.dialogoRegistro()?.nativeElement.showModal();
  }
}
