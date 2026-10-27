-- Comprobantes para los gastos de demostración, para que el flujo se vea completo en la presentación.
-- Se ejecuta DESPUÉS de reportes_demo.sql y se puede repetir (no duplica nada).
--
--   * Sustentado -> comprobante Aprobado, con los 5 criterios del checklist cumplidos.
--   * Pagado     -> comprobante Pendiente de aprobación.
--   * Registrado -> unos pocos con un comprobante Rechazado (con motivo); a otros se les pone
--                   fecha de cobro para que el generador de avisos tenga qué mostrar.
--
-- Los comprobantes reutilizan los archivos que ya están en uploads/ (no copia ni crea archivos).
-- reportes_demo_limpiar.sql también los borra.
--
--   psql "$DATABASE_URL" -f gastos-service/demo/reportes_demo_comprobantes.sql

\set ON_ERROR_STOP on
BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM gasto WHERE creado_por = 'DEMO') THEN
        RAISE EXCEPTION 'No hay gastos DEMO: ejecuta reportes_demo.sql primero';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM comprobante WHERE archivo_url LIKE '/uploads/%' AND gasto_id NOT IN (SELECT id FROM gasto WHERE creado_por = 'DEMO')) THEN
        RAISE EXCEPTION 'No hay comprobantes de muestra en uploads/ para reutilizar';
    END IF;
END $$;

-- Archivos de muestra: los de los comprobantes que no son de demostración
CREATE TEMP TABLE muestras ON COMMIT DROP AS
SELECT row_number() OVER (ORDER BY id) - 1 AS n, archivo_url, tipo_archivo
FROM comprobante
WHERE gasto_id NOT IN (SELECT id FROM gasto WHERE creado_por = 'DEMO')
  AND estado = 'Aprobado';

CREATE TEMP TABLE demo ON COMMIT DROP AS
SELECT g.id, g.estado, g.fecha, g.creado_en,
       row_number() OVER (ORDER BY g.fecha, g.id) AS orden
FROM gasto g
WHERE g.creado_por = 'DEMO'
  AND NOT EXISTS (SELECT 1 FROM comprobante c WHERE c.gasto_id = g.id);

-- Sustentado: aprobado uno a tres días después de registrarse
INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado, aprobado_por, aprobado_en, creado_en)
SELECT d.id, m.archivo_url, m.tipo_archivo, 'Aprobado', 'Administrador',
       d.creado_en + make_interval(days => 1 + d.id % 3, hours => 2 + d.id % 5),
       d.creado_en + make_interval(hours => 3 + d.id % 20)
FROM demo d
JOIN muestras m ON m.n = d.id % (SELECT count(*) FROM muestras)
WHERE d.estado = 'Sustentado';

-- Pagado: adjuntado y esperando revisión
INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado, creado_en)
SELECT d.id, m.archivo_url, m.tipo_archivo, 'Pendiente', d.creado_en + make_interval(hours => 3 + d.id % 20)
FROM demo d
JOIN muestras m ON m.n = d.id % (SELECT count(*) FROM muestras)
WHERE d.estado = 'Pagado';

-- Registrado: cada cuarto gasto tuvo un comprobante rechazado y vuelve a esperar uno nuevo
INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado, motivo_rechazo, aprobado_por, aprobado_en, creado_en)
SELECT d.id, m.archivo_url, m.tipo_archivo, 'Rechazado',
       (ARRAY[
           'El monto del comprobante no coincide con el gasto.',
           'La imagen está borrosa y no se lee el total.',
           'El comprobante no está emitido a nombre de LIZAR SRL.'
       ])[1 + d.id % 3],
       'Administrador',
       d.creado_en + make_interval(days => 1, hours => 1 + d.id % 6),
       d.creado_en + make_interval(hours => 3 + d.id % 20)
FROM demo d
JOIN muestras m ON m.n = d.id % (SELECT count(*) FROM muestras)
WHERE d.estado = 'Registrado' AND d.orden % 4 = 0;

-- Checklist de los aprobados: los cinco criterios estándar, cumplidos
INSERT INTO checklist_comprobante (comprobante_id, criterio, cumple, texto)
SELECT c.id, k.clave, TRUE, k.texto
FROM comprobante c
JOIN gasto g ON g.id = c.gasto_id AND g.creado_por = 'DEMO'
CROSS JOIN criterio_checklist k
WHERE c.estado = 'Aprobado' AND k.cliente_id IS NULL
ON CONFLICT (comprobante_id, criterio) DO NOTHING;

-- Checklist de los rechazados: falla el criterio que dice el motivo, el resto cumple
INSERT INTO checklist_comprobante (comprobante_id, criterio, cumple, texto)
SELECT c.id, k.clave,
       NOT (   (k.clave = 'monto_coincide'   AND c.motivo_rechazo LIKE 'El monto%')
            OR (k.clave = 'legible'          AND c.motivo_rechazo LIKE 'La imagen%')
            OR (k.clave = 'a_nombre_empresa' AND c.motivo_rechazo LIKE 'El comprobante no está%')),
       k.texto
FROM comprobante c
JOIN gasto g ON g.id = c.gasto_id AND g.creado_por = 'DEMO'
CROSS JOIN criterio_checklist k
WHERE c.estado = 'Rechazado' AND k.cliente_id IS NULL
ON CONFLICT (comprobante_id, criterio) DO NOTHING;

-- Fecha de cobro cerca (o vencida) para algunos Registrado/Pagado de los últimos días
UPDATE gasto g
SET fecha_cobro = GREATEST(g.fecha, CURRENT_DATE + (g.id % 5 - 2))
WHERE g.creado_por = 'DEMO'
  AND g.estado IN ('Registrado', 'Pagado')
  AND g.fecha >= CURRENT_DATE - 30
  AND g.id % 3 = 0
  AND g.fecha_cobro IS NULL;

COMMIT;
