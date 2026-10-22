# Checklist documental del comprobante — diseño

**Fecha:** 2026-10-10 · **Tarea del cronograma:** 3.2.2 «Programar el checklist documental» (Módulo de checklist), vence el 16/10/2026.
**Estado:** diseño aprobado en conversación; esta especificación espera revisión.

## 1. Objetivo

Antes de aprobar un comprobante de pago, el Encargado de Logística verifica una lista fija de criterios documentales. Solo puede aprobar si el comprobante cumple todos. El resultado queda guardado junto al comprobante, con quién y cuándo lo revisó, y se ve en el historial del gasto.

Éxito: no existe forma de dejar un gasto «Sustentado» sin un checklist completo guardado, ni desde la pantalla ni llamando a la API.

## 2. Alcance

**Entra:**
- Cinco criterios fijos, iguales para todos los comprobantes.
- Validación en el servidor de gastos (Go): aprobar exige los cinco criterios; solo se revisa un comprobante Pendiente.
- Tabla nueva `checklist_comprobante` y su migración.
- Checklist en la revisión del comprobante (detalle del gasto en estado Pagado) y en el historial de comprobantes.
- Motivo de rechazo propuesto a partir de los criterios no cumplidos.

**No entra:**
- Criterios configurables o un catálogo de criterios.
- Gestión de usuarios y roles reales (RF-19 a RF-21). La persona con sesión sigue actuando como Encargado de Logística.
- Checklist de documentos del viaje (guía, factura) o del vehículo y el conductor.
- Comprobantes revisados antes de este módulo: se muestran sin checklist; no se completa nada hacia atrás.

## 3. Criterios

Orden fijo. La clave es la que viaja en la API y se guarda en la base.

| Clave | Texto en la revisión | Texto corto (historial) |
|---|---|---|
| `legible` | El comprobante es legible y está completo. | Legible y completo |
| `tipo_valido` | Es una boleta o factura válida. | Boleta o factura válida |
| `a_nombre_empresa` | Está emitido a nombre de LIZAR SRL. | A nombre de LIZAR SRL |
| `monto_coincide` | El monto coincide con el gasto (S/ 85.50). | Monto coincide |
| `fecha_coincide` | La fecha corresponde al gasto (30/09/2026). | Fecha coincide |

El monto y la fecha entre paréntesis salen del gasto (`formatMonto(gasto.monto)` y `formatFecha(gasto.fecha)`), para que quien revisa compare sin salir de la pantalla.

## 4. Reglas

1. **Aprobar** exige los cinco criterios presentes y marcados como cumplidos. La pantalla deshabilita el botón hasta entonces y el servidor lo vuelve a comprobar.
2. **Rechazar** guarda el checklist tal como esté (los cinco criterios, cumplidos o no). Si hay criterios sin marcar al abrir el rechazo, el motivo se propone con sus textos cortos, en minúscula y separados por punto y coma: «No cumple: monto coincide; fecha coincide.». El motivo sigue siendo obligatorio y editable; el mensaje «Debes indicar un motivo para rechazar el comprobante.» no cambia (RNF-1).
3. **Solo un comprobante Pendiente** se aprueba o se rechaza. Uno ya Aprobado o Rechazado responde 409. Esto también impide que una aprobación se deshaga con un rechazo posterior.
4. **Quién y cuándo:** se reutilizan `comprobante.aprobado_por` y `comprobante.aprobado_en`, que ya se llenan al aprobar y al rechazar. La tabla del checklist no los duplica.
5. Todo cambio de una revisión (checklist, comprobante y gasto) ocurre en una sola transacción que bloquea el comprobante (`SELECT … FOR UPDATE`).

## 5. Datos

Migración nueva `gastos-service/migrations/0002_checklist_comprobante.sql` (idempotente, como la 0001):

```sql
CREATE TABLE IF NOT EXISTS checklist_comprobante (
    id             SERIAL PRIMARY KEY,
    comprobante_id INTEGER NOT NULL REFERENCES comprobante (id),
    criterio       VARCHAR(30) NOT NULL
                   CHECK (criterio IN ('legible', 'tipo_valido', 'a_nombre_empresa', 'monto_coincide', 'fecha_coincide')),
    cumple         BOOLEAN NOT NULL,
    CONSTRAINT uq_checklist_criterio UNIQUE (comprobante_id, criterio)
);

CREATE INDEX IF NOT EXISTS idx_checklist_comprobante_id ON checklist_comprobante (comprobante_id);
```

## 6. API (gastos-service)

### `POST /comprobantes/{id}/aprobar`
Cuerpo (ahora obligatorio):
```json
{ "checklist": { "legible": true, "tipo_valido": true, "a_nombre_empresa": true, "monto_coincide": true, "fecha_coincide": true } }
```
| Caso | Respuesta |
|---|---|
| Rol distinto de logística | 403 (sin cambios) |
| `id` inválido o cuerpo no JSON | 400 |
| Clave desconocida en `checklist` | 422 `criterio desconocido: <clave>` |
| Falta un criterio o alguno es `false` | 422 `para aprobar, el comprobante debe cumplir los 5 criterios del checklist` |
| Comprobante inexistente | 404 |
| Comprobante no Pendiente | 409 `el comprobante ya fue revisado` |
| Éxito | 200 `{"status":"aprobado"}`; guarda las 5 filas, el comprobante queda Aprobado y el gasto Sustentado |

### `POST /comprobantes/{id}/rechazar`
Cuerpo: `{ "motivo_rechazo": "…", "checklist": { …5 criterios con true o false… } }`. `checklist` es opcional para no romper clientes antiguos; si viene, debe traer las cinco claves conocidas (422 si no). Mismos 403, 400, 404 y 409 que aprobar. El motivo vacío sigue dando 400. En éxito guarda las filas del checklist si vino, el comprobante queda Rechazado y el gasto vuelve a Registrado.

### `GET /comprobantes`
Cada comprobante incluye `checklist`: lista ordenada según la tabla de la sección 3, `[{"criterio":"legible","cumple":true}, …]`. Es una lista vacía si el comprobante no tiene checklist.

## 7. Diseño del código

**Go**
- `internal/checklist/checklist.go`: las cinco claves en orden y las funciones puras `ValidarAprobacion(map[string]bool) error`, `ValidarRechazo(map[string]bool) error` y `Ordenar(...)`. Sin base de datos.
- `internal/handlers/comprobantes.go`: aprobar y rechazar usan esas funciones, verifican el estado con `FOR UPDATE` y escriben el checklist en la misma transacción. `listComprobantes` carga los checklists de los comprobantes listados con una segunda consulta (`WHERE comprobante_id = ANY($1)`).
- `internal/models/models.go`: `Comprobante.Checklist []ItemChecklist` con `criterio` y `cumple`.

**Angular**
- `core/models/checklist.ts`: tipo `CriterioChecklist`, la lista `CRITERIOS_CHECKLIST` (clave, texto y texto corto) y las funciones puras `checklistVacio()`, `checklistCompleto()` y `motivoDesdeChecklist()`.
- `core/models/gasto.model.ts`: `Comprobante.checklist?`.
- `GastosStore.aprobar(id, checklist)` y `rechazar(id, motivo, checklist)` envían el checklist.
- `RevisionComprobanteComponent` recibe además el `gasto` (para el monto y la fecha de los criterios). Entre la vista del comprobante y los botones muestra un `fieldset` «Checklist documental» con las cinco casillas (objetivos de 44 px en celular), el contador «3 de 5 criterios» y, mientras falten, el aviso «Marca los 5 criterios para aprobar.», enlazado al botón con `aria-describedby`.
- Detalle del gasto, historial «Comprobantes»: para cada comprobante con checklist, los cinco textos cortos con ✓ o ✗ (iconos `check` y `x` del registro) y «Revisado por {aprobado_por} el {aprobado_en}».
- Estilos con las clases y los tokens existentes (`.casilla`, `.historial`); sin colores nuevos.

## 8. Errores en pantalla

`error-messages.ts` es texto obligatorio (RNF-1) y no se modifica. El componente de revisión distingue dos respuestas antes de recurrir a `mensajeError`:
- **409:** aviso «Este comprobante ya fue revisado. Actualizamos la vista.» y recarga del store.
- **422:** aviso «Para aprobar, el comprobante debe cumplir los 5 criterios del checklist.»
- Cualquier otro error: `mensajeError(err)` como hasta ahora.

Los avisos de éxito existentes no cambian.

## 9. Pruebas

- **Go, unitarias** (`internal/checklist`): aprobación completa válida; falta un criterio; uno en false; clave desconocida; rechazo con criterios en false válido; rechazo sin checklist válido.
- **Go, integración** (`internal/handlers`): contra PostgreSQL, en un esquema temporal propio con tablas mínimas `programacionviaje` y `cliente`, migraciones aplicadas y borrado al terminar. Se ejecutan con `TEST_DATABASE_URL` y se saltan si no está definida. Casos: aprobar guarda las 5 filas y deja el gasto Sustentado; aprobar incompleto da 422 sin cambios; aprobar o rechazar un comprobante ya revisado da 409; rechazar guarda el checklist y devuelve el gasto a Registrado; el listado devuelve el checklist ordenado.
- **Vitest:** lógica del checklist; Aprobar deshabilitado hasta marcar los cinco; envío del checklist al aprobar y rechazar; motivo propuesto; mensajes de 409 y 422; historial con ✓/✗ y revisor.
- **Navegador:** el simulador acepta y devuelve el checklist, y el recorrido de punta a punta marca los criterios antes de aprobar.

## 10. Riesgos y decisiones

- El rol sigue simulado (cabecera `X-User-Role` que envía el frontend). El checklist no resuelve eso; depende de RF-19 a RF-21.
- Aprobar ahora exige cuerpo: un cliente antiguo que llame sin checklist recibe 422. El único cliente es este frontend, que se actualiza en el mismo cambio.
- Fecha límite: 16/10/2026.
