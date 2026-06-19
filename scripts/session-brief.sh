#!/bin/sh

# session-brief.sh: agent-agnostic session-brief core (ADR-023, HARNESS-026).
#
# Owns the agent-INDEPENDENT vault signals that used to live inline in the
# Claude SessionStart hook: vault detection, vault-health, active/archived spec
# counts, and vault-baseline integrity. It has two uses:
#
#   (1) Sourced as a library — set SESSION_BRIEF_LIB=1 before sourcing to get
#       the find_vault_root + sb_* emitters WITHOUT running main. Each sb_*
#       emitter writes its block to stdout with a single LEADING newline (or
#       nothing when the signal is absent), so a caller can append it verbatim:
#           CONTEXT_LINES="$CONTEXT_LINES$(sb_specs "$cwd")"
#       This is how scripts/claude-session-start.sh (the Claude adapter) keeps
#       its emitted additionalContext byte-for-byte identical.
#
#   (2) Standalone — session-brief.sh --format=stdout|markdown emits the full
#       brief (headline, vault-health, specs, baseline) for file-based agents
#       (opencode/agy/copilot) that the HARNESS-001 compiler injects out-of-band.
#
# POSIX sh; no bashisms (C5 cross-OS). Emitters use the func() ( … ) subshell
# form so internal variables never leak into a sourcing shell and no `local` is
# needed. Dynamic content is always passed as a printf %s argument, never as the
# format string, so vault text containing % or \ is emitted literally.

# --- Signal emitters (sourceable) -------------------------------------------

# find_vault_root <dir>: walk up from <dir> to the nearest .obsidian/ vault root.
# Echoes the root and returns 0, or returns 1 if none found.
find_vault_root() (
    dir="$1"
    while [ "$dir" != "/" ]; do
        if [ -d "$dir/.obsidian" ]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
)

# sb_vault_detect <vault_root>: the "Obsidian vault detected" headline, with NO
# surrounding newlines (the caller owns the blank-line framing). Empty if no root.
sb_vault_detect() (
    vault_root="$1"
    [ -n "$vault_root" ] || return 0
    printf 'Obsidian vault detected: %s (%s)' "$(basename "$vault_root")" "$vault_root"
)

# sb_vault_health <vault_root> <vault_name> <script_dir>: run vault-health.sh and
# format the GUI-down / pass / fail / not-installed cases. Leading newline.
sb_vault_health() (
    vault_root="$1"
    vault_name="$2"
    script_dir="$3"
    vault_health="$script_dir/vault-health.sh"

    if [ -x "$vault_health" ]; then
        health_exit=0
        health_output=$(
            VAULT_DIR="$vault_root" VAULT_NAME="$vault_name" \
            bash "$vault_health" 2>&1
        ) || health_exit=$?

        if [ "$health_exit" -eq 2 ]; then
            # GUI down: GUI-dependent checks skipped, but the git-based integrity
            # check ran anyway — surface any FAIL lines.
            integrity_fails=$(echo "$health_output" | sed 's/\x1b\[[0-9;]*m//g' | grep 'FAIL' || echo "")
            if [ -n "$integrity_fails" ]; then
                printf '\nObsidian GUI not running — GUI-dependent checks skipped. Integrity issues found:\n%s' "$integrity_fails"
            else
                printf '\nObsidian GUI not running — vault health skipped. Run '\''vault-health.sh'\'' manually when GUI is up.'
            fi
        elif [ "$health_exit" -eq 0 ]; then
            printf '\nVault health: ALL CHECKS PASSED'
        else
            summary=$(echo "$health_output" | sed 's/\x1b\[[0-9;]*m//g' | grep '^Results:' || echo "")
            failures=$(echo "$health_output" | sed 's/\x1b\[[0-9;]*m//g' | grep 'FAIL' || echo "")
            printf '\nVault health: %s\nIssues found:\n%s' "$summary" "$failures"
        fi
    else
        printf '\nvault-health.sh not found at %s — run dotfiles setup to install.' "$vault_health"
    fi
)

# sb_specs <cwd>: active/archived spec counts under <cwd>/specs, flagging any
# spec with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags. Leading newline,
# or nothing if there is no specs/ dir or it is entirely empty.
sb_specs() (
    cwd="$1"
    specs_dir="$cwd/specs"
    [ -d "$specs_dir" ] || return 0

    active_count=0
    drafts_specs=""
    for d in "$specs_dir"/*/; do
        [ -d "$d" ] || continue
        name=$(basename "$d")
        [ "$name" = "archive" ] && continue
        active_count=$((active_count + 1))
        if grep -rlE '\[AGENT-DRAFT\]|\[AGENT-SUGGESTION\]' "$d" >/dev/null 2>&1; then
            drafts_specs="$drafts_specs $name"
        fi
    done

    archive_count=0
    if [ -d "$specs_dir/archive" ]; then
        for d in "$specs_dir/archive"/*/; do
            [ -d "$d" ] || continue
            archive_count=$((archive_count + 1))
        done
    fi

    if [ "$active_count" -eq 0 ] && [ "$archive_count" -eq 0 ]; then
        return 0
    fi

    msg="[specs] $active_count active, $archive_count archived"
    if [ -n "$drafts_specs" ]; then
        drafts_specs="${drafts_specs# }"
        draft_count=$(printf '%s' "$drafts_specs" | wc -w | tr -d ' ')
        msg="$msg — $draft_count with unresolved [AGENT-DRAFT]/[AGENT-SUGGESTION] tags:"
        for name in $drafts_specs; do
            msg="$msg
  - $name"
        done
    fi

    printf '\n%s' "$msg"
)

# sb_vault_baseline <vault_root>: defensive check that every 00_meta/skills/<n>/
# has a non-empty SKILL.md and the static critical files exist (SDD-017b).
# Leading newline when issues found, otherwise nothing.
sb_vault_baseline() (
    vault_root="$1"
    [ -n "$vault_root" ] || return 0
    [ -d "$vault_root/00_meta" ] || return 0

    issues=""
    if [ -d "$vault_root/00_meta/skills" ]; then
        for skill_dir in "$vault_root/00_meta/skills"/*/; do
            [ -d "$skill_dir" ] || continue
            skill_md="${skill_dir}SKILL.md"
            skill_name=$(basename "$skill_dir")
            if [ ! -f "$skill_md" ]; then
                issues="${issues}
        MISSING: 00_meta/skills/${skill_name}/SKILL.md (skill dir exists but SKILL.md gone)"
            elif [ ! -s "$skill_md" ]; then
                issues="${issues}
        EMPTY: 00_meta/skills/${skill_name}/SKILL.md (0 bytes — likely truncated)"
            fi
        done
    fi

    # AGENTS.md is intentionally NOT required here: per ADR-010 it is single-SSOT
    # in dotfiles and deployed (not duplicated) to the vault (BUG-023).
    for cf in "00_meta/patterns/_index.md" "00_meta/skills/README.md" "README.md"; do
        if [ ! -f "$vault_root/$cf" ]; then
            issues="${issues}
        MISSING: $cf (canonical file)"
        fi
    done

    if [ -n "$issues" ]; then
        printf '\nVault baseline FAIL — canonical artifacts missing (likely auto-committed delete):%s\n        Recovery: cd %s; git log --diff-filter=D --name-only --pretty=format: -5 | sort -u; git show <commit>:<path> > <path>' "$issues" "$vault_root"
    fi
)

# --- Standalone runner ------------------------------------------------------

sb_usage() {
    printf 'usage: session-brief.sh --format=stdout|markdown\n' >&2
}

sb_main() {
    format=""
    while [ $# -gt 0 ]; do
        case "$1" in
            --format=*) format="${1#--format=}" ;;
            --format)   shift; format="${1:-}" ;;
            -h|--help)  sb_usage; return 0 ;;
            *) printf 'session-brief.sh: unknown argument: %s\n' "$1" >&2
               sb_usage; return 2 ;;
        esac
        shift
    done

    case "$format" in
        stdout|markdown) ;;
        *) printf 'session-brief.sh: --format must be stdout or markdown (got: %s)\n' "$format" >&2
           sb_usage; return 2 ;;
    esac

    cwd="${SESSION_BRIEF_CWD:-$(pwd)}"
    script_dir=$(cd -- "$(dirname -- "$0")" && pwd)
    vault_root=$(find_vault_root "$cwd") || vault_root=""
    vault_name=""
    [ -n "$vault_root" ] && vault_name=$(basename "$vault_root")

    # Canonical agnostic order: headline, vault-health, specs, baseline.
    brief="$(sb_vault_detect "$vault_root")"
    brief="$brief$(sb_vault_health "$vault_root" "$vault_name" "$script_dir")"
    brief="$brief$(sb_specs "$cwd")"
    brief="$brief$(sb_vault_baseline "$vault_root")"
    # Drop a leading blank line when there was no headline (no-vault CWD).
    brief=$(printf '%s' "$brief" | sed '1{/^$/d;}')

    # The markdown branch prints a literal ``` fence — the backticks are literal
    # output, not a command substitution.
    # shellcheck disable=SC2016
    case "$format" in
        stdout)   printf '%s\n' "$brief" ;;
        markdown) printf '```text\n%s\n```\n' "$brief" ;;
    esac
}

# Run main only when executed directly; skip when sourced as a library.
case "${SESSION_BRIEF_LIB:-}" in
    "") sb_main "$@" ;;
esac
