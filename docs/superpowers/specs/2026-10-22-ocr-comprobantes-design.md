# OCR de comprobantes — diseño

**Fecha:** 2026-10-22 · **Módulo:** apoyo al Checklist Documental (revisión del comprobante).
**Estado:** diseño aprobado en conversación; esta especificación espera revisión.

## 1. Objetivo

Al abrir la revisión de un comprobante, el sistema lo lee y **sugiere** qué criterios del checklist se cumplen, con el texto que lo respalda. Logística confirma o corrige. El OCR **nunca aprueba ni rechaza**: aprobar sigue exigiendo los 5 criterios marcados por una persona.

Éxito: con un comprobante legible, quien revisa marca el checklist con un clic en lugar de comparar monto y fecha a ojo; con uno ilegible, la revisión funciona exactamente como hoy.

## 2. Alcance

**Entra:**
- Lectura de comprobantes en gastos-service: PDF con texto con `pdftotext`; fotos y PDF escaneados con Tesseract (español).
- Extracción de monto, fecha, RUC del emisor y del receptor, y tipo de documento.
- Sugerencia de 4 de los 5 criterios, guardada para no leer dos veces.
- Endpoint `GET /comprobantes/{id}/lectura`.
- Sugerencias en la pantalla de revisión, con «Aplicar sugerencias».

**No entra:**
- Rellenar el gasto desde la foto, OCR en la nube, entrenar modelos.
- Sugerir `legible` (es un juicio humano).
- Leer comprobantes ya revisados: solo se lee un comprobante Pendiente.

## 3. Herramientas externas

| Herramienta | Uso | Si falta |
|---|---|---|
| `pdftotext` (poppler) | texto de PDF digitales | se pasa a Tesseract |
| `pdftoppm` (poppler) | PDF escaneado a imagen | el PDF queda «sin texto» |
| `tesseract` con idioma `spa` | fotos y PDF escaneados | estado `no_disponible` |

Se buscan en el `PATH` al iniciar. El servicio arranca igual sin ellas. Cada ejecución tiene tiempo límite de 20 s y la lectura completa de un comprobante, 30 s. Los nombres de archivo nunca pasan por una shell: se invoca con `exec.Command` y argumentos separados.

## 4. Lectura

Entrada: el archivo del comprobante (`archivo_url` resuelto dentro de `UPLOADS_DIR`; una ruta fuera de esa carpeta se rechaza).

1. PDF: `pdftotext -layout`. Si el texto tiene menos de 20 caracteres alfanuméricos, se trata como escaneado: `pdftoppm -r 200 -png -f 1 -l 2` y Tesseract sobre cada imagen.
2. Imagen: Tesseract `-l spa --psm 6`.
3. El texto se normaliza (espacios, saltos, «O» y «l» confundidos con dígitos solo dentro de números).

## 5. Extracción (funciones puras)

| Dato | Regla |
|---|---|
| Monto | Número con 2 decimales junto a «TOTAL», «IMPORTE TOTAL» o «TOTAL A PAGAR»; admite `S/`, coma de miles y coma decimal. Si hay varios candidatos, se toma el mayor junto a esas etiquetas. |
| Fecha | `dd/mm/aaaa`, `dd-mm-aaaa`, `aaaa-mm-dd` y «dd de mes de aaaa»; se descartan fechas imposibles. Prefiere la que sigue a «FECHA». |
| RUC | 11 dígitos que empiezan con 10, 15, 17 o 20. Se guardan todos los encontrados, en orden. |
| Tipo | `factura` si aparece «FACTURA», `boleta` si aparece «BOLETA», si no `otro`. Una «GUÍA» o «NOTA» no cuenta como válida. |

Cada dato lleva `valor`, `texto` (la línea donde se leyó) y `confianza` (`alta` si lo encontró junto a su etiqueta, `media` si lo infirió, `ninguna` si no lo encontró).

## 6. Sugerencia del checklist

| Criterio | Sugiere ✓ cuando | Si no |
|---|---|---|
| `monto_coincide` | monto leído = monto del gasto (diferencia menor a S/ 0.005) | sin sugerencia; si se leyó otro monto, se muestra «Leído S/ X» |
| `fecha_coincide` | fecha leída = fecha del gasto | igual, con la fecha leída |
| `a_nombre_empresa` | aparece el RUC 20481608750 o «LIZAR» | sin sugerencia |
| `tipo_valido` | tipo `factura` o `boleta` | sin sugerencia |
| `legible` | nunca | — |

El sistema **solo sugiere ✓**. Nunca sugiere ✗: no encontrar un dato no prueba que el comprobante esté mal.

## 7. Datos

Migración `0004_lectura_comprobante.sql` (idempotente):

```sql
CREATE TABLE IF NOT EXISTS lectura_comprobante (
    comprobante_id INTEGER PRIMARY KEY REFERENCES comprobante (id),
    estado         VARCHAR(20) NOT NULL CHECK (estado IN ('leido', 'sin_texto', 'no_disponible', 'error')),
    texto          TEXT NOT NULL DEFAULT '',
    datos          JSONB NOT NULL DEFAULT '{}',
    creado_en      TIMESTAMP NOT NULL DEFAULT now()
);
```

Se guarda la lectura del comprobante, no las sugerencias: estas se recalculan contra el gasto en cada consulta (el monto o la fecha del gasto pueden cambiar). Un estado `no_disponible` o `error` no se guarda, para reintentar cuando se instale la herramienta.

## 8. API

`GET /comprobantes/{id}/lectura` → 200:

```json
{
  "estado": "leido",
  "datos": { "monto": {"valor": 85.5, "texto": "TOTAL S/ 85.50", "confianza": "alta"}, "fecha": {…}, "rucs": ["20481608750"], "tipo": {…} },
  "sugerencias": { "monto_coincide": {"sugerido": true, "detalle": "Leído S/ 85.50"}, "fecha_coincide": {…}, "a_nombre_empresa": {…}, "tipo_valido": {…} }
}
```

| Caso | Respuesta |
|---|---|
| id inválido | 400 |
| comprobante inexistente | 404 |
| comprobante ya revisado | 409 `el comprobante ya fue revisado` |
| herramienta ausente | 200 con `estado: "no_disponible"` y sin sugerencias |
| lectura fallida o vencida | 200 con `estado: "error"` |

No exige rol de logística (es de lectura, igual que los demás GET).

## 9. Frontend

- `ReportesService`-style: `LecturaService.lectura(id)` con el `timeout` de 30 s.
- `RevisionComprobanteComponent`: al mostrarse, pide la lectura. Mientras carga: «Leyendo comprobante…» junto al título del checklist. Junto a cada casilla con sugerencia: «Sugerido ✓ · Leído S/ 85.50».
- Botón «Aplicar sugerencias» (solo si hay al menos una): marca esas casillas, sin tocar las demás. El conteo y el motivo propuesto del rechazo se actualizan como con un clic manual.
- Estados `no_disponible`, `error` y `sin_texto`: un aviso discreto («No se pudo leer el comprobante automáticamente») y todo lo demás igual.
- Los errores de red usan `mensajeError`; `error-messages.ts` no se modifica.
- La lectura no bloquea nada: se puede marcar el checklist y aprobar sin esperarla.

## 10. Pruebas

- **Go, unitarias:** extracción con textos de boletas y facturas de ejemplo (formatos de monto y fecha, varios RUC, tipo, textos sin datos); sugerencias (coincide, difiere, ausente); rutas fuera de `UPLOADS_DIR`.
- **Go, integración** (esquema temporal): endpoint con un lector falso (estados, 404, 409, guardado y reuso de la lectura, que `no_disponible` no se guarda).
- **Go, con herramientas reales** (se omite si no están instaladas): `pdftotext` sobre un PDF de `uploads/` de muestra y Tesseract sobre una imagen generada con texto conocido.
- **Vitest:** sugerencias visibles, «Aplicar sugerencias», estados de falla y que la revisión funciona sin lectura.
- **Navegador:** un comprobante real de muestra, aplicar sugerencias y aprobar.

## 11. Riesgos y decisiones

- La precisión de Tesseract baja con fotos torcidas u oscuras. Por eso solo sugiere ✓ y siempre decide una persona.
- Instalar `tesseract` y `tesseract-lang` (y poppler) es requisito del servidor de producción; sin ellos la función se apaga sola.
- Las muestras actuales son facturas digitales de LIZAR: las fotos de celular reales deberán probarse antes de confiar en la precisión.
- No se guardan datos personales fuera de la base ya existente; el texto leído queda en `lectura_comprobante`.
