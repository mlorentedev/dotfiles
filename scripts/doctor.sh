#!/bin/bash
# doctor.sh: verify structural env vars, PATH entries and binaries against
# env-contract.json. Optionally apply safe defaults and invoke known heals.
#
# Usage:
#   doctor.sh             # --check mode (default): report drift, exit 1 if required missing
#   doctor.sh --check     # same as default
#   doctor.sh --fix       # apply defaults to current session, run heals; exit 0 if recoverable
#   doctor.sh --verbose   # show every check (combinable with --check / --fix)
#
# Resolves the contract from $DOTFILES_DIR/env-contract.json, falling back
# to the repo root next to this script. Mirrors scripts/doctor.ps1.

set -euo pipefail

MODE="check"
VERBOSE=0
EXIT_CODE=0
ISSUES=0
APPLIED=0

for arg in "$@"; do
    case "$arg" in
        --check)   MODE="check" ;;
        --fix)     MODE="fix" ;;
        --verbose) VERBOSE=1 ;;
        -h|--help)
            sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) echo "doctor: unknown arg: $arg" >&2; exit 2 ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
CONTRACT=""
for candidate in \
    "${DOTFILES_DIR:-$HOME/.dotfiles}/env-contract.json" \
    "$SCRIPT_DIR/../env-contract.json"
do
    if [ -f "$candidate" ]; then
        CONTRACT="$candidate"
        break
    fi
done

if [ -z "$CONTRACT" ]; then
    echo "doctor: env-contract.json not found (looked in DOTFILES_DIR and repo root)" >&2
    exit 2
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "doctor: jq not found on PATH (required to parse env-contract.json)" >&2
    exit 2
fi

# Helpers. Each call also bumps per-section counters so we can print a
# one-line section summary in non-verbose mode (otherwise the section
# header appears with no body when every item passes -- looks broken).
SEC_OK=0; SEC_WARN=0; SEC_FAIL=0; SEC_INFO=0
section_reset() { SEC_OK=0; SEC_WARN=0; SEC_FAIL=0; SEC_INFO=0; }
section_summary() {
    local total=$((SEC_OK + SEC_WARN + SEC_FAIL + SEC_INFO))
    if [ "$total" -eq 0 ]; then return 0; fi
    if [ "$SEC_WARN" -eq 0 ] && [ "$SEC_FAIL" -eq 0 ]; then
        if [ "$SEC_INFO" -gt 0 ]; then
            printf '  (%d checks: %d ok, %d info)\n' "$total" "$SEC_OK" "$SEC_INFO"
        else
            printf '  (%d checks, all ok)\n' "$total"
        fi
    else
        printf '  (%d total: %d ok, %d warn, %d fail)\n' "$total" "$SEC_OK" "$SEC_WARN" "$SEC_FAIL"
    fi
}
log_pass() { SEC_OK=$((SEC_OK + 1)); [ "$VERBOSE" -eq 1 ] && printf '  [ok]   %s\n' "$1"; return 0; }
log_warn() { SEC_WARN=$((SEC_WARN + 1)); printf '  [warn] %s\n' "$1"; ISSUES=$((ISSUES + 1)); }
log_fail() { SEC_FAIL=$((SEC_FAIL + 1)); printf '  [fail] %s\n' "$1"; ISSUES=$((ISSUES + 1)); EXIT_CODE=1; }
log_fix()  { printf '  [fix]  %s\n' "$1"; APPLIED=$((APPLIED + 1)); }
log_info() { SEC_INFO=$((SEC_INFO + 1)); printf '  [info] %s\n' "$1"; }

expand_path() {
    # Expand $HOME and ${HOME} in a string; leave other vars alone.
    printf '%s' "${1//\$HOME/$HOME}" | sed 's|${HOME}|'"$HOME"'|g'
}

echo "doctor [$MODE] using $CONTRACT"

# 1. Env vars
echo
echo "Environment variables:"
section_reset
# Use '|' as field separator: bash collapses consecutive whitespace IFS
# characters (including tab) even when set explicitly, which would erase
# the empty `required_on` column. '|' is non-whitespace, so empty fields
# are preserved by `read`.
while IFS='|' read -r name required required_on default validation; do
    # Skip vars scoped to the other OS
    if [ "$required_on" = "windows" ]; then
        log_pass "$name (windows-scoped, skipped on Linux)"
        continue
    fi
    current="${!name:-}"
    if [ -z "$current" ]; then
        if [ "$default" != "null" ] && [ -n "$default" ]; then
            expanded=$(expand_path "$default")
            if [ "$MODE" = "fix" ]; then
                export "$name=$expanded"
                log_fix "$name: applied default '$expanded' to current session (add to profile to persist)"
            else
                if [ "$required" = "true" ] || [ "$required_on" = "linux" ]; then
                    log_warn "$name unset (required); default available: '$expanded' -- run with --fix or set in profile"
                else
                    log_warn "$name unset; default '$expanded' would be applied with --fix"
                fi
            fi
            current="$expanded"
        else
            if [ "$required" = "true" ] || [ "$required_on" = "linux" ]; then
                log_fail "$name unset and no default available (required)"
                continue
            else
                log_pass "$name unset (optional, no default)"
                continue
            fi
        fi
    fi
    # Validation
    if [ "$validation" = "path_exists" ]; then
        if [ -d "$current" ]; then
            log_pass "$name=$current (path exists)"
        else
            log_fail "$name=$current (path does not exist)"
        fi
    else
        log_pass "$name=$current"
    fi
done < <(jq -r '.env_vars[] | [.name, (.required // false), (.required_on // ""), (.default.linux // "null"), (.validation // "")] | join("|")' "$CONTRACT")

section_summary

# 2. PATH entries
echo
echo "PATH entries:"
section_reset
while IFS= read -r entry; do
    expanded=$(expand_path "$entry")
    case ":$PATH:" in
        *":$expanded:"*) log_pass "$expanded in PATH" ;;
        *) log_warn "$expanded not in PATH -- check shell profile" ;;
    esac
done < <(jq -r '.required_path_entries.linux[]' "$CONTRACT")
section_summary

# 3. Required binaries (with optional version pinning)
echo
echo "Required binaries:"
section_reset
# Same '|' separator rationale as the env_vars loop above: '\t' would
# collapse empty optional columns (min_version, version_pattern).
while IFS='|' read -r name min_version version_pattern; do
    if ! command -v "$name" >/dev/null 2>&1; then
        log_fail "$name missing"
        continue
    fi
    if [ -z "$min_version" ]; then
        log_pass "$name on PATH"
        continue
    fi
    # Version check: run `<name> --version`, apply pattern, compare with sort -V.
    raw=$("$name" --version 2>&1 | head -1)
    if [[ "$raw" =~ $version_pattern ]]; then
        actual="${BASH_REMATCH[1]}"
        if [ "$(printf '%s\n%s\n' "$min_version" "$actual" | sort -V | head -1)" = "$min_version" ]; then
            log_pass "$name $actual (>= $min_version)"
        else
            log_fail "$name $actual is older than minimum $min_version"
        fi
    else
        log_warn "$name on PATH but version unparseable: '$raw' (pattern: $version_pattern)"
    fi
done < <(jq -r '.required_binaries[] | [.name, (.min_version // ""), (.version_pattern // "")] | join("|")' "$CONTRACT")
section_summary

# 4. Optional binaries
echo
echo "Optional binaries:"
section_reset
while IFS=$'\t' read -r name purpose; do
    if command -v "$name" >/dev/null 2>&1; then
        log_pass "$name on PATH ($purpose)"
    else
        log_info "$name missing -- $purpose"
    fi
done < <(jq -r '.optional_binaries[] | [.name, .purpose] | @tsv' "$CONTRACT")
# Optional-binaries section uses log_info for misses (not warn/fail), so
# count those manually so the section summary reflects them.
section_summary

# 5. --fix mode: invoke known heals
if [ "$MODE" = "fix" ]; then
    echo
    echo "Heal scripts:"
    HEAL="$SCRIPT_DIR/claude-mem-heal.sh"
    if [ -x "$HEAL" ]; then
        if heal_out=$(bash "$HEAL" 2>&1) && [ -n "$heal_out" ]; then
            printf '  [fix]  claude-mem-heal:\n%s\n' "$heal_out" | sed 's/^/         /'
            APPLIED=$((APPLIED + 1))
        else
            log_pass "claude-mem-heal: nothing to heal"
        fi
    else
        log_info "claude-mem-heal.sh not deployed (skipping)"
    fi
fi

echo
echo "Summary: $ISSUES issue(s), $APPLIED action(s) applied"
exit $EXIT_CODE
