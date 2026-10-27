#!/usr/bin/env bash
# Respaldo de los comprobantes (RNF-4): copia uploads/ a una carpeta nueva con la fecha en el nombre.
#
# Variables (todas opcionales):
#   UPLOADS_DIR   carpeta de comprobantes a respaldar   (por defecto: ../uploads, junto a este script)
#   RESPALDO_DIR  carpeta donde se guardan los respaldos (por defecto: ../respaldos)
#
# Cada ejecución crea RESPALDO_DIR/uploads-AAAAMMDD-HHMMSS y no toca los respaldos anteriores.
set -euo pipefail

aqui="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
uploads="${UPLOADS_DIR:-$aqui/../uploads}"
respaldos="${RESPALDO_DIR:-$aqui/../respaldos}"
# FECHA_RESPALDO existe solo para las pruebas; en uso normal se toma la hora actual.
fecha="${FECHA_RESPALDO:-$(date +%Y%m%d-%H%M%S)}"

if [[ ! -d "$uploads" ]]; then
  echo "respaldo: no existe la carpeta de comprobantes (UPLOADS_DIR): $uploads" >&2
  exit 1
fi

destino="$respaldos/uploads-$fecha"
if [[ -e "$destino" ]]; then
  echo "respaldo: ya existe $destino; no se sobrescribe" >&2
  exit 1
fi

mkdir -p "$respaldos"
# -p conserva fechas y permisos (los comprobantes están en solo lectura).
cp -Rp "$uploads" "$destino"
echo "Respaldo creado en $destino"
