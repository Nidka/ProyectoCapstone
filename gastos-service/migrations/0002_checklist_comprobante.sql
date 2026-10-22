-- Módulo de checklist documental: lo que Logística verificó en cada comprobante.
-- Quién y cuándo revisó ya viven en comprobante.aprobado_por y comprobante.aprobado_en.

CREATE TABLE IF NOT EXISTS checklist_comprobante (
    id             SERIAL PRIMARY KEY,
    comprobante_id INTEGER NOT NULL REFERENCES comprobante (id),
    criterio       VARCHAR(30) NOT NULL
                   CHECK (criterio IN ('legible', 'tipo_valido', 'a_nombre_empresa', 'monto_coincide', 'fecha_coincide')),
    cumple         BOOLEAN NOT NULL,
    CONSTRAINT uq_checklist_criterio UNIQUE (comprobante_id, criterio)
);

CREATE INDEX IF NOT EXISTS idx_checklist_comprobante_id ON checklist_comprobante (comprobante_id);
