-- Usuarios del sistema (antes los creaba la API de Python con SQLAlchemy). Idempotente: si la
-- tabla ya existe, con sus datos, no se toca.

CREATE TABLE IF NOT EXISTS usuario (
    id_usuario      SERIAL PRIMARY KEY,
    username        VARCHAR(50)  NOT NULL UNIQUE,
    email           VARCHAR(100) NOT NULL UNIQUE,
    hashed_password VARCHAR(200) NOT NULL,
    nombre          VARCHAR(100) NOT NULL,
    rol             VARCHAR(20)  NOT NULL DEFAULT 'operador',
    estado          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP    NOT NULL DEFAULT now()
);
