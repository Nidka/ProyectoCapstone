-- Borra los datos de demostración de los reportes: todo lo marcado DEMO
-- (RUC 990000000xx, placas DEM-, DNI 990000xx, direcciones con referencia DEMO
-- y gastos creados por DEMO), más lo que se haya registrado sobre ellos.
--
--   psql "$DATABASE_URL" -f gastos-service/demo/reportes_demo_limpiar.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE TEMP TABLE demo_viajes ON COMMIT DROP AS
SELECT id_viaje
FROM programacionviaje
WHERE id_cliente IN (SELECT id_cliente FROM cliente WHERE ruc LIKE '990000000%')
   OR id_tracto IN (SELECT id_unidad FROM unidad WHERE placa LIKE 'DEM-%')
   OR id_carreta IN (SELECT id_unidad FROM unidad WHERE placa LIKE 'DEM-%')
   OR id_conductor_unidad IN (
        SELECT cu.id_conductor_unidad
        FROM conductorunidad cu
        JOIN conductor d ON d.id_conductor = cu.id_conductor
        WHERE d.dni LIKE '990000%')
   OR id_direccion_origen IN (SELECT id_direccion FROM direccion WHERE referencia = 'DEMO')
   OR id_direccion_destino IN (SELECT id_direccion FROM direccion WHERE referencia = 'DEMO');

CREATE TEMP TABLE demo_gastos ON COMMIT DROP AS
SELECT id
FROM gasto
WHERE creado_por = 'DEMO'
   OR viaje_id IN (SELECT id_viaje FROM demo_viajes)
   OR cliente_id IN (SELECT id_cliente FROM cliente WHERE ruc LIKE '990000000%');

DELETE FROM checklist_comprobante
WHERE comprobante_id IN (SELECT id FROM comprobante WHERE gasto_id IN (SELECT id FROM demo_gastos));
DELETE FROM comprobante WHERE gasto_id IN (SELECT id FROM demo_gastos);
-- Los avisos del generador (0009) apuntan al gasto y no se borran solos.
DELETE FROM notificacion WHERE id_gasto IN (SELECT id FROM demo_gastos);
DELETE FROM gasto WHERE id IN (SELECT id FROM demo_gastos);

DELETE FROM detallepresupuesto
WHERE id_presupuesto IN (SELECT id_presupuesto FROM presupuestoviaje WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM presupuestoviaje WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes);
DELETE FROM factura
WHERE id_viaje_realizado IN (
    SELECT vr.id_viaje_realizado
    FROM viajerealizado vr
    JOIN guiatransportista g ON g.id_guia_transportista = vr.id_guia_transportista
    WHERE g.id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM viajerealizado
WHERE id_guia_transportista IN (SELECT id_guia_transportista FROM guiatransportista WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM productotransportado
WHERE id_guia_transportista IN (SELECT id_guia_transportista FROM guiatransportista WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes));
DELETE FROM guiatransportista WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes);
DELETE FROM programacionviaje WHERE id_viaje IN (SELECT id_viaje FROM demo_viajes);

DELETE FROM conductorunidad
WHERE id_conductor IN (SELECT id_conductor FROM conductor WHERE dni LIKE '990000%')
   OR id_unidad IN (SELECT id_unidad FROM unidad WHERE placa LIKE 'DEM-%');
DELETE FROM conductor WHERE dni LIKE '990000%';
DELETE FROM unidad WHERE placa LIKE 'DEM-%';
DELETE FROM cliente WHERE ruc LIKE '990000000%';
DELETE FROM direccion WHERE referencia = 'DEMO';

COMMIT;
