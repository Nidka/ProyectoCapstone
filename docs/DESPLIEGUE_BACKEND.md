# Despliegue en Render + Neon

Guía para poner en internet **LIZAR SRL · Fleet Manager Pro**: la API Go con su disco para comprobantes, el frontend Angular y la base en Neon. El repositorio ya está preparado; aquí queda solo lo que se configura en las plataformas, marcado con **🔧 Te toca**.

## 1. Qué se despliega

| Pieza | Dónde | Archivo que la describe |
|---|---|---|
| API Go (`gastos-service/`) | Render, servicio web Docker `lizar-gastos-api`, **plan gratuito, sin disco** | `render.yaml`, `gastos-service/Dockerfile` |
| Frontend (`frontend/`) | Render, sitio estático `lizar-frontend` (gratis) | `render.yaml` |
| Base de datos | Neon (PostgreSQL), fuera de Render | `docs/DESPLIEGUE_NEON.md` |
| `backend/` (Python) | No se despliega: ya migrado a Go | — |

URLs que quedarán si Render acepta los nombres: API `https://lizar-gastos-api.onrender.com` y frontend `https://lizar-frontend.onrender.com`.

> **Plan gratuito (elegido por ahora):** la API no tiene disco. Los comprobantes que se suban **se borran en cada despliegue o reinicio** (los datos en Neon se conservan), y la API se duerme tras unos 15 minutos sin uso y tarda cerca de un minuto en despertar. Para conservar los comprobantes, en `render.yaml` cambia `plan: free` por una instancia de pago (por ejemplo `0.5c-512mb`) y descomenta el bloque `disk`.

## 2. Lo que ya está listo en el repositorio

- **`render.yaml` (Blueprint):** crea los dos servicios de una vez. Región Ohio, junto a Neon (`us-east-2`), comprobación de salud en `/health`, despliegue automático en cada commit a `main` y una `JWT_SECRET` aleatoria generada por Render.
- **Dockerfile con disco escribible:** el contenedor arranca como `root` solo para crear `uploads/` y `logs/` dentro del disco y entregárselas al usuario `lizar`; la API corre sin privilegios. Probado con un volumen montado como `root`.
- **Frontend apuntando a producción:** la URL de la API está en un solo lugar, `frontend/src/app/core/services/api-origen.ts` (`API_PRODUCCION`). En `localhost` se sigue usando `http://localhost:8081`. Se quitó la IP vieja de la API de Python (`13.222.175.52:8000`).
- **Pruebas:** Go y frontend pasan; el build de producción del frontend compila.

## 3. Pasos

### 3.1 🔧 Te toca: Neon

1. Entra a https://console.neon.tech y abre el proyecto (`ep-sparkling-band-…`).
2. **Respaldo:** en *Branches* crea la rama `respaldo-antes-de-desplegar`.
3. En *Connect* copia la cadena **directa** (host sin `-pooler`). Si trae `&channel_binding=require`, quítalo. Debe terminar en `?sslmode=require`. La pegarás en el paso 3.3; no la subas al repositorio ni la compartas por chat.

### 3.2 🔧 Te toca: decidir los datos

Hoy Neon tiene solo los datos de ejemplo de `db.sql` (2 clientes, 0 gastos). Los datos reales están en tu PostgreSQL local, base `Cloude` (14 clientes, 50 programaciones, 163 gastos, 153 comprobantes).

- **Opción A, recomendada: copiar `Cloude` a Neon** (reemplaza lo que tiene Neon):

  ```bash
  pg_dump "postgres://postgres:TU_CLAVE@localhost:5432/Cloude" \
    --no-owner --no-privileges --format=custom -f cloude.dump
  pg_restore --no-owner --no-privileges --clean --if-exists \
    -d "CADENA_DE_NEON" cloude.dump
  rm cloude.dump
  ```

  Si prefieres que lo corra yo, dame permiso explícito para escribir en Neon.
- **Opción B: empezar limpio.** No hagas nada: al arrancar, la API crea `admin` / `admin123` y pide cambiar la clave en el primer ingreso.

Hazlo **antes** del primer despliegue o con el servicio detenido.

### 3.3 🔧 Te toca: crear los servicios en Render

1. Crea una cuenta en https://render.com y conecta GitHub con acceso a `Nidka/ProyectoCapstone`.
2. *New → Blueprint*, elige el repositorio y la rama `main`. Render lee `render.yaml` y muestra los dos servicios.
3. Te pedirá **`DATABASE_URL`**: pega la cadena de Neon del paso 3.1.
4. Confirma. Los dos servicios usan planes gratuitos.
5. Espera el primer despliegue (unos minutos cada uno).

> Si Render avisa que el nombre `lizar-gastos-api` ya está tomado, quedará con otra URL. En ese caso ve al paso 3.5.

### 3.4 Verificar

1. *Logs* de `lizar-gastos-api`: deben aparecer `lectura de comprobantes: … sí` y `gastos-service escuchando en :…`.
2. Abre `https://lizar-gastos-api.onrender.com/health` → `{"env":"prod","status":"ok"}`.
3. Abre `https://lizar-frontend.onrender.com` e inicia sesión (con tus cuentas si elegiste A; con `admin` / `admin123` si elegiste B).

### 3.5 Solo si la URL de la API es otra

Cambia la línea `API_PRODUCCION` en `frontend/src/app/core/services/api-origen.ts` por la URL real (sin barra final), haz commit y push a `main`. Render vuelve a publicar el frontend solo. También puedo hacerlo yo si me pasas la URL.

### 3.6 🔧 Te toca: copiar los comprobantes al disco

Son 19 archivos (2,9 MB) en `gastos-service/uploads/`. Solo hace falta si elegiste la opción A.

1. En tu PC: `cd gastos-service && tar czf comprobantes.tgz uploads` y luego `wormhole send comprobantes.tgz` (instálalo con `brew install magic-wormhole`). Anota el código que muestra.
2. En Render → `lizar-gastos-api` → *Shell*:

   ```sh
   apk add --no-cache py3-pip && pip install --break-system-packages magic-wormhole
   cd /app/data && wormhole receive CODIGO && tar xzf comprobantes.tgz && rm comprobantes.tgz
   chown -R lizar uploads
   ```

Render también admite SSH y `scp` en instancias de pago, si lo prefieres.

## 4. Variables de la API

| Variable | Valor | De dónde sale |
|---|---|---|
| `DATABASE_URL` | cadena directa de Neon, `?sslmode=require` | la pegas al crear el Blueprint |
| `JWT_SECRET` | aleatoria de 256 bits | la genera Render; no la cambies después o se cierran todas las sesiones |
| `UPLOADS_DIR` | `/app/data/uploads` | `render.yaml` |
| `LOG_ARCHIVO` | `/app/data/logs/errores.log` | `render.yaml` |
| `APP_ENV` | `prod` | `render.yaml` |
| `PORT` | la define Render | — |

## 5. Lo que conviene saber de Render

- Con disco, la API no se puede escalar a varias instancias y cada despliegue tiene unos segundos de corte.
- Render saca una copia del disco cada 24 horas y la guarda 7 días. El disco se puede agrandar, no achicar.
- Cada push a `main` vuelve a desplegar los dos servicios.

## 6. Problemas frecuentes

| Síntoma | Qué revisar |
|---|---|
| La API se cae al arrancar con `JWT_SECRET` | La variable se borró o quedó vacía en *Environment*. |
| `no se pudo conectar a la base de datos` o error de SSL | `DATABASE_URL` sin `?sslmode=require`, con `channel_binding`, o con el host `-pooler`. |
| El frontend abre pero no deja entrar | La API no responde en `/health`, o `API_PRODUCCION` no coincide con la URL real (paso 3.5). |
| Los comprobantes antiguos no se ven | Faltó copiar `uploads/` al disco (paso 3.6). |
| Una ruta del frontend da 404 al recargar | Falta la regla de reescritura `/*` → `/index.html` del sitio estático (está en `render.yaml`). |
