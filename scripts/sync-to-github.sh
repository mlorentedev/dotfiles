#!/bin/bash
set -e

# Go to project root (second-brain) from any location
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$(realpath "$SCRIPT_DIR/../..")"
cd "$ROOT_DIR"

echo "[INFO] Working from: $ROOT_DIR"
echo "[INFO] Running vault automation tools..."

python3 99-System/04-Scripts/vault-tools.py rename  || echo "[WARN] Rename script failed"
python3 99-System/04-Scripts/vault-tools.py toc     || echo "[WARN] TOC script failed"
python3 99-System/04-Scripts/vault-tools.py index   || echo "[WARN] Index script failed"
python3 99-System/04-Scripts/vault-tools.py links   || echo "[WARN] Broken links check failed"
python3 99-System/04-Scripts/vault-tools.py weekly  || echo "[WARN] Weekly note generation failed"

echo "[INFO] Syncing vault to GitHub..."
git add .
git commit -m "chore(vault): daily sync $(date +%F)" || echo "[INFO] Nothing to commit"
git push origin main || echo "[INFO] Git push skipped (no remote or already up-to-date)"

echo "[✅] Vault sync complete: $(date +%F)"
