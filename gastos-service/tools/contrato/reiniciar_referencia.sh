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
