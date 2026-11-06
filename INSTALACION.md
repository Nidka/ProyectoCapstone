# Levantar el proyecto en otra PC

Guía paso a paso para tener **LIZAR SRL · Fleet Manager Pro** funcionando en una computadora nueva (Windows, macOS o Linux).

Al terminar tendrás:

| Parte | Dirección |
|---|---|
| Frontend (Angular) | http://localhost:4200 |
| API (Go) | http://localhost:8081 — comprobación: http://localhost:8081/health |
| Base de datos (PostgreSQL 17) | `localhost:5432` (o Neon, en la nube) |

Hay dos caminos. **Elige uno:**

- **Opción A — con Docker (recomendada):** Docker levanta la base y la API; tú solo instalas Node para el frontend. Menos cosas que instalar.
- **Opción B — todo local:** instalas PostgreSQL y Go en la PC. Útil para desarrollar el backend.

---

## 1. Requisitos

| Programa | Versión | Para qué | Opción A | Opción B |
|---|---|---|---|---|
| [Git](https://git-scm.com/downloads) | cualquiera reciente | Clonar el repositorio | ✅ | ✅ |
| [Node.js](https://nodejs.org/) | 22 LTS o 24 (trae npm) | Frontend Angular 21 | ✅ | ✅ |
| [Docker Desktop](https://www.docker.com/products/docker-desktop/) | reciente | Base + API en contenedores | ✅ | — |
| [Go](https://go.dev/dl/) | 1.25 o superior | API | — | ✅ |
| [PostgreSQL](https://www.postgresql.org/download/) | 17 | Base de datos | — | ✅ |
| Tesseract (con idioma `spa`) y Poppler | opcional | Lectura automática (OCR) de comprobantes | ya viene en la imagen | opcional |

Sin Tesseract y Poppler el sistema funciona igual: solo la lectura automática del comprobante responde «no disponible».

Comprueba las versiones en una terminal:

```bash
git --version
node --version      # v22.x o v24.x
npm --version
docker --version    # solo opción A
go version          # solo opción B: go1.25 o más
psql --version      # solo opción B: 17.x
```

## 2. Clonar el repositorio

```bash
git clone https://github.com/Nidka/ProyectoCapstone.git
cd ProyectoCapstone
```

La rama estable es `main`.

## 3. Generar la clave `JWT_SECRET`

La API **no arranca** sin una clave propia de al menos 32 caracteres, que firma las sesiones. Genera una y guárdala; la usarás en el paso siguiente.

- **macOS / Linux / Git Bash:**
  ```bash
  openssl rand -hex 32
  ```
- **Windows (PowerShell):**
  ```powershell
  -join ((1..64) | ForEach-Object { '{0:x}' -f (Get-Random -Maximum 16) })
  ```

No subas esta clave al repositorio.

---

## 4A. Opción A — Base y API con Docker

Con Docker Desktop abierto, desde la carpeta del proyecto:

- **macOS / Linux:**
  ```bash
  JWT_SECRET=pega-aqui-tu-clave docker compose -f docker-compose.demo.yml up --build -d
  ```
- **Windows (PowerShell):**
  ```powershell
  $env:JWT_SECRET="pega-aqui-tu-clave"
  docker compose -f docker-compose.demo.yml up --build -d
  ```

Esto:

1. Crea un PostgreSQL 17 (base `lizar_demo`, usuario `lizar`) y carga `db.sql` automáticamente la primera vez.
2. Compila y levanta la API en el puerto 8081, que aplica las migraciones al arrancar.

Comprueba que respondió: abre http://localhost:8081/health.

Comandos útiles:

```bash
docker compose -f docker-compose.demo.yml logs -f api   # ver el log de la API
docker compose -f docker-compose.demo.yml down          # apagar (los datos se conservan)
docker compose -f docker-compose.demo.yml down -v       # apagar y BORRAR datos (base y comprobantes)
```

Sigue con el **paso 5**.

## 4B. Opción B — Base y API instaladas en la PC

### Base de datos

Crea la base y carga el esquema con los datos iniciales (una sola vez). Cambia `postgres` y la contraseña por las de tu instalación:

```bash
createdb -U postgres lizar
psql -U postgres -d lizar -f db.sql
```

En Windows, si `createdb` o `psql` no se reconocen, usa la ruta completa (por ejemplo `"C:\Program Files\PostgreSQL\17\bin\psql.exe"`) o crea la base desde pgAdmin y luego ejecuta `db.sql` con su herramienta de consultas.

### API

```bash
cd gastos-service
cp .env.example .env        # en Windows (PowerShell): Copy-Item .env.example .env
```

Abre `gastos-service/.env` y completa:

```env
PORT=8081
DATABASE_URL=postgres://postgres:TU_CLAVE@localhost:5432/lizar?sslmode=disable
UPLOADS_DIR=./uploads
LOG_ARCHIVO=logs/errores.log
JWT_SECRET=pega-aqui-tu-clave
```

Arranca la API:

```bash
go run .
```

La primera vez descarga dependencias, aplica las migraciones (`gastos-service/migrations/`) y crea el usuario administrador inicial. Deja esta terminal abierta.

Comprueba: http://localhost:8081/health.

> **Base en la nube (Neon) en lugar de PostgreSQL local:** sigue [docs/DESPLIEGUE_NEON.md](docs/DESPLIEGUE_NEON.md). Basta con poner en `DATABASE_URL` la cadena de Neon (termina en `?sslmode=require`).

---

## 5. Frontend

En **otra terminal**, desde la carpeta del proyecto:

```bash
cd frontend
npm ci
npx ng serve
```

`npm ci` instala exactamente las versiones de `package-lock.json` (tarda unos minutos la primera vez). Cuando aparezca `Local: http://localhost:4200/`, ábrelo en el navegador.

Si abres el frontend en `localhost`, se conecta solo a la API en `http://localhost:8081`.

## 6. Primer ingreso

| Usuario | Contraseña inicial |
|---|---|
| `admin` | `admin123` |

El sistema obliga a **cambiar la contraseña** en el primer ingreso. Después, desde **Usuarios**, el administrador puede crear las cuentas de los demás roles (logística, etc.).

El admin inicial se crea solo cuando la base no tiene ningún usuario. Si la base ya tenía usuarios (por ejemplo, una copia de otra PC), entra con esas cuentas.

---

## 7. Pruebas (opcional)

```bash
# Backend (sin TEST_DATABASE_URL se omiten las pruebas contra PostgreSQL)
cd gastos-service
go test ./...

# Frontend
cd frontend
npx ng test --watch=false
```

Para correr también las pruebas de integración del backend, crea una base vacía de pruebas y pásala en `TEST_DATABASE_URL=postgres://…`.

## 8. Problemas frecuentes

| Síntoma | Causa y solución |
|---|---|
| La API se cierra al arrancar y menciona `JWT_SECRET` | Falta la clave o tiene menos de 32 caracteres. Repite el **paso 3**. |
| `connection refused` o `password authentication failed` | PostgreSQL no está encendido, o el usuario y la clave de `DATABASE_URL` no coinciden. |
| `relation ... does not exist` | No se cargó `db.sql` en esa base. Ejecuta `psql ... -f db.sql`. |
| `port is already allocated` / `address already in use` | El puerto 5432, 8081 o 4200 ya está en uso. Detén el otro programa (por ejemplo, un PostgreSQL local cuando usas la opción A) o, para el frontend, usa `npx ng serve --port 4300`. |
| El frontend carga pero no trae datos o no deja ingresar | La API no está corriendo (revisa http://localhost:8081/health), o abriste el frontend con una IP en lugar de `localhost`. Ábrelo como `http://localhost:4200` y, si sigue igual, borra los datos del sitio (`localStorage`) en el navegador. |
| La lectura automática del comprobante dice «no disponible» | Faltan Tesseract (con `spa`) o Poppler en la PC. En Docker ya vienen instalados. |
| Docker: la base no tiene datos nuevos de `db.sql` | `db.sql` solo se carga la primera vez. Para recargarla: `docker compose -f docker-compose.demo.yml down -v` y vuelve a levantar (borra los datos). |

## 9. Qué no viaja con el repositorio

Estas cosas no están en Git y hay que crearlas o copiarlas a mano si se necesitan:

- `gastos-service/.env` y `.env.local` (claves y conexiones).
- `gastos-service/uploads/` (comprobantes subidos) y `gastos-service/logs/`.
- Los datos que se hayan cargado en la base de la otra PC. Para llevarlos: `pg_dump` en la PC original y `psql -f` en la nueva.
