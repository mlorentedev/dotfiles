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
    # "Not running" is two different states and only one of them is benign. A unit that
    # was never started is genuinely un-inspectable, and a SKIP is honest. A unit that
    # STARTED AND DIED is a failure, and a SKIP there reports the criterion as merely
    # unverified while the daemon crash-loops in the background.
    #
    # Measured 2026-08-25 on msi: hive.service sat in `activating (auto-restart)` after
    # five consecutive exit-code failures, and this block printed its SKIP. The run
    # reported pass=5 fail=2 with the daemon dead -- and neither failure was this one.
    # Only AC6's dispatch caught it, for an unrelated reason.
    #
    # Result= is the discriminator, not ActiveState=: a crash-looping unit reads
    # ActiveState=activating (indistinguishable from a healthy slow start) while
    # Result= already records exit-code.
    STATE="$(systemctl --user show hive.service -p ActiveState --value 2>/dev/null)"
    RESULT="$(systemctl --user show hive.service -p Result --value 2>/dev/null)"
    RESTARTS="$(systemctl --user show hive.service -p NRestarts --value 2>/dev/null)"
    case "${STATE}/${RESULT}" in
        failed/*|*/exit-code|*/signal|*/core-dump|*/timeout|*/oom-kill)
            bad "hive.service STARTED AND DIED (ActiveState=$STATE Result=$RESULT NRestarts=${RESTARTS:-0}) — journalctl --user -u hive.service -n 20"
            ;;
        *)
            note "daemon not running (ActiveState=${STATE:-unknown}), so its environment could not be inspected"
            ;;
    esac
fi

# The whole point of AC7: nothing on disk. A credential in an EnvironmentFile or
# an environment.d fragment would satisfy every check above and defeat the criterion.
#
# `find`, never a glob in a `for`. The first draft of this block used
# `for f in …/environment.d/*.conf`, which is the prohibited-pattern row in
# .claude/CLAUDE.md, and it failed in the worst possible direction for a SECURITY
# check. Measured with a planted credential in hive.env:
#
#   bash -> CAUGHT: credential in …/hive.env
#   zsh  -> "no matches found" — NOMATCH aborts the whole compound command, so
#           hive.env was never examined either, and the script went on to print
#           its "[OK] nothing on disk" line having looked at nothing.
#
# A false negative here reads as "the credential is safely off disk" when it is
# sitting right there. Caught by the reviewer on #1232.
# The names come from the REGISTRY, not from a guess about what a credential
# looks like. The first draft matched /API_KEY|TOKEN|SECRET|PASSWORD/ and would
# have missed 14 of this repo's 35 declared secrets — BITACORA_PAT, TS_AUTHKEY,
# KUBECONFIG, SSH_KEY, AGE_KEY_PERSONAL, every *_BACKUP_CODE and *_RECOVERY_CODE.
# A 40% blind spot in a check whose only job is to notice a credential on disk.
#
# The heuristic stays as a FALLBACK for a tree with no readable registry, and the
# script says which one it used — a narrower check reported as if it were the
# full one is how the blind spot survived in the first place.
CFG="${XDG_CONFIG_HOME:-$HOME/.config}"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
# A function, not a heredoc inside $( ), which is fragile across shells: the
# first version of this silently took the fallback path on a tree where the
# registry was perfectly readable.
registry_cred_regex() {
    python3 - "$1" <<'PY'
import sys, yaml
doc = yaml.safe_load(open(sys.argv[1]))
names = []
for s in doc.get("secrets", []) or []:
    exp = s.get("expose") or {}
    env = exp.get("env")
    if isinstance(env, str):
        names.append(env)
    elif isinstance(env, list):
        names += [v for v in env if isinstance(v, str)]
    elif isinstance(env, dict):
        names += list(env.keys())
    fil = exp.get("file")
    if isinstance(fil, dict) and fil.get("var"):
        names.append(fil["var"])
if names:
    print("^[[:space:]]*(export[[:space:]]+)?(" + "|".join(sorted(set(names))) + ")=")
PY
}

CRED_RE=""
if [ -n "$REPO_ROOT" ] && [ -f "$REPO_ROOT/secrets/registry.yaml" ]; then
    CRED_RE="$(registry_cred_regex "$REPO_ROOT/secrets/registry.yaml" 2>/dev/null || true)"
fi
if [ -n "$CRED_RE" ]; then
    printf '       (scanning for the %s names declared in secrets/registry.yaml)\n' \
        "$(printf '%s' "$CRED_RE" | awk -F'|' '{print NF}')"
else
    CRED_RE='^[[:space:]]*(export[[:space:]]+)?[A-Za-z_]*(API_KEY|TOKEN|SECRET|PASSWORD)[A-Za-z_]*='
    printf '       (registry unreadable — falling back to a NAME HEURISTIC, which is narrower)\n'
fi
CANDIDATES="$(
    [ -f "$CFG/hive/hive.env" ] && printf '%s\n' "$CFG/hive/hive.env"
    find "$CFG/environment.d" -maxdepth 1 -type f -name '*.conf' 2>/dev/null
)"

# Read line by line rather than `for f in $CANDIDATES`: zsh does not word-split
# an unquoted parameter, so that form yields ONE field containing every path.
# The heredoc keeps the loop in THIS shell, so LEAK survives it — a pipe would
# put it in a subshell and the assignment would be lost.
LEAK=0
while IFS= read -r f; do
    [ -n "$f" ] || continue
    if grep -qE "$CRED_RE" "$f" 2>/dev/null; then
        bad "a credential-shaped assignment is on disk in $f"
        LEAK=1
    fi
done <<CANDIDATE_LIST
$CANDIDATES
CANDIDATE_LIST
[ "$LEAK" -eq 0 ] && ok "no credential-shaped assignment in hive.env or any environment.d fragment"

hdr "AC9 — doctor catches 'probes present, serves nothing'"

# --verbose is load-bearing, not cosmetic. `dotf doctor` SUMMARISES a section whose
# checks all pass -- it prints "(1 checks, all ok)" and never the check's own message --
# so the healthy verdict this block greps for is emitted ONLY under --verbose. Without
# it the `ok` branch below is unreachable: a healthy machine falls through to "printed
# neither verdict" and the criterion reports FAIL precisely when it is satisfied.
#
# Measured 2026-08-25 on msi, after the deploy that made this check green:
#   dotf doctor           -> [hive backend reachability (dotf agent run)]
#                            (1 checks, all ok)
#   dotf doctor --verbose -> [ OK ] hive.service carries both halves of the worker
#                            contract -- the backend can reach its pool
#
# Third time this spec has shipped a check that asks the FORM of a thing instead of
# what it does (lesson 230). The failing half worked -- 'can serve nothing' is printed
# unconditionally, because a FAILING check is never summarised away. Only the passing
# half was unreachable, which is why nothing caught it until a green machine ran it.
if command -v dotf >/dev/null 2>&1; then
    DOC="$(dotf doctor --verbose 2>&1)"
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
