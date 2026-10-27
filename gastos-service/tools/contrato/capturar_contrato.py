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
