-- Checklist documental configurable por cliente (REQ-15, REQ-16).
-- Los criterios estándar (cliente_id NULL) son la lista con que parte todo cliente; un cliente con
-- criterios propios deja de usar el estándar. Idempotente.

CREATE TABLE IF NOT EXISTS criterio_checklist (
    id         SERIAL PRIMARY KEY,
    cliente_id INTEGER REFERENCES cliente (id_cliente) ON DELETE CASCADE,
    clave      VARCHAR(40)  NOT NULL,
    texto      VARCHAR(120) NOT NULL,
    orden      INTEGER      NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_criterio_estandar ON criterio_checklist (clave) WHERE cliente_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_criterio_cliente ON criterio_checklist (cliente_id, clave) WHERE cliente_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_criterio_cliente ON criterio_checklist (cliente_id);

INSERT INTO criterio_checklist (cliente_id, clave, texto, orden) VALUES
    (NULL, 'legible',          'El comprobante es legible y está completo.', 1),
    (NULL, 'tipo_valido',      'Es una boleta o factura válida.',            2),
    (NULL, 'a_nombre_empresa', 'Está emitido a nombre de LIZAR SRL.',        3),
    (NULL, 'monto_coincide',   'El monto coincide con el gasto.',            4),
    (NULL, 'fecha_coincide',   'La fecha corresponde al gasto.',             5)
ON CONFLICT DO NOTHING;

-- Lo ya revisado conserva el texto del criterio tal como estaba, y la clave ya no se limita a 5 valores.
ALTER TABLE checklist_comprobante DROP CONSTRAINT IF EXISTS checklist_comprobante_criterio_check;
ALTER TABLE checklist_comprobante ALTER COLUMN criterio TYPE VARCHAR(40);
ALTER TABLE checklist_comprobante ADD COLUMN IF NOT EXISTS texto VARCHAR(120);

UPDATE checklist_comprobante
SET texto = c.texto
FROM criterio_checklist c
WHERE c.cliente_id IS NULL AND c.clave = checklist_comprobante.criterio AND checklist_comprobante.texto IS NULL;

UPDATE checklist_comprobante SET texto = criterio WHERE texto IS NULL;
ALTER TABLE checklist_comprobante ALTER COLUMN texto SET NOT NULL;
