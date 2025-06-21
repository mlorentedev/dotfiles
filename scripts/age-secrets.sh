#!/usr/bin/env bash

# age-secrets.sh: Cifra archivos *.secret → *.secret.age
#                 Descifra *.age → *.dec
# Clave privada: ~/.config/age/key.txt (o $AGE_KEY_PATH)
# Uso: ./age-secrets.sh encrypt | decrypt

set -e

VAULT_ROOT="$(dirname "$(dirname "$(dirname "$0")")")"
SECRETS_DIR="$VAULT_ROOT/99-System/05-Sensitive"
KEY_PATH="${AGE_KEY_PATH:-$HOME/.config/age/key.txt}"

function usage() {
  echo "Uso: $0 [encrypt|decrypt]"
  echo
  echo "Variables:"
  echo "  AGE_KEY_PATH   Ruta alternativa al archivo de clave (default: $KEY_PATH)"
  exit 1
}

function encrypt_all() {
  echo "[INFO] Buscando archivos *.secret en $SECRETS_DIR"
  for file in "$SECRETS_DIR"/*.secret; do
    [[ ! -f "$file" ]] && continue
    pubkey=$(grep -o 'age1[0-9a-z]*' "$KEY_PATH")
    outfile="$file.age"
    echo "🔐 Cifrando: $file → $outfile"
    age -r "$pubkey" -o "$outfile" "$file"
  done
}

function decrypt_all() {
  echo "[INFO] Descifrando archivos *.age en $SECRETS_DIR"
  for file in "$SECRETS_DIR"/*.age; do
    [[ ! -f "$file" ]] && continue
    outfile="${file%.age}.dec"
    echo "🔓 Descifrando: $file → $outfile"
    age -d -i "$KEY_PATH" -o "$outfile" "$file"
  done
}

# Main
case "$1" in
  encrypt) encrypt_all ;;
  decrypt) decrypt_all ;;
  *) usage ;;
esac
