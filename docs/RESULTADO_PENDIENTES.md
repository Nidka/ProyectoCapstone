# Resultado de los pendientes del software — LIZAR SRL (G04)

Cierre del encargo `docs/PENDIENTES_SOFTWARE.md` (bloques 1 al 5). Fecha: 22/10/2026.
Todo lo que sigue está verificado contra el código y contra ejecuciones reales de hoy; lo que no se pudo verificar está en «Pendientes y límites».

## 1. Estado general

| Bloque | Qué quedó | Estado |
|---|---|---|
| 1 Login real (RNF-6, REQ-31) | Middleware JWT en todas las rutas salvo las públicas; roles `admin` y `logistica`; frontend con interceptor y guardas por rol | Hecho |
| 2 Usuarios (REQ-28 a REQ-32) | 5 endpoints de gestión, cambio de contraseña obligatorio, pantallas «Usuarios» y «Cambiar contraseña» | Hecho |
| 3 Bitácora (REQ-33 a REQ-36, RNF-9 a RNF-12) | Tabla inmutable, auditoría dentro de la misma transacción, login/logout, historial por gasto en el detalle | Hecho |
| 4 Notificaciones (REQ-20, REQ-21, RNF-8) | `fecha_cobro`, generador cada hora, campana con lista | Hecho |
| 5 Menores | Log a archivo (RNF-14), comprobante en solo lectura (RNF-5), script de respaldo (RNF-4), tiempo del OCR a 35 s | Hecho (tesseract opcional no instalado) |

## 2. Endpoints

Todas las rutas exigen `Authorization: Bearer <JWT>` y usuario activo (se consulta la BD en cada petición), salvo las públicas de la sección 3. El rol sale de la BD, no del token. Sin token o con token malo: 401 `{"detail": ...}` (mensajes «No autenticado», «Token inválido», «Usuario no encontrado», «Usuario inactivo») con `WWW-Authenticate: Bearer`.

Mientras `debe_cambiar_clave` sea verdadera, el servicio responde 403 `{"detail":"Debes cambiar tu contraseña"}` a todo salvo `GET /api/v1/auth/me`, `POST /api/v1/auth/cambiar-clave` y `POST /api/v1/auth/logout`.

Rol: «cualquiera» = cualquier usuario autenticado (`admin` o `logistica`). Los endpoints de gastos no llevan prefijo; los de auth, usuarios y maestros van bajo `/api/v1`.

### 2.1 Nuevos

| Método y ruta | Rol | Códigos HTTP |
|---|---|---|
| `GET /api/v1/usuarios` | admin | 200 (arreglo, activos e inactivos); 401; 403 `{"detail":"Solo el administrador puede gestionar usuarios"}` |
| `POST /api/v1/usuarios` | admin | 201; 401; 403; 409 (usuario o correo repetido); 422 (campos inválidos) |
| `PUT /api/v1/usuarios/{id}` | admin | 200; 401; 403; 404; 409 (correo de otro, o dejaría al sistema sin admin activo); 422 |
| `POST /api/v1/usuarios/{id}/desactivar` | admin | 200; 401; 403; 404; 409 (a sí mismo, o último admin); 422 (id no numérico) |
| `POST /api/v1/usuarios/{id}/activar` | admin | 200; 401; 403; 404; 422 |
| `POST /api/v1/auth/cambiar-clave` | cualquiera (también con la clave pendiente) | 200; 400 (clave actual incorrecta, o nueva igual a la actual); 401; 422 (faltan campos o nueva de menos de 8) |
| `POST /api/v1/auth/logout` | cualquiera | 200 `{"detail":"Sesión cerrada"}`; 401; 500 si falla la bitácora |
| `GET /gastos/{id}/historial` | cualquiera | 200 (arreglo ordenado por fecha, `[]` si no hay); 400 id inválido; 401; 404 |
| `GET /notificaciones` | logistica | 200 `{"no_leidas":N,"notificaciones":[...]}` (tope de 100 filas; no leídas primero); 401; 403 |
| `POST /notificaciones/{id}/leida` | logistica | 200 (idempotente); 400 id inválido; 401; 403; 404; 500 si falla la bitácora |

No existe `DELETE /api/v1/usuarios/{id}`: ningún usuario se borra (405).

Notas del contrato de usuarios:
- Se decidió el prefijo `/api/v1/usuarios` en lugar de `/usuarios` (como decía el encargo), por coherencia con `/api/v1/auth` y con la base del frontend.
- Usuario de salida: `{id_usuario, username, email, nombre, rol, estado, debe_cambiar_clave}`; nunca sale el hash.
- `POST /api/v1/usuarios`: `{username, email, password, nombre, rol?}`. Rol por defecto `logistica`. Queda con `debe_cambiar_clave = true`. Los 422 de contraseña llevan `input: null` para no repetirla.
- `PUT`: parcial (nombre, email, rol, password). El `username` no cambia. Si llega `password`, la restablece y vuelve a exigir el cambio.
- El último administrador activo no se puede desactivar ni degradar (se serializa con un candado de PostgreSQL).

### 2.2 Cambiados

| Método y ruta | Rol | Qué cambió |
|---|---|---|
| `POST /api/v1/auth/login` | pública | 200 devuelve además `debe_cambiar_clave` (dentro de `usuario` y en la raíz) y registra `sesion.inicio`; si la bitácora falla responde 500 y no entrega token. Los intentos fallidos (401) no se registran |
| `GET /api/v1/auth/me` | cualquiera | Incluye `debe_cambiar_clave` |
| `POST /api/v1/auth/register` | admin | Antes público. 403 `{"detail":"Solo el administrador puede registrar usuarios"}` a logistica; 401 sin token; 422 si el rol no es `admin` o `logistica`; rol por defecto `logistica`. El frontend ya no tiene ruta de registro |
| `POST /gastos` | logistica | 403 `{"error": ...}` a admin. Acepta `fecha_cobro` (AAAA-MM-DD, opcional): 400 si el formato es malo o si es anterior a la fecha del gasto. `creado_por` sale del usuario autenticado (se ignora el del cuerpo). Registra `gasto.registrar` |
| `GET /gastos`, `GET /gastos/{id}`, `POST /gastos` (respuesta) | cualquiera en los GET | La respuesta trae `fecha_cobro` (ISO o `null`). `creado_en` ahora sale como instante real (ver hallazgo H6) |
| `POST /gastos/{id}/comprobantes` | logistica | 403 a admin. El archivo queda en modo 0444 (RNF-5). Nombre nuevo `gasto-<id>-<32 hex aleatorios><ext>`. Si la operación no se confirma, se borra el archivo. Registra `comprobante.adjuntar`. Sigue: 201; 400 (archivo ausente o mayor de 5 MB); 404 (gasto); 422 (tipo no permitido) |
| `POST /comprobantes/{id}/aprobar` y `/rechazar` | logistica | 403 a admin. `aprobado_por` sale del usuario autenticado (se ignora el del cuerpo). Registran `comprobante.aprobar` y `comprobante.rechazar` con checklist y motivo. 200; 400; 404; 422 (checklist incompleto al aprobar) |
| `PUT` y `DELETE /clientes/{id}/checklist` | logistica | 403 a admin. Registran `checklist.configurar` y `checklist.restablecer` |
| `GET /comprobantes`, `GET /comprobantes/{id}/lectura`, `GET /gastos/exportar.xlsx`, `GET /clientes/{id}/checklist`, `GET /reportes/*` | cualquiera | Solo cambia que ahora exigen token |
| CRUD de maestros `/api/v1/*` (19 recursos y `/advanced/*`) | cualquiera, lectura y escritura | Solo exigen token. Las escrituras corren en transacción y registran `maestro.crear`, `maestro.editar` y `maestro.eliminar` |
| `OPTIONS` (CORS) | pública | `Allow-Headers` ahora es `Content-Type, Authorization`; se quitaron `X-User-Role` y `X-User-Name` en backend y frontend |

### 2.3 Acciones que quedan en la bitácora

`gasto.registrar`, `comprobante.adjuntar`, `comprobante.aprobar`, `comprobante.rechazar`, `checklist.configurar`, `checklist.restablecer`, `usuario.crear`, `usuario.editar`, `usuario.desactivar`, `usuario.activar`, `usuario.cambiar_clave`, `maestro.crear`, `maestro.editar`, `maestro.eliminar`, `sesion.inicio`, `sesion.cierre`, `notificacion.generar` (autor «Sistema») y `notificacion.leer`. El historial del gasto (`GET /gastos/{id}/historial`) muestra solo las de entidad `gasto` y `comprobante` de ese gasto.

## 3. Excepciones y decisiones de seguridad

- Públicas (sin token): `GET|HEAD /health`, `POST /api/v1/auth/login`, `GET|HEAD /uploads/*` y `OPTIONS`.
- `GET /uploads/<archivo>` queda sin token a propósito: el navegador carga los comprobantes con `<img>` y `<a href>`, que no pueden enviar cabeceras. Mitigaciones: `/uploads/` y cualquier carpeta dan 404 (no hay listado) y los nombres nuevos llevan 128 bits aleatorios. Los comprobantes subidos antes de este cambio conservan su nombre anterior (`gasto-<id>-<hora>`), deducible por fuerza bruta. Cerrarlo del todo exige servir los archivos con token (descarga con `fetch` o URLs firmadas), un cambio de frontend que no se hizo.
- La clave JWT ya no tiene valor por defecto: `JWT_SECRET` es obligatoria, de al menos 32 caracteres y distinta de la clave pública antigua; sin ella el servicio no arranca (`log.Fatalf` con el motivo). Un token sin `exp` se rechaza.
- Los 403 de gastos, comprobantes, checklist y notificaciones usan `{"error": ...}`; los de usuarios, register y los 401 usan `{"detail": ...}`. El frontend lee ambas formas.

## 4. Migraciones (`gastos-service/migrations/`)

Todas idempotentes (el runner reaplica todas en cada arranque). Las pruebas las reaplican varias veces con datos dentro.

| Archivo | Qué hace |
|---|---|
| `0005_lectura_comprobante.sql` | Caché de la lectura OCR por comprobante (anterior a este encargo; es la que faltaba cargar en el 8081) |
| `0006_roles.sql` | `UPDATE usuario SET rol='logistica' WHERE rol='operador'` y `DEFAULT 'logistica'` para `rol`. No agrega CHECK de roles: se validan en la API |
| `0007_usuario_gestion.sql` | `ALTER TABLE usuario ADD COLUMN IF NOT EXISTS debe_cambiar_clave BOOLEAN NOT NULL DEFAULT TRUE`. Al crearse la columna, los usuarios existentes quedan en TRUE una sola vez |
| `0008_bitacora.sql` | Tabla `bitacora` (id, fecha timestamptz con `clock_timestamp()`, id_usuario sin FK, usuario, accion, entidad, id_entidad, detalle jsonb), índices por (entidad, id_entidad) y por fecha, y triggers que rechazan UPDATE, DELETE (por fila) y TRUNCATE (por sentencia), incluso al dueño de la tabla |
| `0009_notificaciones.sql` | `gasto.fecha_cobro DATE` con CHECK `fecha_cobro IS NULL OR fecha_cobro >= fecha`; tabla `notificacion` (tipo `sin_comprobante` o `cobro_proximo`, UNIQUE por (tipo, gasto), `leida`, `leida_en`) e índice por (leida, creada_en) |

## 5. Comportamiento de las notificaciones (decisiones)

- REQ-20: un gasto cuenta como «sin comprobante» si no tiene ningún comprobante Pendiente ni Aprobado; se cuenta desde el registro (`creado_en`), y «más de 3 días» es estricto (72 horas exactas no avisa).
- REQ-21: «checklist incompleto» se implementó como gasto aún no Sustentado, con `fecha_cobro` a 2 días o menos del «hoy» de Lima, incluida la fecha vencida.
- Un aviso se crea una sola vez por (tipo, gasto) para siempre. El texto se fija al crearse y no se actualiza con los días.
- El generador corre al arrancar y cada hora (`time.Ticker`), todo o nada por pasada, y se detiene con SIGINT/SIGTERM (el servidor ahora cierra con `Shutdown`, 10 s).
- Abrir la campana no marca nada como leído; se marca al pulsar un aviso. No hay «marcar todas», porque el servicio no tiene ese endpoint.

## 6. Pruebas y cobertura (ejecutadas hoy, 22/10/2026, al cierre)

Comandos: `cd gastos-service && gofmt -l . && go vet ./... && go test ./...` con `TEST_DATABASE_URL` apuntando a PostgreSQL (esquemas temporales); `cd frontend && npx ng test --watch=false --coverage && npx ng build`.

| Medición | Resultado |
|---|---|
| `gofmt -l .` | vacío |
| `go vet ./...` | limpio |
| `go test -count=1 -v ./...` | 241 pruebas de nivel superior en PASS, 0 FAIL, 1 SKIP; 268 líneas PASS contando subpruebas |
| Las de integración | Corrieron contra PostgreSQL (con `TEST_DATABASE_URL` definida). El único SKIP es `TestTesseractConUnaImagenGenerada`, porque tesseract (opcional) no está instalado |
| Angular (Vitest) | 61 archivos, 459 pruebas, todas en verde |
| `ng build` | Sin errores ni avisos |

Cobertura de Go (`go test -cover`, por paquete, solo con sus propias pruebas): `checklist` 97.1 %, `ocr` 96.6 %, `registro` 90.9 %, `notificaciones` 87.3 %, `reportes` 86.1 %, `handlers` 80.5 %, `bitacora` 57.1 %, `config` 50.0 %, `crud` 36.1 %, `auth` 23.5 %, `main` 0 %, `db` 0 %. Total de esa medición: 61.1 %. Esa cifra subestima, porque `auth`, `crud` y `bitacora` se ejercitan sobre todo desde las pruebas de `handlers`.
Cobertura cruzada (`go test -coverpkg=./... `, cuenta lo que ejecutan las pruebas de cualquier paquete): total 86.1 %; por paquete `checklist` 98.6 %, `ocr` 96.6 %, `reportes` 92.9 %, `bitacora` 92.9 %, `registro` 90.9 %, `auth` 89.3 %, `crud` 88.0 %, `notificaciones` 87.3 %, `handlers` 80.5 %, `db` 75.0 %, `config` 50.0 %. `main.go` queda en 0 % (no tiene pruebas).

Cobertura de Angular (`ng test --coverage`, v8): sentencias 92.59 % (4453/4809), ramas 85.70 % (1883/2197), funciones 80.66 % (726/900), líneas 93.45 % (2855/3055). El paquete `@vitest/coverage-v8` no estaba en el proyecto: se instaló con `npm i --no-save` (no cambió `package.json` ni el lock), así que quien quiera repetir la medición debe instalarlo igual.

## 7. Verificación manual de extremo a extremo (puerto 8093 y `ng serve` en 4300)

Se usó una base PostgreSQL aparte (`lizar_e2e`, cargada con `db.sql`; ya borrada), no la base real, para no ensuciar la bitácora inmutable. El binario nuevo corrió en 8093 y el frontend en 4300 (las llamadas a 8081 se redirigieron a 8093 desde el navegador de prueba). Ambos se apagaron al terminar.

Por HTTP, con resultados reales:
- `GET /health` 200; `GET /gastos` sin token 401 «No autenticado».
- Login de admin: 200 con `debe_cambiar_clave=true`; `GET /gastos` con esa clave pendiente: 403 «Debes cambiar tu contraseña»; `GET /auth/me`: 200.
- `cambiar-clave`: 400 con clave actual mala, 422 con clave nueva corta, 200 correcto; luego `GET /gastos` da 200 con el mismo token.
- Crear usuario `logistica`: 201; repetido: 409 «El usuario 'ana.e2e' ya existe»; listar: 200. La logistica con clave pendiente: 403; tras cambiar la clave, accede.
- `POST /gastos` como admin: 403; como logistica: 201 (`creado_por` = nombre autenticado, ignorando el del cuerpo, con `fecha_cobro`); con cobro anterior al gasto: 400. Adjuntar como admin: 403; como logistica: 201 (archivo con modo `-r--r--r--`). Aprobar como admin: 403; como logistica con checklist incompleto: 422; con 5 de 5: 200 y el gasto pasa a Sustentado.
- `GET /gastos/{id}/historial`: 3 entradas en orden (`gasto.registrar`, `comprobante.adjuntar`, `comprobante.aprobar`); id inválido 400; inexistente 404.
- Notificaciones: tras reiniciar el servicio, la pasada creó «Mañana es el cobro del gasto N° 2 (23/10/2026) y su comprobante aún no está aprobado.»; `GET /notificaciones` 200 a logistica y 403 a admin; `POST .../leida` 200 y repetido 200 sin duplicar la bitácora.
- Desactivar a la logistica: 200; con su token vigente recibe 401 «Usuario inactivo» de inmediato; reactivar: 200; el admin no puede desactivarse a sí mismo: 409. Logout 200 con token; 401 sin token. `GET /uploads/` da 404.
- La bitácora quedó con 16 entradas en el recorrido por HTTP; `UPDATE` y `TRUNCATE` sobre ella fallaron con «La bitácora no admite modificar ni borrar registros».

En Chrome headless (Playwright), sin errores de consola, sin fallos de red ni respuestas HTTP 4xx inesperadas en ninguna pantalla:
- Usuario con clave pendiente: el login lo lleva a `/cambiar-clave`; intentar abrir `/app/inicio` lo devuelve allí; valida «Usa al menos 8 caracteres» y «Las contraseñas no coinciden»; al cambiar entra a `/app/inicio`.
- Admin: 6 enlaces en la barra superior (incluye «Usuarios»), sin campana; lista de usuarios, creación por el panel, confirmación de desactivar; `/app/gastos/nuevo` lo redirige a Inicio; ve el «Historial de cambios».
- Logistica: ve la campana con su insignia, la lista de avisos (escritorio y celular de 390 px de ancho), el aviso lleva a `/app/gastos/3` y baja la insignia a 0; el formulario tiene «Fecha de cobro (opcional)» con el error local «La fecha de cobro no puede ser anterior a la fecha del gasto.»; el detalle muestra la fecha de cobro y el «Historial de cambios»; salir lleva a `/login`; `/app/usuarios` lo devuelve a Inicio.

## 8. Hallazgos de revisión

Corregidos antes de esta etapa (con prueba que falló primero):
- H1. Clave JWT pública por defecto: ahora es obligatoria y validada (ver sección 3).
- H2. `/uploads/` listaba carpetas: ahora 404, y los nombres nuevos son no adivinables.
- H3. `reportes_demo_limpiar.sql` fallaba por la FK de `notificacion`: ahora borra primero `notificacion`.
- H4. Los 500 no dejaban causa en el log: ahora todos pasan por `writeFallo` (método, ruta, mensaje y causa) y llegan a `LOG_ARCHIVO`.

Encontrados y corregidos en esta verificación final:
- H5. La campana aparecía suelta abajo a la derecha de la pantalla, fuera de la barra superior. Causa: el contenedor de la campana usaba la clase `.avisos`, que ya es la de los toasts fijos (`position: fixed; bottom; right`) en `componentes.css`; las pruebas unitarias no lo detectan porque no cargan los estilos. Corrección: el contenedor ahora es `.campana` (`campana.component.html` y una regla en `styles/avisos.css`), con una prueba nueva que falló antes. Verificado en navegador: la campana queda en la barra (y=10) y la lista se abre debajo.
- H6. Las horas del detalle del gasto salían 5 horas atrasadas (13:55 al lado de 18:55 del historial). Causa: `gasto.creado_en`, `comprobante.creado_en` y `comprobante.aprobado_en` son `TIMESTAMP` sin zona con la hora de pared del servidor (Lima), y el servicio las enviaba marcadas como UTC (`Z`); el frontend las muestra en hora de Lima. Corrección: las consultas las leen con `AT TIME ZONE current_setting('TimeZone')` (función `instante` en `handlers.go`), de modo que salen como instante real; prueba nueva (`instantes_integration_test.go`) que falló en sus 4 comprobaciones antes del cambio. Las filas ya guardadas en la base real quedan bien interpretadas porque el servidor de base de datos está en `America/Lima`. Si algún día el servidor cambiara de zona horaria, habría que migrar esas columnas a `timestamptz`.

No corregidos:
- Los comprobantes subidos antes del cambio de nombres siguen siendo públicos por su URL exacta (ver sección 3).
- Un superusuario de PostgreSQL aún puede desactivar los triggers de la bitácora; para inmutabilidad real hay que quitar permisos al rol de la aplicación. Está anotado en la migración.
- No hay CHECK de roles en la tabla `usuario`; los roles se validan solo en la API.
- Sin intentos de login fallidos en la bitácora (decisión: el login es público y la tabla no se puede borrar, así cualquiera podría llenarla).
- La contraseña de la base local apareció una vez en la salida de una herramienta durante una de las revisiones. No se escribió en ningún archivo del repositorio. Conviene rotarla si esa salida queda registrada.

## 9. Cambios en la base real («Cloude») al reiniciar el 8081

El servicio de 8081 se reinició al final (era el único permitido; 8000 no se tocó). El nuevo arranque aplicó las migraciones y se comprobó así:
- Existen `bitacora`, `notificacion` y `lectura_comprobante`, y las columnas `usuario.debe_cambiar_clave` y `gasto.fecha_cobro`.
- Había un único usuario real, `admin` (rol `admin`, activo). No había usuarios `operador`, así que la migración 0006 no cambió ninguno a `logistica`.
- Ese usuario `admin` quedó con `debe_cambiar_clave = true`: en su próximo inicio de sesión debe cambiar la contraseña.
- Hay 9 gastos reales y 0 notificaciones tras la primera pasada (ningún gasto cumple las condiciones de aviso en este momento). La bitácora está vacía: no se retroalimentó con historia anterior, y las pruebas manuales no escribieron en la base real.
- `GET /health` responde 200; `GET /gastos` sin token responde 401. Se creó `gastos-service/logs/errores.log` (ignorado por git).
- Todos los tokens emitidos antes del reinicio dejaron de valer, porque se estrenó una `JWT_SECRET` propia (agregada al final de `gastos-service/.env`, ignorado por git y no impreso).

## 10. Pendientes y límites (honestos)

- Docker no está instalado en esta máquina: `docker build` y `docker compose` no se probaron. Se verificó el flujo equivalente con un binario y PostgreSQL local. La demo en Docker ahora exige `JWT_SECRET` en el entorno al levantarla.
- Tesseract (opcional del Bloque 5) no se instaló; por eso 1 prueba de Go queda en SKIP y el OCR no lee imágenes escaneadas.
- Hay un esquema de pruebas huérfano en la base real, `prueba_1792710313578701000`, que dejó una ejecución anterior de las pruebas (no de esta etapa; las ejecuciones de hoy no dejaron otros). No se borró por no ser mío; se puede quitar con `DROP SCHEMA ... CASCADE`.
- El log de errores no rota; el README lo advierte. El respaldo (`scripts/respaldo.sh`) no tiene retención automática; se programa con cron según el README.
- El backend de Python (`backend/`) y `tools/contrato/capturar_contrato.py` no se tocaron: no conocen roles `logistica`, `debe_cambiar_clave`, usuarios, notificaciones ni `fecha_cobro`, y sus tokens no valen en Go. La prueba de contrato con Python se mantiene con 4 diferencias deliberadas documentadas en el propio test.
- Los documentos antiguos (`docs/superpowers/*`) todavía mencionan `X-User-Role`, el rol `operador` y la clave JWT por defecto; no se editaron.
- La lista de notificaciones tiene tope de 100 y no hay paginación; la bitácora global tampoco (el historial por gasto es corto).
- La hoja «Más» y la paleta ⌘K no tienen acceso a los avisos; la campana va solo en la barra superior. En pantallas muy angostas esa barra queda justa; se vio a 390 px y se ve bien.
- Un 403 inesperado puede mostrar dos avisos (el del interceptor y el propio del componente). Como las acciones ya están ocultas al rol que no puede usarlas, es raro.
- La pantalla `/app/checklist` es solo para logistica (la escritura es solo de ese rol); el administrador no puede consultarla. Si se quiere, hay que hacer un modo de solo lectura.
- Un administrador no puede cambiar su propio rol ni desactivarse desde la pantalla «Usuarios» (restricción de la interfaz; el servidor solo impide dejar al sistema sin administrador).
- Cuando se reinicie por primera vez en otra máquina, todos los usuarios existentes deberán cambiar su clave (0007).
