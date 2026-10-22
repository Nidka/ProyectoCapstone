# Rediseño "Hitos de carretera" Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rediseñar todo el frontend de LIZAR SRL · Fleet Manager Pro con el mundo visual "Hitos de carretera", navegación por URL con barra superior (y barra inferior en celular) y un flujo de gastos donde el siguiente paso siempre está a un clic.

**Architecture:**
- Un shell autenticado (`/app`) con rutas hijas de carga diferida reemplaza al dashboard de estado interno.
- Un `GastosStore` con signals comparte gastos y comprobantes entre pantallas.
- La lógica pura (filtros, recorrido del gasto, búsqueda, paleta) vive en archivos `*.logic.ts` con pruebas.
- El sistema visual es CSS global por áreas (`src/styles/*.css`). Así se respeta el presupuesto de 8 kB por componente.

**Tech Stack:**
- Angular 21.2 (standalone, signals, `input()`/`model()`, control flow `@if`/`@for`, `animate.leave`)
- Angular Router (`withComponentInputBinding`, `withViewTransitions`)
- Vitest 4 con `@angular/build:unit-test` y jsdom
- Fuentes Overpass y Overpass Mono auto-alojadas con Fontsource
- Iconos Tabler (MIT) copiados como datos de trazo

**Spec:** `docs/superpowers/specs/2026-10-02-rediseno-hitos-design.md`. Contexto de producto en `PRODUCT.md`; contrato de dirección en `.impeccable/surfaces/frontend-src-app.md`.

## Global Constraints

- Trabajar sobre `main` **sin commits ni ramas** (decisión del usuario). Cada tarea termina en un *checkpoint* verificado, no en un commit.
- No modificar `frontend/src/app/core/models/gasto-estados.ts` ni `frontend/src/app/core/utils/error-messages.ts` (copy exacto RNF-1).
- **Copy exacto que debe aparecer tal cual donde ocurre la misma situación:**
  - **Validaciones del gasto:**
    - "Selecciona el viaje al que corresponde este gasto."
    - "Selecciona el cliente responsable de pago."
    - "Ingresa un monto mayor a S/ 0.00."
    - "Selecciona la fecha del gasto."
    - "Este viaje no existe o no está disponible. Verifica el número de viaje."
  - **Avisos de éxito:**
    - "Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago."
    - "Comprobante adjuntado. El gasto cambió a estado "Pagado"."
    - "Comprobante aprobado. El gasto quedó como "Sustentado"."
    - "Comprobante rechazado. El gasto volvió a estado "Registrado"."
  - **Archivo del comprobante:**
    - "Solo se aceptan archivos JPG, PNG o PDF."
    - "El archivo supera el tamaño máximo permitido (10 MB)."
    - "Selecciona al menos un comprobante antes de continuar."
    - "Formatos permitidos: JPG, PNG o PDF. Tamaño máximo 10 MB."
    - "Arrastra el archivo aquí o haz clic para seleccionar"
    - "Este comprobante no se puede editar. Si es incorrecto, adjunta uno nuevo."
    - "Este gasto aún no tiene comprobantes adjuntos."
  - **Rechazo:**
    - "Debes indicar un motivo para rechazar el comprobante."
    - "Motivo del rechazo"
    - placeholder "Explica por qué se rechaza este comprobante (ej. monto no coincide, comprobante ilegible)"
    - "Confirmar rechazo"
  - **Listas vacías:**
    - "No hay comprobantes pendientes de aprobación."
    - "Todavía no hay gastos registrados."
    - "Presiona "Registrar gasto" para crear el primero."
- Textos nuevos: tipo oración, sin raya larga (—), sin signos de exclamación y sin emojis. El "—" que ya existe como valor vacío en tablas y en `formatFecha` se mantiene.
- **Stack:** Angular 21 standalone + signals, CSS propio, sin librería de componentes.
- **Dependencias nuevas permitidas:**
  - runtime: `@fontsource-variable/overpass@^5.3.0` y `@fontsource/overpass-mono@^5.3.0`
  - desarrollo: `vitest@^4.0.8` y `jsdom`
- **Presupuestos de `angular.json`** (no se cambian):
  - bundle inicial: 500 kB de aviso y 1 MB de error
  - estilos por componente: 4 kB de aviso y 8 kB de error
- **Tokens:**
  - base: asfalto `#1E2124`, suelo `#F7F7F5`, panel `#FDFDFC`, línea `#E2E3E0`, tinta `#1E2124` / `#4A5056` / `#656B71`
  - marca y atención: rojo `#E11D48` (hover `#C8113D`, presionado `#AD0E35`), vía `#F2C200`
  - estados (punto / fondo / texto):
    - Registrado: `#8A9096` / `#EEEFEC` / `#4A5056`
    - Pagado: `#2563EB` / `#E8EFFE` / `#1E4FC2`
    - Sustentado: `#15803D` / `#E6F4EA` / `#166534`
    - Pendiente: `#F2C200` / `#FFF5CC` / `#6B5300`
    - Rechazado: `#E11D48` / `#FDECEF` / `#B4123A`
- **Movimiento:**
  - solo `transform`, `opacity` y `clip-path`
  - curvas `--ease-out: cubic-bezier(0.23, 1, 0.32, 1)`, `--ease-in-out: cubic-bezier(0.77, 0, 0.175, 1)` y `--ease-drawer: cubic-bezier(0.32, 0.72, 0, 1)`
  - todo desplazamiento o escala dentro de `@media (prefers-reduced-motion: no-preference)`
  - la paleta ⌘K no se anima
- **Accesibilidad:**
  - foco visible `outline: 2px solid var(--rojo); outline-offset: 2px`
  - objetivos táctiles de 44 px o más en celular
  - etiquetas encima de cada control y errores debajo con `aria-describedby`
- Todos los comandos se ejecutan desde `frontend/` salvo que se indique otra carpeta. Ruta del scratchpad: `SCRATCH=/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad`.

## Review Focus

1. **Datos relacionados que aún no cargan:** una celda o un selector muestra `#5` mientras la lista de clientes viaja y debe cambiar sola al nombre ("Agro Norte SAC") cuando llega. Lo fija la prueba de `CrudService.etiqueta` en la Tarea 5.
2. **Servicio de gastos caído:** la bandeja debe mostrar "No se pudieron cargar los gastos." con "Reintentar", nunca el vacío "Todavía no hay gastos registrados.", que mentiría. Lo fija una prueba de `GastosListaComponent` en la Tarea 12.
3. **Abrir `/app/gastos/999` directamente:** primero un esqueleto mientras carga y luego "No encontramos el gasto N° 999.", nunca una pantalla en blanco. Lo fija una prueba de `GastoDetalleComponent` en la Tarea 15.
4. **Soltar un archivo no permitido o de más de 10 MB:** el mensaje exacto aparece y el botón no sube nada. Lo fija una prueba de `AdjuntarComprobanteComponent` en la Tarea 15.
5. **Registrar un gasto pasadas las 19:00 en Lima:** la fecha por defecto debe ser la de hoy en Perú, no la de mañana en UTC. Lo fija la prueba de `hoyIso` en la Tarea 1.

## File Structure

**Lógica y estado (`frontend/src/app/core/`):**
- `utils/format.ts`: montos `es-PE`, fechas sin hora en UTC, fecha y hora de Lima y hoy en Lima.
- `utils/texto.ts`: normaliza texto para búsquedas sin tildes.
- `state/gastos.logic.ts`: pestañas, conteos, siguiente paso, recorrido (`pasosRuta`) y búsqueda. Son funciones puras.
- `state/gastos.store.ts`: `GastosStore` con gastos, comprobantes, carga y mutaciones.
- `navigation/navegacion.ts`: secciones, módulos y slugs, enlaces principales, paleta y destino tras login.
- `navigation/url-actual.ts`: signal con la URL actual.
- `models/crud.model.ts` y `models/cruds.data.ts`: campos `ref`, nombres legibles y textos con género.
- `services/crud.service.ts`: carga por módulo con estado, relaciones transitivas, etiquetas, opciones y guardado observable.

**UI compartida (`frontend/src/app/shared/`):**
- `ui/icono/`: iconos Tabler.
- `ui/marca/`: camión del logo y marca.
- `ui/insignia-estado/`: insignia de estado.
- `ui/estado-vacio/`: estado vacío que enseña.
- `ui/ruta-gasto/`: **la firma**.
- `ui/selector-referencia/`: combobox con búsqueda.
- `ui/panel-acceso/`: panel de marca del login y del registro.
- `components/toast/`: avisos.

**Pantallas (`frontend/src/app/features/`):**
- `inicio/`
- `gastos/fila-gasto/`
- `gastos/gastos-lista/`
- `gastos/gasto-form/`
- `gastos/gasto-detalle/`
- `gastos/adjuntar-comprobante/`
- `gastos/revision-comprobante/`
- `crud-page/`
- `landing/`
- `login/`
- `register/`
- `no-encontrada/`

**Shell (`frontend/src/app/shell/`):**
- `shell.component.*`
- `barra-superior/`
- `menu-usuario/`
- `barra-movil/`
- `paleta-comandos/`

**Estilos (`frontend/src/styles/`):** `tokens.css`, `base.css`, `componentes.css`, `ruta.css`, `inicio.css`, `gastos.css`, `crud.css`, `shell.css`, `landing.css` y `acceso.css`. `src/styles.css` los importa en ese orden.

**Se eliminan al final (Tarea 20):**
- `features/dashboard/`
- `features/crud-view/`
- `features/gastos/comprobante-upload/` y `features/gastos/aprobacion-comprobantes/`
- `shared/components/sidebar/`, `shared/components/topbar/` y `shared/components/modal/`
- `assets/css/`

---

### Task 1: Respaldo, dependencias, runner de pruebas y formato

**Files:**
- Modify: `frontend/package.json` y `frontend/package-lock.json` (con `npm install`)
- Modify: `frontend/angular.json` (objetivo `test`)
- Modify: `frontend/src/app/core/utils/format.ts`
- Create: `frontend/src/app/core/utils/format.spec.ts`
- Create: `frontend/src/app/core/utils/texto.ts`
- Create: `frontend/src/app/core/utils/texto.spec.ts`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `formatMonto(valor: number | string | undefined | null): string`
  - `formatFecha(fecha: string | undefined | null): string`
  - `formatFechaHora(fecha: string | undefined | null): string`
  - `hoyIso(ahora?: Date): string`
  - `normalizar(texto: string | null | undefined): string`

- [ ] **Step 1: Respaldar el estado actual fuera del repositorio**

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone"
SCRATCH=/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad
STAMP=$(date +%Y%m%d-%H%M%S)
mkdir -p "$SCRATCH/respaldo-$STAMP"
cp -R frontend/src "$SCRATCH/respaldo-$STAMP/src"
cp frontend/package.json frontend/package-lock.json frontend/angular.json "$SCRATCH/respaldo-$STAMP/"
git diff > "$SCRATCH/respaldo-$STAMP/wip.patch"
git status --porcelain > "$SCRATCH/respaldo-$STAMP/status.txt"
ls "$SCRATCH/respaldo-$STAMP"
```

Expected: lista `angular.json package-lock.json package.json src status.txt wip.patch`.

- [ ] **Step 2: Instalar dependencias**

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend"
npm install @fontsource-variable/overpass@^5.3.0 @fontsource/overpass-mono@^5.3.0
npm install --save-dev vitest@^4.0.8 jsdom
```

Expected: ambos comandos terminan sin `ERR!`; `package.json` lista las cuatro dependencias.

- [ ] **Step 3: Agregar el objetivo `test` en `angular.json`**

En `frontend/angular.json`, dentro de `projects.lizar-ng.architect`, después del bloque `"serve": { ... }` (agrega la coma que falte tras su llave de cierre), añade:

```json
        "test": {
          "builder": "@angular/build:unit-test",
          "options": {
            "tsConfig": "tsconfig.spec.json",
            "runner": "vitest"
          }
        }
```

- [ ] **Step 4: Escribir las pruebas que fallan**

`frontend/src/app/core/utils/format.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { formatFecha, formatFechaHora, formatMonto, hoyIso } from './format';

describe('formatMonto', () => {
  it('usa separador de miles y dos decimales de es-PE', () => {
    expect(formatMonto(1240.5)).toBe('S/ 1,240.50');
    expect(formatMonto(1234567.891)).toBe('S/ 1,234,567.89');
  });

  it('acepta texto, nulos y valores inválidos', () => {
    expect(formatMonto('85.5')).toBe('S/ 85.50');
    expect(formatMonto(null)).toBe('S/ 0.00');
    expect(formatMonto('abc')).toBe('S/ 0.00');
  });
});

describe('formatFecha', () => {
  it('lee las fechas sin hora en UTC', () => {
    expect(formatFecha('2026-10-02T00:00:00Z')).toBe('02/10/2026');
  });

  it('muestra el vacío existente cuando no hay fecha', () => {
    expect(formatFecha(null)).toBe('—');
  });
});

describe('formatFechaHora', () => {
  it('muestra la hora de Lima', () => {
    expect(formatFechaHora('2026-10-02T19:05:00Z')).toBe('02/10/2026 14:05');
  });

  it('muestra el vacío existente cuando no hay fecha', () => {
    expect(formatFechaHora(undefined)).toBe('—');
  });
});

describe('hoyIso', () => {
  it('usa el día de Lima, no el de UTC', () => {
    expect(hoyIso(new Date('2026-10-03T02:30:00Z'))).toBe('2026-10-02');
  });
});
```

`frontend/src/app/core/utils/texto.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { normalizar } from './texto';

describe('normalizar', () => {
  it('quita tildes y pasa a minúsculas', () => {
    expect(normalizar('Logística ÁÉÍ Ñandú')).toBe('logistica aei nandu');
  });

  it('trata nulos como texto vacío', () => {
    expect(normalizar(null)).toBe('');
    expect(normalizar(undefined)).toBe('');
  });
});
```

- [ ] **Step 5: Ejecutar y verificar que fallan**

Run: `npx ng test --watch=false`
Expected: FAIL. `formatMonto(1240.5)` devuelve `S/ 1240.50`, y no existen `formatFechaHora`, `hoyIso` ni `./texto`.

- [ ] **Step 6: Implementar**

Reemplaza todo `frontend/src/app/core/utils/format.ts`:

```ts
/** Formato de moneda y fecha en un lenguaje natural para el usuario final. */

const MONTO = new Intl.NumberFormat('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

/** LIZAR opera en Perú: los momentos con hora se muestran en hora de Lima. */
const ZONA = 'America/Lima';

const FECHA_HORA = new Intl.DateTimeFormat('es-PE', {
  timeZone: ZONA,
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
});

const DIA_LIMA = new Intl.DateTimeFormat('en-CA', { timeZone: ZONA });

export function formatMonto(valor: number | string | undefined | null): string {
  const n = Number(valor ?? 0);
  return `S/ ${MONTO.format(Number.isFinite(n) ? n : 0)}`;
}

/** Fechas sin hora (fecha del gasto, fecha programada): se leen en UTC para no correr el día. */
export function formatFecha(fecha: string | undefined | null): string {
  if (!fecha) return '—';
  const d = new Date(fecha);
  if (isNaN(d.getTime())) return fecha.slice(0, 10);
  const dd = String(d.getUTCDate()).padStart(2, '0');
  const mm = String(d.getUTCMonth() + 1).padStart(2, '0');
  const yyyy = d.getUTCFullYear();
  return `${dd}/${mm}/${yyyy}`;
}

/** Momentos con hora (registro, comprobante, aprobación) en hora de Lima: 02/10/2026 14:05. */
export function formatFechaHora(fecha: string | undefined | null): string {
  if (!fecha) return '—';
  const d = new Date(fecha);
  if (isNaN(d.getTime())) return fecha;
  const p = Object.fromEntries(FECHA_HORA.formatToParts(d).map(x => [x.type, x.value]));
  return `${p['day']}/${p['month']}/${p['year']} ${p['hour']}:${p['minute']}`;
}

/** Fecha de hoy en Lima (AAAA-MM-DD) para los campos de tipo fecha. */
export function hoyIso(ahora: Date = new Date()): string {
  return DIA_LIMA.format(ahora);
}
```

Crea `frontend/src/app/core/utils/texto.ts`:

```ts
/** Minúsculas y sin tildes, para comparar búsquedas: «Logística» y «logistica» coinciden. */
export function normalizar(texto: string | null | undefined): string {
  return (texto ?? '').normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase();
}
```

- [ ] **Step 7: Ejecutar y verificar que pasan**

Run: `npx ng test --watch=false`
Expected: PASS. Son 2 archivos y 9 pruebas.

- [ ] **Step 8: Checkpoint (sin commit)**

Run: `npx ng build`
Expected: build exitoso; las pantallas actuales no cambian.

---

### Task 2: Registro de iconos (Tabler, MIT)

**Files:**
- Create: `frontend/src/app/shared/ui/icono/iconos.ts`
- Create: `frontend/src/app/shared/ui/icono/icono.component.ts`
- Test: `frontend/src/app/shared/ui/icono/icono.component.spec.ts`

**Interfaces:**
- Consumes: nada.
- Produces:
  - `ICONOS` (mapa de trazos)
  - `type IconoNombre = keyof typeof ICONOS`
  - `IconoComponent`, selector `app-icono`, con inputs `nombre: IconoNombre` (requerido), `tamano = 18` y `trazo = 1.75`

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/shared/ui/icono/icono.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { IconoComponent } from './icono.component';
import { ICONOS } from './iconos';

describe('ICONOS', () => {
  it('cada icono tiene al menos un trazo', () => {
    for (const [nombre, trazos] of Object.entries(ICONOS)) {
      expect(trazos.length, nombre).toBeGreaterThan(0);
    }
  });
});

describe('IconoComponent', () => {
  it('dibuja los trazos del icono con el tamaño pedido y oculto para lectores', () => {
    const fixture = TestBed.createComponent(IconoComponent);
    fixture.componentRef.setInput('nombre', 'home');
    fixture.componentRef.setInput('tamano', 22);
    fixture.detectChanges();
    const svg: SVGElement = fixture.nativeElement.querySelector('svg');
    expect(svg.getAttribute('width')).toBe('22');
    expect(svg.getAttribute('aria-hidden')).toBe('true');
    expect(svg.querySelectorAll('path').length).toBe(ICONOS.home.length);
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/shared/ui/icono/icono.component.spec.ts`
Expected: FAIL. No se pueden resolver `./icono.component` ni `./iconos`.

- [ ] **Step 3: Implementar**

`frontend/src/app/shared/ui/icono/iconos.ts`:

```ts
/*
 * Iconos: Tabler Icons v3.48.0, variante outline (https://tabler.io/icons).
 * Licencia MIT, (c) Paweł Kuna. Solo se copian los trazos que usa la aplicación.
 */
export const ICONOS = {
  'home': ['M5 12l-2 0l9 -9l9 9l-2 0', 'M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7', 'M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6'],
  'receipt': ['M5 21v-16a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v16l-3 -2l-2 2l-2 -2l-2 2l-2 -2l-3 2m4 -14h6m-6 4h6m-2 4h2'],
  'route': ['M3 19a2 2 0 1 0 4 0a2 2 0 0 0 -4 0', 'M19 7a2 2 0 1 0 0 -4a2 2 0 0 0 0 4', 'M11 19h5.5a3.5 3.5 0 0 0 0 -7h-8a3.5 3.5 0 0 1 0 -7h4.5'],
  'road': ['M4 19l4 -14', 'M16 5l4 14', 'M12 8v-2', 'M12 13v-2', 'M12 18v-2'],
  'folders': ['M9 3h3l2 2h5a2 2 0 0 1 2 2v7a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2v-9a2 2 0 0 1 2 -2', 'M17 16v2a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2v-9a2 2 0 0 1 2 -2h2'],
  'search': ['M3 10a7 7 0 1 0 14 0a7 7 0 1 0 -14 0', 'M21 21l-6 -6'],
  'dots': ['M4 12a1 1 0 1 0 2 0a1 1 0 1 0 -2 0', 'M11 12a1 1 0 1 0 2 0a1 1 0 1 0 -2 0', 'M18 12a1 1 0 1 0 2 0a1 1 0 1 0 -2 0'],
  'logout': ['M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2', 'M9 12h12l-3 -3', 'M18 15l3 -3'],
  'login-2': ['M9 8v-2a2 2 0 0 1 2 -2h7a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-7a2 2 0 0 1 -2 -2v-2', 'M3 12h13l-3 -3', 'M13 15l3 -3'],
  'user': ['M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0', 'M6 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2'],
  'user-plus': ['M8 7a4 4 0 1 0 8 0a4 4 0 0 0 -8 0', 'M16 19h6', 'M19 16v6', 'M6 21v-2a4 4 0 0 1 4 -4h4'],
  'chevron-down': ['M6 9l6 6l6 -6'],
  'chevron-right': ['M9 6l6 6l-6 6'],
  'chevron-left': ['M15 6l-6 6l6 6'],
  'arrow-left': ['M5 12l14 0', 'M5 12l6 6', 'M5 12l6 -6'],
  'arrow-right': ['M5 12l14 0', 'M13 18l6 -6', 'M13 6l6 6'],
  'x': ['M18 6l-12 12', 'M6 6l12 12'],
  'plus': ['M12 5l0 14', 'M5 12l14 0'],
  'refresh': ['M20 11a8.1 8.1 0 0 0 -15.5 -2m-.5 -4v4h4', 'M4 13a8.1 8.1 0 0 0 15.5 2m.5 4v-4h-4'],
  'check': ['M5 12l5 5l10 -10'],
  'circle-check': ['M3 12a9 9 0 1 0 18 0a9 9 0 1 0 -18 0', 'M9 12l2 2l4 -4'],
  'alert-triangle': ['M12 9v4', 'M10.363 3.591l-8.106 13.534a1.914 1.914 0 0 0 1.636 2.871h16.214a1.914 1.914 0 0 0 1.636 -2.87l-8.106 -13.536a1.914 1.914 0 0 0 -3.274 0', 'M12 16h.01'],
  'info-circle': ['M3 12a9 9 0 1 0 18 0a9 9 0 0 0 -18 0', 'M12 9h.01', 'M11 12h1v4h1'],
  'clock': ['M3 12a9 9 0 1 0 18 0a9 9 0 0 0 -18 0', 'M12 7v5l3 3'],
  'circle-x': ['M3 12a9 9 0 1 0 18 0a9 9 0 1 0 -18 0', 'M10 10l4 4m0 -4l-4 4'],
  'cloud-off': ['M9.58 5.548c.24 -.11 .492 -.207 .752 -.286c1.88 -.572 3.956 -.193 5.444 1c1.488 1.19 2.162 3.007 1.77 4.769h.99c1.913 0 3.464 1.56 3.464 3.486c0 .957 -.383 1.824 -1.003 2.454m-2.997 1.033h-11.343c-2.572 -.004 -4.657 -2.011 -4.657 -4.487c0 -2.475 2.085 -4.482 4.657 -4.482c.13 -.582 .37 -1.128 .7 -1.62', 'M3 3l18 18'],
  'paperclip': ['M15 7l-6.5 6.5a1.5 1.5 0 0 0 3 3l6.5 -6.5a3 3 0 0 0 -6 -6l-6.5 6.5a4.5 4.5 0 0 0 9 9l6.5 -6.5'],
  'upload': ['M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2 -2v-2', 'M7 9l5 -5l5 5', 'M12 4l0 12'],
  'camera': ['M5 7h1a2 2 0 0 0 2 -2a1 1 0 0 1 1 -1h6a1 1 0 0 1 1 1a2 2 0 0 0 2 2h1a2 2 0 0 1 2 2v9a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-9a2 2 0 0 1 2 -2', 'M9 13a3 3 0 1 0 6 0a3 3 0 0 0 -6 0'],
  'file-type-pdf': ['M14 3v4a1 1 0 0 0 1 1h4', 'M5 12v-7a2 2 0 0 1 2 -2h7l5 5v4', 'M5 18h1.5a1.5 1.5 0 0 0 0 -3h-1.5v6', 'M17 18h2', 'M20 15h-3v6', 'M11 15v6h1a2 2 0 0 0 2 -2v-2a2 2 0 0 0 -2 -2h-1'],
  'photo': ['M15 8h.01', 'M3 6a3 3 0 0 1 3 -3h12a3 3 0 0 1 3 3v12a3 3 0 0 1 -3 3h-12a3 3 0 0 1 -3 -3v-12', 'M3 16l5 -5c.928 -.893 2.072 -.893 3 0l5 5', 'M14 14l1 -1c.928 -.893 2.072 -.893 3 0l3 3'],
  'external-link': ['M12 6h-6a2 2 0 0 0 -2 2v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-6', 'M11 13l9 -9', 'M15 4h5v5'],
  'clipboard-check': ['M9 5h-2a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-12a2 2 0 0 0 -2 -2h-2', 'M9 5a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2a2 2 0 0 1 -2 2h-2a2 2 0 0 1 -2 -2', 'M9 14l2 2l4 -4'],
  'calendar': ['M4 7a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-12', 'M16 3v4', 'M8 3v4', 'M4 11h16', 'M11 15h1', 'M12 15v3'],
  'map-2': ['M12 18.5l-3 -1.5l-6 3v-13l6 -3l6 3l6 -3v7.5', 'M9 4v13', 'M15 7v5.5', 'M21.121 20.121a3 3 0 1 0 -4.242 0c.418 .419 1.125 1.045 2.121 1.879c1.051 -.89 1.759 -1.516 2.121 -1.879', 'M19 18v.01'],
  'category': ['M4 4h6v6h-6l0 -6', 'M14 4h6v6h-6l0 -6', 'M4 14h6v6h-6l0 -6', 'M14 17a3 3 0 1 0 6 0a3 3 0 1 0 -6 0'],
  'award': ['M6 9a6 6 0 1 0 12 0a6 6 0 1 0 -12 0', 'M12 15l3.4 5.89l1.598 -3.233l3.598 .232l-3.4 -5.889', 'M6.802 12l-3.4 5.89l3.598 -.233l1.598 3.232l3.4 -5.889'],
  'list-details': ['M13 5h8', 'M13 9h5', 'M13 15h8', 'M13 19h5', 'M3 5a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v4a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1l0 -4', 'M3 15a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v4a1 1 0 0 1 -1 1h-4a1 1 0 0 1 -1 -1l0 -4'],
  'list-tree': ['M9 6h11', 'M12 12h8', 'M15 18h5', 'M5 6v.01', 'M8 12v.01', 'M11 18v.01'],
  'map-pin': ['M9 11a3 3 0 1 0 6 0a3 3 0 0 0 -6 0', 'M17.657 16.657l-4.243 4.243a2 2 0 0 1 -2.827 0l-4.244 -4.243a8 8 0 1 1 11.314 0'],
  'id-badge-2': ['M7 12h3v4h-3l0 -4', 'M10 6h-6a1 1 0 0 0 -1 1v12a1 1 0 0 0 1 1h16a1 1 0 0 0 1 -1v-12a1 1 0 0 0 -1 -1h-6', 'M10 4a1 1 0 0 1 1 -1h2a1 1 0 0 1 1 1v3a1 1 0 0 1 -1 1h-2a1 1 0 0 1 -1 -1l0 -3', 'M14 16h2', 'M14 12h4'],
  'building': ['M3 21l18 0', 'M9 8l1 0', 'M9 12l1 0', 'M9 16l1 0', 'M14 8l1 0', 'M14 12l1 0', 'M14 16l1 0', 'M5 21v-16a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v16'],
  'link': ['M9 15l6 -6', 'M11 6l.463 -.536a5 5 0 0 1 7.071 7.072l-.534 .464', 'M13 18l-.397 .534a5.068 5.068 0 0 1 -7.127 0a4.972 4.972 0 0 1 0 -7.071l.524 -.463'],
  'truck': ['M5 17a2 2 0 1 0 4 0a2 2 0 1 0 -4 0', 'M15 17a2 2 0 1 0 4 0a2 2 0 1 0 -4 0', 'M5 17h-2v-11a1 1 0 0 1 1 -1h9v12m-4 0h6m4 0h2v-6h-8m0 -5h5l3 5'],
  'braces': ['M7 4a2 2 0 0 0 -2 2v3a2 3 0 0 1 -2 3a2 3 0 0 1 2 3v3a2 2 0 0 0 2 2', 'M17 4a2 2 0 0 1 2 2v3a2 3 0 0 0 2 3a2 3 0 0 0 -2 3v3a2 2 0 0 1 -2 2'],
  'eye': ['M10 12a2 2 0 1 0 4 0a2 2 0 0 0 -4 0', 'M21 12c-2.4 4 -5.4 6 -9 6c-3.6 0 -6.6 -2 -9 -6c2.4 -4 5.4 -6 9 -6c3.6 0 6.6 2 9 6'],
  'eye-off': ['M10.585 10.587a2 2 0 0 0 2.829 2.828', 'M16.681 16.673a8.717 8.717 0 0 1 -4.681 1.327c-3.6 0 -6.6 -2 -9 -6c1.272 -2.12 2.712 -3.678 4.32 -4.674m2.86 -1.146a9.055 9.055 0 0 1 1.82 -.18c3.6 0 6.6 2 9 6c-.666 1.11 -1.379 2.067 -2.138 2.87', 'M3 3l18 18'],
  'lock': ['M5 13a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2v-6', 'M11 16a1 1 0 1 0 2 0a1 1 0 0 0 -2 0', 'M8 11v-4a4 4 0 1 1 8 0v4'],
  'mail': ['M3 7a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v10a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2v-10', 'M3 7l9 6l9 -6'],
  'inbox': ['M4 6a2 2 0 0 1 2 -2h12a2 2 0 0 1 2 2v12a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2l0 -12', 'M4 13h3l3 3h4l3 -3h3'],
} as const satisfies Record<string, readonly string[]>;

export type IconoNombre = keyof typeof ICONOS;
```

`frontend/src/app/shared/ui/icono/icono.component.ts`:

```ts
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
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/shared/ui/icono/icono.component.spec.ts`
Expected: PASS (2 pruebas).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false`
Expected: PASS en todas las pruebas.

---

### Task 3: Lógica pura del gasto

**Files:**
- Create: `frontend/src/app/core/state/gastos.logic.ts`
- Test: `frontend/src/app/core/state/gastos.logic.spec.ts`

**Interfaces:**
- Consumes:
  - `Gasto`, `Comprobante` y `EstadoGasto` (de `core/models/gasto.model.ts`)
  - `normalizar` (de la Tarea 1)
- Produces:
  - `type Pestana = 'todos' | 'sin-comprobante' | 'por-aprobar' | 'sustentados'`
  - `PESTANAS` y `ORDEN_ESTADOS`
  - `ESTADO_DE_PESTANA` y `PESTANA_DE_ESTADO`
  - `type SiguientePaso = 'adjuntar' | 'revisar' | 'ninguno'`
  - `interface PasoRuta { estado: EstadoGasto; alcanzado: boolean; actual: boolean; fecha?: string; actor?: string | null }`
  - Funciones:
    - `esPestana(v): v is Pestana`
    - `filtrarPorPestana(gastos, pestana): Gasto[]`
    - `contarPorEstado(gastos): Record<EstadoGasto, number>`
    - `sumaPorEstado(gastos): Record<EstadoGasto, number>`
    - `totalMonto(gastos): number`
    - `siguientePaso(gasto): SiguientePaso`
    - `comprobantesDe(comprobantes, gastoId): Comprobante[]`
    - `ultimoRechazo(comprobantes, gastoId): Comprobante | undefined`
    - `pasosRuta(gasto, comprobantes): PasoRuta[]`
    - `etiquetaRuta(pasos): string`
    - `ordenarRecientes(gastos): Gasto[]`
    - `recientes(gastos, cantidad): Gasto[]`
    - `buscarGastos(gastos, termino, nombreCliente: (id: number) => string): Gasto[]`

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/core/state/gastos.logic.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { Comprobante, Gasto } from '../models/gasto.model';
import {
  buscarGastos, comprobantesDe, contarPorEstado, esPestana, etiquetaRuta, filtrarPorPestana,
  ordenarRecientes, pasosRuta, recientes, siguientePaso, sumaPorEstado, totalMonto, ultimoRechazo,
} from './gastos.logic';

function gasto(p: Partial<Gasto> & Pick<Gasto, 'id' | 'estado'>): Gasto {
  return {
    viaje_id: 1, cliente_id: 1, monto: 100, fecha: '2026-10-01T00:00:00Z',
    creado_por: 'Ana Quispe', creado_en: '2026-10-01T15:00:00Z', ...p,
  };
}

function comprobante(p: Partial<Comprobante> & Pick<Comprobante, 'id' | 'gasto_id' | 'estado'>): Comprobante {
  return { archivo_url: `/uploads/${p.id}.jpg`, tipo_archivo: 'imagen', creado_en: '2026-10-01T16:00:00Z', ...p };
}

const gastos = [
  gasto({ id: 1, estado: 'Registrado', monto: 120 }),
  gasto({ id: 2, estado: 'Pagado', monto: 85.5 }),
  gasto({ id: 3, estado: 'Sustentado', monto: 310 }),
  gasto({ id: 4, estado: 'Registrado', monto: 40.25, cliente_id: 2 }),
];

describe('pestañas', () => {
  it('reconoce solo las pestañas válidas', () => {
    expect(esPestana('por-aprobar')).toBe(true);
    expect(esPestana('otra')).toBe(false);
    expect(esPestana(undefined)).toBe(false);
  });

  it('filtra por el estado de cada pestaña', () => {
    expect(filtrarPorPestana(gastos, 'todos').map(g => g.id)).toEqual([1, 2, 3, 4]);
    expect(filtrarPorPestana(gastos, 'sin-comprobante').map(g => g.id)).toEqual([1, 4]);
    expect(filtrarPorPestana(gastos, 'por-aprobar').map(g => g.id)).toEqual([2]);
    expect(filtrarPorPestana(gastos, 'sustentados').map(g => g.id)).toEqual([3]);
  });
});

describe('conteos y montos', () => {
  it('cuenta y suma por estado', () => {
    expect(contarPorEstado(gastos)).toEqual({ Registrado: 2, Pagado: 1, Sustentado: 1 });
    expect(sumaPorEstado(gastos)).toEqual({ Registrado: 160.25, Pagado: 85.5, Sustentado: 310 });
    expect(totalMonto(gastos)).toBeCloseTo(555.75);
  });
});

describe('siguientePaso', () => {
  it('señala la acción que toca', () => {
    expect(siguientePaso({ estado: 'Registrado' })).toBe('adjuntar');
    expect(siguientePaso({ estado: 'Pagado' })).toBe('revisar');
    expect(siguientePaso({ estado: 'Sustentado' })).toBe('ninguno');
  });
});

describe('comprobantes de un gasto', () => {
  const lista = [
    comprobante({ id: 10, gasto_id: 1, estado: 'Rechazado', creado_en: '2026-10-01T16:00:00Z', motivo_rechazo: 'Ilegible' }),
    comprobante({ id: 11, gasto_id: 1, estado: 'Pendiente', creado_en: '2026-10-02T09:00:00-05:00' }),
    comprobante({ id: 12, gasto_id: 2, estado: 'Pendiente' }),
  ];

  it('ordena del más reciente al más antiguo', () => {
    expect(comprobantesDe(lista, 1).map(c => c.id)).toEqual([11, 10]);
  });

  it('solo devuelve el rechazo si es el último comprobante', () => {
    expect(ultimoRechazo(lista, 1)).toBeUndefined();
    expect(ultimoRechazo(lista.slice(0, 1), 1)?.motivo_rechazo).toBe('Ilegible');
  });
});

describe('pasosRuta', () => {
  it('un gasto registrado está en el primer hito con fecha y autor', () => {
    const pasos = pasosRuta(gastos[0], []);
    expect(pasos.map(p => [p.estado, p.alcanzado, p.actual])).toEqual([
      ['Registrado', true, true], ['Pagado', false, false], ['Sustentado', false, false],
    ]);
    expect(pasos[0].fecha).toBe('2026-10-01T15:00:00Z');
    expect(pasos[0].actor).toBe('Ana Quispe');
  });

  it('un gasto sustentado toma las fechas del comprobante aprobado', () => {
    const aprobado = comprobante({
      id: 20, gasto_id: 3, estado: 'Aprobado', creado_en: '2026-10-01T17:00:00Z',
      aprobado_en: '2026-10-02T10:00:00Z', aprobado_por: 'Encargado de Logística',
    });
    const pasos = pasosRuta(gastos[2], [aprobado]);
    expect(pasos.map(p => p.actual)).toEqual([false, false, true]);
    expect(pasos[1].fecha).toBe('2026-10-01T17:00:00Z');
    expect(pasos[2].fecha).toBe('2026-10-02T10:00:00Z');
    expect(pasos[2].actor).toBe('Encargado de Logística');
  });

  it('describe el recorrido para lectores de pantalla', () => {
    const pasos = pasosRuta(gastos[1], [comprobante({ id: 30, gasto_id: 2, estado: 'Pendiente' })]);
    expect(etiquetaRuta(pasos)).toBe('Recorrido: Registrado, Pagado (actual), Sustentado pendiente');
  });
});

describe('orden y búsqueda', () => {
  const conFechas = [
    gasto({ id: 5, estado: 'Registrado', creado_en: '2026-10-01T10:00:00Z' }),
    gasto({ id: 6, estado: 'Registrado', creado_en: '2026-10-03T10:00:00Z' }),
    gasto({ id: 7, estado: 'Registrado', creado_en: '2026-10-03T10:00:00Z' }),
  ];

  it('ordena por registro más reciente y luego por número', () => {
    expect(ordenarRecientes(conFechas).map(g => g.id)).toEqual([7, 6, 5]);
    expect(recientes(conFechas, 2).map(g => g.id)).toEqual([7, 6]);
  });

  const nombres: Record<number, string> = { 1: 'Agro Norte SAC', 2: 'Logística Andina' };
  const cliente = (id: number) => nombres[id] ?? '';

  it('busca por número con o sin prefijo', () => {
    expect(buscarGastos(gastos, '2', cliente).map(g => g.id)).toEqual([2]);
    expect(buscarGastos(gastos, 'N° 3', cliente).map(g => g.id)).toEqual([3]);
    expect(buscarGastos(gastos, '#4', cliente).map(g => g.id)).toEqual([4]);
  });

  it('busca por cliente sin importar tildes', () => {
    expect(buscarGastos(gastos, 'agro', cliente).map(g => g.id)).toEqual([1, 2, 3]);
    expect(buscarGastos(gastos, 'logistica', cliente).map(g => g.id)).toEqual([4]);
    expect(buscarGastos(gastos, '   ', cliente)).toHaveLength(4);
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/core/state/gastos.logic.spec.ts`
Expected: FAIL. No se puede resolver `./gastos.logic`.

- [ ] **Step 3: Implementar**

`frontend/src/app/core/state/gastos.logic.ts`:

```ts
import { Comprobante, EstadoGasto, Gasto } from '../models/gasto.model';
import { normalizar } from '../utils/texto';

export type Pestana = 'todos' | 'sin-comprobante' | 'por-aprobar' | 'sustentados';

export const PESTANAS: readonly Pestana[] = ['todos', 'sin-comprobante', 'por-aprobar', 'sustentados'];

/** Orden del recorrido de un gasto. */
export const ORDEN_ESTADOS: readonly EstadoGasto[] = ['Registrado', 'Pagado', 'Sustentado'];

export const ESTADO_DE_PESTANA: Record<Exclude<Pestana, 'todos'>, EstadoGasto> = {
  'sin-comprobante': 'Registrado',
  'por-aprobar': 'Pagado',
  sustentados: 'Sustentado',
};

export const PESTANA_DE_ESTADO: Record<EstadoGasto, Exclude<Pestana, 'todos'>> = {
  Registrado: 'sin-comprobante',
  Pagado: 'por-aprobar',
  Sustentado: 'sustentados',
};

export type SiguientePaso = 'adjuntar' | 'revisar' | 'ninguno';

export interface PasoRuta {
  estado: EstadoGasto;
  alcanzado: boolean;
  actual: boolean;
  fecha?: string;
  actor?: string | null;
}

export function esPestana(valor: unknown): valor is Pestana {
  return typeof valor === 'string' && (PESTANAS as readonly string[]).includes(valor);
}

export function filtrarPorPestana(gastos: Gasto[], pestana: Pestana): Gasto[] {
  if (pestana === 'todos') return gastos;
  const estado = ESTADO_DE_PESTANA[pestana];
  return gastos.filter(g => g.estado === estado);
}

export function contarPorEstado(gastos: Gasto[]): Record<EstadoGasto, number> {
  const conteo: Record<EstadoGasto, number> = { Registrado: 0, Pagado: 0, Sustentado: 0 };
  for (const g of gastos) if (g.estado in conteo) conteo[g.estado]++;
  return conteo;
}

export function sumaPorEstado(gastos: Gasto[]): Record<EstadoGasto, number> {
  const suma: Record<EstadoGasto, number> = { Registrado: 0, Pagado: 0, Sustentado: 0 };
  for (const g of gastos) if (g.estado in suma) suma[g.estado] += Number(g.monto) || 0;
  return suma;
}

export function totalMonto(gastos: Gasto[]): number {
  return gastos.reduce((acc, g) => acc + (Number(g.monto) || 0), 0);
}

export function siguientePaso(gasto: Pick<Gasto, 'estado'>): SiguientePaso {
  switch (gasto.estado) {
    case 'Registrado':
      return 'adjuntar';
    case 'Pagado':
      return 'revisar';
    default:
      return 'ninguno';
  }
}

function momento(fecha: string | null | undefined): number {
  const t = fecha ? Date.parse(fecha) : NaN;
  return isNaN(t) ? 0 : t;
}

/** Comprobantes de un gasto, del más reciente al más antiguo. */
export function comprobantesDe(comprobantes: Comprobante[], gastoId: number): Comprobante[] {
  return comprobantes
    .filter(c => c.gasto_id === gastoId)
    .sort((a, b) => momento(b.creado_en) - momento(a.creado_en));
}

/** Si el último comprobante del gasto fue rechazado, lo devuelve para mostrar el motivo. */
export function ultimoRechazo(comprobantes: Comprobante[], gastoId: number): Comprobante | undefined {
  const ultimo = comprobantesDe(comprobantes, gastoId)[0];
  return ultimo?.estado === 'Rechazado' ? ultimo : undefined;
}

/** Los tres hitos del gasto con la fecha y quién en cada uno. */
export function pasosRuta(gasto: Gasto, comprobantes: Comprobante[]): PasoRuta[] {
  const indice = ORDEN_ESTADOS.indexOf(gasto.estado);
  const lista = comprobantesDe(comprobantes, gasto.id);
  const aprobado = lista.find(c => c.estado === 'Aprobado');
  const pendiente = lista.find(c => c.estado === 'Pendiente');
  return ORDEN_ESTADOS.map((estado, i) => {
    const paso: PasoRuta = { estado, alcanzado: i <= indice, actual: i === indice };
    if (estado === 'Registrado') {
      paso.fecha = gasto.creado_en;
      paso.actor = gasto.creado_por;
    } else if (estado === 'Pagado' && paso.alcanzado) {
      paso.fecha = (aprobado ?? pendiente)?.creado_en;
    } else if (estado === 'Sustentado' && paso.alcanzado && aprobado) {
      paso.fecha = aprobado.aprobado_en ?? undefined;
      paso.actor = aprobado.aprobado_por ?? null;
    }
    return paso;
  });
}

export function etiquetaRuta(pasos: PasoRuta[]): string {
  const partes = pasos.map(p => (p.actual ? `${p.estado} (actual)` : p.alcanzado ? p.estado : `${p.estado} pendiente`));
  return `Recorrido: ${partes.join(', ')}`;
}

export function ordenarRecientes(gastos: Gasto[]): Gasto[] {
  return [...gastos].sort((a, b) => momento(b.creado_en) - momento(a.creado_en) || b.id - a.id);
}

export function recientes(gastos: Gasto[], cantidad: number): Gasto[] {
  return ordenarRecientes(gastos).slice(0, cantidad);
}

/** Busca por N° («31», «N° 31», «#31») o por nombre del cliente, sin distinguir tildes. */
export function buscarGastos(gastos: Gasto[], termino: string, nombreCliente: (clienteId: number) => string): Gasto[] {
  const t = normalizar(termino).trim();
  if (!t) return gastos;
  const numero = t.match(/^(?:n[°º.]?\s*|#\s*)?(\d+)$/)?.[1];
  if (numero) return gastos.filter(g => String(g.id) === numero);
  return gastos.filter(g => normalizar(nombreCliente(g.cliente_id)).includes(t));
}
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/core/state/gastos.logic.spec.ts`
Expected: PASS (12 pruebas).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false`
Expected: PASS en todas las pruebas.

---

### Task 4: Navegación (secciones, slugs, paleta y destino tras login)

**Files:**
- Create: `frontend/src/app/core/navigation/navegacion.ts`
- Create: `frontend/src/app/core/navigation/url-actual.ts`
- Test: `frontend/src/app/core/navigation/navegacion.spec.ts`

**Interfaces:**
- Consumes: `IconoNombre` (Tarea 2).
- Produces:
  - Tipos:
    - `type SeccionId = 'operacion' | 'catalogos'`
    - `interface ModuloNav { slug; clave; etiqueta; icono: IconoNombre; seccion: SeccionId }`
    - `interface SeccionNav { id; etiqueta; icono; modulos: ModuloNav[] }`
    - `interface EnlacePrincipal { etiqueta; ruta; icono }`
    - `type GrupoPaleta = 'Acciones' | 'Ir a' | 'Gastos'`
    - `interface EntradaPaleta { id; grupo; etiqueta; detalle?; icono; ruta; queryParams? }`
  - Constantes: `SECCIONES`, `MODULOS`, `ENLACES_PRINCIPALES`, `ACCIONES_PALETA`
  - Funciones:
    - `seccion(id): SeccionNav`
    - `esSeccion(v): v is SeccionId`
    - `moduloPorSlug(seccionId, slug): ModuloNav | undefined`
    - `moduloPorClave(clave): ModuloNav | undefined`
    - `rutaModulo(m): string`
    - `rutaDeClave(clave): string | null`
    - `destinosPaleta(): EntradaPaleta[]`
    - `destinoTrasLogin(volver): string`
    - `urlActual(): Signal<string>` (en contexto de inyección)

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/core/navigation/navegacion.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import {
  ACCIONES_PALETA, ENLACES_PRINCIPALES, MODULOS, SECCIONES, destinoTrasLogin, destinosPaleta,
  esSeccion, moduloPorClave, moduloPorSlug, rutaDeClave, rutaModulo, seccion,
} from './navegacion';

describe('módulos visibles', () => {
  it('cubre exactamente las 11 claves CRUD de esta iteración', () => {
    expect(MODULOS.map(m => m.clave).sort()).toEqual(
      ['cliente', 'cond_uni', 'conductor', 'direccion', 'marca', 'rubro', 'subtipo_unidad', 'tipo_unidad', 'ubigeo', 'unidad', 'viaje'],
    );
  });

  it('no repite slugs dentro de una sección', () => {
    for (const s of SECCIONES) {
      const slugs = s.modulos.map(m => m.slug);
      expect(new Set(slugs).size).toBe(slugs.length);
    }
  });

  it('resuelve módulos por slug y por clave', () => {
    expect(moduloPorSlug('operacion', 'flota')?.clave).toBe('unidad');
    expect(moduloPorSlug('catalogos', 'flota')).toBeUndefined();
    expect(moduloPorClave('cond_uni')?.slug).toBe('asignaciones');
    expect(seccion('catalogos').modulos[0].slug).toBe('ubigeo');
    expect(esSeccion('operacion')).toBe(true);
    expect(esSeccion('gastos')).toBe(false);
  });

  it('arma las rutas de cada módulo', () => {
    expect(rutaModulo(moduloPorClave('viaje')!)).toBe('/app/operacion/viajes');
    expect(rutaDeClave('tipo_unidad')).toBe('/app/catalogos/tipos-unidad');
    expect(rutaDeClave('factura')).toBeNull();
  });
});

describe('enlaces y paleta', () => {
  it('la barra tiene Inicio, Gastos, Operación y Catálogos', () => {
    expect(ENLACES_PRINCIPALES.map(e => e.ruta)).toEqual(['/app/inicio', '/app/gastos', '/app/operacion', '/app/catalogos']);
  });

  it('la paleta ofrece las acciones del flujo y todos los destinos', () => {
    expect(ACCIONES_PALETA.map(a => a.etiqueta)).toEqual(['Registrar gasto', 'Adjuntar comprobante', 'Revisar comprobantes por aprobar']);
    expect(destinosPaleta()).toHaveLength(2 + MODULOS.length);
    expect(destinosPaleta().find(d => d.etiqueta === 'Flota')?.detalle).toBe('Operación');
  });
});

describe('destinoTrasLogin', () => {
  it('vuelve a la pantalla pedida solo si es de la app', () => {
    expect(destinoTrasLogin('/app/gastos?estado=por-aprobar')).toBe('/app/gastos?estado=por-aprobar');
    expect(destinoTrasLogin('https://otro.sitio')).toBe('/app/inicio');
    expect(destinoTrasLogin('/app')).toBe('/app/inicio');
    expect(destinoTrasLogin(null)).toBe('/app/inicio');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/core/navigation/navegacion.spec.ts`
Expected: FAIL. No se puede resolver `./navegacion`.

- [ ] **Step 3: Implementar**

`frontend/src/app/core/navigation/navegacion.ts`:

```ts
import type { IconoNombre } from '../../shared/ui/icono/iconos';

export type SeccionId = 'operacion' | 'catalogos';

export interface ModuloNav {
  slug: string;
  clave: string;
  etiqueta: string;
  icono: IconoNombre;
  seccion: SeccionId;
}

export interface SeccionNav {
  id: SeccionId;
  etiqueta: string;
  icono: IconoNombre;
  modulos: ModuloNav[];
}

export interface EnlacePrincipal {
  etiqueta: string;
  ruta: string;
  icono: IconoNombre;
}

export type GrupoPaleta = 'Acciones' | 'Ir a' | 'Gastos';

export interface EntradaPaleta {
  id: string;
  grupo: GrupoPaleta;
  etiqueta: string;
  detalle?: string;
  icono: IconoNombre;
  ruta: string;
  queryParams?: Record<string, string>;
}

const OPERACION: ModuloNav[] = [
  { slug: 'viajes', clave: 'viaje', etiqueta: 'Viajes', icono: 'road', seccion: 'operacion' },
  { slug: 'clientes', clave: 'cliente', etiqueta: 'Clientes', icono: 'building', seccion: 'operacion' },
  { slug: 'conductores', clave: 'conductor', etiqueta: 'Conductores', icono: 'id-badge-2', seccion: 'operacion' },
  { slug: 'flota', clave: 'unidad', etiqueta: 'Flota', icono: 'truck', seccion: 'operacion' },
  { slug: 'asignaciones', clave: 'cond_uni', etiqueta: 'Asignaciones', icono: 'link', seccion: 'operacion' },
  { slug: 'direcciones', clave: 'direccion', etiqueta: 'Direcciones', icono: 'map-pin', seccion: 'operacion' },
];

const CATALOGOS: ModuloNav[] = [
  { slug: 'ubigeo', clave: 'ubigeo', etiqueta: 'Ubigeo', icono: 'map-2', seccion: 'catalogos' },
  { slug: 'rubros', clave: 'rubro', etiqueta: 'Rubros', icono: 'category', seccion: 'catalogos' },
  { slug: 'marcas', clave: 'marca', etiqueta: 'Marcas', icono: 'award', seccion: 'catalogos' },
  { slug: 'tipos-unidad', clave: 'tipo_unidad', etiqueta: 'Tipos de unidad', icono: 'list-details', seccion: 'catalogos' },
  { slug: 'subtipos-unidad', clave: 'subtipo_unidad', etiqueta: 'Subtipos de unidad', icono: 'list-tree', seccion: 'catalogos' },
];

export const SECCIONES: readonly SeccionNav[] = [
  { id: 'operacion', etiqueta: 'Operación', icono: 'route', modulos: OPERACION },
  { id: 'catalogos', etiqueta: 'Catálogos', icono: 'folders', modulos: CATALOGOS },
];

export const MODULOS: readonly ModuloNav[] = [...OPERACION, ...CATALOGOS];

export const ENLACES_PRINCIPALES: readonly EnlacePrincipal[] = [
  { etiqueta: 'Inicio', ruta: '/app/inicio', icono: 'home' },
  { etiqueta: 'Gastos', ruta: '/app/gastos', icono: 'receipt' },
  { etiqueta: 'Operación', ruta: '/app/operacion', icono: 'route' },
  { etiqueta: 'Catálogos', ruta: '/app/catalogos', icono: 'folders' },
];

export const ACCIONES_PALETA: readonly EntradaPaleta[] = [
  { id: 'accion-registrar', grupo: 'Acciones', etiqueta: 'Registrar gasto', icono: 'plus', ruta: '/app/gastos/nuevo' },
  {
    id: 'accion-adjuntar', grupo: 'Acciones', etiqueta: 'Adjuntar comprobante', icono: 'paperclip',
    ruta: '/app/gastos', queryParams: { estado: 'sin-comprobante' },
  },
  {
    id: 'accion-revisar', grupo: 'Acciones', etiqueta: 'Revisar comprobantes por aprobar', icono: 'clipboard-check',
    ruta: '/app/gastos', queryParams: { estado: 'por-aprobar' },
  },
];

export function seccion(id: SeccionId): SeccionNav {
  return SECCIONES.find(s => s.id === id) ?? SECCIONES[0];
}

export function esSeccion(valor: unknown): valor is SeccionId {
  return valor === 'operacion' || valor === 'catalogos';
}

export function moduloPorSlug(seccionId: SeccionId, slug: string | null | undefined): ModuloNav | undefined {
  return seccion(seccionId).modulos.find(m => m.slug === slug);
}

export function moduloPorClave(clave: string): ModuloNav | undefined {
  return MODULOS.find(m => m.clave === clave);
}

export function rutaModulo(modulo: ModuloNav): string {
  return `/app/${modulo.seccion}/${modulo.slug}`;
}

export function rutaDeClave(clave: string): string | null {
  const modulo = moduloPorClave(clave);
  return modulo ? rutaModulo(modulo) : null;
}

export function destinosPaleta(): EntradaPaleta[] {
  return [
    { id: 'ir-inicio', grupo: 'Ir a', etiqueta: 'Inicio', icono: 'home', ruta: '/app/inicio' },
    { id: 'ir-gastos', grupo: 'Ir a', etiqueta: 'Gastos', icono: 'receipt', ruta: '/app/gastos' },
    ...MODULOS.map<EntradaPaleta>(m => ({
      id: `ir-${m.seccion}-${m.slug}`,
      grupo: 'Ir a',
      etiqueta: m.etiqueta,
      detalle: seccion(m.seccion).etiqueta,
      icono: m.icono,
      ruta: rutaModulo(m),
    })),
  ];
}

/** Tras iniciar sesión se vuelve a la pantalla pedida solo si es parte de la app. */
export function destinoTrasLogin(volver: string | null | undefined): string {
  return volver && volver.startsWith('/app/') ? volver : '/app/inicio';
}
```

`frontend/src/app/core/navigation/url-actual.ts`:

```ts
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
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/core/navigation/navegacion.spec.ts`
Expected: PASS (7 pruebas).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false`
Expected: PASS en todas las pruebas.

---
### Task 5: Módulos CRUD con relaciones por nombre y `CrudService` con estado de carga

**Files:**
- Modify (reemplazo completo): `frontend/src/app/core/models/crud.model.ts`
- Modify (reemplazo completo): `frontend/src/app/core/models/cruds.data.ts`
- Modify (reemplazo completo): `frontend/src/app/core/services/crud.service.ts`
- Modify: `frontend/src/app/features/crud-view/crud-view.component.ts:115` (compatibilidad temporal; se borra en la Tarea 20)
- Test: `frontend/src/app/core/services/crud.service.spec.ts`
- Test: `frontend/src/app/core/models/cruds.data.spec.ts`

**Interfaces:**
- Consumes: `ApiService`, `ToastService` y `formatFecha` (Tarea 1).
- Produces:
  - `type RefLookup = (clave: string, id: unknown) => string`
  - `CrudField.tp` admite `'ref'` y agrega `CrudField.ref?: string`
  - `CrudDef` agrega:
    - `idKey?: string`
    - `nombre?: (item, ref: RefLookup) => string`
    - `textos?: { nuevo; guardado; singular; plural }`
  - `CrudDef.row` pasa a `(item, ref: RefLookup) => string[]`
  - `CrudDef.pay` pasa a `(form: Record<string, { value: string }>) => Record<string, unknown>`
  - `interface OpcionRef { valor: string; etiqueta: string }` (en `crud.model.ts`)
  - `CrudService`:
    - signals: `data`, `cargando`, `cargado`
    - `ref: RefLookup`
    - consultas: `getCrud`, `getItems`, `estaCargando(key)` y `estaCargado(key)`
    - carga: `loadList(key)`, `asegurarCargado(key)` y `asegurarConRelaciones(key)`
    - relaciones: `refsDe(key): string[]`, `etiqueta(clave, id): string` y `opciones(clave): OpcionRef[]`
    - guardado: `guardar(key, payload): Observable<Record<string, unknown> | null>`
    - compatibilidad temporal: `saveItem` y `refreshAll`

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/core/services/crud.service.spec.ts`:

```ts
import { beforeEach, describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { CrudService } from './crud.service';
import { ToastService } from './toast.service';

describe('CrudService', () => {
  let crud: CrudService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
    crud = TestBed.inject(CrudService);
    http = TestBed.inject(HttpTestingController);
  });

  it('refsDe encuentra las relaciones de forma transitiva', () => {
    expect(crud.refsDe('cliente')).toEqual(['rubro']);
    expect(crud.refsDe('rubro')).toEqual([]);
    expect(new Set(crud.refsDe('viaje'))).toEqual(new Set([
      'unidad', 'cond_uni', 'cliente', 'direccion', 'tipo_unidad', 'subtipo_unidad', 'marca', 'conductor', 'rubro', 'ubigeo',
    ]));
  });

  it('etiqueta muestra #id hasta que llega la lista y luego el nombre', () => {
    expect(crud.etiqueta('cliente', 5)).toBe('#5');
    crud.data.set({ cliente: [{ id_cliente: 5, razon_social: 'Agro Norte SAC' }] });
    expect(crud.etiqueta('cliente', 5)).toBe('Agro Norte SAC');
    expect(crud.etiqueta('cliente', null)).toBe('—');
  });

  it('las etiquetas compuestas usan otras relaciones', () => {
    crud.data.set({
      conductor: [{ id_conductor: 4, nombre1: 'Juan', apellido1: 'Pérez' }],
      unidad: [{ id_unidad: 2, placa: 'ABC-123', modelo: 'Actros' }],
      cond_uni: [{ id_conductor_unidad: 9, id_conductor: 4, id_unidad: 2 }],
    });
    expect(crud.etiqueta('cond_uni', 9)).toBe('Juan Pérez · ABC-123 · Actros');
  });

  it('opciones arma valor y etiqueta para los selectores', () => {
    crud.data.set({ rubro: [{ id_rubro: 3, nombre: 'Agroindustria' }] });
    expect(crud.opciones('rubro')).toEqual([{ valor: '3', etiqueta: 'Agroindustria' }]);
  });

  it('loadList marca la carga y guarda los registros', () => {
    crud.loadList('rubro');
    expect(crud.estaCargando('rubro')).toBe(true);
    http.expectOne(r => r.url.endsWith('/rubros')).flush([{ id_rubro: 1, nombre: 'Minería' }]);
    expect(crud.estaCargando('rubro')).toBe(false);
    expect(crud.estaCargado('rubro')).toBe(true);
    expect(crud.getItems('rubro')).toHaveLength(1);
  });

  it('asegurarCargado no repite la petición', () => {
    crud.asegurarCargado('marca');
    crud.asegurarCargado('marca');
    http.expectOne(r => r.url.endsWith('/marcas')).flush([]);
    http.verify();
  });

  it('guardar emite el registro creado y avisa con el texto del módulo', () => {
    let resultado: unknown = 'sin respuesta';
    crud.guardar('marca', { nombre: 'Volvo' }).subscribe(r => (resultado = r));
    http.expectOne(r => r.method === 'POST' && r.url.endsWith('/marcas')).flush({ id_marca: 7, nombre: 'Volvo' });
    expect(resultado).toEqual({ id_marca: 7, nombre: 'Volvo' });
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Marca guardada');
    http.expectOne(r => r.method === 'GET' && r.url.endsWith('/marcas')).flush([]);
  });

  it('guardar emite null y muestra el detalle si la API rechaza', () => {
    let resultado: unknown = 'sin respuesta';
    crud.guardar('marca', {}).subscribe(r => (resultado = r));
    http.expectOne(r => r.method === 'POST').flush({ detail: 'nombre requerido' }, { status: 422, statusText: 'Unprocessable' });
    expect(resultado).toBeNull();
    expect(TestBed.inject(ToastService).toasts()[0].msg).toBe('nombre requerido');
  });
});
```

`frontend/src/app/core/models/cruds.data.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { CRUDS } from './cruds.data';

const def = (clave: string) => CRUDS.find(c => c.key === clave)!;
const ref = (clave: string, id: unknown) => `${clave}:${id}`;
const VISIBLES = ['ubigeo', 'rubro', 'marca', 'tipo_unidad', 'subtipo_unidad', 'direccion', 'conductor', 'cliente', 'unidad', 'cond_uni', 'viaje'];

describe('cruds.data', () => {
  it('los módulos visibles tienen id, nombre y textos con género', () => {
    for (const clave of VISIBLES) {
      const d = def(clave);
      expect(d.idKey, clave).toBeTruthy();
      expect(d.nombre, clave).toBeTypeOf('function');
      expect(d.textos?.nuevo, clave).toBeTruthy();
    }
    expect(def('direccion').textos?.nuevo).toBe('Nueva dirección');
  });

  it('los campos relacionados se eligen por nombre, no por ID', () => {
    const etiquetas = VISIBLES.flatMap(clave => def(clave).fields.map(f => f.l));
    expect(etiquetas.filter(l => /^ID /.test(l))).toEqual([]);
    expect(def('viaje').fields.filter(f => f.tp === 'ref').map(f => f.ref))
      .toEqual(['unidad', 'unidad', 'cond_uni', 'cliente', 'direccion', 'direccion']);
  });

  it('las celdas muestran nombres relacionados y escapan HTML', () => {
    const celdas = def('unidad').row({ placa: 'ABC-123', modelo: '<b>x</b>', id_marca: 2, id_tipo_unidad: 1, anio: 2020 }, ref);
    expect(celdas[1]).toContain('&lt;b&gt;x&lt;/b&gt;');
    expect(celdas[2]).toContain('marca:2');
    expect(celdas[3]).toContain('tipo_unidad:1');
  });

  it('los textos visibles de los módulos no usan raya larga ni emojis', () => {
    for (const clave of VISIBLES) {
      const d = def(clave);
      const visible = `${d.title} ${d.desc} ${d.cols.join(' ')} ${d.fields.map(f => f.l).join(' ')}`;
      expect(visible, clave).not.toMatch(/[—–]|\p{Extended_Pictographic}/u);
    }
  });

  it('el ubigeo muestra el código con seis dígitos', () => {
    expect(def('ubigeo').nombre!({ codigo_ubigeo: 10101, distrito: 'Chachapoyas', provincia: 'Chachapoyas' }, ref))
      .toBe('Chachapoyas, Chachapoyas (010101)');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `npx ng test --watch=false --include src/app/core/services/crud.service.spec.ts --include src/app/core/models/cruds.data.spec.ts`
Expected: FAIL. Faltan `refsDe`, `etiqueta`, `opciones`, `guardar`, `estaCargando`, `idKey` y `textos`.

- [ ] **Step 3: Reemplazar `frontend/src/app/core/models/crud.model.ts`**

```ts
/** Busca el nombre legible de un registro de otro módulo: ref('cliente', 5) → «Agro Norte SAC». */
export type RefLookup = (clave: string, id: unknown) => string;

/** Definición de un campo de formulario CRUD */
export interface CrudField {
  id: string;
  l: string;               // label
  tp: 'text' | 'number' | 'textarea' | 'email' | 'date' | 'ref';
  req?: boolean;
  half?: boolean;          // ocupa la mitad del grid
  placeholder?: string;
  rule?: (v: string) => boolean;
  msg?: string;            // mensaje de error de regla
  ref?: string;            // tp 'ref': clave CRUD del módulo relacionado
}

/** Textos con género de cada módulo: «Nuevo viaje», «Dirección guardada». */
export interface CrudTextos {
  nuevo: string;
  guardado: string;
  singular: string;
  plural: string;
}

/** Definición completa de un módulo CRUD */
export interface CrudDef {
  key: string;
  title: string;
  route: string;           // ruta API: /conductores
  nk: string;              // id histórico del módulo
  group: 'Maestros' | 'Operaciones' | 'Logística' | 'Finanzas';
  desc: string;
  fields: CrudField[];
  pay: (form: Record<string, { value: string }>) => Record<string, unknown>;
  cols: string[];
  row: (item: Record<string, unknown>, ref: RefLookup) => string[];
  idKey?: string;          // campo identificador en la API
  nombre?: (item: Record<string, unknown>, ref: RefLookup) => string;
  textos?: CrudTextos;
}

/** Opción de un selector de registros relacionados */
export interface OpcionRef {
  valor: string;
  etiqueta: string;
}

/** Toast */
export type ToastType = 'ok' | 'err' | 'info' | 'warn';
export interface Toast {
  id: number;
  title: string;
  msg: string;
  type: ToastType;
}
```

- [ ] **Step 4: Reemplazar `frontend/src/app/core/models/cruds.data.ts`**

```ts
import { CrudDef, RefLookup } from './crud.model';
import { formatFecha } from '../utils/format';

/* ── Helpers de render de celdas ── */
const esc = (v: unknown) =>
  String(v ?? '').replace(/[&<>"']/g, m => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m] ?? m));
const VACIO = '<span class="celda-vacia">—</span>';
const p = (v: unknown) => (v == null || v === '' ? VACIO : esc(v));
const mkId = (n: unknown) => `<span class="id-tag">#${p(n)}</span>`;
const mkMono = (s: unknown) => `<span class="mono">${p(s)}</span>`;
const mkFuerte = (s: unknown) => `<strong>${p(s)}</strong>`;
const mkSec = (s: unknown) => `<span class="celda-secundaria">${p(s)}</span>`;
const mkDoble = (principal: unknown, secundario: unknown) =>
  `<span class="celda-doble"><strong>${p(principal)}</strong>${secundario ? `<span class="celda-secundaria">${esc(secundario)}</span>` : ''}</span>`;
const mkMoney = (n: unknown) => {
  if (n == null || n === '') return VACIO;
  return `<span class="money"><span class="sy">S/</span>${Number(n).toLocaleString('es-PE', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>`;
};
const mkPlate = (s: unknown) =>
  s ? `<span class="plate"><span class="plate-pe">PE</span><span class="plate-num">${esc(s)}</span></span>` : VACIO;
const FLECHA =
  '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l14 0"/><path d="M13 18l6 -6"/><path d="M13 6l6 6"/></svg>';
const mkRoute = (a: string, b: string) => `<span class="route"><span>${esc(a)}</span>${FLECHA}<span>${esc(b)}</span></span>`;
const mkSt = (v: unknown, tl = 'Activo', fl = 'Inactivo') => {
  const ok = v ?? true;
  return `<span class="lbl ${ok ? 'lbl-ok' : 'lbl-err'}"><span class="lbl__poste"></span>${ok ? tl : fl}</span>`;
};
const mkRef = (ref: RefLookup, clave: string, id: unknown) => (id == null || id === '' ? VACIO : esc(ref(clave, id)));
const mkFecha = (v: unknown) => (v ? esc(formatFecha(String(v))) : VACIO);
const texto = (v: unknown) => String(v ?? '');

/* ════════════════════════════════════
   MÓDULOS CRUD
   Visibles en esta iteración: los 11 primeros (Maestros, Operaciones y Logística).
   Los del final siguen ocultos (no están en los requerimientos actuales).
════════════════════════════════════ */
export const CRUDS: CrudDef[] = [

  /* ── MAESTROS ── */
  {
    key: 'ubigeo', title: 'Ubigeo', route: '/ubigeos', nk: 'ubigeo', group: 'Maestros',
    desc: 'Distritos, provincias y departamentos del Perú',
    idKey: 'codigo_ubigeo',
    nombre: i => `${texto(i['distrito'])}, ${texto(i['provincia'])} (${texto(i['codigo_ubigeo']).padStart(6, '0')})`,
    textos: { nuevo: 'Nuevo ubigeo', guardado: 'Ubigeo guardado', singular: 'ubigeo', plural: 'ubigeos' },
    fields: [
      { id: 'ub_c', l: 'Código (6 dígitos)', tp: 'number', req: true, half: true, rule: v => /^\d{6}$/.test(v), msg: 'Debe tener 6 dígitos' },
      { id: 'ub_dp', l: 'Departamento', tp: 'text', req: true, half: true },
      { id: 'ub_p', l: 'Provincia', tp: 'text', req: true, half: true },
      { id: 'ub_d', l: 'Distrito', tp: 'text', req: true, half: true },
    ],
    pay: f => ({ codigo_ubigeo: Number(f['ub_c'].value), departamento: f['ub_dp'].value, provincia: f['ub_p'].value, distrito: f['ub_d'].value }),
    cols: ['Código', 'Departamento', 'Provincia', 'Distrito'],
    row: i => [mkMono(texto(i['codigo_ubigeo']).padStart(6, '0')), mkFuerte(i['departamento']), p(i['provincia']), p(i['distrito'])],
  },
  {
    key: 'rubro', title: 'Rubro', route: '/rubros', nk: 'rubro', group: 'Maestros',
    desc: 'Clasificación de actividades económicas de los clientes',
    idKey: 'id_rubro',
    nombre: i => texto(i['nombre']),
    textos: { nuevo: 'Nuevo rubro', guardado: 'Rubro guardado', singular: 'rubro', plural: 'rubros' },
    fields: [
      { id: 'ru_n', l: 'Nombre', tp: 'text', req: true },
      { id: 'ru_d', l: 'Descripción', tp: 'text', req: false },
    ],
    pay: f => ({ nombre: f['ru_n'].value, descripcion: f['ru_d'].value || null }),
    cols: ['#', 'Nombre', 'Descripción'],
    row: i => [mkId(i['id_rubro'] ?? i['id']), mkFuerte(i['nombre']), mkSec(i['descripcion'])],
  },
  {
    key: 'marca', title: 'Marca', route: '/marcas', nk: 'marca', group: 'Maestros',
    desc: 'Fabricantes de vehículos de la flota',
    idKey: 'id_marca',
    nombre: i => texto(i['nombre']),
    textos: { nuevo: 'Nueva marca', guardado: 'Marca guardada', singular: 'marca', plural: 'marcas' },
    fields: [
      { id: 'ma_n', l: 'Nombre', tp: 'text', req: true, half: true },
      { id: 'ma_p', l: 'País de origen', tp: 'text', req: false, half: true, placeholder: 'Ej: Alemania' },
    ],
    pay: f => ({ nombre: f['ma_n'].value, pais_origen: f['ma_p'].value || null }),
    cols: ['#', 'Marca', 'País de origen'],
    row: i => [mkId(i['id_marca'] ?? i['id']), mkFuerte(i['nombre']), p(i['pais_origen'])],
  },
  {
    key: 'tipo_unidad', title: 'Tipo de unidad', route: '/tipos-unidad', nk: 'tipo_unidad', group: 'Maestros',
    desc: 'Tracto, camión, semirremolque y otros tipos de vehículo',
    idKey: 'id_tipo_unidad',
    nombre: i => texto(i['nombre']),
    textos: { nuevo: 'Nuevo tipo de unidad', guardado: 'Tipo de unidad guardado', singular: 'tipo de unidad', plural: 'tipos de unidad' },
    fields: [
      { id: 'tu_n', l: 'Nombre', tp: 'text', req: true, placeholder: 'Ej: Tracto' },
      { id: 'tu_d', l: 'Descripción', tp: 'text', req: false },
    ],
    pay: f => ({ nombre: f['tu_n'].value, descripcion: f['tu_d'].value || null }),
    cols: ['#', 'Tipo', 'Descripción'],
    row: i => [mkId(i['id_tipo_unidad'] ?? i['id']), mkFuerte(i['nombre']), mkSec(i['descripcion'])],
  },
  {
    key: 'subtipo_unidad', title: 'Subtipo de unidad', route: '/subtipos-unidad', nk: 'subtipo_unidad', group: 'Maestros',
    desc: 'Subclasificación dentro de cada tipo de vehículo',
    idKey: 'id_subtipo_unidad',
    nombre: (i, ref) => `${texto(i['nombre'])} · ${ref('tipo_unidad', i['id_tipo_unidad'])}`,
    textos: { nuevo: 'Nuevo subtipo', guardado: 'Subtipo guardado', singular: 'subtipo de unidad', plural: 'subtipos de unidad' },
    fields: [
      { id: 'su_t', l: 'Tipo de unidad', tp: 'ref', ref: 'tipo_unidad', req: true, half: true },
      { id: 'su_n', l: 'Nombre', tp: 'text', req: true, half: true, placeholder: 'Ej: Plataforma' },
      { id: 'su_d', l: 'Descripción', tp: 'text', req: false },
    ],
    pay: f => ({ id_tipo_unidad: Number(f['su_t'].value), nombre: f['su_n'].value, descripcion: f['su_d'].value || null }),
    cols: ['#', 'Subtipo', 'Tipo de unidad', 'Descripción'],
    row: (i, ref) => [mkId(i['id_subtipo_unidad'] ?? i['id']), mkFuerte(i['nombre']), mkRef(ref, 'tipo_unidad', i['id_tipo_unidad']), mkSec(i['descripcion'])],
  },

  /* ── OPERACIONES ── */
  {
    key: 'direccion', title: 'Dirección', route: '/direcciones', nk: 'direccion', group: 'Operaciones',
    desc: 'Puntos de origen, destino y domicilios',
    idKey: 'id_direccion',
    nombre: i => texto(i['direccion_detallada']),
    textos: { nuevo: 'Nueva dirección', guardado: 'Dirección guardada', singular: 'dirección', plural: 'direcciones' },
    fields: [
      { id: 'di_u', l: 'Ubigeo', tp: 'ref', ref: 'ubigeo', req: true },
      { id: 'di_d', l: 'Dirección detallada', tp: 'text', req: true, placeholder: 'Av. o Jr. y número' },
      { id: 'di_r', l: 'Referencia', tp: 'text', req: false, placeholder: 'Ej: frente al parque' },
    ],
    pay: f => ({ codigo_ubigeo: Number(f['di_u'].value), direccion_detallada: f['di_d'].value, referencia: f['di_r'].value || null }),
    cols: ['#', 'Dirección', 'Ubigeo', 'Referencia'],
    row: (i, ref) => [mkId(i['id_direccion'] ?? i['id']), mkFuerte(i['direccion_detallada']), mkRef(ref, 'ubigeo', i['codigo_ubigeo']), mkSec(i['referencia'])],
  },
  {
    key: 'conductor', title: 'Conductor', route: '/conductores', nk: 'conductor', group: 'Operaciones',
    desc: 'Personal conductor habilitado con licencia vigente',
    idKey: 'id_conductor',
    nombre: i => `${texto(i['nombre1'])} ${texto(i['apellido1'])}`.trim(),
    textos: { nuevo: 'Nuevo conductor', guardado: 'Conductor guardado', singular: 'conductor', plural: 'conductores' },
    fields: [
      { id: 'co_dni', l: 'DNI', tp: 'text', req: true, half: true, rule: v => /^\d{8}$/.test(v), msg: '8 dígitos' },
      { id: 'co_lic', l: 'N° Licencia', tp: 'text', req: true, half: true },
      { id: 'co_n1', l: 'Primer nombre', tp: 'text', req: true, half: true },
      { id: 'co_a1', l: 'Apellido paterno', tp: 'text', req: true, half: true },
      { id: 'co_n2', l: 'Segundo nombre', tp: 'text', req: false, half: true },
      { id: 'co_a2', l: 'Apellido materno', tp: 'text', req: false, half: true },
      { id: 'co_tel', l: 'Teléfono', tp: 'text', req: false, half: true, placeholder: '+51 9' },
      { id: 'co_cor', l: 'Correo', tp: 'email', req: false, half: true },
    ],
    pay: f => ({
      dni: f['co_dni'].value, nombre1: f['co_n1'].value, nombre2: f['co_n2'].value || null, apellido1: f['co_a1'].value,
      apellido2: f['co_a2'].value || null, licencia_conducir: f['co_lic'].value, telefono: f['co_tel'].value || null, correo: f['co_cor'].value || null,
    }),
    cols: ['Conductor', 'DNI', 'Licencia', 'Teléfono', 'Estado'],
    row: i => [
      mkDoble(`${texto(i['nombre1'])} ${texto(i['apellido1'])}`.trim(), [i['nombre2'], i['apellido2']].filter(Boolean).join(' ')),
      mkMono(i['dni']), mkMono(i['licencia_conducir']), p(i['telefono']), mkSt(i['estado_conductor'] ?? true),
    ],
  },
  {
    key: 'cliente', title: 'Cliente', route: '/clientes', nk: 'cliente', group: 'Operaciones',
    desc: 'Empresas y personas que contratan el servicio',
    idKey: 'id_cliente',
    nombre: i => texto(i['razon_social']),
    textos: { nuevo: 'Nuevo cliente', guardado: 'Cliente guardado', singular: 'cliente', plural: 'clientes' },
    fields: [
      { id: 'cl_r', l: 'Razón social', tp: 'text', req: true },
      { id: 'cl_ruc', l: 'RUC', tp: 'text', req: true, half: true, rule: v => /^\d{11}$/.test(v), msg: '11 dígitos' },
      { id: 'cl_rb', l: 'Rubro', tp: 'ref', ref: 'rubro', req: true, half: true },
      { id: 'cl_t', l: 'Teléfono', tp: 'text', req: false, half: true, placeholder: '+51' },
      { id: 'cl_e', l: 'Correo', tp: 'email', req: false, half: true },
      { id: 'cl_c', l: 'Persona de contacto', tp: 'text', req: false },
    ],
    pay: f => ({
      razon_social: f['cl_r'].value, ruc: f['cl_ruc'].value, telefono: f['cl_t'].value || null, correo: f['cl_e'].value || null,
      contacto: f['cl_c'].value || null, id_rubro: Number(f['cl_rb'].value),
    }),
    cols: ['Empresa', 'RUC', 'Rubro', 'Teléfono', 'Estado'],
    row: (i, ref) => [mkDoble(i['razon_social'], i['correo']), mkMono(i['ruc']), mkRef(ref, 'rubro', i['id_rubro']), p(i['telefono']), mkSt(i['estado_cliente'] ?? true)],
  },
  {
    key: 'unidad', title: 'Flota', route: '/unidades', nk: 'unidad', group: 'Operaciones',
    desc: 'Tractos, camiones y remolques registrados',
    idKey: 'id_unidad',
    nombre: i => [i['placa'], i['modelo']].filter(Boolean).map(texto).join(' · '),
    textos: { nuevo: 'Nueva unidad', guardado: 'Unidad guardada', singular: 'unidad', plural: 'unidades' },
    fields: [
      { id: 'un_p', l: 'Placa', tp: 'text', req: true, half: true, placeholder: 'ABC-123' },
      { id: 'un_a', l: 'Año', tp: 'number', req: false, half: true, rule: v => !v || Number(v) >= 1900, msg: 'Inválido' },
      { id: 'un_ti', l: 'Tipo de unidad', tp: 'ref', ref: 'tipo_unidad', req: true, half: true },
      { id: 'un_s', l: 'Subtipo', tp: 'ref', ref: 'subtipo_unidad', req: false, half: true },
      { id: 'un_m', l: 'Marca', tp: 'ref', ref: 'marca', req: true, half: true },
      { id: 'un_mo', l: 'Modelo', tp: 'text', req: false, half: true, placeholder: 'Ej: Actros 2646' },
      { id: 'un_pt', l: 'Peso tara (ton)', tp: 'number', req: false, half: true },
      { id: 'un_pm', l: 'Peso máx. (ton)', tp: 'number', req: false, half: true },
    ],
    pay: f => ({
      placa: f['un_p'].value.toUpperCase(), id_tipo_unidad: Number(f['un_ti'].value),
      id_subtipo_unidad: f['un_s'].value ? Number(f['un_s'].value) : null, id_marca: Number(f['un_m'].value),
      modelo: f['un_mo'].value || null, anio: f['un_a'].value ? Number(f['un_a'].value) : null,
      peso_tara: f['un_pt'].value ? Number(f['un_pt'].value) : null, peso_maximo: f['un_pm'].value ? Number(f['un_pm'].value) : null,
    }),
    cols: ['Placa', 'Modelo', 'Marca', 'Tipo', 'Año', 'Estado'],
    row: (i, ref) => [
      mkPlate(i['placa']), p(i['modelo']), mkRef(ref, 'marca', i['id_marca']), mkRef(ref, 'tipo_unidad', i['id_tipo_unidad']),
      p(i['anio']), mkSt(i['estado_unidad'] ?? true, 'Operativa', 'Inactiva'),
    ],
  },
  {
    key: 'cond_uni', title: 'Asignación', route: '/asignaciones-conductor-unidad', nk: 'cond_uni', group: 'Operaciones',
    desc: 'Qué conductor maneja cada unidad',
    idKey: 'id_conductor_unidad',
    nombre: (i, ref) => `${ref('conductor', i['id_conductor'])} · ${ref('unidad', i['id_unidad'])}`,
    textos: { nuevo: 'Nueva asignación', guardado: 'Asignación guardada', singular: 'asignación', plural: 'asignaciones' },
    fields: [
      { id: 'cu_c', l: 'Conductor', tp: 'ref', ref: 'conductor', req: true, half: true },
      { id: 'cu_u', l: 'Unidad', tp: 'ref', ref: 'unidad', req: true, half: true },
      { id: 'cu_f', l: 'Fecha de asignación', tp: 'date', req: true },
    ],
    pay: f => ({ id_conductor: Number(f['cu_c'].value), id_unidad: Number(f['cu_u'].value), fecha_asignacion: f['cu_f'].value }),
    cols: ['#', 'Conductor', 'Unidad', 'Desde', 'Estado'],
    row: (i, ref) => [
      mkId(i['id_conductor_unidad'] ?? i['id']), `<strong>${mkRef(ref, 'conductor', i['id_conductor'])}</strong>`,
      mkRef(ref, 'unidad', i['id_unidad']), mkFecha(i['fecha_asignacion']), mkSt(i['estado_asignacion'] ?? true, 'Activa', 'Finalizada'),
    ],
  },

  /* ── LOGÍSTICA ── */
  {
    key: 'viaje', title: 'Viaje', route: '/viajes', nk: 'viaje', group: 'Logística',
    desc: 'Servicios de transporte de carga programados',
    idKey: 'id_viaje',
    nombre: i => `Viaje N° ${texto(i['id_viaje'])} · ${formatFecha(i['fecha_programado'] as string | null)}`,
    textos: { nuevo: 'Nuevo viaje', guardado: 'Viaje guardado', singular: 'viaje', plural: 'viajes' },
    fields: [
      { id: 'vi_tr', l: 'Tracto', tp: 'ref', ref: 'unidad', req: true, half: true },
      { id: 'vi_ca', l: 'Carreta', tp: 'ref', ref: 'unidad', req: false, half: true },
      { id: 'vi_cu', l: 'Asignación conductor-unidad', tp: 'ref', ref: 'cond_uni', req: true },
      { id: 'vi_cl', l: 'Cliente', tp: 'ref', ref: 'cliente', req: true },
      { id: 'vi_o', l: 'Dirección de origen', tp: 'ref', ref: 'direccion', req: true, half: true },
      { id: 'vi_d', l: 'Dirección de destino', tp: 'ref', ref: 'direccion', req: true, half: true },
      { id: 'vi_f', l: 'Fecha programada', tp: 'date', req: true },
    ],
    pay: f => ({
      id_tracto: Number(f['vi_tr'].value), id_carreta: f['vi_ca'].value ? Number(f['vi_ca'].value) : null,
      id_conductor_unidad: Number(f['vi_cu'].value), id_cliente: Number(f['vi_cl'].value),
      id_direccion_origen: Number(f['vi_o'].value), id_direccion_destino: Number(f['vi_d'].value), fecha_programado: f['vi_f'].value,
    }),
    cols: ['N°', 'Ruta', 'Cliente', 'Tracto', 'Fecha', 'Estado'],
    row: (i, ref) => [
      mkId(i['id_viaje'] ?? i['id']),
      mkRoute(ref('direccion', i['id_direccion_origen']), ref('direccion', i['id_direccion_destino'])),
      mkRef(ref, 'cliente', i['id_cliente']), mkRef(ref, 'unidad', i['id_tracto']), mkFecha(i['fecha_programado']),
      mkSt(i['estado_viaje'] ?? true, 'Programado', 'Cancelado'),
    ],
  },

  /* ── OCULTOS EN ESTA ITERACIÓN ── */
  {
    key: 'tipo_gasto', title: 'Tipo de gasto', route: '/tipos-gasto', nk: 'tipo_gasto', group: 'Maestros',
    desc: 'Combustible, peajes, mantenimiento',
    fields: [{ id: 'tg_n', l: 'Nombre', tp: 'text', req: true, placeholder: 'Ej: Combustible' }, { id: 'tg_d', l: 'Descripción', tp: 'text', req: false }],
    pay: f => ({ nombre: f['tg_n'].value, descripcion: f['tg_d'].value || null }),
    cols: ['#', 'Tipo de gasto', 'Descripción'],
    row: i => [mkId(i['id_tipo_gasto'] ?? i['id']), mkFuerte(i['nombre']), mkSec(i['descripcion'])],
  },
  {
    key: 'cat_prod', title: 'Categoría de producto', route: '/categorias-producto', nk: 'cat_prod', group: 'Maestros',
    desc: 'Clasificación de mercancías transportadas',
    fields: [{ id: 'cp_n', l: 'Nombre', tp: 'text', req: true, placeholder: 'Ej: Alimentos' }, { id: 'cp_d', l: 'Descripción', tp: 'text', req: false }],
    pay: f => ({ nombre: f['cp_n'].value, descripcion: f['cp_d'].value || null }),
    cols: ['#', 'Categoría', 'Descripción'],
    row: i => [mkId(i['id_categoria'] ?? i['id']), `<span class="lbl lbl-info">${p(i['nombre'])}</span>`, mkSec(i['descripcion'])],
  },
  {
    key: 'guia', title: 'Guía de transportista', route: '/guias-transportista', nk: 'guia', group: 'Logística',
    desc: 'Guías de remisión del transportista (GRT)',
    fields: [
      { id: 'gu_n', l: 'N° de guía GRT', tp: 'text', req: true, placeholder: 'GRT-001-0001' },
      { id: 'gu_v', l: 'ID Viaje', tp: 'number', req: true, half: true },
      { id: 'gu_p', l: 'Peso total (kg)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
      { id: 'gu_r', l: 'Guías remitente', tp: 'text', req: false },
    ],
    pay: f => ({ numero_guia: f['gu_n'].value, id_viaje: Number(f['gu_v'].value), peso_total: Number(f['gu_p'].value), guias_remitente: f['gu_r'].value || null }),
    cols: ['N° GRT', 'Viaje', 'Peso', 'Estado'],
    row: i => [
      mkMono(i['numero_guia']), `<span class="lbl lbl-n">Viaje #${p(i['id_viaje'])}</span>`,
      `<strong>${p(i['peso_total'])} <small class="celda-secundaria">kg</small></strong>`, mkSt(i['estado_guia'] ?? true, 'Vigente', 'Anulada'),
    ],
  },
  {
    key: 'viaje_r', title: 'Viaje realizado', route: '/viajes-realizados', nk: 'viaje_r', group: 'Logística',
    desc: 'Servicios de transporte completados con flete',
    fields: [
      { id: 'vr_g', l: 'ID Guía GRT', tp: 'number', req: true, half: true },
      { id: 'vr_f', l: 'Flete (S/)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
      { id: 'vr_o', l: 'Observaciones', tp: 'textarea', req: false, placeholder: 'Incidencias del viaje' },
    ],
    pay: f => ({ id_guia_transportista: Number(f['vr_g'].value), flete: Number(f['vr_f'].value), observaciones: f['vr_o'].value || null }),
    cols: ['#', 'GRT', 'Flete', 'Estado'],
    row: i => [
      mkId(i['id_viaje_realizado'] ?? i['id']), `<span class="mono">GRT #${p(i['id_guia_transportista'])}</span>`,
      mkMoney(i['flete']), mkSt(i['estado_viaje_realizado'] ?? true, 'Completado', 'Pendiente'),
    ],
  },
  {
    key: 'prod_t', title: 'Productos transportados', route: '/productos-transportados', nk: 'prod_t', group: 'Logística',
    desc: 'Detalle de mercancías por guía de remisión',
    fields: [
      { id: 'pt_g', l: 'ID Guía GRT', tp: 'number', req: true, half: true },
      { id: 'pt_c', l: 'ID Categoría', tp: 'number', req: true, half: true },
      { id: 'pt_d', l: 'Descripción', tp: 'text', req: true, placeholder: 'Ej: Harina de trigo' },
      { id: 'pt_ca', l: 'Cantidad', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
      { id: 'pt_p', l: 'Peso (kg)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({
      id_guia_transportista: Number(f['pt_g'].value), id_categoria: Number(f['pt_c'].value), descripcion_producto: f['pt_d'].value,
      cantidad: Number(f['pt_ca'].value), peso: Number(f['pt_p'].value),
    }),
    cols: ['#', 'Producto', 'Cant.', 'Peso', 'GRT'],
    row: i => [
      mkId(i['id_producto_transportado'] ?? i['id']), mkFuerte(i['descripcion_producto']), mkFuerte(i['cantidad']),
      `<strong>${p(i['peso'])} <small class="celda-secundaria">kg</small></strong>`, `<span class="mono">GRT #${p(i['id_guia_transportista'])}</span>`,
    ],
  },
  {
    key: 'factura', title: 'Factura', route: '/facturas', nk: 'factura', group: 'Finanzas',
    desc: 'Documentos de cobro por servicios de transporte',
    fields: [
      { id: 'fa_n', l: 'N° de factura', tp: 'text', req: true, placeholder: 'F001-00001' },
      { id: 'fa_v', l: 'ID Viaje realizado', tp: 'number', req: true, half: true },
      { id: 'fa_m', l: 'Monto total (S/)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({ numero_factura: f['fa_n'].value, id_viaje_realizado: Number(f['fa_v'].value), monto_total: Number(f['fa_m'].value) }),
    cols: ['N° Factura', 'Viaje realizado', 'Monto', 'Estado'],
    row: i => [
      mkMono(i['numero_factura']), `<span class="lbl lbl-n">VR #${p(i['id_viaje_realizado'])}</span>`,
      mkMoney(i['monto_total']), mkSt(i['estado_factura'] ?? true, 'Activa', 'Anulada'),
    ],
  },
  {
    key: 'presupuesto', title: 'Presupuesto de viaje', route: '/presupuestos-viaje', nk: 'presupuesto', group: 'Finanzas',
    desc: 'Estimación de costos operativos por viaje',
    fields: [
      { id: 'pe_v', l: 'ID Viaje', tp: 'number', req: true, half: true },
      { id: 'pe_m', l: 'Monto total (S/)', tp: 'number', req: true, half: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({ id_viaje: Number(f['pe_v'].value), monto_total: Number(f['pe_m'].value) }),
    cols: ['#', 'Viaje', 'Monto', 'Estado'],
    row: i => [
      mkId(i['id_presupuesto'] ?? i['id']), `<span class="lbl lbl-n">Viaje #${p(i['id_viaje'])}</span>`,
      mkMoney(i['monto_total']), mkSt(i['estado_presupuesto'] ?? true, 'Aprobado', 'Pendiente'),
    ],
  },
  {
    key: 'det_pres', title: 'Detalle de presupuesto', route: '/detalles-presupuesto', nk: 'det_pres', group: 'Finanzas',
    desc: 'Desglose de gastos por tipo de cada presupuesto',
    fields: [
      { id: 'dp_p', l: 'ID Presupuesto', tp: 'number', req: true, half: true },
      { id: 'dp_t', l: 'ID Tipo de gasto', tp: 'number', req: true, half: true },
      { id: 'dp_m', l: 'Monto estimado (S/)', tp: 'number', req: true, rule: v => Number(v) > 0, msg: '>0' },
    ],
    pay: f => ({ id_presupuesto_viaje: Number(f['dp_p'].value), id_tipo_gasto: Number(f['dp_t'].value), monto_estimado: Number(f['dp_m'].value) }),
    cols: ['#', 'Presupuesto', 'Tipo de gasto', 'Monto est.'],
    row: i => [
      mkId(i['id_detalle_presupuesto'] ?? i['id']), `<span class="lbl lbl-n">Pres. #${p(i['id_presupuesto'] ?? i['id_presupuesto_viaje'])}</span>`,
      `<span class="lbl lbl-warn">TG #${p(i['id_tipo_gasto'])}</span>`, mkMoney(i['monto_estimado']),
    ],
  },
];
```

- [ ] **Step 5: Reemplazar `frontend/src/app/core/services/crud.service.ts`**

```ts
import { Injectable, inject, signal } from '@angular/core';
import { Observable, catchError, map, of, tap } from 'rxjs';
import { ApiService } from './api.service';
import { ToastService } from './toast.service';
import { CrudDef, OpcionRef, RefLookup } from '../models/crud.model';
import { CRUDS } from '../models/cruds.data';

type RecordMap = Record<string, unknown>;

@Injectable({ providedIn: 'root' })
export class CrudService {
  private readonly api = inject(ApiService);
  private readonly toast = inject(ToastService);

  /** Cache local de todos los módulos */
  readonly data = signal<Record<string, RecordMap[]>>({});
  readonly cargando = signal<Record<string, boolean>>({});
  readonly cargado = signal<Record<string, boolean>>({});

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

  /** Carga una lista desde la API */
  loadList(key: string): void {
    const c = this.getCrud(key);
    if (!c) return;
    this.cargando.update(m => ({ ...m, [key]: true }));
    this.api.get<RecordMap[] | { items?: RecordMap[]; data?: RecordMap[] }>(c.route).pipe(
      map(res => (Array.isArray(res) ? res : res?.items ?? res?.data ?? [])),
      catchError(err => {
        this.toast.show('Sin conexión', err?.message ?? '', 'err');
        return of(null);
      }),
    ).subscribe(items => {
      if (items) {
        this.data.update(d => ({ ...d, [key]: items }));
        this.cargado.update(m => ({ ...m, [key]: true }));
      }
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

  /** Guarda un registro; emite lo creado, o null si la API lo rechazó (ya avisado). */
  guardar(key: string, payload: RecordMap): Observable<RecordMap | null> {
    const c = this.getCrud(key);
    if (!c) return of(null);
    return this.api.post<RecordMap>(c.route, payload).pipe(
      tap(() => {
        this.toast.show(c.textos?.guardado ?? `${c.title} guardado`, 'Registro creado correctamente', 'ok');
        this.loadList(key);
      }),
      catchError(err => {
        const msg = err?.error?.detail ?? err?.message ?? 'Error';
        const texto = Array.isArray(msg) ? msg.map((e: { msg?: string }) => e.msg).join(' · ') : String(msg);
        this.toast.show('No se pudo guardar', texto, 'err');
        return of(null);
      }),
    );
  }

  /** @deprecated Solo para la vista CRUD vieja; se elimina en la Tarea 20. */
  saveItem(key: string, payload: RecordMap): void {
    this.guardar(key, payload).subscribe();
  }

  /** @deprecated Solo para el dashboard viejo; se elimina en la Tarea 20. */
  refreshAll(): void {
    this.cruds.forEach(c => this.loadList(c.key));
    this.toast.show('Datos actualizados', '', 'ok', 2200);
  }
}
```

- [ ] **Step 6: Mantener compilando la vista CRUD vieja hasta la Tarea 20**

En `frontend/src/app/features/crud-view/crud-view.component.ts`, dentro de `renderRow`, cambia:

```ts
    return c.row(item).map(html => this.sanitizer.bypassSecurityTrustHtml(html));
```

por:

```ts
    return c.row(item, this.crud.ref).map(html => this.sanitizer.bypassSecurityTrustHtml(html));
```

- [ ] **Step 7: Ejecutar y verificar que pasan**

Run: `npx ng test --watch=false --include src/app/core/services/crud.service.spec.ts --include src/app/core/models/cruds.data.spec.ts`
Expected: PASS (13 pruebas).

- [ ] **Step 8: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso. El dashboard viejo sigue compilando gracias a `saveItem` y `refreshAll`.

---

### Task 6: `GastosStore` (estado compartido de gastos y comprobantes)

**Files:**
- Create: `frontend/src/app/core/state/gastos.store.ts`
- Test: `frontend/src/app/core/state/gastos.store.spec.ts`

**Interfaces:**
- Consumes:
  - de `GastosApiService`: `get<T>(path)`, `post<T>(path, body)` y `uploadFile<T>(path, file)`
  - `ToastService.show(title, msg, type)`
  - `HTTP_TIMEOUT_MS` y `mensajeError`
  - `contarPorEstado` y `comprobantesDe` (Tarea 3)
- Produces: `GastosStore`
  - signals: `gastos`, `comprobantes`, `cargando`, `cargado` y `error`
  - derivados: `conteo` y `pendientes`
  - consultas: `gasto(id): Gasto | undefined` y `comprobantesDe(gastoId): Comprobante[]`
  - carga: `cargar(): void` y `asegurarCargado(): void`
  - mutaciones:
    - `registrar(datos: NuevoGasto): Observable<Gasto>`
    - `adjuntar(gastoId: number, archivo: File): Observable<Comprobante>`
    - `aprobar(comprobanteId: number): Observable<unknown>`
    - `rechazar(comprobanteId: number, motivo: string): Observable<unknown>`
  - `interface NuevoGasto { viaje_id: number; cliente_id: number; monto: number; fecha: string }`

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/core/state/gastos.store.spec.ts`:

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { GastosStore } from './gastos.store';
import { GastosApiService } from '../services/gastos-api.service';
import { ToastService } from '../services/toast.service';
import { MSG_ERROR_RED } from '../utils/error-messages';
import { Comprobante, Gasto } from '../models/gasto.model';

const gastos: Gasto[] = [
  { id: 1, viaje_id: 1, cliente_id: 1, monto: 120, fecha: '2026-10-01T00:00:00Z', estado: 'Registrado', creado_por: 'Ana', creado_en: '2026-10-01T15:00:00Z' },
  { id: 2, viaje_id: 1, cliente_id: 1, monto: 80, fecha: '2026-10-01T00:00:00Z', estado: 'Pagado', creado_por: 'Ana', creado_en: '2026-10-01T16:00:00Z' },
];
const comprobantes: Comprobante[] = [
  { id: 9, gasto_id: 2, archivo_url: '/uploads/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z' },
];

function apiFalsa() {
  return {
    get: vi.fn((ruta: string) => of(ruta === '/gastos' ? gastos : comprobantes)),
    post: vi.fn(() => of({ id: 3 })),
    uploadFile: vi.fn(() => of({ id: 10 })),
  };
}

describe('GastosStore', () => {
  let api: ReturnType<typeof apiFalsa>;
  let store: GastosStore;

  beforeEach(() => {
    api = apiFalsa();
    TestBed.configureTestingModule({ providers: [{ provide: GastosApiService, useValue: api }] });
    store = TestBed.inject(GastosStore);
  });

  it('carga gastos y comprobantes y deriva conteos y pendientes', () => {
    store.cargar();
    expect(store.cargado()).toBe(true);
    expect(store.gastos()).toHaveLength(2);
    expect(store.conteo()).toEqual({ Registrado: 1, Pagado: 1, Sustentado: 0 });
    expect(store.pendientes().map(c => c.id)).toEqual([9]);
    expect(store.gasto(2)?.monto).toBe(80);
    expect(store.comprobantesDe(2).map(c => c.id)).toEqual([9]);
  });

  it('asegurarCargado no repite la carga', () => {
    store.asegurarCargado();
    store.asegurarCargado();
    expect(api.get).toHaveBeenCalledTimes(2);
  });

  it('las mutaciones llaman a la API correcta y recargan', () => {
    const archivo = new File(['x'], 'boleta.jpg', { type: 'image/jpeg' });
    store.registrar({ viaje_id: 1, cliente_id: 1, monto: 50, fecha: '2026-10-02' }).subscribe();
    store.adjuntar(1, archivo).subscribe();
    store.aprobar(9).subscribe();
    store.rechazar(9, 'Ilegible').subscribe();
    expect(api.post).toHaveBeenCalledWith('/gastos', { viaje_id: 1, cliente_id: 1, monto: 50, fecha: '2026-10-02' });
    expect(api.uploadFile).toHaveBeenCalledWith('/gastos/1/comprobantes', archivo);
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/aprobar', {});
    expect(api.post).toHaveBeenCalledWith('/comprobantes/9/rechazar', { motivo_rechazo: 'Ilegible' });
    expect(api.get).toHaveBeenCalledTimes(8);
  });

  it('si la carga falla marca el error y avisa con el copy exacto', () => {
    api.get.mockReturnValue(throwError(() => new Error('sin red')));
    store.cargar();
    expect(store.error()).toBe(true);
    expect(store.cargado()).toBe(false);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe(MSG_ERROR_RED);
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/core/state/gastos.store.spec.ts`
Expected: FAIL. No se puede resolver `./gastos.store`.

- [ ] **Step 3: Implementar**

`frontend/src/app/core/state/gastos.store.ts`:

```ts
import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, forkJoin, tap, timeout } from 'rxjs';
import { GastosApiService } from '../services/gastos-api.service';
import { ToastService } from '../services/toast.service';
import { Comprobante, Gasto } from '../models/gasto.model';
import { HTTP_TIMEOUT_MS, mensajeError } from '../utils/error-messages';
import { comprobantesDe, contarPorEstado } from './gastos.logic';

export interface NuevoGasto {
  viaje_id: number;
  cliente_id: number;
  monto: number;
  fecha: string;
}

/** Gastos y comprobantes compartidos por Inicio, la bandeja, el detalle, la barra y la paleta. */
@Injectable({ providedIn: 'root' })
export class GastosStore {
  private readonly api = inject(GastosApiService);
  private readonly toast = inject(ToastService);

  readonly gastos = signal<Gasto[]>([]);
  readonly comprobantes = signal<Comprobante[]>([]);
  readonly cargando = signal(false);
  readonly cargado = signal(false);
  readonly error = signal(false);

  readonly conteo = computed(() => contarPorEstado(this.gastos()));
  readonly pendientes = computed(() => this.comprobantes().filter(c => c.estado === 'Pendiente'));

  gasto(id: number): Gasto | undefined {
    return this.gastos().find(g => g.id === id);
  }

  comprobantesDe(gastoId: number): Comprobante[] {
    return comprobantesDe(this.comprobantes(), gastoId);
  }

  cargar(): void {
    this.cargando.set(true);
    forkJoin({
      gastos: this.api.get<Gasto[]>('/gastos'),
      comprobantes: this.api.get<Comprobante[]>('/comprobantes'),
    }).pipe(timeout(HTTP_TIMEOUT_MS)).subscribe({
      next: ({ gastos, comprobantes }) => {
        this.gastos.set(gastos ?? []);
        this.comprobantes.set(comprobantes ?? []);
        this.error.set(false);
        this.cargado.set(true);
        this.cargando.set(false);
      },
      error: err => {
        this.error.set(true);
        this.cargando.set(false);
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }

  asegurarCargado(): void {
    if (!this.cargado() && !this.cargando()) this.cargar();
  }

  registrar(datos: NuevoGasto): Observable<Gasto> {
    return this.api.post<Gasto>('/gastos', datos).pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }

  adjuntar(gastoId: number, archivo: File): Observable<Comprobante> {
    return this.api
      .uploadFile<Comprobante>(`/gastos/${gastoId}/comprobantes`, archivo)
      .pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }

  aprobar(comprobanteId: number): Observable<unknown> {
    return this.api.post(`/comprobantes/${comprobanteId}/aprobar`, {}).pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }

  rechazar(comprobanteId: number, motivo: string): Observable<unknown> {
    return this.api
      .post(`/comprobantes/${comprobanteId}/rechazar`, { motivo_rechazo: motivo })
      .pipe(timeout(HTTP_TIMEOUT_MS), tap(() => this.cargar()));
  }
}
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/core/state/gastos.store.spec.ts`
Expected: PASS (4 pruebas).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false`
Expected: PASS en todas las pruebas.

---
### Task 7: Sistema visual global (tokens, base, componentes), fuentes e `index.html`

**Files:**
- Modify: `frontend/angular.json` (array `styles` del build)
- Modify (reemplazo completo): `frontend/src/index.html`
- Modify (reemplazo completo): `frontend/src/styles.css`
- Create: `frontend/src/styles/tokens.css`
- Create: `frontend/src/styles/base.css`
- Create: `frontend/src/styles/componentes.css`
- Create (fuera del repo): `$SCRATCH/verificar-contraste.mjs`

**Interfaces:**
- Consumes: nada.
- Produces: el vocabulario CSS que usan las tareas siguientes.
  - Tokens (`--asfalto`, `--suelo`, `--rojo`, `--via`, estados, `--e-*`, `--t-*`, `--r-*`, `--ease-*` y `--d-*`).
  - Clases de componentes:
    - página: `.pagina` (`--media`, `--angosta`), `.pagina__encabezado`, `.pagina__titulo`, `.pagina__bajada`, `.pagina__acciones`
    - títulos y navegación de texto: `.migas`, `.seccion-titulo`, `.seccion-cabecera`, `.volver`
    - botones: `.btn` (`--primario`, `--secundario`, `--fantasma`, `--peligro`, `--sobre-asfalto`, `--sm`, `--lg`, `--icono`, `--bloque`) y `.enlace`, `.enlace-boton`
    - formularios: `.formulario` (`--rejilla`), `.formulario__fila`, `.formulario__acciones`, `.campo` (`--medio`, `--con-error`), `.campo__etiqueta`, `.campo__ayuda`, `.campo__error`, `.req`
    - controles: `.control`, `.control-prefijo`, `.control-con-icono`
    - estado: `.lbl` (`lbl-n`, `lbl-info`, `lbl-ok`, `lbl-warn`, `lbl-err`, `lbl-y`), `.lbl__poste`, `.contador` (`--via`), `.alerta` (`--error`, `--exito`, `--via`)
    - contenedores y listas: `.panel`, `.pestanas`, `.pestanas__item` (`--activa`), `.barra-herramientas`, `.buscador`, `.tabla-marco`, `.tabla`
    - estados de carga y vacíos: `.vacio`, `.esqueleto` (`--fila`, `--bloque`, `--franja`)
    - diálogos y avisos: `.hoja` (`--inferior`), `.dialogo`, `.avisos`, `.aviso`
    - selector: `.combo`
    - piezas sueltas: `.girador`, `.icono`, `.marca`, `.camion`
    - utilidades: `.cifra`, `.mono`, `.oculto`, `.oculto-visual` y `.saltar`

> Nota de migración: desde esta tarea las pantallas viejas (dashboard, login, registro y landing) pierden su CSS anterior y se verán sin estilo hasta que su tarea las reemplace (Tareas 12 a 19). Es esperado.

- [ ] **Step 1: Cargar las fuentes desde Fontsource**

En `frontend/angular.json`, en `projects.lizar-ng.architect.build.options.styles`, reemplaza `["src/styles.css"]` por:

```json
            "styles": [
              "node_modules/@fontsource-variable/overpass/index.css",
              "node_modules/@fontsource/overpass-mono/400.css",
              "node_modules/@fontsource/overpass-mono/600.css",
              "src/styles.css"
            ]
```

- [ ] **Step 2: Reemplazar `src/index.html`**

```html
<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <title>LIZAR SRL · Fleet Manager Pro</title>
  <base href="/">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="description" content="Registro de gastos de viaje, comprobantes de pago y aprobación para LIZAR SRL.">
  <meta name="theme-color" content="#1E2124">
  <link rel="icon" type="image/x-icon" href="favicon.ico">
</head>
<body>
  <app-root></app-root>
</body>
</html>
```

- [ ] **Step 3: Reemplazar `src/styles.css`**

```css
/* LIZAR SRL · Fleet Manager Pro. Sistema visual «Hitos de carretera».
   Cada tarea agrega aquí la hoja de su área, siempre después de las existentes. */
@import './styles/tokens.css';
@import './styles/base.css';
@import './styles/componentes.css';
```

- [ ] **Step 4: Crear `src/styles/tokens.css`**

```css
/* ══ Tokens: única fuente de verdad del sistema visual ══ */
:root {
  color-scheme: light;

  /* Superficies */
  --asfalto: #1e2124;
  --asfalto-2: #2a2e33;
  --asfalto-3: #3a3f45;
  --asfalto-tinta: #c9cdd1;
  --suelo: #f7f7f5;
  --panel: #fdfdfc;
  --panel-2: #f0f1ee;
  --linea: #e2e3e0;
  --linea-2: #c9ccc8;

  /* Tinta */
  --tinta: #1e2124;
  --tinta-2: #4a5056;
  --tinta-3: #656b71;

  /* Marca y atención */
  --rojo: #e11d48;
  --rojo-hover: #c8113d;
  --rojo-presionado: #ad0e35;
  --rojo-suave: #fdecef;
  --rojo-tinta: #b4123a;
  --rojo-borde: #f6c7d1;
  --via: #f2c200;
  --via-suave: #fff5cc;
  --via-tinta: #6b5300;

  /* Estados (fijados por el producto; coinciden con la señalización vial) */
  --registrado: #8a9096;
  --registrado-suave: #eeefec;
  --registrado-tinta: #4a5056;
  --pagado: #2563eb;
  --pagado-suave: #e8effe;
  --pagado-tinta: #1e4fc2;
  --sustentado: #15803d;
  --sustentado-suave: #e6f4ea;
  --sustentado-tinta: #166534;
  --pendiente: var(--via);
  --pendiente-suave: var(--via-suave);
  --pendiente-tinta: var(--via-tinta);
  --rechazado: var(--rojo);
  --rechazado-suave: var(--rojo-suave);
  --rechazado-tinta: var(--rojo-tinta);

  /* Tipografía */
  --fuente: 'Overpass Variable', 'Overpass', system-ui, -apple-system, 'Segoe UI', sans-serif;
  --fuente-mono: 'Overpass Mono', ui-monospace, 'SFMono-Regular', Menlo, monospace;
  --t-12: 0.75rem;
  --t-13: 0.8125rem;
  --t-14: 0.875rem;
  --t-15: 0.9375rem;
  --t-16: 1rem;
  --t-18: 1.125rem;
  --t-20: 1.25rem;
  --t-24: 1.5rem;
  --t-32: 2rem;
  --t-40: 2.5rem;

  /* Espaciado (base 4 px) */
  --e-1: 4px;
  --e-2: 8px;
  --e-3: 12px;
  --e-4: 16px;
  --e-5: 20px;
  --e-6: 24px;
  --e-8: 32px;
  --e-10: 40px;
  --e-12: 48px;
  --e-16: 64px;

  /* Forma: controles e insignias 6 px, paneles 10 px, contadores en pastilla */
  --r-control: 6px;
  --r-panel: 10px;
  --r-pastilla: 999px;

  /* Sombras teñidas de asfalto, solo en lo flotante */
  --sombra-1: 0 1px 2px rgb(30 33 36 / 0.06);
  --sombra-2: 0 8px 24px rgb(30 33 36 / 0.12), 0 2px 6px rgb(30 33 36 / 0.08);

  /* Capas */
  --z-barra: 100;
  --z-tabbar: 110;
  --z-avisos: 300;

  /* Movimiento (Emil Kowalski) */
  --ease-out: cubic-bezier(0.23, 1, 0.32, 1);
  --ease-in-out: cubic-bezier(0.77, 0, 0.175, 1);
  --ease-drawer: cubic-bezier(0.32, 0.72, 0, 1);
  --d-press: 140ms;
  --d-fast: 180ms;
  --d-base: 240ms;
  --d-hoja: 320ms;
  --d-ruta: 420ms;

  /* Layout */
  --barra-h: 56px;
  --tabbar-h: 64px;
  --contenido-max: 1240px;
  --margen: 32px;
}

@media (max-width: 767px) {
  :root {
    --barra-h: 52px;
    --margen: 16px;
  }
}
```

- [ ] **Step 5: Crear `src/styles/base.css`**

```css
/* ══ Base: reset, tipografía, foco y utilidades ══ */
*,
*::before,
*::after {
  box-sizing: border-box;
}

* {
  margin: 0;
}

html {
  -webkit-text-size-adjust: 100%;
  text-size-adjust: 100%;
  scroll-padding-top: calc(var(--barra-h) + 16px);
}

body {
  min-height: 100dvh;
  background: var(--suelo);
  color: var(--tinta);
  font-family: var(--fuente);
  font-size: var(--t-15);
  line-height: 1.45;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  -webkit-tap-highlight-color: transparent;
}

img,
svg {
  display: block;
  max-width: 100%;
}

button,
input,
select,
textarea {
  font: inherit;
  color: inherit;
}

button {
  cursor: pointer;
}

a {
  color: inherit;
}

ul,
ol {
  padding: 0;
  list-style: none;
}

h1,
h2,
h3,
h4 {
  font-weight: 600;
  line-height: 1.2;
  text-wrap: balance;
}

p {
  text-wrap: pretty;
}

code,
kbd,
pre {
  font-family: var(--fuente-mono);
}

::selection {
  background: var(--via-suave);
  color: var(--tinta);
}

:focus-visible {
  outline: 2px solid var(--rojo);
  outline-offset: 2px;
}

.cifra {
  font-variant-numeric: tabular-nums;
}

.mono {
  font-family: var(--fuente-mono);
  font-size: 0.92em;
  font-variant-numeric: tabular-nums;
}

.oculto {
  display: none !important;
}

.oculto-visual {
  position: absolute !important;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
  border: 0;
}

/* Enlace de salto: aparece solo con teclado */
.saltar {
  position: fixed;
  left: var(--e-4);
  top: var(--e-2);
  z-index: calc(var(--z-avisos) + 1);
  padding: var(--e-2) var(--e-4);
  border: 0;
  border-radius: var(--r-control);
  background: var(--asfalto);
  color: #fff;
  font-weight: 600;
  transform: translateY(-160%);
}

.saltar:focus-visible {
  transform: translateY(0);
}

/* Transición entre pantallas (withViewTransitions) */
::view-transition-old(root) {
  animation-duration: 120ms;
  animation-timing-function: var(--ease-out);
}

::view-transition-new(root) {
  animation-duration: 160ms;
  animation-timing-function: var(--ease-out);
}

@media (prefers-reduced-motion: reduce) {
  ::view-transition-group(*),
  ::view-transition-old(*),
  ::view-transition-new(*) {
    animation: none !important;
  }
}
```

- [ ] **Step 6: Crear `src/styles/componentes.css` (parte 1 de 3: página, botones y formularios)**

Crea el archivo con este contenido; los Steps 7 y 8 agregan las partes 2 y 3 al final del mismo archivo.

```css
/* ══ Componentes del sistema «Hitos de carretera» ══ */

/* ── Página ── */
.pagina {
  width: 100%;
  max-width: var(--contenido-max);
  margin: 0 auto;
  padding: var(--e-8) var(--margen) var(--e-16);
}

.pagina--media {
  max-width: 880px;
}

.pagina--angosta {
  max-width: 640px;
}

.pagina__encabezado {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--e-4) var(--e-6);
  margin-bottom: var(--e-6);
}

.pagina__titulo {
  font-size: var(--t-32);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.pagina__bajada {
  margin-top: var(--e-1);
  max-width: 62ch;
  color: var(--tinta-2);
}

.pagina__acciones {
  display: flex;
  flex-wrap: wrap;
  gap: var(--e-2);
}

.migas {
  margin-bottom: var(--e-1);
  font-size: var(--t-13);
  font-weight: 600;
  color: var(--tinta-3);
}

.seccion-titulo {
  font-size: var(--t-18);
  font-weight: 600;
}

.seccion-cabecera {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--e-4);
  margin-bottom: var(--e-3);
}

.volver {
  display: inline-flex;
  align-items: center;
  gap: var(--e-1);
  margin-bottom: var(--e-4);
  padding: var(--e-1) 0;
  color: var(--tinta-2);
  font-size: var(--t-14);
  font-weight: 600;
  text-decoration: none;
}

.volver:hover {
  color: var(--tinta);
}

.texto-secundario {
  color: var(--tinta-2);
}

.a-la-derecha {
  text-align: right;
}

@media (max-width: 767px) {
  .pagina {
    padding-top: var(--e-5);
    padding-bottom: calc(var(--tabbar-h) + env(safe-area-inset-bottom) + var(--e-8));
  }

  .pagina__titulo {
    font-size: var(--t-24);
  }

  .pagina__acciones {
    width: 100%;
  }

  .pagina__acciones .btn {
    flex: 1;
  }
}

/* ── Enlaces ── */
.enlace {
  color: var(--rojo-tinta);
  font-weight: 600;
  text-decoration: underline;
  text-decoration-thickness: 1px;
  text-underline-offset: 3px;
}

.enlace:hover {
  color: var(--rojo-presionado);
}

.enlace-boton {
  padding: 0;
  border: 0;
  background: none;
  color: var(--rojo-tinta);
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* ── Botones ── */
.btn {
  --btn-alto: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--e-2);
  min-height: var(--btn-alto);
  padding: 0 var(--e-4);
  border: 1px solid transparent;
  border-radius: var(--r-control);
  font-size: var(--t-14);
  font-weight: 600;
  line-height: 1;
  white-space: nowrap;
  text-decoration: none;
  user-select: none;
  touch-action: manipulation;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.btn:disabled,
.btn[aria-disabled='true'] {
  opacity: 0.45;
  cursor: not-allowed;
}

@media (prefers-reduced-motion: no-preference) {
  .btn {
    transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease,
      transform var(--d-press) var(--ease-out);
  }

  .btn:active:not(:disabled) {
    transform: scale(0.97);
  }
}

.btn--primario {
  background: var(--rojo);
  color: #fff;
}

.btn--primario:active:not(:disabled) {
  background: var(--rojo-presionado);
}

.btn--secundario {
  background: var(--panel);
  border-color: var(--linea-2);
  color: var(--tinta);
}

.btn--fantasma {
  background: transparent;
  color: var(--tinta-2);
}

.btn--peligro {
  background: var(--rojo-suave);
  border-color: var(--rojo-borde);
  color: var(--rojo-tinta);
}

.btn--sobre-asfalto {
  background: var(--asfalto-2);
  border-color: var(--asfalto-3);
  color: #fff;
}

@media (hover: hover) and (pointer: fine) {
  .btn--primario:hover:not(:disabled) {
    background: var(--rojo-hover);
  }

  .btn--secundario:hover:not(:disabled) {
    background: var(--panel-2);
    border-color: var(--tinta-3);
  }

  .btn--fantasma:hover:not(:disabled) {
    background: var(--panel-2);
    color: var(--tinta);
  }

  .btn--peligro:hover:not(:disabled) {
    background: #fbdbe2;
  }

  .btn--sobre-asfalto:hover:not(:disabled) {
    background: var(--asfalto-3);
  }
}

.btn--sm {
  --btn-alto: 32px;
  padding: 0 var(--e-3);
  font-size: var(--t-13);
}

.btn--lg {
  --btn-alto: 48px;
  padding: 0 var(--e-6);
  font-size: var(--t-16);
}

.btn--icono {
  width: var(--btn-alto);
  padding: 0;
}

.btn--bloque {
  width: 100%;
}

@media (max-width: 767px) {
  .btn {
    --btn-alto: 44px;
  }

  .btn--sm {
    --btn-alto: 36px;
  }
}

/* ── Formularios ── */
.formulario {
  display: grid;
  gap: var(--e-5);
}

.formulario--rejilla {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: var(--e-4);
}

.formulario--rejilla .campo {
  grid-column: 1 / -1;
}

.formulario--rejilla .campo--medio {
  grid-column: auto;
}

.formulario__fila {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--e-4);
}

.formulario__acciones {
  display: flex;
  flex-wrap: wrap;
  gap: var(--e-2);
  padding-top: var(--e-2);
}

@media (max-width: 767px) {
  .formulario--rejilla,
  .formulario__fila {
    grid-template-columns: 1fr;
  }
}

.campo {
  display: grid;
  align-content: start;
  gap: var(--e-2);
}

.campo__etiqueta {
  font-size: var(--t-13);
  font-weight: 500;
  color: var(--tinta);
}

.req {
  color: var(--rojo);
}

.campo__ayuda {
  font-size: var(--t-13);
  color: var(--tinta-2);
}

.campo__error {
  font-size: var(--t-13);
  font-weight: 500;
  color: var(--rojo-tinta);
}

.campo--con-error .campo__etiqueta {
  color: var(--rojo-tinta);
}

/* ── Controles ── */
.control {
  width: 100%;
  min-height: 40px;
  padding: 0 var(--e-3);
  border: 1px solid var(--linea-2);
  border-radius: var(--r-control);
  background: var(--panel);
  color: var(--tinta);
  font-size: var(--t-15);
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

textarea.control {
  min-height: 88px;
  padding: var(--e-2) var(--e-3);
  line-height: 1.45;
  resize: vertical;
}

.control::placeholder {
  color: var(--tinta-3);
}

.control:hover {
  border-color: var(--tinta-3);
}

.control:focus,
.control:focus-visible {
  outline: none;
  border-color: var(--tinta);
  box-shadow: 0 0 0 3px rgb(225 29 72 / 0.2);
}

.control[aria-invalid='true'] {
  border-color: var(--rojo);
}

.control:disabled {
  background: var(--panel-2);
  color: var(--tinta-3);
  cursor: not-allowed;
}

@media (max-width: 767px) {
  .control {
    min-height: 44px;
    font-size: var(--t-16);
  }
}

.control-prefijo {
  position: relative;
}

.control-prefijo__texto {
  position: absolute;
  left: var(--e-3);
  top: 50%;
  transform: translateY(-50%);
  color: var(--tinta-2);
  font-weight: 600;
  pointer-events: none;
}

.control-prefijo .control {
  padding-left: 36px;
  font-variant-numeric: tabular-nums;
}

.control-con-icono {
  position: relative;
  display: flex;
  align-items: center;
}

.control-con-icono > .icono {
  position: absolute;
  left: var(--e-3);
  color: var(--tinta-3);
  pointer-events: none;
}

.control-con-icono .control {
  padding-left: 40px;
}

.control-con-icono:has(.control-con-icono__boton) .control {
  padding-right: 44px;
}

.control-con-icono__boton {
  position: absolute;
  right: 4px;
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: var(--r-control);
  background: transparent;
  color: var(--tinta-2);
}

.control-con-icono__boton:hover {
  background: var(--panel-2);
  color: var(--tinta);
}

.casilla {
  display: inline-flex;
  align-items: center;
  gap: var(--e-2);
  font-size: var(--t-14);
  color: var(--tinta-2);
  cursor: pointer;
}

.casilla input {
  width: 18px;
  height: 18px;
  accent-color: var(--rojo);
}
```

- [ ] **Step 7: Continuar `src/styles/componentes.css` (parte 2 de 3: estados, listas y vacíos)**

Agrega al final de `src/styles/componentes.css`:

```css
/* ── Insignias de estado (las clases lbl-* vienen de gasto-estados.ts) ── */
.lbl {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 var(--e-2);
  border-radius: var(--r-control);
  background: var(--registrado-suave);
  color: var(--registrado-tinta);
  font-size: var(--t-12);
  font-weight: 600;
  white-space: nowrap;
}

.lbl__poste {
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  border-radius: 2px;
  background: var(--registrado);
}

.lbl-n {
  background: var(--registrado-suave);
  color: var(--registrado-tinta);
}

.lbl-n .lbl__poste {
  background: var(--registrado);
}

.lbl-info {
  background: var(--pagado-suave);
  color: var(--pagado-tinta);
}

.lbl-info .lbl__poste {
  background: var(--pagado);
}

.lbl-ok {
  background: var(--sustentado-suave);
  color: var(--sustentado-tinta);
}

.lbl-ok .lbl__poste {
  background: var(--sustentado);
}

.lbl-warn {
  background: var(--pendiente-suave);
  color: var(--pendiente-tinta);
}

.lbl-warn .lbl__poste {
  background: var(--pendiente);
  box-shadow: inset 0 0 0 1px rgb(107 83 0 / 0.35);
}

.lbl-err {
  background: var(--rechazado-suave);
  color: var(--rechazado-tinta);
}

.lbl-err .lbl__poste {
  background: var(--rechazado);
}

.lbl-y {
  background: var(--rojo-suave);
  color: var(--rojo-tinta);
}

/* ── Contadores ── */
.contador {
  display: inline-grid;
  place-items: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--r-pastilla);
  background: var(--panel-2);
  color: var(--tinta-2);
  font-size: var(--t-12);
  font-weight: 700;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.contador--via {
  background: var(--via);
  color: var(--asfalto);
}

/* ── Panel y alertas ── */
.panel {
  padding: var(--e-6);
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--panel);
}

@media (max-width: 767px) {
  .panel {
    padding: var(--e-4);
  }
}

.alerta {
  display: flex;
  align-items: flex-start;
  gap: var(--e-3);
  padding: var(--e-3) var(--e-4);
  border-radius: var(--r-control);
  font-size: var(--t-14);
  line-height: 1.45;
}

.alerta > .icono {
  flex-shrink: 0;
  margin-top: 1px;
}

.alerta--error {
  background: var(--rojo-suave);
  color: var(--rojo-tinta);
}

.alerta--exito {
  background: var(--sustentado-suave);
  color: var(--sustentado-tinta);
}

.alerta--via {
  background: var(--via-suave);
  color: var(--via-tinta);
}

.alerta__lista {
  display: flex;
  flex-wrap: wrap;
  gap: var(--e-1) var(--e-3);
  margin-top: var(--e-1);
}

/* ── Pestañas ── */
.pestanas {
  display: flex;
  gap: var(--e-1);
  margin-bottom: var(--e-5);
  border-bottom: 1px solid var(--linea);
  overflow-x: auto;
  scrollbar-width: none;
  scroll-snap-type: x proximity;
}

.pestanas::-webkit-scrollbar {
  display: none;
}

.pestanas__item {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: var(--e-2);
  min-height: 44px;
  padding: 0 var(--e-3);
  color: var(--tinta-2);
  font-size: var(--t-14);
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
  scroll-snap-align: start;
}

.pestanas__item::after {
  content: '';
  position: absolute;
  left: var(--e-2);
  right: var(--e-2);
  bottom: -1px;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: transparent;
}

.pestanas__item:hover,
.pestanas__item--activa {
  color: var(--tinta);
}

.pestanas__item--activa::after {
  background: var(--rojo);
}

/* ── Barra de herramientas y buscador ── */
.barra-herramientas {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--e-3);
  margin-bottom: var(--e-4);
}

.barra-herramientas__conteo {
  margin-left: auto;
  color: var(--tinta-3);
  font-size: var(--t-13);
  font-variant-numeric: tabular-nums;
}

.buscador {
  position: relative;
  flex: 1 1 260px;
  max-width: 360px;
}

.buscador > .icono {
  position: absolute;
  left: var(--e-3);
  top: 50%;
  transform: translateY(-50%);
  color: var(--tinta-3);
  pointer-events: none;
}

.buscador__input {
  padding-left: 36px;
}

@media (max-width: 767px) {
  .buscador {
    flex-basis: 100%;
    max-width: none;
  }

  .barra-herramientas__conteo {
    margin-left: 0;
  }
}

/* ── Tabla ── */
.tabla-marco {
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--panel);
  overflow-x: auto;
}

.tabla {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--t-14);
}

.tabla th {
  padding: var(--e-3) var(--e-4);
  border-bottom: 1px solid var(--linea);
  background: var(--panel);
  color: var(--tinta-2);
  font-size: var(--t-12);
  font-weight: 600;
  text-align: left;
  white-space: nowrap;
}

.tabla td {
  height: 48px;
  padding: var(--e-3) var(--e-4);
  border-bottom: 1px solid var(--linea);
  vertical-align: middle;
}

.tabla tbody tr:last-child td {
  border-bottom: 0;
}

@media (hover: hover) and (pointer: fine) {
  .tabla tbody tr:hover {
    background: var(--panel-2);
  }
}

.tabla__acciones {
  width: 1%;
  text-align: right;
  white-space: nowrap;
}

/* Celdas generadas por cruds.data.ts */
.celda-vacia {
  color: var(--tinta-3);
}

.celda-secundaria {
  color: var(--tinta-2);
}

.celda-doble {
  display: grid;
  gap: 2px;
}

.celda-doble .celda-secundaria {
  font-size: var(--t-13);
}

.id-tag {
  color: var(--tinta-3);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.plate {
  display: inline-flex;
  align-items: stretch;
  overflow: hidden;
  border: 1.5px solid var(--tinta);
  border-radius: 4px;
  background: #fff;
}

.plate-pe {
  display: grid;
  place-items: center;
  padding: 0 4px;
  background: var(--pagado);
  color: #fff;
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.06em;
}

.plate-num {
  padding: 1px 8px;
  color: var(--tinta);
  font-family: var(--fuente-mono);
  font-weight: 600;
  letter-spacing: 0.08em;
}

.route {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.route svg {
  flex-shrink: 0;
  color: var(--tinta-3);
}

.route span {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.money {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.money .sy {
  margin-right: 2px;
  color: var(--tinta-3);
  font-weight: 400;
}

/* ── Estado vacío ── */
.vacio {
  display: grid;
  justify-items: center;
  gap: var(--e-2);
  padding: var(--e-12) var(--e-6);
  border: 1px dashed var(--linea-2);
  border-radius: var(--r-panel);
  background: var(--panel);
  text-align: center;
}

.vacio--compacto {
  padding: var(--e-6);
}

.vacio__icono {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  margin-bottom: var(--e-2);
  border-radius: var(--r-panel);
  background: var(--panel-2);
  color: var(--tinta-2);
}

.vacio__titulo {
  font-size: var(--t-16);
  font-weight: 600;
}

.vacio__texto {
  max-width: 46ch;
  color: var(--tinta-2);
}

.vacio__acciones {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--e-2);
  margin-top: var(--e-3);
}

.vacio__acciones:empty {
  display: none;
}

/* ── Esqueletos de carga ── */
.esqueleto {
  border-radius: var(--r-control);
  background: var(--panel-2);
}

.esqueleto--fila {
  height: 56px;
  border-radius: 0;
  border-bottom: 1px solid var(--panel);
}

.esqueleto--bloque {
  height: 120px;
  border-radius: var(--r-panel);
}

.esqueleto--franja {
  height: 72px;
  border-radius: var(--r-panel);
}

@media (prefers-reduced-motion: no-preference) {
  .esqueleto {
    animation: pulso-esqueleto 1.2s ease-in-out infinite;
  }
}

@keyframes pulso-esqueleto {
  50% {
    opacity: 0.55;
  }
}
```

- [ ] **Step 8: Terminar `src/styles/componentes.css` (parte 3 de 3: hojas, diálogos, avisos, selector y piezas)**

Agrega al final de `src/styles/componentes.css`:

```css
/* ── Hojas y diálogos (<dialog> nativo; capa superior del navegador) ── */
dialog {
  max-width: none;
  max-height: none;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
}

.hoja::backdrop,
.dialogo::backdrop {
  background: rgb(30 33 36 / 0.4);
}

.hoja {
  position: fixed;
  inset: 0 0 0 auto;
  width: min(440px, 100vw);
  height: 100dvh;
  margin: 0;
}

.hoja__marco {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--panel);
  box-shadow: var(--sombra-2);
}

.hoja__cabecera {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--e-3);
  padding: var(--e-4) var(--e-4) var(--e-4) var(--e-6);
  border-bottom: 1px solid var(--linea);
}

.hoja__titulo {
  font-size: var(--t-18);
  font-weight: 600;
}

.hoja__cuerpo {
  flex: 1;
  overflow-y: auto;
  padding: var(--e-5) var(--e-6) var(--e-6);
}

.hoja__nota {
  margin-bottom: var(--e-5);
  color: var(--tinta-2);
  font-size: var(--t-13);
}

.hoja__grupo {
  margin: var(--e-5) 0 var(--e-2);
  color: var(--tinta-3);
  font-size: var(--t-13);
  font-weight: 600;
}

.hoja__grupo:first-child {
  margin-top: 0;
}

.hoja__pie {
  display: flex;
  gap: var(--e-2);
  padding: var(--e-4) var(--e-6) calc(var(--e-4) + env(safe-area-inset-bottom));
  border-top: 1px solid var(--linea);
  background: var(--panel);
}

.hoja--inferior {
  inset: auto 0 0 0;
  width: 100vw;
  height: auto;
  max-height: 85dvh;
}

.hoja--inferior .hoja__marco {
  max-height: 85dvh;
  border-radius: var(--r-panel) var(--r-panel) 0 0;
}

@media (max-width: 767px) {
  .hoja {
    inset: auto 0 0 0;
    width: 100vw;
    height: auto;
    max-height: 92dvh;
  }

  .hoja__marco {
    max-height: 92dvh;
    border-radius: var(--r-panel) var(--r-panel) 0 0;
  }

  .hoja__cuerpo {
    padding: var(--e-4);
  }

  .hoja__pie {
    padding-left: var(--e-4);
    padding-right: var(--e-4);
  }

  .hoja__pie .btn {
    flex: 1;
  }
}

.dialogo {
  width: min(640px, calc(100vw - 32px));
  margin: auto;
}

.dialogo__marco {
  overflow: hidden;
  border-radius: var(--r-panel);
  background: var(--panel);
  box-shadow: var(--sombra-2);
}

.dialogo__cabecera {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--e-3);
  padding: var(--e-4) var(--e-4) var(--e-4) var(--e-6);
  border-bottom: 1px solid var(--linea);
}

.dialogo__cabecera h2 {
  font-size: var(--t-16);
}

.dialogo__json {
  max-height: 60dvh;
  margin: 0;
  padding: var(--e-5) var(--e-6);
  overflow: auto;
  background: var(--suelo);
  color: var(--tinta-2);
  font-size: var(--t-13);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

/* Entradas y salidas: la hoja se desliza (320 ms ease-drawer), sale en 200 ms */
@media (prefers-reduced-motion: no-preference) {
  .hoja {
    transform: translateX(100%);
    transition: transform 200ms var(--ease-out), display 200ms allow-discrete, overlay 200ms allow-discrete;
  }

  .hoja[open] {
    transform: translateX(0);
    transition-duration: var(--d-hoja);
    transition-timing-function: var(--ease-drawer);
  }

  @starting-style {
    .hoja[open] {
      transform: translateX(100%);
    }
  }

  .hoja--inferior {
    transform: translateY(100%);
  }

  .hoja--inferior[open] {
    transform: translateY(0);
  }

  @starting-style {
    .hoja--inferior[open] {
      transform: translateY(100%);
    }
  }

  .dialogo {
    opacity: 0;
    transform: scale(0.97);
    transition: opacity 150ms ease, transform 200ms var(--ease-out), display 200ms allow-discrete, overlay 200ms allow-discrete;
  }

  .dialogo[open] {
    opacity: 1;
    transform: scale(1);
  }

  @starting-style {
    .dialogo[open] {
      opacity: 0;
      transform: scale(0.97);
    }
  }

  .hoja::backdrop,
  .dialogo::backdrop {
    opacity: 0;
    transition: opacity 200ms ease, display 200ms allow-discrete, overlay 200ms allow-discrete;
  }

  .hoja[open]::backdrop,
  .dialogo[open]::backdrop {
    opacity: 1;
  }

  @starting-style {
    .hoja[open]::backdrop,
    .dialogo[open]::backdrop {
      opacity: 0;
    }
  }
}

@media (prefers-reduced-motion: no-preference) and (max-width: 767px) {
  .hoja {
    transform: translateY(100%);
  }

  .hoja[open] {
    transform: translateY(0);
  }

  @starting-style {
    .hoja[open] {
      transform: translateY(100%);
    }
  }
}

/* ── Avisos (toasts) ── */
.avisos {
  position: fixed;
  z-index: var(--z-avisos);
  right: var(--e-5);
  bottom: var(--e-5);
  display: grid;
  gap: var(--e-2);
  width: min(380px, calc(100vw - 32px));
  pointer-events: none;
}

@media (max-width: 767px) {
  .avisos {
    left: 50%;
    right: auto;
    bottom: var(--e-4);
    transform: translateX(-50%);
  }

  body:has(.tabbar) .avisos {
    bottom: calc(var(--tabbar-h) + env(safe-area-inset-bottom) + var(--e-3));
  }
}

.aviso {
  display: flex;
  align-items: flex-start;
  gap: var(--e-3);
  padding: var(--e-3) var(--e-3) var(--e-3) var(--e-4);
  border-left: 4px solid var(--asfalto-3);
  border-radius: var(--r-panel);
  background: var(--asfalto);
  color: #fff;
  box-shadow: var(--sombra-2);
  pointer-events: auto;
}

.aviso__icono {
  flex-shrink: 0;
  margin-top: 1px;
}

.aviso--ok {
  border-left-color: var(--sustentado);
}

.aviso--ok .aviso__icono {
  color: #6bd08a;
}

.aviso--err {
  border-left-color: var(--rojo);
}

.aviso--err .aviso__icono {
  color: #ff8aa3;
}

.aviso--warn {
  border-left-color: var(--via);
}

.aviso--warn .aviso__icono {
  color: var(--via);
}

.aviso--info {
  border-left-color: var(--pagado);
}

.aviso--info .aviso__icono {
  color: #8fb3ff;
}

.aviso__texto {
  flex: 1;
  min-width: 0;
}

.aviso__titulo {
  font-size: var(--t-14);
  font-weight: 600;
  line-height: 1.4;
}

.aviso__mensaje {
  margin-top: 2px;
  color: var(--asfalto-tinta);
  font-size: var(--t-13);
}

.aviso__cerrar {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: var(--r-control);
  background: transparent;
  color: var(--asfalto-tinta);
}

.aviso__cerrar:hover {
  background: var(--asfalto-2);
  color: #fff;
}

@media (prefers-reduced-motion: no-preference) {
  .aviso {
    transition: opacity var(--d-base) var(--ease-out), transform var(--d-base) var(--ease-out);
  }

  @starting-style {
    .aviso {
      opacity: 0;
      transform: translateY(16px);
    }
  }

  .aviso--saliendo {
    animation: aviso-salida 150ms var(--ease-out) forwards;
  }
}

@media (prefers-reduced-motion: reduce) {
  .aviso--saliendo {
    animation: aviso-salida-suave 120ms ease forwards;
  }
}

@keyframes aviso-salida {
  to {
    opacity: 0;
    transform: translateY(8px) scale(0.98);
  }
}

@keyframes aviso-salida-suave {
  to {
    opacity: 0;
  }
}

/* ── Selector de registros relacionados (combobox con lista en el flujo) ── */
.combo {
  position: relative;
  display: block;
}

.combo__control {
  position: relative;
}

.combo__input {
  padding-right: 36px;
  text-overflow: ellipsis;
}

.combo__chevron {
  position: absolute;
  right: var(--e-3);
  top: 50%;
  transform: translateY(-50%);
  color: var(--tinta-3);
  pointer-events: none;
}

.combo__lista {
  max-height: 240px;
  margin-top: var(--e-1);
  padding: var(--e-1);
  overflow-y: auto;
  border: 1px solid var(--linea-2);
  border-radius: var(--r-control);
  background: var(--panel);
  box-shadow: var(--sombra-1);
}

.combo__opcion {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--e-2);
  min-height: 36px;
  padding: var(--e-2) var(--e-3);
  border-radius: 4px;
  font-size: var(--t-14);
  cursor: pointer;
}

.combo__opcion--activa {
  background: var(--panel-2);
}

.combo__opcion[aria-selected='true'] {
  font-weight: 600;
}

.combo__nota {
  padding: var(--e-2) var(--e-3);
  color: var(--tinta-2);
  font-size: var(--t-13);
}

@media (max-width: 767px) {
  .combo__opcion {
    min-height: 44px;
  }
}

/* ── Piezas pequeñas ── */
kbd {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 6px;
  border: 1px solid currentColor;
  border-radius: 4px;
  font-size: var(--t-12);
  opacity: 0.75;
}

.girador {
  width: 16px;
  height: 16px;
  border: 2px solid rgb(255 255 255 / 0.35);
  border-top-color: #fff;
  border-radius: 50%;
}

@media (prefers-reduced-motion: no-preference) {
  .girador {
    animation: girar 0.7s linear infinite;
  }
}

@keyframes girar {
  to {
    transform: rotate(360deg);
  }
}

.icono {
  display: inline-flex;
  flex-shrink: 0;
}

/* ── Marca: insignia roja plana con el camión del logo ── */
.marca {
  display: inline-flex;
  align-items: center;
  gap: var(--e-2);
  color: #fff;
  text-decoration: none;
}

.marca--clara {
  color: var(--tinta);
}

.marca__insignia {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--rojo);
  color: #fff;
}

.marca__insignia .camion {
  width: 18px;
  height: 18px;
}

.marca__texto {
  display: grid;
  line-height: 1.1;
}

.marca__nombre {
  font-size: var(--t-16);
  font-weight: 800;
  letter-spacing: 0.02em;
}

.marca__sub {
  font-size: var(--t-12);
  font-weight: 500;
  opacity: 0.72;
}

.camion {
  display: block;
}

.camion svg {
  width: 100%;
  height: 100%;
}
```

- [ ] **Step 9: Verificar contraste AA de los pares de tokens**

Crea `$SCRATCH/verificar-contraste.mjs`:

```js
// Verifica contraste WCAG (AA = 4.5:1 para texto normal) de los pares de color del sistema.
const lum = hex => {
  const [r, g, b] = hex.match(/\w\w/g).map(h => parseInt(h, 16) / 255)
    .map(c => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
};
const ratio = (a, b) => {
  const [x, y] = [lum(a), lum(b)].sort((m, n) => n - m);
  return (x + 0.05) / (y + 0.05);
};
const pares = [
  ['tinta / suelo', '#1e2124', '#f7f7f5'], ['tinta-2 / suelo', '#4a5056', '#f7f7f5'],
  ['tinta-3 / suelo', '#656b71', '#f7f7f5'], ['tinta-3 / panel', '#656b71', '#fdfdfc'],
  ['tinta-3 / panel-2', '#656b71', '#f0f1ee'], ['blanco / rojo', '#ffffff', '#e11d48'],
  ['blanco / rojo-hover', '#ffffff', '#c8113d'], ['blanco / asfalto', '#ffffff', '#1e2124'],
  ['asfalto-tinta / asfalto', '#c9cdd1', '#1e2124'], ['asfalto-tinta / asfalto-2', '#c9cdd1', '#2a2e33'],
  ['asfalto / vía', '#1e2124', '#f2c200'], ['rojo-tinta / suelo', '#b4123a', '#f7f7f5'],
  ['rojo-tinta / rojo-suave', '#b4123a', '#fdecef'], ['registrado', '#4a5056', '#eeefec'],
  ['pagado', '#1e4fc2', '#e8effe'], ['sustentado', '#166534', '#e6f4ea'],
  ['pendiente', '#6b5300', '#fff5cc'], ['blanco / pagado (placa)', '#ffffff', '#2563eb'],
];
let fallos = 0;
for (const [nombre, a, b] of pares) {
  const r = ratio(a, b);
  if (r < 4.5) fallos++;
  console.log(`${r >= 4.5 ? 'OK   ' : 'FALLA'} ${r.toFixed(2)}  ${nombre}`);
}
process.exit(fallos ? 1 : 0);
```

Run: `node "$SCRATCH/verificar-contraste.mjs"`
Expected: todas las líneas `OK` y código de salida 0.

- [ ] **Step 10: Checkpoint (sin commit)**

Run: `npx ng build`
Expected: build exitoso, sin errores de presupuesto. En `dist/` aparecen los archivos `.woff2` de Overpass y Overpass Mono.

---
### Task 8: Primitivas de UI: camión y marca, insignia de estado y estado vacío

**Files:**
- Create: `frontend/src/app/shared/ui/marca/camion.component.ts`
- Create: `frontend/src/app/shared/ui/marca/marca.component.ts`
- Create: `frontend/src/app/shared/ui/insignia-estado/insignia-estado.component.ts`
- Create: `frontend/src/app/shared/ui/estado-vacio/estado-vacio.component.ts`
- Test: `frontend/src/app/shared/ui/marca/marca.component.spec.ts`
- Test: `frontend/src/app/shared/ui/insignia-estado/insignia-estado.component.spec.ts`
- Test: `frontend/src/app/shared/ui/estado-vacio/estado-vacio.component.spec.ts`

**Interfaces:**
- Consumes:
  - `ESTADO_GASTO_LABEL`, `ESTADO_GASTO_CLASS`, `ESTADO_COMPROBANTE_LABEL` y `ESTADO_COMPROBANTE_CLASS` (sin modificar)
  - `IconoComponent` (Tarea 2)
  - las clases `.marca`, `.camion`, `.lbl` y `.vacio` (Tarea 7)
- Produces:
  - `CamionComponent`, selector `app-camion`: el pictograma del logo, con la geometría original.
  - `MarcaComponent`, selector `app-marca`, con inputs `conTexto = true` y `sobreAsfalto = true`.
  - `InsigniaEstadoComponent`, selector `app-insignia-estado`, con inputs `estado` (requerido) y `tipo: 'gasto' | 'comprobante' = 'gasto'`.
  - `EstadoVacioComponent`, selector `app-estado-vacio`, con inputs `icono: IconoNombre = 'inbox'`, `titulo` (requerido), `texto = ''` y `compacto = false`. Proyecta las acciones con `<ng-content>`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/shared/ui/marca/marca.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { MarcaComponent } from './marca.component';

describe('MarcaComponent', () => {
  it('muestra el camión del logo con LIZAR y Fleet Manager Pro', () => {
    const f = TestBed.createComponent(MarcaComponent);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.marca__insignia app-camion svg')).not.toBeNull();
    expect(el.querySelector('.marca__nombre')?.textContent).toBe('LIZAR');
    expect(el.querySelector('.marca__sub')?.textContent).toBe('Fleet Manager Pro');
  });

  it('puede ir sin texto y en versión clara', () => {
    const f = TestBed.createComponent(MarcaComponent);
    f.componentRef.setInput('conTexto', false);
    f.componentRef.setInput('sobreAsfalto', false);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.marca__texto')).toBeNull();
    expect(el.classList).toContain('marca--clara');
  });
});
```

`frontend/src/app/shared/ui/insignia-estado/insignia-estado.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { InsigniaEstadoComponent } from './insignia-estado.component';

function insignia(estado: string, tipo?: 'gasto' | 'comprobante'): HTMLElement {
  const f = TestBed.createComponent(InsigniaEstadoComponent);
  f.componentRef.setInput('estado', estado);
  if (tipo) f.componentRef.setInput('tipo', tipo);
  f.detectChanges();
  return f.nativeElement.querySelector('.lbl');
}

describe('InsigniaEstadoComponent', () => {
  it('usa la etiqueta y el color exactos de un gasto', () => {
    const el = insignia('Pagado');
    expect(el.textContent?.trim()).toBe('Pagado');
    expect(el.classList).toContain('lbl-info');
  });

  it('usa la etiqueta exacta de un comprobante pendiente', () => {
    const el = insignia('Pendiente', 'comprobante');
    expect(el.textContent?.trim()).toBe('Pendiente de aprobación');
    expect(el.classList).toContain('lbl-warn');
  });
});
```

`frontend/src/app/shared/ui/estado-vacio/estado-vacio.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { EstadoVacioComponent } from './estado-vacio.component';

@Component({
  imports: [EstadoVacioComponent],
  template: `
    <app-estado-vacio icono="inbox" titulo="Aún no hay rubros." texto="Agrega el primero.">
      <button type="button">Nuevo rubro</button>
    </app-estado-vacio>
  `,
})
class Anfitrion {}

describe('EstadoVacioComponent', () => {
  it('muestra título, texto y la acción proyectada', () => {
    const f = TestBed.createComponent(Anfitrion);
    f.detectChanges();
    const el: HTMLElement = f.nativeElement;
    expect(el.querySelector('.vacio__titulo')?.textContent).toBe('Aún no hay rubros.');
    expect(el.querySelector('.vacio__texto')?.textContent).toBe('Agrega el primero.');
    expect(el.querySelector('.vacio__acciones button')?.textContent).toBe('Nuevo rubro');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `npx ng test --watch=false --include "src/app/shared/ui/**/*.spec.ts"`
Expected: FAIL. No se pueden resolver `./marca.component`, `./insignia-estado.component` ni `./estado-vacio.component`; la prueba de iconos sigue pasando.

- [ ] **Step 3: Implementar**

`frontend/src/app/shared/ui/marca/camion.component.ts`:

```ts
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
```

`frontend/src/app/shared/ui/marca/marca.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { CamionComponent } from './camion.component';

@Component({
  selector: 'app-marca',
  imports: [CamionComponent],
  template: `
    <span class="marca__insignia" aria-hidden="true"><app-camion /></span>
    @if (conTexto()) {
      <span class="marca__texto">
        <span class="marca__nombre">LIZAR</span>
        <span class="marca__sub">Fleet Manager Pro</span>
      </span>
    }
  `,
  host: { class: 'marca', '[class.marca--clara]': '!sobreAsfalto()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MarcaComponent {
  readonly conTexto = input(true);
  readonly sobreAsfalto = input(true);
}
```

`frontend/src/app/shared/ui/insignia-estado/insignia-estado.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { EstadoComprobante, EstadoGasto } from '../../../core/models/gasto.model';
import {
  ESTADO_COMPROBANTE_CLASS, ESTADO_COMPROBANTE_LABEL, ESTADO_GASTO_CLASS, ESTADO_GASTO_LABEL,
} from '../../../core/models/gasto-estados';

/** Estado de un gasto o comprobante con el copy y el color exactos de gasto-estados.ts. */
@Component({
  selector: 'app-insignia-estado',
  template: `<span [class]="clase()"><span class="lbl__poste" aria-hidden="true"></span>{{ etiqueta() }}</span>`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class InsigniaEstadoComponent {
  readonly estado = input.required<EstadoGasto | EstadoComprobante>();
  readonly tipo = input<'gasto' | 'comprobante'>('gasto');

  protected readonly etiqueta = computed(() => {
    const e = this.estado();
    return this.tipo() === 'gasto'
      ? ESTADO_GASTO_LABEL[e as EstadoGasto] ?? e
      : ESTADO_COMPROBANTE_LABEL[e as EstadoComprobante] ?? e;
  });

  protected readonly clase = computed(() => {
    const e = this.estado();
    const c = this.tipo() === 'gasto' ? ESTADO_GASTO_CLASS[e as EstadoGasto] : ESTADO_COMPROBANTE_CLASS[e as EstadoComprobante];
    return `lbl ${c ?? 'lbl-n'}`;
  });
}
```

`frontend/src/app/shared/ui/estado-vacio/estado-vacio.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { IconoComponent } from '../icono/icono.component';
import { IconoNombre } from '../icono/iconos';

/** Estado vacío que enseña qué hacer: ícono, título, una frase y las acciones proyectadas. */
@Component({
  selector: 'app-estado-vacio',
  imports: [IconoComponent],
  template: `
    <div class="vacio" [class.vacio--compacto]="compacto()">
      <span class="vacio__icono"><app-icono [nombre]="icono()" [tamano]="22" /></span>
      <p class="vacio__titulo">{{ titulo() }}</p>
      @if (texto()) {
        <p class="vacio__texto">{{ texto() }}</p>
      }
      <div class="vacio__acciones"><ng-content /></div>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EstadoVacioComponent {
  readonly icono = input<IconoNombre>('inbox');
  readonly titulo = input.required<string>();
  readonly texto = input('');
  readonly compacto = input(false);
}
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: `npx ng test --watch=false --include "src/app/shared/ui/**/*.spec.ts"`
Expected: PASS (7 pruebas: 2 de iconos, 2 de marca, 2 de insignia y 1 de estado vacío).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false`
Expected: PASS en todas las pruebas.

---
### Task 9: La firma: la ruta del gasto (`RutaGastoComponent`)

**Files:**
- Create: `frontend/src/app/shared/ui/ruta-gasto/ruta-gasto.component.ts`
- Create: `frontend/src/app/shared/ui/ruta-gasto/ruta-gasto.component.html`
- Create: `frontend/src/styles/ruta.css`
- Modify: `frontend/src/styles.css` (agregar import)
- Test: `frontend/src/app/shared/ui/ruta-gasto/ruta-gasto.component.spec.ts`

**Interfaces:**
- Consumes:
  - `PasoRuta` y `etiquetaRuta` (Tarea 3)
  - `formatFechaHora` (Tarea 1)
  - `ESTADO_GASTO_LABEL`
  - `CamionComponent` (Tarea 8)
  - el tipo `Comprobante`
- Produces: `RutaGastoComponent`, selector `app-ruta-gasto`.
  - Inputs: `pasos: PasoRuta[]` (requerido), `variante: 'compacta' | 'completa' = 'compacta'` y `rechazo: Comprobante | undefined`.
  - Expone `--actual` (índice del hito actual) como estilo del host.
  - La variante compacta lleva `role="img"` y `aria-label`.

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/shared/ui/ruta-gasto/ruta-gasto.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { RutaGastoComponent } from './ruta-gasto.component';
import { PasoRuta } from '../../../core/state/gastos.logic';
import { Comprobante } from '../../../core/models/gasto.model';

const pasos: PasoRuta[] = [
  { estado: 'Registrado', alcanzado: true, actual: false, fecha: '2026-10-01T15:00:00Z', actor: 'Ana Quispe' },
  { estado: 'Pagado', alcanzado: true, actual: true, fecha: '2026-10-01T17:00:00Z' },
  { estado: 'Sustentado', alcanzado: false, actual: false },
];

function crear(variante: 'compacta' | 'completa', rechazo?: Comprobante): HTMLElement {
  const f = TestBed.createComponent(RutaGastoComponent);
  f.componentRef.setInput('pasos', pasos);
  f.componentRef.setInput('variante', variante);
  if (rechazo) f.componentRef.setInput('rechazo', rechazo);
  f.detectChanges();
  return f.nativeElement;
}

describe('RutaGastoComponent', () => {
  it('la versión compacta es una imagen descrita para lectores de pantalla', () => {
    const el = crear('compacta');
    expect(el.getAttribute('role')).toBe('img');
    expect(el.getAttribute('aria-label')).toBe('Recorrido: Registrado, Pagado (actual), Sustentado pendiente');
    expect(el.querySelectorAll('.ruta__hito')).toHaveLength(3);
    expect(el.querySelector('.ruta__nombre')).toBeNull();
  });

  it('la versión completa muestra estado, hora de Lima y quién', () => {
    const el = crear('completa');
    const hitos = el.querySelectorAll('.ruta__hito');
    expect(el.getAttribute('role')).toBeNull();
    expect(hitos[1].classList).toContain('ruta__hito--actual');
    expect(hitos[1].getAttribute('aria-current')).toBe('step');
    expect(hitos[0].textContent).toContain('01/10/2026 10:00');
    expect(hitos[0].textContent).toContain('Ana Quispe');
    expect(hitos[2].textContent).toContain('Pendiente');
  });

  it('dibuja el desvío cuando el último comprobante fue rechazado', () => {
    const el = crear('completa', {
      id: 5, gasto_id: 1, archivo_url: '/uploads/5.jpg', tipo_archivo: 'imagen', estado: 'Rechazado',
      motivo_rechazo: 'Monto no coincide', aprobado_en: '2026-10-01T16:30:00Z', creado_en: '2026-10-01T16:00:00Z',
    });
    expect(el.querySelector('.ruta__desvio')?.textContent).toContain('Comprobante rechazado el 01/10/2026 11:30: Monto no coincide');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/shared/ui/ruta-gasto/ruta-gasto.component.spec.ts`
Expected: FAIL. No se puede resolver `./ruta-gasto.component`.

- [ ] **Step 3: Implementar el componente**

`frontend/src/app/shared/ui/ruta-gasto/ruta-gasto.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, afterNextRender, computed, input, signal } from '@angular/core';
import { Comprobante } from '../../../core/models/gasto.model';
import { ESTADO_GASTO_LABEL } from '../../../core/models/gasto-estados';
import { PasoRuta, etiquetaRuta } from '../../../core/state/gastos.logic';
import { formatFechaHora } from '../../../core/utils/format';
import { CamionComponent } from '../marca/camion.component';

/**
 * La ruta del gasto: tres hitos (Registrado, Pagado, Sustentado) sobre una línea de carretera,
 * con el camión del logo en el hito actual. Firma visual de «Hitos de carretera».
 */
@Component({
  selector: 'app-ruta-gasto',
  imports: [CamionComponent],
  templateUrl: './ruta-gasto.component.html',
  host: {
    class: 'ruta',
    '[class.ruta--compacta]': "variante() === 'compacta'",
    '[class.ruta--completa]': "variante() === 'completa'",
    '[class.ruta--lista]': 'lista()',
    '[style.--actual]': 'indiceActual()',
    '[attr.role]': "variante() === 'compacta' ? 'img' : null",
    '[attr.aria-label]': "variante() === 'compacta' ? etiqueta() : null",
  },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RutaGastoComponent {
  readonly pasos = input.required<PasoRuta[]>();
  readonly variante = input<'compacta' | 'completa'>('compacta');
  readonly rechazo = input<Comprobante | undefined>(undefined);

  protected readonly etiqueta = computed(() => etiquetaRuta(this.pasos()));
  protected readonly indiceActual = computed(() => Math.max(0, this.pasos().findIndex(p => p.actual)));
  /** Se activa tras el primer render: el camión no se anima al aparecer, solo cuando el gasto avanza. */
  protected readonly lista = signal(false);
  protected readonly nombres = ESTADO_GASTO_LABEL;
  protected readonly formatFechaHora = formatFechaHora;

  constructor() {
    afterNextRender(() => this.lista.set(true));
  }
}
```

`frontend/src/app/shared/ui/ruta-gasto/ruta-gasto.component.html`:

```html
<ol class="ruta__via" [attr.aria-hidden]="variante() === 'compacta' ? 'true' : null">
  @for (p of pasos(); track p.estado) {
    <li
      class="ruta__hito"
      [attr.data-estado]="p.estado"
      [class.ruta__hito--alcanzado]="p.alcanzado"
      [class.ruta__hito--actual]="p.actual"
      [attr.aria-current]="p.actual ? 'step' : null"
    >
      <span class="ruta__poste" aria-hidden="true"></span>
      @if (variante() === 'completa') {
        <span class="ruta__nombre">{{ nombres[p.estado] }}</span>
        <span class="ruta__dato">{{ p.fecha ? formatFechaHora(p.fecha) : 'Pendiente' }}</span>
        @if (p.actor) {
          <span class="ruta__dato">{{ p.actor }}</span>
        }
      }
    </li>
  }
</ol>
<span class="ruta__camion" aria-hidden="true"><app-camion /></span>
@if (variante() === 'completa') {
  @if (rechazo(); as r) {
    <p class="ruta__desvio">
      <span class="ruta__desvio-marca" aria-hidden="true"></span>
      <span>Comprobante rechazado el {{ formatFechaHora(r.aprobado_en ?? r.creado_en) }}{{ r.motivo_rechazo ? ': ' + r.motivo_rechazo : '.' }}</span>
    </p>
  }
}
```

- [ ] **Step 4: Crear `src/styles/ruta.css` e importarlo**

`frontend/src/styles/ruta.css`:

```css
/* ══ La ruta del gasto (firma) ══ */
.ruta {
  --poste: 10px;
  --camion: 18px;
  --camion-pad: 3px;
  --centro: 9px;
  position: relative;
  display: block;
}

.ruta--compacta {
  flex-shrink: 0;
  width: 88px;
}

.ruta--compacta .ruta__via {
  height: 18px;
}

.ruta--completa {
  --poste: 18px;
  --camion: 32px;
  --camion-pad: 6px;
  --centro: 16px;
}

.ruta__via {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

/* La carretera va del centro del primer hito al centro del último */
.ruta__via::before,
.ruta__via::after {
  content: '';
  position: absolute;
  top: calc(var(--centro) - 1px);
  left: calc(100% / 6);
  width: calc(100% * 2 / 3);
}

.ruta__via::before {
  border-top: 2px dashed var(--linea-2);
}

.ruta__via::after {
  border-top: 2px solid var(--tinta-2);
  transform: scaleX(calc(var(--actual, 0) / 2));
  transform-origin: left center;
}

.ruta__hito {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  min-width: 0;
  text-align: center;
}

.ruta__hito[data-estado='Registrado'] {
  --c: var(--registrado);
}

.ruta__hito[data-estado='Pagado'] {
  --c: var(--pagado);
}

.ruta__hito[data-estado='Sustentado'] {
  --c: var(--sustentado);
}

.ruta__poste {
  width: var(--poste);
  height: var(--poste);
  margin-top: calc(var(--centro) - var(--poste) / 2);
  border: 2px solid var(--c);
  border-radius: 2px;
  background: var(--panel);
}

.ruta__hito--alcanzado .ruta__poste {
  background: var(--c);
}

.ruta__nombre {
  margin-top: var(--e-3);
  font-size: var(--t-14);
  font-weight: 600;
}

.ruta__hito:not(.ruta__hito--alcanzado) .ruta__nombre {
  color: var(--tinta-3);
}

.ruta__dato {
  color: var(--tinta-2);
  font-size: var(--t-13);
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}

.ruta__camion {
  position: absolute;
  z-index: 2;
  top: calc(var(--centro) - var(--camion) / 2);
  left: 0;
  display: flex;
  justify-content: center;
  width: calc(100% / 3);
  transform: translateX(calc(var(--actual, 0) * 100%));
  pointer-events: none;
}

.ruta__camion .camion {
  width: var(--camion);
  height: var(--camion);
  padding: var(--camion-pad);
  border-radius: 4px;
  background: var(--rojo);
  color: #fff;
  box-shadow: 0 0 0 2px var(--panel);
}

.ruta__desvio {
  display: flex;
  align-items: flex-start;
  gap: var(--e-2);
  margin-top: var(--e-5);
  padding: var(--e-3) var(--e-4);
  border-radius: var(--r-control);
  background: var(--rojo-suave);
  color: var(--rojo-tinta);
  font-size: var(--t-14);
}

.ruta__desvio-marca {
  flex-shrink: 0;
  width: 10px;
  height: 10px;
  margin-top: 5px;
  border-radius: 2px;
  background: var(--rojo);
}

/* El camión se desliza al hito nuevo cuando el gasto avanza en la misma vista */
@media (prefers-reduced-motion: no-preference) {
  .ruta--lista .ruta__camion,
  .ruta--lista .ruta__via::after {
    transition: transform var(--d-ruta) var(--ease-in-out);
  }
}

/* Celular: la versión completa se vuelve vertical */
@media (max-width: 767px) {
  .ruta--completa {
    --alto: 72px;
  }

  .ruta--completa .ruta__via {
    grid-template-columns: 1fr;
    grid-auto-rows: var(--alto);
  }

  .ruta--completa .ruta__via::before,
  .ruta--completa .ruta__via::after {
    top: calc(var(--alto) / 2);
    left: 15px;
    width: 0;
    height: calc(var(--alto) * 2);
    border-top: 0;
  }

  .ruta--completa .ruta__via::before {
    border-left: 2px dashed var(--linea-2);
  }

  .ruta--completa .ruta__via::after {
    border-left: 2px solid var(--tinta-2);
    transform: scaleY(calc(var(--actual, 0) / 2));
    transform-origin: center top;
  }

  .ruta--completa .ruta__hito {
    display: grid;
    grid-template-columns: 32px 1fr;
    grid-auto-rows: min-content;
    align-content: center;
    justify-items: start;
    column-gap: var(--e-3);
    text-align: left;
  }

  .ruta--completa .ruta__poste {
    grid-row: 1 / span 3;
    align-self: center;
    justify-self: center;
    margin: 0;
  }

  .ruta--completa .ruta__nombre {
    margin-top: 0;
  }

  .ruta--completa .ruta__nombre,
  .ruta--completa .ruta__dato {
    grid-column: 2;
  }

  .ruta--completa .ruta__camion {
    top: calc(var(--alto) / 2 - var(--camion) / 2);
    width: 32px;
    transform: translateY(calc(var(--actual, 0) * var(--alto)));
  }
}
```

En `frontend/src/styles.css`, agrega al final:

```css
@import './styles/ruta.css';
```

- [ ] **Step 5: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/shared/ui/ruta-gasto/ruta-gasto.component.spec.ts`
Expected: PASS (3 pruebas).

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso.

---
### Task 10: Selector de registros relacionados (combobox con búsqueda)

**Files:**
- Create: `frontend/src/app/shared/ui/selector-referencia/selector-referencia.component.ts`
- Create: `frontend/src/app/shared/ui/selector-referencia/selector-referencia.component.html`
- Test: `frontend/src/app/shared/ui/selector-referencia/selector-referencia.component.spec.ts`

**Interfaces:**
- Consumes:
  - `OpcionRef` (Tarea 5)
  - `normalizar` (Tarea 1)
  - `IconoComponent` (Tarea 2)
  - las clases `.combo*` y `.control` (Tarea 7)
- Produces:
  - `SelectorReferenciaComponent`, selector `app-selector-referencia`:
    - inputs: `idControl` (requerido), `opciones: OpcionRef[]` (requerido), `invalido = false`, `describedBy: string | null = null` y `cargando = false`
    - textos: `placeholder = 'Escribe para buscar'`, `vacioTexto = 'Aún no hay registros.'`, `crearRuta: string | null = null` y `crearTexto = 'Crear registro'`
    - two-way: `valor = model('')`
    - output: `creando` (cuando el usuario sigue el enlace para crear el registro que falta)
  - `coincidencias(opciones, termino): OpcionRef[]`
  - `MAX_OPCIONES = 50`

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/shared/ui/selector-referencia/selector-referencia.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { MAX_OPCIONES, SelectorReferenciaComponent, coincidencias } from './selector-referencia.component';
import { OpcionRef } from '../../../core/models/crud.model';

const opciones: OpcionRef[] = [
  { valor: '1', etiqueta: 'Agroindustria' },
  { valor: '2', etiqueta: 'Minería' },
  { valor: '3', etiqueta: 'Logística' },
];

describe('coincidencias', () => {
  it('filtra sin distinguir tildes ni mayúsculas', () => {
    expect(coincidencias(opciones, 'MINERIA').map(o => o.valor)).toEqual(['2']);
    expect(coincidencias(opciones, '  ')).toHaveLength(3);
  });
});

@Component({
  imports: [SelectorReferenciaComponent],
  template: `<app-selector-referencia idControl="rubro" [opciones]="opciones" [(valor)]="valor" />`,
})
class Anfitrion {
  opciones: OpcionRef[] = opciones;
  valor = signal('');
}

function montar() {
  TestBed.configureTestingModule({ providers: [provideRouter([])] });
  const f = TestBed.createComponent(Anfitrion);
  f.detectChanges();
  const input: HTMLInputElement = f.nativeElement.querySelector('input');
  return { f, input };
}

describe('SelectorReferenciaComponent', () => {
  it('se elige con teclado: escribir y Enter', () => {
    const { f, input } = montar();
    input.dispatchEvent(new FocusEvent('focus'));
    input.value = 'log';
    input.dispatchEvent(new Event('input'));
    f.detectChanges();
    expect(f.nativeElement.querySelectorAll('[role="option"]')).toHaveLength(1);
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    f.detectChanges();
    expect(f.componentInstance.valor()).toBe('3');
    expect(input.value).toBe('Logística');
    expect(input.getAttribute('aria-expanded')).toBe('false');
  });

  it('con muchas opciones muestra 50 y avisa cuántas hay', () => {
    const { f, input } = montar();
    f.componentInstance.opciones = Array.from({ length: 80 }, (_, i) => ({ valor: String(i), etiqueta: `Distrito ${i}` }));
    f.detectChanges();
    input.dispatchEvent(new FocusEvent('focus'));
    f.detectChanges();
    expect(f.nativeElement.querySelectorAll('[role="option"]')).toHaveLength(MAX_OPCIONES);
    expect(f.nativeElement.textContent).toContain('Mostrando 50 de 80');
  });

  it('sin registros ofrece el enlace para crearlos', () => {
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
    const f = TestBed.createComponent(SelectorReferenciaComponent);
    f.componentRef.setInput('idControl', 'rubro');
    f.componentRef.setInput('opciones', []);
    f.componentRef.setInput('vacioTexto', 'Aún no hay rubros.');
    f.componentRef.setInput('crearRuta', '/app/catalogos/rubros');
    f.componentRef.setInput('crearTexto', 'Nuevo rubro');
    f.detectChanges();
    f.nativeElement.querySelector('input').dispatchEvent(new FocusEvent('focus'));
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('Aún no hay rubros.');
    expect(f.nativeElement.querySelector('a')?.getAttribute('href')).toBe('/app/catalogos/rubros');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/shared/ui/selector-referencia/selector-referencia.component.spec.ts`
Expected: FAIL. No se puede resolver `./selector-referencia.component`.

- [ ] **Step 3: Implementar**

`frontend/src/app/shared/ui/selector-referencia/selector-referencia.component.ts`:

```ts
import {
  ChangeDetectionStrategy, Component, afterRenderEffect, computed, input, model, output, signal,
} from '@angular/core';
import { RouterLink } from '@angular/router';
import { OpcionRef } from '../../../core/models/crud.model';
import { normalizar } from '../../../core/utils/texto';
import { IconoComponent } from '../icono/icono.component';

export const MAX_OPCIONES = 50;

export function coincidencias(opciones: OpcionRef[], termino: string): OpcionRef[] {
  const t = normalizar(termino).trim();
  return t ? opciones.filter(o => normalizar(o.etiqueta).includes(t)) : opciones;
}

/**
 * Elige un registro de otro módulo por su nombre. La lista se despliega dentro del flujo
 * (no flota), así no se recorta dentro de hojas con scroll. Teclado: ↑ ↓ Enter Esc.
 */
@Component({
  selector: 'app-selector-referencia',
  imports: [RouterLink, IconoComponent],
  templateUrl: './selector-referencia.component.html',
  host: { class: 'combo', '[class.combo--abierto]': 'abierto()' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SelectorReferenciaComponent {
  readonly idControl = input.required<string>();
  readonly opciones = input.required<OpcionRef[]>();
  readonly valor = model('');
  readonly invalido = input(false);
  readonly describedBy = input<string | null>(null);
  readonly cargando = input(false);
  readonly placeholder = input('Escribe para buscar');
  readonly vacioTexto = input('Aún no hay registros.');
  readonly crearRuta = input<string | null>(null);
  readonly crearTexto = input('Crear registro');
  readonly creando = output<void>();

  protected readonly abierto = signal(false);
  protected readonly termino = signal('');
  protected readonly activo = signal(0);
  protected readonly todas = computed(() => coincidencias(this.opciones(), this.termino()));
  protected readonly visibles = computed(() => this.todas().slice(0, MAX_OPCIONES));
  protected readonly seleccionada = computed(() => this.opciones().find(o => o.valor === this.valor()));
  protected readonly listaId = computed(() => `${this.idControl()}-lista`);
  protected readonly texto = computed(() => (this.abierto() ? this.termino() : this.seleccionada()?.etiqueta ?? ''));
  protected readonly activoId = computed(() =>
    this.abierto() && this.visibles().length ? `${this.listaId()}-${this.activo()}` : null,
  );

  constructor() {
    afterRenderEffect(() => {
      const id = this.activoId();
      if (id) document.getElementById(id)?.scrollIntoView?.({ block: 'nearest' });
    });
  }

  protected abrir(): void {
    if (this.abierto()) return;
    this.termino.set('');
    this.activo.set(Math.max(0, this.visibles().findIndex(o => o.valor === this.valor())));
    this.abierto.set(true);
  }

  protected cerrar(): void {
    this.abierto.set(false);
    this.termino.set('');
  }

  protected escribir(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
    this.activo.set(0);
    this.abierto.set(true);
  }

  protected elegir(opcion: OpcionRef): void {
    this.valor.set(opcion.valor);
    this.cerrar();
  }

  protected teclear(evento: KeyboardEvent): void {
    const total = this.visibles().length;
    switch (evento.key) {
      case 'ArrowDown':
        evento.preventDefault();
        if (!this.abierto()) {
          this.abrir();
          return;
        }
        this.activo.set(Math.min(this.activo() + 1, Math.max(total - 1, 0)));
        break;
      case 'ArrowUp':
        evento.preventDefault();
        this.activo.set(Math.max(this.activo() - 1, 0));
        break;
      case 'Enter':
        if (this.abierto() && total) {
          evento.preventDefault();
          this.elegir(this.visibles()[this.activo()]);
        }
        break;
      case 'Escape':
        if (this.abierto()) {
          evento.preventDefault();
          evento.stopPropagation();
          this.cerrar();
        }
        break;
    }
  }
}
```

`frontend/src/app/shared/ui/selector-referencia/selector-referencia.component.html`:

```html
<div class="combo__control">
  <input
    class="control combo__input"
    type="text"
    role="combobox"
    autocomplete="off"
    spellcheck="false"
    aria-autocomplete="list"
    [id]="idControl()"
    [value]="texto()"
    [placeholder]="placeholder()"
    [attr.aria-expanded]="abierto()"
    [attr.aria-controls]="listaId()"
    [attr.aria-activedescendant]="activoId()"
    [attr.aria-invalid]="invalido() || null"
    [attr.aria-describedby]="describedBy()"
    (focus)="abrir()"
    (click)="abrir()"
    (input)="escribir($event)"
    (keydown)="teclear($event)"
    (blur)="cerrar()"
  />
  <app-icono class="combo__chevron" nombre="chevron-down" [tamano]="16" />
</div>
@if (abierto()) {
  <ul class="combo__lista" role="listbox" [id]="listaId()" (pointerdown)="$event.preventDefault()">
    @for (o of visibles(); track o.valor; let i = $index) {
      <li
        class="combo__opcion"
        role="option"
        [id]="listaId() + '-' + i"
        [attr.aria-selected]="o.valor === valor()"
        [class.combo__opcion--activa]="i === activo()"
        (click)="elegir(o)"
      >
        <span>{{ o.etiqueta }}</span>
        @if (o.valor === valor()) {
          <app-icono nombre="check" [tamano]="16" />
        }
      </li>
    } @empty {
      <li class="combo__nota" role="presentation">
        @if (cargando()) {
          Cargando…
        } @else if (opciones().length === 0) {
          {{ vacioTexto() }}
          @if (crearRuta(); as ruta) {
            <a class="enlace" [routerLink]="ruta" (click)="creando.emit()">{{ crearTexto() }}</a>
          }
        } @else {
          Sin coincidencias.
        }
      </li>
    }
    @if (todas().length > visibles().length) {
      <li class="combo__nota" role="presentation">Mostrando {{ visibles().length }} de {{ todas().length }}. Escribe para filtrar.</li>
    }
  </ul>
}
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/shared/ui/selector-referencia/selector-referencia.component.spec.ts`
Expected: PASS (4 pruebas).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false`
Expected: PASS en todas las pruebas.

---
### Task 11: Avisos (toasts) en la raíz de la app

**Files:**
- Modify (reemplazo completo): `frontend/src/app/shared/components/toast/toast.component.ts`
- Modify (reemplazo completo): `frontend/src/app/app.ts`
- Modify (reemplazo completo): `frontend/src/app/app.html`
- Modify: `frontend/src/app/features/dashboard/dashboard.component.ts` y `dashboard.component.html` (quitar el `<app-toast>` duplicado; el dashboard se borra en la Tarea 20)
- Test: `frontend/src/app/shared/components/toast/toast.component.spec.ts`

**Interfaces:**
- Consumes:
  - `ToastService` (sin cambios): `toasts()`, `show()` y `remove()`
  - `IconoComponent` e `IconoNombre` (Tarea 2)
  - las clases `.avisos` y `.aviso*` (Tarea 7)
- Produces: `ToastComponent`, selector `app-toast`, montado una sola vez en `app.html`.

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/shared/components/toast/toast.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ToastComponent } from './toast.component';
import { ToastService } from '../../../core/services/toast.service';

describe('ToastComponent', () => {
  it('anuncia el aviso con su tipo y permite cerrarlo', () => {
    const f = TestBed.createComponent(ToastComponent);
    const avisos = TestBed.inject(ToastService);
    avisos.show('Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.', '', 'ok', 0);
    f.detectChanges();
    const region: HTMLElement = f.nativeElement.querySelector('.avisos');
    expect(region.getAttribute('aria-live')).toBe('polite');
    expect(region.querySelector('.aviso--ok .aviso__titulo')?.textContent).toBe(
      'Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.',
    );
    (region.querySelector('.aviso__cerrar') as HTMLButtonElement).click();
    expect(avisos.toasts()).toHaveLength(0);
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/shared/components/toast/toast.component.spec.ts`
Expected: FAIL. No existe la región `.avisos`; el componente viejo usa `.toast-area`.

- [ ] **Step 3: Implementar**

Reemplaza `frontend/src/app/shared/components/toast/toast.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ToastService } from '../../../core/services/toast.service';
import { ToastType } from '../../../core/models/crud.model';
import { IconoComponent } from '../../ui/icono/icono.component';
import { IconoNombre } from '../../ui/icono/iconos';

/** Avisos transitorios. Entran con @starting-style y salen con animate.leave (más rápido que la entrada). */
@Component({
  selector: 'app-toast',
  imports: [IconoComponent],
  template: `
    <section class="avisos" aria-live="polite" aria-label="Avisos">
      @for (t of toast.toasts(); track t.id) {
        <div [class]="'aviso aviso--' + t.type" role="status" animate.leave="aviso--saliendo">
          <app-icono class="aviso__icono" [nombre]="iconos[t.type]" [tamano]="18" />
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
    ok: 'circle-check',
    err: 'alert-triangle',
    info: 'info-circle',
    warn: 'alert-triangle',
  };
}
```

Reemplaza `frontend/src/app/app.ts`:

```ts
import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ToastComponent } from './shared/components/toast/toast.component';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, ToastComponent],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App {}
```

Reemplaza `frontend/src/app/app.html`:

```html
<router-outlet />
<app-toast />
```

En `frontend/src/app/features/dashboard/dashboard.component.html` borra la línea `  <app-toast></app-toast>`. En `frontend/src/app/features/dashboard/dashboard.component.ts`, borra la línea `import { ToastComponent } from '../../shared/components/toast/toast.component';` y quita `ToastComponent, ` del arreglo `imports`.

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/shared/components/toast/toast.component.spec.ts`
Expected: PASS (1 prueba).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso. Un solo `app-toast` en toda la app (`grep -rn "<app-toast" src/app` devuelve solo `app.html`).

---
### Task 12: Fila de gasto y bandeja de gastos (`/app/gastos`)

**Files:**
- Create: `frontend/src/app/features/gastos/fila-gasto/fila-gasto.component.ts`
- Create: `frontend/src/app/features/gastos/fila-gasto/fila-gasto.component.html`
- Create: `frontend/src/app/features/gastos/gastos-lista/gastos-lista.component.ts`
- Create: `frontend/src/app/features/gastos/gastos-lista/gastos-lista.component.html`
- Create: `frontend/src/styles/gastos.css`
- Modify: `frontend/src/styles.css` (agregar import)
- Test: `frontend/src/app/features/gastos/gastos-lista/gastos-lista.component.spec.ts`

**Interfaces:**
- Consumes:
  - de `GastosStore`: `gastos`, `comprobantes`, `cargando`, `cargado`, `error`, `conteo`, `asegurarCargado()` y `cargar()`
  - de `CrudService`: `etiqueta(clave, id)` y `asegurarCargado(clave)`
  - de la lógica (Tarea 3): `Pestana`, `esPestana`, `filtrarPorPestana`, `ordenarRecientes`, `buscarGastos`, `totalMonto`, `pasosRuta`, `siguientePaso` y `ultimoRechazo`
  - componentes: `RutaGastoComponent`, `InsigniaEstadoComponent`, `IconoComponent` y `EstadoVacioComponent`
- Produces:
  - `FilaGastoComponent`, selector `li[app-fila-gasto]`, con inputs `gasto: Gasto` (requerido), `origen: Pestana = 'todos'` y `atenuar = false`. Toda la fila enlaza a `/app/gastos/:id` con `state: { origen }`, y la acción usa `fragment: 'paso'`.
  - `GastosListaComponent`, con input `estado` (`?estado=` de la URL vía `withComponentInputBinding`).

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/features/gastos/gastos-lista/gastos-lista.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { computed, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { GastosListaComponent } from './gastos-lista.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { CrudService } from '../../../core/services/crud.service';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { contarPorEstado } from '../../../core/state/gastos.logic';

function gasto(id: number, estado: Gasto['estado'], monto: number): Gasto {
  return {
    id, estado, monto, viaje_id: 1, cliente_id: id, fecha: '2026-10-01T00:00:00Z',
    creado_por: 'Ana', creado_en: `2026-10-0${id}T12:00:00Z`,
  };
}

function storeFalso(gastos: Gasto[], estado: { error?: boolean; cargado?: boolean } = {}) {
  const lista = signal(gastos);
  return {
    gastos: lista,
    comprobantes: signal<Comprobante[]>([]),
    cargando: signal(false),
    cargado: signal(estado.cargado ?? true),
    error: signal(estado.error ?? false),
    conteo: computed(() => contarPorEstado(lista())),
    pendientes: signal<Comprobante[]>([]),
    asegurarCargado: vi.fn(),
    cargar: vi.fn(),
  };
}

const crudFalso = {
  etiqueta: (clave: string, id: unknown) => (clave === 'cliente' ? `Cliente ${id}` : `Viaje N° ${id}`),
  asegurarCargado: vi.fn(),
};

function montar(store: ReturnType<typeof storeFalso>, estado?: string) {
  TestBed.configureTestingModule({
    providers: [provideRouter([]), { provide: GastosStore, useValue: store }, { provide: CrudService, useValue: crudFalso }],
  });
  const f = TestBed.createComponent(GastosListaComponent);
  if (estado) f.componentRef.setInput('estado', estado);
  f.detectChanges();
  return f;
}

describe('GastosListaComponent', () => {
  const gastos = [gasto(1, 'Registrado', 120), gasto(2, 'Pagado', 85.5), gasto(3, 'Sustentado', 310)];

  it('la pestaña Por aprobar muestra solo gastos pagados, con Revisar y el total', () => {
    const el: HTMLElement = montar(storeFalso(gastos), 'por-aprobar').nativeElement;
    const filas = el.querySelectorAll('li.fila-gasto');
    expect(filas).toHaveLength(1);
    expect(filas[0].textContent).toContain('N° 2');
    expect(filas[0].querySelector('.fila-gasto__accion')?.textContent).toContain('Revisar');
    expect(el.querySelector('.lista-gastos__total')?.textContent).toContain('S/ 85.50');
  });

  it('cada pestaña muestra su conteo', () => {
    const el: HTMLElement = montar(storeFalso(gastos)).nativeElement;
    const textos = [...el.querySelectorAll('.pestanas__item')].map(a => a.textContent?.replace(/\s+/g, ' ').trim());
    expect(textos).toEqual(['Todos 3', 'Sin comprobante 1', 'Por aprobar 1', 'Sustentados 1']);
  });

  it('busca por número de gasto', () => {
    const f = montar(storeFalso(gastos));
    const input: HTMLInputElement = f.nativeElement.querySelector('input[type="search"]');
    input.value = '3';
    input.dispatchEvent(new Event('input'));
    f.detectChanges();
    const filas = f.nativeElement.querySelectorAll('li.fila-gasto');
    expect(filas).toHaveLength(1);
    expect(filas[0].textContent).toContain('N° 3');
  });

  it('si el servicio de gastos falla muestra el error con Reintentar, no el vacío', () => {
    const store = storeFalso([], { error: true, cargado: false });
    const el: HTMLElement = montar(store).nativeElement;
    expect(el.textContent).toContain('No se pudieron cargar los gastos.');
    expect(el.textContent).not.toContain('Todavía no hay gastos registrados.');
    (el.querySelector('.vacio button') as HTMLButtonElement).click();
    expect(store.cargar).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/features/gastos/gastos-lista/gastos-lista.component.spec.ts`
Expected: FAIL. No se puede resolver `./gastos-lista.component`.

- [ ] **Step 3: Implementar la fila de gasto**

`frontend/src/app/features/gastos/fila-gasto/fila-gasto.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Gasto } from '../../../core/models/gasto.model';
import { CrudService } from '../../../core/services/crud.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { Pestana, pasosRuta, siguientePaso, ultimoRechazo } from '../../../core/state/gastos.logic';
import { formatFecha, formatMonto } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';
import { InsigniaEstadoComponent } from '../../../shared/ui/insignia-estado/insignia-estado.component';
import { RutaGastoComponent } from '../../../shared/ui/ruta-gasto/ruta-gasto.component';

/** Una fila de la bandeja: toda la fila abre el gasto y la acción lleva directo al paso que toca. */
@Component({
  selector: 'li[app-fila-gasto]',
  imports: [RouterLink, IconoComponent, InsigniaEstadoComponent, RutaGastoComponent],
  templateUrl: './fila-gasto.component.html',
  host: { class: 'fila-gasto', '[class.fila-gasto--cerrada]': "atenuar() && gasto().estado === 'Sustentado'" },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FilaGastoComponent {
  readonly gasto = input.required<Gasto>();
  readonly origen = input<Pestana>('todos');
  readonly atenuar = input(false);

  private readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);

  protected readonly pasos = computed(() => pasosRuta(this.gasto(), this.store.comprobantes()));
  protected readonly paso = computed(() => siguientePaso(this.gasto()));
  protected readonly rechazo = computed(() =>
    this.gasto().estado === 'Registrado' ? ultimoRechazo(this.store.comprobantes(), this.gasto().id) : undefined,
  );
  protected readonly cliente = computed(() => this.crud.etiqueta('cliente', this.gasto().cliente_id));
  protected readonly viaje = computed(() => this.crud.etiqueta('viaje', this.gasto().viaje_id));
  protected readonly enlace = computed(() => ['/app/gastos', this.gasto().id]);
  protected readonly estadoNavegacion = computed(() => ({ origen: this.origen() }));
  protected readonly formatMonto = formatMonto;
  protected readonly formatFecha = formatFecha;
}
```

`frontend/src/app/features/gastos/fila-gasto/fila-gasto.component.html`:

```html
<a class="fila-gasto__numero" [routerLink]="enlace()" [state]="estadoNavegacion()">N° {{ gasto().id }}</a>
<span class="fila-gasto__cliente">
  <span class="fila-gasto__nombre">{{ cliente() }}</span>
  @if (rechazo(); as r) {
    <span class="fila-gasto__rechazo">
      <app-icono nombre="alert-triangle" [tamano]="14" />
      <span>Comprobante rechazado: {{ r.motivo_rechazo }}</span>
    </span>
  }
</span>
<span class="fila-gasto__viaje">{{ viaje() }}</span>
<span class="fila-gasto__fecha">{{ formatFecha(gasto().fecha) }}</span>
<span class="fila-gasto__monto">{{ formatMonto(gasto().monto) }}</span>
<span class="fila-gasto__estado">
  <app-ruta-gasto [pasos]="pasos()" variante="compacta" />
  <app-insignia-estado [estado]="gasto().estado" />
</span>
<span class="fila-gasto__accion">
  @switch (paso()) {
    @case ('adjuntar') {
      <a class="btn btn--secundario btn--sm" [routerLink]="enlace()" [state]="estadoNavegacion()" fragment="paso">
        <app-icono nombre="paperclip" [tamano]="16" />Adjuntar comprobante
      </a>
    }
    @case ('revisar') {
      <a class="btn btn--primario btn--sm" [routerLink]="enlace()" [state]="estadoNavegacion()" fragment="paso">
        <app-icono nombre="clipboard-check" [tamano]="16" />Revisar
      </a>
    }
    @default {
      <a class="btn btn--fantasma btn--sm" [routerLink]="enlace()" [state]="estadoNavegacion()">Ver</a>
    }
  }
</span>
```

- [ ] **Step 4: Implementar la bandeja**

`frontend/src/app/features/gastos/gastos-lista/gastos-lista.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, OnInit, computed, inject, input, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { CrudService } from '../../../core/services/crud.service';
import { GastosStore } from '../../../core/state/gastos.store';
import {
  Pestana, buscarGastos, esPestana, filtrarPorPestana, ordenarRecientes, totalMonto,
} from '../../../core/state/gastos.logic';
import { formatMonto } from '../../../core/utils/format';
import { EstadoVacioComponent } from '../../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';
import { FilaGastoComponent } from '../fila-gasto/fila-gasto.component';

const VACIOS: Record<Pestana, { titulo: string; texto: string }> = {
  todos: { titulo: 'Todavía no hay gastos registrados.', texto: 'Presiona "Registrar gasto" para crear el primero.' },
  'sin-comprobante': { titulo: 'No hay gastos sin comprobante.', texto: 'Todos los gastos registrados ya tienen su comprobante.' },
  'por-aprobar': { titulo: 'No hay comprobantes pendientes de aprobación.', texto: 'Cuando se adjunte un comprobante aparecerá aquí para revisarlo.' },
  sustentados: { titulo: 'Todavía no hay gastos sustentados.', texto: 'Un gasto queda sustentado cuando se aprueba su comprobante.' },
};

@Component({
  selector: 'app-gastos-lista',
  imports: [RouterLink, IconoComponent, EstadoVacioComponent, FilaGastoComponent],
  templateUrl: './gastos-lista.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GastosListaComponent implements OnInit {
  /** ?estado= de la URL */
  readonly estado = input<string | undefined>(undefined);

  protected readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);

  protected readonly termino = signal('');
  protected readonly pestanas: { id: Pestana; etiqueta: string }[] = [
    { id: 'todos', etiqueta: 'Todos' },
    { id: 'sin-comprobante', etiqueta: 'Sin comprobante' },
    { id: 'por-aprobar', etiqueta: 'Por aprobar' },
    { id: 'sustentados', etiqueta: 'Sustentados' },
  ];
  protected readonly pestana = computed<Pestana>(() => {
    const e = this.estado();
    return esPestana(e) ? e : 'todos';
  });
  protected readonly conteos = computed<Record<Pestana, number>>(() => {
    const c = this.store.conteo();
    return { todos: this.store.gastos().length, 'sin-comprobante': c.Registrado, 'por-aprobar': c.Pagado, sustentados: c.Sustentado };
  });
  protected readonly visibles = computed(() =>
    buscarGastos(
      filtrarPorPestana(ordenarRecientes(this.store.gastos()), this.pestana()),
      this.termino(),
      id => this.crud.etiqueta('cliente', id),
    ),
  );
  protected readonly total = computed(() => totalMonto(this.visibles()));
  protected readonly vacio = computed(() => VACIOS[this.pestana()]);
  protected readonly formatMonto = formatMonto;

  ngOnInit(): void {
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    this.crud.asegurarCargado('viaje');
  }

  protected queryDe(pestana: Pestana): Record<string, string | null> {
    return { estado: pestana === 'todos' ? null : pestana };
  }

  protected buscar(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
  }
}
```

`frontend/src/app/features/gastos/gastos-lista/gastos-lista.component.html`:

```html
<div class="pagina">
  <header class="pagina__encabezado">
    <h1 class="pagina__titulo">Gastos</h1>
    <div class="pagina__acciones">
      <a class="btn btn--primario" routerLink="/app/gastos/nuevo"><app-icono nombre="plus" />Registrar gasto</a>
    </div>
  </header>

  <nav class="pestanas" aria-label="Filtrar gastos por estado">
    @for (p of pestanas; track p.id) {
      <a
        class="pestanas__item"
        routerLink="/app/gastos"
        [queryParams]="queryDe(p.id)"
        [class.pestanas__item--activa]="pestana() === p.id"
        [attr.aria-current]="pestana() === p.id ? 'page' : null"
      >
        {{ p.etiqueta }}
        <span class="contador" [class.contador--via]="p.id === 'por-aprobar' && conteos()[p.id] > 0">{{ conteos()[p.id] }}</span>
      </a>
    }
  </nav>

  <div class="barra-herramientas">
    <label class="buscador">
      <app-icono nombre="search" [tamano]="16" />
      <span class="oculto-visual">Buscar gastos</span>
      <input class="control buscador__input" type="search" placeholder="Buscar por N° o cliente" [value]="termino()" (input)="buscar($event)" />
    </label>
    <button type="button" class="btn btn--fantasma btn--sm" (click)="store.cargar()" [disabled]="store.cargando()">
      <app-icono nombre="refresh" [tamano]="16" />Actualizar
    </button>
  </div>

  @if (store.cargando() && !store.cargado()) {
    <div class="lista-gastos" aria-busy="true">
      @for (i of [1, 2, 3, 4, 5]; track i) {
        <div class="esqueleto esqueleto--fila"></div>
      }
    </div>
  } @else if (store.error() && !store.cargado()) {
    <app-estado-vacio icono="cloud-off" titulo="No se pudieron cargar los gastos." texto="Revisa tu conexión e inténtalo de nuevo.">
      <button type="button" class="btn btn--secundario" (click)="store.cargar()">Reintentar</button>
    </app-estado-vacio>
  } @else if (visibles().length === 0) {
    @if (termino()) {
      <app-estado-vacio icono="search" titulo="Ningún gasto coincide con la búsqueda." [texto]="'Probaste con «' + termino() + '».'" />
    } @else {
      <app-estado-vacio icono="inbox" [titulo]="vacio().titulo" [texto]="vacio().texto">
        @if (pestana() === 'todos') {
          <a class="btn btn--primario" routerLink="/app/gastos/nuevo">Registrar gasto</a>
        }
      </app-estado-vacio>
    }
  } @else {
    <div class="lista-gastos__cabecera" aria-hidden="true">
      <span>N°</span>
      <span>Cliente</span>
      <span class="lista-gastos__cabecera-viaje">Viaje</span>
      <span>Fecha</span>
      <span class="a-la-derecha">Monto</span>
      <span>Estado</span>
      <span></span>
    </div>
    <ul class="lista-gastos">
      @for (g of visibles(); track g.id) {
        <li app-fila-gasto [gasto]="g" [origen]="pestana()" [atenuar]="pestana() === 'todos'"></li>
      }
    </ul>
    <p class="lista-gastos__total">
      <span class="lista-gastos__total-etiqueta">Total</span>
      <span class="lista-gastos__total-valor">{{ formatMonto(total()) }}</span>
    </p>
  }
</div>
```

- [ ] **Step 5: Crear `src/styles/gastos.css` (bandeja) e importarlo**

`frontend/src/styles/gastos.css`:

```css
/* ══ Gastos: bandeja ══ */
.lista-gastos,
.lista-gastos__cabecera,
.lista-gastos__total {
  --columnas: 84px minmax(160px, 1.5fr) minmax(140px, 1fr) 104px 128px 236px 200px;
  --col-monto: 5;
}

.lista-gastos__cabecera {
  display: grid;
  grid-template-columns: var(--columnas);
  gap: var(--e-4);
  padding: 0 var(--e-4) var(--e-2);
  color: var(--tinta-2);
  font-size: var(--t-12);
  font-weight: 600;
}

.lista-gastos {
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--panel);
}

.fila-gasto {
  position: relative;
  display: grid;
  grid-template-columns: var(--columnas);
  align-items: center;
  gap: var(--e-4);
  min-height: 60px;
  padding: var(--e-2) var(--e-4);
  border-bottom: 1px solid var(--linea);
}

.fila-gasto:last-child {
  border-bottom: 0;
}

@media (hover: hover) and (pointer: fine) {
  .fila-gasto:hover {
    background: var(--panel-2);
  }
}

.fila-gasto__numero {
  color: var(--tinta);
  font-family: var(--fuente-mono);
  font-weight: 600;
  text-decoration: none;
}

/* Enlace estirado: toda la fila abre el gasto */
.fila-gasto__numero::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: var(--r-control);
}

.fila-gasto__numero:focus-visible {
  outline: none;
}

.fila-gasto__numero:focus-visible::after {
  outline: 2px solid var(--rojo);
  outline-offset: -2px;
}

.fila-gasto__cliente {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.fila-gasto__nombre,
.fila-gasto__viaje,
.fila-gasto__fecha,
.fila-gasto__rechazo span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fila-gasto__nombre {
  font-weight: 600;
}

.fila-gasto__rechazo {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  color: var(--rojo-tinta);
  font-size: var(--t-12);
}

.fila-gasto__viaje,
.fila-gasto__fecha {
  color: var(--tinta-2);
  font-size: var(--t-14);
}

.fila-gasto__monto {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.fila-gasto__estado {
  display: flex;
  align-items: center;
  gap: var(--e-3);
}

.fila-gasto__accion {
  position: relative;
  z-index: 1;
  display: flex;
  justify-content: flex-end;
}

/* Lo cerrado retrocede para que lo pendiente mande */
.fila-gasto--cerrada .fila-gasto__numero,
.fila-gasto--cerrada .fila-gasto__nombre,
.fila-gasto--cerrada .fila-gasto__monto {
  color: var(--tinta-2);
  font-weight: 500;
}

/* Total alineado bajo la columna Monto, con filete como en un talonario */
.lista-gastos__total {
  display: grid;
  grid-template-columns: var(--columnas);
  gap: var(--e-4);
  padding: var(--e-3) var(--e-4);
  font-weight: 600;
}

.lista-gastos__total-etiqueta {
  grid-column: 1 / var(--col-monto);
  color: var(--tinta-2);
  text-align: right;
}

.lista-gastos__total-valor {
  grid-column: var(--col-monto);
  padding-top: var(--e-2);
  border-top: 2px solid var(--tinta);
  font-variant-numeric: tabular-nums;
  text-align: right;
}

/* Lista compacta de Inicio: sin viaje ni fecha */
.lista-gastos--compacta {
  --columnas: 84px minmax(160px, 1fr) 128px 236px 200px;
}

.lista-gastos--compacta .fila-gasto__viaje,
.lista-gastos--compacta .fila-gasto__fecha {
  display: none;
}

/* Escritorio angosto: se oculta la columna Viaje */
@media (max-width: 1199px) {
  .lista-gastos,
  .lista-gastos__cabecera,
  .lista-gastos__total {
    --columnas: 76px minmax(150px, 1fr) 100px 120px 220px 176px;
    --col-monto: 4;
  }

  .fila-gasto__viaje,
  .lista-gastos__cabecera-viaje {
    display: none;
  }

  .lista-gastos--compacta {
    --columnas: 76px minmax(150px, 1fr) 120px 220px 176px;
  }
}

/* Celular: cada gasto es una tarjeta apilada */
@media (max-width: 767px) {
  .lista-gastos__cabecera {
    display: none;
  }

  .fila-gasto,
  .lista-gastos--compacta .fila-gasto {
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      'numero monto'
      'cliente fecha'
      'estado estado'
      'accion accion';
    gap: var(--e-2) var(--e-3);
    padding: var(--e-4);
  }

  .fila-gasto__numero {
    grid-area: numero;
  }

  .fila-gasto__monto {
    grid-area: monto;
  }

  .fila-gasto__cliente {
    grid-area: cliente;
  }

  .fila-gasto__fecha,
  .lista-gastos--compacta .fila-gasto__fecha {
    display: block;
    grid-area: fecha;
    text-align: right;
  }

  .fila-gasto__estado {
    grid-area: estado;
  }

  .fila-gasto__accion {
    grid-area: accion;
  }

  .fila-gasto__accion .btn {
    width: 100%;
  }

  .lista-gastos__total {
    grid-template-columns: 1fr auto;
  }

  .lista-gastos__total-etiqueta {
    grid-column: 1;
    text-align: left;
  }

  .lista-gastos__total-valor {
    grid-column: 2;
  }
}
```

En `frontend/src/styles.css`, agrega al final:

```css
@import './styles/gastos.css';
```

- [ ] **Step 6: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/features/gastos/gastos-lista/gastos-lista.component.spec.ts`
Expected: PASS (4 pruebas).

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso.

---
### Task 13: Inicio (`/app/inicio`)

**Files:**
- Create: `frontend/src/app/features/inicio/inicio.component.ts`
- Create: `frontend/src/app/features/inicio/inicio.component.html`
- Create: `frontend/src/styles/inicio.css`
- Modify: `frontend/src/styles.css` (agregar import)
- Test: `frontend/src/app/features/inicio/inicio.component.spec.ts`

**Interfaces:**
- Consumes:
  - de `GastosStore`: `gastos`, `comprobantes`, `cargando`, `cargado`, `conteo`, `pendientes` y `asegurarCargado()`
  - de `CrudService`: `asegurarCargado` y `etiqueta`
  - de `AuthService`: `usuario()`
  - de la lógica (Tarea 3): `ORDEN_ESTADOS`, `PESTANA_DE_ESTADO`, `sumaPorEstado` y `recientes`
  - `ESTADO_GASTO_LABEL`
  - componentes: `FilaGastoComponent` (Tarea 12), `EstadoVacioComponent` e `IconoComponent`
- Produces: `InicioComponent`, selector `app-inicio`.

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/features/inicio/inicio.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { computed, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { InicioComponent } from './inicio.component';
import { GastosStore } from '../../core/state/gastos.store';
import { CrudService } from '../../core/services/crud.service';
import { AuthService } from '../../core/services/auth.service';
import { Comprobante, Gasto } from '../../core/models/gasto.model';
import { contarPorEstado } from '../../core/state/gastos.logic';

function gasto(id: number, estado: Gasto['estado'], monto: number): Gasto {
  return { id, estado, monto, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: `2026-10-0${id}T12:00:00Z` };
}

function montar(gastos: Gasto[], pendientes: Comprobante[]) {
  const lista = signal(gastos);
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      {
        provide: GastosStore,
        useValue: {
          gastos: lista, comprobantes: signal(pendientes), cargando: signal(false), cargado: signal(true), error: signal(false),
          conteo: computed(() => contarPorEstado(lista())), pendientes: signal(pendientes), asegurarCargado: vi.fn(),
        },
      },
      { provide: CrudService, useValue: { asegurarCargado: vi.fn(), etiqueta: () => 'Agro Norte SAC' } },
      { provide: AuthService, useValue: { usuario: signal({ nombre: 'Ana Quispe' }) } },
    ],
  });
  const f = TestBed.createComponent(InicioComponent);
  f.detectChanges();
  return f.nativeElement as HTMLElement;
}

describe('InicioComponent', () => {
  it('sin gastos enseña por dónde empezar', () => {
    const el = montar([], []);
    expect(el.textContent).toContain('Todavía no hay gastos registrados.');
    expect(el.querySelector('a[href="/app/operacion/viajes"]')).not.toBeNull();
  });

  it('muestra lo pendiente, el recorrido con montos y los últimos gastos', () => {
    const pendiente: Comprobante = {
      id: 9, gasto_id: 3, archivo_url: '/u/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-03T12:00:00Z',
    };
    const el = montar([gasto(1, 'Registrado', 120), gasto(2, 'Registrado', 30.5), gasto(3, 'Pagado', 85.5)], [pendiente]);
    const porAprobar = el.querySelector('.atender__item--via');
    expect(porAprobar?.textContent).toContain('1');
    expect(porAprobar?.textContent).toContain('comprobante por aprobar');
    const hitos = [...el.querySelectorAll('.recorrido__hito')].map(h => [
      h.querySelector('.recorrido__estado')?.textContent,
      h.querySelector('.recorrido__numero')?.textContent,
      h.querySelector('.recorrido__monto')?.textContent,
    ]);
    expect(hitos).toEqual([
      ['Registrado', '2', 'S/ 150.50'],
      ['Pagado', '1', 'S/ 85.50'],
      ['Sustentado', '0', 'S/ 0.00'],
    ]);
    expect(el.querySelectorAll('li.fila-gasto')).toHaveLength(3);
    expect(el.textContent).toContain('Hola, Ana Quispe.');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/features/inicio/inicio.component.spec.ts`
Expected: FAIL. No se puede resolver `./inicio.component`.

- [ ] **Step 3: Implementar**

`frontend/src/app/features/inicio/inicio.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, OnInit, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { ESTADO_GASTO_LABEL } from '../../core/models/gasto-estados';
import { AuthService } from '../../core/services/auth.service';
import { CrudService } from '../../core/services/crud.service';
import { ORDEN_ESTADOS, PESTANA_DE_ESTADO, recientes, sumaPorEstado } from '../../core/state/gastos.logic';
import { GastosStore } from '../../core/state/gastos.store';
import { formatMonto } from '../../core/utils/format';
import { EstadoVacioComponent } from '../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { FilaGastoComponent } from '../gastos/fila-gasto/fila-gasto.component';

@Component({
  selector: 'app-inicio',
  imports: [RouterLink, IconoComponent, EstadoVacioComponent, FilaGastoComponent],
  templateUrl: './inicio.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class InicioComponent implements OnInit {
  protected readonly store = inject(GastosStore);
  protected readonly auth = inject(AuthService);
  private readonly crud = inject(CrudService);

  protected readonly orden = ORDEN_ESTADOS;
  protected readonly etiqueta = ESTADO_GASTO_LABEL;
  protected readonly pestanaDe = PESTANA_DE_ESTADO;
  protected readonly pendientes = computed(() => this.store.pendientes().length);
  protected readonly sumas = computed(() => sumaPorEstado(this.store.gastos()));
  protected readonly ultimos = computed(() => recientes(this.store.gastos(), 5));
  protected readonly formatMonto = formatMonto;

  ngOnInit(): void {
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    this.crud.asegurarCargado('viaje');
  }
}
```

`frontend/src/app/features/inicio/inicio.component.html`:

```html
<div class="pagina">
  <header class="pagina__encabezado">
    <div>
      <h1 class="pagina__titulo">Inicio</h1>
      @if (auth.usuario(); as u) {
        <p class="pagina__bajada">Hola, {{ u.nombre }}.</p>
      }
    </div>
    <div class="pagina__acciones">
      <a class="btn btn--secundario" routerLink="/app/gastos" [queryParams]="{ estado: 'sin-comprobante' }">
        <app-icono nombre="paperclip" />Adjuntar comprobante
      </a>
      <a class="btn btn--primario" routerLink="/app/gastos/nuevo"><app-icono nombre="plus" />Registrar gasto</a>
    </div>
  </header>

  @if (store.cargando() && !store.cargado()) {
    <div class="inicio__esqueleto" aria-busy="true">
      <div class="esqueleto esqueleto--bloque"></div>
      <div class="esqueleto esqueleto--bloque"></div>
      <div class="esqueleto esqueleto--franja"></div>
    </div>
  } @else if (store.cargado() && store.gastos().length === 0) {
    <app-estado-vacio icono="receipt" titulo="Todavía no hay gastos registrados." texto="Para registrar un gasto necesitas al menos un viaje y un cliente.">
      <a class="btn btn--primario" routerLink="/app/gastos/nuevo">Registrar gasto</a>
      <a class="btn btn--secundario" routerLink="/app/operacion/viajes">Ver viajes</a>
      <a class="btn btn--secundario" routerLink="/app/operacion/clientes">Ver clientes</a>
    </app-estado-vacio>
  } @else {
    <section class="inicio__bloque" aria-labelledby="t-atender">
      <h2 id="t-atender" class="seccion-titulo">Para atender</h2>
      <div class="atender">
        <a class="atender__item" [class.atender__item--via]="pendientes() > 0" routerLink="/app/gastos" [queryParams]="{ estado: 'por-aprobar' }">
          <span class="atender__numero">{{ pendientes() }}</span>
          <span class="atender__texto">{{ pendientes() === 1 ? 'comprobante por aprobar' : 'comprobantes por aprobar' }}</span>
          <app-icono class="atender__flecha" nombre="arrow-right" />
        </a>
        <a class="atender__item" routerLink="/app/gastos" [queryParams]="{ estado: 'sin-comprobante' }">
          <span class="atender__numero">{{ store.conteo().Registrado }}</span>
          <span class="atender__texto">{{ store.conteo().Registrado === 1 ? 'gasto sin comprobante' : 'gastos sin comprobante' }}</span>
          <app-icono class="atender__flecha" nombre="arrow-right" />
        </a>
      </div>
    </section>

    <section class="inicio__bloque" aria-labelledby="t-recorrido">
      <h2 id="t-recorrido" class="seccion-titulo">Recorrido de los gastos</h2>
      <ol class="recorrido">
        @for (e of orden; track e) {
          <li class="recorrido__hito" [attr.data-estado]="e">
            <a class="recorrido__enlace" routerLink="/app/gastos" [queryParams]="{ estado: pestanaDe[e] }">
              <span class="recorrido__poste" aria-hidden="true"></span>
              <span class="recorrido__estado">{{ etiqueta[e] }}</span>
              <span class="recorrido__numero">{{ store.conteo()[e] }}</span>
              <span class="recorrido__monto">{{ formatMonto(sumas()[e]) }}</span>
            </a>
          </li>
        }
      </ol>
    </section>

    <section class="inicio__bloque" aria-labelledby="t-ultimos">
      <div class="seccion-cabecera">
        <h2 id="t-ultimos" class="seccion-titulo">Últimos gastos</h2>
        <a class="enlace" routerLink="/app/gastos">Ver todos</a>
      </div>
      <ul class="lista-gastos lista-gastos--compacta">
        @for (g of ultimos(); track g.id) {
          <li app-fila-gasto [gasto]="g"></li>
        }
      </ul>
    </section>
  }
</div>
```

- [ ] **Step 4: Crear `src/styles/inicio.css` e importarlo**

`frontend/src/styles/inicio.css`:

```css
/* ══ Inicio ══ */
.inicio__bloque {
  margin-bottom: var(--e-8);
}

.inicio__bloque > .seccion-titulo {
  margin-bottom: var(--e-3);
}

.inicio__esqueleto {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--e-4);
}

.inicio__esqueleto .esqueleto--franja {
  grid-column: 1 / -1;
}

/* Para atender: el pendiente de aprobación se pinta como señal preventiva */
.atender {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--e-4);
}

.atender__item {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: var(--e-2) var(--e-4);
  padding: var(--e-5) var(--e-6);
  border: 1px solid var(--linea);
  border-left: 6px solid var(--registrado);
  border-radius: var(--r-panel);
  background: var(--panel);
  color: var(--tinta);
  text-decoration: none;
  transition: border-color 150ms ease, background-color 150ms ease;
}

.atender__item--via {
  border-color: var(--via);
  border-left-color: var(--asfalto);
  background: var(--via);
}

.atender__numero {
  font-size: var(--t-40);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.atender__texto {
  font-size: var(--t-16);
  font-weight: 600;
}

.atender__flecha {
  color: var(--tinta-2);
}

.atender__item--via .atender__flecha {
  color: var(--asfalto);
}

@media (hover: hover) and (pointer: fine) {
  .atender__item:hover {
    border-color: var(--tinta-3);
  }

  .atender__item--via:hover {
    border-color: #e3b600;
    background: #f7cc1c;
  }
}

@media (prefers-reduced-motion: no-preference) {
  .atender__flecha {
    transition: transform var(--d-fast) var(--ease-out);
  }

  .atender__item:hover .atender__flecha {
    transform: translateX(3px);
  }
}

/* Recorrido de los gastos: la carretera en asfalto con un hito por estado */
.recorrido {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  overflow: hidden;
  border-radius: var(--r-panel);
  background: var(--asfalto);
  color: #fff;
}

.recorrido::before {
  content: '';
  position: absolute;
  top: 44px;
  left: var(--e-6);
  right: var(--e-6);
  border-top: 2px dashed rgb(255 255 255 / 0.35);
}

.recorrido__hito + .recorrido__hito .recorrido__enlace {
  border-left: 1px solid var(--asfalto-3);
}

.recorrido__enlace {
  position: relative;
  display: grid;
  gap: 2px;
  height: 100%;
  padding: 34px var(--e-6) var(--e-6);
  color: inherit;
  text-decoration: none;
}

.recorrido__hito[data-estado='Registrado'] {
  --c: var(--registrado);
}

.recorrido__hito[data-estado='Pagado'] {
  --c: var(--pagado);
}

.recorrido__hito[data-estado='Sustentado'] {
  --c: var(--sustentado);
}

.recorrido__poste {
  width: 20px;
  height: 20px;
  margin-bottom: var(--e-3);
  border: 2px solid #fff;
  border-radius: 3px;
  background: var(--c);
}

.recorrido__estado {
  color: var(--asfalto-tinta);
  font-size: var(--t-14);
  font-weight: 600;
}

.recorrido__numero {
  font-size: var(--t-32);
  font-weight: 700;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

.recorrido__monto {
  color: var(--asfalto-tinta);
  font-size: var(--t-14);
  font-variant-numeric: tabular-nums;
}

@media (hover: hover) and (pointer: fine) {
  .recorrido__enlace:hover {
    background: var(--asfalto-2);
  }
}

@media (max-width: 767px) {
  .atender,
  .inicio__esqueleto {
    grid-template-columns: 1fr;
  }

  .atender__item {
    padding: var(--e-4);
  }

  .atender__numero {
    font-size: var(--t-32);
  }

  .recorrido {
    grid-template-columns: 1fr;
  }

  .recorrido::before {
    top: var(--e-6);
    bottom: var(--e-6);
    left: 33px;
    right: auto;
    border-top: 0;
    border-left: 2px dashed rgb(255 255 255 / 0.35);
  }

  .recorrido__hito + .recorrido__hito .recorrido__enlace {
    border-left: 0;
    border-top: 1px solid var(--asfalto-3);
  }

  .recorrido__enlace {
    grid-template-columns: 20px 1fr auto;
    align-items: center;
    column-gap: var(--e-4);
    padding: var(--e-4) var(--e-6);
  }

  .recorrido__poste {
    grid-row: 1 / span 2;
    margin: 0;
  }

  .recorrido__numero {
    grid-column: 3;
    grid-row: 1 / span 2;
    font-size: var(--t-24);
  }
}
```

En `frontend/src/styles.css`, agrega al final:

```css
@import './styles/inicio.css';
```

- [ ] **Step 5: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/features/inicio/inicio.component.spec.ts`
Expected: PASS (2 pruebas).

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso.

---
### Task 14: Registrar gasto (`/app/gastos/nuevo`)

**Files:**
- Modify (reemplazo completo): `frontend/src/app/features/gastos/gasto-form/gasto-form.component.ts`
- Modify (reemplazo completo): `frontend/src/app/features/gastos/gasto-form/gasto-form.component.html`
- Modify: `frontend/src/styles/gastos.css` (agregar al final)
- Test: `frontend/src/app/features/gastos/gasto-form/gasto-form.component.spec.ts`

**Interfaces:**
- Consumes:
  - `GastosStore.registrar(datos: NuevoGasto): Observable<Gasto>` (Tarea 6)
  - de `CrudService`: `opciones`, `estaCargado`, `estaCargando`, `getItems` y `asegurarCargado`
  - `SelectorReferenciaComponent` (Tarea 10)
  - `hoyIso` (Tarea 1)
  - `HTTP_TIMEOUT_MS` y `mensajeError`
- Produces: `GastoFormComponent`, selector `app-gasto-form` (mismo selector que antes). Al registrar navega a `['/app/gastos', id]` con `fragment: 'paso'`.

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/features/gastos/gasto-form/gasto-form.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { HttpErrorResponse } from '@angular/common/http';
import { Router, provideRouter } from '@angular/router';
import { Observable, of, throwError } from 'rxjs';
import { GastoFormComponent } from './gasto-form.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { CrudService } from '../../../core/services/crud.service';
import { ToastService } from '../../../core/services/toast.service';

interface Formulario {
  viajeId: { set(v: string): void };
  clienteId: { set(v: string): void };
  monto: string;
  fecha: string;
}

function montar(registrar: () => Observable<unknown> = vi.fn(() => of({ id: 31 }))) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: GastosStore, useValue: { registrar } },
      {
        provide: CrudService,
        useValue: { asegurarCargado: vi.fn(), opciones: () => [], estaCargado: () => true, estaCargando: () => false, getItems: () => [] },
      },
    ],
  });
  const f = TestBed.createComponent(GastoFormComponent);
  f.detectChanges();
  return { f, registrar, form: f.componentInstance as unknown as Formulario };
}

function enviar(el: HTMLElement): void {
  el.querySelector('form')!.dispatchEvent(new Event('submit'));
}

describe('GastoFormComponent', () => {
  it('sin datos muestra los cuatro mensajes exactos', () => {
    const { f, form } = montar();
    form.fecha = '';
    enviar(f.nativeElement);
    f.detectChanges();
    const errores = [...f.nativeElement.querySelectorAll('.campo__error')].map((e: Element) => e.textContent?.trim());
    expect(errores).toEqual([
      'Selecciona el viaje al que corresponde este gasto.',
      'Selecciona el cliente responsable de pago.',
      'Ingresa un monto mayor a S/ 0.00.',
      'Selecciona la fecha del gasto.',
    ]);
  });

  it('al registrar abre el gasto nuevo justo en el paso de adjuntar', () => {
    const { f, form, registrar } = montar();
    const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    form.viajeId.set('12');
    form.clienteId.set('5');
    form.monto = '120.5';
    form.fecha = '2026-10-02';
    enviar(f.nativeElement);
    expect(registrar).toHaveBeenCalledWith({ viaje_id: 12, cliente_id: 5, monto: 120.5, fecha: '2026-10-02' });
    expect(navegar).toHaveBeenCalledWith(['/app/gastos', 31], { fragment: 'paso' });
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe(
      'Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.',
    );
  });

  it('si la API responde 422 marca el viaje con el mensaje exacto', () => {
    const { f, form } = montar(vi.fn(() => throwError(() => new HttpErrorResponse({ status: 422 }))));
    form.viajeId.set('99');
    form.clienteId.set('5');
    form.monto = '10';
    form.fecha = '2026-10-02';
    enviar(f.nativeElement);
    f.detectChanges();
    expect(f.nativeElement.querySelector('#gf-viaje-error')?.textContent?.trim()).toBe(
      'Este viaje no existe o no está disponible. Verifica el número de viaje.',
    );
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/features/gastos/gasto-form/gasto-form.component.spec.ts`
Expected: FAIL. El componente actual no tiene `viajeId.set`, no usa `GastosStore.registrar` y no navega.

- [ ] **Step 3: Implementar**

Reemplaza `frontend/src/app/features/gastos/gasto-form/gasto-form.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, OnInit, computed, inject, signal } from '@angular/core';
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

  ngOnInit(): void {
    // RF-1/RF-2/RF-3 necesitan viajes y clientes existentes para el formulario.
    this.crud.asegurarCargado('viaje');
    this.crud.asegurarCargado('cliente');
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
```

Reemplaza `frontend/src/app/features/gastos/gasto-form/gasto-form.component.html`:

```html
<div class="pagina pagina--angosta">
  <a class="volver" routerLink="/app/gastos"><app-icono nombre="arrow-left" [tamano]="16" />Gastos</a>
  <header class="pagina__encabezado">
    <h1 class="pagina__titulo">Registrar gasto adicional</h1>
  </header>

  <form class="panel formulario formulario-gasto" (ngSubmit)="save()" novalidate>
    <div class="campo" [class.campo--con-error]="viajeError()">
      <label class="campo__etiqueta" for="gf-viaje">Viaje <span class="req" aria-hidden="true">*</span></label>
      <app-selector-referencia
        idControl="gf-viaje"
        [opciones]="opcionesViaje()"
        [(valor)]="viajeId"
        [invalido]="!!viajeError()"
        [describedBy]="viajeError() ? 'gf-viaje-error' : null"
        [cargando]="cargandoViajes()"
        placeholder="Busca por número o fecha"
        vacioTexto="Aún no hay viajes."
        crearRuta="/app/operacion/viajes"
        crearTexto="Crear viaje"
      />
      @if (sinViajes()) {
        <p class="campo__ayuda">Para registrar un gasto primero necesitas un viaje. <a class="enlace" routerLink="/app/operacion/viajes">Crear viaje</a></p>
      }
      @if (viajeError()) {
        <p class="campo__error" id="gf-viaje-error">{{ viajeError() }}</p>
      }
    </div>

    <div class="campo" [class.campo--con-error]="clienteError()">
      <label class="campo__etiqueta" for="gf-cliente">Cliente <span class="req" aria-hidden="true">*</span></label>
      <app-selector-referencia
        idControl="gf-cliente"
        [opciones]="opcionesCliente()"
        [(valor)]="clienteId"
        [invalido]="!!clienteError()"
        [describedBy]="clienteError() ? 'gf-cliente-error' : null"
        [cargando]="cargandoClientes()"
        placeholder="Busca por razón social"
        vacioTexto="Aún no hay clientes."
        crearRuta="/app/operacion/clientes"
        crearTexto="Crear cliente"
      />
      @if (sinClientes()) {
        <p class="campo__ayuda">También necesitas al menos un cliente. <a class="enlace" routerLink="/app/operacion/clientes">Crear cliente</a></p>
      }
      @if (clienteError()) {
        <p class="campo__error" id="gf-cliente-error">{{ clienteError() }}</p>
      }
    </div>

    <div class="formulario__fila">
      <div class="campo" [class.campo--con-error]="montoError()">
        <label class="campo__etiqueta" for="gf-monto">Monto (S/) <span class="req" aria-hidden="true">*</span></label>
        <div class="control-prefijo">
          <span class="control-prefijo__texto" aria-hidden="true">S/</span>
          <input
            class="control"
            id="gf-monto"
            name="monto"
            type="number"
            inputmode="decimal"
            min="0.01"
            step="0.01"
            placeholder="0.00"
            [(ngModel)]="monto"
            [attr.aria-invalid]="!!montoError() || null"
            [attr.aria-describedby]="montoError() ? 'gf-monto-error' : null"
          />
        </div>
        @if (montoError()) {
          <p class="campo__error" id="gf-monto-error">{{ montoError() }}</p>
        }
      </div>
      <div class="campo" [class.campo--con-error]="fechaError()">
        <label class="campo__etiqueta" for="gf-fecha">Fecha del gasto <span class="req" aria-hidden="true">*</span></label>
        <input
          class="control"
          id="gf-fecha"
          name="fecha"
          type="date"
          [(ngModel)]="fecha"
          [attr.aria-invalid]="!!fechaError() || null"
          [attr.aria-describedby]="fechaError() ? 'gf-fecha-error' : null"
        />
        @if (fechaError()) {
          <p class="campo__error" id="gf-fecha-error">{{ fechaError() }}</p>
        }
      </div>
    </div>

    <div class="formulario__acciones">
      <button class="btn btn--primario" type="submit" [disabled]="saving()">{{ saving() ? 'Registrando…' : 'Registrar gasto' }}</button>
      <a class="btn btn--fantasma" routerLink="/app/gastos">Cancelar</a>
    </div>
  </form>
</div>
```

Agrega al final de `frontend/src/styles/gastos.css`:

```css
/* ══ Gastos: registrar ══ */
@media (max-width: 767px) {
  /* En el celular el botón de registrar queda a mano, sobre la barra inferior */
  .formulario-gasto .formulario__acciones {
    position: sticky;
    bottom: calc(var(--tabbar-h) + env(safe-area-inset-bottom));
    margin: 0 calc(-1 * var(--e-4)) calc(-1 * var(--e-4));
    padding: var(--e-3) var(--e-4);
    border-top: 1px solid var(--linea);
    border-radius: 0 0 var(--r-panel) var(--r-panel);
    background: var(--panel);
  }

  .formulario-gasto .formulario__acciones .btn {
    flex: 1;
  }
}
```

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/features/gastos/gasto-form/gasto-form.component.spec.ts`
Expected: PASS (3 pruebas).

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso. El dashboard viejo sigue compilando porque el selector `app-gasto-form` no cambió.

---
### Task 15: Detalle del gasto con el paso que toca (`/app/gastos/:id`)

**Files:**
- Create: `frontend/src/app/features/gastos/adjuntar-comprobante/adjuntar-comprobante.component.ts`
- Create: `frontend/src/app/features/gastos/adjuntar-comprobante/adjuntar-comprobante.component.html`
- Create: `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.ts`
- Create: `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.html`
- Create: `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.ts`
- Create: `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.html`
- Modify: `frontend/src/styles/gastos.css` (agregar al final)
- Test: `frontend/src/app/features/gastos/adjuntar-comprobante/adjuntar-comprobante.component.spec.ts`
- Test: `frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.spec.ts`
- Test: `frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.spec.ts`

**Interfaces:**
- Consumes:
  - de `GastosStore`:
    - estado: `gastos`, `comprobantes`, `cargando` y `cargado`
    - consultas: `gasto(id)` y `comprobantesDe(id)`
    - acciones: `asegurarCargado()`, `cargar()`, `adjuntar(gastoId, archivo)`, `aprobar(id)` y `rechazar(id, motivo)`
  - `GastosApiService.fileUrl(archivoUrl)`
  - `CrudService.etiqueta`
  - de la lógica (Tarea 3): `pasosRuta`, `ultimoRechazo` y `esPestana`
  - componentes: `RutaGastoComponent`, `InsigniaEstadoComponent`, `EstadoVacioComponent` e `IconoComponent`
  - `formatMonto`, `formatFecha`, `formatFechaHora` y `mensajeError`
- Produces:
  - `AdjuntarComprobanteComponent`, selector `app-adjuntar-comprobante`, con inputs `gasto: Gasto` (requerido) y `rechazoPrevio: Comprobante | undefined`. Método público `elegir(archivo: File): void`.
  - `RevisionComprobanteComponent`, selector `app-revision-comprobante`, con input `comprobante: Comprobante` (requerido).
  - `GastoDetalleComponent`, con input `id: string` (parámetro de ruta).
  - `consultaDeOrigen(estado: unknown): Record<string, string>`

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/features/gastos/adjuntar-comprobante/adjuntar-comprobante.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { AdjuntarComprobanteComponent } from './adjuntar-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { ToastService } from '../../../core/services/toast.service';
import { Gasto } from '../../../core/models/gasto.model';

const gasto: Gasto = {
  id: 7, viaje_id: 1, cliente_id: 1, monto: 120, fecha: '2026-10-01T00:00:00Z', estado: 'Registrado',
  creado_por: 'Ana', creado_en: '2026-10-01T15:00:00Z',
};

function montar() {
  const adjuntar = vi.fn(() => of({ id: 1 }));
  TestBed.configureTestingModule({ providers: [{ provide: GastosStore, useValue: { adjuntar } }] });
  const f = TestBed.createComponent(AdjuntarComprobanteComponent);
  f.componentRef.setInput('gasto', gasto);
  f.detectChanges();
  const el: HTMLElement = f.nativeElement;
  const subir = () => (el.querySelector('.formulario__acciones .btn') as HTMLButtonElement).click();
  const error = () => el.querySelector('.campo__error')?.textContent?.trim();
  return { f, el, adjuntar, subir, error };
}

describe('AdjuntarComprobanteComponent', () => {
  it('al soltar un archivo no permitido muestra el mensaje exacto y no sube nada', () => {
    const { f, el, adjuntar, subir, error } = montar();
    const soltar = new Event('drop', { bubbles: true, cancelable: true });
    Object.defineProperty(soltar, 'dataTransfer', { value: { files: [new File(['x'], 'notas.txt', { type: 'text/plain' })] } });
    el.querySelector('.soltar')!.dispatchEvent(soltar);
    f.detectChanges();
    expect(error()).toBe('Solo se aceptan archivos JPG, PNG o PDF.');
    subir();
    f.detectChanges();
    expect(adjuntar).not.toHaveBeenCalled();
  });

  it('rechaza archivos de más de 10 MB', () => {
    const { f, error } = montar();
    f.componentInstance.elegir(new File([new Uint8Array(10 * 1024 * 1024 + 1)], 'boleta.pdf', { type: 'application/pdf' }));
    f.detectChanges();
    expect(error()).toBe('El archivo supera el tamaño máximo permitido (10 MB).');
  });

  it('sin archivo pide elegir uno', () => {
    const { f, subir, error } = montar();
    subir();
    f.detectChanges();
    expect(error()).toBe('Selecciona al menos un comprobante antes de continuar.');
  });

  it('sube el archivo elegido al gasto y avisa con el copy exacto', () => {
    const { f, adjuntar, subir } = montar();
    const archivo = new File(['x'], 'boleta.jpg', { type: 'image/jpeg' });
    f.componentInstance.elegir(archivo);
    f.detectChanges();
    subir();
    expect(adjuntar).toHaveBeenCalledWith(7, archivo);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Comprobante adjuntado. El gasto cambió a estado "Pagado".');
  });
});
```

`frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { RevisionComprobanteComponent } from './revision-comprobante.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ToastService } from '../../../core/services/toast.service';
import { Comprobante } from '../../../core/models/gasto.model';

const comprobante: Comprobante = {
  id: 9, gasto_id: 7, archivo_url: '/uploads/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z',
};

function montar() {
  const store = { aprobar: vi.fn(() => of({})), rechazar: vi.fn(() => of({})) };
  TestBed.configureTestingModule({
    providers: [
      { provide: GastosStore, useValue: store },
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => `http://localhost:8081${u}` } },
    ],
  });
  const f = TestBed.createComponent(RevisionComprobanteComponent);
  f.componentRef.setInput('comprobante', comprobante);
  f.detectChanges();
  const boton = (texto: string) =>
    [...(f.nativeElement as HTMLElement).querySelectorAll('button')].find(b => b.textContent?.includes(texto)) as HTMLButtonElement;
  return { f, store, boton };
}

describe('RevisionComprobanteComponent', () => {
  it('muestra el comprobante para revisarlo', () => {
    const { f } = montar();
    expect(f.nativeElement.querySelector('img')?.getAttribute('src')).toBe('http://localhost:8081/uploads/9.jpg');
  });

  it('aprobar avisa con el copy exacto', () => {
    const { store, boton } = montar();
    boton('Aprobar').click();
    expect(store.aprobar).toHaveBeenCalledWith(9);
    expect(TestBed.inject(ToastService).toasts()[0].title).toBe('Comprobante aprobado. El gasto quedó como "Sustentado".');
  });

  it('rechazar exige un motivo y lo envía sin espacios sobrantes', () => {
    const { f, store, boton } = montar();
    boton('Rechazar').click();
    f.detectChanges();
    boton('Confirmar rechazo').click();
    f.detectChanges();
    expect(f.nativeElement.querySelector('#motivo-error')?.textContent?.trim()).toBe('Debes indicar un motivo para rechazar el comprobante.');
    expect(store.rechazar).not.toHaveBeenCalled();
    const area: HTMLTextAreaElement = f.nativeElement.querySelector('#motivo-rechazo');
    area.value = '  Comprobante ilegible  ';
    area.dispatchEvent(new Event('input'));
    boton('Confirmar rechazo').click();
    expect(store.rechazar).toHaveBeenCalledWith(9, 'Comprobante ilegible');
    expect(TestBed.inject(ToastService).toasts().at(-1)?.title).toBe('Comprobante rechazado. El gasto volvió a estado "Registrado".');
  });
});
```

`frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { GastoDetalleComponent, consultaDeOrigen } from './gasto-detalle.component';
import { GastosStore } from '../../../core/state/gastos.store';
import { CrudService } from '../../../core/services/crud.service';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { comprobantesDe } from '../../../core/state/gastos.logic';

function gasto(id: number, estado: Gasto['estado']): Gasto {
  return { id, estado, monto: 120.5, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-01T15:00:00Z' };
}

function storeFalso(gastos: Gasto[], comprobantes: Comprobante[], cargado = true) {
  const g = signal(gastos);
  const c = signal(comprobantes);
  return {
    gastos: g, comprobantes: c, cargando: signal(!cargado), cargado: signal(cargado), error: signal(false),
    gasto: (id: number) => g().find(x => x.id === id),
    comprobantesDe: (id: number) => comprobantesDe(c(), id),
    asegurarCargado: vi.fn(), cargar: vi.fn(), adjuntar: vi.fn(), aprobar: vi.fn(), rechazar: vi.fn(),
  };
}

function montar(store: ReturnType<typeof storeFalso>, id: string) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: GastosStore, useValue: store },
      { provide: CrudService, useValue: { etiqueta: () => 'Agro Norte SAC', asegurarCargado: vi.fn() } },
      { provide: GastosApiService, useValue: { fileUrl: (u: string) => u } },
    ],
  });
  const f = TestBed.createComponent(GastoDetalleComponent);
  f.componentRef.setInput('id', id);
  f.detectChanges();
  return f;
}

describe('GastoDetalleComponent', () => {
  it('mientras carga muestra el esqueleto y luego avisa si el gasto no existe', () => {
    const store = storeFalso([], [], false);
    const f = montar(store, '999');
    expect(f.nativeElement.querySelector('.detalle__esqueleto')).not.toBeNull();
    store.cargando.set(false);
    store.cargado.set(true);
    f.detectChanges();
    expect(f.nativeElement.textContent).toContain('No encontramos el gasto N° 999.');
  });

  it('un gasto registrado muestra el paso de adjuntar', () => {
    const f = montar(storeFalso([gasto(31, 'Registrado')], []), '31');
    const paso: HTMLElement = f.nativeElement.querySelector('#paso');
    expect(paso.textContent).toContain('Adjuntar comprobante de pago');
    expect(paso.querySelector('app-adjuntar-comprobante')).not.toBeNull();
    expect(f.nativeElement.textContent).toContain('S/ 120.50');
  });

  it('un gasto pagado muestra la revisión del comprobante pendiente', () => {
    const pendiente: Comprobante = {
      id: 9, gasto_id: 32, archivo_url: '/u/9.jpg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: '2026-10-01T17:00:00Z',
    };
    const f = montar(storeFalso([gasto(32, 'Pagado')], [pendiente]), '32');
    expect(f.nativeElement.querySelector('#paso app-revision-comprobante')).not.toBeNull();
  });

  it('«Gastos» vuelve a la pestaña de origen', () => {
    expect(consultaDeOrigen({ origen: 'por-aprobar' })).toEqual({ estado: 'por-aprobar' });
    expect(consultaDeOrigen({ origen: 'todos' })).toEqual({});
    expect(consultaDeOrigen(null)).toEqual({});
    expect(consultaDeOrigen({ origen: 'otra' })).toEqual({});
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `npx ng test --watch=false --include "src/app/features/gastos/{adjuntar-comprobante,revision-comprobante,gasto-detalle}/*.spec.ts"`
Expected: FAIL. No existen los tres componentes.

- [ ] **Step 3: Implementar el paso «Adjuntar comprobante»**

`frontend/src/app/features/gastos/adjuntar-comprobante/adjuntar-comprobante.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, DestroyRef, inject, input, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { Comprobante, Gasto } from '../../../core/models/gasto.model';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';

const TIPOS_PERMITIDOS = ['image/png', 'image/jpeg', 'image/jpg', 'application/pdf'];
const TAMANO_MAXIMO = 10 * 1024 * 1024; // 10 MB

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
      this.error.set('El archivo supera el tamaño máximo permitido (10 MB).');
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
```

`frontend/src/app/features/gastos/adjuntar-comprobante/adjuntar-comprobante.component.html`:

```html
@if (rechazoPrevio(); as r) {
  <p class="alerta alerta--error">
    <app-icono nombre="alert-triangle" [tamano]="18" />
    <span>El comprobante anterior fue rechazado{{ r.motivo_rechazo ? ': ' + r.motivo_rechazo : '.' }}</span>
  </p>
}
<div
  class="soltar"
  role="button"
  tabindex="0"
  aria-describedby="ayuda-comprobante"
  [class.soltar--arrastrando]="arrastrando()"
  [class.soltar--con-archivo]="!!archivo()"
  (click)="selector.click()"
  (keydown.enter)="selector.click()"
  (keydown.space)="$event.preventDefault(); selector.click()"
  (dragover)="alArrastrar($event)"
  (dragleave)="alSalir($event)"
  (drop)="alSoltar($event)"
>
  <input #selector class="oculto" type="file" accept="image/png,image/jpeg,image/jpg,application/pdf" (change)="alSeleccionar($event)" />
  @if (archivo(); as f) {
    @if (previa(); as url) {
      <img class="soltar__previa" [src]="url" alt="Vista previa del comprobante seleccionado" />
    } @else {
      <app-icono class="soltar__icono" nombre="file-type-pdf" [tamano]="32" />
    }
    <span class="soltar__archivo">{{ f.name }}</span>
    <span class="soltar__cambiar">Cambiar archivo</span>
  } @else {
    <app-icono class="soltar__icono" nombre="upload" [tamano]="28" />
    <span class="soltar__texto soltar__texto--escritorio">Arrastra el archivo aquí o haz clic para seleccionar</span>
    <span class="soltar__texto soltar__texto--movil"><app-icono nombre="camera" [tamano]="20" />Tomar foto o elegir archivo</span>
  }
</div>
<p class="campo__ayuda" id="ayuda-comprobante">Formatos permitidos: JPG, PNG o PDF. Tamaño máximo 10 MB.</p>
@if (error()) {
  <p class="campo__error" role="alert">{{ error() }}</p>
}
<div class="formulario__acciones">
  <button class="btn btn--primario" type="button" [disabled]="subiendo()" (click)="subir()">
    <app-icono nombre="upload" [tamano]="18" />{{ subiendo() ? 'Subiendo…' : 'Subir comprobante' }}
  </button>
</div>
```

- [ ] **Step 4: Implementar el paso «Revisar comprobante»**

`frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, Injector, afterNextRender, computed, inject, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Comprobante } from '../../../core/models/gasto.model';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { ToastService } from '../../../core/services/toast.service';
import { GastosStore } from '../../../core/state/gastos.store';
import { mensajeError } from '../../../core/utils/error-messages';
import { formatFechaHora } from '../../../core/utils/format';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';

/** Revisión del comprobante pendiente: aprobar o rechazar con motivo, en línea y sin ventana modal. */
@Component({
  selector: 'app-revision-comprobante',
  imports: [FormsModule, IconoComponent],
  templateUrl: './revision-comprobante.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RevisionComprobanteComponent {
  readonly comprobante = input.required<Comprobante>();

  private readonly store = inject(GastosStore);
  private readonly toast = inject(ToastService);
  private readonly gastosApi = inject(GastosApiService);
  private readonly injector = inject(Injector);

  protected readonly rechazando = signal(false);
  protected readonly enviando = signal(false);
  protected readonly motivoError = signal('');
  protected motivo = '';
  protected readonly url = computed(() => this.gastosApi.fileUrl(this.comprobante().archivo_url));
  protected readonly esPdf = computed(() => this.comprobante().tipo_archivo === 'pdf');
  protected readonly formatFechaHora = formatFechaHora;

  protected aprobar(): void {
    this.enviando.set(true);
    this.store.aprobar(this.comprobante().id).subscribe({
      next: () => {
        this.enviando.set(false);
        this.toast.show('Comprobante aprobado. El gasto quedó como "Sustentado".', '', 'ok');
      },
      error: err => {
        this.enviando.set(false);
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }

  protected abrirRechazo(): void {
    this.rechazando.set(true);
    this.motivo = '';
    this.motivoError.set('');
    afterNextRender(() => document.getElementById('motivo-rechazo')?.focus(), { injector: this.injector });
  }

  protected cancelarRechazo(): void {
    this.rechazando.set(false);
    this.motivo = '';
    this.motivoError.set('');
  }

  protected confirmarRechazo(): void {
    const motivo = this.motivo.trim();
    if (!motivo) {
      this.motivoError.set('Debes indicar un motivo para rechazar el comprobante.');
      return;
    }
    this.enviando.set(true);
    this.store.rechazar(this.comprobante().id, motivo).subscribe({
      next: () => {
        this.enviando.set(false);
        this.toast.show('Comprobante rechazado. El gasto volvió a estado "Registrado".', '', 'ok');
        this.cancelarRechazo();
      },
      error: err => {
        this.enviando.set(false);
        this.toast.show(mensajeError(err), '', 'err');
      },
    });
  }
}
```

`frontend/src/app/features/gastos/revision-comprobante/revision-comprobante.component.html`:

```html
<div class="revision">
  <figure class="revision__vista">
    @if (esPdf()) {
      <div class="revision__pdf">
        <app-icono nombre="file-type-pdf" [tamano]="40" />
        <span>Comprobante en PDF</span>
      </div>
    } @else {
      <img class="revision__imagen" [src]="url()" alt="Comprobante de pago adjuntado" loading="lazy" />
    }
    <figcaption class="revision__pie">
      <span>Adjuntado el {{ formatFechaHora(comprobante().creado_en) }}</span>
      <a class="enlace" [href]="url()" target="_blank" rel="noopener">{{ esPdf() ? 'Abrir PDF' : 'Ver en tamaño completo' }}</a>
    </figcaption>
  </figure>

  @if (!rechazando()) {
    <div class="formulario__acciones">
      <button type="button" class="btn btn--primario" [disabled]="enviando()" (click)="aprobar()">
        <app-icono nombre="check" [tamano]="18" />Aprobar
      </button>
      <button type="button" class="btn btn--peligro" [disabled]="enviando()" (click)="abrirRechazo()">
        <app-icono nombre="x" [tamano]="18" />Rechazar
      </button>
    </div>
  } @else {
    <div class="revision__rechazo">
      <div class="campo" [class.campo--con-error]="motivoError()">
        <label class="campo__etiqueta" for="motivo-rechazo">Motivo del rechazo <span class="req" aria-hidden="true">*</span></label>
        <textarea
          class="control"
          id="motivo-rechazo"
          name="motivo"
          rows="3"
          [(ngModel)]="motivo"
          placeholder="Explica por qué se rechaza este comprobante (ej. monto no coincide, comprobante ilegible)"
          [attr.aria-invalid]="!!motivoError() || null"
          [attr.aria-describedby]="motivoError() ? 'motivo-error' : null"
        ></textarea>
        @if (motivoError()) {
          <p class="campo__error" id="motivo-error">{{ motivoError() }}</p>
        }
      </div>
      <div class="formulario__acciones">
        <button type="button" class="btn btn--peligro" [disabled]="enviando()" (click)="confirmarRechazo()">Confirmar rechazo</button>
        <button type="button" class="btn btn--fantasma" (click)="cancelarRechazo()">Cancelar</button>
      </div>
    </div>
  }
</div>
```

- [ ] **Step 5: Implementar el detalle**

`frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.ts`:

```ts
import { Location } from '@angular/common';
import {
  ChangeDetectionStrategy, Component, OnInit, afterRenderEffect, computed, inject, input,
} from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { Comprobante } from '../../../core/models/gasto.model';
import { CrudService } from '../../../core/services/crud.service';
import { GastosApiService } from '../../../core/services/gastos-api.service';
import { esPestana, pasosRuta, ultimoRechazo } from '../../../core/state/gastos.logic';
import { GastosStore } from '../../../core/state/gastos.store';
import { formatFecha, formatFechaHora, formatMonto } from '../../../core/utils/format';
import { EstadoVacioComponent } from '../../../shared/ui/estado-vacio/estado-vacio.component';
import { IconoComponent } from '../../../shared/ui/icono/icono.component';
import { InsigniaEstadoComponent } from '../../../shared/ui/insignia-estado/insignia-estado.component';
import { RutaGastoComponent } from '../../../shared/ui/ruta-gasto/ruta-gasto.component';
import { AdjuntarComprobanteComponent } from '../adjuntar-comprobante/adjuntar-comprobante.component';
import { RevisionComprobanteComponent } from '../revision-comprobante/revision-comprobante.component';

/** La bandeja pasa { origen } en el state de la navegación; «← Gastos» vuelve a esa pestaña. */
export function consultaDeOrigen(estado: unknown): Record<string, string> {
  const origen = (estado as { origen?: unknown } | null)?.origen;
  return esPestana(origen) && origen !== 'todos' ? { estado: origen } : {};
}

@Component({
  selector: 'app-gasto-detalle',
  imports: [
    RouterLink, IconoComponent, InsigniaEstadoComponent, RutaGastoComponent, EstadoVacioComponent,
    AdjuntarComprobanteComponent, RevisionComprobanteComponent,
  ],
  templateUrl: './gasto-detalle.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GastoDetalleComponent implements OnInit {
  /** :id de la ruta */
  readonly id = input.required<string>();

  protected readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);
  private readonly gastosApi = inject(GastosApiService);
  private readonly ruta = inject(ActivatedRoute);
  private desplazado = false;

  protected readonly volverQuery = consultaDeOrigen(inject(Location).getState());
  protected readonly gasto = computed(() => this.store.gasto(Number(this.id())));
  protected readonly comprobantes = computed(() => this.store.comprobantesDe(Number(this.id())));
  protected readonly pasos = computed(() => {
    const g = this.gasto();
    return g ? pasosRuta(g, this.store.comprobantes()) : [];
  });
  protected readonly pendiente = computed(() => this.comprobantes().find(c => c.estado === 'Pendiente'));
  protected readonly aprobado = computed(() => this.comprobantes().find(c => c.estado === 'Aprobado'));
  protected readonly rechazoReciente = computed(() => this.comprobantes().find(c => c.estado === 'Rechazado'));
  protected readonly rechazoPrevio = computed(() => {
    const g = this.gasto();
    return g?.estado === 'Registrado' ? ultimoRechazo(this.store.comprobantes(), g.id) : undefined;
  });
  protected readonly cliente = computed(() => {
    const g = this.gasto();
    return g ? this.crud.etiqueta('cliente', g.cliente_id) : '';
  });
  protected readonly viaje = computed(() => {
    const g = this.gasto();
    return g ? this.crud.etiqueta('viaje', g.viaje_id) : '';
  });
  protected readonly formatMonto = formatMonto;
  protected readonly formatFecha = formatFecha;
  protected readonly formatFechaHora = formatFechaHora;

  constructor() {
    // Al llegar con #paso (desde la bandeja o tras registrar), lleva la vista al paso que toca.
    afterRenderEffect(() => {
      if (!this.gasto() || this.desplazado || this.ruta.snapshot.fragment !== 'paso') return;
      this.desplazado = true;
      document.getElementById('paso')?.scrollIntoView?.({ block: 'start' });
    });
  }

  ngOnInit(): void {
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    this.crud.asegurarCargado('viaje');
  }

  protected archivo(c: Comprobante): string {
    return this.gastosApi.fileUrl(c.archivo_url);
  }
}
```

`frontend/src/app/features/gastos/gasto-detalle/gasto-detalle.component.html`:

```html
<div class="pagina pagina--media">
  <a class="volver" routerLink="/app/gastos" [queryParams]="volverQuery"><app-icono nombre="arrow-left" [tamano]="16" />Gastos</a>

  @if (gasto(); as g) {
    <header class="detalle__cabecera">
      <div class="detalle__titulo-fila">
        <h1 class="pagina__titulo">Gasto <span class="mono">N° {{ g.id }}</span></h1>
        <app-insignia-estado [estado]="g.estado" />
      </div>
      <p class="detalle__monto">{{ formatMonto(g.monto) }}</p>
      <dl class="detalle__datos">
        <div><dt>Cliente</dt><dd>{{ cliente() }}</dd></div>
        <div><dt>Viaje</dt><dd>{{ viaje() }}</dd></div>
        <div><dt>Fecha del gasto</dt><dd>{{ formatFecha(g.fecha) }}</dd></div>
        <div><dt>Registrado por</dt><dd>{{ g.creado_por }} · {{ formatFechaHora(g.creado_en) }}</dd></div>
      </dl>
    </header>

    <section class="panel detalle__ruta" aria-labelledby="t-ruta">
      <h2 id="t-ruta" class="seccion-titulo">Recorrido</h2>
      <app-ruta-gasto [pasos]="pasos()" variante="completa" [rechazo]="rechazoReciente()" />
    </section>

    <section class="panel detalle__paso" id="paso" aria-labelledby="t-paso">
      @switch (g.estado) {
        @case ('Registrado') {
          <h2 id="t-paso" class="seccion-titulo">Adjuntar comprobante de pago</h2>
          <app-adjuntar-comprobante [gasto]="g" [rechazoPrevio]="rechazoPrevio()" />
        }
        @case ('Pagado') {
          <h2 id="t-paso" class="seccion-titulo">Revisar comprobante</h2>
          @if (pendiente(); as c) {
            <app-revision-comprobante [comprobante]="c" />
          } @else {
            <p class="texto-secundario">
              No encontramos el comprobante pendiente.
              <button type="button" class="enlace-boton" (click)="store.cargar()">Actualizar</button>
            </p>
          }
        }
        @default {
          <h2 id="t-paso" class="seccion-titulo">Gasto sustentado</h2>
          @if (aprobado(); as c) {
            <p class="detalle__cierre">
              <app-icono nombre="circle-check" [tamano]="20" />
              Aprobado por {{ c.aprobado_por ?? 'Logística' }} el {{ formatFechaHora(c.aprobado_en) }}.
            </p>
            <a class="btn btn--secundario btn--sm" [href]="archivo(c)" target="_blank" rel="noopener">
              <app-icono nombre="external-link" [tamano]="16" />Ver comprobante
            </a>
          }
        }
      }
    </section>

    <section class="detalle__historial" aria-labelledby="t-historial">
      <h2 id="t-historial" class="seccion-titulo">Comprobantes</h2>
      @if (comprobantes().length === 0) {
        <p class="texto-secundario">Este gasto aún no tiene comprobantes adjuntos.</p>
      } @else {
        <ul class="historial">
          @for (c of comprobantes(); track c.id) {
            <li class="historial__item">
              <app-icono [nombre]="c.tipo_archivo === 'pdf' ? 'file-type-pdf' : 'photo'" />
              <div class="historial__texto">
                <span class="historial__fecha">{{ formatFechaHora(c.creado_en) }}</span>
                @if (c.estado === 'Rechazado' && c.motivo_rechazo) {
                  <span class="historial__motivo">Motivo: {{ c.motivo_rechazo }}</span>
                }
              </div>
              <app-insignia-estado tipo="comprobante" [estado]="c.estado" />
              <a class="enlace" [href]="archivo(c)" target="_blank" rel="noopener">Ver comprobante</a>
            </li>
          }
        </ul>
        <p class="campo__ayuda">Este comprobante no se puede editar. Si es incorrecto, adjunta uno nuevo.</p>
      }
    </section>
  } @else if (store.cargando() || !store.cargado()) {
    <div class="detalle__esqueleto" aria-busy="true">
      <div class="esqueleto esqueleto--franja"></div>
      <div class="esqueleto esqueleto--bloque"></div>
      <div class="esqueleto esqueleto--bloque"></div>
    </div>
  } @else {
    <app-estado-vacio icono="receipt" [titulo]="'No encontramos el gasto N° ' + id() + '.'" texto="Puede que el número no exista o que ya no esté en la lista.">
      <a class="btn btn--secundario" routerLink="/app/gastos">Ir a Gastos</a>
    </app-estado-vacio>
  }
</div>
```

- [ ] **Step 6: Estilos del detalle, la zona de carga y la revisión**

Agrega al final de `frontend/src/styles/gastos.css`:

```css
/* ══ Gastos: detalle ══ */
.detalle__cabecera {
  margin-bottom: var(--e-6);
}

.detalle__titulo-fila {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--e-3);
}

.detalle__monto {
  margin-top: var(--e-2);
  font-size: var(--t-40);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.05;
  font-variant-numeric: tabular-nums;
}

.detalle__datos {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--e-4) var(--e-6);
  margin-top: var(--e-5);
}

.detalle__datos dt {
  color: var(--tinta-3);
  font-size: var(--t-12);
  font-weight: 600;
}

.detalle__datos dd {
  margin: 2px 0 0;
  font-weight: 500;
}

.detalle__ruta,
.detalle__paso {
  margin-bottom: var(--e-5);
}

.detalle__ruta .seccion-titulo,
.detalle__paso .seccion-titulo {
  margin-bottom: var(--e-5);
}

/* El paso que toca lleva el filete rojo de la cinta reflectiva */
.detalle__paso {
  border-top: 3px solid var(--rojo);
  scroll-margin-top: calc(var(--barra-h) + 16px);
}

.detalle__cierre {
  display: flex;
  align-items: center;
  gap: var(--e-2);
  margin-bottom: var(--e-4);
  color: var(--sustentado-tinta);
  font-weight: 600;
}

.detalle__historial .seccion-titulo {
  margin-bottom: var(--e-3);
}

.detalle__esqueleto {
  display: grid;
  gap: var(--e-4);
}

.historial {
  margin: var(--e-3) 0;
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--panel);
}

.historial__item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--e-3);
  padding: var(--e-3) var(--e-4);
  border-bottom: 1px solid var(--linea);
}

.historial__item:last-child {
  border-bottom: 0;
}

.historial__item > .icono {
  color: var(--tinta-2);
}

.historial__texto {
  flex: 1;
  display: grid;
  min-width: 160px;
}

.historial__fecha {
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.historial__motivo {
  color: var(--rojo-tinta);
  font-size: var(--t-13);
}

/* ── Zona para soltar el comprobante ── */
.soltar {
  display: grid;
  justify-items: center;
  gap: var(--e-2);
  padding: var(--e-8) var(--e-4);
  border: 2px dashed var(--linea-2);
  border-radius: var(--r-panel);
  background: var(--suelo);
  color: var(--tinta-2);
  text-align: center;
  cursor: pointer;
  transition: border-color 150ms ease, background-color 150ms ease;
}

.soltar:hover,
.soltar--arrastrando {
  border-color: var(--pagado);
  background: var(--pagado-suave);
  color: var(--tinta);
}

.soltar--con-archivo {
  border-style: solid;
  border-color: var(--tinta-3);
  background: var(--panel);
}

.soltar + .campo__ayuda {
  margin-top: var(--e-2);
}

.soltar__icono {
  color: var(--tinta-3);
}

.soltar__texto {
  font-weight: 600;
}

.soltar__texto--movil {
  display: none;
  align-items: center;
  gap: var(--e-2);
}

.soltar__previa {
  width: auto;
  max-height: 180px;
  border: 1px solid var(--linea);
  border-radius: var(--r-control);
}

.soltar__archivo {
  color: var(--tinta);
  font-weight: 600;
  word-break: break-all;
}

.soltar__cambiar {
  color: var(--rojo-tinta);
  font-size: var(--t-13);
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 3px;
}

/* En pantallas táctiles y celulares se ofrece cámara o archivos */
@media (hover: none), (max-width: 767px) {
  .soltar {
    padding: var(--e-6) var(--e-4);
  }

  .soltar__texto--escritorio {
    display: none;
  }

  .soltar__texto--movil {
    display: inline-flex;
  }
}

/* ── Revisión del comprobante ── */
.revision {
  display: grid;
  gap: var(--e-5);
}

.revision__vista {
  margin: 0;
  overflow: hidden;
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--suelo);
}

.revision__imagen {
  width: 100%;
  max-height: 460px;
  object-fit: contain;
  background: var(--panel-2);
}

.revision__pdf {
  display: grid;
  justify-items: center;
  gap: var(--e-2);
  padding: var(--e-10) var(--e-4);
  color: var(--tinta-2);
  font-weight: 600;
}

.revision__pie {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--e-3);
  padding: var(--e-3) var(--e-4);
  border-top: 1px solid var(--linea);
  background: var(--panel);
  color: var(--tinta-2);
  font-size: var(--t-13);
}

.revision__rechazo {
  display: grid;
  gap: var(--e-4);
  padding: var(--e-4);
  border: 1px solid var(--rojo-borde);
  border-radius: var(--r-panel);
  background: #fff8f9;
}
```

- [ ] **Step 7: Ejecutar y verificar que pasan**

Run: `npx ng test --watch=false --include "src/app/features/gastos/{adjuntar-comprobante,revision-comprobante,gasto-detalle}/*.spec.ts"`
Expected: PASS (11 pruebas: 4 de adjuntar, 3 de revisión y 4 del detalle).

- [ ] **Step 8: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso.

---
### Task 16: Operación y Catálogos (`/app/operacion/:modulo`, `/app/catalogos/:modulo`)

**Files:**
- Create: `frontend/src/app/features/crud-page/crud-page.logic.ts`
- Create: `frontend/src/app/features/crud-page/crud-page.component.ts`
- Create: `frontend/src/app/features/crud-page/crud-page.component.html`
- Create: `frontend/src/styles/crud.css`
- Modify: `frontend/src/styles.css` (agregar import)
- Test: `frontend/src/app/features/crud-page/crud-page.logic.spec.ts`
- Test: `frontend/src/app/features/crud-page/crud-page.component.spec.ts`

**Interfaces:**
- Consumes:
  - de `CrudService` (Tarea 5): `getCrud`, `getItems`, `estaCargando`, `estaCargado`, `asegurarConRelaciones`, `loadList`, `opciones`, `ref` y `guardar`
  - de la navegación (Tarea 4): `SeccionId`, `seccion`, `moduloPorSlug`, `rutaModulo` y `rutaDeClave`
  - `normalizar` (Tarea 1)
  - componentes: `SelectorReferenciaComponent`, `EstadoVacioComponent` e `IconoComponent`
  - `ToastService`
  - las clases `.tabla`, `.hoja` y `.dialogo` (Tarea 7)
- Produces:
  - `CrudPageComponent`, con inputs `modulo` (parámetro `:modulo`) y `seccion` (de `data.seccion`)
  - `interface EstadoCampo { field: CrudField; value: string; error: string }`
  - `validarCampos(campos: EstadoCampo[]): boolean`
  - `enumerar(partes: string[]): string`

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/features/crud-page/crud-page.logic.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { EstadoCampo, enumerar, validarCampos } from './crud-page.logic';

describe('validarCampos', () => {
  it('marca obligatorios, reglas y números con los mensajes de siempre', () => {
    const campos: EstadoCampo[] = [
      { field: { id: 'a', l: 'Nombre', tp: 'text', req: true }, value: '   ', error: '' },
      { field: { id: 'b', l: 'RUC', tp: 'text', req: true, rule: v => /^\d{11}$/.test(v), msg: '11 dígitos' }, value: '123', error: '' },
      { field: { id: 'c', l: 'Año', tp: 'number' }, value: 'dos mil', error: '' },
      { field: { id: 'd', l: 'Rubro', tp: 'ref', ref: 'rubro', req: true }, value: '3', error: 'viejo' },
    ];
    expect(validarCampos(campos)).toBe(false);
    expect(campos.map(c => c.error)).toEqual(['Campo obligatorio', '11 dígitos', 'Debe ser número', '']);
  });

  it('acepta un formulario completo', () => {
    const campos: EstadoCampo[] = [{ field: { id: 'a', l: 'Nombre', tp: 'text', req: true }, value: 'Volvo', error: '' }];
    expect(validarCampos(campos)).toBe(true);
  });
});

describe('enumerar', () => {
  it('une con comas y una «y» final', () => {
    expect(enumerar([])).toBe('');
    expect(enumerar(['clientes'])).toBe('clientes');
    expect(enumerar(['asignaciones', 'clientes', 'direcciones'])).toBe('asignaciones, clientes y direcciones');
  });
});
```

`frontend/src/app/features/crud-page/crud-page.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { Router, provideRouter } from '@angular/router';
import { CrudPageComponent } from './crud-page.component';
import { CrudService } from '../../core/services/crud.service';

function montar(seccion: 'operacion' | 'catalogos', modulo: string, preparar?: (crud: CrudService) => void) {
  TestBed.configureTestingModule({ providers: [provideRouter([]), provideHttpClient(), provideHttpClientTesting()] });
  const crud = TestBed.inject(CrudService);
  preparar?.(crud);
  const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
  const f = TestBed.createComponent(CrudPageComponent);
  f.componentRef.setInput('seccion', seccion);
  f.componentRef.setInput('modulo', modulo);
  f.detectChanges();
  return { el: f.nativeElement as HTMLElement, navegar };
}

describe('CrudPageComponent', () => {
  it('un módulo desconocido redirige al primero de la sección', () => {
    const { navegar } = montar('catalogos', 'xyz');
    expect(navegar).toHaveBeenCalledWith(['/app', 'catalogos', 'ubigeo'], { replaceUrl: true });
  });

  it('la tabla muestra nombres relacionados en lugar de IDs', () => {
    const { el } = montar('operacion', 'clientes', crud => {
      crud.data.set({
        cliente: [{ id_cliente: 1, razon_social: 'Agro Norte SAC', ruc: '20123456789', id_rubro: 3 }],
        rubro: [{ id_rubro: 3, nombre: 'Agroindustria' }],
      });
      crud.cargado.set({ cliente: true, rubro: true });
    });
    expect(el.querySelector('h1')?.textContent).toBe('Clientes');
    const fila = el.querySelector('tbody tr');
    expect(fila?.textContent).toContain('Agro Norte SAC');
    expect(fila?.textContent).toContain('Agroindustria');
  });

  it('avisa los datos previos que faltan con enlaces para crearlos', () => {
    const { el } = montar('operacion', 'viajes', crud => {
      crud.data.set({ viaje: [], unidad: [], cond_uni: [], cliente: [{ id_cliente: 1, razon_social: 'Agro Norte SAC' }], direccion: [] });
      crud.cargado.set({ viaje: true, unidad: true, cond_uni: true, cliente: true, direccion: true });
    });
    const enlaces = [...el.querySelectorAll('.alerta a')].map(a => a.textContent?.trim());
    expect(enlaces).toEqual(['unidades', 'asignaciones', 'direcciones']);
    expect(el.textContent).toContain('Antes de crear viajes necesitas: unidades, asignaciones, clientes y direcciones.');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `npx ng test --watch=false --include "src/app/features/crud-page/*.spec.ts"`
Expected: FAIL. No existen `./crud-page.logic` ni `./crud-page.component`.

- [ ] **Step 3: Implementar la lógica del formulario**

`frontend/src/app/features/crud-page/crud-page.logic.ts`:

```ts
import { CrudField } from '../../core/models/crud.model';

export interface EstadoCampo {
  field: CrudField;
  value: string;
  error: string;
}

/** Valida con las reglas de cada módulo y los mismos mensajes de la vista anterior. Escribe `error` en cada campo. */
export function validarCampos(campos: EstadoCampo[]): boolean {
  let valido = true;
  for (const campo of campos) {
    campo.error = '';
    const v = String(campo.value ?? '').trim();
    if (campo.field.req && !v) {
      campo.error = 'Campo obligatorio';
      valido = false;
      continue;
    }
    if (v && campo.field.rule && !campo.field.rule(v)) {
      campo.error = campo.field.msg ?? 'Inválido';
      valido = false;
      continue;
    }
    if (campo.field.tp === 'number' && v && isNaN(Number(v))) {
      campo.error = 'Debe ser número';
      valido = false;
    }
  }
  return valido;
}

/** «asignaciones, clientes y direcciones» */
export function enumerar(partes: string[]): string {
  if (partes.length <= 1) return partes.join('');
  return `${partes.slice(0, -1).join(', ')} y ${partes[partes.length - 1]}`;
}
```

- [ ] **Step 4: Implementar el componente**

`frontend/src/app/features/crud-page/crud-page.component.ts`:

```ts
import {
  ChangeDetectionStrategy, Component, ElementRef, Injector, afterNextRender, computed, effect, inject, input,
  signal, untracked, viewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { DomSanitizer, Title } from '@angular/platform-browser';
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
  protected readonly registro = signal<{ titulo: string; json: string } | null>(null);

  protected readonly cargando = computed(() => {
    const d = this.def();
    return !!d && this.crud.estaCargando(d.key) && !this.crud.estaCargado(d.key);
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
    const porClave: Record<string, OpcionRef[]> = {};
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
        this.crud.asegurarConRelaciones(m.clave);
        this.termino.set('');
        this.titulo.setTitle(`${m.etiqueta} · LIZAR`);
      });
    });
  }

  protected buscar(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
  }

  protected actualizar(): void {
    const d = this.def();
    if (d) this.crud.loadList(d.key);
  }

  protected nuevo(): void {
    const d = this.def();
    if (!d) return;
    this.campos.set(d.fields.map(field => ({ field, value: '', error: '' })));
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
    this.crud.guardar(d.key, d.pay(formulario)).subscribe(creado => {
      this.guardando.set(false);
      if (creado) this.cerrarHoja();
    });
  }

  protected verRegistro(item: Record<string, unknown>): void {
    const d = this.def();
    const m = this.mod();
    if (!d || !m) return;
    const nombre = d.nombre?.(item, this.crud.ref);
    this.registro.set({ titulo: nombre ? `${m.etiqueta} · ${nombre}` : m.etiqueta, json: JSON.stringify(item, null, 2) });
    this.dialogoRegistro()?.nativeElement.showModal();
  }
}
```

`frontend/src/app/features/crud-page/crud-page.component.html`:

```html
@if (def(); as d) {
  <div class="pagina">
    <header class="pagina__encabezado">
      <div>
        <p class="migas">{{ seccionNav().etiqueta }}</p>
        <h1 class="pagina__titulo">{{ mod()?.etiqueta }}</h1>
        <p class="pagina__bajada">{{ d.desc }}</p>
      </div>
      <div class="pagina__acciones">
        <button type="button" class="btn btn--primario" (click)="nuevo()">
          <app-icono nombre="plus" />{{ d.textos?.nuevo ?? 'Nuevo registro' }}
        </button>
      </div>
    </header>

    <nav class="pestanas pestanas--seccion" [attr.aria-label]="'Módulos de ' + seccionNav().etiqueta">
      @for (m of seccionNav().modulos; track m.slug) {
        <a
          class="pestanas__item"
          [routerLink]="rutaModulo(m)"
          [class.pestanas__item--activa]="m.slug === mod()?.slug"
          [attr.aria-current]="m.slug === mod()?.slug ? 'page' : null"
        >
          <app-icono [nombre]="m.icono" [tamano]="16" />{{ m.etiqueta }}
        </a>
      }
    </nav>

    @if (faltantes().length) {
      <div class="alerta alerta--via">
        <app-icono nombre="alert-triangle" [tamano]="18" />
        <div>
          <p>Antes de crear {{ d.textos?.plural ?? 'registros' }} necesitas registrar:</p>
          <ul class="alerta__lista">
            @for (r of faltantes(); track r.clave) {
              <li>
                @if (r.ruta) {
                  <a class="enlace" [routerLink]="r.ruta">{{ r.texto }}</a>
                } @else {
                  {{ r.texto }}
                }
              </li>
            }
          </ul>
        </div>
      </div>
    }

    <div class="barra-herramientas">
      <label class="buscador">
        <app-icono nombre="search" [tamano]="16" />
        <span class="oculto-visual">Buscar en {{ mod()?.etiqueta }}</span>
        <input class="control buscador__input" type="search" placeholder="Buscar" [value]="termino()" (input)="buscar($event)" />
      </label>
      <span class="barra-herramientas__conteo">{{ filas().length }} {{ filas().length === 1 ? 'registro' : 'registros' }}</span>
      <button type="button" class="btn btn--fantasma btn--sm" (click)="actualizar()">
        <app-icono nombre="refresh" [tamano]="16" />Actualizar
      </button>
    </div>

    @if (cargando()) {
      <div class="tabla-marco" aria-busy="true">
        @for (i of [1, 2, 3, 4, 5]; track i) {
          <div class="esqueleto esqueleto--fila"></div>
        }
      </div>
    } @else if (filas().length === 0) {
      @if (termino()) {
        <app-estado-vacio icono="search" titulo="Ningún registro coincide con la búsqueda." />
      } @else {
        <app-estado-vacio [icono]="mod()?.icono ?? 'inbox'" [titulo]="'Aún no hay ' + (d.textos?.plural ?? 'registros') + '.'" [texto]="requisitosTexto()">
          <button type="button" class="btn btn--primario" (click)="nuevo()">{{ d.textos?.nuevo ?? 'Nuevo registro' }}</button>
        </app-estado-vacio>
      }
    } @else {
      <div class="tabla-marco">
        <table class="tabla">
          <thead>
            <tr>
              @for (col of d.cols; track col) {
                <th scope="col">{{ col }}</th>
              }
              <th scope="col"><span class="oculto-visual">Acciones</span></th>
            </tr>
          </thead>
          <tbody>
            @for (fila of filas(); track $index) {
              <tr>
                @for (celda of fila.celdas; track $index) {
                  <td [innerHTML]="celda"></td>
                }
                <td class="tabla__acciones">
                  <button type="button" class="btn btn--fantasma btn--sm" (click)="verRegistro(fila.item)">
                    <app-icono nombre="braces" [tamano]="16" />Ver registro
                  </button>
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>
    }
  </div>

  <dialog #hoja class="hoja" aria-labelledby="hoja-titulo" (click)="cerrarSiFondo($event, hoja)">
    <form class="hoja__marco" (ngSubmit)="guardar()" novalidate>
      <header class="hoja__cabecera">
        <h2 id="hoja-titulo" class="hoja__titulo">{{ d.textos?.nuevo ?? 'Nuevo registro' }}</h2>
        <button type="button" class="btn btn--fantasma btn--icono" (click)="cerrarHoja()" aria-label="Cerrar">
          <app-icono nombre="x" />
        </button>
      </header>
      <div class="hoja__cuerpo">
        <p class="hoja__nota">Los campos con <span class="req">*</span> son obligatorios.</p>
        <div class="formulario formulario--rejilla">
          @for (c of campos(); track c.field.id) {
            <div class="campo" [class.campo--medio]="c.field.half" [class.campo--con-error]="!!c.error">
              <label class="campo__etiqueta" [for]="c.field.id">
                {{ c.field.l }}
                @if (c.field.req) {
                  <span class="req" aria-hidden="true">*</span>
                }
              </label>
              @switch (c.field.tp) {
                @case ('ref') {
                  <app-selector-referencia
                    [idControl]="c.field.id"
                    [opciones]="opciones()[c.field.ref!]"
                    [(valor)]="c.value"
                    [invalido]="!!c.error"
                    [describedBy]="c.error ? c.field.id + '-error' : null"
                    [cargando]="crud.estaCargando(c.field.ref!)"
                    [vacioTexto]="textoVacio(c.field.ref!)"
                    [crearRuta]="rutaDeClave(c.field.ref!)"
                    [crearTexto]="textoNuevo(c.field.ref!)"
                    (creando)="cerrarHoja()"
                  />
                }
                @case ('textarea') {
                  <textarea
                    class="control"
                    rows="3"
                    [id]="c.field.id"
                    [name]="c.field.id"
                    [(ngModel)]="c.value"
                    [placeholder]="c.field.placeholder ?? ''"
                    [attr.aria-invalid]="!!c.error || null"
                    [attr.aria-describedby]="c.error ? c.field.id + '-error' : null"
                  ></textarea>
                }
                @default {
                  <input
                    class="control"
                    [id]="c.field.id"
                    [name]="c.field.id"
                    [type]="c.field.tp"
                    [(ngModel)]="c.value"
                    [placeholder]="c.field.placeholder ?? ''"
                    [attr.inputmode]="c.field.tp === 'number' ? 'decimal' : null"
                    [attr.aria-invalid]="!!c.error || null"
                    [attr.aria-describedby]="c.error ? c.field.id + '-error' : null"
                  />
                }
              }
              @if (c.error) {
                <p class="campo__error" [id]="c.field.id + '-error'">{{ c.error }}</p>
              }
            </div>
          }
        </div>
      </div>
      <footer class="hoja__pie">
        <button type="submit" class="btn btn--primario" [disabled]="guardando()">{{ guardando() ? 'Guardando…' : 'Guardar' }}</button>
        <button type="button" class="btn btn--fantasma" (click)="cerrarHoja()">Cancelar</button>
      </footer>
    </form>
  </dialog>

  <dialog #dialogoRegistro class="dialogo" aria-labelledby="registro-titulo" (click)="cerrarSiFondo($event, dialogoRegistro)">
    @if (registro(); as r) {
      <div class="dialogo__marco">
        <header class="dialogo__cabecera">
          <h2 id="registro-titulo">{{ r.titulo }}</h2>
          <button type="button" class="btn btn--fantasma btn--icono" (click)="dialogoRegistro.close()" aria-label="Cerrar">
            <app-icono nombre="x" />
          </button>
        </header>
        <pre class="dialogo__json">{{ r.json }}</pre>
      </div>
    }
  </dialog>
}
```

- [ ] **Step 5: Crear `src/styles/crud.css` e importarlo**

`frontend/src/styles/crud.css`:

```css
/* ══ Operación y Catálogos ══ */
.pestanas--seccion {
  margin-bottom: var(--e-4);
}

.pestanas--seccion .pestanas__item .icono {
  color: var(--tinta-3);
}

.pestanas--seccion .pestanas__item--activa .icono {
  color: var(--rojo);
}

.pagina > .alerta {
  margin-bottom: var(--e-4);
}

.formulario--rejilla .combo__lista {
  max-height: 200px;
}
```

En `frontend/src/styles.css`, agrega al final:

```css
@import './styles/crud.css';
```

- [ ] **Step 6: Ejecutar y verificar que pasan**

Run: `npx ng test --watch=false --include "src/app/features/crud-page/*.spec.ts"`
Expected: PASS (6 pruebas).

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso.

---
### Task 17: Shell, rutas y guard (barra superior, menú de usuario, barra móvil, paleta ⌘K y página no encontrada)

**Files:**
- Create: `frontend/src/app/shell/shell.component.ts` y `shell.component.html`
- Create: `frontend/src/app/shell/barra-superior/barra-superior.component.ts` y `barra-superior.component.html`
- Create: `frontend/src/app/shell/menu-usuario/menu-usuario.component.ts`
- Create: `frontend/src/app/shell/barra-movil/barra-movil.component.ts` y `barra-movil.component.html`
- Create: `frontend/src/app/shell/paleta-comandos/paleta.logic.ts`, `paleta-comandos.component.ts` y `paleta-comandos.component.html`
- Create: `frontend/src/app/features/no-encontrada/no-encontrada.component.ts`
- Create: `frontend/src/styles/shell.css`
- Modify (reemplazo completo): `frontend/src/app/app.routes.ts`
- Modify (reemplazo completo): `frontend/src/app/app.config.ts`
- Modify (reemplazo completo): `frontend/src/app/core/guards/auth.guard.ts`
- Modify: `frontend/src/styles.css` (agregar import)
- Test: `frontend/src/app/core/guards/auth.guard.spec.ts`
- Test: `frontend/src/app/shell/paleta-comandos/paleta.logic.spec.ts`
- Test: `frontend/src/app/shell/menu-usuario/menu-usuario.component.spec.ts`
- Test: `frontend/src/app/app.routes.spec.ts`

**Interfaces:**
- Consumes:
  - de `GastosStore`: `gastos`, `pendientes` y `asegurarCargado()`
  - de `CrudService`: `etiqueta` y `asegurarCargado`
  - de `ApiService`: `health()` y `checkHealth()`
  - de `AuthService`: `usuario()`, `logout()` e `isAuth()`
  - de la navegación (Tarea 4): `ENLACES_PRINCIPALES`, `ACCIONES_PALETA`, `destinosPaleta`, `seccion`, `rutaModulo`, `EntradaPaleta` y `urlActual()`
  - de la lógica (Tarea 3): `buscarGastos` y `ordenarRecientes`
  - componentes: `MarcaComponent` e `IconoComponent`
  - `formatMonto` y `normalizar`
  - las páginas de las Tareas 12 a 16
- Produces:
  - `ShellComponent`
  - `resultadosPaleta(termino, gastos, nombreCliente): EntradaPaleta[]`
  - `iniciales(nombre): string`
  - `authGuard`, que redirige a `/login?volver=<url>`
  - `routes` con todo el árbol de la spec §3
  - `NoEncontradaComponent`

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/core/guards/auth.guard.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot, UrlTree, provideRouter } from '@angular/router';
import { authGuard } from './auth.guard';
import { AuthService } from '../services/auth.service';

function ejecutar(autenticado: boolean, url: string) {
  TestBed.configureTestingModule({
    providers: [provideRouter([]), { provide: AuthService, useValue: { isAuth: () => autenticado } }],
  });
  return TestBed.runInInjectionContext(() => authGuard({} as ActivatedRouteSnapshot, { url } as RouterStateSnapshot));
}

describe('authGuard', () => {
  it('deja pasar con sesión iniciada', () => {
    expect(ejecutar(true, '/app/gastos')).toBe(true);
  });

  it('sin sesión manda al login recordando a dónde iba', () => {
    const destino = ejecutar(false, '/app/gastos?estado=por-aprobar') as UrlTree;
    expect(TestBed.inject(Router).serializeUrl(destino)).toBe('/login?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
  });
});
```

`frontend/src/app/shell/paleta-comandos/paleta.logic.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { resultadosPaleta } from './paleta.logic';
import { Gasto } from '../../core/models/gasto.model';

const gastos: Gasto[] = [
  { id: 31, estado: 'Registrado', monto: 120, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-01T12:00:00Z' },
  { id: 32, estado: 'Pagado', monto: 85.5, viaje_id: 1, cliente_id: 2, fecha: '2026-10-02T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-02T12:00:00Z' },
];
const nombre = (id: number) => (id === 1 ? 'Agro Norte SAC' : 'Logística Andina');

describe('resultadosPaleta', () => {
  it('sin término ofrece las acciones y todos los destinos, sin gastos', () => {
    const r = resultadosPaleta('', gastos, nombre);
    expect(r.filter(e => e.grupo === 'Acciones')).toHaveLength(3);
    expect(r.filter(e => e.grupo === 'Ir a')).toHaveLength(13);
    expect(r.some(e => e.grupo === 'Gastos')).toBe(false);
  });

  it('encuentra módulos sin importar tildes', () => {
    expect(resultadosPaleta('flota', gastos, nombre).map(e => e.ruta)).toEqual(['/app/operacion/flota']);
    expect(resultadosPaleta('direccion', gastos, nombre).map(e => e.etiqueta)).toEqual(['Direcciones']);
  });

  it('encuentra gastos por número y por cliente', () => {
    expect(resultadosPaleta('31', gastos, nombre)).toEqual([
      { id: 'gasto-31', grupo: 'Gastos', etiqueta: 'Gasto N° 31', detalle: 'Agro Norte SAC · S/ 120.00', icono: 'receipt', ruta: '/app/gastos/31' },
    ]);
    expect(resultadosPaleta('andina', gastos, nombre).map(e => e.id)).toEqual(['gasto-32']);
  });
});
```

`frontend/src/app/shell/menu-usuario/menu-usuario.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { iniciales } from './menu-usuario.component';

describe('iniciales', () => {
  it('usa las iniciales de nombre y apellido, o las dos primeras letras', () => {
    expect(iniciales('Ana Quispe')).toBe('AQ');
    expect(iniciales('admin')).toBe('AD');
    expect(iniciales('   ')).toBe('US');
  });
});
```

`frontend/src/app/app.routes.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { routes } from './app.routes';
import { authGuard } from './core/guards/auth.guard';

describe('rutas', () => {
  it('cada pantalla tiene su URL y /app pide sesión', () => {
    const app = routes.find(r => r.path === 'app')!;
    expect(app.canActivate).toEqual([authGuard]);
    expect(app.children?.map(c => c.path)).toEqual([
      '', 'inicio', 'gastos', 'gastos/nuevo', 'gastos/:id', 'operacion', 'operacion/:modulo', 'catalogos', 'catalogos/:modulo', '**',
    ]);
    expect(app.children?.find(c => c.path === 'catalogos/:modulo')?.data).toEqual({ seccion: 'catalogos' });
  });

  it('los enlaces viejos siguen funcionando y lo desconocido cae en «no encontrada»', () => {
    expect(routes.find(r => r.path === 'dashboard')?.redirectTo).toBe('app/inicio');
    expect(routes.at(-1)?.path).toBe('**');
    expect(routes.at(-1)?.loadComponent).toBeTypeOf('function');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `npx ng test --watch=false --include src/app/core/guards/auth.guard.spec.ts --include "src/app/shell/**/*.spec.ts" --include src/app/app.routes.spec.ts`
Expected: FAIL. El guard no agrega `volver`, no existen `paleta.logic` ni `menu-usuario`, y las rutas viejas no tienen `app`.

- [ ] **Step 3: Guard, rutas y configuración del router**

Reemplaza `frontend/src/app/core/guards/auth.guard.ts`:

```ts
import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = (_ruta, estado) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (auth.isAuth()) return true;
  return router.createUrlTree(['/login'], { queryParams: { volver: estado.url } });
};
```

Reemplaza `frontend/src/app/app.routes.ts`:

```ts
import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  {
    path: '',
    title: 'LIZAR SRL · Fleet Manager Pro',
    loadComponent: () => import('./features/landing/landing.component').then(m => m.LandingComponent),
  },
  {
    path: 'login',
    title: 'Iniciar sesión · LIZAR',
    loadComponent: () => import('./features/login/login.component').then(m => m.LoginComponent),
  },
  {
    path: 'register',
    title: 'Crear usuario · LIZAR',
    loadComponent: () => import('./features/register/register.component').then(m => m.RegisterComponent),
  },
  { path: 'dashboard', redirectTo: 'app/inicio' },
  {
    path: 'app',
    canActivate: [authGuard],
    loadComponent: () => import('./shell/shell.component').then(m => m.ShellComponent),
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'inicio' },
      {
        path: 'inicio',
        title: 'Inicio · LIZAR',
        loadComponent: () => import('./features/inicio/inicio.component').then(m => m.InicioComponent),
      },
      {
        path: 'gastos',
        title: 'Gastos · LIZAR',
        loadComponent: () => import('./features/gastos/gastos-lista/gastos-lista.component').then(m => m.GastosListaComponent),
      },
      {
        path: 'gastos/nuevo',
        title: 'Registrar gasto · LIZAR',
        loadComponent: () => import('./features/gastos/gasto-form/gasto-form.component').then(m => m.GastoFormComponent),
      },
      {
        path: 'gastos/:id',
        title: 'Gasto · LIZAR',
        loadComponent: () => import('./features/gastos/gasto-detalle/gasto-detalle.component').then(m => m.GastoDetalleComponent),
      },
      { path: 'operacion', pathMatch: 'full', redirectTo: 'operacion/viajes' },
      {
        path: 'operacion/:modulo',
        data: { seccion: 'operacion' },
        loadComponent: () => import('./features/crud-page/crud-page.component').then(m => m.CrudPageComponent),
      },
      { path: 'catalogos', pathMatch: 'full', redirectTo: 'catalogos/ubigeo' },
      {
        path: 'catalogos/:modulo',
        data: { seccion: 'catalogos' },
        loadComponent: () => import('./features/crud-page/crud-page.component').then(m => m.CrudPageComponent),
      },
      { path: '**', redirectTo: 'inicio' },
    ],
  },
  {
    path: '**',
    title: 'Página no encontrada · LIZAR',
    loadComponent: () => import('./features/no-encontrada/no-encontrada.component').then(m => m.NoEncontradaComponent),
  },
];
```

Reemplaza `frontend/src/app/app.config.ts`:

```ts
import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter, withComponentInputBinding, withInMemoryScrolling, withViewTransitions } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { authInterceptor } from './core/interceptors/auth.interceptor';

import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(
      routes,
      withComponentInputBinding(),
      withViewTransitions({ skipInitialTransition: true }),
      withInMemoryScrolling({ scrollPositionRestoration: 'enabled', anchorScrolling: 'enabled' }),
    ),
    provideHttpClient(withInterceptors([authInterceptor])),
  ],
};
```

- [ ] **Step 4: Página no encontrada**

`frontend/src/app/features/no-encontrada/no-encontrada.component.ts`:

```ts
import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';

@Component({
  selector: 'app-no-encontrada',
  imports: [RouterLink, MarcaComponent],
  template: `
    <main class="no-encontrada">
      <app-marca [sobreAsfalto]="false" />
      <p class="no-encontrada__codigo">Error 404</p>
      <h1>Página no encontrada</h1>
      <p>La dirección que abriste no existe.</p>
      <a class="btn btn--primario" routerLink="/">Ir al inicio</a>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NoEncontradaComponent {}
```

- [ ] **Step 5: Lógica y componente de la paleta de comandos**

`frontend/src/app/shell/paleta-comandos/paleta.logic.ts`:

```ts
import { Gasto } from '../../core/models/gasto.model';
import { ACCIONES_PALETA, EntradaPaleta, destinosPaleta } from '../../core/navigation/navegacion';
import { buscarGastos, ordenarRecientes } from '../../core/state/gastos.logic';
import { formatMonto } from '../../core/utils/format';
import { normalizar } from '../../core/utils/texto';

export const MAX_GASTOS_PALETA = 6;

/** Acciones, destinos y gastos que coinciden con lo escrito en la paleta (⌘K). */
export function resultadosPaleta(termino: string, gastos: Gasto[], nombreCliente: (clienteId: number) => string): EntradaPaleta[] {
  const fijas = [...ACCIONES_PALETA, ...destinosPaleta()];
  const t = normalizar(termino).trim();
  if (!t) return fijas;
  const coinciden = fijas.filter(e => normalizar(`${e.etiqueta} ${e.detalle ?? ''}`).includes(t));
  const deGastos = buscarGastos(ordenarRecientes(gastos), termino, nombreCliente)
    .slice(0, MAX_GASTOS_PALETA)
    .map<EntradaPaleta>(g => ({
      id: `gasto-${g.id}`,
      grupo: 'Gastos',
      etiqueta: `Gasto N° ${g.id}`,
      detalle: `${nombreCliente(g.cliente_id)} · ${formatMonto(g.monto)}`,
      icono: 'receipt',
      ruta: `/app/gastos/${g.id}`,
    }));
  return [...coinciden, ...deGastos];
}
```

`frontend/src/app/shell/paleta-comandos/paleta-comandos.component.ts`:

```ts
import {
  ChangeDetectionStrategy, Component, ElementRef, afterRenderEffect, computed, inject, signal, viewChild,
} from '@angular/core';
import { Router } from '@angular/router';
import { EntradaPaleta, GrupoPaleta } from '../../core/navigation/navegacion';
import { CrudService } from '../../core/services/crud.service';
import { GastosStore } from '../../core/state/gastos.store';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { resultadosPaleta } from './paleta.logic';

const GRUPOS: readonly GrupoPaleta[] = ['Acciones', 'Ir a', 'Gastos'];

/** Buscar o ir a… (⌘K / Ctrl+K). Sin animación: se usa muchas veces al día. */
@Component({
  selector: 'app-paleta-comandos',
  imports: [IconoComponent],
  templateUrl: './paleta-comandos.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PaletaComandosComponent {
  private readonly router = inject(Router);
  private readonly store = inject(GastosStore);
  private readonly crud = inject(CrudService);
  private readonly dialogo = viewChild.required<ElementRef<HTMLDialogElement>>('dialogo');
  private readonly campo = viewChild.required<ElementRef<HTMLInputElement>>('campo');

  protected readonly termino = signal('');
  protected readonly activo = signal(0);
  protected readonly resultados = computed(() =>
    resultadosPaleta(this.termino(), this.store.gastos(), id => this.crud.etiqueta('cliente', id)),
  );
  protected readonly grupos = computed(() => {
    const conIndice = this.resultados().map((entrada, indice) => ({ entrada, indice }));
    return GRUPOS.map(nombre => ({ nombre, entradas: conIndice.filter(x => x.entrada.grupo === nombre) })).filter(
      g => g.entradas.length,
    );
  });
  protected readonly activoId = computed(() => this.resultados()[this.activo()]?.id ?? null);

  constructor() {
    afterRenderEffect(() => {
      const id = this.activoId();
      if (id) document.getElementById(id)?.scrollIntoView?.({ block: 'nearest' });
    });
  }

  abrir(): void {
    this.termino.set('');
    this.activo.set(0);
    this.store.asegurarCargado();
    this.crud.asegurarCargado('cliente');
    const dialogo = this.dialogo().nativeElement;
    if (!dialogo.open) dialogo.showModal();
    const campo = this.campo().nativeElement;
    campo.value = '';
    campo.focus();
  }

  protected cerrar(): void {
    this.dialogo().nativeElement.close();
  }

  protected cerrarSiFondo(evento: MouseEvent): void {
    if (evento.target === this.dialogo().nativeElement) this.cerrar();
  }

  protected escribir(evento: Event): void {
    this.termino.set((evento.target as HTMLInputElement).value);
    this.activo.set(0);
  }

  protected ir(entrada: EntradaPaleta): void {
    this.cerrar();
    this.router.navigate([entrada.ruta], { queryParams: entrada.queryParams });
  }

  protected teclear(evento: KeyboardEvent): void {
    const total = this.resultados().length;
    if (evento.key === 'ArrowDown') {
      evento.preventDefault();
      this.activo.set(total ? (this.activo() + 1) % total : 0);
    } else if (evento.key === 'ArrowUp') {
      evento.preventDefault();
      this.activo.set(total ? (this.activo() - 1 + total) % total : 0);
    } else if (evento.key === 'Enter') {
      const entrada = this.resultados()[this.activo()];
      if (entrada) {
        evento.preventDefault();
        this.ir(entrada);
      }
    }
  }
}
```

`frontend/src/app/shell/paleta-comandos/paleta-comandos.component.html`:

```html
<dialog #dialogo class="paleta" aria-label="Buscar o ir a" (click)="cerrarSiFondo($event)">
  <div class="paleta__marco">
    <div class="paleta__campo">
      <app-icono nombre="search" [tamano]="18" />
      <input
        #campo
        class="paleta__input"
        type="text"
        role="combobox"
        aria-expanded="true"
        aria-controls="paleta-lista"
        aria-autocomplete="list"
        autocomplete="off"
        spellcheck="false"
        placeholder="Buscar o ir a…"
        [attr.aria-activedescendant]="activoId()"
        (input)="escribir($event)"
        (keydown)="teclear($event)"
      />
      <kbd class="paleta__esc">Esc</kbd>
    </div>
    <div class="paleta__lista" id="paleta-lista" role="listbox" aria-label="Resultados">
      @for (g of grupos(); track g.nombre) {
        <div class="paleta__grupo" role="group" [attr.aria-label]="g.nombre">
          <p class="paleta__grupo-titulo" aria-hidden="true">{{ g.nombre }}</p>
          @for (x of g.entradas; track x.entrada.id) {
            <div
              class="paleta__opcion"
              role="option"
              [id]="x.entrada.id"
              [attr.aria-selected]="x.indice === activo()"
              [class.paleta__opcion--activa]="x.indice === activo()"
              (click)="ir(x.entrada)"
              (pointermove)="activo.set(x.indice)"
            >
              <app-icono [nombre]="x.entrada.icono" [tamano]="18" />
              <span class="paleta__etiqueta">{{ x.entrada.etiqueta }}</span>
              @if (x.entrada.detalle) {
                <span class="paleta__detalle">{{ x.entrada.detalle }}</span>
              }
            </div>
          }
        </div>
      } @empty {
        <p class="paleta__vacio">Sin resultados para «{{ termino() }}».</p>
      }
    </div>
  </div>
</dialog>
```

- [ ] **Step 6: Menú de usuario, barra superior y barra móvil**

`frontend/src/app/shell/menu-usuario/menu-usuario.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { AuthService } from '../../core/services/auth.service';
import { IconoComponent } from '../../shared/ui/icono/icono.component';

/** «AQ» para «Ana Quispe»; «AD» para «admin». */
export function iniciales(nombre: string): string {
  const partes = nombre.trim().split(/\s+/).filter(Boolean);
  if (!partes.length) return 'US';
  return (partes.length > 1 ? partes[0][0] + partes[1][0] : partes[0].slice(0, 2)).toUpperCase();
}

/** Menú de cuenta con la API Popover nativa (descarte con clic afuera y Esc). */
@Component({
  selector: 'app-menu-usuario',
  imports: [IconoComponent],
  template: `
    <button type="button" class="usuario" popovertarget="menu-usuario" aria-haspopup="menu" [attr.aria-label]="'Cuenta de ' + nombre()">
      <span class="usuario__iniciales" aria-hidden="true">{{ siglas() }}</span>
      <span class="usuario__nombre">{{ nombre() }}</span>
      <app-icono class="usuario__chevron" nombre="chevron-down" [tamano]="14" />
    </button>
    <div id="menu-usuario" class="menu" popover role="menu">
      <div class="menu__cabecera">
        <span class="menu__nombre">{{ nombre() }}</span>
        @if (rol()) {
          <span class="menu__rol">{{ rol() }}</span>
        }
      </div>
      <button type="button" class="menu__item" role="menuitem" (click)="salir()">
        <app-icono nombre="logout" [tamano]="16" />Cerrar sesión
      </button>
    </div>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MenuUsuarioComponent {
  private readonly auth = inject(AuthService);
  protected readonly nombre = computed(() => this.auth.usuario()?.nombre ?? 'Usuario');
  protected readonly rol = computed(() => this.auth.usuario()?.rol ?? '');
  protected readonly siglas = computed(() => iniciales(this.nombre()));

  protected salir(): void {
    (document.getElementById('menu-usuario') as (HTMLElement & { hidePopover?: () => void }) | null)?.hidePopover?.();
    this.auth.logout();
  }
}
```

`frontend/src/app/shell/barra-superior/barra-superior.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, computed, inject, output } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { ENLACES_PRINCIPALES } from '../../core/navigation/navegacion';
import { urlActual } from '../../core/navigation/url-actual';
import { ApiService } from '../../core/services/api.service';
import { GastosStore } from '../../core/state/gastos.store';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { MenuUsuarioComponent } from '../menu-usuario/menu-usuario.component';

@Component({
  selector: 'app-barra-superior',
  imports: [RouterLink, RouterLinkActive, MarcaComponent, IconoComponent, MenuUsuarioComponent],
  templateUrl: './barra-superior.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BarraSuperiorComponent {
  readonly buscar = output<void>();

  protected readonly api = inject(ApiService);
  private readonly store = inject(GastosStore);
  private readonly url = urlActual();

  protected readonly enlaces = ENLACES_PRINCIPALES;
  protected readonly pendientes = computed(() => this.store.pendientes().length);
  protected readonly atajo = /Mac|iPhone|iPad/.test(navigator.userAgent) ? '⌘K' : 'Ctrl K';
  protected readonly titulo = computed(() => ENLACES_PRINCIPALES.find(e => this.url().startsWith(e.ruta))?.etiqueta ?? '');
  /** Solo se muestra cuando la verificación terminó y el servicio no responde. */
  protected readonly servicioCaido = computed(() => {
    const h = this.api.health();
    return !h.connected && h.env !== '…';
  });
}
```

`frontend/src/app/shell/barra-superior/barra-superior.component.html`:

```html
<header class="barra">
  <div class="barra__interior">
    <a class="barra__marca" routerLink="/app/inicio" aria-label="LIZAR, ir a Inicio"><app-marca /></a>
    <span class="barra__titulo-movil" aria-hidden="true">{{ titulo() }}</span>
    <nav class="barra__nav" aria-label="Principal">
      @for (e of enlaces; track e.ruta) {
        <a class="barra__enlace" [routerLink]="e.ruta" routerLinkActive="barra__enlace--activo" ariaCurrentWhenActive="page">
          {{ e.etiqueta }}
          @if (e.ruta === '/app/gastos' && pendientes() > 0) {
            <span class="contador contador--via">
              {{ pendientes() }}<span class="oculto-visual"> por aprobar</span>
            </span>
          }
        </a>
      }
    </nav>
    <div class="barra__acciones">
      @if (servicioCaido()) {
        <button type="button" class="barra__servicio" (click)="api.checkHealth()">
          <app-icono nombre="cloud-off" [tamano]="16" /><span>Servicio no disponible</span>
        </button>
      }
      <button type="button" class="barra__buscar" (click)="buscar.emit()" aria-label="Buscar o ir a" aria-keyshortcuts="Meta+K Control+K">
        <app-icono nombre="search" [tamano]="16" />
        <span class="barra__buscar-texto">Buscar o ir a…</span>
        <kbd class="barra__atajo">{{ atajo }}</kbd>
      </button>
      <app-menu-usuario />
    </div>
  </div>
  <div class="cinta" aria-hidden="true"></div>
</header>
```

`frontend/src/app/shell/barra-movil/barra-movil.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, ElementRef, computed, inject, viewChild } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { rutaModulo, seccion } from '../../core/navigation/navegacion';
import { urlActual } from '../../core/navigation/url-actual';
import { AuthService } from '../../core/services/auth.service';
import { GastosStore } from '../../core/state/gastos.store';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { iniciales } from '../menu-usuario/menu-usuario.component';

/** Barra inferior del celular: Inicio, Gastos, Operación y Más (Catálogos y cuenta). */
@Component({
  selector: 'app-barra-movil',
  imports: [RouterLink, RouterLinkActive, IconoComponent],
  templateUrl: './barra-movil.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BarraMovilComponent {
  protected readonly auth = inject(AuthService);
  private readonly store = inject(GastosStore);
  private readonly url = urlActual();
  private readonly hoja = viewChild.required<ElementRef<HTMLDialogElement>>('hojaMas');

  protected readonly pendientes = computed(() => this.store.pendientes().length);
  protected readonly catalogos = seccion('catalogos').modulos;
  protected readonly enCatalogos = computed(() => this.url().startsWith('/app/catalogos'));
  protected readonly nombre = computed(() => this.auth.usuario()?.nombre ?? 'Usuario');
  protected readonly siglas = computed(() => iniciales(this.nombre()));
  protected readonly rutaModulo = rutaModulo;

  protected abrirMas(): void {
    this.hoja().nativeElement.showModal();
  }

  protected cerrarMas(): void {
    this.hoja().nativeElement.close();
  }

  protected cerrarSiFondo(evento: MouseEvent): void {
    if (evento.target === this.hoja().nativeElement) this.cerrarMas();
  }

  protected salir(): void {
    this.cerrarMas();
    this.auth.logout();
  }
}
```

`frontend/src/app/shell/barra-movil/barra-movil.component.html`:

```html
<nav class="tabbar" aria-label="Principal (celular)">
  <a class="tabbar__item" routerLink="/app/inicio" routerLinkActive="tabbar__item--activo" ariaCurrentWhenActive="page">
    <app-icono nombre="home" [tamano]="22" /><span>Inicio</span>
  </a>
  <a class="tabbar__item" routerLink="/app/gastos" routerLinkActive="tabbar__item--activo" ariaCurrentWhenActive="page">
    <span class="tabbar__icono">
      <app-icono nombre="receipt" [tamano]="22" />
      @if (pendientes() > 0) {
        <span class="tabbar__punto">{{ pendientes() }}<span class="oculto-visual"> por aprobar</span></span>
      }
    </span>
    <span>Gastos</span>
  </a>
  <a class="tabbar__item" routerLink="/app/operacion" routerLinkActive="tabbar__item--activo" ariaCurrentWhenActive="page">
    <app-icono nombre="route" [tamano]="22" /><span>Operación</span>
  </a>
  <button type="button" class="tabbar__item" [class.tabbar__item--activo]="enCatalogos()" aria-haspopup="dialog" (click)="abrirMas()">
    <app-icono nombre="dots" [tamano]="22" /><span>Más</span>
  </button>
</nav>

<dialog #hojaMas class="hoja hoja--inferior" aria-labelledby="mas-titulo" (click)="cerrarSiFondo($event)">
  <div class="hoja__marco">
    <header class="hoja__cabecera">
      <h2 id="mas-titulo" class="hoja__titulo">Más</h2>
      <button type="button" class="btn btn--fantasma btn--icono" (click)="cerrarMas()" aria-label="Cerrar"><app-icono nombre="x" /></button>
    </header>
    <div class="hoja__cuerpo">
      <p class="hoja__grupo">Catálogos</p>
      <ul class="lista-enlaces">
        @for (m of catalogos; track m.slug) {
          <li>
            <a class="lista-enlaces__item" [routerLink]="rutaModulo(m)" (click)="cerrarMas()">
              <app-icono [nombre]="m.icono" [tamano]="20" />{{ m.etiqueta }}
              <app-icono class="lista-enlaces__flecha" nombre="chevron-right" [tamano]="16" />
            </a>
          </li>
        }
      </ul>
      <p class="hoja__grupo">Cuenta</p>
      <div class="lista-enlaces__cuenta">
        <span class="usuario__iniciales" aria-hidden="true">{{ siglas() }}</span>
        <span>{{ nombre() }}</span>
      </div>
      <button type="button" class="btn btn--secundario btn--bloque" (click)="salir()">
        <app-icono nombre="logout" [tamano]="18" />Cerrar sesión
      </button>
    </div>
  </div>
</dialog>
```

- [ ] **Step 7: El shell**

`frontend/src/app/shell/shell.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, OnInit, inject, viewChild } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ApiService } from '../core/services/api.service';
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
  private readonly store = inject(GastosStore);
  private readonly paleta = viewChild.required(PaletaComandosComponent);

  ngOnInit(): void {
    this.api.checkHealth();
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
```

`frontend/src/app/shell/shell.component.html`:

```html
<button type="button" class="saltar" (click)="saltarAlContenido()">Saltar al contenido</button>
<app-barra-superior (buscar)="abrirPaleta()" />
<main id="contenido" class="contenido" tabindex="-1">
  <router-outlet />
</main>
<app-barra-movil />
<app-paleta-comandos />
```

- [ ] **Step 8: Crear `src/styles/shell.css` e importarlo**

`frontend/src/styles/shell.css`:

```css
/* ══ Shell: barra superior de asfalto, cinta reflectiva, barra inferior y paleta ══ */
.barra {
  position: sticky;
  top: 0;
  z-index: var(--z-barra);
  background: var(--asfalto);
  color: #fff;
  view-transition-name: barra;
}

.barra__interior {
  display: flex;
  align-items: center;
  gap: var(--e-6);
  max-width: var(--contenido-max);
  height: var(--barra-h);
  margin: 0 auto;
  padding: 0 var(--margen);
}

.barra__marca {
  display: inline-flex;
  border-radius: var(--r-control);
  text-decoration: none;
}

.barra__nav {
  display: flex;
  align-self: stretch;
  align-items: stretch;
  gap: var(--e-1);
}

.barra__enlace {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: var(--e-2);
  padding: 0 var(--e-3);
  color: var(--asfalto-tinta);
  font-size: var(--t-14);
  font-weight: 600;
  text-decoration: none;
  transition: color 150ms ease;
}

.barra__enlace::after {
  content: '';
  position: absolute;
  left: var(--e-3);
  right: var(--e-3);
  bottom: 0;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: transparent;
}

.barra__enlace:hover,
.barra__enlace--activo {
  color: #fff;
}

.barra__enlace--activo::after {
  background: var(--rojo);
}

.barra__acciones {
  display: flex;
  align-items: center;
  gap: var(--e-2);
  margin-left: auto;
}

.barra__buscar {
  display: inline-flex;
  align-items: center;
  gap: var(--e-2);
  min-width: 240px;
  height: 36px;
  padding: 0 var(--e-2) 0 var(--e-3);
  border: 1px solid var(--asfalto-3);
  border-radius: var(--r-control);
  background: var(--asfalto-2);
  color: var(--asfalto-tinta);
  font-size: var(--t-14);
}

.barra__buscar:hover {
  border-color: #545a61;
  color: #fff;
}

.barra__buscar-texto {
  flex: 1;
  text-align: left;
}

.barra__servicio {
  display: inline-flex;
  align-items: center;
  gap: var(--e-2);
  height: 32px;
  padding: 0 var(--e-3);
  border: 0;
  border-radius: var(--r-pastilla);
  background: var(--rojo);
  color: #fff;
  font-size: var(--t-13);
  font-weight: 600;
}

.barra__titulo-movil {
  display: none;
}

/* Cinta reflectiva roja y blanca, como la de los camiones */
.cinta {
  height: 4px;
  background: repeating-linear-gradient(90deg, var(--rojo) 0 24px, var(--suelo) 24px 48px);
}

/* ── Usuario y menú ── */
.usuario {
  display: inline-flex;
  align-items: center;
  gap: var(--e-2);
  height: 36px;
  padding: 0 var(--e-2) 0 4px;
  border: 0;
  border-radius: var(--r-pastilla);
  background: transparent;
  color: #fff;
}

.usuario:hover {
  background: var(--asfalto-2);
}

.usuario__iniciales {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: var(--asfalto-3);
  color: #fff;
  font-size: var(--t-12);
  font-weight: 700;
  letter-spacing: 0.04em;
}

.usuario__nombre {
  max-width: 140px;
  overflow: hidden;
  font-size: var(--t-14);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.usuario__chevron {
  color: var(--asfalto-tinta);
}

.menu {
  position: fixed;
  inset: auto;
  top: calc(var(--barra-h) + 8px);
  right: max(var(--margen), calc((100vw - var(--contenido-max)) / 2 + var(--margen)));
  min-width: 220px;
  margin: 0;
  padding: var(--e-1);
  border: 1px solid var(--linea);
  border-radius: var(--r-panel);
  background: var(--panel);
  color: var(--tinta);
  box-shadow: var(--sombra-2);
  transform-origin: top right;
}

.menu__cabecera {
  display: grid;
  margin-bottom: var(--e-1);
  padding: var(--e-3);
  border-bottom: 1px solid var(--linea);
}

.menu__nombre {
  font-weight: 600;
}

.menu__rol {
  color: var(--tinta-2);
  font-size: var(--t-13);
  text-transform: capitalize;
}

.menu__item {
  display: flex;
  align-items: center;
  gap: var(--e-2);
  width: 100%;
  min-height: 40px;
  padding: 0 var(--e-3);
  border: 0;
  border-radius: var(--r-control);
  background: transparent;
  font-size: var(--t-14);
  text-align: left;
}

.menu__item:hover,
.menu__item:focus-visible {
  background: var(--panel-2);
}

@media (prefers-reduced-motion: no-preference) {
  .menu {
    opacity: 0;
    transform: scale(0.97);
    transition: opacity 120ms ease, transform 120ms var(--ease-out), display 120ms allow-discrete, overlay 120ms allow-discrete;
  }

  .menu:popover-open {
    opacity: 1;
    transform: scale(1);
    transition-duration: 160ms;
  }

  @starting-style {
    .menu:popover-open {
      opacity: 0;
      transform: scale(0.97);
    }
  }
}

/* ── Contenido ── */
.contenido {
  display: block;
  min-height: calc(100dvh - var(--barra-h) - 4px);
  outline: none;
}

/* ── Barra inferior (solo celular) ── */
.tabbar {
  display: none;
}

.lista-enlaces {
  display: grid;
}

.lista-enlaces__item {
  display: flex;
  align-items: center;
  gap: var(--e-3);
  min-height: 48px;
  padding: 0 var(--e-2);
  border-radius: var(--r-control);
  font-weight: 500;
  text-decoration: none;
}

.lista-enlaces__item:active {
  background: var(--panel-2);
}

.lista-enlaces__flecha {
  margin-left: auto;
  color: var(--tinta-3);
}

.lista-enlaces__cuenta {
  display: flex;
  align-items: center;
  gap: var(--e-3);
  margin-bottom: var(--e-3);
  padding: var(--e-2);
  font-weight: 600;
}

.lista-enlaces__cuenta .usuario__iniciales {
  background: var(--rojo);
}

@media (max-width: 1023px) {
  .barra__buscar {
    min-width: 0;
    padding: 0 var(--e-2);
  }

  .barra__buscar-texto,
  .barra__atajo,
  .usuario__nombre,
  .usuario__chevron {
    display: none;
  }
}

@media (max-width: 767px) {
  .barra__interior {
    gap: var(--e-3);
  }

  .barra__nav,
  .barra .marca__texto {
    display: none;
  }

  .barra__titulo-movil {
    display: block;
    font-size: var(--t-16);
    font-weight: 600;
  }

  .barra__servicio {
    justify-content: center;
    width: 32px;
    padding: 0;
  }

  .barra__servicio span {
    display: none;
  }

  .tabbar {
    position: fixed;
    z-index: var(--z-tabbar);
    left: 0;
    right: 0;
    bottom: 0;
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    height: calc(var(--tabbar-h) + env(safe-area-inset-bottom));
    padding-bottom: env(safe-area-inset-bottom);
    border-top: 1px solid var(--asfalto-3);
    background: var(--asfalto);
  }

  .tabbar__item {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    border: 0;
    background: transparent;
    color: var(--asfalto-tinta);
    font-size: var(--t-12);
    font-weight: 600;
    text-decoration: none;
    touch-action: manipulation;
  }

  .tabbar__item--activo {
    color: #fff;
  }

  .tabbar__item--activo .icono {
    color: var(--rojo);
  }

  .tabbar__icono {
    position: relative;
    display: inline-flex;
  }

  .tabbar__punto {
    position: absolute;
    top: -6px;
    right: -12px;
    display: grid;
    place-items: center;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    border-radius: var(--r-pastilla);
    background: var(--via);
    color: var(--asfalto);
    font-size: 11px;
    font-weight: 700;
  }
}

/* ── Paleta de comandos (sin animación a propósito) ── */
.paleta {
  width: min(640px, calc(100vw - 24px));
  margin: 12vh auto auto;
}

.paleta::backdrop {
  background: rgb(30 33 36 / 0.4);
}

.paleta__marco {
  overflow: hidden;
  border-radius: var(--r-panel);
  background: var(--panel);
  box-shadow: var(--sombra-2);
}

.paleta__campo {
  display: flex;
  align-items: center;
  gap: var(--e-3);
  padding: 0 var(--e-4);
  border-bottom: 1px solid var(--linea);
  color: var(--tinta-3);
}

.paleta__input {
  flex: 1;
  height: 56px;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--tinta);
  font-size: var(--t-16);
}

.paleta__esc {
  color: var(--tinta-3);
}

.paleta__lista {
  max-height: min(420px, 60dvh);
  padding: var(--e-2);
  overflow-y: auto;
}

.paleta__grupo + .paleta__grupo {
  margin-top: var(--e-2);
}

.paleta__grupo-titulo {
  padding: var(--e-2) var(--e-3) var(--e-1);
  color: var(--tinta-3);
  font-size: var(--t-12);
  font-weight: 600;
}

.paleta__opcion {
  display: flex;
  align-items: center;
  gap: var(--e-3);
  min-height: 44px;
  padding: 0 var(--e-3);
  border-radius: var(--r-control);
  color: var(--tinta-2);
  cursor: pointer;
}

.paleta__opcion--activa {
  background: var(--panel-2);
  color: var(--tinta);
}

.paleta__opcion--activa .icono {
  color: var(--rojo);
}

.paleta__etiqueta {
  color: var(--tinta);
  font-weight: 600;
}

.paleta__detalle {
  margin-left: auto;
  color: var(--tinta-3);
  font-size: var(--t-13);
}

.paleta__vacio {
  padding: var(--e-6) var(--e-3);
  color: var(--tinta-2);
  text-align: center;
}

@media (max-width: 767px) {
  .paleta {
    margin-top: var(--e-4);
  }

  .paleta__detalle {
    display: none;
  }
}

/* ── Página no encontrada ── */
.no-encontrada {
  display: grid;
  justify-items: start;
  align-content: center;
  gap: var(--e-4);
  max-width: 560px;
  min-height: 100dvh;
  margin: 0 auto;
  padding: var(--e-8) var(--margen);
}

.no-encontrada__codigo {
  color: var(--tinta-3);
  font-family: var(--fuente-mono);
  font-size: var(--t-14);
}

.no-encontrada h1 {
  font-size: var(--t-32);
  font-weight: 700;
}

.no-encontrada p {
  color: var(--tinta-2);
}
```

En `frontend/src/styles.css`, agrega al final:

```css
@import './styles/shell.css';
```

- [ ] **Step 9: Ejecutar y verificar que pasan**

Run: `npx ng test --watch=false --include src/app/core/guards/auth.guard.spec.ts --include "src/app/shell/**/*.spec.ts" --include src/app/app.routes.spec.ts`
Expected: PASS (8 pruebas).

- [ ] **Step 10: Checkpoint con la app corriendo (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso. En la salida, cada pantalla aparece como *lazy chunk* (inicio, gastos-lista, gasto-form, gasto-detalle, crud-page, landing, login, register, no-encontrada y shell).

Luego levanta la app (`npx ng serve --port 4200`) y abre `http://localhost:4200/dashboard` en el navegador. Sin sesión debe terminar en `/login?volver=%2Fapp%2Finicio`; con sesión, en `/app/inicio` con la barra superior. La landing, el login y el registro aún son los viejos (sin estilo): las Tareas 18 y 19 los rehacen.

---
### Task 18: Landing (`/`)

**Files:**
- Modify (reemplazo completo): `frontend/src/app/features/landing/landing.component.ts`
- Modify (reemplazo completo): `frontend/src/app/features/landing/landing.component.html`
- Delete: `frontend/src/app/features/landing/landing.component.css`
- Create: `frontend/src/styles/landing.css`
- Modify: `frontend/src/styles.css` (agregar import)
- Test: `frontend/src/app/features/landing/landing.component.spec.ts`

**Interfaces:**
- Consumes:
  - `GastosApiService.get`
  - `AuthService.isAuth()`
  - `contarPorEstado` (Tarea 3)
  - componentes: `MarcaComponent`, `CamionComponent`, `IconoComponent` e `InsigniaEstadoComponent`
  - `HTTP_TIMEOUT_MS`
- Produces: `LandingComponent`, selector `app-landing` (lo carga la ruta `''`).

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/features/landing/landing.component.spec.ts`:

```ts
import { describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { Observable, of, throwError } from 'rxjs';
import { LandingComponent } from './landing.component';
import { GastosApiService } from '../../core/services/gastos-api.service';
import { AuthService } from '../../core/services/auth.service';
import { Gasto } from '../../core/models/gasto.model';

function gasto(id: number, estado: Gasto['estado']): Gasto {
  return { id, estado, monto: 10, viaje_id: 1, cliente_id: 1, fecha: '2026-10-01T00:00:00Z', creado_por: 'Ana', creado_en: '2026-10-01T12:00:00Z' };
}

function montar(get: (ruta: string) => Observable<unknown>, conSesion = false): HTMLElement {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([]),
      { provide: GastosApiService, useValue: { get } },
      { provide: AuthService, useValue: { isAuth: () => conSesion } },
    ],
  });
  const f = TestBed.createComponent(LandingComponent);
  f.detectChanges();
  return f.nativeElement;
}

describe('LandingComponent', () => {
  it('muestra los conteos reales del servicio de gastos', () => {
    const el = montar(ruta => (ruta === '/gastos' ? of([gasto(1, 'Registrado'), gasto(2, 'Registrado'), gasto(3, 'Pagado')]) : of([{ id: 9 }])));
    expect([...el.querySelectorAll('.via__numero')].map(n => n.textContent?.trim())).toEqual(['2', '1', '0']);
    expect(el.querySelector('.via__pendientes')?.textContent).toContain('1 comprobante por aprobar');
  });

  it('si el servicio no responde no inventa cifras', () => {
    const el = montar(() => throwError(() => new Error('sin red')));
    expect(el.querySelector('.via__nota')?.textContent?.trim()).toBe('Los indicadores no están disponibles en este momento.');
    expect(el.querySelector('.via__pendientes')).toBeNull();
  });

  it('sin sesión ofrece Ingresar, con sesión Ir a Inicio, y no usa raya larga', () => {
    const sin = montar(() => of([]));
    expect(sin.querySelector('.portada__acciones a')?.getAttribute('href')).toBe('/login');
    expect(sin.textContent).not.toMatch(/[—–]/);
    TestBed.resetTestingModule();
    const con = montar(() => of([]), true);
    expect(con.querySelector('.portada__acciones a')?.getAttribute('href')).toBe('/app/inicio');
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `npx ng test --watch=false --include src/app/features/landing/landing.component.spec.ts`
Expected: FAIL. La landing vieja no tiene `.via__numero` ni `.portada__acciones`.

- [ ] **Step 3: Implementar**

Reemplaza `frontend/src/app/features/landing/landing.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, OnInit, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { forkJoin, timeout } from 'rxjs';
import { Comprobante, EstadoGasto, Gasto } from '../../core/models/gasto.model';
import { AuthService } from '../../core/services/auth.service';
import { GastosApiService } from '../../core/services/gastos-api.service';
import { contarPorEstado } from '../../core/state/gastos.logic';
import { HTTP_TIMEOUT_MS } from '../../core/utils/error-messages';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { InsigniaEstadoComponent } from '../../shared/ui/insignia-estado/insignia-estado.component';
import { CamionComponent } from '../../shared/ui/marca/camion.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';

@Component({
  selector: 'app-landing',
  imports: [RouterLink, MarcaComponent, CamionComponent, IconoComponent, InsigniaEstadoComponent],
  templateUrl: './landing.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LandingComponent implements OnInit {
  protected readonly auth = inject(AuthService);
  private readonly gastosApi = inject(GastosApiService);

  /** Solo indicadores reales del servicio de gastos; mientras cargan se muestra «…». */
  protected readonly conteo = signal<Record<EstadoGasto, number> | null>(null);
  protected readonly pendientes = signal<number | null>(null);
  protected readonly sinDatos = signal(false);
  protected readonly hitos: { estado: EstadoGasto; texto: string }[] = [
    { estado: 'Registrado', texto: 'registrados' },
    { estado: 'Pagado', texto: 'pagados' },
    { estado: 'Sustentado', texto: 'sustentados' },
  ];

  ngOnInit(): void {
    forkJoin({
      gastos: this.gastosApi.get<Gasto[]>('/gastos'),
      pendientes: this.gastosApi.get<Comprobante[]>('/comprobantes?estado=Pendiente'),
    })
      .pipe(timeout(HTTP_TIMEOUT_MS))
      .subscribe({
        next: ({ gastos, pendientes }) => {
          this.conteo.set(contarPorEstado(gastos ?? []));
          this.pendientes.set((pendientes ?? []).length);
        },
        error: () => this.sinDatos.set(true),
      });
  }
}
```

Reemplaza `frontend/src/app/features/landing/landing.component.html`:

```html
<div class="landing">
  <header class="landing__barra">
    <div class="landing__barra-interior">
      <a class="landing__marca" routerLink="/" aria-label="LIZAR SRL, inicio"><app-marca /></a>
      @if (auth.isAuth()) {
        <a class="btn btn--primario" routerLink="/app/inicio">Ir a Inicio<app-icono nombre="arrow-right" [tamano]="16" /></a>
      } @else {
        <a class="btn btn--primario" routerLink="/login">Ingresar<app-icono nombre="arrow-right" [tamano]="16" /></a>
      }
    </div>
    <div class="cinta" aria-hidden="true"></div>
  </header>

  <main>
    <section class="portada" aria-labelledby="t-portada">
      <div class="portada__texto">
        <h1 id="t-portada" class="portada__titulo">Gastos de viaje con su comprobante, de la ruta a la oficina.</h1>
        <p class="portada__bajada">Registra cada gasto adicional, adjunta su comprobante desde el celular y apruébalo desde la oficina.</p>
        <div class="portada__acciones">
          @if (auth.isAuth()) {
            <a class="btn btn--primario btn--lg" routerLink="/app/inicio">Ir a Inicio</a>
          } @else {
            <a class="btn btn--primario btn--lg" routerLink="/login">Ingresar</a>
            <a class="enlace" routerLink="/register">Crear usuario</a>
          }
        </div>
      </div>

      <div class="via" role="group" aria-label="Estado actual de los gastos">
        <span class="via__linea" aria-hidden="true"></span>
        <span class="via__camion" aria-hidden="true"><app-camion /></span>
        <ol class="via__hitos">
          @for (h of hitos; track h.estado) {
            <li class="via__hito" [attr.data-estado]="h.estado">
              <span class="via__poste" aria-hidden="true"></span>
              <span class="via__numero">{{ conteo()?.[h.estado] ?? '…' }}</span>
              <span class="via__texto">{{ h.texto }}</span>
            </li>
          }
        </ol>
        @if (sinDatos()) {
          <p class="via__nota">Los indicadores no están disponibles en este momento.</p>
        } @else {
          <p class="via__pendientes">
            <span class="via__senal" aria-hidden="true"><span>!</span></span>
            {{ pendientes() ?? '…' }} {{ pendientes() === 1 ? 'comprobante por aprobar' : 'comprobantes por aprobar' }}
          </p>
        }
      </div>
    </section>

    <section class="como" aria-labelledby="t-como">
      <h2 id="t-como" class="landing__h2">Cómo avanza un gasto</h2>
      <ol class="como__pasos">
        <li class="como__paso" data-estado="Registrado">
          <span class="como__poste" aria-hidden="true"></span>
          <h3>Registra</h3>
          <p>Anota el gasto adicional con su viaje, cliente, monto y fecha.</p>
          <app-insignia-estado estado="Registrado" />
        </li>
        <li class="como__paso" data-estado="Pagado">
          <span class="como__poste" aria-hidden="true"></span>
          <h3>Adjunta</h3>
          <p>Sube la foto o el PDF del comprobante, también desde el celular.</p>
          <app-insignia-estado estado="Pagado" />
        </li>
        <li class="como__paso" data-estado="Sustentado">
          <span class="como__poste" aria-hidden="true"></span>
          <h3>Aprueba</h3>
          <p>Logística revisa el comprobante y lo aprueba, o lo rechaza con un motivo.</p>
          <app-insignia-estado estado="Sustentado" />
        </li>
      </ol>
    </section>

    <section class="lugares" aria-labelledby="t-lugares">
      <figure class="lugares__foto">
        <img
          src="https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=1400&q=75&auto=format&fit=crop"
          srcset="
            https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=640&q=75&auto=format&fit=crop 640w,
            https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=960&q=75&auto=format&fit=crop 960w,
            https://images.unsplash.com/photo-1519003722824-194d4455a60c?w=1400&q=75&auto=format&fit=crop 1400w
          "
          sizes="(min-width: 1024px) 56vw, 100vw"
          width="1400"
          height="928"
          alt="Camión de carga avanzando por una carretera de montaña con postes y señales de tránsito"
          loading="lazy"
          decoding="async"
        />
      </figure>
      <div class="lugares__texto">
        <h2 id="t-lugares" class="landing__h2">Desde la oficina y desde la ruta</h2>
        <div class="lugares__bloque">
          <app-icono nombre="building" [tamano]="20" />
          <h3>En la oficina</h3>
          <p>Logística ve qué gastos esperan revisión y aprueba cada comprobante con un clic.</p>
        </div>
        <div class="lugares__bloque">
          <app-icono nombre="truck" [tamano]="20" />
          <h3>En la ruta</h3>
          <p>El conductor busca su gasto en el celular y adjunta la foto del comprobante en el momento.</p>
        </div>
      </div>
    </section>
  </main>

  <footer class="landing__pie">
    <span>LIZAR SRL · Trujillo, Perú</span>
    <span>© 2026</span>
  </footer>
</div>
```

Borra `frontend/src/app/features/landing/landing.component.css`:

```bash
rm "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend/src/app/features/landing/landing.component.css"
```

- [ ] **Step 4: Crear `src/styles/landing.css` e importarlo**

`frontend/src/styles/landing.css`:

```css
/* ══ Landing (modo Persuadir dentro del mismo mundo) ══ */
.landing {
  display: flex;
  flex-direction: column;
  min-height: 100dvh;
  background: var(--suelo);
}

.landing main {
  flex: 1;
}

.landing__barra {
  position: sticky;
  top: 0;
  z-index: var(--z-barra);
  background: var(--asfalto);
  color: #fff;
}

.landing__barra-interior {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--e-4);
  max-width: var(--contenido-max);
  height: 64px;
  margin: 0 auto;
  padding: 0 var(--margen);
}

.landing__marca {
  display: inline-flex;
  text-decoration: none;
}

.landing__h2 {
  max-width: 22ch;
  font-size: var(--t-32);
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

/* ── Portada: cabe en la primera vista ── */
.portada {
  display: grid;
  grid-template-columns: 7fr 5fr;
  align-items: center;
  gap: var(--e-12);
  max-width: var(--contenido-max);
  min-height: calc(100dvh - 68px);
  margin: 0 auto;
  padding: var(--e-12) var(--margen);
}

.portada__titulo {
  font-size: 2.5rem;
  font-weight: 800;
  letter-spacing: -0.03em;
  line-height: 1.05;
}

.portada__bajada {
  max-width: 46ch;
  margin-top: var(--e-5);
  color: var(--tinta-2);
  font-size: var(--t-18);
  line-height: 1.55;
}

.portada__acciones {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--e-5);
  margin-top: var(--e-8);
}

/* La carretera con los conteos reales */
.via {
  position: relative;
  padding: var(--e-8) var(--e-6) var(--e-6) 0;
  overflow: hidden;
  border-radius: var(--r-panel);
  background: var(--asfalto);
  color: #fff;
}

.via__linea {
  position: absolute;
  top: var(--e-6);
  bottom: 88px;
  left: 43px;
  border-left: 3px dashed rgb(255 255 255 / 0.5);
}

.via__camion {
  position: relative;
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  margin-left: 26px;
  padding: 7px;
  border-radius: 8px;
  background: var(--rojo);
  color: #fff;
}

.via__hitos {
  display: grid;
  gap: var(--e-6);
  margin-top: var(--e-6);
}

.via__hito {
  position: relative;
  display: grid;
  grid-template-columns: 88px auto 1fr;
  align-items: baseline;
  column-gap: var(--e-3);
}

.via__hito[data-estado='Registrado'] {
  --c: var(--registrado);
}

.via__hito[data-estado='Pagado'] {
  --c: var(--pagado);
}

.via__hito[data-estado='Sustentado'] {
  --c: var(--sustentado);
}

.via__poste {
  align-self: center;
  justify-self: center;
  width: 22px;
  height: 22px;
  border: 2px solid #fff;
  border-radius: 3px;
  background: var(--c);
}

.via__numero {
  font-size: var(--t-40);
  font-weight: 700;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.via__texto {
  color: var(--asfalto-tinta);
  font-weight: 600;
}

.via__pendientes {
  display: flex;
  align-items: center;
  gap: var(--e-3);
  margin: var(--e-8) 0 0 var(--e-6);
  padding: var(--e-3) var(--e-4);
  border-radius: var(--r-control);
  background: var(--via);
  color: var(--asfalto);
  font-weight: 700;
}

/* Señal preventiva: rombo amarillo con borde negro */
.via__senal {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  border: 2px solid var(--asfalto);
  border-radius: 3px;
  transform: rotate(45deg);
}

.via__senal > span {
  font-size: 12px;
  font-weight: 800;
  transform: rotate(-45deg);
}

.via__nota {
  margin: var(--e-6) 0 0 var(--e-6);
  color: var(--asfalto-tinta);
  font-size: var(--t-14);
}

@media (prefers-reduced-motion: no-preference) {
  .via__linea {
    animation: trazar-via 700ms var(--ease-in-out) 120ms both;
  }

  .via__hito,
  .via__pendientes {
    animation: aparecer-hito 400ms var(--ease-out) both;
  }

  .via__hito:nth-child(1) {
    animation-delay: 360ms;
  }

  .via__hito:nth-child(2) {
    animation-delay: 420ms;
  }

  .via__hito:nth-child(3) {
    animation-delay: 480ms;
  }

  .via__pendientes {
    animation-delay: 560ms;
  }
}

@keyframes trazar-via {
  from {
    clip-path: inset(0 0 100% 0);
  }

  to {
    clip-path: inset(0 0 0 0);
  }
}

@keyframes aparecer-hito {
  from {
    opacity: 0;
    transform: translateY(8px);
  }

  to {
    opacity: 1;
    transform: none;
  }
}

/* ── Cómo avanza un gasto ── */
.como {
  max-width: var(--contenido-max);
  margin: 0 auto;
  padding: var(--e-16) var(--margen);
  border-top: 1px solid var(--linea);
}

.como__pasos {
  position: relative;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--e-8);
  margin-top: var(--e-10);
}

.como__pasos::before {
  content: '';
  position: absolute;
  top: 11px;
  left: 12px;
  right: 0;
  border-top: 2px dashed var(--linea-2);
}

.como__paso {
  position: relative;
  display: grid;
  justify-items: start;
  gap: var(--e-3);
}

.como__paso[data-estado='Registrado'] {
  --c: var(--registrado);
}

.como__paso[data-estado='Pagado'] {
  --c: var(--pagado);
}

.como__paso[data-estado='Sustentado'] {
  --c: var(--sustentado);
}

.como__poste {
  width: 24px;
  height: 24px;
  border: 3px solid var(--suelo);
  border-radius: 3px;
  background: var(--c);
  box-shadow: 0 0 0 2px var(--c);
}

.como__paso h3 {
  margin-top: var(--e-2);
  font-size: var(--t-24);
  font-weight: 700;
}

.como__paso p {
  max-width: 32ch;
  color: var(--tinta-2);
}

/* ── Desde la oficina y desde la ruta ── */
.lugares {
  display: grid;
  grid-template-columns: 7fr 5fr;
  align-items: center;
  gap: var(--e-10);
  max-width: var(--contenido-max);
  margin: 0 auto;
  padding: 0 var(--margen) var(--e-16);
}

.lugares__foto {
  margin: 0;
  overflow: hidden;
  aspect-ratio: 3 / 2;
  border-radius: var(--r-panel);
  background: var(--panel-2);
}

.lugares__foto img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.lugares__texto {
  display: grid;
  gap: var(--e-6);
}

.lugares__bloque {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--e-1) var(--e-3);
}

.lugares__bloque > .icono {
  grid-row: 1 / span 2;
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: var(--r-control);
  background: var(--asfalto);
  color: #fff;
}

.lugares__bloque h3 {
  font-size: var(--t-18);
  font-weight: 700;
}

.lugares__bloque p {
  grid-column: 2;
  color: var(--tinta-2);
}

.landing__pie {
  display: flex;
  justify-content: space-between;
  gap: var(--e-4);
  width: 100%;
  max-width: var(--contenido-max);
  margin: 0 auto;
  padding: var(--e-6) var(--margen);
  border-top: 1px solid var(--linea);
  color: var(--tinta-2);
  font-size: var(--t-14);
}

@media (max-width: 1023px) {
  .portada,
  .lugares {
    grid-template-columns: 1fr;
  }

  .portada {
    min-height: 0;
    gap: var(--e-10);
    padding-top: var(--e-10);
  }

  .portada__titulo {
    font-size: var(--t-32);
  }
}

@media (max-width: 767px) {
  .portada__titulo {
    font-size: 1.75rem;
  }

  .portada__bajada {
    font-size: var(--t-16);
  }

  .landing__h2 {
    font-size: var(--t-24);
  }

  .como__pasos {
    grid-template-columns: 1fr;
    gap: var(--e-6);
    padding-left: var(--e-10);
  }

  .como__pasos::before {
    top: 0;
    bottom: 0;
    left: 11px;
    right: auto;
    border-top: 0;
    border-left: 2px dashed var(--linea-2);
  }

  .como__poste {
    position: absolute;
    top: 2px;
    left: calc(-1 * var(--e-10));
  }
}
```

En `frontend/src/styles.css`, agrega al final:

```css
@import './styles/landing.css';
```

- [ ] **Step 5: Ejecutar y verificar que pasa**

Run: `npx ng test --watch=false --include src/app/features/landing/landing.component.spec.ts`
Expected: PASS (3 pruebas).

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso.

---
### Task 19: Iniciar sesión y crear usuario

**Files:**
- Create: `frontend/src/app/shared/ui/panel-acceso/panel-acceso.component.ts`
- Modify (reemplazo completo): `frontend/src/app/features/login/login.component.ts` y `login.component.html`
- Modify (reemplazo completo): `frontend/src/app/features/register/register.component.ts` y `register.component.html`
- Delete: `frontend/src/app/features/login/login.component.css` y `frontend/src/app/features/register/register.component.css`
- Create: `frontend/src/styles/acceso.css`
- Modify: `frontend/src/styles.css` (agregar import)
- Test: `frontend/src/app/features/login/login.component.spec.ts`
- Test: `frontend/src/app/features/register/register.component.spec.ts`

**Interfaces:**
- Consumes:
  - de `AuthService`: `login()` y `register()` (sin cambios)
  - `destinoTrasLogin` (Tarea 4)
  - componentes: `MarcaComponent` e `IconoComponent`
  - las clases `.campo`, `.control-con-icono`, `.alerta`, `.girador` y `.casilla` (Tarea 7)
- Produces:
  - `PanelAccesoComponent`, selector `app-panel-acceso`
  - `LoginComponent`, que tras iniciar sesión navega a `destinoTrasLogin(?volver)`
  - `RegisterComponent`

- [ ] **Step 1: Escribir las pruebas que fallan**

`frontend/src/app/features/login/login.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';
import { Observable, of } from 'rxjs';
import { LoginComponent } from './login.component';
import { AuthService } from '../../core/services/auth.service';

async function abrir(url: string, login: () => Observable<unknown> = vi.fn(() => of({}))) {
  TestBed.configureTestingModule({
    providers: [
      provideRouter([{ path: 'login', component: LoginComponent }, { path: 'register', children: [] }]),
      { provide: AuthService, useValue: { login } },
    ],
  });
  const harness = await RouterTestingHarness.create();
  await harness.navigateByUrl(url, LoginComponent);
  return { harness, el: harness.routeNativeElement as HTMLElement, login };
}

function escribir(el: HTMLElement, id: string, valor: string): void {
  const campo = el.querySelector(`#${id}`) as HTMLInputElement;
  campo.value = valor;
  campo.dispatchEvent(new Event('input'));
}

describe('LoginComponent', () => {
  it('pide usuario y contraseña', async () => {
    const { harness, el } = await abrir('/login');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    harness.detectChanges();
    expect(el.querySelector('.alerta--error')?.textContent?.trim()).toBe('Ingresa usuario y contraseña');
  });

  it('tras iniciar sesión vuelve a la pantalla que se pidió', async () => {
    const { el, login } = await abrir('/login?volver=%2Fapp%2Fgastos%3Festado%3Dpor-aprobar');
    const navegar = vi.spyOn(TestBed.inject(Router), 'navigateByUrl').mockResolvedValue(true);
    escribir(el, 'username', 'admin');
    escribir(el, 'password', 'admin123');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    expect(login).toHaveBeenCalledWith({ username: 'admin', password: 'admin123' });
    expect(navegar).toHaveBeenCalledWith('/app/gastos?estado=por-aprobar');
  });

  it('avisa que el usuario se creó y el botón de la contraseña es accesible', async () => {
    const { harness, el } = await abrir('/login?registered=1');
    expect(el.querySelector('.alerta--exito')?.textContent?.trim()).toBe('Usuario creado. Ya puedes iniciar sesión.');
    const ojo = el.querySelector('.control-con-icono__boton') as HTMLButtonElement;
    expect(ojo.getAttribute('aria-label')).toBe('Mostrar contraseña');
    ojo.click();
    harness.detectChanges();
    expect((el.querySelector('#password') as HTMLInputElement).type).toBe('text');
    expect(ojo.getAttribute('aria-label')).toBe('Ocultar contraseña');
  });
});
```

`frontend/src/app/features/register/register.component.spec.ts`:

```ts
import { describe, expect, it, vi } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { RouterTestingHarness } from '@angular/router/testing';
import { of } from 'rxjs';
import { RegisterComponent } from './register.component';
import { AuthService } from '../../core/services/auth.service';

describe('RegisterComponent', () => {
  it('valida que las contraseñas coincidan y al crear vuelve al login con aviso', async () => {
    const register = vi.fn(() => of({}));
    TestBed.configureTestingModule({
      providers: [
        provideRouter([{ path: 'register', component: RegisterComponent }, { path: 'login', children: [] }]),
        { provide: AuthService, useValue: { register } },
      ],
    });
    const harness = await RouterTestingHarness.create();
    await harness.navigateByUrl('/register', RegisterComponent);
    const el = harness.routeNativeElement as HTMLElement;
    const escribir = (id: string, valor: string) => {
      const campo = el.querySelector(`#${id}`) as HTMLInputElement;
      campo.value = valor;
      campo.dispatchEvent(new Event('input'));
    };
    escribir('nombre', 'Ana Quispe');
    escribir('username', 'aquispe');
    escribir('email', 'ana@lizar.pe');
    escribir('password', 'secreto1');
    escribir('confirmPassword', 'otro');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    harness.detectChanges();
    expect(el.querySelector('.alerta--error')?.textContent?.trim()).toBe('Las contraseñas no coinciden');

    const navegar = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    escribir('confirmPassword', 'secreto1');
    el.querySelector('form')!.dispatchEvent(new Event('submit'));
    expect(register).toHaveBeenCalledWith({ username: 'aquispe', email: 'ana@lizar.pe', nombre: 'Ana Quispe', rol: 'operador', password: 'secreto1' });
    expect(navegar).toHaveBeenCalledWith(['/login'], { queryParams: { registered: '1' } });
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `npx ng test --watch=false --include "src/app/features/{login,register}/*.spec.ts"`
Expected: FAIL. El login viejo no lee `volver` ni muestra el aviso de usuario creado; el botón de la contraseña no tiene `aria-label`, y no existen las clases `.alerta--error` ni `.alerta--exito`.

- [ ] **Step 3: Panel de marca compartido**

`frontend/src/app/shared/ui/panel-acceso/panel-acceso.component.ts`:

```ts
import { ChangeDetectionStrategy, Component } from '@angular/core';
import { MarcaComponent } from '../marca/marca.component';

/** Panel de asfalto del login y el registro: la marca y la carretera de los tres hitos. */
@Component({
  selector: 'app-panel-acceso',
  imports: [MarcaComponent],
  template: `
    <app-marca />
    <ol class="acceso__via" aria-hidden="true">
      <li class="acceso__hito" data-estado="Registrado">Registrado</li>
      <li class="acceso__hito" data-estado="Pagado">Pagado</li>
      <li class="acceso__hito" data-estado="Sustentado">Sustentado</li>
    </ol>
    <p class="acceso__lema">Gastos de viaje con su comprobante, de la ruta a la oficina.</p>
    <div class="cinta acceso__cinta" aria-hidden="true"></div>
  `,
  host: { class: 'acceso__marca' },
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PanelAccesoComponent {}
```

- [ ] **Step 4: Iniciar sesión**

Reemplaza `frontend/src/app/features/login/login.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { destinoTrasLogin } from '../../core/navigation/navegacion';
import { AuthService } from '../../core/services/auth.service';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { PanelAccesoComponent } from '../../shared/ui/panel-acceso/panel-acceso.component';

@Component({
  selector: 'app-login',
  imports: [FormsModule, RouterLink, MarcaComponent, IconoComponent, PanelAccesoComponent],
  templateUrl: './login.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly ruta = inject(ActivatedRoute);

  protected username = '';
  protected password = '';
  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly showPass = signal(false);
  protected readonly registrado = this.ruta.snapshot.queryParamMap.get('registered') === '1';

  protected submit(): void {
    if (!this.username.trim() || !this.password) {
      this.error.set('Ingresa usuario y contraseña');
      return;
    }
    this.error.set('');
    this.loading.set(true);
    this.auth.login({ username: this.username, password: this.password }).subscribe({
      next: () => this.router.navigateByUrl(destinoTrasLogin(this.ruta.snapshot.queryParamMap.get('volver'))),
      error: e => {
        this.error.set(e?.error?.detail ?? 'Credenciales incorrectas');
        this.loading.set(false);
      },
    });
  }
}
```

Reemplaza `frontend/src/app/features/login/login.component.html`:

```html
<div class="acceso">
  <app-panel-acceso />
  <header class="acceso__cabecera-movil">
    <div class="acceso__cabecera-movil-interior"><app-marca /></div>
    <div class="cinta" aria-hidden="true"></div>
  </header>

  <main class="acceso__principal">
    <div class="acceso__tarjeta">
      <h1 class="acceso__titulo">Iniciar sesión</h1>
      <p class="acceso__bajada">Ingresa tus credenciales para acceder al sistema</p>

      @if (registrado) {
        <p class="alerta alerta--exito" role="status">
          <app-icono nombre="circle-check" [tamano]="18" /><span>Usuario creado. Ya puedes iniciar sesión.</span>
        </p>
      }
      @if (error()) {
        <p class="alerta alerta--error" role="alert">
          <app-icono nombre="alert-triangle" [tamano]="18" /><span>{{ error() }}</span>
        </p>
      }

      <form class="formulario" (ngSubmit)="submit()" novalidate>
        <div class="campo">
          <label class="campo__etiqueta" for="username">Usuario</label>
          <div class="control-con-icono">
            <app-icono nombre="user" [tamano]="18" />
            <input class="control" id="username" name="username" type="text" autocomplete="username" placeholder="Ej: admin" [(ngModel)]="username" [disabled]="loading()" />
          </div>
        </div>
        <div class="campo">
          <label class="campo__etiqueta" for="password">Contraseña</label>
          <div class="control-con-icono">
            <app-icono nombre="lock" [tamano]="18" />
            <input class="control" id="password" name="password" [type]="showPass() ? 'text' : 'password'" autocomplete="current-password" [(ngModel)]="password" [disabled]="loading()" />
            <button
              type="button"
              class="control-con-icono__boton"
              [attr.aria-label]="showPass() ? 'Ocultar contraseña' : 'Mostrar contraseña'"
              [attr.aria-pressed]="showPass()"
              (click)="showPass.set(!showPass())"
            >
              <app-icono [nombre]="showPass() ? 'eye-off' : 'eye'" [tamano]="18" />
            </button>
          </div>
        </div>
        <button class="btn btn--primario btn--lg btn--bloque" type="submit" [disabled]="loading()">
          @if (loading()) {
            <span class="girador" aria-hidden="true"></span>Verificando…
          } @else {
            <app-icono nombre="login-2" [tamano]="18" />Ingresar al sistema
          }
        </button>
      </form>

      <p class="acceso__pista">Usuario por defecto: <code>admin</code> / <code>admin123</code></p>
      <p class="acceso__enlace">¿Necesitas una cuenta? <a routerLink="/register">Crear usuario</a></p>
    </div>
    <p class="acceso__pie">LIZAR SRL · Trujillo, Perú</p>
  </main>
</div>
```

- [ ] **Step 5: Crear usuario**

Reemplaza `frontend/src/app/features/register/register.component.ts`:

```ts
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { IconoComponent } from '../../shared/ui/icono/icono.component';
import { MarcaComponent } from '../../shared/ui/marca/marca.component';
import { PanelAccesoComponent } from '../../shared/ui/panel-acceso/panel-acceso.component';

@Component({
  selector: 'app-register',
  imports: [FormsModule, RouterLink, MarcaComponent, IconoComponent, PanelAccesoComponent],
  templateUrl: './register.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RegisterComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  protected username = '';
  protected email = '';
  protected nombre = '';
  protected rol: 'admin' | 'operador' = 'operador';
  protected password = '';
  protected confirmPassword = '';

  protected readonly loading = signal(false);
  protected readonly error = signal('');
  protected readonly showPass = signal(false);

  protected submit(): void {
    if (!this.username.trim() || !this.email.trim() || !this.nombre.trim() || !this.password) {
      this.error.set('Completa todos los campos obligatorios');
      return;
    }
    if (this.password.length < 6) {
      this.error.set('La contraseña debe tener al menos 6 caracteres');
      return;
    }
    if (this.password !== this.confirmPassword) {
      this.error.set('Las contraseñas no coinciden');
      return;
    }

    this.error.set('');
    this.loading.set(true);
    this.auth
      .register({ username: this.username.trim(), email: this.email.trim(), nombre: this.nombre.trim(), rol: this.rol, password: this.password })
      .subscribe({
        next: () => this.router.navigate(['/login'], { queryParams: { registered: '1' } }),
        error: e => {
          this.error.set(e?.error?.detail ?? 'No se pudo crear el usuario');
          this.loading.set(false);
        },
      });
  }
}
```

Reemplaza `frontend/src/app/features/register/register.component.html`:

```html
<div class="acceso">
  <app-panel-acceso />
  <header class="acceso__cabecera-movil">
    <div class="acceso__cabecera-movil-interior"><app-marca /></div>
    <div class="cinta" aria-hidden="true"></div>
  </header>

  <main class="acceso__principal">
    <div class="acceso__tarjeta acceso__tarjeta--ancha">
      <h1 class="acceso__titulo">Crear usuario</h1>
      <p class="acceso__bajada">Registra una cuenta para acceder al sistema</p>

      @if (error()) {
        <p class="alerta alerta--error" role="alert">
          <app-icono nombre="alert-triangle" [tamano]="18" /><span>{{ error() }}</span>
        </p>
      }

      <form class="formulario" (ngSubmit)="submit()" novalidate>
        <div class="formulario__fila">
          <div class="campo">
            <label class="campo__etiqueta" for="nombre">Nombre</label>
            <div class="control-con-icono">
              <app-icono nombre="user" [tamano]="18" />
              <input class="control" id="nombre" name="nombre" type="text" autocomplete="name" placeholder="Nombre completo" [(ngModel)]="nombre" [disabled]="loading()" />
            </div>
          </div>
          <div class="campo">
            <label class="campo__etiqueta" for="username">Usuario</label>
            <div class="control-con-icono">
              <app-icono nombre="id-badge-2" [tamano]="18" />
              <input class="control" id="username" name="username" type="text" autocomplete="username" placeholder="Ej: operador01" [(ngModel)]="username" [disabled]="loading()" />
            </div>
          </div>
        </div>

        <div class="campo">
          <label class="campo__etiqueta" for="email">Correo</label>
          <div class="control-con-icono">
            <app-icono nombre="mail" [tamano]="18" />
            <input class="control" id="email" name="email" type="email" autocomplete="email" placeholder="usuario@empresa.com" [(ngModel)]="email" [disabled]="loading()" />
          </div>
        </div>

        <div class="formulario__fila">
          <div class="campo">
            <label class="campo__etiqueta" for="password">Contraseña</label>
            <div class="control-con-icono">
              <app-icono nombre="lock" [tamano]="18" />
              <input class="control" id="password" name="password" [type]="showPass() ? 'text' : 'password'" autocomplete="new-password" placeholder="Mínimo 6 caracteres" [(ngModel)]="password" [disabled]="loading()" />
            </div>
          </div>
          <div class="campo">
            <label class="campo__etiqueta" for="confirmPassword">Confirmar contraseña</label>
            <div class="control-con-icono">
              <app-icono nombre="check" [tamano]="18" />
              <input class="control" id="confirmPassword" name="confirmPassword" [type]="showPass() ? 'text' : 'password'" autocomplete="new-password" placeholder="Repite la contraseña" [(ngModel)]="confirmPassword" [disabled]="loading()" />
            </div>
          </div>
        </div>

        <label class="casilla">
          <input type="checkbox" [checked]="showPass()" (change)="showPass.set(!showPass())" [disabled]="loading()" />
          Mostrar contraseñas
        </label>

        <button class="btn btn--primario btn--lg btn--bloque" type="submit" [disabled]="loading()">
          @if (loading()) {
            <span class="girador" aria-hidden="true"></span>Creando usuario…
          } @else {
            <app-icono nombre="user-plus" [tamano]="18" />Crear cuenta
          }
        </button>
      </form>

      <p class="acceso__enlace">¿Ya tienes cuenta? <a routerLink="/login">Iniciar sesión</a></p>
    </div>
    <p class="acceso__pie">LIZAR SRL · Trujillo, Perú</p>
  </main>
</div>
```

Borra las hojas de estilo viejas:

```bash
rm "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend/src/app/features/login/login.component.css" \
   "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend/src/app/features/register/register.component.css"
```

- [ ] **Step 6: Crear `src/styles/acceso.css` e importarlo**

`frontend/src/styles/acceso.css`:

```css
/* ══ Acceso: iniciar sesión y crear usuario ══ */
.acceso {
  display: grid;
  grid-template-columns: minmax(360px, 2fr) 3fr;
  grid-template-areas: 'marca principal';
  min-height: 100dvh;
  background: var(--suelo);
}

.acceso__marca {
  grid-area: marca;
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: var(--e-10);
  padding: var(--e-10) var(--e-10) var(--e-12);
  overflow: hidden;
  background: var(--asfalto);
  color: #fff;
}

.acceso__via {
  position: relative;
  display: grid;
  gap: var(--e-8);
  padding-left: var(--e-10);
}

.acceso__via::before {
  content: '';
  position: absolute;
  top: 4px;
  bottom: 4px;
  left: 10px;
  border-left: 3px dashed rgb(255 255 255 / 0.4);
}

.acceso__hito {
  position: relative;
  color: var(--asfalto-tinta);
  font-weight: 600;
}

.acceso__hito::before {
  content: '';
  position: absolute;
  top: 50%;
  left: calc(-1 * var(--e-10) + 1px);
  width: 20px;
  height: 20px;
  border: 2px solid #fff;
  border-radius: 3px;
  background: var(--c);
  transform: translateY(-50%);
}

.acceso__hito[data-estado='Registrado'] {
  --c: var(--registrado);
}

.acceso__hito[data-estado='Pagado'] {
  --c: var(--pagado);
}

.acceso__hito[data-estado='Sustentado'] {
  --c: var(--sustentado);
}

.acceso__lema {
  max-width: 18ch;
  font-size: var(--t-24);
  font-weight: 700;
  line-height: 1.25;
}

.acceso__cinta {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
}

.acceso__cabecera-movil {
  display: none;
}

.acceso__principal {
  grid-area: principal;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--e-6);
  padding: var(--e-10) var(--margen);
}

.acceso__tarjeta {
  width: 100%;
  max-width: 420px;
}

.acceso__tarjeta--ancha {
  max-width: 560px;
}

.acceso__titulo {
  font-size: var(--t-32);
  font-weight: 700;
  letter-spacing: -0.02em;
}

.acceso__bajada {
  margin: var(--e-1) 0 var(--e-6);
  color: var(--tinta-2);
}

.acceso__tarjeta > .alerta {
  margin-bottom: var(--e-5);
}

.acceso__pista {
  margin-top: var(--e-5);
  color: var(--tinta-2);
  font-size: var(--t-13);
}

.acceso__pista code {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--panel-2);
  color: var(--tinta);
}

.acceso__enlace {
  margin-top: var(--e-3);
  color: var(--tinta-2);
  font-size: var(--t-14);
}

.acceso__enlace a {
  color: var(--rojo-tinta);
  font-weight: 600;
}

.acceso__pie {
  color: var(--tinta-3);
  font-size: var(--t-13);
}

@media (max-width: 899px) {
  .acceso {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
    grid-template-areas:
      'cabecera'
      'principal';
  }

  .acceso__marca {
    display: none;
  }

  .acceso__cabecera-movil {
    grid-area: cabecera;
    display: block;
    background: var(--asfalto);
  }

  .acceso__cabecera-movil-interior {
    display: flex;
    align-items: center;
    height: 56px;
    padding: 0 var(--margen);
  }

  .acceso__principal {
    justify-content: flex-start;
    padding-top: var(--e-8);
  }
}
```

En `frontend/src/styles.css`, agrega al final:

```css
@import './styles/acceso.css';
```

- [ ] **Step 7: Ejecutar y verificar que pasan**

Run: `npx ng test --watch=false --include "src/app/features/{login,register}/*.spec.ts"`
Expected: PASS (4 pruebas).

- [ ] **Step 8: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected: PASS y build exitoso.

---
### Task 20: Limpieza: eliminar lo reemplazado y la compatibilidad temporal

**Files:**
- Delete:
  - `frontend/src/app/features/dashboard/`
  - `frontend/src/app/features/crud-view/`
  - `frontend/src/app/features/gastos/comprobante-upload/` y `frontend/src/app/features/gastos/aprobacion-comprobantes/`
  - `frontend/src/app/shared/components/sidebar/`, `frontend/src/app/shared/components/topbar/` y `frontend/src/app/shared/components/modal/`
  - `frontend/src/assets/css/`
- Modify: `frontend/src/app/core/services/crud.service.ts` (quitar `saveItem` y `refreshAll`)

**Interfaces:**
- Consumes: nada nuevo.
- Produces: `CrudService` sin métodos de compatibilidad. Ningún archivo referencia las piezas borradas.

- [ ] **Step 1: Verificar que nada vivo usa lo que se va a borrar**

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend"
grep -rn "dashboard.component\|crud-view\|comprobante-upload\|aprobacion-comprobantes\|sidebar.component\|topbar.component\|modal.component\|assets/css" src --include='*.ts' --include='*.html' --include='*.css' \
  | grep -v "^src/app/features/dashboard/\|^src/app/features/crud-view/\|^src/app/features/gastos/comprobante-upload/\|^src/app/features/gastos/aprobacion-comprobantes/\|^src/app/shared/components/sidebar/\|^src/app/shared/components/topbar/\|^src/app/shared/components/modal/\|^src/assets/css/"
```

Expected: sin salida. Las rutas ya no cargan el dashboard (Tarea 17) y ningún componente nuevo importa las piezas viejas.

- [ ] **Step 2: Borrar los archivos reemplazados**

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend/src"
rm -r app/features/dashboard app/features/crud-view \
      app/features/gastos/comprobante-upload app/features/gastos/aprobacion-comprobantes \
      app/shared/components/sidebar app/shared/components/topbar app/shared/components/modal \
      assets/css
rmdir assets 2>/dev/null || true
```

- [ ] **Step 3: Quitar la compatibilidad temporal de `CrudService`**

En `frontend/src/app/core/services/crud.service.ts`, borra estos dos métodos completos (con su comentario `@deprecated`):

```ts
  /** @deprecated Solo para la vista CRUD vieja; se elimina en la Tarea 20. */
  saveItem(key: string, payload: RecordMap): void {
    this.guardar(key, payload).subscribe();
  }

  /** @deprecated Solo para el dashboard viejo; se elimina en la Tarea 20. */
  refreshAll(): void {
    this.cruds.forEach(c => this.loadList(c.key));
    this.toast.show('Datos actualizados', '', 'ok', 2200);
  }
```

- [ ] **Step 4: Verificar referencias, rayas largas y emojis**

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend"
grep -rn "saveItem\|refreshAll\|app-sidebar\|app-topbar\|app-modal" src || echo "OK: sin referencias viejas"
grep -rn "—\|–" src/app --include='*.html' || echo "OK: sin rayas largas en plantillas"
node -e '
const fs = require("fs"), path = require("path");
const archivos = [];
(function recorrer(d) { for (const f of fs.readdirSync(d)) { const p = path.join(d, f); fs.statSync(p).isDirectory() ? recorrer(p) : /\.(ts|html|css)$/.test(p) && archivos.push(p); } })("src");
const con = archivos.filter(f => /\p{Extended_Pictographic}/u.test(fs.readFileSync(f, "utf8").replace(/[©®]/g, "")));
console.log(con.length ? "Emojis en: " + con.join(", ") : "OK: sin emojis");'
```

Expected: las tres verificaciones imprimen `OK`. El "—" que queda en archivos `.ts` es el valor vacío existente de `formatFecha` y de las celdas, y está permitido.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build`
Expected:
- PASS en todas las pruebas: 29 archivos de prueba y 101 pruebas.
- Build exitoso, sin errores de presupuesto ni avisos de `anyComponentStyle`.

Anota en el checkpoint el tamaño del bundle inicial que reporta el build.

---
### Task 21: Recorrido visual con datos simulados (escritorio 1440 y celular 390)

**Files (todo fuera del repositorio, en el scratchpad):**
- Create: `$SCRATCH/visual/package.json` (con `npm init` + `playwright-core`)
- Create: `$SCRATCH/visual/simulador.mjs` (API simulada con estado en memoria)
- Create: `$SCRATCH/visual/capturar.mjs` (capturas de todas las pantallas)
- Create: `$SCRATCH/visual/flujos.mjs` (flujos de punta a punta)
- Copy al repo: `.impeccable/review/desktop.png` y `.impeccable/review/mobile.png` (evidencia para la revisión final)

**Interfaces:**
- Consumes:
  - la app corriendo con `ng serve` en `http://localhost:4200`
  - las bases de datos de desarrollo, que `ApiService` y `GastosApiService` resuelven solas: `http://localhost:8000/api/v1` y `http://localhost:8081`
- Produces: capturas en `$SCRATCH/visual/capturas/` y la evidencia en `.impeccable/review/`.

> Los datos son sintéticos y solo sirven para verificar la interfaz; nunca se copian al repositorio ni a la app.

- [ ] **Step 1: Preparar Playwright con el Chrome del sistema**

```bash
SCRATCH=/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad
mkdir -p "$SCRATCH/visual" && cd "$SCRATCH/visual"
npm init -y >/dev/null
npm install playwright-core@1
```

Expected: instala sin descargar navegadores; se usa `/Applications/Google Chrome.app`.

- [ ] **Step 2: Escribir la API simulada**

`$SCRATCH/visual/simulador.mjs`:

```js
// API simulada (datos sintéticos) para verificar la interfaz sin backends.
const ahora = Date.now();
const hace = horas => new Date(ahora - horas * 3600e3).toISOString();

export const USUARIO = { id_usuario: 1, username: 'aquispe', email: 'ana.quispe@lizar.pe', nombre: 'Ana Quispe', rol: 'admin', estado: true };

const api = {
  rubros: [
    { id_rubro: 1, nombre: 'Agroindustria', descripcion: 'Exportación de espárrago y palta' },
    { id_rubro: 2, nombre: 'Minería', descripcion: 'Insumos y concentrados' },
    { id_rubro: 3, nombre: 'Consumo masivo', descripcion: 'Distribución a mayoristas' },
  ],
  marcas: [
    { id_marca: 1, nombre: 'Volvo', pais_origen: 'Suecia' },
    { id_marca: 2, nombre: 'Scania', pais_origen: 'Suecia' },
    { id_marca: 3, nombre: 'Freightliner', pais_origen: 'Estados Unidos' },
  ],
  'tipos-unidad': [
    { id_tipo_unidad: 1, nombre: 'Tracto', descripcion: 'Cabina tractora' },
    { id_tipo_unidad: 2, nombre: 'Semirremolque', descripcion: 'Carreta de tres ejes' },
  ],
  'subtipos-unidad': [{ id_subtipo_unidad: 1, id_tipo_unidad: 2, nombre: 'Plataforma', descripcion: 'Carga paletizada' }],
  ubigeos: [
    { codigo_ubigeo: 130101, departamento: 'La Libertad', provincia: 'Trujillo', distrito: 'Trujillo' },
    { codigo_ubigeo: 140101, departamento: 'Lambayeque', provincia: 'Chiclayo', distrito: 'Chiclayo' },
    { codigo_ubigeo: 70101, departamento: 'Callao', provincia: 'Callao', distrito: 'Callao' },
  ],
  direcciones: [
    { id_direccion: 1, codigo_ubigeo: 130101, direccion_detallada: 'Av. Nicolás de Piérola 1250', referencia: 'Frente al terminal terrestre' },
    { id_direccion: 2, codigo_ubigeo: 140101, direccion_detallada: 'Carretera Panamericana Norte 772', referencia: 'Almacén 3' },
    { id_direccion: 3, codigo_ubigeo: 70101, direccion_detallada: 'Av. Argentina 3093', referencia: 'Puerta 2' },
  ],
  conductores: [
    { id_conductor: 1, dni: '45879312', nombre1: 'Julio', apellido1: 'Rojas', apellido2: 'Vega', licencia_conducir: 'Q45879312', telefono: '949 381 207', estado_conductor: true },
    { id_conductor: 2, dni: '41230987', nombre1: 'Rosa', apellido1: 'Huamán', apellido2: 'Silva', licencia_conducir: 'Q41230987', telefono: '956 120 448', estado_conductor: true },
  ],
  clientes: [
    { id_cliente: 1, razon_social: 'Agrícola del Norte SAC', ruc: '20481234567', id_rubro: 1, correo: 'pagos@agricolanorte.pe', telefono: '044 287 110', estado_cliente: true },
    { id_cliente: 2, razon_social: 'Minera Andina Huamachuco', ruc: '20539876541', id_rubro: 2, telefono: '044 441 902', estado_cliente: true },
    { id_cliente: 3, razon_social: 'Distribuidora La Esperanza', ruc: '20600321789', id_rubro: 3, estado_cliente: true },
  ],
  unidades: [
    { id_unidad: 1, placa: 'T4K-829', modelo: 'FH 460', id_marca: 1, id_tipo_unidad: 1, anio: 2021, estado_unidad: true },
    { id_unidad: 2, placa: 'C9X-114', modelo: 'Plataforma 3 ejes', id_marca: 2, id_tipo_unidad: 2, id_subtipo_unidad: 1, anio: 2019, estado_unidad: true },
  ],
  'asignaciones-conductor-unidad': [{ id_conductor_unidad: 1, id_conductor: 1, id_unidad: 1, fecha_asignacion: '2026-08-12', estado_asignacion: true }],
  viajes: [
    { id_viaje: 14, id_tracto: 1, id_carreta: 2, id_conductor_unidad: 1, id_cliente: 1, id_direccion_origen: 1, id_direccion_destino: 3, fecha_programado: '2026-09-29T00:00:00Z', estado_viaje: true },
    { id_viaje: 15, id_tracto: 1, id_carreta: null, id_conductor_unidad: 1, id_cliente: 2, id_direccion_origen: 1, id_direccion_destino: 2, fecha_programado: '2026-10-01T00:00:00Z', estado_viaje: true },
  ],
};

const ID_DE = {
  rubros: 'id_rubro', marcas: 'id_marca', 'tipos-unidad': 'id_tipo_unidad', 'subtipos-unidad': 'id_subtipo_unidad',
  direcciones: 'id_direccion', conductores: 'id_conductor', clientes: 'id_cliente', unidades: 'id_unidad',
  'asignaciones-conductor-unidad': 'id_conductor_unidad', viajes: 'id_viaje',
};

const gastos = [
  { id: 29, viaje_id: 14, cliente_id: 1, monto: 310, fecha: '2026-09-29T00:00:00Z', estado: 'Sustentado', creado_por: 'Ana Quispe', creado_en: hace(96) },
  { id: 30, viaje_id: 14, cliente_id: 1, monto: 85.5, fecha: '2026-09-30T00:00:00Z', estado: 'Pagado', creado_por: 'Ana Quispe', creado_en: hace(50) },
  { id: 31, viaje_id: 15, cliente_id: 2, monto: 120, fecha: '2026-10-01T00:00:00Z', estado: 'Registrado', creado_por: 'Julio Rojas', creado_en: hace(30) },
  { id: 32, viaje_id: 15, cliente_id: 2, monto: 47.8, fecha: '2026-10-02T00:00:00Z', estado: 'Pagado', creado_por: 'Julio Rojas', creado_en: hace(20) },
  { id: 33, viaje_id: 15, cliente_id: 3, monto: 1240.5, fecha: '2026-10-02T00:00:00Z', estado: 'Registrado', creado_por: 'Ana Quispe', creado_en: hace(6) },
];

const comprobantes = [
  { id: 70, gasto_id: 29, archivo_url: '/uploads/70.svg', tipo_archivo: 'imagen', estado: 'Aprobado', aprobado_por: 'Ana Quispe', aprobado_en: hace(90), creado_en: hace(94) },
  { id: 71, gasto_id: 31, archivo_url: '/uploads/71.svg', tipo_archivo: 'imagen', estado: 'Rechazado', motivo_rechazo: 'El monto del comprobante no coincide con el gasto', aprobado_por: 'Ana Quispe', aprobado_en: hace(26), creado_en: hace(28) },
  { id: 72, gasto_id: 30, archivo_url: '/uploads/72.svg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: hace(40) },
  { id: 73, gasto_id: 32, archivo_url: '/uploads/73.pdf', tipo_archivo: 'pdf', estado: 'Pendiente', creado_en: hace(18) },
];

// Comprobante de ejemplo (rotulado como ejemplo para que nadie lo confunda con uno real).
const TICKET = `<svg xmlns="http://www.w3.org/2000/svg" width="360" height="520" viewBox="0 0 360 520">
<rect width="360" height="520" fill="#fbfaf6"/>
<g font-family="monospace" fill="#222" font-size="16">
<text x="28" y="48" font-size="13">COMPROBANTE DE EJEMPLO</text>
<text x="28" y="88">PEAJE PANAMERICANA NORTE</text>
<text x="28" y="118">BOLETA B012-004812</text>
<text x="28" y="158">Vehículo 6 ejes</text>
<text x="28" y="188">Placa T4K-829</text>
<text x="28" y="252" font-size="22">TOTAL S/ 85.50</text>
<text x="28" y="292">30/09/2026 09:41</text>
</g></svg>`;

const CORS = {
  'access-control-allow-origin': 'http://localhost:4200',
  'access-control-allow-headers': '*',
  'access-control-allow-methods': 'GET,POST,PUT,DELETE,OPTIONS',
};

const siguienteId = lista => Math.max(0, ...lista.map(x => x.id)) + 1;

/** Responde a la API principal (:8000) y al servicio de gastos (:8081) con estado en memoria. */
export async function responder(route) {
  const req = route.request();
  const url = new URL(req.url());
  const metodo = req.method();
  const json = (cuerpo, status = 200) =>
    route.fulfill({ status, headers: CORS, contentType: 'application/json', body: JSON.stringify(cuerpo) });

  if (metodo === 'OPTIONS') return route.fulfill({ status: 204, headers: CORS });

  if (url.port === '8000') {
    if (url.pathname === '/health') return json({ status: 'ok', env: 'demo' });
    const ruta = url.pathname.replace('/api/v1/', '');
    if (ruta === 'auth/login' && metodo === 'POST') return json({ access_token: 'demo', token_type: 'bearer', usuario: USUARIO });
    if (api[ruta] && metodo === 'GET') return json(api[ruta]);
    if (api[ruta] && metodo === 'POST') {
      const clave = ID_DE[ruta];
      const nuevo = { ...req.postDataJSON(), ...(clave ? { [clave]: 100 + api[ruta].length } : {}) };
      api[ruta].push(nuevo);
      return json(nuevo, 201);
    }
    return json([]);
  }

  if (url.port === '8081') {
    if (url.pathname === '/gastos' && metodo === 'GET') return json(gastos);
    if (url.pathname === '/gastos' && metodo === 'POST') {
      const datos = req.postDataJSON();
      const gasto = { id: siguienteId(gastos), ...datos, fecha: `${datos.fecha}T00:00:00Z`, estado: 'Registrado', creado_por: USUARIO.nombre, creado_en: new Date().toISOString() };
      gastos.push(gasto);
      return json(gasto, 201);
    }
    const subida = url.pathname.match(/^\/gastos\/(\d+)\/comprobantes$/);
    if (subida && metodo === 'POST') {
      const gastoId = Number(subida[1]);
      const nuevo = { id: siguienteId(comprobantes), gasto_id: gastoId, archivo_url: '/uploads/nuevo.svg', tipo_archivo: 'imagen', estado: 'Pendiente', creado_en: new Date().toISOString() };
      comprobantes.push(nuevo);
      gastos.find(g => g.id === gastoId).estado = 'Pagado';
      return json(nuevo, 201);
    }
    if (url.pathname === '/comprobantes' && metodo === 'GET') {
      const estado = url.searchParams.get('estado');
      return json(estado ? comprobantes.filter(c => c.estado === estado) : comprobantes);
    }
    const accion = url.pathname.match(/^\/comprobantes\/(\d+)\/(aprobar|rechazar)$/);
    if (accion && metodo === 'POST') {
      const c = comprobantes.find(x => x.id === Number(accion[1]));
      const g = gastos.find(x => x.id === c.gasto_id);
      c.aprobado_por = USUARIO.nombre;
      c.aprobado_en = new Date().toISOString();
      if (accion[2] === 'aprobar') {
        c.estado = 'Aprobado';
        g.estado = 'Sustentado';
      } else {
        c.estado = 'Rechazado';
        c.motivo_rechazo = req.postDataJSON().motivo_rechazo;
        g.estado = 'Registrado';
      }
      return json(c);
    }
    if (url.pathname.startsWith('/uploads/')) {
      return route.fulfill({ status: 200, headers: CORS, contentType: 'image/svg+xml', body: TICKET });
    }
    return json({ error: 'no encontrado' }, 404);
  }

  return route.continue();
}

/** Contexto de navegador con la API simulada; con sesión iniciada si se pide. */
export async function nuevoContexto(navegador, opciones, conSesion) {
  const contexto = await navegador.newContext({ reducedMotion: 'reduce', ...opciones });
  await contexto.route('http://localhost:8000/**', responder);
  await contexto.route('http://localhost:8081/**', responder);
  if (conSesion) {
    await contexto.addInitScript(usuario => {
      localStorage.setItem('lizar_token', 'demo');
      localStorage.setItem('lizar_user', JSON.stringify(usuario));
    }, USUARIO);
  }
  return contexto;
}

export const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
export const BASE = 'http://localhost:4200';

/** Espera a que ng serve responda (hasta 3 minutos). */
export async function esperarServidor() {
  for (let i = 0; i < 90; i++) {
    try {
      if ((await fetch(BASE)).ok) return;
    } catch {
      /* todavía compilando */
    }
    await new Promise(r => setTimeout(r, 2000));
  }
  throw new Error('ng serve no respondió en http://localhost:4200');
}
```

- [ ] **Step 3: Escribir el script de capturas**

`$SCRATCH/visual/capturar.mjs`:

```js
import { chromium } from 'playwright-core';
import { copyFileSync, mkdirSync } from 'node:fs';
import { BASE, CHROME, esperarServidor, nuevoContexto } from './simulador.mjs';

const SALIDA = new URL('./capturas/', import.meta.url).pathname;
const REVISION = '/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/.impeccable/review/';
mkdirSync(SALIDA, { recursive: true });
mkdirSync(REVISION, { recursive: true });

const TAMANOS = {
  escritorio: { viewport: { width: 1440, height: 900 } },
  celular: { viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, deviceScaleFactor: 2 },
};

const PUBLICAS = [
  ['landing', '/'],
  ['login', '/login'],
  ['registro', '/register'],
  ['no-encontrada', '/no-existe'],
];

const PRIVADAS = [
  ['inicio', '/app/inicio'],
  ['gastos', '/app/gastos'],
  ['gastos-sin-comprobante', '/app/gastos?estado=sin-comprobante'],
  ['gastos-por-aprobar', '/app/gastos?estado=por-aprobar'],
  ['gastos-sustentados', '/app/gastos?estado=sustentados'],
  ['detalle-registrado-con-rechazo', '/app/gastos/31'],
  ['detalle-pagado-imagen', '/app/gastos/30'],
  ['detalle-pagado-pdf', '/app/gastos/32'],
  ['detalle-sustentado', '/app/gastos/29'],
  ['registrar', '/app/gastos/nuevo'],
  ['operacion-viajes', '/app/operacion/viajes'],
  ['operacion-clientes', '/app/operacion/clientes'],
  ['catalogos-rubros', '/app/catalogos/rubros'],
  ['viajes-hoja-nuevo', '/app/operacion/viajes', async p => p.getByRole('button', { name: 'Nuevo viaje' }).first().click()],
  ['paleta', '/app/inicio', async p => { await p.keyboard.press('Control+k'); await p.keyboard.type('flo'); }],
  ['hoja-mas', '/app/inicio', async p => p.getByRole('button', { name: 'Más' }).click(), 'celular'],
];

async function capturar(contexto, tamano, [nombre, ruta, accion, solo]) {
  if (solo && solo !== tamano) return;
  const pagina = await contexto.newPage();
  await pagina.goto(BASE + ruta, { waitUntil: 'networkidle' });
  await pagina.evaluate(() => document.fonts.ready);
  const desborde = await pagina.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  if (desborde > 0) console.warn(`DESBORDE horizontal de ${desborde}px en ${nombre} (${tamano})`);
  if (accion) await accion(pagina);
  await pagina.waitForTimeout(400);
  const archivo = `${SALIDA}${nombre}-${tamano}.png`;
  await pagina.screenshot({ path: archivo, fullPage: !accion });
  await pagina.close();
  console.log('captura', archivo);
}

await esperarServidor();
const navegador = await chromium.launch({ executablePath: CHROME, headless: true });
for (const [tamano, opciones] of Object.entries(TAMANOS)) {
  const publico = await nuevoContexto(navegador, opciones, false);
  for (const pantalla of PUBLICAS) await capturar(publico, tamano, pantalla);
  await publico.close();
  const privado = await nuevoContexto(navegador, opciones, true);
  for (const pantalla of PRIVADAS) await capturar(privado, tamano, pantalla);
  await privado.close();
}
await navegador.close();

copyFileSync(`${SALIDA}gastos-escritorio.png`, `${REVISION}desktop.png`);
copyFileSync(`${SALIDA}gastos-celular.png`, `${REVISION}mobile.png`);
console.log('Evidencia copiada a .impeccable/review/');
```

- [ ] **Step 4: Escribir el script de flujos de punta a punta**

`$SCRATCH/visual/flujos.mjs`:

```js
import { chromium } from 'playwright-core';
import { BASE, CHROME, esperarServidor, nuevoContexto } from './simulador.mjs';

const SALIDA = new URL('./capturas/', import.meta.url).pathname;
const verificar = (condicion, mensaje) => {
  if (!condicion) throw new Error(`Falló: ${mensaje}`);
  console.log('OK', mensaje);
};

await esperarServidor();
const navegador = await chromium.launch({ executablePath: CHROME, headless: true });
const contexto = await nuevoContexto(navegador, { viewport: { width: 1440, height: 900 } }, true);
const p = await contexto.newPage();

// 1. Registrar un gasto eligiendo viaje y cliente por nombre
await p.goto(`${BASE}/app/gastos/nuevo`, { waitUntil: 'networkidle' });
await p.locator('#gf-viaje').click();
await p.locator('#gf-viaje').fill('15');
await p.keyboard.press('Enter');
await p.locator('#gf-cliente').click();
await p.locator('#gf-cliente').fill('andina');
await p.keyboard.press('Enter');
await p.locator('#gf-monto').fill('64.30');
await p.getByRole('button', { name: 'Registrar gasto' }).click();
await p.waitForURL(/\/app\/gastos\/34(#paso)?$/);
await p.getByText('Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago.').waitFor();
verificar(true, 'registrar abre el gasto nuevo en el paso de adjuntar, con el aviso exacto');
await p.screenshot({ path: `${SALIDA}flujo-1-registrado.png` });

// 2. Adjuntar el comprobante (foto)
const png = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=', 'base64');
await p.locator('input[type="file"]').setInputFiles({ name: 'boleta.png', mimeType: 'image/png', buffer: png });
await p.getByRole('button', { name: 'Subir comprobante' }).click();
await p.getByText('Comprobante adjuntado. El gasto cambió a estado "Pagado".').waitFor();
await p.getByRole('heading', { name: 'Revisar comprobante' }).waitFor();
verificar(true, 'adjuntar deja el gasto listo para revisar');
await p.screenshot({ path: `${SALIDA}flujo-2-pagado.png` });

// 3. Aprobar
await p.getByRole('button', { name: 'Aprobar' }).click();
await p.getByText('Comprobante aprobado. El gasto quedó como "Sustentado".').waitFor();
await p.getByRole('heading', { name: 'Gasto sustentado' }).waitFor();
verificar(true, 'aprobar deja el gasto sustentado');
await p.screenshot({ path: `${SALIDA}flujo-3-sustentado.png` });

// 4. Rechazar con motivo desde la pestaña «Por aprobar»
await p.goto(`${BASE}/app/gastos?estado=por-aprobar`, { waitUntil: 'networkidle' });
await p.locator('li.fila-gasto', { hasText: 'N° 32' }).getByRole('link', { name: 'Revisar' }).click();
await p.getByRole('button', { name: 'Rechazar' }).click();
await p.getByRole('button', { name: 'Confirmar rechazo' }).click();
verificar(await p.getByText('Debes indicar un motivo para rechazar el comprobante.').isVisible(), 'rechazar sin motivo muestra el error exacto');
await p.locator('#motivo-rechazo').fill('El PDF no corresponde a este viaje');
await p.getByRole('button', { name: 'Confirmar rechazo' }).click();
await p.getByText('Comprobante rechazado. El gasto volvió a estado "Registrado".').waitFor();
await p.getByText('El comprobante anterior fue rechazado: El PDF no corresponde a este viaje').waitFor();
verificar(true, 'el rechazo devuelve el gasto a Registrado con el motivo a la vista');
await p.screenshot({ path: `${SALIDA}flujo-4-rechazado.png`, fullPage: true });
await p.locator('a.volver').click();
await p.waitForURL(/\/app\/gastos\?estado=por-aprobar$/);
verificar(true, '«← Gastos» vuelve a la pestaña de origen');

// 5. Crear un cliente eligiendo el rubro por nombre
await p.goto(`${BASE}/app/operacion/clientes`, { waitUntil: 'networkidle' });
await p.getByRole('button', { name: 'Nuevo cliente' }).first().click();
await p.locator('#cl_r').fill('Transportes Moche EIRL');
await p.locator('#cl_ruc').fill('20612345678');
await p.locator('#cl_rb').click();
await p.locator('#cl_rb').fill('mine');
await p.keyboard.press('Enter');
await p.getByRole('button', { name: 'Guardar' }).click();
await p.getByText('Cliente guardado').waitFor();
const fila = p.locator('tbody tr', { hasText: 'Transportes Moche EIRL' });
await fila.waitFor();
verificar((await fila.textContent()).includes('Minería'), 'el cliente nuevo aparece con su rubro por nombre');
await p.screenshot({ path: `${SALIDA}flujo-5-cliente.png` });

// 6. Paleta ⌘K y botón atrás del navegador
await p.goto(`${BASE}/app/inicio`, { waitUntil: 'networkidle' });
await p.keyboard.press('Control+k');
await p.keyboard.type('33');
await p.keyboard.press('Enter');
await p.waitForURL(/\/app\/gastos\/33$/);
verificar(true, 'la paleta busca un gasto por número y lo abre');
await p.goBack();
await p.waitForURL(/\/app\/inicio$/);
verificar(true, 'el botón atrás del navegador funciona');

await navegador.close();
console.log('Todos los flujos pasaron.');
```

- [ ] **Step 5: Levantar la app y ejecutar ambos scripts**

En una terminal en segundo plano (desde `frontend/`): `npx ng serve --port 4200`.

Luego:

```bash
cd "$SCRATCH/visual"
node capturar.mjs
node flujos.mjs
```

Expected:
- `capturar.mjs` imprime una línea `captura …` por pantalla (19 de escritorio y 20 de celular), sin ningún `DESBORDE`, y termina con `Evidencia copiada a .impeccable/review/`.
- `flujos.mjs` imprime 9 líneas `OK …` y `Todos los flujos pasaron.`

- [ ] **Step 6: Revisar las capturas contra el contrato (una ronda de arreglos, máximo dos)**

Abre cada PNG de `$SCRATCH/visual/capturas/` y comprueba:

- **Barra y estados:**
  - barra superior de asfalto de 56 px con la cinta reflectiva roja y blanca
  - enlace activo con subrayado rojo
  - contador amarillo en Gastos
  - en celular, barra inferior de 4 destinos y el contenido sin quedar tapado
- **Gastos (escritorio):**
  - título de 32 px con «Registrar gasto» a la derecha
  - pestañas con conteo
  - columnas alineadas, con montos en cifras tabulares a la derecha y el total con filete bajo la columna Monto
  - ruta compacta con el camión en el hito correcto
  - una acción por fila
  - las filas sustentadas atenuadas en «Todos»
- **Detalle:**
  - ruta completa horizontal (vertical en celular)
  - desvío rojo con el motivo en el gasto 31
  - panel del paso con el filete rojo superior
  - previsualización del comprobante y PDF con su ícono
- **Formularios:** etiquetas encima y errores debajo; en celular, el botón de registrar queda fijo sobre la barra inferior.
- **Landing:**
  - la portada cabe en 1440×900 (titular en 2 líneas, botón visible)
  - conteos reales en la carretera
  - sin indicador de scroll
  - la foto carga
- **General:** nada se desborda, el contraste se lee bien y no queda texto en inglés ni rayas largas en la landing y el acceso.
- **Celular:** repasa la lista de la skill `mobile-native` (Emil Kowalski):
  - hover pegajoso solo con puntero fino
  - sin destello de toque
  - `100dvh`
  - controles de 16 px (sin zoom en iOS)
  - zonas seguras con la barra inferior
  - objetivos de 44 px

Corrige lo encontrado en **un solo lote**, vuelve a correr `node capturar.mjs` y revisa una sola vez más (regla de impeccable: dos rondas como máximo). Luego detén `ng serve`.

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `npx ng test --watch=false && npx ng build` (desde `frontend/`)
Expected: PASS y build exitoso después de los arreglos. `.impeccable/review/desktop.png` y `mobile.png` existen.

---
### Task 22: Detector de impeccable, pre-flight de Taste, revisión final y `DESIGN.md`

**Files:**
- Create (fuera del repo): `$SCRATCH/detector.json`
- Create: `DESIGN.md` y `.impeccable/design.json` (raíz del repositorio; los escribe el documentador)
- Modify: los archivos que señalen el detector o la revisión (un lote por ronda)

**Interfaces:**
- Consumes:
  - todo lo anterior
  - de impeccable: `scripts/impeccable`, `agents/impeccable_finish_reviewer.toml`, `agents/impeccable_documenter.toml`, `reference/craft-floor.md` y `reference/document.md`
- Produces:
  - el veredicto de la revisión final: `ship`, `fix`, `rebuild` o `recapture`
  - `DESIGN.md` y `.impeccable/design.json` con los tokens del sistema

- [ ] **Step 1: Ejecutar el detector de impeccable una vez**

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone"
SCRATCH=/private/tmp/claude-501/-Users-helcorp-Downloads-Proyectos-Universidad-Proyecto-Campstone/00d73130-eac8-4a38-b8ba-011054089b50/scratchpad
~/.claude/skills/impeccable/scripts/impeccable detect --json frontend/src/app frontend/src/styles frontend/src/styles.css > "$SCRATCH/detector.json"
node -e 'const r = require(process.argv[1]); const h = r.findings ?? r.results ?? r; console.log(JSON.stringify(h, null, 1).slice(0, 4000));' "$SCRATCH/detector.json"
```

Expected: un JSON de hallazgos.
- Corrige en un lote lo mecánico: contraste, foco, `transition: all`, animaciones de propiedades de layout, raya larga, emojis, iconos dibujados a mano fuera del registro y medidas fuera de los tokens.
- Excepciones esperadas, que se dejan con su justificación para la revisión:
  - el camión del logo (compromiso de marca)
  - la flecha de `mkRoute`, que es un trazo Tabler en una celda HTML
- Vuelve a correr el detector una sola vez.

- [ ] **Step 2: Pre-flight de Taste en landing y acceso**

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend/src"
grep -n "—\|–" app/features/landing/*.html app/features/login/*.html app/features/register/*.html || echo "OK: sin rayas largas"
grep -in "scroll" app/features/landing/landing.component.html || echo "OK: sin indicador de scroll"
grep -rn "background-clip: *text\|-webkit-background-clip" styles || echo "OK: sin texto con degradado"
grep -rn "cursor: *url\|#cursor" styles app || echo "OK: sin cursor personalizado"
```

Expected: las cuatro líneas `OK`. Revisa también a mano, con `landing-escritorio.png` y `landing-celular.png`:
- **Portada:**
  - titular de máximo 2 líneas en 1440 px, bajada de 20 palabras o menos y CTA visible sin hacer scroll
  - una sola etiqueta por intención ("Ingresar" e "Ir a Inicio" nunca juntos; "Crear usuario" es otra intención)
- **Sin decoración de plantilla:**
  - sin eyebrows en mayúsculas espaciadas
  - sin puntos de estado decorativos
  - sin franjas de texto decorativas
- **Movimiento:** todo `transform` y `clip-path` animado vive dentro de `@media (prefers-reduced-motion: no-preference)`.
- **Imagen:** la foto real carga con `alt` descriptivo y `loading="lazy"`.

- [ ] **Step 3: Revisión final en contexto limpio**

Lanza un subagente `general-purpose` **nuevo** (sin el historial de esta sesión). Usa este prompt, completando las rutas con las reales:

```
Eres el revisor final de impeccable. Lee y sigue al pie de la letra las instrucciones de
~/.claude/skills/impeccable/agents/impeccable_finish_reviewer.toml (y el piso de calidad en
~/.claude/skills/impeccable/reference/craft-floor.md). No tienes navegador: juzga con las capturas.

Pedido original del usuario (español): «rediseña toda la web para que quede mucho mejor visualmente,
además ajusta la experiencia de usuario de los módulos: actualmente todo está en la izquierda, mejora el
flujo de cómo interactuar entre ellos». Respuestas confirmadas: usuarios = logística en oficina (PC) y
conductores en ruta (celular); identidad = se conservan nombre LIZAR SRL, logo del camión y rojo #E11D48;
navegación = barra superior + flujo de Gastos (barra inferior en celular); dirección = «Hitos de carretera».

Insumos:
- Spec: docs/superpowers/specs/2026-10-02-rediseno-hitos-design.md
- Contrato de dirección: .impeccable/surfaces/frontend-src-app.md
- PRODUCT.md
- Código: frontend/src (estilos en frontend/src/styles)
- Capturas requeridas: .impeccable/review/desktop.png y .impeccable/review/mobile.png
- Resto de capturas: <SCRATCH>/visual/capturas/*.png (escritorio y celular de cada pantalla y flujo)
- Hallazgos del detector: <SCRATCH>/detector.json
- Build: code-led (no hay comp aprobado; no hay generación de imágenes).

Devuelve las cinco secciones de contrato que pide el archivo .toml, con una disposición final:
ship, fix, rebuild o recapture, y la lista de hallazgos materiales con archivo y arreglo propuesto.
```

Actúa según la disposición:
- **`ship`:** pasa al Step 4.
- **`fix`:** aplica los arreglos materiales en un lote, recaptura las mismas pantallas (`node capturar.mjs`) y pide al mismo revisor un veredicto solo sobre esos arreglos. Como máximo, dos rondas.
- **`rebuild`:** rehaz las regiones señaladas y pide una revisión completa nueva.
- **`recapture`:** repite las capturas y pide la revisión otra vez.

Si después de dos rondas quedan puntos abiertos, preséntalos al usuario y deja que decida.

- [ ] **Step 4: Documentar el sistema (`DESIGN.md`)**

Lanza un subagente `general-purpose` nuevo con este prompt:

```
Eres el documentador de impeccable. Sigue ~/.claude/skills/impeccable/agents/impeccable_documenter.toml
y ~/.claude/skills/impeccable/reference/document.md. Raíz del proyecto:
/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone. Artefacto: frontend/src (tokens en
frontend/src/styles/tokens.css). Contrato: .impeccable/surfaces/frontend-src-app.md. Producto: PRODUCT.md.
Solo puedes escribir DESIGN.md y .impeccable/design.json en la raíz del proyecto; no toques el código.
Registra la procedencia de la única imagen de la landing: Unsplash, foto
https://images.unsplash.com/photo-1519003722824-194d4455a60c (licencia Unsplash), enlazada, no copiada al repo.
```

Verifica:

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone"
test -s DESIGN.md && test -s .impeccable/design.json && grep -c "#1E2124\|#1e2124" DESIGN.md
```

Expected: ambos archivos existen y `DESIGN.md` contiene los tokens (al menos una coincidencia de asfalto).

- [ ] **Step 5: Verificación final antes de dar por terminado**

Usa `superpowers:verification-before-completion`:

```bash
cd "/Users/helcorp/Downloads/Proyectos/Universidad/Proyecto Campstone/frontend"
npx ng test --watch=false && npx ng build
cd .. && git status --short
```

Expected:
- PASS en todas las pruebas y build exitoso.
- `git status` lista los cambios del rediseño sobre los cambios previos del usuario, **sin commits** (decisión del usuario).

Informa al usuario:
- qué cambió
- dónde están las capturas
- el veredicto de la revisión
- el respaldo en `$SCRATCH/respaldo-*`
- que los archivos viejos borrados (dashboard, sidebar y vistas de comprobante y aprobación) siguen disponibles en ese respaldo
