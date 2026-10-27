-- Más clientes y viajes de demostración para tener qué elegir al registrar un gasto.
--
-- Agrega 8 clientes, 6 direcciones, 3 tractos con su carreta, 3 conductores con su asignación y
-- 24 viajes (de 90 días atrás a unas 3 semanas adelante). Los viajes no tienen gastos: están
-- libres para registrarlos a mano. Todo va marcado DEMO con las mismas marcas que reportes_demo.sql
-- (RUC 990000000xx, placas DEM-, DNI 990000xx, direcciones con referencia DEMO), así que
-- reportes_demo_limpiar.sql también lo borra.
--
-- Se ejecuta DESPUÉS de reportes_demo.sql (necesita sus direcciones, tractos y conductores).
-- Si vuelves a ejecutar reportes_demo.sql, que borra todo lo DEMO, vuelve a ejecutar este.
--
--   psql "$DATABASE_URL" -f gastos-service/demo/seleccion_demo.sql

\set ON_ERROR_STOP on
BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM unidad WHERE placa = 'DEM-101') THEN
        RAISE EXCEPTION 'Faltan los datos de reportes_demo.sql: ejecútalo antes';
    END IF;
    IF EXISTS (SELECT 1 FROM cliente WHERE ruc = '99000000005') THEN
        RAISE EXCEPTION 'Estos datos ya están cargados: ejecuta reportes_demo_limpiar.sql, reportes_demo.sql y este script otra vez';
    END IF;
END $$;

INSERT INTO direccion (codigo_ubigeo, direccion_detallada, referencia, descripcion, estado_direccion)
SELECT (SELECT min(codigo_ubigeo) FROM ubigeo), d, 'DEMO', 'DEMO', TRUE
FROM unnest(ARRAY[
    'Fundo Santa Rosa, Virú km 8',
    'Obra Huaca Prieta, Chicama',
    'Mina Cerro Blanco, Cajamarca',
    'Puerto de Salaverry, muelle principal',
    'Parque Industrial Trujillo, Mz B lote 4',
    'Terminal Pacasmayo, Av. Industrial 300'
]) AS d;

INSERT INTO cliente (razon_social, ruc, contacto, id_rubro, estado_cliente)
SELECT c.nombre, c.ruc, 'DEMO',
       (SELECT id_rubro FROM rubro ORDER BY id_rubro OFFSET c.n % (SELECT count(*) FROM rubro) LIMIT 1), TRUE
FROM (VALUES
    (0, 'Agrícola Santa Rosa SAC', '99000000005'),
    (1, 'Constructora Huaca Prieta SAC', '99000000006'),
    (2, 'Minera Cerro Blanco SA', '99000000007'),
    (3, 'Frutas del Valle EIRL', '99000000008'),
    (4, 'Industrias Chicama SAC', '99000000009'),
    (5, 'Logística Pacasmayo SRL', '99000000010'),
    (6, 'Cementos del Norte SA', '99000000011'),
    (7, 'Textil Moche SAC', '99000000012')
) AS c (n, nombre, ruc);

-- Tractos DEM-106..108 y carretas DEM-206..208
INSERT INTO unidad (placa, id_tipo_unidad, id_subtipo_unidad, id_marca, modelo, anio, estado_unidad)
SELECT 'DEM-' || (100 * t.grupo + n), t.id_tipo,
       (SELECT min(id_subtipo_unidad) FROM subtipounidad WHERE id_tipo_unidad = t.id_tipo),
       (SELECT min(id_marca) FROM marca), 'DEMO', 2020 + n % 4, TRUE
FROM generate_series(6, 8) AS n
CROSS JOIN (
    SELECT 1 AS grupo, COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'tracto%'), (SELECT min(id_tipo_unidad) FROM tipounidad)) AS id_tipo
    UNION ALL
    SELECT 2, COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'carreta%'), (SELECT max(id_tipo_unidad) FROM tipounidad))
) AS t;

INSERT INTO conductor (dni, nombre1, nombre2, apellido1, apellido2, licencia_conducir, estado_conductor)
SELECT '9900000' || n, nombre, 'DEMO', ap1, ap2, 'DEMO-000' || n, TRUE
FROM (VALUES
    (6, 'Pedro', 'Linares', 'Salazar'),
    (7, 'Ana', 'Gutiérrez', 'Neyra'),
    (8, 'Miguel', 'Cabanillas', 'Ruiz')
) AS c (n, nombre, ap1, ap2);

-- Cada conductor nuevo maneja el tracto de su mismo número
INSERT INTO conductorunidad (id_conductor, id_unidad, fecha_asignacion, estado_asignacion)
SELECT d.id_conductor, u.id_unidad, CURRENT_DATE - interval '6 months', TRUE
FROM conductor d
JOIN unidad u ON u.placa = 'DEM-10' || right(d.dni, 1)
WHERE d.dni IN ('99000006', '99000007', '99000008');

-- 24 viajes, uno cada 5 días desde hace 90 días, repartidos entre todos los clientes, tractos,
-- conductores y direcciones de demostración
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
SELECT ids.tractos[n % cardinality(ids.tractos) + 1],
       ids.carretas[n % cardinality(ids.carretas) + 1],
       ids.asignaciones[n % cardinality(ids.asignaciones) + 1],
       ids.direcciones[n % cardinality(ids.direcciones) + 1],
       ids.direcciones[(n + 4) % cardinality(ids.direcciones) + 1],
       CURRENT_DATE - 90 + n * 5 + time '06:00',
       ids.clientes[(n * 5 + 4) % cardinality(ids.clientes) + 1],
       TRUE
FROM ids, generate_series(0, 23) AS n;

COMMIT;
