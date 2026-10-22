# Migración del backend de Python a Go — diseño

**Fecha:** 2026-10-22 · **Estado:** diseño aprobado en conversación; esta especificación espera revisión.
**Prioridad:** va antes que el OCR (`2026-10-22-ocr-comprobantes-design.md`, en pausa).

## 1. Objetivo

Todo el backend corre en **un solo servicio Go** (`gastos-service`, puerto 8081). El servicio FastAPI (`backend/`, puerto 8000) deja de ser necesario y el frontend usa un único servidor sin cambiar su comportamiento.

Éxito: con el Python apagado, cada pantalla del frontend funciona igual que antes, y un test de contrato demuestra que cada respuesta de Go coincide con la que daba Python.

## 2. Alcance

**Entra:**
- Las 118 operaciones de la API de Python: 19 recursos × 6 operaciones (listar, obtener, crear, actualizar, borrar, lista avanzada), más `/auth/login`, `/auth/register`, `/auth/me` y `/health`.
- Autenticación (bcrypt + JWT HS256), creación de la tabla `usuario` y siembra del admin.
- Cambio de la dirección base del frontend, `Dockerfile` y `docker-compose.demo.yml`.
- Pruebas de contrato contra respuestas guardadas de Python.

**No entra:**
- Proteger rutas hoy abiertas ni roles reales (RF-19 a RF-21): se mantiene que solo `/auth/me` exige sesión.
- Borrar la carpeta `backend/` (lo decide el usuario al final) ni renombrar `gastos-service`.
- SQLite: el servicio Go solo usa PostgreSQL.
- Cambios de comportamiento de la API: se porta tal cual, errores incluidos.

## 3. Recursos (fuente de verdad: `openapi.json` de Python)

Los 19 recursos, con su ruta y su clave primaria:

| Ruta (`/api/v1/…`) | Tabla | Clave |
|---|---|---|
| ubigeos | ubigeo | codigo_ubigeo |
| rubros | rubro | id_rubro |
| marcas | marca | id_marca |
| tipos-unidad | tipounidad | id_tipo_unidad |
| subtipos-unidad | subtipounidad | id_subtipo_unidad |
| tipos-gasto | tipogasto | id_tipo_gasto |
| categorias-producto | productocategoria | id_categoria |
| direcciones | direccion | id_direccion |
| conductores | conductor | id_conductor |
| clientes | cliente | id_cliente |
| unidades | unidad | id_unidad |
| asignaciones-conductor-unidad | conductorunidad | id_conductor_unidad |
| viajes | programacionviaje | id_viaje |
| guias-transportista | guiatransportista | id_guia_transportista |
| viajes-realizados | viajerealizado | id_viaje_realizado |
| facturas | factura | id_factura |
| presupuestos-viaje | presupuestoviaje | id_presupuesto |
| detalles-presupuesto | detallepresupuesto | id_detalle_presupuesto |
| productos-transportados | productotransportado | id_producto_transportado |

Columnas, tipos, obligatoriedad y campos de búsqueda de cada recurso salen de `backend/app/infrastructure/db/models.py` y de los `schemas.py` de cada módulo; el plan las transcribe a una tabla de definiciones en Go (`internal/crud/recursos.go`).

## 4. Contrato de cada operación

Para cada recurso `R` con clave `K`:

| Operación | Ruta | Éxito | Errores |
|---|---|---|---|
| Listar | `GET /R?limit=50&offset=0` | 200 lista (`limit` 1–200, `offset` ≥ 0) | 422 si el límite está fuera de rango |
| Obtener | `GET /R/{K}` | 200 objeto | 404 `{"detail": …}` |
| Crear | `POST /R` | 201 objeto creado | 422 si falta un campo obligatorio o el tipo no coincide |
| Actualizar | `PUT /R/{id}` | 200 objeto | 404 `{"detail": "Recurso no encontrado"}` |
| Borrar | `DELETE /R/{id}` | 200 `{"deleted": true, "<K>": id}` | 404 |
| Avanzada | `GET /advanced/R` | 200 lista | 422 |

Lista avanzada: `limit`, `offset`, `search` (coincidencia parcial sin distinguir mayúsculas sobre `search_fields` o, por defecto, los campos de búsqueda del recurso; los campos desconocidos se ignoran), `order_by` (campo válido o la clave), `order_dir` (`asc` o `desc`, otro valor da 422) y `filters` (`campo:valor,campo:valor`, comparación exacta; los campos desconocidos se ignoran).

Errores en `/api/v1`: JSON `{"detail": "<mensaje>"}`; los 422 de validación llevan `detail` como lista, igual que FastAPI. Una violación de llave foránea o de unicidad en la base responde con el mismo código y mensaje que hoy (a confirmar con el test de contrato; si Python respondía 500, Go responde 500).

## 5. Autenticación

- `POST /auth/login` `{username, password}` → 200 `{access_token, token_type: "bearer", usuario}`; 401 `Credenciales inválidas` o `Usuario inactivo`.
- `POST /auth/register` → 201 `usuario`; 409 si el usuario o el email ya existen; `rol` solo `admin` u `operador` (422 si no).
- `GET /auth/me` con `Authorization: Bearer` → 200 `usuario`; 401 `No autenticado`, `Token inválido` o `Usuario no encontrado`.
- Contraseñas con bcrypt (los hashes existentes siguen valiendo). Token JWT HS256 con `sub`, `rol`, `id` y `exp` a 8 horas.
- La clave se lee de `JWT_SECRET`; sin ella se usa la clave actual de Python, para que los tokens ya emitidos sigan siendo válidos. Anotado como deuda: cambiar esa clave y el usuario `admin/admin123` en producción.
- Siembra: al arrancar, si no hay usuarios, se crea `admin` / `admin123`.
- Migración `0003_usuario.sql` (idempotente): tabla `usuario` con las columnas de `UsuarioModel`.

## 6. Salud, CORS y configuración

- `GET /health` responde `{"status": "ok", "env": <APP_ENV>}` además de lo que ya devuelve el servicio; si ambos formatos chocan, se conserva el de Python (la única diferencia, el campo `env`).
- CORS: el existente, con `Content-Disposition` expuesto.
- Variables: `PORT`, `DATABASE_URL`, `UPLOADS_DIR` (existentes), `JWT_SECRET` y `APP_ENV` (nuevas, opcionales).

## 7. Código

- `internal/crud/`: `motor.go` (listar, obtener, crear, actualizar, borrar y avanzada sobre una definición de recurso, con SQL parametrizado y nombres de columna tomados solo de la definición, nunca del usuario), `validacion.go` (obligatoriedad y tipos, con el formato de error de FastAPI), `recursos.go` (las 19 definiciones).
- `internal/auth/`: hash, token, repositorio de usuarios y manejadores.
- `internal/handlers/`: registra las rutas de `/api/v1` junto a las existentes.
- Dependencias nuevas: `golang.org/x/crypto` (bcrypt) y `github.com/golang-jwt/jwt/v5`, ambas en una versión publicada hace más de dos semanas.

## 8. Frontend y despliegue

- `core/services/api.service.ts`: la base local por defecto pasa de `http://localhost:8000/api/v1` a `http://localhost:8081/api/v1`; las bases guardadas que apunten a `localhost:8000` se reemplazan como hoy ya se hace con `127.0.0.1:8000`. La base remota por defecto (`13.222.175.52:8000`) no se toca: depende del despliegue.
- `Dockerfile` del servicio Go y `docker-compose.demo.yml` sin el servicio `api` de Python, con el servicio Go en el 8081.
- La documentación de arranque se actualiza.

## 9. Pruebas

- **Captura (antes de cualquier cambio):** con el Python corriendo sobre una base temporal con `db.sql`, un script guarda la respuesta de cada una de las 118 operaciones (casos de éxito, 404, 422 y filtros) como archivos de referencia en `gastos-service/internal/crud/contrato/`.
- **Contrato (Go):** las mismas llamadas contra el motor de Go sobre un esquema temporal con el DDL de `db.sql`; se compara la forma del JSON (claves, tipos numéricos, fechas, orden) ignorando valores generados (ids, hora).
- **Unitarias:** validación, filtros, `order_by`, hash y token (incluido un token emitido por Python, que Go debe aceptar).
- **Frontend:** ajuste de la prueba de la base de la API.
- **Navegador:** recorrido de Operación y Catálogos (crear, editar, buscar, borrar) y el login, con Python apagado.

## 10. Riesgos y decisiones

- **Tipos numéricos:** Python serializa `Numeric` de una forma específica (número o texto); el test de contrato lo fija y Go lo imita. Es el riesgo principal de compatibilidad.
- **Fechas y zonas horarias:** se conserva el formato de Python (`T` sin zona para `timestamp`, `YYYY-MM-DD` para `date`).
- **Validación:** se replica lo que Pydantic exige hoy; no se agregan validaciones nuevas, para no cambiar el comportamiento.
- **Seguridad heredada:** rutas abiertas, clave JWT por defecto y admin sembrado; se documentan, no se cambian en esta migración.
- **Reversibilidad:** `backend/` queda intacto y el frontend vuelve a la base anterior con un solo cambio hasta que el usuario decida retirarlo.
