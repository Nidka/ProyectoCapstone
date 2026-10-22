# Reportes estadísticos de gastos — diseño

**Fecha:** 2026-10-17 · **Entregable:** Cuarto Informe de Estado (módulo estadístico corporativo, dashboard gerencial, exportación PDF y botones de descarga).
**Estado:** diseño aprobado en conversación; esta especificación espera revisión.

## 1. Objetivo

Gerencia y Logística ven, para un rango de fechas, cuánto se gastó en gastos adicionales de viaje, cuánto está sustentado y dónde se concentra el gasto (cliente, unidad y conductor), con gráficos interactivos, y descargan esa misma información en dos PDF.

Éxito:
- El dashboard y los PDF muestran exactamente las mismas cifras para el mismo rango.
- Las cifras salen de consultas SQL de agregación ejecutadas en una transacción de solo lectura.
- Los botones descargan los PDF reales generados por gastos-service.

## 2. Alcance

**Entra:**
- Paquete Go `internal/reportes` en gastos-service: filtro de fechas, consultas de agregación y generación de los dos PDF.
- Tres endpoints: resumen JSON, PDF estadístico y PDF de detalle.
- Sección «Reportes» en el frontend con filtro, indicadores, gráficos Chart.js interactivos y dos botones de descarga.
- Scripts SQL de datos de demostración y de limpieza, que se ejecutan a mano.

**No entra:**
- Comparación con el presupuesto del viaje y calidad documental (checklist): no se eligieron.
- Exportar a Excel o CSV.
- Roles y permisos reales (RF-19 a RF-21). Como los demás GET del servicio, estos endpoints no exigen sesión.
- Reportes programados o enviados por correo.

**Versiones:** se usa lo que ya tiene el proyecto: Go 1.25 (el código no usa nada posterior a Go 1.22), PostgreSQL 17 y Angular 21.

## 3. Filtro de fechas

Parámetros `desde` y `hasta` en formato `AAAA-MM-DD`, sobre la fecha del gasto (`gasto.fecha`), ambos inclusive.

| Caso | Resultado |
|---|---|
| Sin `hasta` | hoy en America/Lima |
| Sin `desde` | primer día del mes, 11 meses antes del mes de `hasta` (12 meses calendario) |
| Fecha con formato inválido | 400 `fecha inválida: use AAAA-MM-DD` |
| `desde` posterior a `hasta` | 400 `la fecha inicial no puede ser posterior a la final` |
| Rango de más de 731 días | 400 `el rango no puede superar 24 meses` |

## 4. Estadísticas (SQL)

Todas las consultas de un reporte corren en **una sola transacción** `REPEATABLE READ, READ ONLY`, para que indicadores, series y rankings vengan de la misma foto de los datos. Cada consulta es una sola sentencia con CTE.

**Indicadores**
- `total`: suma de montos.
- `cantidad`: número de gastos.
- `promedio`: total entre cantidad, o 0 si no hay gastos.
- `sustentado`: suma de los gastos en estado Sustentado.
- `pendiente`: suma de los gastos en Registrado o Pagado.
- `porcentaje_sustentado`: sustentado entre total por 100, con 1 decimal, o 0 si no hay gastos.

**Por mes:** un elemento por cada mes desde el mes de `desde` hasta el de `hasta`, generado con `generate_series`, también para los meses sin gastos. Cada elemento tiene `mes` (`AAAA-MM`) y los montos `registrado`, `pagado`, `sustentado` y `total`, calculados con `SUM(monto) FILTER (WHERE estado = …)`.

**Por estado:** siempre tres filas, en el orden Registrado, Pagado, Sustentado. Cada una tiene `cantidad`, `monto` y `porcentaje`; el porcentaje sale de una función ventana `SUM() OVER ()`.

**Rankings:** por cliente, por unidad y por conductor.
- Hasta 10 filas cada uno, ordenadas por monto descendente y, si empatan, por etiqueta ascendente.
- Cada fila tiene `puesto` (`RANK()`), `etiqueta`, `cantidad`, `monto` y `porcentaje` sobre el total del período.
- **Cliente:** `gasto.cliente_id` → `cliente.razon_social`.
- **Unidad:** `gasto.viaje_id` → `programacionviaje.id_tracto` → `unidad.placa`.
- **Conductor:** `programacionviaje.id_conductor_unidad` → `conductorunidad.id_conductor` → `conductor`, con la etiqueta «nombre1 apellido1 apellido2», sin espacios sobrantes.
- Lo que no se pueda unir aparece como «Sin asignar». Todos los `JOIN` hacia viaje, unidad y conductor son `LEFT JOIN`.

**Detalle** (solo para el PDF de detalle): cada gasto del rango con `id`, `fecha`, `cliente`, `viaje_id`, `unidad`, `conductor`, `estado` y `monto`, ordenado por fecha y luego por id.

Los montos viajan en el JSON como números con 2 decimales y se suman en `NUMERIC` en la base, no en flotante.

## 5. API (gastos-service)

| Método y ruta | Respuesta |
|---|---|
| `GET /reportes/resumen?desde=&hasta=` | 200 JSON: `{ desde, hasta, indicadores, por_mes, por_estado, por_cliente, por_unidad, por_conductor }` |
| `GET /reportes/estadistico.pdf?desde=&hasta=` | 200 `application/pdf`, `Content-Disposition: attachment; filename="reporte-estadistico-AAAA-MM-DD_AAAA-MM-DD.pdf"` |
| `GET /reportes/detalle.pdf?desde=&hasta=` | 200 `application/pdf`, `filename="detalle-gastos-AAAA-MM-DD_AAAA-MM-DD.pdf"` |

- `desde` y `hasta` en el JSON y en los nombres de archivo son las fechas ya resueltas, después de aplicar los valores por defecto.
- Errores de filtro: 400 con los mensajes de la sección 3. Cualquier otro error: 500 `no se pudo generar el reporte`.
- CORS expone `Content-Disposition`, para que el navegador lea el nombre del archivo.

## 6. PDF (gastos-service, `signintech/gopdf`)

**Común a los dos:**
- Fuente Overpass Regular y Bold embebida con `go:embed`, con su licencia OFL junto a los archivos.
- Cabecera con banda roja `#E11D48`, «LIZAR SRL · Fleet Manager Pro», el título, «Período: 01/11/2025 – 31/10/2026» y «Emitido: 17/10/2026 10:32» (hora de Lima).
- Pie con «Página N de M».
- Montos con el mismo formato que el frontend (`formatMonto`).
- Los colores de estado son los de la app: Registrado `#8a9096`, Pagado `#2563eb`, Sustentado `#15803d`.

**Reporte estadístico (A4 vertical):**
1. Cuatro recuadros de indicadores: total, cantidad, promedio y % sustentado (con el monto pendiente debajo).
2. «Gasto por mes»: barras verticales apiladas por estado, dibujadas como vectores, con el eje de meses abreviados («nov 25») y una leyenda.
3. «Distribución por estado»: tabla de las tres filas, con una barra horizontal proporcional en cada una.
4. Tres tablas de ranking (cliente, unidad, conductor) con las columnas Puesto, Nombre, Gastos, Monto y %, y una barra proporcional en la columna %.
5. Sin gastos en el rango: los indicadores en 0 y el texto «No hay gastos registrados en este período.» en lugar de gráficos y tablas.

**Detalle de gastos (A4 horizontal):**
- Tabla con las columnas N°, Fecha, Cliente, Viaje, Unidad, Conductor, Estado y Monto.
- Encabezado de tabla repetido en cada página y filas alternadas.
- Al final, los totales por estado y el total general.
- Sin gastos: el mismo texto vacío.

## 7. Frontend (Angular 21)

**Navegación**
- Ruta `/app/reportes`, cargada de forma diferida, para que Chart.js solo se descargue al entrar.
- Entrada «Reportes» en `ENLACES_PRINCIPALES`, entre Gastos y Operación, con el icono nuevo `chart-bar`. En el celular aparece donde la barra inferior ubique los enlaces que no caben (menú «Más»).
- También queda disponible en la paleta de búsqueda.

**Filtro**
- Atajos «Últimos 12 meses» (por defecto), «Este año» y «Este mes».
- Dos campos de fecha, Desde y Hasta.
- El rango vive en la URL (`?desde=&hasta=`): se puede recargar y compartir.
- Un rango inválido se marca en los campos con el mensaje del servidor y no se consulta.

**Contenido**, de arriba abajo:
1. Cuatro tarjetas de indicadores, con las mismas cifras que el PDF.
2. «Gasto por mes», en barras apiladas por estado con los colores de estado:
   - el tooltip muestra el monto en S/ por estado y el total del mes;
   - la leyenda permite ocultar o mostrar cada estado;
   - un clic en un mes cambia el rango a ese mes completo y muestra el aviso «Viendo: octubre 2026 · Quitar»;
   - «Quitar» vuelve al rango anterior.
3. «Distribución por estado», en barras horizontales: monto y % por estado.
4. Tres rankings en barras horizontales (cliente, unidad, conductor), con tooltip de monto, cantidad y %.
5. Botones «Descargar reporte estadístico (PDF)» y «Descargar detalle de gastos (PDF)», junto al filtro:
   - usan el rango actual;
   - mientras se genera, el botón dice «Generando…» y queda deshabilitado;
   - el archivo se guarda con el nombre que manda el servidor;
   - si falla, aparece un toast con `mensajeError(err)`.

**Accesibilidad y estados**
- Cada gráfico tiene `role="img"` con un resumen en `aria-label`, y un desplegable «Ver datos» con la tabla equivalente.
- Mientras carga, los esqueletos usan las clases existentes. Si la consulta falla, el estado de error existente ofrece «Reintentar».
- Si no hay gastos en el rango, aparece un estado vacío «No hay gastos en este período» y los botones de descarga siguen activos (generan el PDF vacío).
- Estilos globales con los tokens existentes. Iconos nuevos en el registro: `chart-bar` y `download`.

**Código**
- `core/models/reporte.model.ts`: tipos de la respuesta.
- `core/services/reportes.service.ts`:
  - `resumen(filtro)`;
  - `descargar(tipo, filtro)`, que pide el PDF como blob con el `timeout` habitual y devuelve el blob con su nombre de archivo.
- `features/reportes/`:
  - lógica pura del filtro (atajos, rango de un mes, validación);
  - armado de datos para Chart.js, con funciones puras y probadas;
  - un componente de gráfico que envuelve Chart.js (crea, actualiza y destruye);
  - la página del dashboard.

## 8. Datos de demostración

- `gastos-service/demo/reportes_demo.sql` crea, con datos marcados DEMO:
  - 4 clientes, 5 unidades tracto (y sus carretas), 5 conductores con sus asignaciones y unos 20 viajes;
  - unos 150 gastos repartidos en los 12 meses anteriores a `CURRENT_DATE`;
  - los meses antiguos, sobre todo Sustentado; los recientes, con más Registrado y Pagado.
- Usa `setseed` para que los datos salgan siempre iguales.
- Si ya hay datos DEMO, primero los borra.
- `gastos-service/demo/reportes_demo_limpiar.sql` borra todo lo marcado DEMO, en orden seguro para las llaves foráneas.
- Ninguno está en `migrations/`: el servicio nunca los aplica solo.

## 9. Pruebas

**Go, unitarias:**
- filtro: valores por defecto, formatos, desde > hasta, rango máximo;
- formato de montos y de fechas;
- generación de ambos PDF con datos fijos: empiezan con `%PDF`, tienen el número de páginas esperado y no fallan con datos vacíos.

**Go, integración** (con `TEST_DATABASE_URL`, en un esquema temporal propio con las tablas mínimas de viaje, cliente, unidad, conductor y asignación, igual que en el checklist):
- indicadores con montos conocidos;
- meses sin gastos en 0;
- orden y empates de los rankings;
- «Sin asignar»;
- límites del rango (inclusive);
- endpoints con 200, cabeceras PDF y 400.

**Vitest:**
- atajos y validación del filtro;
- rango de un mes desde el clic;
- armado de los datos para Chart.js (colores por estado, etiquetas de mes, tooltips);
- página: indicadores, estado vacío, error, aviso «Viendo…»;
- descarga: estado «Generando…», nombre de archivo y error.
- Chart.js se simula en jsdom, que no tiene canvas.

**Navegador:** recorrido en escritorio y celular contra gastos-service real, sobre un esquema temporal con datos de demo:
- filtro y clic en un mes;
- tooltips;
- descarga real de ambos PDF, verificando que sean PDF válidos.

## 10. Riesgos y decisiones

- Endpoints sin autenticación, igual que el resto de los GET del servicio; depende de RF-19 a RF-21.
- `signintech/gopdf` es la única dependencia Go nueva, y `chart.js` la única de npm. Ambas se fijan en una versión publicada hace más de dos semanas.
- `go-pdf/fpdf` está archivado; por eso se descartó.
- Con los datos reales actuales (7 gastos) los gráficos se ven escasos. Para la demostración se usan los scripts de la sección 8.
