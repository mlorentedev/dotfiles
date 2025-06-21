#!/bin/bash
echo "[INFO] Instalando dependencias para pre-commit..."

# Instala pre-commit si no está disponible
if ! command -v pre-commit &> /dev/null; then
  echo "[INFO] Instalando pre-commit con pip"
  pip install pre-commit
fi

# Instala los hooks, incluyendo tipos especiales
pre-commit install --hook-type prepare-commit-msg --hook-type commit-msg

echo "[INFO] Hooks de pre-commit instalados correctamente."
