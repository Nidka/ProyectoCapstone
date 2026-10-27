-- Lectura automática (OCR) de comprobantes: el texto y los datos leídos de cada comprobante, para no
-- leerlo dos veces. Las sugerencias del checklist no se guardan: se recalculan contra el gasto en
-- cada consulta. Solo se guardan los estados leido y sin_texto. Idempotente.

CREATE TABLE IF NOT EXISTS lectura_comprobante (
    comprobante_id INTEGER PRIMARY KEY REFERENCES comprobante (id),
    estado         VARCHAR(20) NOT NULL CHECK (estado IN ('leido', 'sin_texto', 'no_disponible', 'error')),
    texto          TEXT NOT NULL DEFAULT '',
    datos          JSONB NOT NULL DEFAULT '{}',
    creado_en      TIMESTAMP NOT NULL DEFAULT now()
);
