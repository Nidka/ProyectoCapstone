# Pendientes del software — LIZAR SRL (G04)

Encargo para completar el sistema según el Documento de Requerimientos de Software (DRS v2.1).
Fecha de corte: 22/10/2026.

## Contexto

- Un solo sistema: `gastos-service` (Go 1.25.4, pgx v5, net/http, puerto 8081) más `frontend` (Angular 21, standalone, signals, Vitest).
- PostgreSQL local, base «Cloude». Migraciones en `gastos-service/migrations/` (la última es `0005_lectura_comprobante.sql`; la siguiente debe ser `0006_…`). El runner reaplica todas al iniciar, así que cada migración debe ser idempotente (`IF NOT EXISTS`).
- La API Python de `backend/` ya fue migrada a Go. No se usa, pero tampoco se borra (la retira el equipo).
- Ya hecho, no tocar salvo que haga falta: gastos, comprobantes JPG/PNG/PDF, aprobar/rechazar con motivo, checklist por cliente, filtros, reportes BI con PDF/Excel, lectura OCR (monto, fecha, RUC), catálogos REQ-37 a REQ-47, login JWT (`/auth/login`, `/auth/me`).

## Reglas de trabajo

1. Trabajar en `main`, sin ramas ni commits (decisión del equipo).
2. No detener ni ocupar los puertos 8000, 4200. El 8081 se puede reiniciar al final para cargar las migraciones nuevas. Para pruebas manuales usar otros puertos (por ejemplo 8093 con 4300) y apagarlos al terminar.
3. Las pruebas de integración usan esquemas temporales (`internal/pruebas/esquema.go`, `NuevoEsquemaCompleto`); nunca escriben en las tablas reales.
4. No modificar `frontend/src/app/**/gasto-estados.ts` ni `error-messages.ts`.
5. Textos de la interfaz en español sencillo (RNF-3); estilos solo en `frontend/src/styles/*.css` con los tokens existentes.
6. TDD: cada cambio con su prueba. Al cerrar: `cd gastos-service && gofmt -l . && go vet ./... && go test ./...` limpio; `cd frontend && npx ng test --watch=false && npx ng build` limpio.

## Bloque 1 — Login real en el backend (RNF-6, REQ-31) · hacer primero

Hoy los endpoints de gastos no validan el token: el rol llega en la cabecera `X-User-Role` que envía el frontend (`internal/handlers/middleware.go`, `requireLogistica`).

- Middleware que exige `Authorization: Bearer <JWT>` en todas las rutas salvo `/health`, `/auth/login`; responde 401 sin token válido o con usuario desactivado.
- El rol se toma del token, no de cabeceras. Eliminar `X-User-Role` / `X-User-Name` del backend y del frontend (interceptor HTTP que añade el Bearer).
- Roles: `admin` (Administrador del Sistema) con `logistica` (Encargado de Logística). Migrar los usuarios `operador` existentes a `logistica`.
- Reglas: solo `logistica` registra, edita, adjunta, aprueba o rechaza gastos; solo `admin` gestiona usuarios; 403 en otro caso.
- `POST /auth/register` deja de ser público: solo `admin`.
- Frontend: guardas de ruta por rol; ocultar menús que el rol no puede usar.

## Bloque 2 — Gestión de usuarios (REQ-28 a REQ-32)

- Endpoints (solo `admin`): `GET /usuarios`, `POST /usuarios`, `PUT /usuarios/{id}`, `POST /usuarios/{id}/desactivar`, `POST /usuarios/{id}/activar`.
- No se borra ningún usuario; se desactiva (`estado = false`) y ya no puede iniciar sesión.
- REQ-32: columna `debe_cambiar_clave BOOLEAN NOT NULL DEFAULT TRUE`. Si es verdadera, el login responde con esa marca, el frontend lleva a la pantalla «Cambiar contraseña» y bloquea el resto hasta cambiarla. Endpoint `POST /auth/cambiar-clave` (clave actual + nueva, mínimo 8 caracteres).
- Pantalla «Usuarios» (lista, crear, editar, desactivar) visible solo para `admin`.

## Bloque 3 — Bitácora de auditoría (REQ-33 a REQ-36, RNF-9 a RNF-12)

- Tabla `bitacora`: `id`, `fecha` (timestamptz), `id_usuario`, `usuario` (nombre), `accion` (texto corto: `gasto.registrar`, `comprobante.aprobar`, `usuario.desactivar`, `sesion.inicio`, …), `entidad`, `id_entidad`, `detalle` (jsonb con valores antes/después).
- Registrar dentro de la misma transacción cada cambio de gasto, comprobante, checklist, usuario. Regla de negocio: todo cambio genera su entrada.
- RNF-9/10: trigger en PostgreSQL que rechaza `UPDATE` o `DELETE` sobre `bitacora` para todos.
- RNF-11/12: registrar inicio de sesión (en `/auth/login`) con cierre (nuevo `POST /auth/logout`, que el frontend llama al salir).
- REQ-36: `GET /gastos/{id}/historial` con una sección «Historial de cambios» en el detalle del gasto (fecha, usuario, acción, en orden cronológico).

## Bloque 4 — Notificaciones (REQ-20, REQ-21, RNF-8)

- REQ-21 necesita una fecha de cobro: agregar `fecha_cobro DATE NULL` al gasto (opcional en el formulario de registro).
- Tarea periódica dentro del servicio Go (cada hora, con un `time.Ticker`) que genera notificaciones en una tabla `notificacion` sin duplicarlas:
  - REQ-20: gastos con más de 3 días sin comprobante adjunto.
  - REQ-21: gastos con checklist incompleto a 2 días o menos de su fecha de cobro.
- RNF-8: la notificación debe existir antes de 24 horas desde que se cumple la condición (la frecuencia horaria lo cumple).
- `GET /notificaciones` con `POST /notificaciones/{id}/leida`, solo `logistica`.
- Frontend: campana en la barra superior con el número de no leídas; al pulsar, lista con enlace al gasto.

## Bloque 5 — Pendientes menores

- RNF-14: el servicio escribe los errores técnicos en un archivo (`LOG_ARCHIVO`, por defecto `logs/errores.log`) además de la consola.
- RNF-5: al guardar un comprobante, dejar el archivo en solo lectura (`os.Chmod(ruta, 0o444)`).
- RNF-4: script `gastos-service/scripts/respaldo.sh` que copia `uploads/` a una carpeta de respaldo con fecha; documentarlo en el README para programarlo con cron.
- OCR (hallazgo de la revisión final): el frontend corta la lectura a los 30 s igual que el servidor, así que en lecturas lentas el usuario ve un error de conexión. Subir el tiempo del frontend a 35 s en `lectura.service.ts` (con su prueba).
- Opcional: `brew install tesseract tesseract-lang` para que el OCR lea también imágenes escaneadas.

## Al terminar

- Reiniciar el servicio del 8081 (`cd gastos-service && go run .`) para aplicar las migraciones 0005 en adelante.
- Dejar en `docs/RESULTADO_PENDIENTES.md`: endpoints nuevos (método, ruta, rol, códigos HTTP), migraciones creadas, número de pruebas Go con Angular, cobertura (`go test -cover ./...`, `npx ng test --watch=false --coverage`). Ese archivo lo usa el equipo para actualizar los documentos del curso.
