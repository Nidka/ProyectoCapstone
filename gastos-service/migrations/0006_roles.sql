-- Roles del sistema: solo «admin» (Administrador del Sistema) y «logistica» (Encargado de
-- Logística). Los usuarios que tenían el rol «operador» pasan a «logistica». Idempotente: el
-- runner reaplica todas las migraciones en cada arranque y repetirla no cambia nada.

UPDATE usuario SET rol = 'logistica' WHERE rol = 'operador';

ALTER TABLE usuario ALTER COLUMN rol SET DEFAULT 'logistica';
