#!/usr/bin/env python3
"""Comprobantes de demostración para el OCR (macOS: usa cupsfilter, pdftoppm y psql).

Toma los comprobantes PENDIENTES de los gastos DEMO más recientes, crea una factura ficticia por
cada uno en uploads/ y los apunta a ella. El OCR solo lee comprobantes pendientes, y las muestras
que ya hay en uploads/ (guías de remisión) no traen monto, así que no sirven para lucirlo.

Casos (en orden, del gasto más reciente al más antiguo):
  1-3  PDF digital con monto y fecha iguales a los del gasto     -> 4 sugerencias
  4    foto PNG (obliga a usar Tesseract), todo igual             -> 4 sugerencias
  5    PDF con un monto distinto al del gasto                     -> «Leído S/ X», sin ✓ en el monto
  6    PDF a nombre de otra empresa                               -> sin ✓ en «a nombre de LIZAR»

Se puede repetir. Los archivos son demo-ocr-*; reportes_demo_limpiar.sql borra los comprobantes
pero no estos archivos: bórralos a mano (rm uploads/demo-ocr-*).

  DATABASE_URL=postgres://… python3 gastos-service/demo/generar_comprobantes_ocr.py
"""
import os
import subprocess
import sys
import tempfile
from datetime import date
from pathlib import Path

UPLOADS = Path(__file__).resolve().parent.parent / "uploads"
DB = os.environ.get("DATABASE_URL")
if not DB:
    sys.exit("Define DATABASE_URL")


def psql(sql: str) -> str:
    r = subprocess.run(["psql", "-X", "-At", "-F", "|", "-v", "ON_ERROR_STOP=1", "-c", sql, DB],
                       capture_output=True, text=True)
    if r.returncode:
        sys.exit(r.stderr)
    return r.stdout


def factura(numero: int, fecha: date, total: float, receptor: str, ruc_receptor: str) -> str:
    subtotal = round(total / 1.18, 2)
    return f"""
        TRANSPORTES Y SERVICIOS ANDINOS SAC (DEMO)
        R.U.C. 20999999991
        AV. LOS INCAS 1450 - TRUJILLO - LA LIBERTAD

        FACTURA ELECTRONICA
        F001 - {numero:08d}

        FECHA DE EMISION: {fecha:%d/%m/%Y}

        SENORES: {receptor}
        R.U.C.: {ruc_receptor}

        DESCRIPCION                          CANT     IMPORTE
        SERVICIO DE PEAJE Y VIATICOS           1     S/ {subtotal:.2f}

        SUBTOTAL                                     S/ {subtotal:.2f}
        IGV 18%                                      S/ {total - subtotal:.2f}
        TOTAL A PAGAR: S/ {total:.2f}

        Documento de demostracion - no es un comprobante real
"""


filas = psql("""
    SELECT c.id, g.id, g.monto, g.fecha
    FROM comprobante c JOIN gasto g ON g.id = c.gasto_id
    WHERE g.creado_por = 'DEMO' AND c.estado = 'Pendiente'
    ORDER BY g.fecha DESC, g.id DESC LIMIT 6""").strip().splitlines()
if len(filas) < 6:
    sys.exit("Faltan comprobantes pendientes de gastos DEMO: ejecuta reportes_demo.sql y reportes_demo_comprobantes.sql")

LIZAR = ("EMPRESA DE TRANSPORTES LIZAR SRL", "20481608750")
OTRA = ("COMERCIAL PACIFICO EIRL", "20555555551")

for i, fila in enumerate(filas, start=1):
    comprobante_id, gasto_id, monto, fecha = fila.split("|")
    monto, fecha = float(monto), date.fromisoformat(fecha)
    leido = monto + 12.50 if i == 5 else monto
    receptor = OTRA if i == 6 else LIZAR
    texto = factura(5000 + int(gasto_id), fecha, leido, *receptor)

    with tempfile.TemporaryDirectory() as tmp:
        txt = Path(tmp) / "f.txt"
        txt.write_text(texto, encoding="utf-8")
        pdf = Path(tmp) / "f.pdf"
        with open(pdf, "wb") as out:
            subprocess.run(["cupsfilter", "-m", "application/pdf", str(txt)], stdout=out,
                           stderr=subprocess.DEVNULL, check=True)
        if i == 4:
            destino, tipo = UPLOADS / f"demo-ocr-{gasto_id}.png", "imagen"
            subprocess.run(["pdftoppm", "-r", "150", "-png", "-singlefile", str(pdf), str(destino.with_suffix(""))], check=True)
        else:
            destino, tipo = UPLOADS / f"demo-ocr-{gasto_id}.pdf", "pdf"
            destino.unlink(missing_ok=True)
            destino.write_bytes(pdf.read_bytes())
    destino.chmod(0o444)
    psql(f"UPDATE comprobante SET archivo_url = '/uploads/{destino.name}', tipo_archivo = '{tipo}' WHERE id = {comprobante_id}")
    print(f"gasto {gasto_id} (S/ {monto:.2f}, {fecha:%d/%m/%Y}) -> {destino.name}; factura por S/ {leido:.2f} a {receptor[0]}")
