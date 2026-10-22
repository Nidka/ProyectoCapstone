import { Injectable, inject, signal } from '@angular/core';
import { EMPTY, Observable, catchError, expand, map, of, reduce, tap, throwError } from 'rxjs';
import { ApiService } from './api.service';
import { ToastService } from './toast.service';
import { CrudDef, OpcionRef, RefLookup } from '../models/crud.model';
import { CRUDS } from '../models/cruds.data';

type RecordMap = Record<string, unknown>;

/** Página más grande que acepta el API (limit ≤ 200); las listas se piden completas, página por página. */
const TAM_PAGINA = 200;
/** Tope de seguridad por si un endpoint ignora limit/offset y devuelve siempre páginas llenas. */
const MAX_PAGINAS = 50;

@Injectable({ providedIn: 'root' })
export class CrudService {
  private readonly api = inject(ApiService);
  private readonly toast = inject(ToastService);

  /** Cache local de todos los módulos */
  readonly data = signal<Record<string, RecordMap[]>>({});
  readonly cargando = signal<Record<string, boolean>>({});
  readonly cargado = signal<Record<string, boolean>>({});
  readonly error = signal<Record<string, boolean>>({});

  readonly cruds: CrudDef[] = CRUDS;

  /** Nombre legible de un registro relacionado; se pasa como `ref` a las celdas. */
  readonly ref: RefLookup = (clave, id) => this.etiqueta(clave, id);

  getCrud(key: string): CrudDef | undefined {
    return this.cruds.find(c => c.key === key);
  }

  getItems(key: string): RecordMap[] {
    return this.data()[key] ?? [];
  }

  estaCargando(key: string): boolean {
    return this.cargando()[key] ?? false;
  }

  estaCargado(key: string): boolean {
    return this.cargado()[key] ?? false;
  }

  estaConError(key: string): boolean {
    return this.error()[key] ?? false;
  }

  /** Carga la lista completa desde la API, de 200 en 200, hasta recibir una página corta. */
  loadList(key: string): void {
    const c = this.getCrud(key);
    if (!c) return;
    this.cargando.update(m => ({ ...m, [key]: true }));
    const separador = c.route.includes('?') ? '&' : '?';
    const pagina = (n: number) =>
      this.api
        .get<RecordMap[] | { items?: RecordMap[]; data?: RecordMap[] }>(`${c.route}${separador}limit=${TAM_PAGINA}&offset=${n * TAM_PAGINA}`)
        .pipe(map(res => (Array.isArray(res) ? res : res?.items ?? res?.data ?? [])));
    pagina(0).pipe(
      expand((items, i) => (items.length === TAM_PAGINA && i + 1 < MAX_PAGINAS ? pagina(i + 1) : EMPTY)),
      reduce((todos, items) => todos.concat(items), [] as RecordMap[]),
      catchError(err => {
        this.toast.show('Sin conexión', err?.message ?? '', 'err');
        return of(null);
      }),
    ).subscribe(items => {
      if (items) {
        this.data.update(d => ({ ...d, [key]: items }));
        this.cargado.update(m => ({ ...m, [key]: true }));
      }
      // Una lista que no cargó queda marcada: la página no debe mostrarla como vacía
      this.error.update(m => ({ ...m, [key]: !items }));
      this.cargando.update(m => ({ ...m, [key]: false }));
    });
  }

  asegurarCargado(key: string): void {
    if (!this.estaCargado(key) && !this.estaCargando(key)) this.loadList(key);
  }

  /** Carga un módulo y los módulos que necesita para mostrar nombres en lugar de IDs. */
  asegurarConRelaciones(key: string): void {
    for (const k of [key, ...this.refsDe(key)]) this.asegurarCargado(k);
  }

  /** Claves relacionadas de un módulo, de forma transitiva y sin repetir. */
  refsDe(key: string): string[] {
    const vistos = new Set<string>();
    const pendientes = [key];
    while (pendientes.length) {
      const actual = pendientes.pop()!;
      for (const f of this.getCrud(actual)?.fields ?? []) {
        if (f.tp === 'ref' && f.ref && f.ref !== key && !vistos.has(f.ref)) {
          vistos.add(f.ref);
          pendientes.push(f.ref);
        }
      }
    }
    return [...vistos];
  }

  /** «Agro Norte SAC» si el registro ya cargó; «#5» mientras tanto; «—» si no hay id. */
  etiqueta(clave: string, id: unknown): string {
    if (id == null || id === '') return '—';
    const def = this.getCrud(clave);
    const idKey = def?.idKey;
    const item = idKey ? this.getItems(clave).find(i => String(i[idKey]) === String(id)) : undefined;
    return item && def?.nombre ? def.nombre(item, this.ref) : `#${id}`;
  }

  opciones(clave: string): OpcionRef[] {
    const def = this.getCrud(clave);
    const idKey = def?.idKey;
    if (!def || !idKey) return [];
    return this.getItems(clave).map(i => ({
      valor: String(i[idKey]),
      etiqueta: def.nombre ? def.nombre(i, this.ref) : String(i[idKey]),
    }));
  }

  /** Guarda un registro y emite lo creado; si la API lo rechaza, falla con su detalle para mostrarlo dentro de la hoja. */
  guardar(key: string, payload: RecordMap): Observable<RecordMap> {
    const c = this.getCrud(key);
    if (!c) return throwError(() => new Error('Módulo desconocido'));
    return this.api.post<RecordMap>(c.route, payload).pipe(
      tap(() => {
        this.toast.show(c.textos?.guardado ?? `${c.title} guardado`, 'Registro creado correctamente', 'ok');
        this.loadList(key);
      }),
      catchError(err => {
        const msg = err?.error?.detail ?? err?.message ?? 'Error';
        const texto = Array.isArray(msg) ? msg.map((e: { msg?: string }) => e.msg).join(' · ') : String(msg);
        return throwError(() => new Error(texto));
      }),
    );
  }
}
