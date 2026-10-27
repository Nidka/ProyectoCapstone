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
