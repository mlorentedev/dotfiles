#!/usr/bin/env bash
# orca-tune.sh — reproducible Orca (Stably AI) global-settings tuning.
#
# Orca keeps every global setting in a single JSON file
# (${ORCA_USER_DATA_PATH:-~/.config/orca}/orca-data.json) and exposes NO
# `orca config set` CLI. Settings can therefore only be changed via the in-app
# UI or by editing that JSON file *while Orca is fully closed* — a running Orca
# persists the file from memory on its own schedule and would clobber a live
# edit. This script is the config-as-code form of that edit:
#
#   * idempotent  — re-running with the baseline already applied is a no-op
#   * safe        — timestamped backup before any write, atomic replace
#   * guarded     — refuses to run while an Orca process is alive
#
# Re-run after a reinstall / new machine to restore the tuned baseline.
# Tuning decisions are tracked in the bitácora (see CHANGELOG / ticket).
#
# Usage:
#   scripts/orca-tune.sh            # apply the baseline (Orca must be closed)
#   scripts/orca-tune.sh --dry-run  # show what would change, write nothing
set -euo pipefail

ORCA_DIR="${ORCA_USER_DATA_PATH:-$HOME/.config/orca}"
DATA="$ORCA_DIR/orca-data.json"
DRY_RUN=0
[ "${1:-}" = "--dry-run" ] && DRY_RUN=1

[ -f "$DATA" ] || {
  echo "ERROR: $DATA not found — is Orca installed for this user?" >&2
  exit 1
}

# Refuse to WRITE while Orca is live (it would overwrite our change on next
# save). --dry-run is always allowed since it writes nothing.
# Note: /usr/bin/orca is the unrelated GNOME screen reader — we only match the
# Stably Orca binaries (orca-ide / the AppImage), never the screen reader.
if [ "$DRY_RUN" -eq 0 ] && pgrep -f 'orca-ide|orca-linux[^ ]*\.AppImage' >/dev/null 2>&1; then
  echo "ERROR: Orca is running. Quit it fully (incl. the background daemon)," >&2
  echo "       then re-run. To force-stop a lingering daemon:" >&2
  echo "         pkill -f 'orca-ide|orca-linux.*AppImage'" >&2
  exit 1
fi

python3 - "$DATA" "$DRY_RUN" <<'PY'
import json, sys, os, datetime, shutil

data_path, dry = sys.argv[1], sys.argv[2] == "1"

# --- Tuned baseline (edit here to change desired state) ---------------------
# Chosen 2026-06-17 (Tier 1+2, symlinks left OFF, mobile left ON):
#   experimentalAgentHibernation=True + 10min idle  -> biggest RAM saver for
#                                                       parallel agent fleets
#   refreshLocalBaseRefOnWorktreeCreate=True         -> new worktrees branch
#                                                       from a fresh base ref
#   telemetry.optedIn=False                          -> opt out of PostHog
DESIRED = {
    "experimentalAgentHibernation": True,
    "agentHibernationIdleMs": 600000,
    "refreshLocalBaseRefOnWorktreeCreate": True,
    "telemetry.optedIn": False,
}

with open(data_path) as f:
    d = json.load(f)
settings = d.setdefault("settings", {})

changes = []
def set_dotted(root, dotted, val):
    obj = root
    for part in dotted.split(".")[:-1]:
        obj = obj.setdefault(part, {})
    leaf = dotted.split(".")[-1]
    cur = obj.get(leaf, "<absent>")
    if cur != val:
        changes.append((dotted, cur, val))
        obj[leaf] = val

for key, val in DESIRED.items():
    set_dotted(settings, key, val)

if not changes:
    print("No changes — Orca settings already at the tuned baseline.")
    sys.exit(0)

print("Planned changes:")
for dotted, old, new in changes:
    print(f"  settings.{dotted}: {old} -> {new}")

if dry:
    print("(dry-run: nothing written)")
    sys.exit(0)

bak = data_path + ".bak." + datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
shutil.copy2(data_path, bak)
tmp = data_path + ".tmp"
with open(tmp, "w") as f:
    json.dump(d, f, indent=2, ensure_ascii=False)
os.replace(tmp, data_path)
print(f"Applied {len(changes)} change(s). Backup: {bak}")
print("Reopen Orca:  orca-ide open")
PY
