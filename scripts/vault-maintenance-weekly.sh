#!/usr/bin/env bash
# vault-maintenance-weekly.sh: Automated weekly vault maintenance
#
# Runs dotf vault crystallize --all + dotf vault health.
# Logs results and sends desktop notification (best-effort).
#
# Deployed to crontab by setup-linux.sh: Sundays 10:00 AM
# Usage: ./scripts/vault-maintenance-weekly.sh

set -euo pipefail

# cron runs with a minimal PATH that does not include ~/.local/bin (install.sh's
# install target), unlike an interactive shell. The old knowledge-crystallize.sh and
# vault-health.sh calls used absolute paths and never needed this; `dotf` is invoked by
# bare name below, so without this both steps silently no-op under `|| true` every Sunday.
export PATH="$HOME/.local/bin:$PATH"

LOG_DIR="$HOME/.local/share/vault-maintenance"
LOG_FILE="$LOG_DIR/latest.log"

mkdir -p "$LOG_DIR"

{
    printf '=== Vault Maintenance: %s ===\n\n' "$(date)"

    printf '%s\n' '--- dotf vault crystallize --all ---'
    dotf vault crystallize --all 2>&1 || true
    printf '\n'

    printf '%s\n' '--- vault-health ---'
    dotf vault health 2>&1 || true
    printf '\n'

    printf '=== Done: %s ===\n' "$(date)"
} > "$LOG_FILE" 2>&1

# Count issues for notification
issues=$(grep -ciE "warning|fail|action|stale" "$LOG_FILE" 2>/dev/null || printf '0')

# Desktop notification (best-effort -- may fail in headless/cron context)
if command -v notify-send >/dev/null 2>&1; then
    export DISPLAY="${DISPLAY:-:0}"
    export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=/run/user/$(id -u)/bus}"
    if [ "$issues" -gt 0 ]; then
        notify-send -u normal "Vault Maintenance" "$issues potential issues. Run /insights on active projects." 2>/dev/null || true
    else
        notify-send -u low "Vault Maintenance" "All clean. No action needed." 2>/dev/null || true
    fi
fi

printf 'Log written to %s\n' "$LOG_FILE"
