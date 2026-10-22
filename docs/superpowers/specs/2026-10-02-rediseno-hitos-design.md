# Rediseño "Hitos de carretera" · Fleet Manager Pro (frontend)

- **Fecha:** 2026-10-02
- **Estado:** diseño aprobado en conversación; pendiente de revisión escrita
- **Alcance:** `frontend/` (Angular 21). Backend y servicio de gastos sin cambios.
- **Contexto de producto:** [`PRODUCT.md`](../../../PRODUCT.md)

## 1. Objetivo

Rediseñar toda la web (landing, acceso y aplicación) para que se vea mucho mejor y sea más original, y reorganizar la navegación. Hoy todos los módulos están en una barra lateral izquierda de 15 enlaces. El rediseño también debe mejorar el flujo entre módulos.

### Lo que confirmó el usuario

| Tema | Decisión |
|---|---|
| Usuarios | Logística en oficina (PC) y conductores u operadores en ruta (celular) |
| Identidad | Se conservan el nombre, el logo del camión y el rojo `#E11D48` |
| Navegación | Barra superior + flujo de trabajo de Gastos; barra inferior en móvil |
| Dirección visual | "Hitos de carretera" (hitos kilométricos y señalización de la Panamericana Norte) |
| Git | Trabajar sobre `main`, sin rama ni commits (decisión del usuario) |

### Criterios de éxito

1. Desde cualquier gasto, el siguiente paso (adjuntar, revisar, ver) está a un clic.
2. Cada pantalla tiene URL propia: atrás, recargar y compartir enlaces funcionan.
3. Ningún formulario pide un ID numérico de otro módulo: se elige por nombre.
4. El diseño se ve coherente y propio en escritorio (1440 px) y en celular (390 px), sin patrones genéricos (brillos, degradados en texto, partículas, emojis como iconos).
5. `ng build` pasa con los presupuestos actuales y las pruebas unitarias pasan.

## 2. Alcance y restricciones

**Dentro:**
- todas las pantallas de `frontend/src`
- el sistema visual (tokens y CSS global)
- la navegación y las rutas
- el estado de datos del frontend
- `PRODUCT.md`, `DESIGN.md` y el surface brief de impeccable

**Fuera:**
- backend (FastAPI), servicio de gastos (Go), APIs y base de datos
- la lógica de sesión
- la gestión de usuarios (RF-19 a RF-21)
- los módulos ocultos (tipos de gasto, categorías, guías, viajes realizados, productos transportados, facturas, presupuestos)
- el tema oscuro: los colores quedan como tokens para sumarlo después

**Restricciones:**
- **Copy exacto (RNF-1).** `gasto-estados.ts` y `error-messages.ts` no se modifican. Los textos de validación, avisos y notas que ya existen en los componentes de gastos se conservan palabra por palabra donde la misma situación reaparece (ver §10).
- **Stack.** Angular 21 standalone + signals, CSS propio y sin librería de componentes.
- **Presupuestos de `angular.json`:**
  - bundle inicial: 500 kB de aviso y 1 MB de error
  - CSS por componente: 4 kB de aviso y **8 kB de error**
  - Por eso el sistema visual vive en CSS global y los componentes solo llevan estilos propios pequeños.
- **Dependencias nuevas:**
  - runtime: `@fontsource-variable/overpass` y `@fontsource/overpass-mono`
  - desarrollo, solo para pruebas: `vitest@^4` y `jsdom`
  - Los iconos se copian de Tabler Icons (licencia MIT) como datos de trazo, sin paquete en runtime.
- **Sin commits.** Antes de tocar código se respalda el estado actual de `frontend/src` (copia y `git diff`) fuera del repositorio.

## 3. Rutas

| URL | Pantalla | Notas |
|---|---|---|
| `/` | Landing | pública |
| `/login` | Iniciar sesión | lee `?registered=1` y `?volver=` |
| `/register` | Crear usuario | |
| `/dashboard` | redirige a `/app/inicio` | compatibilidad con enlaces viejos |
| `/app` | Shell autenticado (`authGuard`) | redirige a `inicio` |
| `/app/inicio` | Inicio | |
| `/app/gastos` | Bandeja de gastos | `?estado=sin-comprobante \| por-aprobar \| sustentados` (sin parámetro: todos) |
| `/app/gastos/nuevo` | Registrar gasto | |
| `/app/gastos/:id` | Detalle del gasto | |
| `/app/operacion/:modulo` | Módulos de Operación | `/app/operacion` redirige a `viajes` |
| `/app/catalogos/:modulo` | Módulos de Catálogos | `/app/catalogos` redirige a `ubigeo` |
| `/app/**` | redirige a `/app/inicio` | |
| `/**` | Página no encontrada | |

- Todas las pantallas se cargan con `loadComponent` (carga diferida).
- `provideRouter` agrega `withComponentInputBinding()`, `withViewTransitions()` y `withInMemoryScrolling({ scrollPositionRestoration: 'enabled' })`.
- `authGuard` redirige a `/login?volver=<url>`. Tras iniciar sesión se navega a `volver` si empieza con `/app`; si no, a `/app/inicio`.

**Slugs de módulos** (fuente única: `core/navigation/navegacion.ts`):

| Sección | Slug → clave CRUD |
|---|---|
| Operación | `viajes`→`viaje`, `clientes`→`cliente`, `conductores`→`conductor`, `flota`→`unidad`, `asignaciones`→`cond_uni`, `direcciones`→`direccion` |
| Catálogos | `ubigeo`→`ubigeo`, `rubros`→`rubro`, `marcas`→`marca`, `tipos-unidad`→`tipo_unidad`, `subtipos-unidad`→`subtipo_unidad` |

## 4. Shell y navegación

`ShellComponent` envuelve todo `/app` con estas piezas: enlace de salto, barra superior, cinta reflectiva, `<main id="contenido" tabindex="-1">`, barra inferior móvil, paleta de comandos y estado del servicio. Los avisos (toasts) viven en la raíz (`app.html`) para funcionar también en páginas públicas.

### Barra superior

- **Escritorio (≥ 1024 px):**
  - asfalto `#1E2124`, 56 px de alto
  - a la izquierda: marca (insignia roja con el camión y "LIZAR"), luego Inicio, Gastos, Operación y Catálogos
  - Gastos lleva un contador amarillo con los comprobantes pendientes
  - a la derecha: el botón "Buscar o ir a… ⌘K" y el menú de usuario (iniciales, nombre y rol; opción "Cerrar sesión")
- **Enlace activo:** `routerLinkActive` con `ariaCurrentWhenActive="page"` y un subrayado rojo de 3 px. Operación y Catálogos quedan activos en cualquiera de sus módulos.
- **Debajo de la barra:** cinta reflectiva de 4 px con segmentos rojo `#E11D48` y blanco de 24 px.
- **Tableta (768–1023 px):** el buscador se reduce a ícono.
- **Celular (< 768 px):** barra de 52 px con marca, título de la sección, botón de búsqueda y avatar.

### Barra inferior móvil (< 768 px)

- Cuatro destinos: Inicio, Gastos (con contador), Operación y Más.
- "Más" abre una hoja inferior con los módulos de Catálogos, los datos del usuario y "Cerrar sesión".
- Altura de 64 px más `env(safe-area-inset-bottom)`; objetivos táctiles de 44 px o más.

### Pestañas de sección

En `/app/operacion/*` y `/app/catalogos/*` aparece una fila de pestañas (enlaces) con los módulos de la sección. En celular se desplaza horizontalmente con `scroll-snap`.

### Paleta de comandos

- Se abre con ⌘K o Ctrl+K desde cualquier pantalla de `/app`, y también con el botón de búsqueda.
- Es un `<dialog>` nativo con `showModal()`, para tener foco atrapado y Esc.
- Incluye un campo "Buscar o ir a…" y tres grupos:
  - **Acciones:**
    - Registrar gasto → `/app/gastos/nuevo`
    - Adjuntar comprobante → `/app/gastos?estado=sin-comprobante`
    - Revisar comprobantes por aprobar → `/app/gastos?estado=por-aprobar`
  - **Ir a:** Inicio, Gastos y cada módulo de Operación y Catálogos.
  - **Gastos:** hasta 6 coincidencias por N° o por cliente, que llevan a `/app/gastos/:id`.
- La búsqueda ignora mayúsculas y tildes.
- Teclado:
  - ↑ y ↓ mueven la opción activa (`aria-activedescendant`)
  - Enter abre
  - Esc cierra
- Sin animación de apertura: se usa muchas veces al día.
- Reemplaza el buscador actual de la barra superior, que no hace nada.

### Estado del servicio

Usa `ApiService.health`. Solo se muestra cuando hay un problema: un chip "Servicio no disponible" en la barra, que reintenta al hacer clic. Cuando todo funciona no ocupa lugar.

## 5. Estado de datos

### `GastosStore` (`core/state/gastos.store.ts`)

Servicio raíz con signals, compartido por Inicio, la bandeja, el detalle, el contador de la barra y la paleta.

- **Estado:**
  - `gastos: Signal<Gasto[]>`
  - `comprobantes: Signal<Comprobante[]>`
  - `cargando`
  - `cargado`
  - `error`
- **Derivados (`computed`):**
  - conteo por estado de gasto
  - comprobantes pendientes
  - total por pestaña
- **`cargar()`:**
  - pide `/gastos` y `/comprobantes` en paralelo, con `timeout(HTTP_TIMEOUT_MS)`
  - si falla, guarda el error y avisa con `mensajeError(err)`
  - `asegurarCargado()` evita pedidos repetidos
- **Mutaciones:** devuelven `Observable` y llaman a `cargar()` al terminar bien.
  - `registrar(payload)`
  - `adjuntar(gastoId, archivo)`
  - `aprobar(comprobanteId)`
  - `rechazar(comprobanteId, motivo)`
- **Datos de cliente y viaje:** se resuelven con `CrudService` (claves `cliente` y `viaje`).

### Lógica pura (`core/state/gastos.logic.ts`, con pruebas)

- `filtrarPorPestana(gastos, pestana)` usa las pestañas `todos`, `sin-comprobante`, `por-aprobar` y `sustentados`. El mapeo a estados es:
  - `sin-comprobante` = Registrado
  - `por-aprobar` = Pagado
  - `sustentados` = Sustentado
- `contarPorEstado(gastos)`.
- `siguientePaso(gasto)` devuelve `'adjuntar' | 'revisar' | 'ninguno'`:
  - Registrado → adjuntar
  - Pagado → revisar
  - Sustentado → ninguno
- `comprobantesDe(comprobantes, gastoId)` devuelve los comprobantes del gasto, del más reciente al más antiguo.
- `ultimoRechazo(comprobantes, gastoId)` devuelve el último comprobante rechazado de un gasto Registrado, para mostrar su motivo.
- `pasosRuta(gasto, comprobantes)` devuelve los 3 hitos `{ estado, alcanzado, actual, fecha?, actor? }`:
  - Registrado toma `creado_en` y `creado_por`.
  - Pagado toma el `creado_en` del comprobante vigente.
  - Sustentado toma `aprobado_en` y `aprobado_por` del aprobado.
- `totalMonto(gastos)` y `buscarGastos(gastos, termino, nombreCliente)`.

### `CrudService`

- Agrega señales de carga por clave y `asegurarCargado(clave)`. Así se elimina el `setTimeout(600)` de `crud-view`, que hoy adivina cuándo terminó la petición.
- `guardar(clave, payload)` devuelve `Observable` para que la vista sepa cuándo cerrar el formulario. Mantiene sus avisos.
- `etiqueta(clave, id)` devuelve el nombre legible de un registro relacionado, o `#id` si la lista aún no carga.
- `refsDe(clave)` devuelve las claves relacionadas de un módulo, de forma transitiva. Por ejemplo, un viaje necesita la asignación, y la asignación necesita conductor y unidad.
- Se elimina la carga de los 18 módulos al entrar (incluidos los ocultos) y el aviso "Datos actualizados" en cada inicio. Cada pantalla carga lo que usa.

## 6. Pantallas

### 6.1 Inicio (`/app/inicio`)

- Título "Inicio".
- **Para atender:** dos contadores que son enlaces.
  - "Sin comprobante": gastos Registrados, lleva a `?estado=sin-comprobante`.
  - "Por aprobar": comprobantes pendientes, lleva a `?estado=por-aprobar`.
  - El de "Por aprobar" se pinta como campo amarillo de vía (`#F2C200`, tinta oscura) cuando es mayor que 0; el otro, en panel neutro con su poste gris. Con 0 se muestran en calma.
- **Acciones:** "Registrar gasto" (primaria) y "Adjuntar comprobante" (secundaria, lleva a la bandeja filtrada).
- **Recorrido de los gastos:**
  - La ruta grande de 3 hitos con el número de gastos y la suma de montos en cada estado.
  - Cada hito lleva a su pestaña.
  - Son datos reales del servicio, sin cifras inventadas.
- **Últimos gastos:** los 5 más recientes, con ruta compacta y acción siguiente.
- **Sin gastos todavía:**
  - Estado vacío compuesto: "Todavía no hay gastos registrados."
  - Botón Registrar gasto.
  - Recordatorio: se necesita al menos un viaje y un cliente, con enlaces a Viajes y Clientes.
- Carga con esqueletos de la forma final.

### 6.2 Bandeja de gastos (`/app/gastos`)

- Encabezado "Gastos" y botón "Registrar gasto".
- Pestañas con conteo: Todos, Sin comprobante, Por aprobar y Sustentados. Son enlaces con `queryParams`, así que se pueden compartir.
- Barra de herramientas: búsqueda por N° o cliente, y "Actualizar".
- **Escritorio (≥ 1024 px):** tabla con estas columnas:
  - N°, en mono
  - Cliente
  - Viaje
  - Fecha
  - Monto, con cifras tabulares, alineado a la derecha
  - Recorrido, con la ruta compacta
  - Acción
- **Comportamiento de las filas:**
  - Toda la fila abre el detalle (enlace estirado en la celda N°); la acción conserva su propio clic.
  - Acciones: "Adjuntar comprobante" en Registrado, "Revisar" en Pagado y "Ver" en Sustentado.
  - Si el gasto Registrado tuvo un rechazo, se muestra "Comprobante rechazado: {motivo}" bajo el cliente.
  - En "Todos", las filas Sustentado van con tinta atenuada: lo cerrado retrocede.
- **Al pie:** total de la pestaña alineado bajo la columna Monto.
- **Celular (< 768 px):** lista apilada.
  - Primera línea: N°, insignia de estado y monto.
  - Segunda línea: cliente y fecha.
  - Tercera línea: ruta compacta y botón de acción.
- **Estados:**
  - Carga con filas esqueleto.
  - Vacío por pestaña, por ejemplo "No hay comprobantes pendientes de aprobación." (copy existente) o "No hay gastos sin comprobante.".
  - Error en línea con "Reintentar".

### 6.3 Registrar gasto (`/app/gastos/nuevo`)

- Enlace "← Gastos" y título "Registrar gasto adicional" (copy existente). Formulario de una columna con máximo 560 px.
- **Campos**, con etiqueta arriba y error abajo usando los textos exactos de hoy:
  - Viaje: selector por nombre "Viaje N° 12 · 05/10/2026"; ya no usa raya larga.
  - Cliente: selector por razón social.
  - Monto: prefijo `S/` dentro del control e `inputmode="decimal"`.
  - Fecha: hoy por defecto.
- **Sin viajes o sin clientes:** aviso en línea con enlace "Crear viaje" o "Crear cliente" a su módulo.
- **Botones:** "Registrar gasto" (texto "Registrando…" mientras guarda) y "Cancelar", que vuelve a la bandeja.
- **Al guardar:**
  - aviso exacto "Gasto registrado correctamente. Ya puedes adjuntar el comprobante de pago."
  - navegación a `/app/gastos/:id` del gasto nuevo, que ya muestra el paso "Adjuntar comprobante"
- **Error 422:** el mensaje exacto bajo Viaje.
- **Celular:** controles a lo ancho, fuente de 16 px y botón de envío fijo abajo.

### 6.4 Detalle del gasto (`/app/gastos/:id`)

- **Encabezado:**
  - "← Gastos" vuelve a la bandeja con la pestaña de origen. La bandeja pasa `{ estado }` en el `state` de la navegación y el enlace lo usa como `queryParams`; sin origen, va a `/app/gastos`
  - "Gasto N° 31", la insignia de estado y el monto en grande
  - datos: cliente, viaje, fecha del gasto, y "Registrado por {creado_por} el {fecha}"
- **La ruta del gasto (firma),** en tamaño completo:
  - Muestra los 3 hitos con fecha y quién, y el camión del logo en el hito actual.
  - Es horizontal en escritorio y vertical en celular.
  - Si hubo rechazos, aparece una nota de desvío entre Registrado y Pagado: "Comprobante rechazado el {fecha}: {motivo}".
- **Siguiente paso**, según el estado:
  - **Registrado → Adjuntar comprobante:**
    - zona de arrastre con el texto existente "Arrastra el archivo aquí o haz clic para seleccionar"
    - en celular se reemplaza por un botón grande "Tomar foto o elegir archivo", con `accept` de imágenes y PDF y sin `capture`, para elegir cámara o archivos
    - ayuda y errores con el copy exacto
    - botón "Subir comprobante" (texto "Subiendo…" mientras sube)
    - si el anterior fue rechazado, un aviso con el motivo
  - **Pagado → Revisar comprobante:**
    - vista previa del comprobante pendiente: imagen en línea, o PDF con enlace "Abrir PDF"
    - botones "Aprobar" y "Rechazar"
    - "Rechazar" despliega en línea, sin modal, el área "Motivo del rechazo" con el placeholder existente y los botones "Confirmar rechazo" y "Cancelar", con error exacto si falta el motivo
  - **Sustentado → panel de cierre:** "Aprobado por {aprobado_por} el {fecha}" y "Ver comprobante".
- **Historial de comprobantes:** todos los del gasto, del más reciente al más antiguo, con insignia de estado de comprobante, fecha, motivo si fue rechazado y "Ver comprobante". Al pie va la nota exacta: "Este comprobante no se puede editar. Si es incorrecto, adjunta uno nuevo."
- **Tras subir, aprobar o rechazar:** aviso exacto, `GastosStore.cargar()` y el camión se desliza al nuevo hito.
- **Gasto inexistente:** estado vacío "No encontramos el gasto N° {id}." con enlace a Gastos.
- "Comprobante de pago" y "Aprobación" dejan de ser pantallas con desplegables: sus funciones viven aquí y en la bandeja.

### 6.5 Operación y Catálogos (`/app/operacion/:modulo`, `/app/catalogos/:modulo`)

`CrudPageComponent` reemplaza a `crud-view`. Un slug desconocido redirige al primer módulo de la sección.

- **Encabezado:** migas "Operación" o "Catálogos", el título del módulo, una descripción corta y el botón "Nuevo …" con el texto propio de cada módulo (por ejemplo "Nuevo viaje" o "Nueva dirección").
- **Pestañas de sección** como en §4.
- **Barra:** búsqueda, conteo "N registros" y "Actualizar".
- **Tabla primero.** Las columnas siguen `def.cols`. Las celdas resuelven nombres en lugar de IDs:
  - "Tracto" en vez de "Tipo #1"
  - la razón social en vez de "CLI-5"
  - la placa en vez de "🚛 #2"
  - "Juan Pérez · ABC-123" en vez de "Cond. #4 / Unid. #2"
- Las celdas dejan de usar emojis. Cada fila tiene la acción "Ver registro", que abre el JSON en un diálogo.
- **Formulario en hoja lateral:**
  - "Nuevo …" abre un `<dialog>` de 440 px pegado a la derecha en escritorio, y a pantalla completa desde abajo en celular.
  - Campos con etiqueta arriba, ayuda y error abajo; pares de campos en dos columnas solo en escritorio.
  - Pie fijo con "Guardar" y "Cancelar".
  - Al guardar bien: cierra, recarga y avisa.
  - Si hay errores: enfoca el primer campo inválido.
- **Campos relacionados** (`tp: 'ref'`): usan `SelectorReferencia`, un combobox con búsqueda (ver §7.3).

  | Módulo | Campo actual | Pasa a |
  |---|---|---|
  | Subtipos | ID Tipo unidad | Tipo de unidad |
  | Direcciones | Código Ubigeo | Ubigeo (distrito, provincia y código) |
  | Clientes | ID Rubro | Rubro |
  | Flota | ID Tipo unidad · ID Subtipo · ID Marca | Tipo de unidad · Subtipo · Marca |
  | Asignaciones | ID Conductor · ID Unidad | Conductor · Unidad |
  | Viajes | ID Tracto · ID Carreta · ID Asignación C/U · ID Cliente · ID Dir. Origen · ID Dir. Destino | Tracto · Carreta · Asignación · Cliente · Dirección de origen · Dirección de destino |
- **Sin registros previos:** si una lista relacionada está vacía, el selector muestra "Aún no hay {plural}." y un enlace "Crear {singular}" al módulo correspondiente.
- **Estado vacío del módulo:**
  - "Aún no hay {plural}." y el botón "Nuevo …"
  - para módulos con relaciones, la cadena de datos previa con enlaces; por ejemplo "Para crear un viaje necesitas: asignación, cliente y direcciones"
- Carga con esqueletos de filas.

### 6.6 Landing (`/`)

Modo Persuadir dentro del mismo mundo.

- **Barra pública asfalto** con la marca y un único botón: "Ingresar" (a `/login`), o "Ir a Inicio" si hay sesión. Debajo, la cinta reflectiva.
- **Portada (cabe en la primera vista, sin indicador de scroll):**
  - **Columna izquierda (7/12):**
    - titular de máximo 2 líneas: "Gastos de viaje con su comprobante, de la ruta a la oficina."
    - bajada de máximo 20 palabras: "Registra cada gasto adicional, adjunta su comprobante desde el celular y apruébalo desde la oficina."
    - botón "Ingresar" y enlace secundario "Crear usuario"
  - **Columna derecha (5/12):**
    - un panel asfalto que es la carretera: una línea central discontinua con los 3 hitos y los **conteos reales** del servicio de gastos (registrados, pagados, sustentados y comprobantes por aprobar)
    - si el servicio no responde, se muestra "—" como hoy
- **Cómo avanza un gasto:**
  - Los 3 hitos con verbo y efecto: "Registra" lleva a Registrado, "Adjunta" a Pagado y "Aprueba" a Sustentado.
  - Cada uno con una frase corta y la insignia del estado.
  - Horizontal en escritorio, vertical en celular. Sin rótulos "Paso 1/2/3".
- **Desde la oficina y desde la ruta:**
  - Composición asimétrica con una foto real verificada de un camión en carretera. Se reutiliza una de las URLs actuales si muestra eso y responde; si no, se elige otra de Unsplash y se verifica.
  - Dos bloques: "En la oficina" (revisar y aprobar) y "En la ruta" (adjuntar desde el celular).
- **Pie:** "LIZAR SRL · Trujillo, Perú" y "© 2026".
- **Se eliminan:**
  - partículas, scanline y orbes
  - texto con degradado
  - píldoras con emoji
  - indicador "Scroll"
  - el cursor personalizado del CSS viejo
- **Movimiento:** una sola entrada al cargar: la línea de la carretera se traza (`clip-path`, 700 ms) y los hitos aparecen con 60 ms de escalonado. Con "reducir movimiento", queda estática.

### 6.7 Iniciar sesión (`/login`)

- **Escritorio:** dos columnas.
  - A la izquierda, un panel asfalto (40 %) con la marca, "Fleet Manager Pro" y la carretera vertical con los 3 hitos sin cifras.
  - A la derecha, el formulario sobre fondo blanco hito.
- **Celular:** marca compacta arriba y el formulario.
- **Textos:** "Iniciar sesión" y "Ingresa tus credenciales para acceder al sistema".
- **Aviso de registro:** si llega `?registered=1`, se muestra "Usuario creado. Ya puedes iniciar sesión.".
- **Campos:**
  - Usuario y Contraseña, con etiquetas en tipo oración (ya no en mayúsculas).
  - El botón de mostrar u ocultar la contraseña es accesible: se puede enfocar y tiene `aria-label` "Mostrar contraseña" u "Ocultar contraseña".
- **Envío y error:**
  - error con `role="alert"`
  - botón "Ingresar al sistema" (texto "Verificando…" mientras valida)
- **Al pie:**
  - la pista "Usuario por defecto: admin / admin123"
  - "¿Necesitas una cuenta? Crear usuario"

### 6.8 Crear usuario (`/register`)

- Mismo marco que el login.
- Campos en 2 columnas en escritorio y 1 en celular.
- Opción "Mostrar contraseñas".
- Botón "Crear cuenta" ("Creando usuario…" mientras guarda).
- Lógica y validaciones sin cambios.

### 6.9 Página no encontrada (`/**`)

"Página no encontrada", "La dirección que abriste no existe." y el botón "Ir al inicio".

## 7. Sistema visual

### 7.1 Tokens (`src/styles/tokens.css`)

**Paleta y rol:**

| Token | Valor | Rol |
|---|---|---|
| `--asfalto` | `#1E2124` | barra superior, paneles de carretera, tinta principal |
| `--asfalto-2` | `#2A2E33` | hover sobre asfalto |
| `--asfalto-3` | `#3A3F45` | filetes sobre asfalto |
| `--asfalto-tinta` | `#C9CDD1` | texto secundario sobre asfalto |
| `--suelo` | `#F7F7F5` | fondo de página (blanco hito) |
| `--panel` | `#FDFDFC` | superficies elevadas |
| `--panel-2` | `#F0F1EE` | hover, hundidos, esqueletos |
| `--linea` | `#E2E3E0` | filetes de 1 px |
| `--linea-2` | `#C9CCC8` | bordes de controles |
| `--tinta` | `#1E2124` | texto principal |
| `--tinta-2` | `#4A5056` | texto secundario |
| `--tinta-3` | `#656B71` | texto terciario (≥ 4.5:1 sobre `--suelo`) |
| `--rojo` | `#E11D48` | marca, acción principal, foco, cinta |
| `--rojo-hover` / `--rojo-presionado` | `#C8113D` / `#AD0E35` | estados del botón primario |
| `--rojo-suave` / `--rojo-tinta` | `#FDECEF` / `#B4123A` | fondos y texto de alerta |
| `--via` / `--via-suave` / `--via-tinta` | `#F2C200` / `#FFF5CC` / `#6B5300` | atención y "pendiente" |

**Estados** (colores fijados por el producto; coinciden con la señalización vial):

| Estado | Señal vial | Punto / hito | Fondo | Texto | Clase existente |
|---|---|---|---|---|---|
| Registrado | — | `#8A9096` | `#EEEFEC` | `#4A5056` | `lbl-n` |
| Pagado | informativa (azul) | `#2563EB` | `#E8EFFE` | `#1E4FC2` | `lbl-info` |
| Sustentado / Aprobado | guía (verde) | `#15803D` | `#E6F4EA` | `#166534` | `lbl-ok` |
| Pendiente de aprobación | preventiva (amarilla) | `#F2C200` | `#FFF5CC` | `#6B5300` | `lbl-warn` |
| Rechazado | reglamentaria (roja) | `#E11D48` | `#FDECEF` | `#B4123A` | `lbl-err` |

Las clases `lbl-*` se mantienen porque las usa `gasto-estados.ts` (que no se toca); el CSS nuevo las dibuja como insignias del sistema.

**Tipografía:**
- `--fuente: 'Overpass Variable', 'Overpass', system-ui, sans-serif`.
- `--fuente-mono: 'Overpass Mono', ui-monospace, Menlo, monospace`, para N°, placas, RUC, DNI y códigos.
- Cifras con `font-variant-numeric: tabular-nums` en montos, conteos y tablas.
- Escala fija en rem (razón aproximada de 1.2): 12 · 13 · 14 · 15 (texto base) · 16 · 18 · 20 · 24 · 32 · 40 px.
- Pesos: 400 texto, 500 etiquetas, 600 botones y títulos, 700 cifras grandes, 800 marca.
- Interlineado: 1.45 en texto, 1.2 en títulos y 1.05 en cifras grandes.
- Los títulos no van en mayúsculas.

**Espaciado:** base de 4 px (`--e-1` a `--e-16`: 4, 8, 12, 16, 20, 24, 32, 40, 48, 64). Contenido con máximo 1240 px y márgenes de 32 px en escritorio y 16 px en celular.

**Forma:**
- Regla única de radios:
  - controles e insignias: 6 px
  - paneles, hojas y diálogos: 10 px
  - contadores dentro de pestañas o de la barra: 999 px
- Filetes de 1 px.
- Las sombras, teñidas de asfalto, solo se usan en superficies flotantes (menú, hoja, paleta, toast):
  - `--sombra-1: 0 1px 2px rgb(30 33 36 / .06)`
  - `--sombra-2: 0 8px 24px rgb(30 33 36 / .12), 0 2px 6px rgb(30 33 36 / .08)`

**Capas:** barra 100, barra inferior 110 y avisos 300. Los diálogos usan la capa superior nativa de `<dialog>`.

**Densidad:**
- filas de tabla de 48 px
- controles de 40 px en escritorio y 44 px en celular
- texto de controles de 16 px en celular para evitar el zoom de iOS

### 7.2 La firma: `RutaGastoComponent`

La ruta del gasto: tres hitos sobre una línea de carretera, con el camión del logo en el actual.

- **Entradas:** `pasos` (de `pasosRuta`) y `variante: 'compacta' | 'completa'`.
- **Hito:** un poste cuadrado de 12 px (compacta) o 20 px (completa) con borde de 2 px del color del estado.
  - Relleno cuando ya se alcanzó; solo borde cuando está pendiente.
  - El tramo de línea entre hitos alcanzados va sólido en `--tinta-2`; el pendiente, discontinuo en `--linea-2`.
- **Camión:** el pictograma del logo, en blanco sobre insignia roja (16 px compacta, 28 px completa), sobre el hito actual.
- **Compacta** (aproximadamente 88 px de ancho): solo postes y camión, con `aria-label`, por ejemplo "Recorrido: Registrado, Pagado (actual), Sustentado pendiente".
- **Completa:**
  - debajo de cada hito, el nombre del estado (copy exacto), la fecha y quién
  - horizontal en ≥ 768 px; vertical en celular, con la línea a la izquierda
  - la nota de rechazo se dibuja como un desvío entre el primer y el segundo hito
- **Movimiento:**
  - Cuando el estado avanza en la misma vista, el camión se desliza al hito nuevo (`transform`, 420 ms, `--ease-in-out`).
  - Con "reducir movimiento" el cambio es instantáneo.
  - Al primer render no se anima.

### 7.3 Componentes

- **Botones** (`.btn`):
  - Variantes:
    - `primario`: rojo, texto blanco
    - `secundario`: panel con borde
    - `fantasma`
    - `peligro`: tinte rojo
    - `sobre-asfalto`
  - Tamaños `md` (40 px) y `sm` (32 px), más el modo ícono.
  - Al presionar: `scale(0.97)`.
  - Deshabilitado: opacidad .45 y sin cursor de mano.
  - El texto nunca se parte en dos líneas.
- **Campos** (`.campo`):
  - Etiqueta arriba (13 px, 500), asterisco rojo en los obligatorios, ayuda y error abajo (13 px).
  - Error con borde rojo y `aria-invalid`, más `aria-describedby` hacia ayuda y error.
  - Nunca se usa el placeholder como etiqueta.
- **`SelectorReferencia`** (combobox):
  - Input con `role="combobox"` y `aria-expanded`, `aria-controls` y `aria-activedescendant`.
  - La lista de opciones se despliega **dentro del flujo**, debajo del input, para que no se recorte dentro de hojas con scroll. Tiene 240 px máximo, `role="listbox"` y un máximo de 50 opciones visibles filtradas.
  - El filtro ignora mayúsculas y tildes.
  - Teclado: ↑, ↓, Enter y Esc.
  - Es valor de dos vías con `model()`.
  - Lista vacía: "Aún no hay {plural}." con enlace para crear.
- **Insignia de estado** (`InsigniaEstadoComponent`): usa `ESTADO_*_LABEL` y `ESTADO_*_CLASS`, más un poste de 8 px del color del estado. Ese es el único punto de color con significado.
- **Tabla** (`.tabla`):
  - Encabezados de 12 px, 600, en tipo oración, con `--tinta-2`.
  - Filas separadas por un solo filete inferior; hover `--panel-2` en dispositivos con puntero.
  - Números a la derecha.
- **Pestañas** (`.pestanas`): enlaces con contador y subrayado rojo de 3 px en la activa.
- **Estado vacío** (`EstadoVacioComponent`): ícono, título, frase y una acción. Enseña qué hacer.
- **Esqueleto** (`.esqueleto`): bloques `--panel-2` con la forma final y un pulso de opacidad lento.
- **Hoja** (`.hoja`, `<dialog>`): lateral derecha en escritorio y desde abajo en celular. El fondo usa `::backdrop` con `rgb(30 33 36 / .4)`.
- **Diálogo** (`.dialogo`, `<dialog>`): centrado; se usa solo para "Ver registro".
- **Aviso** (`ToastComponent`):
  - Abajo a la derecha en escritorio; abajo y centrado sobre la barra inferior en celular.
  - Usa `role="status"` y `aria-live="polite"`.
  - Lleva un ícono por tipo (no letras sueltas) y un botón cerrar con `aria-label`.
- **Menú de usuario:** un popover anclado arriba a la derecha.

### 7.4 Iconos y marca

- **Iconos:** un subconjunto de Tabler Icons (outline, MIT) copiado a `shared/ui/icono/iconos.ts`.
  - Se usan con `<app-icono nombre="…">` y trazo de 1.75.
  - Reemplazan todos los trazos dibujados a mano y los emojis de las celdas.
  - El archivo lleva el aviso de licencia.
- **Marca:**
  - El pictograma del camión de hoy conserva su geometría (compromiso de marca), sobre una insignia roja plana de 32 px con radio de 8 px, sin degradado ni brillo.
  - Al lado van "LIZAR" (800) y "Fleet Manager Pro" (500, 12 px), en los lugares donde cabe.

### 7.5 Fuentes y `index.html`

- `styles.css` importa `@fontsource-variable/overpass` y `@fontsource/overpass-mono` (400 y 600) y deja de usar Google Fonts.
- `index.html`:
  - quita el `style` en línea del `body`
  - agrega `meta description`
  - agrega `meta name="theme-color" content="#1E2124"`
  - agrega `viewport-fit=cover` para las zonas seguras
- Cada ruta tiene título propio, por ejemplo "Gastos · LIZAR".

## 8. Movimiento

Reglas de Emil Kowalski aplicadas a una herramienta de trabajo: rápido, con propósito, y nada en acciones de teclado.

```css
--ease-out:     cubic-bezier(0.23, 1, 0.32, 1);
--ease-in-out:  cubic-bezier(0.77, 0, 0.175, 1);
--ease-drawer:  cubic-bezier(0.32, 0.72, 0, 1);
--d-press: 140ms; --d-fast: 180ms; --d-base: 240ms; --d-hoja: 320ms; --d-ruta: 420ms;
```

| Elemento | Frecuencia | Movimiento | Tiempo |
|---|---|---|---|
| Botón al presionar | muy alta | `scale(0.97)` | 140 ms `--ease-out`, solo `transform` |
| Hover de filas y botones | alta | color y fondo | 150 ms `ease`, solo con `(hover: hover) and (pointer: fine)` |
| Cambio de pantalla | alta | View Transition: fundido cruzado | 160 ms entrada / 120 ms salida |
| Paleta ⌘K | muy alta | **ninguno** | — |
| Menú de usuario | media | opacidad y `scale(0.97)` desde arriba a la derecha | 160 ms entrada / 120 ms salida |
| Hoja de formulario | media | `translateX(100%)` (escritorio) o `translateY(100%)` (celular) | 320 ms `--ease-drawer` entrada / 200 ms salida |
| Diálogo "Ver registro" | baja | opacidad y `scale(0.97)`, centrado | 200 ms `--ease-out` |
| Aviso | media | `@starting-style`: `translateY(16px)` y opacidad | 240 ms entrada / 150 ms salida (transiciones, no keyframes) |
| Camión de la ruta | baja | `translateX` (o `translateY` en vertical) al hito nuevo | 420 ms `--ease-in-out` |
| Esqueleto | — | pulso de opacidad | 1.2 s, infinito |
| Landing: trazo de la ruta | una vez | `clip-path: inset(0 100% 0 0)` hacia `inset(0)` con hitos escalonados 60 ms | 700 ms `--ease-in-out` |

- Solo se animan `transform`, `opacity` y `clip-path`.
- Nunca se anima desde `scale(0)`.
- Las salidas son más rápidas que las entradas.
- Con `prefers-reduced-motion: reduce`:
  - sin desplazamientos ni escalas
  - opacidades de 150 ms o menos
  - sin View Transition
  - el esqueleto queda estático
- Se elimina todo el movimiento decorativo actual: pulsos, partículas, scanline, `translateY` en hover de tarjetas y brillos.

## 9. Accesibilidad y celular

- Enlace "Saltar al contenido": un botón que enfoca `#contenido`, para evitar que un `href="#…"` recargue la landing por el `<base href="/">`.
- Se elimina la regla global `*:focus { outline: none }`. El foco visible es un anillo de 2 px `--rojo` con separación de 2 px (`:focus-visible`).
- Hitos semánticos:
  - `header`, `nav` (con `aria-label`) y `main`
  - `aria-current` en navegación y pestañas
  - avisos en región `aria-live`
- Contraste AA en todo el texto y los controles (verificado con script y con el detector de impeccable).
- Objetivos táctiles de 44 px o más en celular.
- Uso de `env(safe-area-inset-*)`.
- `100dvh` en lugar de `100vh`.
- `-webkit-tap-highlight-color: transparent` con estado `:active` visible.
- `touch-action: manipulation` en controles.
- Al implementar se aplica la guía `mobile-native` (Emil Kowalski).

## 10. Copy

- **Se conservan textualmente:**
  - etiquetas de estado (`gasto-estados.ts`) y mensajes de error de red (`error-messages.ts`)
  - en gastos y comprobantes: validaciones, avisos de éxito, "Formatos permitidos: JPG, PNG o PDF. Tamaño máximo 10 MB.", "Arrastra el archivo aquí o haz clic para seleccionar", "Este comprobante no se puede editar. Si es incorrecto, adjunta uno nuevo.", "Motivo del rechazo" con su placeholder, "Confirmar rechazo" y "No hay comprobantes pendientes de aprobación."
- **Textos nuevos:** tipo oración, verbos concretos, sin raya larga (—), sin signos de exclamación y sin emojis.
- **Cambios de formato:**
  - Las etiquetas propias que hoy usan raya larga pasan a usar "·", por ejemplo "Viaje N° 12 · 05/10/2026" y "Gasto N° 31 · S/ 120.00".
  - "Sistema en linea" pasa a escribirse con tilde donde aparezca.
- **Montos:** `formatMonto` usa `Intl.NumberFormat('es-PE')` con 2 decimales y separador de miles, por ejemplo "S/ 1,240.50".

## 11. Archivos

**Crear** (`frontend/src/`):
- `styles/tokens.css`, `styles/base.css`, `styles/componentes.css`, `styles/shell.css`, `styles/paginas.css` (inicio, gastos, landing y acceso)
- `app/core/navigation/navegacion.ts`
- `app/core/state/gastos.store.ts` y `app/core/state/gastos.logic.ts`, con sus `.spec.ts`
- `app/shell/shell.component.{ts,html}` y, dentro de `app/shell/`:
  - `barra-superior/`
  - `barra-movil/`
  - `paleta-comandos/`
  - `pestanas-seccion/`
- dentro de `app/shared/ui/`:
  - `icono/` (componente e `iconos.ts`)
  - `marca/`
  - `insignia-estado/`
  - `ruta-gasto/`
  - `estado-vacio/`
  - `selector-referencia/`
- `app/features/inicio/`
- dentro de `app/features/gastos/`:
  - `gastos-lista/`
  - `gasto-detalle/`
  - `adjuntar-comprobante/`
  - `revision-comprobante/`
- `app/features/crud-page/`
- `app/features/no-encontrada/`

**Modificar:**
- configuración: `app.routes.ts`, `app.config.ts`, `app.ts`, `app.html`, `index.html`, `styles.css`, `angular.json` (objetivo `test`), `package.json` y `tsconfig.spec.json`
- modelos y servicios: `core/models/crud.model.ts`, `core/models/cruds.data.ts`, `core/services/crud.service.ts`, `core/guards/auth.guard.ts` y `core/utils/format.ts`
- componentes y pantallas: `shared/components/toast/toast.component.ts`, `features/gastos/gasto-form/*`, `features/landing/*`, `features/login/*` y `features/register/*`

**Eliminar:**
- `features/dashboard/*`
- `features/crud-view/*`, reemplazado por `crud-page`
- `features/gastos/comprobante-upload/*` y `features/gastos/aprobacion-comprobantes/*`, reemplazados por `adjuntar-comprobante` y `revision-comprobante`
- `shared/components/sidebar/*` y `shared/components/topbar/*`
- `shared/components/modal/*`, que nadie usa
- `assets/css/*`, reemplazado por `styles/*`

**Raíz del repositorio:** `PRODUCT.md` (hecho), el surface brief de impeccable con el contrato de dirección, y `DESIGN.md` con `.impeccable/design.json` al final.

## 12. Verificación

1. **Respaldo** de `frontend/src` y del `git diff` en el scratchpad antes de empezar.
2. **Pruebas unitarias** (Vitest con `@angular/build:unit-test`) para `gastos.logic.ts`, `navegacion.ts` (cada módulo visible tiene slug y sección, sin duplicados), `CrudService.refsDe` y la resolución de etiquetas, y `format.ts`.
3. **`ng build`** de producción sin errores de presupuesto.
4. **Recorrido visual con datos simulados:**
   - `playwright-core`, instalado solo en el scratchpad, maneja el Chrome del sistema e intercepta la API con datos de ejemplo.
   - Las capturas se toman en 1440×900 y 390×844 de:
     - landing, login e inicio
     - gastos en sus 4 pestañas
     - detalle en sus 3 estados (y con rechazo)
     - registrar gasto
     - un módulo de Operación con la hoja abierta
     - un módulo de Catálogos
     - la paleta abierta
   - Además se recorren a mano los flujos: registrar → adjuntar → aprobar, rechazar con motivo, y crear un cliente eligiendo el rubro por nombre.
5. **`impeccable detect --json`** sobre los archivos cambiados; se corrige lo mecánico.
6. **Revisión final:** un revisor de impeccable en contexto limpio, con capturas, este spec y el contrato de dirección. Luego el documentador escribe `DESIGN.md`.
7. **Pre-flight de Taste** en la landing y el acceso:
   - sin rayas largas
   - portada en la primera vista
   - un solo CTA por intención
   - sin indicador de scroll
   - contraste de botones y formularios

## 13. Riesgos

| Riesgo | Mitigación |
|---|---|
| Trabajar sobre `main` sin commits | Respaldo previo fuera del repo; cambios por etapas verificadas con `ng build` |
| Listas grandes en selectores (Ubigeo tiene ~1 900 distritos) | Combobox con filtro y máximo 50 opciones visibles |
| CSS de componente > 8 kB rompe el build | Sistema visual en CSS global; componentes con estilos mínimos |
| View Transitions no existe en Firefox | Angular lo detecta y navega sin transición |
| Sin backends locales para probar | Datos simulados interceptados en el navegador para el recorrido visual |
