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
