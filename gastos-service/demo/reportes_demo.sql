-- Datos de demostración para los reportes estadísticos (Cuarto Informe de Estado).
--
-- Crea 4 clientes, 5 tractos con sus carretas, 5 conductores con sus asignaciones,
-- 24 viajes y unos 150 gastos repartidos en los 12 meses anteriores a hoy, todos
-- marcados como DEMO. Los gastos no tienen comprobantes adjuntos: sirven para los
-- reportes. Primero borra los datos DEMO que ya existan, así que se puede repetir.
-- Necesita los catálogos de db.sql (rubro, tipounidad, marca, ubigeo).
--
--   psql "$DATABASE_URL" -f gastos-service/demo/reportes_demo.sql
--
-- Para quitarlos: reportes_demo_limpiar.sql. No es una migración: el servicio nunca lo aplica.

\set ON_ERROR_STOP on
\ir reportes_demo_limpiar.sql

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM rubro) OR NOT EXISTS (SELECT 1 FROM tipounidad)
       OR NOT EXISTS (SELECT 1 FROM marca) OR NOT EXISTS (SELECT 1 FROM ubigeo) THEN
        RAISE EXCEPTION 'Faltan catálogos (rubro, tipounidad, marca, ubigeo): carga db.sql antes de los datos de demostración';
    END IF;
END $$;

-- Mismos datos en cada ejecución
SELECT setseed(0.17);

INSERT INTO direccion (codigo_ubigeo, direccion_detallada, referencia, descripcion, estado_direccion)
SELECT (SELECT min(codigo_ubigeo) FROM ubigeo), d, 'DEMO', 'DEMO', TRUE
FROM unnest(ARRAY[
    'Planta Chao, Panamericana Norte km 512',
    'Puerto de Paita, muelle 2',
    'Almacén Callao, Av. Néstor Gambetta 1200'
]) AS d;

INSERT INTO cliente (razon_social, ruc, contacto, id_rubro, estado_cliente)
SELECT nombre, ruc, 'DEMO', (SELECT min(id_rubro) FROM rubro), TRUE
FROM (VALUES
    ('Agroexportadora Norte SAC', '99000000001'),
    ('Minera Andina del Sur SA', '99000000002'),
    ('Distribuidora Costa Verde SAC', '99000000003'),
    ('Pesquera Bahía Azul SAC', '99000000004')
) AS c (nombre, ruc);

-- Tractos DEM-101..105 y carretas DEM-201..205
INSERT INTO unidad (placa, id_tipo_unidad, id_subtipo_unidad, id_marca, modelo, anio, estado_unidad)
SELECT 'DEM-' || (100 * t.grupo + n), t.id_tipo,
       (SELECT min(id_subtipo_unidad) FROM subtipounidad WHERE id_tipo_unidad = t.id_tipo),
       (SELECT min(id_marca) FROM marca), 'DEMO', 2020 + n % 4, TRUE
FROM generate_series(1, 5) AS n
CROSS JOIN (
    SELECT 1 AS grupo, COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'tracto%'), (SELECT min(id_tipo_unidad) FROM tipounidad)) AS id_tipo
    UNION ALL
    SELECT 2, COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'carreta%'), (SELECT max(id_tipo_unidad) FROM tipounidad))
) AS t;

INSERT INTO conductor (dni, nombre1, nombre2, apellido1, apellido2, licencia_conducir, estado_conductor)
SELECT '9900000' || n, nombre, 'DEMO', ap1, ap2, 'DEMO-000' || n, TRUE
FROM (VALUES
    (1, 'Carlos', 'Quispe', 'Rojas'),
    (2, 'María', 'Huamán', 'Torres'),
    (3, 'José', 'Ñique', 'Paredes'),
    (4, 'Luis', 'Castillo', 'Vega'),
    (5, 'Rosa', 'Mendoza', 'Alva')
) AS c (n, nombre, ap1, ap2);

-- Cada conductor maneja el tracto de su mismo número
INSERT INTO conductorunidad (id_conductor, id_unidad, fecha_asignacion, estado_asignacion)
SELECT d.id_conductor, u.id_unidad, CURRENT_DATE - interval '1 year', TRUE
FROM conductor d
JOIN unidad u ON u.placa = 'DEM-10' || right(d.dni, 1)
WHERE d.dni LIKE '990000%';

-- 24 viajes: dos por mes (días 3 y 17) en los últimos 12 meses
WITH ids AS (
    SELECT
        (SELECT array_agg(id_cliente ORDER BY ruc) FROM cliente WHERE ruc LIKE '990000000%') AS clientes,
        (SELECT array_agg(id_unidad ORDER BY placa) FROM unidad WHERE placa LIKE 'DEM-1%') AS tractos,
        (SELECT array_agg(id_unidad ORDER BY placa) FROM unidad WHERE placa LIKE 'DEM-2%') AS carretas,
        (SELECT array_agg(cu.id_conductor_unidad ORDER BY d.dni)
           FROM conductorunidad cu JOIN conductor d ON d.id_conductor = cu.id_conductor
          WHERE d.dni LIKE '990000%') AS asignaciones,
        (SELECT array_agg(id_direccion ORDER BY id_direccion) FROM direccion WHERE referencia = 'DEMO') AS direcciones
)
INSERT INTO programacionviaje (id_tracto, id_carreta, id_conductor_unidad, id_direccion_origen, id_direccion_destino,
                               fecha_programado, id_cliente, estado_viaje)
SELECT ids.tractos[n % 5 + 1], ids.carretas[n % 5 + 1], ids.asignaciones[n % 5 + 1],
       ids.direcciones[n % 3 + 1], ids.direcciones[(n + 1) % 3 + 1],
       date_trunc('month', CURRENT_DATE) - make_interval(months => n / 2) + make_interval(days => 2 + (n % 2) * 14),
       ids.clientes[n % 4 + 1], TRUE
FROM ids, generate_series(0, 23) AS n;

-- 150 gastos: los meses antiguos casi todos Sustentado; los recientes, con más Registrado y Pagado
WITH viajes AS (
    SELECT v.id_viaje, v.id_cliente, v.fecha_programado::date AS dia,
           row_number() OVER (ORDER BY v.fecha_programado, v.id_viaje) - 1 AS orden
    FROM programacionviaje v
    JOIN cliente c ON c.id_cliente = v.id_cliente
    WHERE c.ruc LIKE '990000000%'
),
sorteo AS (
    SELECT n, floor(random() * 24)::int AS orden, random() AS r_dia, random() AS r_estado, random() AS r_monto
    FROM generate_series(1, 150) AS n
),
gastos AS (
    SELECT v.id_viaje, v.id_cliente, LEAST(v.dia + floor(s.r_dia * 12)::int, CURRENT_DATE) AS fecha, s.r_estado, s.r_monto
    FROM sorteo s
    JOIN viajes v ON v.orden = s.orden
)
INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en)
SELECT id_viaje, id_cliente,
       round((25 + r_monto * r_monto * 875)::numeric, 2),
       fecha,
       CASE
           WHEN fecha < date_trunc('month', CURRENT_DATE) - interval '2 months' THEN
               CASE WHEN r_estado < 0.85 THEN 'Sustentado' WHEN r_estado < 0.95 THEN 'Pagado' ELSE 'Registrado' END
           WHEN fecha < date_trunc('month', CURRENT_DATE) THEN
               CASE WHEN r_estado < 0.5 THEN 'Sustentado' WHEN r_estado < 0.8 THEN 'Pagado' ELSE 'Registrado' END
           ELSE
               CASE WHEN r_estado < 0.2 THEN 'Sustentado' WHEN r_estado < 0.55 THEN 'Pagado' ELSE 'Registrado' END
       END,
       'DEMO', fecha + time '09:00'
FROM gastos;

COMMIT;
