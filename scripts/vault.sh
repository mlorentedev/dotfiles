#!/usr/bin/env bash

# vault.sh: Thin dispatcher for vault tooling (REFACTOR-005).
#
# The three vault scripts (vault-health.sh, vault-maintenance-weekly.sh,
# check-md-escapes.sh) do genuinely independent things and share almost no
# code, so they stay separate on disk. This dispatcher only adds a single
# discoverable entry point: `vault <subcommand>`. Each backing script remains
# runnable standalone — the dispatcher just `exec`s into it.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

usage() {
    cat <<'EOF'
Usage: vault <subcommand> [args...]

Subcommands:
  health [-v|--verbose] [--vault NAME]
      Vault health report: orphans, dead-ends, tags, frontmatter coverage,
      working-tree integrity. Requires Obsidian GUI running.

  maintenance
      Weekly automated maintenance: knowledge-crystallize + health + desktop
      notification. Logs to ~/.local/share/vault-maintenance/latest.log.
      Normally invoked via cron (Sundays 10:07).

  check-escapes <path>...
      Scan markdown files for literal '\n' corruption (Hive vault_patch bug).
      Path may be a file or directory; directories scanned recursively.

  check-tasks <tasks-file>...
      Scan task files for backlog drift: duplicate ticket IDs and status
      contradictions (same ID marked both [ ] and [x]). One ticket = one entry.

  help, -h, --help
      Show this message.

Each subcommand is also runnable standalone (vault-health.sh,
vault-maintenance-weekly.sh, check-md-escapes.sh, check-backlog-integrity.sh) —
this dispatcher provides a single discoverable entry point without changing the
underlying scripts.
EOF
}

if [ "$#" -eq 0 ]; then
    usage >&2
    exit 2
fi

sub="$1"
shift

case "$sub" in
    health)
        exec "$SCRIPT_DIR/vault-health.sh" "$@"
        ;;
    maintenance|weekly)
        exec "$SCRIPT_DIR/vault-maintenance-weekly.sh" "$@"
        ;;
    check-escapes|check)
        exec "$SCRIPT_DIR/check-md-escapes.sh" "$@"
        ;;
    check-tasks|tasks)
        exec "$SCRIPT_DIR/check-backlog-integrity.sh" "$@"
        ;;
    help|-h|--help)
        usage
        exit 0
        ;;
    *)
        printf 'vault: unknown subcommand: %s\n\n' "$sub" >&2
        usage >&2
        exit 2
        ;;
esac
