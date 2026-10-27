-- Notificaciones (REQ-20, REQ-21, RNF-8). Idempotente: el runner reaplica todas las migraciones
-- en cada arranque.
--
-- fecha_cobro es la fecha en que se espera cobrar el gasto al cliente; es opcional y sin ella el
-- gasto no genera avisos de cobro. No puede ser anterior a la fecha del gasto.

ALTER TABLE gasto ADD COLUMN IF NOT EXISTS fecha_cobro DATE;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'gasto_fecha_cobro_valida' AND conrelid = 'gasto'::regclass
    ) THEN
        ALTER TABLE gasto ADD CONSTRAINT gasto_fecha_cobro_valida
            CHECK (fecha_cobro IS NULL OR fecha_cobro >= fecha);
    END IF;
END $$;

-- Un aviso por tipo y gasto: UNIQUE (tipo, id_gasto) evita duplicados aunque el generador corra
-- cada hora, o dos veces a la vez. Si la condición deja de cumplirse el aviso no se borra: queda
-- como constancia hasta que se lea.
CREATE TABLE IF NOT EXISTS notificacion (
    id        SERIAL PRIMARY KEY,
    tipo      VARCHAR(30)  NOT NULL CHECK (tipo IN ('sin_comprobante', 'cobro_proximo')),
    id_gasto  INTEGER      NOT NULL REFERENCES gasto (id),
    mensaje   TEXT         NOT NULL,
    creada_en TIMESTAMPTZ  NOT NULL DEFAULT now(),
    leida     BOOLEAN      NOT NULL DEFAULT FALSE,
    leida_en  TIMESTAMPTZ,
    CONSTRAINT notificacion_tipo_gasto_unico UNIQUE (tipo, id_gasto)
);

CREATE INDEX IF NOT EXISTS idx_notificacion_leida ON notificacion (leida, creada_en DESC);
