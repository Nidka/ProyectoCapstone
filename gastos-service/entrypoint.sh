#!/bin/sh
# Arranque del contenedor. Un disco persistente (Render, un volumen de Docker) puede llegar montado
# como root: se crean las carpetas de comprobantes y del registro de errores, se le entregan al
# usuario lizar y el servicio corre sin privilegios.
set -eu

UPLOADS_DIR="${UPLOADS_DIR:-/app/uploads}"
LOG_ARCHIVO="${LOG_ARCHIVO:-/app/logs/errores.log}"
LOG_DIR="$(dirname "$LOG_ARCHIVO")"

if [ "$(id -u)" = "0" ]; then
	mkdir -p "$UPLOADS_DIR" "$LOG_DIR"
	chown lizar "$UPLOADS_DIR" "$LOG_DIR"
	exec su-exec lizar ./gastos-service
fi

exec ./gastos-service
