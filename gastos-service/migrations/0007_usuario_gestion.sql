-- Gestión de usuarios (REQ-28 a REQ-32): marca que obliga a cambiar la contraseña antes de usar el
-- sistema. Al agregar la columna, los usuarios que ya existen (incluido el admin sembrado con una
-- clave conocida) quedan obligados a cambiarla una vez. Idempotente: el runner reaplica todas las
-- migraciones en cada arranque y, con la columna ya creada, repetirla no toca ningún valor.

ALTER TABLE usuario ADD COLUMN IF NOT EXISTS debe_cambiar_clave BOOLEAN NOT NULL DEFAULT TRUE;
