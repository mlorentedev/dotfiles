#!/usr/bin/env bash
# ai/hermes/setup.sh — remote provisioner for the Hermes agent (HERMES-001).
#
# Hermes is a remote ops agent (Debian 13 on NaN infra, Telegram gateway). This
# script is NOT the Hermes product installer (that is Nous Research's
# `curl … install.sh` + `hermes setup --portal`); it layers the vault/Hive
# integration on top of an already-installed Hermes. A fresh box curls this.
#
# Properties: idempotent, non-interactive (safe under `curl … | bash`). It runs
# as root on the NaN box and MAY apt-install the `cron` package if absent. It
# touches NONE of the local-deploy surface (setup-linux.sh, setup-windows.ps1,
# mcp-servers.json).
#
# Seams (env overrides; mirror the repo's VAULT_PATH / HARNESS_REPO_ROOT idiom):
#   HERMES_HOME         dir for ~/.hermes config + secrets   (default ~/.hermes)
#   HERMES_VAULT_PATH   persistent vault clone               (default ~/.local/state/hermes/vault)
#   HERMES_VAULT_REPO   vault git remote                     (default knowledge repo)
#   HERMES_VAULT_BRANCH vault default branch                 (default master)
#   HERMES_SETUP_DRY_RUN=1  skip network/system mutations (tests)
#
# Secrets: GITHUB_TOKEN_KNOWLEDGE comes from the environment or $HERMES_HOME/.env
# (never from sensitive/ — the remote box has neither the dotfiles repo nor the
# age key). The token is needed to clone the private vault and to push to
# 80_agents/. See specs/HERMES-001-add-hermes-agent/proposal.md.

set -euo pipefail

# --- Config (env-overridable seams) ---
HERMES_HOME="${HERMES_HOME:-$HOME/.hermes}"
HERMES_VAULT_PATH="${HERMES_VAULT_PATH:-$HOME/.local/state/hermes/vault}"
HERMES_VAULT_REPO="${HERMES_VAULT_REPO:-https://github.com/mlorentedev/knowledge.git}"
HERMES_VAULT_BRANCH="${HERMES_VAULT_BRANCH:-master}"
DRY_RUN="${HERMES_SETUP_DRY_RUN:-0}"

AGENT_DIR_REL="80_agents/hermes-nan"

log() { printf '[hermes-setup] %s\n' "$*"; }
die() { printf '[hermes-setup] ERROR: %s\n' "$*" >&2; exit 1; }

# Git invocation that supplies the token via a credential helper sourcing
# $HERMES_HOME/.env at call time, so the token never lands in .git/config. Used
# for the first clone (ensure_hermes_home populates .env before this runs).
git_token() {
    git -c "credential.helper=!f(){ . $HERMES_HOME/.env 2>/dev/null; echo username=x-access-token; echo \"password=\$GITHUB_TOKEN_KNOWLEDGE\"; };f" "$@"
}

# --- 1. Prerequisites (fail fast) ---
check_prereqs() {
    command -v uv >/dev/null 2>&1 || die "uv not found. Install: https://docs.astral.sh/uv/"

    # Token: environment wins; otherwise source $HERMES_HOME/.env; otherwise fail.
    if [ -z "${GITHUB_TOKEN_KNOWLEDGE:-}" ] && [ -f "$HERMES_HOME/.env" ]; then
        # shellcheck disable=SC1091
        . "$HERMES_HOME/.env"
    fi
    [ -n "${GITHUB_TOKEN_KNOWLEDGE:-}" ] || \
        die "GITHUB_TOKEN_KNOWLEDGE required (set in the environment or $HERMES_HOME/.env)"
    log "prerequisites ok (uv present, token resolved)"
}

# --- 2. ~/.hermes home + secrets file ---
ensure_hermes_home() {
    mkdir -p "$HERMES_HOME"
    chmod 700 "$HERMES_HOME"
    env_file="$HERMES_HOME/.env"
    if [ ! -f "$env_file" ]; then
        : > "$env_file"
    fi
    chmod 600 "$env_file"
    # Persist the token only if not already recorded (idempotent).
    if ! grep -q '^GITHUB_TOKEN_KNOWLEDGE=' "$env_file" 2>/dev/null; then
        printf 'GITHUB_TOKEN_KNOWLEDGE=%s\n' "$GITHUB_TOKEN_KNOWLEDGE" >> "$env_file"
        log "recorded GITHUB_TOKEN_KNOWLEDGE in $env_file"
    fi
}

# --- 3. Hive MCP launcher ---
install_hive() {
    if [ "$DRY_RUN" = 1 ]; then
        log "[dry-run] would: uv tool install --upgrade hive-vault"
        return 0
    fi
    uv tool install --upgrade hive-vault
    log "hive-vault installed/updated"
}

# --- 3.5 Preflight: verify the token actually reaches the vault remote ---
# Fail BEFORE any heavy mutation (clone, package install) rather than half-
# provisioning, so a bad/expired token is a clean abort, not a wedged box.
check_vault_access() {
    if [ "$DRY_RUN" = 1 ]; then
        log "[dry-run] would: verify vault remote access with the token"
        return 0
    fi
    if ! git_token ls-remote --heads "$HERMES_VAULT_REPO" >/dev/null 2>&1; then
        die "cannot reach vault remote ($HERMES_VAULT_REPO) with the provided token — check scope/expiry"
    fi
    log "vault remote reachable, token valid"
}

# --- 4. Vault clone (persistent path, idempotent) ---
ensure_vault_clone() {
    mkdir -p "$(dirname "$HERMES_VAULT_PATH")"
    if [ -d "$HERMES_VAULT_PATH/.git" ]; then
        if [ "$DRY_RUN" = 1 ]; then
            log "[dry-run] would: git -C $HERMES_VAULT_PATH pull --ff-only"
        else
            git_token -C "$HERMES_VAULT_PATH" pull --ff-only --quiet
            log "vault updated at $HERMES_VAULT_PATH"
        fi
    else
        if [ "$DRY_RUN" = 1 ]; then
            log "[dry-run] would: git clone $HERMES_VAULT_REPO -> $HERMES_VAULT_PATH"
        else
            git_token clone --branch "$HERMES_VAULT_BRANCH" "$HERMES_VAULT_REPO" "$HERMES_VAULT_PATH"
            log "vault cloned to $HERMES_VAULT_PATH"
        fi
    fi
    if [ -d "$HERMES_VAULT_PATH/.git" ]; then
        # Strip any token embedded in the remote URL (the existing box has
        # https://x-access-token:***@github.com/...). The credential helper
        # below supplies the token instead, keeping it out of .git/config.
        if git -C "$HERMES_VAULT_PATH" remote get-url origin 2>/dev/null | grep -q '@'; then
            git -C "$HERMES_VAULT_PATH" remote set-url origin "$HERMES_VAULT_REPO"
            log "stripped embedded token from origin remote URL"
        fi
        # Persistent credential helper that sources $HERMES_HOME/.env at call
        # time — works for the cron/non-interactive post-commit push too.
        git -C "$HERMES_VAULT_PATH" config credential.helper \
            "!f(){ . $HERMES_HOME/.env 2>/dev/null; echo username=x-access-token; echo \"password=\$GITHUB_TOKEN_KNOWLEDGE\"; };f"
    fi
}

# --- 5. Vault SSOT bootstrap (ensure the agent workspace context exists) ---
ensure_agent_workspace() {
    ctx="$HERMES_VAULT_PATH/$AGENT_DIR_REL/context.md"
    if [ -d "$HERMES_VAULT_PATH/.git" ] && [ ! -f "$ctx" ]; then
        log "warning: $AGENT_DIR_REL/context.md missing in vault (agent will bootstrap it)"
    fi
    # Ensure-local only: we do NOT commit here. The agent commits in its own flow
    # (write-only 80_agents/ per the commit policy).
}

# --- 6. Vault sync mechanism: robust cron pull + post-commit auto-push ---

# Robust auto-pull wrapper: aborts a conflicted rebase instead of leaving the
# clone wedged (a junior agent cannot recover a half-rebased repo). Paths are
# baked in because cron jobs run with no environment.
write_sync_script() {
    sync_script="$HERMES_HOME/vault-pull.sh"
    cat > "$sync_script" <<EOF
#!/bin/sh
# hermes-managed: robust vault auto-pull (abort rebase on conflict, never wedge)
cd "$HERMES_VAULT_PATH" || exit 0
if ! git pull --rebase --quiet 2>/dev/null; then
    git rebase --abort 2>/dev/null || true
    printf '%s vault pull conflict; rebase aborted\n' "\$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$HERMES_HOME/sync.log"
fi
EOF
    chmod +x "$sync_script"
}

ensure_sync() {
    # The cron PACKAGE may be absent on a minimal Debian box. Install it if so
    # (the box runs as root). The crontab ENTRY itself is userspace.
    if ! command -v crontab >/dev/null 2>&1; then
        if [ "$DRY_RUN" = 1 ]; then
            log "[dry-run] would: apt-get install -y cron"
        elif command -v apt-get >/dev/null 2>&1; then
            apt-get update -qq && apt-get install -y cron
            log "installed cron package"
        else
            log "warning: crontab not found and apt-get unavailable; skipping cron setup"
            return 0
        fi
    fi

    write_sync_script
    cron_line="*/5 * * * * $HERMES_HOME/vault-pull.sh"
    if [ "$DRY_RUN" = 1 ]; then
        log "[dry-run] would: install crontab entry + start cron"
    else
        # Rebuild the crontab: drop the stale pre-HERMES-001 inline /tmp pull
        # line and any prior wrapper entry, then add exactly one wrapper entry.
        # Idempotent: the end state is identical on every run.
        new_crontab="$(crontab -l 2>/dev/null | grep -vF 'cd /tmp/hermes-vault && git pull' | grep -vF "$HERMES_HOME/vault-pull.sh" || true)"
        { printf '%s\n' "$new_crontab"; printf '%s\n' "$cron_line"; } | grep -v '^[[:space:]]*$' | crontab -
        log "ensured vault-pull crontab entry (single; stale /tmp entry removed)"
        if command -v service >/dev/null 2>&1; then
            service cron start >/dev/null 2>&1 || true
        fi
    fi

    # Post-commit auto-push hook (idempotent; marker-guarded).
    hook="$HERMES_VAULT_PATH/.git/hooks/post-commit"
    if [ -d "$HERMES_VAULT_PATH/.git" ]; then
        if [ ! -f "$hook" ] || ! grep -q 'hermes-managed' "$hook" 2>/dev/null; then
            cat > "$hook" <<EOF
#!/bin/sh
# hermes-managed: auto-push after every commit
exec git push --quiet origin $HERMES_VAULT_BRANCH 2>&1
EOF
            chmod +x "$hook"
            log "installed post-commit auto-push hook"
        fi
    fi
}

# --- 7. Mechanical guardrails (Hermes commits via git CLI, so hooks fire) ---
# A junior agent needs a hard boundary, not an instruction. These local hooks
# live in Hermes's clone only (never tracked, never cloned) and cannot be
# bypassed by a normal commit/push.
install_guardrails() {
    [ -d "$HERMES_VAULT_PATH/.git" ] || return 0
    hooks="$HERMES_VAULT_PATH/.git/hooks"
    mkdir -p "$hooks"

    # pre-commit: reject any staged path outside the agent workspace, and any
    # token-like content (defense in depth even within the zone).
    cat > "$hooks/pre-commit" <<EOF
#!/bin/sh
# hermes-managed: write-zone + secret guard (HERMES-001)
zone="$AGENT_DIR_REL/"
outside=\$(git diff --cached --name-only | grep -v "^\$zone" || true)
if [ -n "\$outside" ]; then
    echo "pre-commit: refusing commit touching paths outside \$zone:" >&2
    echo "\$outside" >&2
    exit 1
fi
if git diff --cached -U0 | grep -qE '(ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|AGE-SECRET-KEY-1|AKIA[0-9A-Z]{16})'; then
    echo "pre-commit: refusing commit with token-like content" >&2
    exit 1
fi
exit 0
EOF
    chmod +x "$hooks/pre-commit"

    # pre-push: reject non-fast-forward (force) pushes that rewrite history.
    cat > "$hooks/pre-push" <<'EOF'
#!/bin/sh
# hermes-managed: reject non-fast-forward (force) pushes (HERMES-001)
z=0000000000000000000000000000000000000000
while read -r _lref lsha _rref rsha; do
    [ "$lsha" = "$z" ] && continue
    [ "$rsha" = "$z" ] && continue
    if ! git merge-base --is-ancestor "$rsha" "$lsha" 2>/dev/null; then
        echo "pre-push: refusing non-fast-forward (force) push" >&2
        exit 1
    fi
done
exit 0
EOF
    chmod +x "$hooks/pre-push"
    log "installed write-zone + secret + no-force guardrails"
}

# --- 8. Register the Hive MCP server with Hermes (native CLI) ---
# Hermes exposes `hermes mcp add`; we do NOT hand-edit config.yaml (which lives
# at /hermes-home/config.yaml and holds the product's own provider config).
# Idempotent via `hermes mcp list`. Non-fatal: the agent operates over git today.
register_hive_mcp() {
    hermes_bin="${HERMES_BIN:-}"
    if [ -z "$hermes_bin" ]; then
        if command -v hermes >/dev/null 2>&1; then
            hermes_bin="hermes"
        elif [ -x /opt/hermes/.venv/bin/hermes ]; then
            hermes_bin="/opt/hermes/.venv/bin/hermes"
        fi
    fi
    if [ -z "$hermes_bin" ]; then
        log "warning: hermes binary not found; skipping Hive MCP registration"
        return 0
    fi
    if [ "$DRY_RUN" = 1 ]; then
        log "[dry-run] would: $hermes_bin mcp add hive --command uvx --args hive-vault --env HIVE_VAULT_PATH=$HERMES_VAULT_PATH"
        return 0
    fi
    if "$hermes_bin" mcp list 2>/dev/null | grep -qi hive; then
        log "Hive MCP already registered"
        return 0
    fi
    # hive-vault reads HIVE_VAULT_PATH from its env; hermes injects it at start.
    if "$hermes_bin" mcp add hive --command uvx --args hive-vault \
        --env "HIVE_VAULT_PATH=$HERMES_VAULT_PATH"; then
        log "registered Hive MCP server (HIVE_VAULT_PATH=$HERMES_VAULT_PATH)"
    else
        log "warning: 'hermes mcp add hive' failed"
    fi
}

main() {
    check_prereqs
    ensure_hermes_home
    check_vault_access
    install_hive
    ensure_vault_clone
    ensure_agent_workspace
    ensure_sync
    install_guardrails
    register_hive_mcp
    log "done. vault: $HERMES_VAULT_PATH | home: $HERMES_HOME"
}

main "$@"
