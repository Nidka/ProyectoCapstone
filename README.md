# LIZAR SRL · Fleet Manager Pro

Sistema de gestión de flota: maestros y operación (clientes, flota, viajes…), gastos adicionales con comprobantes, checklist documental, aprobación y reportes.

## Servicios

| Carpeta | Qué es | Puerto |
|---|---|---|
| `gastos-service/` | **API única en Go**: `/api/v1` (maestros, operación, login), gastos, comprobantes y reportes | 8081 |
| `frontend/` | Angular 21 | 4200 |
| `backend/` | API anterior en Python/FastAPI, **ya migrada a Go**: se conserva solo como referencia | 8000 |

## Arranque local

1. PostgreSQL con la base cargada: `psql "$DATABASE_URL" -f db.sql` (una sola vez).
2. API: `cd gastos-service && cp .env.example .env` (si existe) o define `DATABASE_URL`, y `go run .`. Al arrancar aplica las migraciones y crea el usuario `admin` / `admin123` si no hay usuarios.
3. Frontend: `cd frontend && npm install && npx ng serve`.

Variables: `PORT` (8081), `DATABASE_URL`, `UPLOADS_DIR`, `LOG_ARCHIVO` (`logs/errores.log`), `JWT_SECRET` y `APP_ENV` (`dev`). `JWT_SECRET` es obligatoria (al menos 32 caracteres, por ejemplo `openssl rand -hex 32`): sin ella, o con la clave pública de versiones anteriores, el servicio no arranca. Cambia además la contraseña de `admin`. `GET /uploads/...` no pide token (el navegador carga los comprobantes con `<img>` y `<a href>`), pero no lista la carpeta y los nombres nuevos llevan una parte aleatoria.

## Registro de errores y respaldo

- **Errores técnicos (RNF-14):** el servicio los escribe en la consola y también en el archivo `LOG_ARCHIVO` (por defecto `gastos-service/logs/errores.log`). El directorio se crea solo y el archivo crece al final; no se rota solo, así que conviene archivarlo de vez en cuando. Si no se puede abrir el archivo, el servicio arranca igual y avisa por consola.
- **Comprobantes (RNF-5):** cada archivo guardado en `uploads/` queda en solo lectura (permisos `0444`).
- **Respaldo de comprobantes (RNF-4):** `gastos-service/scripts/respaldo.sh` copia `uploads/` a una carpeta nueva con la fecha, por ejemplo `respaldos/uploads-20261022-030000`, sin tocar los respaldos anteriores. Usa dos variables opcionales: `UPLOADS_DIR` (por defecto `gastos-service/uploads`) y `RESPALDO_DIR` (por defecto `gastos-service/respaldos`). Para probarlo a mano: `UPLOADS_DIR=/ruta/uploads RESPALDO_DIR=/ruta/respaldos gastos-service/scripts/respaldo.sh`.

  Para programarlo todos los días a las 3:00 con cron, ejecuta `crontab -e` y agrega una línea (con rutas absolutas):

  ```
  0 3 * * * UPLOADS_DIR=/ruta/a/gastos-service/uploads RESPALDO_DIR=/ruta/a/respaldos /ruta/a/gastos-service/scripts/respaldo.sh >> /ruta/a/respaldos/respaldo.log 2>&1
  ```

  Cada ejecución deja un respaldo completo: borra a mano los más viejos cuando ya no los necesites. Para que el respaldo sea útil, `RESPALDO_DIR` debería estar en otro disco o equipo. En Docker, el volumen `lizar_demo_uploads` guarda los comprobantes y `lizar_demo_logs` el archivo de errores.

## Pruebas

- Go: `cd gastos-service && TEST_DATABASE_URL=postgres://… go test ./...` (sin la variable, las pruebas con base de datos se omiten).
- Frontend: `cd frontend && npx ng test --watch=false`.
- Contrato con la API de Python: las respuestas capturadas están en `gastos-service/internal/crud/contrato/`; para regenerarlas, `gastos-service/tools/contrato/`.

## Demo en Docker

`docker compose -f docker-compose.demo.yml up --build` levanta PostgreSQL (con `db.sql`) y la API Go en el puerto 8081.
