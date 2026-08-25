#!/usr/bin/env bash
# AI-030 (#1224) AC4-AC7: drive the REAL reconcile block against a stubbed pi.
#
# The block is extracted from setup-linux.sh by anchor rather than restated
# here. A copy of the logic would be a two-file agreement nobody checks: it
# would keep passing after the shipped block changed, which is the exact class
# of defect this repository keeps cataloguing. If the anchors move, extraction
# fails loudly instead of testing nothing.
#
# `pi` is stubbed because the real one would install nine third-party packages
# from the network into the invoking user's ~/.pi. What the stub CANNOT prove is
# that pi accepts these arguments (BUG-055's limitation, see
# tests/stub-real-pairing.bats); that claim comes from a real setup run, which
# is what the PR records separately.
#
# Usage: specs/AI-030-pi-packages-manifest/verify-reconcile.sh
# Exit:  0 all scenarios behaved, 1 otherwise

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/../.." && pwd)"
SETUP="$REPO/setup-linux.sh"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

fail() { printf '[FAIL] %s\n' "$*" >&2; exit 1; }

# --- extract the block -------------------------------------------------------

# shellcheck disable=SC2016  # the literal `$CURRENT_DIR` IS the pattern being matched
START="$(grep -n 'PI_PACKAGES_SRC="\$CURRENT_DIR/ai/pi/packages.json"' "$SETUP" | cut -d: -f1)"
[ -n "$START" ] || fail "start anchor not found in setup-linux.sh — the reconcile block moved or was renamed"
END="$(awk -v s="$START" 'NR>s && /^# Deploy opencode TUI config/ {print NR-2; exit}' "$SETUP")"
[ -n "$END" ] || fail "end anchor not found in setup-linux.sh"
sed -n "${START},${END}p" "$SETUP" > "$WORK/block.sh"
# shellcheck disable=SC2016  # likewise: the literal `$PI_BIN` is the pattern
grep -q '"\$PI_BIN" install' "$WORK/block.sh" \
    || fail "the extracted block does not install anything — extraction captured the wrong lines"

# --- harness -----------------------------------------------------------------

cat > "$WORK/pi" <<'STUB'
#!/usr/bin/env bash
[ "$1" = "install" ] || exit 1
printf '%s\n' "$2" >> "$PI_SIM_LOG"
python3 - "$PI_SIM_SETTINGS" "$2" <<'PY'
import json, sys
path, source = sys.argv[1], sys.argv[2]
doc = json.load(open(path))
doc.setdefault("packages", []).append(source)
json.dump(doc, open(path, "w"), indent=2)
PY
STUB
chmod +x "$WORK/pi"

cat > "$WORK/run.sh" <<'RUN'
CURRENT_DIR="$1"; PI_BIN="$2"; PI_SETTINGS_DST="$3"
log_info()    { printf '  [info] %s\n' "$*"; }
log_warning() { printf '  [warn] %s\n' "$*"; }
log_success() { printf '  [ ok ] %s\n' "$*"; }
set -euo pipefail
. "$4"
RUN

export PI_SIM_LOG="$WORK/installed.log"
export PI_SIM_SETTINGS="$WORK/settings.json"
printf '{"defaultProvider":"nan"}\n' > "$PI_SIM_SETTINGS"

# installed_count PI_BINARY -> how many packages that run installed
installed_count() {
    : > "$PI_SIM_LOG"
    bash "$WORK/run.sh" "$REPO" "$1" "$PI_SIM_SETTINGS" "$WORK/block.sh" >"$WORK/out" 2>&1 \
        || fail "the reconcile block exited non-zero: $(cat "$WORK/out")"
    wc -l < "$PI_SIM_LOG" | tr -d ' '
}

DECLARED="$(jq -r '.packages | length' "$REPO/ai/pi/packages.json")"

# --- AC4: a first run installs everything declared ---------------------------
n="$(installed_count "$WORK/pi")"
[ "$n" -eq "$DECLARED" ] || fail "AC4: first run installed $n of $DECLARED declared packages"
printf '[OK] AC4  first run installed all %s declared packages\n' "$DECLARED"

# --- AC5: a second run installs nothing --------------------------------------
n="$(installed_count "$WORK/pi")"
[ "$n" -eq 0 ] || fail "AC5: second run installed $n packages — the reconcile is not idempotent"
grep -q 'already reconciled' "$WORK/out" || fail "AC5: idempotent run did not report itself as unchanged"
printf '[OK] AC5  second run installed 0 (changed=0)\n'

# --- AC6: object-form entries are recognised ---------------------------------
python3 - "$PI_SIM_SETTINGS" <<'PY'
import json, sys
path = sys.argv[1]
doc = json.load(open(path))
# Upstream's per-resource filtering shape, for the first two entries.
doc["packages"] = [
    {"source": s, "skills": []} if i < 2 else s
    for i, s in enumerate(doc["packages"])
]
json.dump(doc, open(path, "w"), indent=2)
PY
n="$(installed_count "$WORK/pi")"
[ "$n" -eq 0 ] || fail "AC6: object-form entries were reinstalled ($n) — only the string form is being read"
printf '[OK] AC6  object-form entries recognised, 0 reinstalled\n'

# --- AC7: pi absent warns and continues --------------------------------------
: > "$PI_SIM_LOG"
if bash "$WORK/run.sh" "$REPO" "$WORK/no-such-pi" "$PI_SIM_SETTINGS" "$WORK/block.sh" >"$WORK/out" 2>&1; then
    grep -q 'skipping pi package reconcile' "$WORK/out" \
        || fail "AC7: pi absent produced no warning: $(cat "$WORK/out")"
    printf '[OK] AC7  pi absent: warned, exit 0, bootstrap continues\n'
else
    fail "AC7: pi absent aborted the block — a missing extension host must not fail setup"
fi

printf '\n[OK] AC4-AC7 verified against the block extracted from setup-linux.sh:%s-%s\n' "$START" "$END"
