# Despliegue de la base de datos en Neon

Neon aloja PostgreSQL en la nube. Solo reemplaza al PostgreSQL local: el servicio en Go, el frontend y la carpeta de comprobantes se despliegan aparte.

## 1. Crear la base en Neon

1. Entra a https://console.neon.tech y crea un proyecto: PostgreSQL **17** (la versión que documenta este repositorio) y una región cercana (por ejemplo, São Paulo, `sa-east-1`).
2. En el panel pulsa **Connect**, elige el rol y la base (`neondb` por defecto) y copia la cadena de conexión.
3. Úsala con `sslmode=require` y el host **directo** (sin `-pooler`). Si prefieres el host con `-pooler`, añade `&default_query_exec_mode=simple_protocol`. Quita `channel_binding=require` si aparece.

## 2. Cargar los maestros (una sola vez)

`db.sql` crea las tablas de maestros y operación con sus datos de prueba. No es idempotente: ejecútalo una vez, con `psql` instalado:

```
psql "postgres://USUARIO:CLAVE@ep-xxxx.sa-east-1.aws.neon.tech/neondb?sslmode=require" -f db.sql
```

## 3. Arrancar el servicio

Define las variables y arranca; al iniciar aplica solas las migraciones `0001` a `0009` (son idempotentes) y crea el usuario `admin` si no hay usuarios, que debe cambiar su clave en el primer ingreso (REQ-32).

```
cd gastos-service
DATABASE_URL="postgres://USUARIO:CLAVE@ep-xxxx.sa-east-1.aws.neon.tech/neondb?sslmode=require" \
JWT_SECRET="$(openssl rand -hex 32)" go run .
```

Con Docker: `NEON_DATABASE_URL=... JWT_SECRET=... docker compose -f docker-compose.neon.yml up --build`.

## 4. Verificar

- `GET http://localhost:8081/health` responde 200.
- En el panel de Neon, **Tables** muestra `gasto`, `comprobante`, `usuario`, `bitacora` y `notificacion`.
- Inicia sesión con el usuario `admin` y cambia su clave.

## Notas

- **Clave y cadena de conexión:** no se suben al repositorio. Van en `gastos-service/.env` (ignorado por git) o en las variables del servidor.
- **Suspensión automática:** en el plan gratuito, Neon suspende el cómputo tras unos minutos sin uso; la primera petición después tarda un poco más.
- **Comprobantes:** Neon guarda solo la base de datos. Los archivos siguen en `UPLOADS_DIR` del servidor donde corra la API; allí hay que programar el respaldo (`scripts/respaldo.sh`).
- **Respaldo de la base:** Neon conserva un historial para restaurar a un punto anterior; revisa en el panel cuántos días incluye tu plan.
- **Pruebas de Go:** apúntalas a una rama o base de pruebas, no a la de producción (`TEST_DATABASE_URL`).
