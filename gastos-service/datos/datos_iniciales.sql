-- Datos iniciales para presentar el sistema con operación realista (sin marcas DEMO).
--
-- Agrega 8 clientes, 13 direcciones (con 5 destinos fuera de Trujillo), 6 tractos y 6 carretas,
-- 8 conductores con su asignación, unos 48 viajes en los últimos 12 meses (más 4 programados) y
-- sus gastos en los tres estados, con comprobantes, checklist y bitácora coherentes con el flujo:
--   Registrado: sin comprobante o con el último rechazado
--   Pagado:     comprobante pendiente de revisión
--   Sustentado: comprobante aprobado
-- Los gastos se reparten entre los usuarios de logística que ya existen; no crea usuarios ni toca
-- lo que ya está cargado. Los avisos (REQ-20, REQ-21) los genera el servicio en su pasada horaria.
-- Las empresas, personas y placas son ficticias. Los comprobantes apuntan a archivos que no se
-- incluyen: el registro existe, pero el archivo no se puede abrir.
--
--   psql "$DATABASE_URL" -f gastos-service/datos/datos_iniciales.sql
--
-- No es una migración: el servicio nunca lo aplica. Se niega a correr dos veces.

\set ON_ERROR_STOP on
BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM cliente WHERE ruc = '20601847354') THEN
        RAISE EXCEPTION 'Los datos iniciales ya están cargados';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM usuario WHERE rol = 'logistica' AND estado) THEN
        RAISE EXCEPTION 'Falta al menos un usuario de logística activo: los gastos se registran a su nombre';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM criterio_checklist WHERE cliente_id IS NULL) THEN
        RAISE EXCEPTION 'Falta el checklist general (migración 0003): arranca el servicio una vez antes';
    END IF;
END $$;

-- Mismos datos en cada base donde se cargue
SELECT setseed(0.42);

-- Destinos fuera de Trujillo (códigos INEI de las capitales)
INSERT INTO ubigeo (codigo_ubigeo, departamento, provincia, distrito, estado_ubigeo)
SELECT * FROM (VALUES
    (150101, 'Lima', 'Lima', 'Lima', TRUE),
    (70101, 'Callao', 'Callao', 'Callao', TRUE),
    (140101, 'Lambayeque', 'Chiclayo', 'Chiclayo', TRUE),
    (200101, 'Piura', 'Piura', 'Piura', TRUE),
    (60101, 'Cajamarca', 'Cajamarca', 'Cajamarca', TRUE)
) AS u (codigo_ubigeo, departamento, provincia, distrito, estado_ubigeo)
WHERE NOT EXISTS (SELECT 1 FROM ubigeo x WHERE x.codigo_ubigeo = u.codigo_ubigeo);

CREATE TEMP TABLE carga_direccion (clave text PRIMARY KEY, id integer) ON COMMIT DROP;

WITH nuevas AS (
    INSERT INTO direccion (codigo_ubigeo, direccion_detallada, referencia, descripcion, estado_direccion)
    SELECT codigo, detalle, referencia, descripcion, TRUE
    FROM (VALUES
        ('sanisidro', 130107, 'Carretera Industrial Moche km 3.5, Fundo San Isidro', 'Frente al grifo Primax', 'Planta de empaque'),
        ('losandes', 130104, 'Parque Industrial La Esperanza Mz. C Lt. 12', 'Altura del óvalo Sinchi Roca', 'Centro de acopio'),
        ('vallemoche', 130107, 'Av. Industrial 2450, Moche', 'A dos cuadras del puente Moche', 'Planta de procesamiento'),
        ('ribera', 130101, 'Av. Larco 1180, Urb. San Andrés', 'Cerca a la Universidad Privada', 'Oficina de obra'),
        ('elgolf', 130101, 'Av. Juan Pablo II 785, Urb. El Golf', 'Frente al parque El Golf', 'Almacén de materiales'),
        ('concretos', 130102, 'Prolongación Av. Túpac Amaru 4120, El Porvenir', 'Junto a la cantera', 'Planta de concreto'),
        ('sombrero', 130103, 'Av. Perú 1850, Florencia de Mora', 'A media cuadra del mercado', 'Almacén de insumos mineros'),
        ('andino', 130104, 'Av. Condorcanqui 3010, La Esperanza', 'Al costado de la comisaría', 'Taller y patio de maniobras'),
        ('callao', 70101, 'Av. Néstor Gambetta 3450', 'Ingreso al terminal norte', 'Terminal portuario del Callao'),
        ('lima', 150101, 'Av. Argentina 2833, Cercado de Lima', 'Altura de la cuadra 28', 'Centro de distribución Lima'),
        ('chiclayo', 140101, 'Carretera a Lambayeque km 4.5', 'Frente a la planta de gas', 'Almacén Chiclayo'),
        ('piura', 200101, 'Zona Industrial Mz. G Lt. 5', 'Antigua Panamericana Norte', 'Almacén Piura'),
        ('cajamarca', 60101, 'Av. Vía de Evitamiento Norte 1520', 'Salida a Bambamarca', 'Almacén Cajamarca')
    ) AS d (clave, codigo, detalle, referencia, descripcion)
    RETURNING id_direccion, direccion_detallada
)
INSERT INTO carga_direccion
SELECT d.clave, n.id_direccion
FROM nuevas n
JOIN (VALUES
    ('sanisidro', 'Carretera Industrial Moche km 3.5, Fundo San Isidro'),
    ('losandes', 'Parque Industrial La Esperanza Mz. C Lt. 12'),
    ('vallemoche', 'Av. Industrial 2450, Moche'),
    ('ribera', 'Av. Larco 1180, Urb. San Andrés'),
    ('elgolf', 'Av. Juan Pablo II 785, Urb. El Golf'),
    ('concretos', 'Prolongación Av. Túpac Amaru 4120, El Porvenir'),
    ('sombrero', 'Av. Perú 1850, Florencia de Mora'),
    ('andino', 'Av. Condorcanqui 3010, La Esperanza'),
    ('callao', 'Av. Néstor Gambetta 3450'),
    ('lima', 'Av. Argentina 2833, Cercado de Lima'),
    ('chiclayo', 'Carretera a Lambayeque km 4.5'),
    ('piura', 'Zona Industrial Mz. G Lt. 5'),
    ('cajamarca', 'Av. Vía de Evitamiento Norte 1520')
) AS d (clave, detalle) ON d.detalle = n.direccion_detallada;

-- Clientes (RUC con dígito verificador válido). destinos: a dónde suelen ir sus cargas.
CREATE TEMP TABLE carga_cliente (id integer, origen integer, destinos integer[]) ON COMMIT DROP;

WITH datos AS (
    SELECT * FROM (VALUES
        ('Agroindustrias San Isidro SAC', '20601847354', '044281530', 'logistica@agrosanisidro.com.pe', 'Rocío Valverde', 'Agroindustria', 'sanisidro', ARRAY['callao', 'lima']),
        ('Frutícola Los Andes del Norte SAC', '20602931588', '044295417', 'despachos@fruticolalosandes.com.pe', 'Martín Cabrera', 'Agroindustria', 'losandes', ARRAY['callao', 'piura']),
        ('Exportadora Valle Moche SAC', '20604372161', '044263908', 'operaciones@vallemoche.com.pe', 'Gabriela Ruiz', 'Agroindustria', 'vallemoche', ARRAY['callao', 'chiclayo']),
        ('Constructora Ribera Norte SAC', '20605189424', '044207761', 'compras@riberanorte.com.pe', 'Jorge Alvarado', 'Construcción', 'ribera', ARRAY['chiclayo', 'cajamarca']),
        ('Inmobiliaria y Construcciones El Golf SAC', '20606523875', '044219034', 'logistica@construccioneselgolf.com.pe', 'Patricia Linares', 'Construcción', 'elgolf', ARRAY['lima', 'piura']),
        ('Concretos y Agregados Liberteños SAC', '20607814539', '044241186', 'ventas@concretosliberteños.com.pe', 'Raúl Zavaleta', 'Construcción', 'concretos', ARRAY['cajamarca', 'chiclayo']),
        ('Minera Cerro Sombrero SAC', '20608962141', '044236652', 'abastecimiento@cerrosombrero.com.pe', 'Fernando Paredes', 'Minería', 'sombrero', ARRAY['cajamarca', 'lima']),
        ('Servicios Mineros Andino Norte SAC', '20610235761', '044258473', 'logistica@andinonorte.com.pe', 'Silvia Rebaza', 'Minería', 'andino', ARRAY['cajamarca', 'callao'])
    ) AS c (razon, ruc, telefono, correo, contacto, rubro, origen, destinos)
),
nuevos AS (
    INSERT INTO cliente (razon_social, ruc, telefono, correo, contacto, id_rubro, id_direccion_domicilio, estado_cliente)
    SELECT d.razon, d.ruc, d.telefono, d.correo, d.contacto,
           COALESCE((SELECT min(id_rubro) FROM rubro WHERE nombre = d.rubro), (SELECT min(id_rubro) FROM rubro)),
           (SELECT id FROM carga_direccion WHERE clave = d.origen), TRUE
    FROM datos d
    RETURNING id_cliente, ruc
)
INSERT INTO carga_cliente
SELECT n.id_cliente,
       (SELECT id FROM carga_direccion WHERE clave = d.origen),
       ARRAY(SELECT cd.id FROM unnest(d.destinos) AS x (clave) JOIN carga_direccion cd ON cd.clave = x.clave)
FROM nuevos n JOIN datos d ON d.ruc = n.ruc;

-- Tractos y carretas
CREATE TEMP TABLE carga_unidad (n integer, tracto integer, carreta integer) ON COMMIT DROP;

WITH datos AS (
    SELECT * FROM (VALUES
        (1, 'T3K-482', 'Volvo', 'FH 460', 2021, 'C4L-527'),
        (2, 'T6M-915', 'Scania', 'R450', 2022, 'C7P-384'),
        (3, 'T2X-736', 'Mercedes-Benz', 'Actros 2645', 2020, 'C2H-961'),
        (4, 'T8B-251', 'Volvo', 'FH16 540', 2023, 'C5V-243'),
        (5, 'T4R-608', 'Scania', 'G410', 2019, 'C8N-670'),
        (6, 'T7D-139', 'Mercedes-Benz', 'Actros 3344', 2024, 'C3T-815')
    ) AS u (n, placa, marca, modelo, anio, carreta)
),
tractos AS (
    INSERT INTO unidad (placa, id_tipo_unidad, id_subtipo_unidad, id_marca, modelo, anio, peso_tara, peso_maximo, estado_unidad)
    SELECT d.placa, t.id, (SELECT min(id_subtipo_unidad) FROM subtipounidad WHERE id_tipo_unidad = t.id),
           COALESCE((SELECT min(id_marca) FROM marca WHERE nombre = d.marca), (SELECT min(id_marca) FROM marca)),
           d.modelo, d.anio, 8000 + d.n * 150, 42000, TRUE
    FROM datos d,
         (SELECT COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'tracto%'), (SELECT min(id_tipo_unidad) FROM tipounidad)) AS id) t
    RETURNING id_unidad, placa
),
carretas AS (
    INSERT INTO unidad (placa, id_tipo_unidad, id_subtipo_unidad, id_marca, modelo, anio, peso_tara, peso_maximo, estado_unidad)
    SELECT d.carreta, t.id, (SELECT min(id_subtipo_unidad) FROM subtipounidad WHERE id_tipo_unidad = t.id),
           COALESCE((SELECT min(id_marca) FROM marca WHERE nombre = d.marca), (SELECT min(id_marca) FROM marca)),
           'Plataforma 3 ejes', d.anio, 3600 + d.n * 100, 32000, TRUE
    FROM datos d,
         (SELECT COALESCE((SELECT min(id_tipo_unidad) FROM tipounidad WHERE nombre ILIKE 'carreta%'), (SELECT max(id_tipo_unidad) FROM tipounidad)) AS id) t
    RETURNING id_unidad, placa
)
INSERT INTO carga_unidad
SELECT d.n, t.id_unidad, c.id_unidad
FROM datos d
JOIN tractos t ON t.placa = d.placa
JOIN carretas c ON c.placa = d.carreta;

-- Conductores; los seis primeros manejan un tracto cada uno, los otros dos son relevo
CREATE TEMP TABLE carga_asignacion (n integer, id integer) ON COMMIT DROP;

WITH datos AS (
    SELECT * FROM (VALUES
        (1, '42617385', 'Wilmer', 'Antonio', 'Castañeda', 'Rodríguez', 'AIII-C', '948217365'),
        (2, '43958217', 'Segundo', 'Elías', 'Vásquez', 'Cruz', 'AIII-C', '957381462'),
        (3, '45180736', 'Hernán', NULL, 'Gamboa', 'Sánchez', 'AIII-B', '961523874'),
        (4, '46329581', 'Edwin', 'Rafael', 'Polo', 'Marquina', 'AIII-C', '972648135'),
        (5, '47206814', 'Julio', 'César', 'Benites', 'Aguilar', 'AIII-B', '983715246'),
        (6, '48451927', 'Óscar', NULL, 'Rojas', 'Valderrama', 'AIII-C', '994826357'),
        (7, '41873650', 'Víctor', 'Hugo', 'Reyes', 'Avalos', 'AIII-B', '925937468'),
        (8, '44592038', 'Marco', 'Antonio', 'León', 'Ganoza', 'AIII-C', '936148579')
    ) AS c (n, dni, nombre1, nombre2, apellido1, apellido2, licencia, telefono)
),
nuevos AS (
    INSERT INTO conductor (dni, nombre1, nombre2, apellido1, apellido2, licencia_conducir, telefono, estado_conductor)
    SELECT dni, nombre1, nombre2, apellido1, apellido2, licencia, telefono, TRUE
    FROM datos
    RETURNING id_conductor, dni
),
asignadas AS (
    INSERT INTO conductorunidad (id_conductor, id_unidad, fecha_asignacion, estado_asignacion)
    SELECT n.id_conductor, u.tracto, (CURRENT_DATE - interval '14 months')::timestamp, TRUE
    FROM nuevos n
    JOIN datos d ON d.dni = n.dni
    JOIN carga_unidad u ON u.n = d.n
    RETURNING id_conductor_unidad, id_unidad
)
INSERT INTO carga_asignacion
SELECT u.n, a.id_conductor_unidad
FROM asignadas a JOIN carga_unidad u ON u.tracto = a.id_unidad;

-- Viajes y gastos
DO $$
DECLARE
    u_ids     integer[];
    u_noms    text[];
    c_claves  text[];
    c_textos  text[];
    cli       carga_cliente;
    clientes  carga_cliente[];
    uni       carga_unidad;
    v_id      integer;
    v_dia     date;
    g_id      integer;
    c_id      integer;
    k         integer;
    j         integer;
    n_gastos  integer;
    autor     integer;
    revisor   integer;
    g_monto     numeric;
    g_fecha     date;
    cobro     date;
    g_estado    text;
    edad      integer;
    r         double precision;
    registro  timestamp;
    adjunto   timestamp;
    revision  timestamp;
    ahora     timestamp := (now() AT TIME ZONE 'UTC') - interval '1 hour';
    c_tipo      text;
    url       text;
    falla     integer;
    lista     jsonb;
    motivo    text;
    rechazar  boolean;
    datos_g   jsonb;
BEGIN
    SELECT array_agg(id_usuario ORDER BY id_usuario), array_agg(nombre ORDER BY id_usuario) INTO u_ids, u_noms
    FROM usuario WHERE rol = 'logistica' AND usuario.estado;
    SELECT array_agg(clave ORDER BY orden), array_agg(texto ORDER BY orden) INTO c_claves, c_textos
    FROM criterio_checklist WHERE cliente_id IS NULL;
    SELECT array_agg(c ORDER BY c.id) INTO clientes FROM carga_cliente c;

    -- 4 viajes por mes en los 12 meses anteriores y 4 programados en las próximas semanas
    FOR k IN 0..51 LOOP
        IF k < 48 THEN
            v_dia := (date_trunc('month', CURRENT_DATE) - make_interval(months => 11 - k / 4))::date + (2 + (k % 4) * 7);
            CONTINUE WHEN v_dia >= CURRENT_DATE;
        ELSE
            v_dia := CURRENT_DATE + (ARRAY[3, 7, 12, 18])[k - 47];
        END IF;

        cli := clientes[1 + floor(random() * array_length(clientes, 1))::int];
        SELECT * INTO uni FROM carga_unidad WHERE n = 1 + (k % 6);

        INSERT INTO programacionviaje (id_tracto, id_carreta, id_conductor_unidad, id_direccion_origen, id_direccion_destino,
                                       fecha_programado, id_cliente, estado_viaje)
        VALUES (uni.tracto, uni.carreta, (SELECT id FROM carga_asignacion WHERE n = uni.n),
                cli.origen, cli.destinos[1 + floor(random() * array_length(cli.destinos, 1))::int],
                v_dia + time '06:00' + make_interval(hours => floor(random() * 4)::int), cli.id, TRUE)
        RETURNING id_viaje INTO v_id;

        CONTINUE WHEN v_dia >= CURRENT_DATE;

        -- Combustible, peajes, viáticos y, a veces, mantenimiento
        n_gastos := 2 + floor(random() * 3)::int;
        FOR j IN 1..n_gastos LOOP
            g_monto := round((CASE j
                WHEN 1 THEN 650 + random() * 1150      -- combustible
                WHEN 2 THEN 45 + random() * 140        -- peajes
                WHEN 3 THEN 60 + random() * 190        -- viáticos
                ELSE 180 + random() * 1020             -- mantenimiento
            END)::numeric, 2);
            g_fecha := LEAST(v_dia + floor(random() * 3)::int, CURRENT_DATE);
            edad := CURRENT_DATE - g_fecha;
            r := random();
            g_estado := CASE
                WHEN edad > 60 THEN CASE WHEN r < 0.88 THEN 'Sustentado' WHEN r < 0.95 THEN 'Pagado' ELSE 'Registrado' END
                WHEN edad > 15 THEN CASE WHEN r < 0.6 THEN 'Sustentado' WHEN r < 0.82 THEN 'Pagado' ELSE 'Registrado' END
                ELSE CASE WHEN r < 0.25 THEN 'Sustentado' WHEN r < 0.6 THEN 'Pagado' ELSE 'Registrado' END
            END;
            cobro := CASE
                WHEN edad <= 15 AND g_estado <> 'Sustentado' AND random() < 0.35 THEN CURRENT_DATE + floor(random() * 3)::int
                WHEN random() < 0.65 THEN g_fecha + 15 + floor(random() * 31)::int
            END;
            autor := 1 + floor(random() * array_length(u_ids, 1))::int;
            registro := ((g_fecha + time '08:30' + make_interval(mins => floor(random() * 540)::int)) AT TIME ZONE 'America/Lima') AT TIME ZONE 'UTC';
            registro := LEAST(registro, ahora - interval '3 hours');

            INSERT INTO gasto (viaje_id, cliente_id, monto, fecha, estado, creado_por, creado_en, fecha_cobro)
            VALUES (v_id, cli.id, g_monto, g_fecha, g_estado, u_noms[autor], registro, cobro)
            RETURNING id INTO g_id;

            datos_g := jsonb_strip_nulls(jsonb_build_object(
                'fecha', g_fecha, 'monto', g_monto, 'estado', 'Registrado', 'viaje_id', v_id,
                'cliente_id', cli.id, 'creado_por', u_noms[autor], 'fecha_cobro', cobro));
            INSERT INTO bitacora (fecha, id_usuario, usuario, accion, entidad, id_entidad, detalle)
            VALUES (registro AT TIME ZONE 'UTC', u_ids[autor], u_noms[autor], 'gasto.registrar', 'gasto', g_id,
                    jsonb_build_object('despues', datos_g));

            -- Comprobantes: un rechazo previo en algunos sustentados y en parte de los registrados
            adjunto := registro;
            FOR rechazar IN
                SELECT x FROM unnest(CASE
                    WHEN g_estado = 'Sustentado' AND random() < 0.15 THEN ARRAY[TRUE, FALSE]
                    WHEN g_estado = 'Sustentado' THEN ARRAY[FALSE]
                    WHEN g_estado = 'Pagado' THEN ARRAY[NULL::boolean]
                    WHEN random() < 0.3 THEN ARRAY[TRUE]
                    ELSE ARRAY[]::boolean[]
                END) AS x
            LOOP
                adjunto := LEAST(adjunto + make_interval(hours => 1 + floor(random() * 30)::int), ahora - interval '2 hours');
                revision := LEAST(adjunto + make_interval(hours => 2 + floor(random() * 46)::int), ahora);
                revisor := 1 + floor(random() * array_length(u_ids, 1))::int;
                c_tipo := CASE WHEN random() < 0.65 THEN 'imagen' ELSE 'pdf' END;
                url := '/uploads/gasto-' || g_id || '-' || md5(random()::text) || CASE c_tipo WHEN 'imagen' THEN '.jpg' ELSE '.pdf' END;

                -- Checklist: todo cumple, salvo un criterio en los rechazados
                -- (legible, monto o fecha: los motivos más comunes; a veces, a nombre de otra empresa)
                falla := CASE WHEN rechazar THEN
                    array_position(c_claves, (ARRAY['legible', 'monto_coincide', 'monto_coincide', 'fecha_coincide', 'fecha_coincide', 'a_nombre_empresa'])[1 + floor(random() * 6)::int])
                END;
                IF rechazar AND falla IS NULL THEN
                    falla := 1;
                END IF;
                lista := (SELECT jsonb_agg(jsonb_build_object('texto', t, 'cumple', falla IS NULL OR n <> falla, 'criterio', c) ORDER BY n)
                          FROM unnest(c_claves, c_textos) WITH ORDINALITY AS x (c, t, n));
                motivo := CASE WHEN rechazar THEN
                    'No cumple: ' || lower(left(c_textos[falla], 1)) || rtrim(substr(c_textos[falla], 2), '.') || '.' END;

                INSERT INTO comprobante (gasto_id, archivo_url, tipo_archivo, estado, motivo_rechazo, aprobado_por, aprobado_en, creado_en)
                VALUES (g_id, url, c_tipo,
                        CASE WHEN rechazar IS NULL THEN 'Pendiente' WHEN rechazar THEN 'Rechazado' ELSE 'Aprobado' END,
                        motivo,
                        CASE WHEN rechazar IS NOT NULL THEN u_noms[revisor] END,
                        CASE WHEN rechazar IS NOT NULL THEN revision END,
                        adjunto)
                RETURNING id INTO c_id;

                INSERT INTO bitacora (fecha, id_usuario, usuario, accion, entidad, id_entidad, detalle)
                VALUES (adjunto AT TIME ZONE 'UTC', u_ids[autor], u_noms[autor], 'comprobante.adjuntar', 'comprobante', c_id,
                        jsonb_build_object('antes', jsonb_build_object('gasto_estado', 'Registrado'),
                                           'despues', jsonb_build_object('estado', 'Pendiente', 'archivo_url', url, 'gasto_estado', 'Pagado', 'tipo_archivo', c_tipo),
                                           'gasto_id', g_id));

                CONTINUE WHEN rechazar IS NULL;

                INSERT INTO checklist_comprobante (comprobante_id, criterio, cumple, texto)
                SELECT c_id, e->>'criterio', (e->>'cumple')::boolean, e->>'texto' FROM jsonb_array_elements(lista) AS e;

                INSERT INTO bitacora (fecha, id_usuario, usuario, accion, entidad, id_entidad, detalle)
                VALUES (revision AT TIME ZONE 'UTC', u_ids[revisor], u_noms[revisor],
                        CASE WHEN rechazar THEN 'comprobante.rechazar' ELSE 'comprobante.aprobar' END, 'comprobante', c_id,
                        jsonb_build_object('antes', jsonb_build_object('estado', 'Pendiente', 'gasto_estado', 'Pagado'),
                                           'despues', CASE WHEN rechazar
                                               THEN jsonb_build_object('estado', 'Rechazado', 'gasto_estado', 'Registrado', 'motivo_rechazo', motivo)
                                               ELSE jsonb_build_object('estado', 'Aprobado', 'gasto_estado', 'Sustentado') END,
                                           'gasto_id', g_id, 'checklist', lista));
                adjunto := revision;
            END LOOP;
        END LOOP;
    END LOOP;
END $$;

COMMIT;
