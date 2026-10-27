-- Bitácora de auditoría (REQ-33 a REQ-36, RNF-9 a RNF-12): quién hizo qué y cuándo, con los
-- valores de antes y de después en detalle. Idempotente: el runner reaplica todas las migraciones
-- en cada arranque.
--
-- Sin clave foránea a usuario a propósito: la bitácora debe poder registrar a cualquier usuario
-- y no puede bloquear, ni quedar atada a, cambios en la tabla usuario; por eso guarda también
-- una copia del nombre. fecha usa clock_timestamp() (el instante real del INSERT, no el inicio de
-- la transacción) para que el historial salga en el orden en que ocurrieron las cosas.

CREATE TABLE IF NOT EXISTS bitacora (
    id         BIGSERIAL PRIMARY KEY,
    fecha      TIMESTAMPTZ  NOT NULL DEFAULT clock_timestamp(),
    id_usuario INTEGER,
    usuario    VARCHAR(100) NOT NULL,
    accion     VARCHAR(50)  NOT NULL,
    entidad    VARCHAR(50)  NOT NULL,
    id_entidad BIGINT,
    detalle    JSONB        NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_bitacora_entidad ON bitacora (entidad, id_entidad);
CREATE INDEX IF NOT EXISTS idx_bitacora_fecha ON bitacora (fecha);

-- RNF-9 y RNF-10: la bitácora solo se escribe y se lee. Un trigger rechaza UPDATE y DELETE por
-- fila, y TRUNCATE por sentencia, para todos los usuarios de la base, dueño incluido. (Un
-- superusuario aún podría desactivar el trigger a propósito; eso queda fuera del alcance de la
-- aplicación.)
CREATE OR REPLACE FUNCTION bitacora_inmutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'La bitácora no admite modificar ni borrar registros (%)', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_bitacora_inmutable ON bitacora;
CREATE TRIGGER trg_bitacora_inmutable
    BEFORE UPDATE OR DELETE ON bitacora
    FOR EACH ROW EXECUTE FUNCTION bitacora_inmutable();

DROP TRIGGER IF EXISTS trg_bitacora_sin_truncate ON bitacora;
CREATE TRIGGER trg_bitacora_sin_truncate
    BEFORE TRUNCATE ON bitacora
    FOR EACH STATEMENT EXECUTE FUNCTION bitacora_inmutable();
