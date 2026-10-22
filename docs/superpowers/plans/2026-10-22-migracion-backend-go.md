# Migración del backend de Python a Go — plan de implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Todo el backend corre en un solo servicio Go (`gastos-service`, puerto 8081): las 118 operaciones de la API de Python pasan a Go con las mismas URL, campos, códigos y errores, y el frontend usa un único servidor.

**Architecture:** Un paquete `internal/crud` con un motor genérico y una tabla generada de 19 definiciones de recursos reemplaza las 19 copias router + casos de uso + repositorio de Python. `internal/auth` porta login, registro y `me` (bcrypt + JWT HS256, compatibles con los de Python). Un test de contrato repite sobre Go un escenario de 552 pasos capturado de la API de Python real y exige las mismas respuestas.

**Tech Stack:** Go 1.25 + pgx v5 + PostgreSQL 17; `github.com/golang-jwt/jwt/v5` v5.3.1 y `golang.org/x/crypto` v0.55.0; Angular 21 + Vitest; Python 3.14 (solo como referencia para capturar el contrato).

**Spec:** `docs/superpowers/specs/2026-10-22-migracion-backend-go-design.md`

**Nota:** todo el código de este plan se probó antes en una copia del proyecto en el scratchpad: contrato 552/552, 59 pruebas Go, 106 consultas comparadas contra el Python real sobre los datos reales sin diferencias, y compatibilidad de tokens y hashes en ambos sentidos.

## Global Constraints

- Se trabaja en `main` sin ramas ni commits (decisión del usuario). Los pasos «Checkpoint» no hacen commit.
- No se modifican `frontend/src/app/core/models/gasto-estados.ts` ni `frontend/src/app/core/utils/error-messages.ts` (RNF-1).
- **No se borra ni se modifica la carpeta `backend/`** (Python queda intacto hasta que el usuario decida retirarla) ni se renombra `gastos-service`.
- Dependencias nuevas, exactamente: `github.com/golang-jwt/jwt/v5 v5.3.1` y `golang.org/x/crypto v0.55.0` (la última compatible con Go 1.25: la v0.56 exige Go 1.26). La directiva `go 1.25.4` de `go.mod` no cambia. Ninguna otra dependencia directa.
- Mismas URL bajo `/api/v1`, mismos nombres y orden de campos, mismos códigos (200, 201, 401, 404, 409, 422, 500). Errores de `/api/v1`: `{"detail": "<mensaje>"}`; los de gastos, comprobantes y reportes siguen como `{"error": …}`.
- Decimales como texto con escala 2 (`"8000.00"`); fechas con hora como `2026-09-29T08:00:00` (microsegundos solo si los hay); fechas como `2026-09-29`.
- Los nombres de tabla y de columna salen solo de las definiciones; los valores siempre van como parámetros SQL (nunca concatenados).
- Clave JWT: variable `JWT_SECRET`; sin ella, `lizar-srl-fleet-manager-secret-2025` (la de Python, para que los tokens ya emitidos sigan valiendo). Admin sembrado `admin` / `admin123` solo si no hay usuarios. Ambos quedan anotados como deuda a cambiar en producción.
- Sin protección nueva: solo `/auth/me` exige sesión, igual que hoy.
- Las pruebas de integración usan `TEST_DATABASE_URL` y un esquema temporal `prueba_<n>` que se borra al terminar; nunca escriben en las tablas reales. La base de referencia `lizar_contrato` y la de verificación en navegador son temporales y se borran al final.
- El `ng serve` del usuario en el puerto 4200 y su gastos-service en 8081 / Python en 8000 no se tocan durante el desarrollo; las pruebas usan los puertos 8092 (Python de referencia), 8093 y 4300. Al final (Tarea 8) se reinicia gastos-service en 8081 con el código nuevo, como se hizo antes.
- Respaldar fuentes y scripts en `.superpowers/sdd/2026-10-22-migracion-backend-go/fuentes/` (el scratchpad pierde archivos); las fuentes de este plan ya están allí.
- Comandos: Go `cd gastos-service && TEST_DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" go test ./... -count=1`; frontend `cd frontend && npx ng test --watch=false` y `npx ng build`. Si el chequeo de seguridad rechaza un `bash -c` con comillas anidadas, usar un script en el scratchpad.

## Review Focus

1. **Escala de los decimales:** `peso_tara`, `monto_total`, etc. salen como `"8000.00"` aunque la base guarde `8000`, y se aceptan como número o texto al crear. Lo fijan el contrato (Tarea 5), `TestCoercionarBoolDecimalYFechas` (Tarea 2) y `TestAltaDeRecursosConPadreUnico` (Tarea 3).
2. **Tokens y hashes ya existentes:** un usuario creado por Python entra en Go y su token vale en ambos. Lo fijan `TestSeAceptaElHashDePython`, `TestSeAceptaElTokenDePython` y `TestContrasenasLargasSeRecortanA72BytesComoEnPython` (Tarea 4) y la comprobación en vivo de la Tarea 8.
3. **Inyección SQL por `search`, `filters` y `order_by`:** texto malicioso no ejecuta nada y no cambia el resultado. Lo fija `TestLaBusquedaNoEsUnaInyeccionSQL` (Tarea 3).
4. **Ids de secuencia deterministas:** el contrato solo es válido si se captura y se repite sobre bases recién cargadas; capturas parciales o repetidas desplazan los ids. Lo fija el paso de reinicio de la referencia (Tarea 1) y la verificación de que el contrato pasa 552/552 (Tarea 5).
5. **Clientes del navegador:** PUT y DELETE deben pasar el CORS (antes `gastos-service` solo permitía GET y POST), el `/health` conserva `status` y `env`, y una base guardada en el navegador que apunte al 8000 se reemplaza. Lo fijan `TestCORSPermitePUTyDELETEYLaSaludIncluyeElEntorno` (Tarea 3) y `api.service.spec.ts` (Tarea 6).

## Mapa de archivos

| Archivo | Responsabilidad |
|---|---|
| `gastos-service/tools/contrato/generar_recursos.py` | Genera `recursos.go` desde el OpenAPI, los modelos y los casos de uso de Python |
| `gastos-service/tools/contrato/capturar_contrato.py` | Arma el escenario de contrato y captura las respuestas de Python |
| `gastos-service/tools/contrato/reiniciar_referencia.sh` | Deja el Python de referencia sobre una base recién cargada |
| `gastos-service/tools/contrato/comparar_en_vivo.py` | Compara Python y Go sobre la misma base (solo lecturas) |
| `gastos-service/internal/crud/recursos.go` | Las 19 definiciones de recursos (generado) |
| `gastos-service/internal/crud/tipos.go` | Tipos de la definición y salida JSON igual a la de Python |
| `gastos-service/internal/crud/validacion.go` | Validación y conversión con los errores de FastAPI |
| `gastos-service/internal/crud/motor.go` | SQL de listar, obtener, crear, actualizar, borrar y lista avanzada |
| `gastos-service/internal/crud/http.go` | Rutas y parámetros de los 19 recursos |
| `gastos-service/internal/crud/contrato/` | Escenario y respuestas capturadas de Python |
| `gastos-service/internal/auth/` | bcrypt, JWT, usuarios, siembra del admin y rutas `/auth/*` |
| `gastos-service/migrations/0003_usuario.sql` | Tabla `usuario` |
| `gastos-service/internal/pruebas/esquema.go` | Esquemas temporales (mínimo y completo con `db.sql`) |
| `gastos-service/Dockerfile`, `.dockerignore`, `docker-compose.demo.yml`, `README.md` | Despliegue y arranque |

---

### Task 1: Herramientas de contrato, definiciones generadas y captura de Python

**Files:**
- Create: `gastos-service/tools/contrato/generar_recursos.py`, `capturar_contrato.py`, `reiniciar_referencia.sh`, `comparar_en_vivo.py`
- Create: `gastos-service/internal/crud/recursos.go` (generado)
- Create: `gastos-service/internal/crud/contrato/escenario.json`, `python.json` (capturados)

**Interfaces:**
- Consumes: el Python real (`backend/`, con su `.venv`), `db.sql`, PostgreSQL local.
- Produces: `crud.Recursos []Recurso` (19 definiciones; compila recién con la Tarea 2) y las fixtures del contrato: 552 pasos en `escenario.json` y su respuesta de Python en `python.json`, en el mismo orden.

- [ ] **Step 1: Copiar las herramientas**

`gastos-service/tools/contrato/generar_recursos.py`:

```python
"""Genera internal/crud/recursos.go desde el OpenAPI de Python, models.py y los casos de uso.
Uso: python3 generar_recursos.py <backend_dir> > recursos.go   (con el Python corriendo en :8000)"""
import json, re, sys, urllib.request, pathlib

backend = pathlib.Path(sys.argv[1])
api = json.load(urllib.request.urlopen('http://localhost:8000/openapi.json', timeout=10))
S = api['components']['schemas']
router = (backend / 'app/features/advanced_crud/presentation/router.py').read_text()
models = (backend / 'app/infrastructure/db/models.py').read_text()

tablas = dict(re.findall(r'class (\w+Model)\(Base\):\s*\n\s*__tablename__ = "(\w+)"', models))
modulo_de = dict(re.findall(r'from app\.features\.(\w+)\.application\.schemas import \w+, (\w+Read)', router))
modulo_de = {v: k for k, v in modulo_de.items()}

patron = re.compile(r'ResourceConfig\("([\w-]+)", "(\w+)", (\w+), (\w+), (\w+), \[([^\]]*)\]\)')
recursos = []
for ruta, pk, modelo, leer, crear, busq in patron.findall(router):
    recursos.append(dict(ruta=ruta, pk=pk, tabla=tablas[modelo], leer=leer, crear=crear,
                         busq=re.findall(r'"(\w+)"', busq), modulo=modulo_de[leer]))
assert len(recursos) == 19, len(recursos)

def tipo(p):
    nulo = False
    if 'anyOf' in p:
        partes = [x for x in p['anyOf'] if x.get('type') != 'null']
        nulo = len(partes) != len(p['anyOf'])
        # Decimal de lectura = string con pattern; de creación = number|string con pattern
        if any('pattern' in x for x in partes):
            return 'TDecimal', nulo, {}
        p = partes[0]
    if 'pattern' in p:  # Decimal no nulo: string con pattern
        return 'TDecimal', nulo, {}
    t, f = p.get('type'), p.get('format')
    extra = {k: p[k] for k in ('maxLength', 'minLength') if k in p}
    if f == 'date-time': return 'TFechaHora', nulo, extra
    if f == 'date': return 'TFecha', nulo, extra
    if f == 'email': return 'TTexto', nulo, extra
    return {'integer': 'TEntero', 'string': 'TTexto', 'boolean': 'TBool', 'number': 'TDecimal'}[t], nulo, extra

def reglas(modulo):
    txt = (backend / f'app/features/{modulo}/application/use_cases.py').read_text()
    no_encontrado = re.search(r'raise NotFoundError\("([^"]+)"\)', txt).group(1)
    conflicto = re.search(r'get_by_\w+\(payload\.(\w+)\):\s*\n\s*raise ConflictError\("([^"]+)"\)', txt)
    return no_encontrado, (conflicto.group(1), conflicto.group(2)) if conflicto else None

def q(s): return json.dumps(s, ensure_ascii=False)

out = ['// Código generado por generar_recursos.py a partir del OpenAPI de la API de Python; no editar a mano.',
       'package crud', '', '// Recursos son los 19 recursos CRUD de /api/v1, en el orden del OpenAPI de Python.', 'var Recursos = []Recurso{']
for r in recursos:
    leer, crear = S[r['leer']], S[r['crear']]
    ne, conf = reglas(r['modulo'])
    pk_dado = r['pk'] in crear['properties']
    out.append('\t{')
    out.append(f"\t\tRuta: {q(r['ruta'])}, Tabla: {q(r['tabla'])}, PK: {q(r['pk'])}, PKDado: {str(pk_dado).lower()},")
    out.append(f"\t\tNoEncontrado: {q(ne)},")
    if conf:
        out.append(f"\t\tUnico: &Unico{{Campo: {q(conf[0])}, Mensaje: {q(conf[1])}}},")
    out.append(f"\t\tBusqueda: []string{{{', '.join(q(b) for b in r['busq'])}}},")
    out.append('\t\tCampos: []Campo{')
    for nombre, p in leer['properties'].items():
        t, nulo, extra = tipo(p)
        c = crear['properties'].get(nombre)
        partes = [f"Nombre: {q(nombre)}", f"Tipo: {t}"]
        if c is not None:
            if nombre in crear.get('required', []): partes.append('Obligatorio: true')
            if 'default' in c: partes.append(f"Defecto: {json.dumps(c['default'])}")
            else: pass
        else:
            partes.append('SoloLectura: true')
        if nulo: partes.append('Nulo: true')
        if 'maxLength' in extra: partes.append(f"Max: {extra['maxLength']}")
        if 'minLength' in extra: partes.append(f"Min: {extra['minLength']}")
        if nombre == 'email': partes.append('Email: true')
        out.append('\t\t\t{' + ', '.join(partes) + '},')
    out.append('\t\t},')
    out.append('\t},')
out.append('}')
print('\n'.join(out))
```

`gastos-service/tools/contrato/capturar_contrato.py`:

```python
"""Arma el escenario de contrato (solo con lecturas) y captura las respuestas de la API de Python.
Uso: python3 capturar_contrato.py <url_python> <carpeta_salida>
Corre contra una base recién cargada con db.sql (nunca contra datos reales): el escenario
se arma con GET y luego se reproduce completo, de modo que Go pueda repetirlo sobre otra base
recién cargada y obtener los mismos ids de las secuencias."""
import json, sys, urllib.parse, urllib.request, urllib.error, pathlib

base, salida = sys.argv[1].rstrip('/'), pathlib.Path(sys.argv[2])
api = json.load(urllib.request.urlopen(base + '/openapi.json', timeout=10))
S = api['components']['schemas']


def llamar(metodo, ruta, cuerpo=None, token=None, cabeceras=None):
    datos = None if cuerpo is None else json.dumps(cuerpo).encode()
    req = urllib.request.Request(base + ruta, data=datos, method=metodo)
    if datos is not None:
        req.add_header('Content-Type', 'application/json')
    if token:
        req.add_header('Authorization', 'Bearer ' + token)
    for k, v in (cabeceras or {}).items():
        req.add_header(k, v)
    try:
        r = urllib.request.urlopen(req, timeout=20)
        estado, texto = r.status, r.read().decode()
    except urllib.error.HTTPError as e:
        estado, texto = e.code, e.read().decode()
    try:
        return estado, json.loads(texto)
    except ValueError:
        return estado, texto


# Recursos y reglas leídos de la fuente de verdad: el propio OpenAPI de Python
recursos = []
for ruta, ops in api['paths'].items():
    if ruta.startswith('/api/v1/advanced/'):
        nombre = ruta.split('/')[-1]
        leer = ops['get']['responses']['200']['content']['application/json']['schema']['items']['$ref'].split('/')[-1]
        recursos.append((nombre, leer, leer.replace('Read', 'Create')))

def q(txt):
    return urllib.parse.quote(txt, safe='')


pasos = []  # lista del escenario: {metodo, ruta, cuerpo?, token?, cabeceras?}


def paso(metodo, ruta, cuerpo=None, token=None, cabeceras=None, guarda=None):
    p = {'metodo': metodo, 'ruta': ruta}
    if guarda: p['guarda'] = guarda
    if cuerpo is not None: p['cuerpo'] = cuerpo
    if token: p['token'] = token
    if cabeceras: p['cabeceras'] = cabeceras
    pasos.append(p)


def inventar(prop):
    p = [x for x in prop['anyOf'] if x.get('type') != 'null'][0] if 'anyOf' in prop else prop
    if any('pattern' in x for x in prop.get('anyOf', [])): return '1.50'
    t, f = p.get('type'), p.get('format')
    if f == 'date-time': return '2026-01-01T08:00:00'
    if f == 'date': return '2026-01-01'
    return {'integer': 1, 'string': 'X', 'boolean': True, 'number': 1.5}[t]


def cuerpo_nuevo(nombre, leer, crear, it0, unico):
    props, req = S[crear]['properties'], S[crear].get('required', [])
    cuerpo = {}
    for k, v in props.items():
        if it0 is not None and k in it0 and it0[k] is not None:
            cuerpo[k] = it0[k]
        elif k in req:
            cuerpo[k] = inventar(v)
    if unico and unico in cuerpo:
        v = cuerpo[unico]
        spec = [x for x in props[unico].get('anyOf', [props[unico]]) if x.get('type') != 'null'][0]
        if isinstance(v, int):
            cuerpo[unico] = 990000 + v
        elif spec.get('minLength') and spec.get('minLength') == spec.get('maxLength'):
            cuerpo[unico] = '9' * (spec['minLength'] - 1) + '7'
        else:
            mx = spec.get('maxLength', 100)
            cuerpo[unico] = (str(v)[:mx - 3] + '-X7')
    return cuerpo


# Campo con restricción de unicidad en el caso de uso de Python (para provocar el 409)
unicos = {'ubigeos': 'codigo_ubigeo', 'rubros': 'nombre', 'marcas': 'nombre', 'tipos-unidad': 'nombre', 'tipos-gasto': 'nombre',
          'categorias-producto': 'nombre', 'clientes': 'ruc', 'conductores': 'dni', 'unidades': 'placa',
          'guias-transportista': 'numero_guia', 'facturas': 'numero_factura'}

for nombre, leer, crear in sorted(recursos):
    r = f'/api/v1/{nombre}'
    estado, lista = llamar('GET', r + '?limit=2')
    it0 = lista[0] if lista else None
    pk = next(k for k in S[leer]['properties'] if k not in S[crear]['properties'] or k == unicos.get(nombre) and nombre == 'ubigeos')
    if nombre == 'ubigeos': pk = 'codigo_ubigeo'
    paso('GET', r + '?limit=2')
    paso('GET', r + '?limit=1&offset=1')
    paso('GET', r + '?limit=0')
    paso('GET', r + '?limit=201')
    paso('GET', r + '?offset=-1')
    if it0:
        paso('GET', f'{r}/{it0[pk]}')
    paso('GET', f'{r}/999999')
    paso('GET', f'{r}/abc')
    # búsqueda y orden avanzados
    paso('GET', f'/api/v1/advanced/{nombre}?limit=3&order_dir=desc')
    paso('GET', f'/api/v1/advanced/{nombre}?limit=3&order_by={pk}&order_dir=asc&offset=1')
    paso('GET', f'/api/v1/advanced/{nombre}?order_dir=sideways')
    paso('GET', f'/api/v1/advanced/{nombre}?limit=500')
    if it0:
        texto = next((k for k, v in it0.items() if isinstance(v, str) and len(v) > 3 and not v[0].isdigit()), None)
        if texto:
            paso('GET', f'/api/v1/advanced/{nombre}?search={q(it0[texto][1:4])}&limit=5')
            paso('GET', f'/api/v1/advanced/{nombre}?search={q(it0[texto][1:4])}&search_fields={texto},campo_falso')
            paso('GET', f'/api/v1/advanced/{nombre}?filters={q(texto + ":" + str(it0[texto]))}')
        paso('GET', f'/api/v1/advanced/{nombre}?filters={pk}:{it0[pk]},campo_falso:x')
        paso('GET', f'/api/v1/advanced/{nombre}?order_by=campo_falso&limit=2')
    # alta, duplicado, validaciones, cambio y baja
    nuevo = cuerpo_nuevo(nombre, leer, crear, it0, unicos.get(nombre))
    paso('POST', r, nuevo, guarda=pk)  # {id} apunta a lo que crea este paso
    paso('POST', r, nuevo)
    paso('POST', r, {})
    campo_int = next((k for k, v in S[crear]['properties'].items() if v.get('type') == 'integer' and k in S[crear].get('required', []) and not k.startswith('id_') and k != 'codigo_ubigeo'), None)
    fk = next((k for k in S[crear]['properties'] if k.startswith('id_') and k in S[crear].get('required', [])), None)
    if fk:
        malo = dict(nuevo); malo[fk] = 999999
        if unicos.get(nombre) and unicos[nombre] in malo:
            malo[unicos[nombre]] = nuevo[unicos[nombre]] if isinstance(nuevo[unicos[nombre]], int) else nuevo[unicos[nombre]][:-1] + '8'
        paso('POST', r, malo)
        tipo_malo = dict(nuevo); tipo_malo[fk] = 'abc'
        paso('POST', r, tipo_malo)
    # ids de lo que se crea: se calculan repitiendo la lista del propio Python en tiempo de captura
    paso('GET', r + '?limit=200')
    def es_texto(k):
        p = S[crear]['properties'][k]
        return p.get('type') == 'string' and 'format' not in p or any(x.get('type') == 'string' and 'format' not in x and 'pattern' not in x for x in p.get('anyOf', []))
    texto_edit = next((k for k, v in nuevo.items() if isinstance(v, str) and es_texto(k) and k != unicos.get(nombre) and not k.startswith('id_') and k not in ('ruc', 'dni')), None)
    cambio = {texto_edit: 'Editado'} if texto_edit else {'estado_' + nombre[:3]: False}
    paso('PUT', f'{r}/{{id}}', cambio)
    paso('PUT', f'{r}/999999', cambio)
    paso('PUT', f'{r}/abc', cambio)
    paso('DELETE', f'{r}/{{id}}')
    paso('DELETE', f'{r}/{{id}}')
    paso('GET', f'{r}/{{id}}')
    paso('DELETE', f'{r}/abc')

# --- autenticación y salud ---
paso('GET', '/health')
paso('POST', '/api/v1/auth/register', {'username': 'ana', 'email': 'ana@lizar.pe', 'password': 'secreta1', 'nombre': 'Ana Quispe', 'rol': 'operador'})
paso('POST', '/api/v1/auth/register', {'username': 'ana', 'email': 'otra@lizar.pe', 'password': 'secreta1', 'nombre': 'Ana', 'rol': 'operador'})
paso('POST', '/api/v1/auth/register', {'username': 'luis', 'email': 'ana@lizar.pe', 'password': 'secreta1', 'nombre': 'Luis', 'rol': 'operador'})
paso('POST', '/api/v1/auth/register', {'username': 'luis', 'email': 'luis@lizar.pe', 'password': 'secreta1', 'nombre': 'Luis', 'rol': 'gerente'})
paso('POST', '/api/v1/auth/register', {'username': 'luis', 'email': 'no-es-correo', 'password': 'secreta1', 'nombre': 'Luis'})
paso('POST', '/api/v1/auth/register', {'username': 'luis'})
paso('POST', '/api/v1/auth/login', {'username': 'admin', 'password': 'admin123'})
paso('POST', '/api/v1/auth/login', {'username': 'admin', 'password': 'mala'})
paso('POST', '/api/v1/auth/login', {'username': 'nadie', 'password': 'x'})
paso('POST', '/api/v1/auth/login', {'username': 'ana', 'password': 'secreta1'})
paso('POST', '/api/v1/auth/login', {})
paso('GET', '/api/v1/auth/me')
paso('GET', '/api/v1/auth/me', token='@login-admin')
paso('GET', '/api/v1/auth/me', token='token-malo')
paso('GET', '/api/v1/auth/me', cabeceras={'Authorization': 'Basic abc'})

# --- reproducción: se registran las respuestas de Python ---
ultimo_id, tokens, respuestas = {}, {}, []
for p in pasos:
    ruta = p['ruta']
    if '{id}' in ruta:
        ruta = ruta.replace('{id}', str(ultimo_id[ruta.split('/{id}')[0]]))
    token = tokens.get(p['token'], p['token']) if p.get('token') else None
    estado, cuerpo = llamar(p['metodo'], ruta, p.get('cuerpo'), token, p.get('cabeceras'))
    if p.get('guarda') and estado != 201:
        print('ALTA FALLIDA', p['ruta'], estado, str(cuerpo)[:160], json.dumps(p.get('cuerpo'))[:300])
    if p.get('guarda') and estado == 201:
        ultimo_id[p['ruta']] = cuerpo[p['guarda']]
    elif p.get('guarda'):
        ultimo_id[p['ruta']] = 0
    if p['ruta'] == '/api/v1/auth/login' and estado == 200 and p.get('cuerpo', {}).get('username') == 'admin':
        tokens['@login-admin'] = cuerpo['access_token']
    respuestas.append({'estado': estado, 'cuerpo': cuerpo})

salida.mkdir(parents=True, exist_ok=True)
(salida / 'escenario.json').write_text(json.dumps(pasos, ensure_ascii=False, indent=1))
(salida / 'python.json').write_text(json.dumps(respuestas, ensure_ascii=False, indent=1))
print(len(pasos), 'pasos; estados:', sorted({r['estado'] for r in respuestas}))
```

`gastos-service/tools/contrato/reiniciar_referencia.sh`:

```bash
#!/bin/bash
# Deja la API de Python de referencia (puerto 8092) sobre una base recién cargada con db.sql.
# Uso: reiniciar_referencia.sh <raiz-del-proyecto> <carpeta-de-trabajo>
set -e
RAIZ="$1"; TRABAJO="$2"
BASE="$(sed -n 's/^DATABASE_URL=//p' "$RAIZ/gastos-service/.env" | tr -d "\"'")"
TMP=$(python3 -c "import sys,urllib.parse as u; p=u.urlsplit(sys.argv[1]); print(u.urlunsplit(p._replace(path='/lizar_contrato')))" "$BASE")
PIDS=$(lsof -nP -iTCP:8092 -sTCP:LISTEN -t || true); [ -n "$PIDS" ] && kill $PIDS && sleep 1
psql "$BASE" -qc "DROP DATABASE IF EXISTS lizar_contrato" && psql "$BASE" -qc "CREATE DATABASE lizar_contrato"
psql "$TMP" -q -v ON_ERROR_STOP=1 -f "$RAIZ/db.sql" > /dev/null
URLPY=$(python3 -c "import re,sys; print(re.sub(r'^postgres(ql)?://','postgresql+psycopg://',sys.argv[1]))" "$TMP")
cd "$RAIZ/backend"
SITIO="$(ls -d "$PWD"/.venv/lib/*/site-packages)"
PYTHONPATH="$SITIO" DATABASE_URL="$URLPY" APP_ENV=dev nohup /opt/homebrew/bin/python3.14 -m uvicorn app.main:app --port 8092 > "$TRABAJO/py-contrato.log" 2>&1 &
for i in $(seq 1 30); do python3 -c "import urllib.request;urllib.request.urlopen('http://localhost:8092/health',timeout=2)" 2>/dev/null && echo "referencia lista" && exit 0; sleep 1; done
echo "la referencia no arrancó"; exit 1
```

`gastos-service/tools/contrato/comparar_en_vivo.py`:

```python
"""Compara, solo con lecturas, la API de Python con la de Go sobre la misma base.
Uso: python3 comparar_en_vivo.py <url_python> <url_go>
Para cada recurso compara la lista (y su lista avanzada) y cada registro por su id."""
import json, sys, urllib.error, urllib.request

py, go = sys.argv[1].rstrip('/'), sys.argv[2].rstrip('/')
RECURSOS = {
    'ubigeos': 'codigo_ubigeo', 'rubros': 'id_rubro', 'marcas': 'id_marca', 'tipos-unidad': 'id_tipo_unidad',
    'subtipos-unidad': 'id_subtipo_unidad', 'tipos-gasto': 'id_tipo_gasto', 'categorias-producto': 'id_categoria',
    'direcciones': 'id_direccion', 'conductores': 'id_conductor', 'clientes': 'id_cliente', 'unidades': 'id_unidad',
    'asignaciones-conductor-unidad': 'id_conductor_unidad', 'viajes': 'id_viaje', 'guias-transportista': 'id_guia_transportista',
    'viajes-realizados': 'id_viaje_realizado', 'facturas': 'id_factura', 'presupuestos-viaje': 'id_presupuesto',
    'detalles-presupuesto': 'id_detalle_presupuesto', 'productos-transportados': 'id_producto_transportado',
}


def pedir(base, ruta):
    try:
        r = urllib.request.urlopen(base + ruta, timeout=20)
        return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read().decode() or 'null')


comparados = diferencias = 0
for nombre, pk in RECURSOS.items():
    rutas = [f'/api/v1/{nombre}?limit=200', f'/api/v1/advanced/{nombre}?limit=200&order_dir=desc', f'/api/v1/{nombre}/999999']
    estado, lista = pedir(py, rutas[0])
    rutas += [f'/api/v1/{nombre}/{x[pk]}' for x in lista[:50]]
    for ruta in rutas:
        a, b = pedir(py, ruta), pedir(go, ruta)
        comparados += 1
        if a != b:
            diferencias += 1
            print('DIFERENCIA', ruta, '\n  Python:', json.dumps(a, ensure_ascii=False)[:300], '\n  Go:    ', json.dumps(b, ensure_ascii=False)[:300])
print(f'{comparados} consultas comparadas, {diferencias} con diferencias')
sys.exit(1 if diferencias else 0)
```

Luego `chmod +x gastos-service/tools/contrato/reiniciar_referencia.sh`.

- [ ] **Step 2: Dejar el Python de referencia sobre una base recién cargada**

Run (desde la raíz del proyecto; `$SCRATCH` es la carpeta temporal de la sesión):
```bash
gastos-service/tools/contrato/reiniciar_referencia.sh "$PWD" "$SCRATCH"
```
Expected: `referencia lista`. Crea la base `lizar_contrato` con `db.sql` y arranca el Python en el puerto 8092 sobre ella. El Python de producción en el 8000 no se toca.

- [ ] **Step 3: Capturar el contrato (una sola vez por base nueva)**

Run:
```bash
python3 gastos-service/tools/contrato/capturar_contrato.py http://localhost:8092 gastos-service/internal/crud/contrato
```
Expected: `552 pasos; estados: [200, 201, 401, 404, 409, 422, 500]` y cuatro líneas `ALTA FALLIDA` (guías, presupuestos, viajes realizados y facturas devuelven 500 al crear con el único padre ya ocupado: es lo que hace Python). Si la captura se interrumpe o se repite, vuelve al Step 2: una base usada desplaza los ids de las secuencias.

- [ ] **Step 4: Generar las definiciones**

Run:
```bash
python3 gastos-service/tools/contrato/generar_recursos.py "$PWD/backend" | gofmt > gastos-service/internal/crud/recursos.go
grep -c "Ruta:" gastos-service/internal/crud/recursos.go && grep -c TDecimal gastos-service/internal/crud/recursos.go
```
Expected: `19` y `8` (los ocho campos decimales: `peso_tara`, `peso_maximo`, `peso_total`, `flete`, dos `monto_total`, `monto_estimado` y `peso`). El archivo no compila hasta la Tarea 2 (usa los tipos `Recurso`, `Campo`, `Unico` y las constantes `T…`).

- [ ] **Step 5: Detener la referencia y borrar su base**

Run:
```bash
kill $(lsof -nP -iTCP:8092 -sTCP:LISTEN -t)
psql "$(sed -n 's/^DATABASE_URL=//p' gastos-service/.env | tr -d "\"'")" -qc "DROP DATABASE lizar_contrato"
ls -la gastos-service/internal/crud/contrato/
```
Expected: ambos JSON presentes (~440 KB en total).

- [ ] **Step 6: Checkpoint (sin commit)**

Nada que compilar todavía. Anota en el ledger el número de pasos capturados.

---

### Task 2: Tipos y validación del motor (Go)

**Files:**
- Create: `gastos-service/internal/crud/tipos.go`, `validacion.go`
- Test: `gastos-service/internal/crud/validacion_test.go`, `tipos_test.go`

**Interfaces:**
- Consumes: `recursos.go` (Tarea 1).
- Produces: `crud.Tipo` (`TEntero`, `TTexto`, `TBool`, `TDecimal`, `TFechaHora`, `TFecha`); `Campo{Nombre, Tipo, Obligatorio, Nulo, SoloLectura, Email, Defecto, Max, Min}`; `Unico{Campo, Mensaje}`; `Recurso{Ruta, Tabla, PK, PKDado, NoEncontrado, Unico, Busqueda, Campos}` con `Campo(nombre)` y `TipoPK()`; `Fila` (JSON en orden y sin escapar HTML) con `Valor(nombre)`; `FormatoFechaHora(time.Time) string`; `ErrorCampo{Tipo, Loc, Msg, Input, Ctx}`; `Coercionar(c Campo, v any) (any, *ErrorCampo)`; `ValidarCreacion(r *Recurso, cuerpo map[string]any) (map[string]any, []ErrorCampo)`; `ValidarCambios(r, cuerpo)`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`gastos-service/internal/crud/validacion_test.go`:

```go
package crud

import (
	"testing"
	"time"
)

func TestCoercionarEntero(t *testing.T) {
	c := Campo{Nombre: "n", Tipo: TEntero}
	for _, v := range []any{float64(12), "12", " 12 ", float64(12.0)} {
		if got, e := Coercionar(c, v); e != nil || got != int64(12) {
			t.Errorf("Coercionar(%v) = %v, %v; se esperaba 12", v, got, e)
		}
	}
	casos := map[string]any{"int_parsing": "abc", "int_from_float": 1.5, "int_type": true}
	for tipo, v := range casos {
		if _, e := Coercionar(c, v); e == nil || e.Tipo != tipo {
			t.Errorf("Coercionar(%v): error %v, se esperaba %s", v, e, tipo)
		}
	}
	if _, e := Coercionar(c, nil); e == nil || e.Tipo != "int_type" {
		t.Errorf("null en un campo no nulo: %v", e)
	}
	if got, e := Coercionar(Campo{Tipo: TEntero, Nulo: true}, nil); e != nil || got != nil {
		t.Errorf("null en un campo nulo: %v %v", got, e)
	}
}

func TestCoercionarTextoConLargos(t *testing.T) {
	c := Campo{Tipo: TTexto, Min: 3, Max: 5}
	if got, e := Coercionar(c, "abcde"); e != nil || got != "abcde" {
		t.Fatalf("abcde: %v %v", got, e)
	}
	if _, e := Coercionar(c, "abcdef"); e == nil || e.Tipo != "string_too_long" || e.Msg != "String should have at most 5 characters" || e.Ctx["max_length"] != 5 {
		t.Errorf("largo máximo: %+v", e)
	}
	if _, e := Coercionar(c, "ab"); e == nil || e.Tipo != "string_too_short" {
		t.Errorf("largo mínimo: %+v", e)
	}
	if _, e := Coercionar(c, float64(12345)); e == nil || e.Tipo != "string_type" {
		t.Errorf("un número no es texto: %+v", e)
	}
	// Se cuentan caracteres, no bytes: «ñandú» son 5 caracteres
	if _, e := Coercionar(Campo{Tipo: TTexto, Max: 5}, "ñandú"); e != nil {
		t.Errorf("ñandú debía caber en 5: %+v", e)
	}
	if _, e := Coercionar(Campo{Tipo: TTexto, Email: true}, "no-es-correo"); e == nil {
		t.Error("un correo sin arroba ni dominio debía rechazarse")
	}
}

func TestCoercionarBoolDecimalYFechas(t *testing.T) {
	if got, e := Coercionar(Campo{Tipo: TBool}, "true"); e != nil || got != true {
		t.Errorf("bool desde texto: %v %v", got, e)
	}
	if _, e := Coercionar(Campo{Tipo: TBool}, "quizás"); e == nil || e.Tipo != "bool_parsing" {
		t.Errorf("bool inválido: %v", e)
	}
	if got, e := Coercionar(Campo{Tipo: TDecimal}, float64(8000.5)); e != nil || got != "8000.5" {
		t.Errorf("decimal desde número: %v %v", got, e)
	}
	if got, e := Coercionar(Campo{Tipo: TDecimal}, "8000.50"); e != nil || got != "8000.50" {
		t.Errorf("decimal desde texto: %v %v", got, e)
	}
	if _, e := Coercionar(Campo{Tipo: TDecimal}, "mucho"); e == nil || e.Tipo != "decimal_parsing" {
		t.Errorf("decimal inválido: %v", e)
	}
	esperado := time.Date(2026, 9, 29, 8, 30, 0, 0, time.UTC)
	for _, v := range []string{"2026-09-29T08:30:00", "2026-09-29 08:30:00", "2026-09-29T08:30:00Z"} {
		if got, e := Coercionar(Campo{Tipo: TFechaHora}, v); e != nil || !got.(time.Time).Equal(esperado) {
			t.Errorf("fecha-hora %q: %v %v", v, got, e)
		}
	}
	if got, e := Coercionar(Campo{Tipo: TFecha}, "2026-09-29"); e != nil || !got.(time.Time).Equal(time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("fecha: %v %v", got, e)
	}
	if _, e := Coercionar(Campo{Tipo: TFecha}, "29/09/2026"); e == nil {
		t.Error("una fecha dd/mm/aaaa debía rechazarse")
	}
}

func TestValidarCreacionAplicaDefectosYJuntaLosErrores(t *testing.T) {
	r := &Recurso{Campos: []Campo{
		{Nombre: "nombre", Tipo: TTexto, Obligatorio: true, Max: 10},
		{Nombre: "detalle", Tipo: TTexto, Nulo: true},
		{Nombre: "estado", Tipo: TBool, Defecto: true},
		{Nombre: "id", Tipo: TEntero, SoloLectura: true},
	}}
	valores, errs := ValidarCreacion(r, map[string]any{"nombre": "Agro"})
	if len(errs) != 0 || valores["estado"] != true || valores["detalle"] != nil {
		t.Fatalf("valores = %v, errores = %v", valores, errs)
	}
	if _, ok := valores["id"]; ok {
		t.Error("la clave generada no se envía a la base")
	}
	_, errs = ValidarCreacion(r, map[string]any{"detalle": 5.0, "estado": "talvez"})
	if len(errs) != 3 || errs[0].Tipo != "missing" || errs[0].Loc[1] != "nombre" || errs[1].Tipo != "string_type" || errs[2].Tipo != "bool_parsing" {
		t.Fatalf("errores = %+v", errs)
	}
}

func TestValidarCambiosIgnoraLoDesconocidoYLaClave(t *testing.T) {
	r := &Recurso{PK: "id", Campos: []Campo{{Nombre: "nombre", Tipo: TTexto}, {Nombre: "id", Tipo: TEntero}}}
	valores, errs := ValidarCambios(r, map[string]any{"nombre": "Nuevo", "id": 99.0, "inventado": "x"})
	if len(errs) != 0 || len(valores) != 1 || valores["nombre"] != "Nuevo" {
		t.Fatalf("valores = %v, errores = %v", valores, errs)
	}
}
```

`gastos-service/internal/crud/tipos_test.go`:

```go
package crud

import (
	"testing"
	"time"
)

func TestFilaSeEscribeEnOrdenYSinEscaparHTML(t *testing.T) {
	r := &Recurso{Campos: []Campo{
		{Nombre: "nombre", Tipo: TTexto}, {Nombre: "monto", Tipo: TDecimal}, {Nombre: "creado", Tipo: TFechaHora},
		{Nombre: "dia", Tipo: TFecha}, {Nombre: "n", Tipo: TEntero}, {Nombre: "nota", Tipo: TTexto},
	}}
	f := Fila{recurso: r, valores: []any{
		"A&B <C>", "12.50", time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC), time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC), int32(7), nil,
	}}
	directo, err := f.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"nombre":"A&B <C>","monto":"12.50","creado":"2026-09-29T08:00:00","dia":"2026-09-29","n":7,"nota":null}`
	if string(directo) != esperado {
		t.Fatalf("MarshalJSON = %s\nesperado     %s", directo, esperado)
	}
}

func TestFormatoFechaHoraComoIsoformat(t *testing.T) {
	if got := FormatoFechaHora(time.Date(2026, 9, 29, 8, 5, 3, 0, time.UTC)); got != "2026-09-29T08:05:03" {
		t.Errorf("sin microsegundos: %s", got)
	}
	if got := FormatoFechaHora(time.Date(2026, 9, 29, 8, 5, 3, 120000, time.UTC)); got != "2026-09-29T08:05:03.000120" {
		t.Errorf("con microsegundos: %s", got)
	}
}

func TestLasDefinicionesSonCoherentes(t *testing.T) {
	if len(Recursos) != 19 {
		t.Fatalf("se esperaban 19 recursos, hay %d", len(Recursos))
	}
	rutas := map[string]bool{}
	for _, r := range Recursos {
		if rutas[r.Ruta] {
			t.Errorf("ruta repetida: %s", r.Ruta)
		}
		rutas[r.Ruta] = true
		if c, ok := r.Campo(r.PK); !ok || c.Tipo != TEntero {
			t.Errorf("%s: la clave %q debe ser un campo entero", r.Ruta, r.PK)
		}
		for _, b := range r.Busqueda {
			if _, ok := r.Campo(b); !ok {
				t.Errorf("%s: el campo de búsqueda %q no existe", r.Ruta, b)
			}
		}
		if r.Unico != nil {
			if _, ok := r.Campo(r.Unico.Campo); !ok {
				t.Errorf("%s: el campo único %q no existe", r.Ruta, r.Unico.Campo)
			}
		}
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd gastos-service && go test ./internal/crud/ -count=1 2>&1 | head -6`
Expected: FAIL de compilación: `undefined: Recurso`, `undefined: Campo`, `undefined: Coercionar`…

- [ ] **Step 3: Implementar**

`gastos-service/internal/crud/tipos.go`:

```go
// Package crud es el motor genérico de los recursos de /api/v1 (antes en la API de Python):
// una definición por recurso, y listar, obtener, crear, actualizar, borrar y la lista
// avanzada sobre ella, con las mismas respuestas y errores que daba FastAPI.
package crud

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"
)

// Tipo es el tipo de una columna tal como lo expone la API.
type Tipo int

const (
	TEntero    Tipo = iota
	TTexto          // cadena
	TBool           // verdadero o falso
	TDecimal        // NUMERIC: se lee y se escribe como texto para no perder decimales ("12.50")
	TFechaHora      // TIMESTAMP sin zona: 2026-09-29T08:00:00
	TFecha          // DATE: 2026-09-29
)

// Campo es una columna de un recurso, en el orden en que sale en el JSON.
type Campo struct {
	Nombre      string
	Tipo        Tipo
	Obligatorio bool // obligatorio al crear
	Nulo        bool // admite null
	SoloLectura bool // solo sale en las respuestas (la clave generada por la base)
	Email       bool
	Defecto     any // valor al crear si no viene; nil = NULL
	Max, Min    int // largo del texto; 0 = sin límite
}

// Unico es la regla de unicidad que responde 409 al crear.
type Unico struct{ Campo, Mensaje string }

// Recurso describe una tabla expuesta en /api/v1/<Ruta>.
type Recurso struct {
	Ruta         string
	Tabla        string
	PK           string
	PKDado       bool // la clave la envía quien crea (ubigeo); si no, la genera la base
	NoEncontrado string
	Unico        *Unico
	Busqueda     []string
	Campos       []Campo
}

// Campo devuelve la columna con ese nombre.
func (r *Recurso) Campo(nombre string) (Campo, bool) {
	for _, c := range r.Campos {
		if c.Nombre == nombre {
			return c, true
		}
	}
	return Campo{}, false
}

// TipoPK es el tipo de la clave primaria.
func (r *Recurso) TipoPK() Tipo {
	c, _ := r.Campo(r.PK)
	return c.Tipo
}

// Fila es un registro leído, con sus valores en el orden de Recurso.Campos.
type Fila struct {
	recurso *Recurso
	valores []any
}

// MarshalJSON escribe el objeto con las claves en orden y sin escapar <, > ni &, como FastAPI.
func (f Fila) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, c := range f.recurso.Campos {
		if i > 0 {
			b.WriteByte(',')
		}
		nombre, _ := jsonSinEscape(c.Nombre)
		b.Write(nombre)
		b.WriteByte(':')
		v, err := jsonSinEscape(valorSalida(c.Tipo, f.valores[i]))
		if err != nil {
			return nil, err
		}
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Valor devuelve el valor de una columna, ya con el formato de salida.
func (f Fila) Valor(nombre string) any {
	for i, c := range f.recurso.Campos {
		if c.Nombre == nombre {
			return valorSalida(c.Tipo, f.valores[i])
		}
	}
	return nil
}

func jsonSinEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// valorSalida da a cada valor leído de la base el formato que tenía la API de Python.
func valorSalida(t Tipo, v any) any {
	if v == nil {
		return nil
	}
	switch t {
	case TFechaHora:
		if ts, ok := v.(time.Time); ok {
			return FormatoFechaHora(ts)
		}
	case TFecha:
		if ts, ok := v.(time.Time); ok {
			return ts.Format("2006-01-02")
		}
	case TEntero:
		switch n := v.(type) {
		case int16:
			return int64(n)
		case int32:
			return int64(n)
		}
	}
	return v
}

// FormatoFechaHora escribe un instante como datetime.isoformat() de Python: sin zona y con
// microsegundos solo si los hay.
func FormatoFechaHora(t time.Time) string {
	s := t.Format("2006-01-02T15:04:05")
	if us := t.Nanosecond() / 1000; us != 0 {
		s += "." + padIzquierda(strconv.Itoa(us), 6)
	}
	return s
}

func padIzquierda(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}
```

`gastos-service/internal/crud/validacion.go`:

```go
package crud

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ErrorCampo es un error de validación con la forma de los de FastAPI: type, loc, msg, input y ctx.
type ErrorCampo struct {
	Tipo  string         `json:"type"`
	Loc   []any          `json:"loc"`
	Msg   string         `json:"msg"`
	Input any            `json:"input"`
	Ctx   map[string]any `json:"ctx,omitempty"`
}

var (
	reEmail   = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	reDecimal = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)$`)
)

// Coercionar convierte un valor JSON al tipo del campo con la misma tolerancia que Pydantic en
// modo normal (por ejemplo, "12" es un entero válido). Devuelve el valor listo para la base o el
// error. Los decimales viajan como texto y los instantes como time.Time.
func Coercionar(c Campo, v any) (any, *ErrorCampo) {
	if v == nil {
		if c.Nulo {
			return nil, nil
		}
		return nil, errTipo(c.Tipo, v)
	}
	switch c.Tipo {
	case TEntero:
		return coercionarEntero(v)
	case TTexto:
		s, ok := v.(string)
		if !ok {
			return nil, &ErrorCampo{Tipo: "string_type", Msg: "Input should be a valid string", Input: v}
		}
		n := utf8.RuneCountInString(s)
		if c.Max > 0 && n > c.Max {
			return nil, &ErrorCampo{Tipo: "string_too_long", Msg: fmt.Sprintf("String should have at most %d characters", c.Max), Input: v, Ctx: map[string]any{"max_length": c.Max}}
		}
		if c.Min > 0 && n < c.Min {
			return nil, &ErrorCampo{Tipo: "string_too_short", Msg: fmt.Sprintf("String should have at least %d characters", c.Min), Input: v, Ctx: map[string]any{"min_length": c.Min}}
		}
		if c.Email && !reEmail.MatchString(s) {
			return nil, &ErrorCampo{Tipo: "value_error", Msg: "value is not a valid email address", Input: v}
		}
		return s, nil
	case TBool:
		return coercionarBool(v)
	case TDecimal:
		return coercionarDecimal(v)
	case TFechaHora:
		return coercionarFechaHora(v)
	case TFecha:
		return coercionarFecha(v)
	}
	return nil, errTipo(c.Tipo, v)
}

func errTipo(t Tipo, v any) *ErrorCampo {
	switch t {
	case TEntero:
		return &ErrorCampo{Tipo: "int_type", Msg: "Input should be a valid integer", Input: v}
	case TTexto:
		return &ErrorCampo{Tipo: "string_type", Msg: "Input should be a valid string", Input: v}
	case TBool:
		return &ErrorCampo{Tipo: "bool_type", Msg: "Input should be a valid boolean", Input: v}
	case TDecimal:
		return &ErrorCampo{Tipo: "decimal_type", Msg: "Decimal input should be an integer, float, string or Decimal object", Input: v}
	case TFechaHora:
		return &ErrorCampo{Tipo: "datetime_type", Msg: "Input should be a valid datetime", Input: v}
	default:
		return &ErrorCampo{Tipo: "date_type", Msg: "Input should be a valid date", Input: v}
	}
}

func coercionarEntero(v any) (any, *ErrorCampo) {
	switch n := v.(type) {
	case float64:
		if n != math.Trunc(n) || math.Abs(n) > 9e15 {
			return nil, &ErrorCampo{Tipo: "int_from_float", Msg: "Input should be a valid integer, got a number with a fractional part", Input: v}
		}
		return int64(n), nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return nil, &ErrorCampo{Tipo: "int_parsing", Msg: "Input should be a valid integer, unable to parse string as an integer", Input: v}
		}
		return i, nil
	}
	return nil, errTipo(TEntero, v)
}

func coercionarBool(v any) (any, *ErrorCampo) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case float64:
		if b == 0 || b == 1 {
			return b == 1, nil
		}
	case string:
		switch strings.ToLower(strings.TrimSpace(b)) {
		case "true", "t", "yes", "y", "on", "1":
			return true, nil
		case "false", "f", "no", "n", "off", "0":
			return false, nil
		}
		return nil, &ErrorCampo{Tipo: "bool_parsing", Msg: "Input should be a valid boolean, unable to interpret input", Input: v}
	default:
		return nil, errTipo(TBool, v)
	}
	return nil, &ErrorCampo{Tipo: "bool_parsing", Msg: "Input should be a valid boolean, unable to interpret input", Input: v}
}

func coercionarDecimal(v any) (any, *ErrorCampo) {
	switch d := v.(type) {
	case float64:
		return strconv.FormatFloat(d, 'f', -1, 64), nil
	case string:
		s := strings.TrimSpace(d)
		if !reDecimal.MatchString(s) {
			return nil, &ErrorCampo{Tipo: "decimal_parsing", Msg: "Input should be a valid decimal", Input: v}
		}
		return s, nil
	}
	return nil, errTipo(TDecimal, v)
}

var formatosFechaHora = []string{
	"2006-01-02T15:04:05.999999999Z07:00", "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999", "2006-01-02T15:04Z07:00", "2006-01-02T15:04", "2006-01-02",
}

func coercionarFechaHora(v any) (any, *ErrorCampo) {
	s, ok := v.(string)
	if !ok {
		return nil, errTipo(TFechaHora, v)
	}
	for _, f := range formatosFechaHora {
		if t, err := time.Parse(f, strings.TrimSpace(s)); err == nil {
			// La columna es TIMESTAMP sin zona: se guarda la hora tal como se escribió
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
		}
	}
	return nil, &ErrorCampo{Tipo: "datetime_from_date_parsing", Msg: "Input should be a valid datetime or date, invalid character in year", Input: v}
}

func coercionarFecha(v any) (any, *ErrorCampo) {
	s, ok := v.(string)
	if !ok {
		return nil, errTipo(TFecha, v)
	}
	for _, f := range []string{"2006-01-02", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(f, strings.TrimSpace(s)); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		}
	}
	return nil, &ErrorCampo{Tipo: "date_from_datetime_parsing", Msg: "Input should be a valid date or datetime, invalid character in year", Input: v}
}

// ValidarCreacion valida el cuerpo de un POST contra el recurso. Devuelve los valores en el orden
// de las columnas de creación (con los defectos aplicados) o los errores, todos juntos.
func ValidarCreacion(r *Recurso, cuerpo map[string]any) (map[string]any, []ErrorCampo) {
	valores := map[string]any{}
	var errs []ErrorCampo
	for _, c := range r.Campos {
		if c.SoloLectura {
			continue
		}
		v, presente := cuerpo[c.Nombre]
		if !presente {
			if c.Obligatorio {
				errs = append(errs, ErrorCampo{Tipo: "missing", Loc: []any{"body", c.Nombre}, Msg: "Field required", Input: cuerpo})
				continue
			}
			valores[c.Nombre] = c.Defecto
			continue
		}
		val, e := Coercionar(c, v)
		if e != nil {
			e.Loc = []any{"body", c.Nombre}
			errs = append(errs, *e)
			continue
		}
		valores[c.Nombre] = val
	}
	return valores, errs
}

// ValidarCambios valida los campos de un PUT: solo se aplican las columnas conocidas (la clave
// primaria no se cambia) y las demás claves se ignoran, como hacía la API de Python.
func ValidarCambios(r *Recurso, cuerpo map[string]any) (map[string]any, []ErrorCampo) {
	valores := map[string]any{}
	var errs []ErrorCampo
	for _, c := range r.Campos {
		v, presente := cuerpo[c.Nombre]
		if !presente || c.Nombre == r.PK {
			continue
		}
		val, e := Coercionar(c, v)
		if e != nil {
			e.Loc = []any{"body", c.Nombre}
			errs = append(errs, *e)
			continue
		}
		valores[c.Nombre] = val
	}
	return valores, errs
}
```

- [ ] **Step 4: Ejecutar y verificar que pasan**

Run: `cd gastos-service && go test ./internal/crud/ -count=1 -v 2>&1 | grep -E "^(--- |ok|FAIL)"`
Expected: 8 `--- PASS` y `ok`.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . && go vet ./internal/crud/`
Expected: sin salida de `gofmt -l`; vet limpio.

---

### Task 3: Motor SQL, rutas, esquema de pruebas completo y cableado (Go)

**Files:**
- Create: `gastos-service/internal/crud/motor.go`, `http.go`
- Modify: `gastos-service/internal/pruebas/esquema.go` (reemplazo completo)
- Modify: `gastos-service/internal/config/config.go`, `internal/handlers/handlers.go`, `internal/handlers/middleware.go`
- Test: `gastos-service/internal/handlers/maestros_integration_test.go`

**Interfaces:**
- Consumes: Tarea 2.
- Produces:
  - `crud.Motor{DB Ejecutor}` con `Listar`, `Obtener`, `ExisteUnico`, `Crear`, `Actualizar`, `Borrar`, `Avanzada(ctx, r, ParamsAvanzada)`.
  - `crud.Registrar(mux, m *Motor, prefijo string, recursos []Recurso)`, `crud.EscribirJSON(w, estado, v)` y `crud.Detalle(w, estado, mensaje)`.
  - `pruebas.NuevoEsquemaCompleto(t) *pgxpool.Pool` (esquema temporal con `db.sql` y las migraciones) y `pruebas.NuevoEsquema(t)` como antes (las tablas mínimas).
  - `config.Config.JWTSecret` y `config.Config.AppEnv`; `GET /health` → `{"status":"ok","env":"<APP_ENV>"}`; CORS con `GET, POST, PUT, DELETE, OPTIONS`.

- [ ] **Step 1: Esquema de pruebas completo (reemplazo)**

`gastos-service/internal/pruebas/esquema.go` pasa a ser:

```go
// Package pruebas prepara una base de datos aislada para las pruebas de integración.
// Solo lo importan archivos _test.go.
package pruebas

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gastos-service/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tablasMinimas imita, con solo las columnas que usa el servicio, las tablas del
// sistema principal a las que apuntan gasto y los reportes.
const tablasMinimas = `
	CREATE TABLE cliente (id_cliente SERIAL PRIMARY KEY, razon_social VARCHAR(150));
	CREATE TABLE unidad (id_unidad SERIAL PRIMARY KEY, placa VARCHAR(20));
	CREATE TABLE conductor (
		id_conductor SERIAL PRIMARY KEY,
		nombre1 VARCHAR(50), apellido1 VARCHAR(50), apellido2 VARCHAR(50)
	);
	CREATE TABLE conductorunidad (id_conductor_unidad SERIAL PRIMARY KEY, id_conductor INTEGER REFERENCES conductor (id_conductor));
	CREATE TABLE programacionviaje (
		id_viaje SERIAL PRIMARY KEY,
		id_tracto INTEGER REFERENCES unidad (id_unidad),
		id_conductor_unidad INTEGER REFERENCES conductorunidad (id_conductor_unidad)
	);`

// NuevoEsquema crea un esquema temporal con las tablas mínimas y las migraciones
// del servicio, y lo borra al terminar la prueba. Sin TEST_DATABASE_URL, la prueba se omite.
func NuevoEsquema(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return nuevoEsquema(t, tablasMinimas)
}

// NuevoEsquemaCompleto es como NuevoEsquema pero con el esquema y los datos reales de db.sql
// (todas las tablas del sistema, no solo las mínimas), para las pruebas de la API de maestros.
func NuevoEsquemaCompleto(t testing.TB) *pgxpool.Pool {
	t.Helper()
	sql, err := os.ReadFile(filepath.Join(raizModulo(t), "..", "db.sql"))
	if err != nil {
		t.Fatal(err)
	}
	return nuevoEsquema(t, string(sql))
}

func nuevoEsquema(t testing.TB, ddl string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: se omiten las pruebas contra PostgreSQL")
	}
	ctx := context.Background()
	esquema := fmt.Sprintf("prueba_%d", time.Now().UnixNano())

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+esquema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+esquema+" CASCADE")
		admin.Close()
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = esquema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	if err := db.RunMigrations(pool, directorioMigraciones(t)); err != nil {
		t.Fatal(err)
	}
	return pool
}

// directorioMigraciones ubica gastos-service/migrations.
func directorioMigraciones(t testing.TB) string {
	return filepath.Join(raizModulo(t), "migrations")
}

// raizModulo sube desde el directorio de la prueba hasta encontrar go.mod; así funciona desde
// cualquier paquete y también con go test -trimpath.
func raizModulo(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			t.Fatal("no se encontró go.mod subiendo desde el directorio de la prueba")
		}
		dir = padre
	}
}
```

(Se agregaron `NuevoEsquemaCompleto`, `raizModulo` y `directorioMigraciones(t)`; `NuevoEsquema` conserva su comportamiento. Ya no usa `runtime.Caller`, que con `-trimpath` devolvía rutas relativas.)

Run: `cd gastos-service && TEST_DATABASE_URL=… go test ./internal/handlers/ ./internal/reportes/ -count=1`
Expected: ok (las pruebas existentes de checklist y reportes siguen en verde).

- [ ] **Step 2: Escribir las pruebas que fallan**

`gastos-service/internal/handlers/maestros_integration_test.go`:

```go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gastos-service/internal/config"
	"gastos-service/internal/handlers"
	"gastos-service/internal/pruebas"
)

func servidorCompleto(t *testing.T) http.Handler {
	t.Helper()
	pool := pruebas.NuevoEsquemaCompleto(t)
	mux := http.NewServeMux()
	handlers.New(pool, config.Config{UploadsDir: t.TempDir(), AppEnv: "dev", JWTSecret: "prueba"}).Register(mux)
	return handlers.CORS(mux)
}

func llamar(t *testing.T, h http.Handler, metodo, ruta string, cuerpo any) (int, map[string]any) {
	t.Helper()
	var b []byte
	if cuerpo != nil {
		b, _ = json.Marshal(cuerpo)
	}
	req := httptest.NewRequest(metodo, ruta, bytes.NewReader(b))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var obj map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &obj)
	return rec.Code, obj
}

// Guías, presupuestos, viajes realizados y facturas cuelgan de un padre que solo admite un hijo:
// el contrato las cubre con su error; aquí se prueba el alta que sí funciona, creando el padre.
func TestAltaDeRecursosConPadreUnico(t *testing.T) {
	h := servidorCompleto(t)
	_, viaje := llamar(t, h, "POST", "/api/v1/viajes", map[string]any{
		"id_tracto": 2, "id_carreta": 1, "id_conductor_unidad": 1, "id_direccion_origen": 1, "id_direccion_destino": 1,
		"fecha_programado": "2026-10-01T08:00:00", "id_cliente": 1,
	})
	idViaje := viaje["id_viaje"]
	if idViaje == nil {
		t.Fatalf("no se creó el viaje: %v", viaje)
	}

	estado, presupuesto := llamar(t, h, "POST", "/api/v1/presupuestos-viaje", map[string]any{"id_viaje": idViaje, "monto_total": 4000, "fecha_registro": "2026-10-01T00:00:00"})
	if estado != 201 || presupuesto["monto_total"] != "4000.00" {
		t.Fatalf("presupuesto: %d %v", estado, presupuesto)
	}
	estado, guia := llamar(t, h, "POST", "/api/v1/guias-transportista", map[string]any{"numero_guia": "GT-NUEVA-1", "id_viaje": idViaje, "peso_total": "11000.5"})
	if estado != 201 || guia["peso_total"] != "11000.50" || guia["guias_remitente"] != nil {
		t.Fatalf("guía: %d %v", estado, guia)
	}
	estado, _ = llamar(t, h, "POST", "/api/v1/guias-transportista", map[string]any{"numero_guia": "GT-NUEVA-1", "id_viaje": idViaje, "peso_total": 1})
	if estado != 409 {
		t.Fatalf("una guía con el mismo número debía dar 409, dio %d", estado)
	}
	estado, realizado := llamar(t, h, "POST", "/api/v1/viajes-realizados", map[string]any{"id_guia_transportista": guia["id_guia_transportista"], "flete": 8000, "fecha_llegada": "2026-10-02T10:00:00"})
	if estado != 201 || realizado["flete"] != "8000.00" {
		t.Fatalf("viaje realizado: %d %v", estado, realizado)
	}
	estado, factura := llamar(t, h, "POST", "/api/v1/facturas", map[string]any{"numero_factura": "F001-1", "id_viaje_realizado": realizado["id_viaje_realizado"], "monto_total": 8200})
	if estado != 201 || factura["estado_factura"] != true {
		t.Fatalf("factura: %d %v", estado, factura)
	}
	// Un viaje con presupuesto, guía y factura no se puede borrar: viola una llave foránea (500 en Python también)
	estado, _ = llamar(t, h, "DELETE", fmt.Sprintf("/api/v1/viajes/%v", idViaje), nil)
	if estado != 500 {
		t.Fatalf("borrar un viaje con hijos daba 500 en Python; dio %d", estado)
	}
}

func TestLaBusquedaNoEsUnaInyeccionSQL(t *testing.T) {
	h := servidorCompleto(t)
	req := httptest.NewRequest("GET", "/api/v1/advanced/clientes?search=%27%3B+DROP+TABLE+cliente%3B+--&filters=ruc%3A%27+OR+1%3D1&order_by=ruc%3B+DROP+TABLE+cliente", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != "[]\n" {
		t.Fatalf("código %d, cuerpo %q", rec.Code, rec.Body.String())
	}
	if estado, _ := llamar(t, h, "GET", "/api/v1/clientes/1", nil); estado != 200 {
		t.Fatalf("la tabla cliente debía seguir existiendo, dio %d", estado)
	}
}

func TestCORSPermitePUTyDELETEYLaSaludIncluyeElEntorno(t *testing.T) {
	h := servidorCompleto(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/rubros/1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	metodos := rec.Header().Get("Access-Control-Allow-Methods")
	if rec.Code != http.StatusNoContent || metodos != "GET, POST, PUT, DELETE, OPTIONS" {
		t.Fatalf("preflight: %d %q", rec.Code, metodos)
	}
	estado, salud := llamar(t, h, "GET", "/health", nil)
	if estado != 200 || salud["status"] != "ok" || salud["env"] != "dev" {
		t.Fatalf("health: %d %v", estado, salud)
	}
}
```

- [ ] **Step 3: Ejecutar y verificar que fallan**

Run: `cd gastos-service && TEST_DATABASE_URL=… go test ./internal/handlers/ -count=1 -run 'Alta|Busqueda|CORS' 2>&1 | head -8`
Expected: FAIL: `unknown field AppEnv in struct literal` (la configuración aún no existe).

- [ ] **Step 4: Configuración, salud y CORS**

En `gastos-service/internal/config/config.go`, en `type Config`, después de `UploadsDir  string`, agrega:

```go
	// JWTSecret firma los tokens de /api/v1/auth. Sin definirla se usa la clave que tenía la API de
	// Python, para que los tokens ya emitidos sigan valiendo; en producción debe definirse.
	JWTSecret string
	AppEnv    string
```

y en `Load()`, después de `UploadsDir:  getEnv("UPLOADS_DIR", "./uploads"),`:

```go
		JWTSecret:   getEnv("JWT_SECRET", "lizar-srl-fleet-manager-secret-2025"),
		AppEnv:      getEnv("APP_ENV", "dev"),
```

En `gastos-service/internal/handlers/handlers.go` reemplaza el cuerpo de `health` por:

```go
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "env": h.Cfg.AppEnv})
}
```

(El campo `service` desaparece: ningún cliente lo usa; el frontend solo lee `env`.)

En `gastos-service/internal/handlers/middleware.go` cambia `"GET, POST, OPTIONS"` por `"GET, POST, PUT, DELETE, OPTIONS"`.

- [ ] **Step 5: Implementar el motor y las rutas**

`gastos-service/internal/crud/motor.go`:

```go
package crud

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Ejecutor es lo que el motor necesita de la base (lo cumple *pgxpool.Pool).
type Ejecutor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Motor ejecuta las operaciones de los recursos. Los nombres de tabla y de columna salen siempre
// de las definiciones (constantes), nunca de lo que envía quien llama; los valores van como parámetros.
type Motor struct{ DB Ejecutor }

// ParamsAvanzada son los parámetros de GET /advanced/<recurso>.
type ParamsAvanzada struct {
	Limit, Offset int
	Search        string
	SearchFields  string
	OrderBy       string
	OrderDir      string
	Filters       string
}

func ident(s string) string { return `"` + s + `"` }

// escalaDecimal es la escala con que la API de Python entregaba los decimales: todas las columnas
// NUMERIC de los recursos eran Numeric(10, 2) ("8000.00", aunque la base guarde 8000).
const escalaDecimal = 2

// columnas arma la lista del SELECT; los decimales salen como texto con su escala para no perder el formato.
func columnas(r *Recurso) string {
	partes := make([]string, len(r.Campos))
	for i, c := range r.Campos {
		if c.Tipo == TDecimal {
			partes[i] = fmt.Sprintf("%s::numeric(38,%d)::text", ident(c.Nombre), escalaDecimal)
		} else {
			partes[i] = ident(c.Nombre)
		}
	}
	return strings.Join(partes, ", ")
}

// marcador devuelve el parámetro $n, con el cast que necesita el tipo.
func marcador(c Campo, n int) string {
	if c.Tipo == TDecimal {
		return fmt.Sprintf("$%d::numeric", n)
	}
	return fmt.Sprintf("$%d", n)
}

func leerFilas(r *Recurso, rows pgx.Rows) ([]Fila, error) {
	defer rows.Close()
	filas := []Fila{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		filas = append(filas, Fila{recurso: r, valores: vals})
	}
	return filas, rows.Err()
}

// Listar devuelve una página ordenada por la clave primaria.
func (m *Motor) Listar(ctx context.Context, r *Recurso, limit, offset int) ([]Fila, error) {
	sql := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s OFFSET $1 LIMIT $2", columnas(r), ident(r.Tabla), ident(r.PK))
	rows, err := m.DB.Query(ctx, sql, offset, limit)
	if err != nil {
		return nil, err
	}
	return leerFilas(r, rows)
}

// Obtener devuelve un registro por su clave; el segundo valor es false si no existe.
func (m *Motor) Obtener(ctx context.Context, r *Recurso, id any) (Fila, bool, error) {
	sql := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", columnas(r), ident(r.Tabla), ident(r.PK))
	rows, err := m.DB.Query(ctx, sql, id)
	if err != nil {
		return Fila{}, false, err
	}
	filas, err := leerFilas(r, rows)
	if err != nil || len(filas) == 0 {
		return Fila{}, false, err
	}
	return filas[0], true, nil
}

// ExisteUnico dice si ya hay un registro con ese valor en el campo de unicidad del recurso.
func (m *Motor) ExisteUnico(ctx context.Context, r *Recurso, valor any) (bool, error) {
	c, _ := r.Campo(r.Unico.Campo)
	sql := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = %s)", ident(r.Tabla), ident(c.Nombre), marcador(c, 1))
	var existe bool
	err := m.DB.QueryRow(ctx, sql, valor).Scan(&existe)
	return existe, err
}

// Crear inserta el registro con los valores ya validados y devuelve lo guardado.
func (m *Motor) Crear(ctx context.Context, r *Recurso, valores map[string]any) (Fila, error) {
	var cols, marcas []string
	var args []any
	for _, c := range r.Campos {
		if c.SoloLectura {
			continue
		}
		args = append(args, valores[c.Nombre])
		cols = append(cols, ident(c.Nombre))
		marcas = append(marcas, marcador(c, len(args)))
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s", ident(r.Tabla), strings.Join(cols, ", "), strings.Join(marcas, ", "), columnas(r))
	rows, err := m.DB.Query(ctx, sql, args...)
	if err != nil {
		return Fila{}, err
	}
	filas, err := leerFilas(r, rows)
	if err != nil || len(filas) == 0 {
		return Fila{}, err
	}
	return filas[0], nil
}

// Actualizar cambia las columnas indicadas; el segundo valor es false si el registro no existe.
func (m *Motor) Actualizar(ctx context.Context, r *Recurso, id any, valores map[string]any) (Fila, bool, error) {
	if len(valores) == 0 {
		return m.Obtener(ctx, r, id)
	}
	var sets []string
	var args []any
	for _, c := range r.Campos {
		v, ok := valores[c.Nombre]
		if !ok {
			continue
		}
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = %s", ident(c.Nombre), marcador(c, len(args))))
	}
	args = append(args, id)
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d RETURNING %s", ident(r.Tabla), strings.Join(sets, ", "), ident(r.PK), len(args), columnas(r))
	rows, err := m.DB.Query(ctx, sql, args...)
	if err != nil {
		return Fila{}, false, err
	}
	filas, err := leerFilas(r, rows)
	if err != nil || len(filas) == 0 {
		return Fila{}, false, err
	}
	return filas[0], true, nil
}

// Borrar elimina el registro; devuelve false si no existía.
func (m *Motor) Borrar(ctx context.Context, r *Recurso, id any) (bool, error) {
	tag, err := m.DB.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", ident(r.Tabla), ident(r.PK)), id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Avanzada es la lista con búsqueda parcial, filtros exactos y orden. Los nombres de campo que no
// son columnas del recurso se ignoran, como hacía la API de Python.
func (m *Motor) Avanzada(ctx context.Context, r *Recurso, p ParamsAvanzada) ([]Fila, error) {
	var donde []string
	var args []any

	if p.Search != "" {
		campos := r.Busqueda
		if p.SearchFields != "" {
			campos = nil
			for _, nombre := range strings.Split(p.SearchFields, ",") {
				nombre = strings.TrimSpace(nombre)
				if _, ok := r.Campo(nombre); ok && nombre != "" {
					campos = append(campos, nombre)
				}
			}
		}
		if len(campos) > 0 {
			args = append(args, "%"+p.Search+"%")
			var ors []string
			for _, nombre := range campos {
				ors = append(ors, fmt.Sprintf("CAST(%s AS TEXT) ILIKE $%d", ident(nombre), len(args)))
			}
			donde = append(donde, "("+strings.Join(ors, " OR ")+")")
		}
	}
	for _, par := range strings.Split(p.Filters, ",") {
		par = strings.TrimSpace(par)
		nombre, valor, ok := strings.Cut(par, ":")
		if !ok {
			continue
		}
		nombre = strings.TrimSpace(nombre)
		if _, existe := r.Campo(nombre); !existe {
			continue
		}
		args = append(args, strings.TrimSpace(valor))
		donde = append(donde, fmt.Sprintf("CAST(%s AS TEXT) = $%d", ident(nombre), len(args)))
	}

	orden := r.PK
	if _, ok := r.Campo(p.OrderBy); ok {
		orden = p.OrderBy
	}
	dir := "ASC"
	if p.OrderDir == "desc" {
		dir = "DESC"
	}
	sql := fmt.Sprintf("SELECT %s FROM %s", columnas(r), ident(r.Tabla))
	if len(donde) > 0 {
		sql += " WHERE " + strings.Join(donde, " AND ")
	}
	sql += fmt.Sprintf(" ORDER BY %s %s, %s OFFSET $%d LIMIT $%d", ident(orden), dir, ident(r.PK), len(args)+1, len(args)+2)
	args = append(args, p.Offset, p.Limit)

	rows, err := m.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return leerFilas(r, rows)
}
```

`gastos-service/internal/crud/http.go`:

```go
package crud

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
)

// Registrar expone los recursos bajo el prefijo (p. ej. /api/v1): listar, obtener, crear,
// actualizar, borrar y la lista avanzada de cada uno.
func Registrar(mux *http.ServeMux, m *Motor, prefijo string, recursos []Recurso) {
	for i := range recursos {
		r := &recursos[i]
		base := prefijo + "/" + r.Ruta
		mux.HandleFunc("GET "+base, func(w http.ResponseWriter, req *http.Request) { m.listar(w, req, r) })
		mux.HandleFunc("GET "+base+"/{id}", func(w http.ResponseWriter, req *http.Request) { m.obtener(w, req, r) })
		mux.HandleFunc("POST "+base, func(w http.ResponseWriter, req *http.Request) { m.crear(w, req, r) })
		mux.HandleFunc("PUT "+base+"/{id}", func(w http.ResponseWriter, req *http.Request) { m.actualizar(w, req, r) })
		mux.HandleFunc("DELETE "+base+"/{id}", func(w http.ResponseWriter, req *http.Request) { m.borrar(w, req, r) })
		mux.HandleFunc("GET "+prefijo+"/advanced/"+r.Ruta, func(w http.ResponseWriter, req *http.Request) { m.avanzada(w, req, r) })
	}
}

// EscribirJSON responde con JSON sin escapar <, > ni &, como FastAPI.
func EscribirJSON(w http.ResponseWriter, estado int, v any) {
	cuerpo, err := jsonSinEscape(v)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(estado)
	_, _ = w.Write(append(cuerpo, '\n'))
}

// Detalle responde {"detail": "<mensaje>"}, el formato de error de /api/v1.
func Detalle(w http.ResponseWriter, estado int, mensaje string) {
	EscribirJSON(w, estado, map[string]string{"detail": mensaje})
}

func validacion(w http.ResponseWriter, errs []ErrorCampo) {
	EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
}

// falloInterno registra la causa y responde como un error no controlado de FastAPI.
func falloInterno(w http.ResponseWriter, err error) {
	log.Printf("crud: %v", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// entero lee un parámetro entero con límites opcionales (min/max con ok=false si no aplican).
func entero(origen, nombre, txt string, def int, min, max *int) (int, *ErrorCampo) {
	if txt == "" {
		return def, nil
	}
	n, err := strconv.Atoi(txt)
	if err != nil {
		return 0, &ErrorCampo{Tipo: "int_parsing", Loc: []any{origen, nombre}, Msg: "Input should be a valid integer, unable to parse string as an integer", Input: txt}
	}
	if min != nil && n < *min {
		return 0, &ErrorCampo{Tipo: "greater_than_equal", Loc: []any{origen, nombre}, Msg: "Input should be greater than or equal to " + strconv.Itoa(*min), Input: txt, Ctx: map[string]any{"ge": *min}}
	}
	if max != nil && n > *max {
		return 0, &ErrorCampo{Tipo: "less_than_equal", Loc: []any{origen, nombre}, Msg: "Input should be less than or equal to " + strconv.Itoa(*max), Input: txt, Ctx: map[string]any{"le": *max}}
	}
	return n, nil
}

func paginacion(req *http.Request) (limit, offset int, errs []ErrorCampo) {
	uno, cero, doscientos := 1, 0, 200
	q := req.URL.Query()
	limit, e1 := entero("query", "limit", q.Get("limit"), 50, &uno, &doscientos)
	offset, e2 := entero("query", "offset", q.Get("offset"), 0, &cero, nil)
	for _, e := range []*ErrorCampo{e1, e2} {
		if e != nil {
			errs = append(errs, *e)
		}
	}
	return limit, offset, errs
}

// claveRuta lee {id}; nombre es el nombre del parámetro en el error (la clave en GET, item_id en PUT/DELETE).
func claveRuta(req *http.Request, nombre string) (int64, *ErrorCampo) {
	txt := req.PathValue("id")
	n, err := strconv.ParseInt(txt, 10, 64)
	if err != nil {
		return 0, &ErrorCampo{Tipo: "int_parsing", Loc: []any{"path", nombre}, Msg: "Input should be a valid integer, unable to parse string as an integer", Input: txt}
	}
	return n, nil
}

func (m *Motor) listar(w http.ResponseWriter, req *http.Request, r *Recurso) {
	limit, offset, errs := paginacion(req)
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	filas, err := m.Listar(req.Context(), r, limit, offset)
	if err != nil {
		falloInterno(w, err)
		return
	}
	EscribirJSON(w, http.StatusOK, filas)
}

func (m *Motor) obtener(w http.ResponseWriter, req *http.Request, r *Recurso) {
	id, e := claveRuta(req, r.PK)
	if e != nil {
		validacion(w, []ErrorCampo{*e})
		return
	}
	fila, ok, err := m.Obtener(req.Context(), r, id)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if !ok {
		Detalle(w, http.StatusNotFound, r.NoEncontrado)
		return
	}
	EscribirJSON(w, http.StatusOK, fila)
}

// leerObjeto lee el cuerpo como un objeto JSON; si no lo es, responde 422 y devuelve false.
func leerObjeto(w http.ResponseWriter, req *http.Request) (map[string]any, bool) {
	crudo, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		falloInterno(w, err)
		return nil, false
	}
	var v any
	if err := json.Unmarshal(crudo, &v); err != nil {
		validacion(w, []ErrorCampo{{Tipo: "json_invalid", Loc: []any{"body", 0}, Msg: "JSON decode error", Input: map[string]any{}}})
		return nil, false
	}
	obj, ok := v.(map[string]any)
	if !ok {
		validacion(w, []ErrorCampo{{Tipo: "model_attributes_type", Loc: []any{"body"}, Msg: "Input should be a valid dictionary or object to extract fields from", Input: v}})
		return nil, false
	}
	return obj, true
}

func (m *Motor) crear(w http.ResponseWriter, req *http.Request, r *Recurso) {
	cuerpo, ok := leerObjeto(w, req)
	if !ok {
		return
	}
	valores, errs := ValidarCreacion(r, cuerpo)
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	if r.Unico != nil {
		existe, err := m.ExisteUnico(req.Context(), r, valores[r.Unico.Campo])
		if err != nil {
			falloInterno(w, err)
			return
		}
		if existe {
			Detalle(w, http.StatusConflict, r.Unico.Mensaje)
			return
		}
	}
	fila, err := m.Crear(req.Context(), r, valores)
	if err != nil {
		falloInterno(w, err)
		return
	}
	EscribirJSON(w, http.StatusCreated, fila)
}

func (m *Motor) actualizar(w http.ResponseWriter, req *http.Request, r *Recurso) {
	id, e := claveRuta(req, "item_id")
	cuerpo, ok := leerObjeto(w, req)
	if !ok {
		return
	}
	var errs []ErrorCampo
	if e != nil {
		errs = append(errs, *e)
	}
	valores, errsCuerpo := ValidarCambios(r, cuerpo)
	errs = append(errs, errsCuerpo...)
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	fila, existe, err := m.Actualizar(req.Context(), r, id, valores)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if !existe {
		Detalle(w, http.StatusNotFound, "Recurso no encontrado")
		return
	}
	EscribirJSON(w, http.StatusOK, fila)
}

func (m *Motor) borrar(w http.ResponseWriter, req *http.Request, r *Recurso) {
	id, e := claveRuta(req, "item_id")
	if e != nil {
		validacion(w, []ErrorCampo{*e})
		return
	}
	existia, err := m.Borrar(req.Context(), r, id)
	if err != nil {
		falloInterno(w, err)
		return
	}
	if !existia {
		Detalle(w, http.StatusNotFound, "Recurso no encontrado")
		return
	}
	EscribirJSON(w, http.StatusOK, borrado{r: r, id: id})
}

// borrado es la respuesta de DELETE: {"deleted": true, "<clave>": id}.
type borrado struct {
	r  *Recurso
	id int64
}

func (b borrado) MarshalJSON() ([]byte, error) {
	clave, _ := jsonSinEscape(b.r.PK)
	return []byte(`{"deleted":true,` + string(clave) + `:` + strconv.FormatInt(b.id, 10) + `}`), nil
}

func (m *Motor) avanzada(w http.ResponseWriter, req *http.Request, r *Recurso) {
	limit, offset, errs := paginacion(req)
	q := req.URL.Query()
	dir := q.Get("order_dir")
	if dir == "" {
		dir = "asc"
	}
	if dir != "asc" && dir != "desc" {
		errs = append(errs, ErrorCampo{Tipo: "string_pattern_mismatch", Loc: []any{"query", "order_dir"}, Msg: "String should match pattern '^(asc|desc)$'", Input: dir, Ctx: map[string]any{"pattern": "^(asc|desc)$"}})
	}
	if len(errs) > 0 {
		validacion(w, errs)
		return
	}
	filas, err := m.Avanzada(req.Context(), r, ParamsAvanzada{
		Limit: limit, Offset: offset, Search: q.Get("search"), SearchFields: q.Get("search_fields"),
		OrderBy: q.Get("order_by"), OrderDir: dir, Filters: q.Get("filters"),
	})
	if err != nil {
		falloInterno(w, err)
		return
	}
	EscribirJSON(w, http.StatusOK, filas)
}
```

En `gastos-service/internal/handlers/handlers.go`, agrega `"gastos-service/internal/crud"` a los imports (junto a `config`) y, dentro de `Register`, antes de `	// Archivos de comprobantes servidos estáticamente`:

```go
	// API de maestros y operación (antes en la API de Python): 19 recursos CRUD
	crud.Registrar(mux, &crud.Motor{DB: h.DB}, "/api/v1", crud.Recursos)

```

- [ ] **Step 6: Ejecutar y verificar que pasan**

Run: `cd gastos-service && gofmt -l . ; go vet ./... && TEST_DATABASE_URL=… go test ./... -count=1`
Expected: todos los paquetes en `ok`; las tres pruebas nuevas en PASS (`TestAltaDeRecursosConPadreUnico`, `TestLaBusquedaNoEsUnaInyeccionSQL`, `TestCORSPermitePUTyDELETEYLaSaludIncluyeElEntorno`).

- [ ] **Step 7: Checkpoint (sin commit)**

Run: `psql … -tAc "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'prueba_%'"`
Expected: `0` esquemas sobrantes.

---

### Task 4: Autenticación (Go)

**Files:**
- Create: `gastos-service/internal/auth/auth.go`, `http.go`, `gastos-service/migrations/0003_usuario.sql`
- Modify: `gastos-service/go.mod`, `go.sum`, `internal/handlers/handlers.go`, `main.go`
- Test: `gastos-service/internal/auth/auth_test.go`

**Interfaces:**
- Consumes: `crud.EscribirJSON`, `crud.Detalle`, `crud.ErrorCampo` (Tareas 2–3); `config.JWTSecret`.
- Produces: `auth.Servicio{DB Consultas, Clave []byte, Ahora func() time.Time}` con `CrearToken`, `UsuarioDeToken`, `PorUsername`, `PorEmail`, `Crear`, `SembrarAdmin`, `Registrar(mux, prefijo)`; `auth.Hash`, `auth.Verificar`, `auth.ClaveDefecto`; `handlers.Handler.Auth`.

- [ ] **Step 1: Escribir las pruebas que fallan**

`gastos-service/internal/auth/auth_test.go` (el token y los hashes de Python se generaron con `python-jose` y `bcrypt` 4.2.1 y la clave por defecto):

```go
package auth

import (
	"strings"
	"testing"
	"time"

	"gastos-service/internal/pruebas"
)

// Credenciales emitidas por la API de Python (python-jose y bcrypt 4.2.1) con la clave por defecto.
const (
	tokenDePython = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsInJvbCI6ImFkbWluIiwiaWQiOjEsImV4cCI6NDEwMjQ0NDgwMH0.xlL_fAko0xnKCrwjzZfEnHLrPSmrUl2BdTaYzo99oZY"
	hashDePython  = "$2b$12$XB21nbr7C1dvI4aRdexRMuKDzyVBhkDhbIQJoPLFAweJzAuQmMWMW" // «admin123»
	hash80xDePy   = "$2b$12$mTvnxMsig9wJW4uhbJ41Pecj8wMh..Ke7In5SDkd6euDbD7YZrQTO" // 80 letras «x»
)

func servicioDePrueba() *Servicio {
	return &Servicio{Clave: []byte(ClaveDefecto), Ahora: func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC) }}
}

func TestSeAceptaElHashDePython(t *testing.T) {
	if !Verificar("admin123", hashDePython) {
		t.Fatal("un hash bcrypt de Python debía validar su contraseña")
	}
	if Verificar("otra", hashDePython) {
		t.Fatal("una contraseña distinta no debía validar")
	}
}

func TestContrasenasLargasSeRecortanA72BytesComoEnPython(t *testing.T) {
	if !Verificar(strings.Repeat("x", 80), hash80xDePy) || !Verificar(strings.Repeat("x", 72), hash80xDePy) {
		t.Fatal("bcrypt solo mira 72 bytes: ambas contraseñas debían validar contra el hash de Python")
	}
	h, err := Hash(strings.Repeat("y", 100))
	if err != nil || !Verificar(strings.Repeat("y", 100), h) {
		t.Fatalf("hash propio de una contraseña larga: %v", err)
	}
	if !strings.HasPrefix(h, "$2a$12$") && !strings.HasPrefix(h, "$2b$12$") {
		t.Fatalf("el hash debía usar bcrypt con costo 12: %s", h[:7])
	}
}

func TestSeAceptaElTokenDePython(t *testing.T) {
	usuario, err := servicioDePrueba().UsuarioDeToken(tokenDePython)
	if err != nil || usuario != "admin" {
		t.Fatalf("UsuarioDeToken = %q, %v", usuario, err)
	}
}

func TestTokenPropioLlevaLosMismosDatos(t *testing.T) {
	s := servicioDePrueba()
	token, err := s.CrearToken(Usuario{ID: 3, Username: "ana", Rol: "operador"})
	if err != nil {
		t.Fatal(err)
	}
	if usuario, err := s.UsuarioDeToken(token); err != nil || usuario != "ana" {
		t.Fatalf("UsuarioDeToken = %q, %v", usuario, err)
	}
}

func TestTokensInvalidos(t *testing.T) {
	s := servicioDePrueba()
	vencido, _ := (&Servicio{Clave: s.Clave, Ahora: func() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) }}).CrearToken(Usuario{Username: "ana"})
	otraClave, _ := (&Servicio{Clave: []byte("otra"), Ahora: s.Ahora}).CrearToken(Usuario{Username: "ana"})
	for nombre, tok := range map[string]string{"vacío": "", "basura": "no.es.token", "vencido": vencido, "otra clave": otraClave} {
		if _, err := s.UsuarioDeToken(tok); err == nil {
			t.Errorf("%s: el token debía rechazarse", nombre)
		}
	}
}

func TestSembrarAdminSoloSiNoHayUsuarios(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	s := &Servicio{DB: pool, Clave: []byte(ClaveDefecto)}
	if err := s.SembrarAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	admin, err := s.PorUsername(t.Context(), "admin")
	if err != nil || admin == nil || admin.Rol != "admin" || !admin.Estado || !Verificar("admin123", admin.Hash) {
		t.Fatalf("admin sembrado = %+v, %v", admin, err)
	}
	if err := s.SembrarAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM usuario").Scan(&n); err != nil || n != 1 {
		t.Fatalf("usuarios = %d, %v; con uno ya creado no debía sembrar otro", n, err)
	}
}
```

- [ ] **Step 2: Ejecutar y verificar que fallan**

Run: `cd gastos-service && go test ./internal/auth/ -count=1 2>&1 | head -4`
Expected: FAIL de compilación: `undefined: Servicio`, `undefined: Verificar`.

- [ ] **Step 3: Dependencias y migración**

Run:
```bash
cd gastos-service
go get github.com/golang-jwt/jwt/v5@v5.3.1 golang.org/x/crypto@v0.55.0
grep -n "^go " go.mod
```
Expected: `go 1.25.4` (si cambiara a 1.26, se usó una versión equivocada de `x/crypto`: reinstalar la v0.55.0).

`gastos-service/migrations/0003_usuario.sql`:

```sql
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
```

- [ ] **Step 4: Implementar**

`gastos-service/internal/auth/auth.go`:

```go
// Package auth reemplaza la autenticación de la API de Python: contraseñas con bcrypt y
// tokens JWT HS256 que llevan sub (usuario), rol, id y exp (8 horas).
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ClaveDefecto es la clave con que la API de Python firmaba los tokens. Se conserva como valor
// por defecto para que los tokens ya emitidos sigan valiendo; en producción debe definirse JWT_SECRET.
const ClaveDefecto = "lizar-srl-fleet-manager-secret-2025"

const vigenciaToken = 8 * time.Hour

// Usuario es la fila de la tabla usuario.
type Usuario struct {
	ID       int
	Username string
	Email    string
	Hash     string
	Nombre   string
	Rol      string
	Estado   bool
}

// Servicio agrupa la clave de firma y el acceso a la tabla usuario.
type Servicio struct {
	DB    Consultas
	Clave []byte
	Ahora func() time.Time
}

// Consultas es lo que necesita de la base (lo cumple *pgxpool.Pool).
type Consultas interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Hash genera el hash bcrypt (costo 12, como la API de Python). bcrypt solo mira los primeros 72
// bytes: se recorta igual que la versión de Python que se usaba, en vez de rechazar la contraseña.
func Hash(contrasena string) (string, error) {
	h, err := bcrypt.GenerateFromPassword(recortar(contrasena), 12)
	return string(h), err
}

// Verificar compara una contraseña con su hash (los hashes $2b$ de Python son válidos).
func Verificar(contrasena, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), recortar(contrasena)) == nil
}

func recortar(s string) []byte {
	b := []byte(s)
	if len(b) > 72 {
		b = b[:72]
	}
	return b
}

// CrearToken emite un token con la vigencia estándar.
func (s *Servicio) CrearToken(u Usuario) (string, error) {
	claims := jwt.MapClaims{"sub": u.Username, "rol": u.Rol, "id": u.ID, "exp": s.ahora().Add(vigenciaToken).Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.Clave)
}

// UsuarioDeToken devuelve el nombre de usuario del token, o error si no es válido o venció.
func (s *Servicio) UsuarioDeToken(token string) (string, error) {
	t, err := jwt.Parse(token, func(*jwt.Token) (any, error) { return s.Clave, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithTimeFunc(s.ahora))
	if err != nil || !t.Valid {
		return "", errors.New("token inválido")
	}
	sub, _ := t.Claims.(jwt.MapClaims)["sub"].(string)
	if sub == "" {
		return "", errors.New("token inválido")
	}
	return sub, nil
}

func (s *Servicio) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

const columnasUsuario = "id_usuario, username, email, hashed_password, nombre, rol, estado"

func leer(row pgx.Row) (*Usuario, error) {
	var u Usuario
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Hash, &u.Nombre, &u.Rol, &u.Estado)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

// PorUsername busca un usuario; devuelve nil si no existe.
func (s *Servicio) PorUsername(ctx context.Context, username string) (*Usuario, error) {
	return leer(s.DB.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE username = $1", username))
}

// PorEmail busca un usuario por correo; devuelve nil si no existe.
func (s *Servicio) PorEmail(ctx context.Context, email string) (*Usuario, error) {
	return leer(s.DB.QueryRow(ctx, "SELECT "+columnasUsuario+" FROM usuario WHERE email = $1", email))
}

// Crear guarda un usuario nuevo con la contraseña ya hasheada.
func (s *Servicio) Crear(ctx context.Context, u Usuario, contrasena string) (*Usuario, error) {
	hash, err := Hash(contrasena)
	if err != nil {
		return nil, err
	}
	return leer(s.DB.QueryRow(ctx, `
		INSERT INTO usuario (username, email, hashed_password, nombre, rol, estado, created_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6)
		RETURNING `+columnasUsuario, u.Username, u.Email, hash, u.Nombre, u.Rol, s.ahora().UTC()))
}

// SembrarAdmin crea el usuario admin/admin123 si todavía no hay ningún usuario. La contraseña de
// ese usuario debe cambiarse en producción.
func (s *Servicio) SembrarAdmin(ctx context.Context) error {
	var n int
	if err := s.DB.QueryRow(ctx, "SELECT count(*) FROM usuario").Scan(&n); err != nil || n > 0 {
		return err
	}
	_, err := s.Crear(ctx, Usuario{Username: "admin", Email: "admin@lizar.pe", Nombre: "Administrador", Rol: "admin"}, "admin123")
	return err
}
```

`gastos-service/internal/auth/http.go`:

```go
package auth

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"gastos-service/internal/crud"
)

var reCorreo = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// usuarioOut es la respuesta pública de un usuario (sin el hash).
type usuarioOut struct {
	ID       int    `json:"id_usuario"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nombre   string `json:"nombre"`
	Rol      string `json:"rol"`
	Estado   bool   `json:"estado"`
}

func salida(u *Usuario) usuarioOut {
	return usuarioOut{u.ID, u.Username, u.Email, u.Nombre, u.Rol, u.Estado}
}

// Registrar expone /auth/login, /auth/register y /auth/me bajo el prefijo.
func (s *Servicio) Registrar(mux *http.ServeMux, prefijo string) {
	mux.HandleFunc("POST "+prefijo+"/auth/login", s.login)
	mux.HandleFunc("POST "+prefijo+"/auth/register", s.registro)
	mux.HandleFunc("GET "+prefijo+"/auth/me", s.me)
}

func faltante(campo string, cuerpo any) crud.ErrorCampo {
	return crud.ErrorCampo{Tipo: "missing", Loc: []any{"body", campo}, Msg: "Field required", Input: cuerpo}
}

// texto lee un campo de texto obligatorio; añade el error y devuelve false si falta o no es texto.
func texto(cuerpo map[string]any, campo string, errs *[]crud.ErrorCampo) (string, bool) {
	v, ok := cuerpo[campo]
	if !ok {
		*errs = append(*errs, faltante(campo, cuerpo))
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		*errs = append(*errs, crud.ErrorCampo{Tipo: "string_type", Loc: []any{"body", campo}, Msg: "Input should be a valid string", Input: v})
		return "", false
	}
	return s, true
}

func leerCuerpo(w http.ResponseWriter, req *http.Request) (map[string]any, bool) {
	var obj map[string]any
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20)).Decode(&obj); err != nil || obj == nil {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": []crud.ErrorCampo{
			{Tipo: "json_invalid", Loc: []any{"body", 0}, Msg: "JSON decode error", Input: map[string]any{}}}})
		return nil, false
	}
	return obj, true
}

func (s *Servicio) login(w http.ResponseWriter, req *http.Request) {
	cuerpo, ok := leerCuerpo(w, req)
	if !ok {
		return
	}
	var errs []crud.ErrorCampo
	username, _ := texto(cuerpo, "username", &errs)
	contrasena, _ := texto(cuerpo, "password", &errs)
	if len(errs) > 0 {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
		return
	}
	u, err := s.PorUsername(req.Context(), username)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if u == nil || !Verificar(contrasena, u.Hash) {
		crud.Detalle(w, http.StatusUnauthorized, "Credenciales inválidas")
		return
	}
	if !u.Estado {
		crud.Detalle(w, http.StatusUnauthorized, "Usuario inactivo")
		return
	}
	token, err := s.CrearToken(*u)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	crud.EscribirJSON(w, http.StatusOK, struct {
		AccessToken string     `json:"access_token"`
		TokenType   string     `json:"token_type"`
		Usuario     usuarioOut `json:"usuario"`
	}{token, "bearer", salida(u)})
}

func (s *Servicio) registro(w http.ResponseWriter, req *http.Request) {
	cuerpo, ok := leerCuerpo(w, req)
	if !ok {
		return
	}
	var errs []crud.ErrorCampo
	username, _ := texto(cuerpo, "username", &errs)
	email, okEmail := texto(cuerpo, "email", &errs)
	if okEmail && !reCorreo.MatchString(email) {
		errs = append(errs, crud.ErrorCampo{Tipo: "value_error", Loc: []any{"body", "email"}, Msg: "value is not a valid email address", Input: email})
	}
	contrasena, _ := texto(cuerpo, "password", &errs)
	nombre, _ := texto(cuerpo, "nombre", &errs)
	rol := "operador"
	if v, presente := cuerpo["rol"]; presente {
		if r, ok := texto(cuerpo, "rol", &errs); ok {
			rol = r
			if rol != "admin" && rol != "operador" {
				errs = append(errs, crud.ErrorCampo{Tipo: "value_error", Loc: []any{"body", "rol"}, Msg: "Value error, Rol debe ser 'admin' u 'operador'", Input: v})
			}
		}
	}
	if len(errs) > 0 {
		crud.EscribirJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
		return
	}
	if existe, err := s.PorUsername(req.Context(), username); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	} else if existe != nil {
		crud.Detalle(w, http.StatusConflict, "El usuario '"+username+"' ya existe")
		return
	}
	if existe, err := s.PorEmail(req.Context(), email); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	} else if existe != nil {
		crud.Detalle(w, http.StatusConflict, "El email '"+email+"' ya está registrado")
		return
	}
	u, err := s.Crear(req.Context(), Usuario{Username: username, Email: email, Nombre: nombre, Rol: rol}, contrasena)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	crud.EscribirJSON(w, http.StatusCreated, salida(u))
}

func (s *Servicio) me(w http.ResponseWriter, req *http.Request) {
	cabecera := req.Header.Get("Authorization")
	if !strings.HasPrefix(cabecera, "Bearer ") {
		crud.Detalle(w, http.StatusUnauthorized, "No autenticado")
		return
	}
	username, err := s.UsuarioDeToken(strings.SplitN(cabecera, " ", 2)[1])
	if err != nil {
		crud.Detalle(w, http.StatusUnauthorized, "Token inválido")
		return
	}
	u, err := s.PorUsername(req.Context(), username)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if u == nil || !u.Estado {
		crud.Detalle(w, http.StatusUnauthorized, "Usuario no encontrado")
		return
	}
	crud.EscribirJSON(w, http.StatusOK, salida(u))
}
```

En `gastos-service/internal/handlers/handlers.go`: agrega `"gastos-service/internal/auth"` a los imports; cambia

```go
type Handler struct {
	DB   *pgxpool.Pool
	Cfg  config.Config
	Auth *auth.Servicio
}

func New(pool *pgxpool.Pool, cfg config.Config) *Handler {
	return &Handler{DB: pool, Cfg: cfg, Auth: &auth.Servicio{DB: pool, Clave: []byte(cfg.JWTSecret)}}
}
```

y en `Register`, justo después de la línea de `crud.Registrar(…)`, agrega `	h.Auth.Registrar(mux, "/api/v1")`.

En `gastos-service/main.go`: agrega `"context"` a los imports y reemplaza `handlers.New(pool, cfg).Register(mux)` por `handler.Register(mux)`, creando antes (justo antes de `os.MkdirAll`):

```go
	handler := handlers.New(pool, cfg)
	if err := handler.Auth.SembrarAdmin(context.Background()); err != nil {
		log.Fatalf("no se pudo sembrar el usuario admin: %v", err)
	}

```

Run: `cd gastos-service && go mod tidy && go build ./... && go vet ./...`
Expected: sin errores; `jwt/v5` y `x/crypto` quedan como dependencias directas.

- [ ] **Step 5: Ejecutar y verificar que pasan**

Run: `cd gastos-service && TEST_DATABASE_URL=… go test ./internal/auth/ -count=1 -v 2>&1 | grep -E "^(--- |ok|FAIL)"`
Expected: 6 `--- PASS` (hash de Python, contraseñas largas, token de Python, token propio, tokens inválidos, siembra del admin) y `ok`.

- [ ] **Step 6: Checkpoint (sin commit)**

Run: `cd gastos-service && gofmt -l . ; go vet ./... && TEST_DATABASE_URL=… go test ./... -count=1`
Expected: todo en `ok`.

---

### Task 5: Test de contrato contra la API de Python

**Files:**
- Test: `gastos-service/internal/handlers/contrato_test.go`

**Interfaces:**
- Consumes: todo lo anterior; `internal/crud/contrato/escenario.json` y `python.json` (Tarea 1).
- Produces: `TestContratoConLaApiDePython`, que repite sobre Go los 552 pasos del escenario y compara cada respuesta con la de Python.

- [ ] **Step 1: Escribir la prueba**

`gastos-service/internal/handlers/contrato_test.go`:

```go
package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"gastos-service/internal/config"
	"gastos-service/internal/handlers"
	"gastos-service/internal/pruebas"
)

type pasoContrato struct {
	Metodo    string            `json:"metodo"`
	Ruta      string            `json:"ruta"`
	Cuerpo    any               `json:"cuerpo"`
	Token     string            `json:"token"`
	Cabeceras map[string]string `json:"cabeceras"`
	Guarda    string            `json:"guarda"`
}

type respuestaContrato struct {
	Estado int `json:"estado"`
	Cuerpo any `json:"cuerpo"`
}

func leerJSON(t *testing.T, nombre string, destino any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "crud", "contrato", nombre))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, destino); err != nil {
		t.Fatal(err)
	}
}

// TestContratoConLaApiDePython repite sobre Go el escenario capturado de la API de Python
// (internal/crud/contrato) y exige el mismo código y la misma forma de respuesta en cada paso.
func TestContratoConLaApiDePython(t *testing.T) {
	pool := pruebas.NuevoEsquemaCompleto(t)
	h := handlers.New(pool, config.Config{UploadsDir: t.TempDir(), AppEnv: "dev", JWTSecret: "prueba"})
	if err := h.Auth.SembrarAdmin(t.Context()); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h.Register(mux)

	var pasos []pasoContrato
	var esperadas []respuestaContrato
	leerJSON(t, "escenario.json", &pasos)
	leerJSON(t, "python.json", &esperadas)
	if len(pasos) != len(esperadas) {
		t.Fatalf("escenario (%d) y respuestas de Python (%d) no coinciden", len(pasos), len(esperadas))
	}

	ultimoID := map[string]any{}
	tokens := map[string]string{}
	fallos := 0
	for i, p := range pasos {
		ruta := p.Ruta
		if strings.Contains(ruta, "{id}") {
			id, _ := ultimoID[strings.Split(ruta, "/{id}")[0]].(float64)
			ruta = strings.Replace(ruta, "{id}", strconv.FormatFloat(id, 'f', -1, 64), 1)
		}
		var cuerpo *bytes.Reader
		if p.Cuerpo != nil {
			b, _ := json.Marshal(p.Cuerpo)
			cuerpo = bytes.NewReader(b)
		} else {
			cuerpo = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(p.Metodo, ruta, cuerpo)
		if p.Cuerpo != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if p.Token != "" {
			tok := p.Token
			if v, ok := tokens[p.Token]; ok {
				tok = v
			}
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		for k, v := range p.Cabeceras {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		handlers.CORS(mux).ServeHTTP(rec, req)

		var obtenido any
		if err := json.Unmarshal(rec.Body.Bytes(), &obtenido); err != nil {
			obtenido = rec.Body.String()
		}
		if p.Guarda != "" && rec.Code == http.StatusCreated {
			ultimoID[p.Ruta] = obtenido.(map[string]any)[p.Guarda]
		} else if p.Guarda != "" {
			ultimoID[p.Ruta] = 0
		}
		if p.Ruta == "/api/v1/auth/login" && rec.Code == http.StatusOK {
			if m, ok := p.Cuerpo.(map[string]any); ok && m["username"] == "admin" {
				tokens["@login-admin"] = obtenido.(map[string]any)["access_token"].(string)
			}
		}

		if msg := comparar(esperadas[i], rec.Code, obtenido); msg != "" {
			fallos++
			if fallos <= 25 {
				t.Errorf("paso %d %s %s: %s", i, p.Metodo, ruta, msg)
			}
		}
	}
	if fallos > 25 {
		t.Errorf("... y %d pasos más con diferencias", fallos-25)
	}
}

// comparar devuelve la diferencia entre la respuesta de Python y la de Go, o "" si coinciden.
// Un 500 solo se compara por su código (el cuerpo de Python es una traza de depuración); en un
// 422 se comparan el tipo y la ubicación de cada error; el token cambia en cada emisión.
func comparar(esperada respuestaContrato, estado int, obtenido any) string {
	if esperada.Estado != estado {
		return fmt.Sprintf("código %d, Python daba %d (%.200v)", estado, esperada.Estado, obtenido)
	}
	switch estado {
	case http.StatusInternalServerError:
		return ""
	case http.StatusUnprocessableEntity:
		a, b := resumen422(esperada.Cuerpo), resumen422(obtenido)
		if !reflect.DeepEqual(a, b) {
			return fmt.Sprintf("errores 422 distintos:\n  Python: %v\n  Go:     %v", a, b)
		}
		return ""
	}
	esp, obt := normalizar(esperada.Cuerpo), normalizar(obtenido)
	if !reflect.DeepEqual(esp, obt) {
		return fmt.Sprintf("cuerpo distinto:\n  Python: %.300v\n  Go:     %.300v", esp, obt)
	}
	return ""
}

func normalizar(v any) any {
	if m, ok := v.(map[string]any); ok {
		if _, tiene := m["access_token"]; tiene {
			c := map[string]any{}
			for k, x := range m {
				c[k] = x
			}
			c["access_token"] = "<token>"
			return c
		}
	}
	return v
}

func resumen422(v any) []string {
	m, _ := v.(map[string]any)
	detalle, _ := m["detail"].([]any)
	var out []string
	for _, d := range detalle {
		e, _ := d.(map[string]any)
		out = append(out, fmt.Sprintf("%v %v", e["type"], e["loc"]))
	}
	return out
}
```

Reglas de comparación (en `comparar`): mismo código HTTP; un 500 se compara solo por su código (el cuerpo de Python es una traza de depuración); un 422 se compara por el `type` y el `loc` de cada error, en orden; el `access_token` se normaliza porque cambia en cada emisión; el resto, JSON idéntico (el orden de las claves no cuenta).

- [ ] **Step 2: Ejecutar**

Run: `cd gastos-service && TEST_DATABASE_URL=… go test ./internal/handlers/ -run Contrato -count=1 2>&1 | grep -v "^20[0-9][0-9]/" | head -40`
Expected: `ok  gastos-service/internal/handlers` (las líneas `crud: ERROR: …` del log son los 500 esperados, filtrados por el `grep`). Si hay diferencias, cada una aparece como `paso N MÉTODO ruta: …` con la respuesta de Python y la de Go.

- [ ] **Step 3: Si el contrato falla, diagnosticar antes de tocar nada**

- Si los ids de las altas no coinciden (`id_xxx:9` en Python y `:3` en Go): la captura se hizo sobre una base usada. Vuelve a la Tarea 1 (Step 2 y 3).
- Si un decimal sale sin escala: el campo quedó como `TTexto` en `recursos.go`; regenera con el generador (Tarea 1, Step 4).
- Cualquier otra diferencia es un defecto del motor: corrígelo en `crud` con una prueba unitaria que lo reproduzca primero.

- [ ] **Step 4: Comprobar que la prueba es sensible**

Cambia temporalmente `const escalaDecimal = 2` a `3` en `internal/crud/motor.go`, ejecuta de nuevo el contrato y confirma que falla con diferencias de decimales; deja `2` otra vez.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd gastos-service && TEST_DATABASE_URL=… go test ./... -count=1 -v 2>&1 | grep -cE "^--- PASS"`
Expected: `59` (más las pruebas de checklist y reportes ya existentes si se cuentan todas: en total no hay `--- FAIL` ni `SKIP`).

---

### Task 6: Frontend: dirección base única

**Files:**
- Modify: `frontend/src/app/core/services/api.service.ts`
- Test: `frontend/src/app/core/services/api.service.spec.ts` (nuevo)

**Interfaces:**
- Consumes: la API en `http://localhost:8081/api/v1`.
- Produces: `ApiService.base()` en desarrollo local = `http://localhost:8081/api/v1`; las bases guardadas con `localhost:8000` o `127.0.0.1:8000` se reemplazan por la base por defecto.

- [ ] **Step 1: Escribir la prueba que falla**

`frontend/src/app/core/services/api.service.spec.ts`:

```ts
import { beforeEach, describe, expect, it } from 'vitest';
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { ApiService } from './api.service';

const CLAVE = 'lizar_api_base';

function servicio(): { api: ApiService; http: HttpTestingController } {
  TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] });
  return { api: TestBed.inject(ApiService), http: TestBed.inject(HttpTestingController) };
}

describe('ApiService', () => {
  beforeEach(() => localStorage.clear());

  it('en desarrollo local usa la API única en el puerto 8081', () => {
    const { api } = servicio();
    expect(api.base()).toBe('http://localhost:8081/api/v1');
    expect(localStorage.getItem(CLAVE)).toBe('http://localhost:8081/api/v1');
  });

  it('una base guardada que apunta al antiguo servicio del 8000 se reemplaza', () => {
    localStorage.setItem(CLAVE, 'http://localhost:8000/api/v1');
    const { api } = servicio();
    expect(api.base()).toBe('http://localhost:8081/api/v1');
  });

  it('la salud se consulta en la raíz del servicio, sin /api/v1', () => {
    const { api, http } = servicio();
    api.checkHealth();
    http.expectOne('http://localhost:8081/health').flush({ status: 'ok', env: 'dev' });
    expect(api.health()).toEqual({ connected: true, env: 'dev' });
  });
});
```

- [ ] **Step 2: Ejecutar y verificar que falla**

Run: `cd frontend && npx ng test --watch=false --include src/app/core/services/api.service.spec.ts 2>&1 | sed 's/\x1b\[[0-9;]*m//g' | grep -E "×|Tests |Expected|Received"`
Expected: 3 pruebas fallan; `Expected: "http://localhost:8081/api/v1"` / `Received: "http://localhost:8000/api/v1"`.

- [ ] **Step 3: Implementar**

En `frontend/src/app/core/services/api.service.ts`, en `resolveInitialBase`, cambia `const localBase = 'http://localhost:8000/api/v1';` por `const localBase = 'http://localhost:8081/api/v1';`. (La base remota por defecto `http://13.222.175.52:8000/api/v1` depende del servidor desplegado: no se toca; cuando se despliegue la versión Go habrá que apuntarla al nuevo puerto o dominio.)

- [ ] **Step 4: Ejecutar y verificar que pasa**

Run: el mismo comando del Step 2.
Expected: `Tests  3 passed (3)`.

- [ ] **Step 5: Checkpoint (sin commit)**

Run: `cd frontend && npx ng test --watch=false && npx ng build 2>&1 | grep -E "WARNING|ERROR|Initial total"`
Expected: toda la suite en PASS y build sin avisos.

---

### Task 7: Despliegue y documentación

**Files:**
- Create: `gastos-service/Dockerfile`, `gastos-service/.dockerignore`
- Modify: `docker-compose.demo.yml`, `README.md`

**Interfaces:**
- Consumes: el binario del servicio y `db.sql`.
- Produces: la demo en Docker con un solo servicio Go (puerto 8081) sobre PostgreSQL con los datos de `db.sql`.

- [ ] **Step 1: Dockerfile y .dockerignore**

`gastos-service/Dockerfile`:

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/gastos-service .

FROM alpine:3.22
RUN adduser -D -u 10001 lizar && mkdir -p /app/uploads && chown lizar /app/uploads
WORKDIR /app
COPY --from=build /out/gastos-service .
COPY migrations ./migrations
USER lizar
EXPOSE 8081
CMD ["./gastos-service"]
```

`gastos-service/.dockerignore`:

```
.env
uploads/
*.exe
tools/
internal/crud/contrato/
```

- [ ] **Step 2: Compose**

En `docker-compose.demo.yml`:
- en el servicio `db`, dentro de `volumes:` agrega la línea `      - ./db.sql:/docker-entrypoint-initdb.d/01-db.sql:ro` (la base de la demo ahora nace con el esquema y los datos de `db.sql`; antes Python solo creaba las tablas vacías);
- reemplaza el servicio `api` completo por:

```yaml
  api:
    build:
      context: ./gastos-service
    container_name: lizar-demo-api
    restart: unless-stopped
    environment:
      PORT: "8081"
      DATABASE_URL: postgres://lizar:lizar_demo_pass@db:5432/lizar_demo?sslmode=disable
      UPLOADS_DIR: /app/uploads
      APP_ENV: demo
    volumes:
      - lizar_demo_uploads:/app/uploads
    depends_on:
      db:
        condition: service_healthy
    ports:
      - "8081:8081"
```

- en `volumes:` al final agrega `  lizar_demo_uploads:`.

- [ ] **Step 3: README**

Reemplaza el contenido de `README.md` por:

```markdown
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

Variables: `PORT` (8081), `DATABASE_URL`, `UPLOADS_DIR`, `JWT_SECRET` y `APP_ENV` (`dev`). En producción define `JWT_SECRET` y cambia la contraseña de `admin`.

## Pruebas

- Go: `cd gastos-service && TEST_DATABASE_URL=postgres://… go test ./...` (sin la variable, las pruebas con base de datos se omiten).
- Frontend: `cd frontend && npx ng test --watch=false`.
- Contrato con la API de Python: las respuestas capturadas están en `gastos-service/internal/crud/contrato/`; para regenerarlas, `gastos-service/tools/contrato/`.

## Demo en Docker

`docker compose -f docker-compose.demo.yml up --build` levanta PostgreSQL (con `db.sql`) y la API Go en el puerto 8081.
```

- [ ] **Step 4: Verificar lo que se puede verificar sin Docker**

Run:
```bash
cd gastos-service && CGO_ENABLED=0 go build -trimpath -o /dev/null . && echo "BUILD-ESTATICO-OK"
python3 -c "import re,sys; t=open('../docker-compose.demo.yml').read(); assert 'lizar_demo_uploads' in t and 'gastos-service' in t and 'uvicorn' not in t; print('COMPOSE-OK')"
```
Expected: `BUILD-ESTATICO-OK` y `COMPOSE-OK`. **El `docker build` y el `docker compose up` no se pueden probar aquí (no hay Docker instalado): se informa al usuario como no verificado.**

- [ ] **Step 5: Checkpoint (sin commit)**

Revisa que `git status --short backend` no muestre cambios (la carpeta de Python no se toca).

---

### Task 8: Verificación punta a punta con Python apagado y cierre

**Files (fuera del repositorio, en el scratchpad):**
- `$SCRATCH/visual/sin-python.mjs`

**Interfaces:**
- Consumes: todo lo anterior; el gastos-service reiniciado con el código nuevo; la base real del usuario (solo se lee y se prueba con un rubro temporal que se borra).
- Produces: evidencia de que el frontend funciona sin tocar el puerto 8000.

- [ ] **Step 1: Comparación en vivo contra el Python real (solo lecturas)**

Run:
```bash
cd gastos-service && go build -o "$SCRATCH/gs-vivo" . && \
DATABASE_URL="$(sed -n 's/^DATABASE_URL=//p' .env | tr -d "\"'")" PORT=8093 UPLOADS_DIR="$SCRATCH/uploads-vivo" "$SCRATCH/gs-vivo" > "$SCRATCH/gs-vivo.log" 2>&1 &
sleep 3 && python3 tools/contrato/comparar_en_vivo.py http://localhost:8000 http://localhost:8093
```
Expected: `<N> consultas comparadas, 0 con diferencias` (con los datos actuales, 106). Detén la instancia 8093 al terminar (`kill $(lsof -nP -iTCP:8093 -sTCP:LISTEN -t)`).

- [ ] **Step 2: Compatibilidad de sesión en vivo**

Run:
```bash
python3 - <<'EOF'
import json, urllib.request
def llamar(base, metodo, ruta, cuerpo=None, token=None):
    req = urllib.request.Request(base + ruta, data=None if cuerpo is None else json.dumps(cuerpo).encode(), method=metodo)
    if cuerpo is not None: req.add_header('Content-Type', 'application/json')
    if token: req.add_header('Authorization', 'Bearer ' + token)
    r = urllib.request.urlopen(req, timeout=10); return r.status, json.loads(r.read())
_, d = llamar('http://localhost:8000', 'POST', '/api/v1/auth/login', {'username': 'admin', 'password': 'admin123'})
print('me en Go con token de Python:', llamar('http://localhost:8093', 'GET', '/api/v1/auth/me', token=d['access_token'])[0])
_, g = llamar('http://localhost:8093', 'POST', '/api/v1/auth/login', {'username': 'admin', 'password': 'admin123'})
print('me en Python con token de Go:', llamar('http://localhost:8000', 'GET', '/api/v1/auth/me', token=g['access_token'])[0])
EOF
```
(Hazlo antes de detener la instancia 8093.) Expected: `200` en ambas. Si el admin real tiene otra contraseña, usa una cuenta conocida o salta este paso anotándolo.

- [ ] **Step 3: Reiniciar gastos-service en el 8081 con el código nuevo**

Run:
```bash
PIDS=$(lsof -nP -iTCP:8081 -sTCP:LISTEN -t); PADRE=$(ps -o ppid= -p $PIDS | tr -d ' '); kill $PIDS $PADRE; sleep 2
cd gastos-service && nohup go run . > "$SCRATCH/gastos-service.log" 2>&1 &
sleep 8; python3 -c "import urllib.request;print(urllib.request.urlopen('http://localhost:8081/health').read(), urllib.request.urlopen('http://localhost:8081/api/v1/rubros').status)"
```
Expected: `{"env":"dev","status":"ok"}` y `200`. La API principal y la de gastos corren ahora en un solo proceso.

- [ ] **Step 4: Recorrido en Chrome con el puerto 8000 bloqueado**

`$SCRATCH/visual/sin-python.mjs` (levanta Playwright con el Chrome del sistema; si falta `playwright-core` en `$SCRATCH/visual`, instalarlo con `npm install --no-audit --no-fund playwright-core@1.48.2`):

```js
// Recorrido del frontend con el servicio Python apagado: todo lo que antes iba al puerto 8000 debe ir a Go.
// Uso: node sin-python.mjs <url-frontend> [url-go]   (por defecto http://localhost:8081)
// Con una url-go distinta de 8081, las peticiones al 8081 se reenvían ahí (para probar una instancia aparte).
import { chromium } from 'playwright-core';

const FRONT = process.argv[2] ?? 'http://localhost:4300';
const GO = (process.argv[3] ?? 'http://localhost:8081').replace(/\/$/, '');
const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const RUBRO = `PRUEBA-MIGRACION-${Date.now()}`;

const verificar = (condicion, mensaje) => {
  if (!condicion) throw new Error(`Falló: ${mensaje}`);
  console.log('OK', mensaje);
};

const navegador = await chromium.launch({ executablePath: CHROME, headless: true });
const ctx = await navegador.newContext({ viewport: { width: 1440, height: 900 } });
const alPuerto8000 = [];
const erroresDeRed = [];
await ctx.route(/localhost:8000|127\.0\.0\.1:8000/, ruta => {
  alPuerto8000.push(ruta.request().url());
  ruta.abort();
});
if (GO !== 'http://localhost:8081') {
  await ctx.route(/localhost:8081/, ruta => ruta.continue({ url: ruta.request().url().replace('http://localhost:8081', GO) }));
}
const p = await ctx.newPage();
p.on('response', r => {
  if (r.status() >= 500) erroresDeRed.push(`${r.status()} ${r.url()}`);
});

// 1. Login real contra Go (el hash del usuario lo creó Python)
await p.goto(`${FRONT}/login`, { waitUntil: 'networkidle' });
await p.locator('#username').fill('admin');
await p.locator('#password').fill('admin123');
await p.getByRole('button', { name: /ingresar|entrar|iniciar/i }).last().click();
await p.waitForURL(/\/app\//, { timeout: 15000 });
verificar(true, 'el login con el usuario creado por Python entra por Go');

// 2. Operación y Catálogos cargan sus listas desde Go
for (const ruta of ['operacion/viajes', 'operacion/clientes', 'operacion/conductores', 'operacion/flota', 'catalogos/ubigeo', 'catalogos/rubros', 'catalogos/marcas']) {
  await p.goto(`${FRONT}/app/${ruta}`, { waitUntil: 'networkidle' });
  const hayFilas = (await p.locator('tbody tr').count()) > 0;
  const hayVacio = (await p.getByText(/Aún no hay/).count()) > 0;
  verificar(hayFilas || hayVacio, `${ruta} carga desde Go (${hayFilas ? 'con registros' : 'vacía'})`);
}

// 3. Alta de un rubro desde la pantalla, y limpieza por la API
await p.goto(`${FRONT}/app/catalogos/rubros`, { waitUntil: 'networkidle' });
await p.getByRole('button', { name: /Nuevo rubro/ }).first().click();
await p.locator('#ru_n').fill(RUBRO);
await p.getByRole('button', { name: 'Guardar' }).click();
await p.getByText(RUBRO).first().waitFor({ timeout: 10000 });
verificar(true, 'un rubro nuevo se guarda en Go y aparece en la tabla');
const lista = await (await fetch(`${GO}/api/v1/rubros?limit=200`)).json();
const creado = lista.find(r => r.nombre === RUBRO);
const borrado = await fetch(`${GO}/api/v1/rubros/${creado.id_rubro}`, { method: 'DELETE' });
verificar(borrado.status === 200, 'el rubro de prueba se borra por la API (PUT y DELETE pasan el CORS en el navegador)');

// 4. Gastos y Reportes siguen funcionando contra la misma API
await p.goto(`${FRONT}/app/gastos`, { waitUntil: 'networkidle' });
verificar((await p.locator('tbody tr, .fila-gasto, li').count()) > 0, 'Gastos carga');
await p.goto(`${FRONT}/app/reportes`, { waitUntil: 'networkidle' });
await p.locator('.indicador__valor').first().waitFor({ timeout: 15000 });
verificar(true, 'Reportes carga');

verificar(alPuerto8000.length === 0, `ninguna petición salió al puerto 8000${alPuerto8000.length ? ': ' + alPuerto8000[0] : ''}`);
verificar(erroresDeRed.length === 0, `ninguna respuesta 5xx${erroresDeRed.length ? ': ' + erroresDeRed[0] : ''}`);
await navegador.close();
console.log('Recorrido sin Python terminado.');
```

Run (con el frontend en el 4200 del usuario si está arriba, o con `npx ng serve --port 4300` propio): `node "$SCRATCH/visual/sin-python.mjs" http://localhost:4300`
Expected: líneas `OK …` para el login real contra Go, la carga de Operación y Catálogos, el alta, la edición y el borrado de un rubro temporal, la carga de Gastos y Reportes, y `ninguna petición salió al puerto 8000`.

- [ ] **Step 5: Verificación final (sin commit)**

Run:
```bash
cd gastos-service && gofmt -l . ; go vet ./... && TEST_DATABASE_URL=… go test ./... -count=1
psql … -tAc "SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'prueba_%'"
cd ../frontend && npx ng test --watch=false && npx ng build 2>&1 | grep -E "WARNING|ERROR|Initial total"
cd .. && git status --short backend | head -3 && git log --oneline | head -1
```
Expected: Go en `ok`; `0` esquemas sobrantes; frontend en verde y build sin avisos; `git status --short backend` vacío; `git log` sigue en `1953886 first commit`.

Informa al usuario: qué quedó en Go; que **gastos-service ya corre en el 8081 con todo**; que `backend/` quedó intacto; la deuda de seguridad heredada (clave JWT por defecto, `admin/admin123`, rutas abiertas); que el Docker no se pudo probar aquí; y que la base remota por defecto del frontend (`13.222.175.52:8000`) hay que apuntarla al desplegar.
