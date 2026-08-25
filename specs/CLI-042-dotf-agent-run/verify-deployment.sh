#!/usr/bin/env bash
# CLI-042 — post-deploy verification, by consequence.
#
# The criteria this closes cannot be verified from the repository. AC6 asks
# whether hive ANSWERS and AC7 asks whether the daemon holds a credential it was
# never given at rest — both are properties of a machine after `./setup-linux.sh`
# has run, not of a diff.
#
# Run this AFTER the deploy. It replaces the checklist in verification.md with a
# single command, so the closing session reads output instead of re-deriving what
# to check.
#
#   ./specs/CLI-042-dotf-agent-run/verify-deployment.sh
#   ./specs/CLI-042-dotf-agent-run/verify-deployment.sh --no-dispatch   # skip the live request
#
# SECRETS: this script never prints a credential. It reports variable NAMES and
# counts, and proves the credential works by CONSEQUENCE (the daemon answers).
# Printing a value to show it exists is the failure, not the verification.

set -uo pipefail

DISPATCH=1
[ "${1:-}" = "--no-dispatch" ] && DISPATCH=0

pass=0; fail=0; skip=0
ok()   { printf '[OK]   %s\n' "$1"; pass=$((pass + 1)); }
bad()  { printf '[FAIL] %s\n' "$1"; fail=$((fail + 1)); }
note() { printf '[SKIP] %s\n' "$1"; skip=$((skip + 1)); }
hdr()  { printf '\n=== %s ===\n' "$1"; }

hdr "AC7 — the credential reaches the daemon, and lives in no file"

DROPIN="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user/hive.service.d/10-dotf-secrets.conf"
if [ -f "$DROPIN" ]; then
    ok "drop-in deployed at $DROPIN"
else
    bad "drop-in NOT deployed — ./setup-linux.sh has not run, or its hive block was skipped"
fi

# The unit systemd actually loaded, not the file on disk: a drop-in that failed
# to parse leaves the base unit in force and `systemctl` is the only witness.
EXECSTART="$(systemctl --user show hive.service -p ExecStart 2>/dev/null)"
case "$EXECSTART" in
    *"secrets run"*NAN_API_KEY*) ok "loaded ExecStart injects the credential via dotf secrets run" ;;
    "")                          bad "hive.service not found — is the daemon installed?" ;;
    *)                           bad "loaded ExecStart has NO credential injection (the drop-in did not take effect)" ;;
esac

ENVIRON="$(systemctl --user show hive.service -p Environment 2>/dev/null)"
case "$ENVIRON" in
    *HIVE_WORKER_BASE_URL=*) ok "loaded unit declares HIVE_WORKER_BASE_URL (the contract's other half)" ;;
    *)                       bad "HIVE_WORKER_BASE_URL absent — the worker stays unconfigured even WITH a key" ;;
esac

# NAMES only. A count is evidence; a value in a transcript is an incident.
PID="$(systemctl --user show hive.service -p MainPID --value 2>/dev/null)"
if [ -n "$PID" ] && [ "$PID" != "0" ] && [ -r "/proc/$PID/environ" ]; then
    if tr '\0' '\n' < "/proc/$PID/environ" | cut -d= -f1 | grep -qx 'HIVE_WORKER_API_KEY'; then
        ok "the RUNNING daemon holds HIVE_WORKER_API_KEY (name observed, value never read)"
    else
        bad "the running daemon does NOT hold HIVE_WORKER_API_KEY — restart it, or the injection failed"
    fi
else
    note "daemon not running, so its environment could not be inspected"
fi

# The whole point of AC7: nothing on disk. A credential in an EnvironmentFile or
# an environment.d fragment would satisfy every check above and defeat the criterion.
LEAK=0
for f in "$HOME/.config/hive/hive.env" "${XDG_CONFIG_HOME:-$HOME/.config}"/environment.d/*.conf; do
    [ -f "$f" ] || continue
    if grep -qE '^[[:space:]]*(export[[:space:]]+)?[A-Za-z_]*(API_KEY|TOKEN|SECRET|PASSWORD)[A-Za-z_]*=' "$f"; then
        bad "a credential-shaped assignment is on disk in $f"
        LEAK=1
    fi
done
[ "$LEAK" -eq 0 ] && ok "no credential-shaped assignment in hive.env or any environment.d fragment"

hdr "AC9 — doctor catches 'probes present, serves nothing'"

if command -v dotf >/dev/null 2>&1; then
    DOC="$(dotf doctor 2>&1)"
    if printf '%s' "$DOC" | grep -q 'can serve nothing'; then
        bad "dotf doctor still reports the backend serving nothing — the deploy did not take"
    elif printf '%s' "$DOC" | grep -q 'can reach its pool'; then
        ok "dotf doctor reports the hive backend can reach its pool"
    else
        bad "dotf doctor printed neither verdict — is the installed binary older than this check?"
    fi
else
    note "dotf not on PATH"
fi

hdr "AC6 — the backend actually answers (one live request)"

if [ "$DISPATCH" -eq 0 ]; then
    note "live dispatch skipped by --no-dispatch"
elif ! command -v dotf >/dev/null 2>&1; then
    note "dotf not on PATH"
elif ! dotf agent run --help >/dev/null 2>&1; then
    bad "the INSTALLED dotf has no 'agent run' — the binary predates the epic; re-run ./setup-linux.sh"
else
    REC="$(dotf agent run --backend hive --tier mid --role reviewer \
            --task 'Reply with the single word: ok' --timeout 2m 2>/dev/null)"
    STATUS="$(printf '%s' "$REC" | jq -r '.status // "unparseable"' 2>/dev/null)"
    POOL="$(printf '%s' "$REC" | jq -r '.pool // "-"' 2>/dev/null)"
    if [ "$STATUS" = "ok" ] && [ "$POOL" = "nan" ]; then
        ok "hive answered; record reports pool=nan as the chain declared"
    else
        bad "dispatch did not answer as expected (status=$STATUS pool=$POOL)"
    fi
fi

hdr "AI-030 — the pi packages the same deploy installs"

PI_SETTINGS="$HOME/.pi/agent/settings.json"
DECLARED="$(jq -r '.packages | length' "$(git rev-parse --show-toplevel)/ai/pi/packages.json" 2>/dev/null || echo 0)"
LIVE="$(jq -r '.packages // [] | length' "$PI_SETTINGS" 2>/dev/null || echo 0)"
if [ "$DECLARED" -gt 0 ] && [ "$LIVE" -ge "$DECLARED" ]; then
    ok "$LIVE pi packages installed (manifest declares $DECLARED)"
else
    bad "pi packages: $LIVE installed, manifest declares $DECLARED"
fi

hdr "Idempotence"
printf 'Not asserted here — run ./setup-linux.sh a SECOND time and confirm it prints\n'
printf '  "hive.service credential drop-in already current (no restart)"\n'
printf 'and installs 0 pi packages. A deploy that changes something on every pass is not IaC.\n'

printf '\n%s\n' "----------------------------------------"
printf 'pass=%d fail=%d skip=%d\n' "$pass" "$fail" "$skip"
[ "$fail" -eq 0 ] || exit 1
