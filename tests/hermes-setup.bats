#!/usr/bin/env bats
# Tests for ai/hermes/setup.sh — the Hermes remote provisioner (HERMES-001).
#
# Hermetic: setup.sh runs under a stub seam — redirected HERMES_HOME /
# HERMES_VAULT_PATH (into the per-test tmpdir) and stubbed external commands
# (uv/git/crontab) on a restricted PATH. The guardrail tests provision into a
# REAL git repo (so the installed hooks can be exercised with real git), while
# setup.sh itself still runs against the stubbed git for clone/pull.

setup() {
    DOTFILES_DIR="$BATS_TEST_DIRNAME/.."
    SETUP="$DOTFILES_DIR/ai/hermes/setup.sh"
    export HERMES_HOME="$BATS_TEST_TMPDIR/dot-hermes"
    export HERMES_VAULT_PATH="$BATS_TEST_TMPDIR/state/vault"
    STUB="$BATS_TEST_TMPDIR/stub-bin"
    mkdir -p "$STUB"
}

# Simple stub on PATH that succeeds and logs its argv.
stub() {
    {
        printf '#!/usr/bin/env bash\n'
        printf 'printf "%%s\\n" "%s $*" >> "%s/calls.log"\n' "$1" "$BATS_TEST_TMPDIR"
        printf 'exit 0\n'
    } > "$STUB/$1"
    chmod +x "$STUB/$1"
}

# git stub: ignores -c/-C options, dispatches on the subcommand. ls-remote
# returns $GIT_STUB_LSREMOTE_RC (default 0) so the auth-check path is testable.
stub_git() {
    cat > "$STUB/git" <<'EOF'
#!/usr/bin/env bash
sub=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        -c|-C) shift 2; continue ;;
        -*) shift; continue ;;
        *) sub="$1"; break ;;
    esac
done
case "$sub" in
    ls-remote) exit "${GIT_STUB_LSREMOTE_RC:-0}" ;;
    *) exit 0 ;;
esac
EOF
    chmod +x "$STUB/git"
}

# Stateful crontab stub: `-l` prints the store, `crontab -` installs from stdin.
stub_crontab() {
    export CRONTAB_STORE="$BATS_TEST_TMPDIR/crontab.store"
    cat > "$STUB/crontab" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "-l" ]; then cat "$CRONTAB_STORE" 2>/dev/null; exit 0; fi
cat > "$CRONTAB_STORE"
EOF
    chmod +x "$STUB/crontab"
}

# hermes CLI stub: logs argv to $HERMES_CALLS; `mcp list` prints nothing (so the
# server is treated as unregistered and the add path runs).
stub_hermes() {
    cat > "$STUB/hermes" <<'EOF'
#!/usr/bin/env bash
printf 'hermes %s\n' "$*" >> "$HERMES_CALLS"
exit 0
EOF
    chmod +x "$STUB/hermes"
}

# Provision into a REAL git repo so the installed hooks are exercisable with real
# git; setup.sh runs with stubbed git/crontab/uv (no real network/cron).
provision() {
    stub uv; stub_git; stub_crontab
    export GITHUB_TOKEN_KNOWLEDGE=dummy
    git init -q "$HERMES_VAULT_PATH"
    git -C "$HERMES_VAULT_PATH" config user.email t@t
    git -C "$HERMES_VAULT_PATH" config user.name t
    PATH="$STUB:/usr/bin:/bin" bash "$SETUP" >/dev/null 2>&1
}

@test "fail fast: non-zero exit + message when uv is missing" {
    export HERMES_SETUP_DRY_RUN=1
    export GITHUB_TOKEN_KNOWLEDGE=dummy   # isolate the uv check
    PATH="$STUB:/usr/bin:/bin" run bash "$SETUP"
    [ "$status" -ne 0 ]
    [[ "$output" == *"uv"* ]]
}

@test "fail fast: non-zero exit + message when GITHUB_TOKEN_KNOWLEDGE is missing" {
    export HERMES_SETUP_DRY_RUN=1
    stub uv
    unset GITHUB_TOKEN_KNOWLEDGE
    PATH="$STUB:/usr/bin:/bin" run bash "$SETUP"   # HERMES_HOME has no .env
    [ "$status" -ne 0 ]
    [[ "$output" == *"GITHUB_TOKEN_KNOWLEDGE"* ]]
}

@test "preflight: aborts when the token cannot reach the vault remote" {
    stub uv
    stub_git
    export GITHUB_TOKEN_KNOWLEDGE=dummy
    export GIT_STUB_LSREMOTE_RC=1     # ls-remote fails -> token invalid/unreachable
    mkdir -p "$HERMES_VAULT_PATH/.git/hooks"
    PATH="$STUB:/usr/bin:/bin" run bash "$SETUP"
    [ "$status" -ne 0 ]
    [[ "$output" == *"cannot reach vault remote"* ]]
}

@test "guard: setup.sh touches none of the local-deploy surface (comments excluded)" {
    ! grep -vE '^[[:space:]]*#' "$SETUP" | grep -qE 'setup-linux\.sh|setup-windows\.ps1|mcp-servers\.json'
}

@test "sync wrapper aborts a conflicted rebase (never wedges the clone)" {
    export HERMES_SETUP_DRY_RUN=1
    stub uv
    export GITHUB_TOKEN_KNOWLEDGE=dummy
    PATH="$STUB:/usr/bin:/bin" run bash "$SETUP"
    [ "$status" -eq 0 ]
    [ -x "$HERMES_HOME/vault-pull.sh" ]
    grep -qF 'rebase --abort' "$HERMES_HOME/vault-pull.sh"
}

@test "idempotent (full, DRY_RUN=0): two real runs are a no-op — no dupes" {
    stub uv
    stub_git
    stub_crontab
    export GITHUB_TOKEN_KNOWLEDGE=dummy
    mkdir -p "$HERMES_VAULT_PATH/.git/hooks"   # pretend the clone exists -> pull path

    PATH="$STUB:/usr/bin:/bin" run bash "$SETUP"
    [ "$status" -eq 0 ]
    PATH="$STUB:/usr/bin:/bin" run bash "$SETUP"
    [ "$status" -eq 0 ]

    [ "$(grep -c '^GITHUB_TOKEN_KNOWLEDGE=' "$HERMES_HOME/.env")" -eq 1 ]
    [ "$(stat -c '%a' "$HERMES_HOME/.env")" = "600" ]
    [ "$(grep -c 'vault-pull.sh' "$CRONTAB_STORE")" -eq 1 ]
    [ "$(grep -c 'hermes-managed' "$HERMES_VAULT_PATH/.git/hooks/post-commit")" -eq 1 ]
    [ -x "$HERMES_HOME/vault-pull.sh" ]
}

@test "guardrail: pre-commit rejects a commit outside the write-zone" {
    provision
    [ -x "$HERMES_VAULT_PATH/.git/hooks/pre-commit" ]
    mkdir -p "$HERMES_VAULT_PATH/00_meta"
    echo x > "$HERMES_VAULT_PATH/00_meta/foo.md"
    git -C "$HERMES_VAULT_PATH" add 00_meta/foo.md
    run git -C "$HERMES_VAULT_PATH" commit -m "out of zone"
    [ "$status" -ne 0 ]
    [[ "$output" == *"80_agents"* || "$output" == *"outside"* ]]
}

@test "guardrail: pre-commit allows a commit inside the write-zone" {
    provision
    mkdir -p "$HERMES_VAULT_PATH/80_agents/hermes-nan/sessions"
    echo ok > "$HERMES_VAULT_PATH/80_agents/hermes-nan/sessions/x.md"
    git -C "$HERMES_VAULT_PATH" add 80_agents/hermes-nan/sessions/x.md
    run git -C "$HERMES_VAULT_PATH" commit -m "in zone"
    [ "$status" -eq 0 ]
}

@test "guardrail: pre-commit rejects token-like content inside the zone" {
    provision
    mkdir -p "$HERMES_VAULT_PATH/80_agents/hermes-nan"
    printf 'leak: ghp_%s\n' "AbCdEfGh0123456789AbCdEfGhIj" > "$HERMES_VAULT_PATH/80_agents/hermes-nan/leak.md"
    git -C "$HERMES_VAULT_PATH" add 80_agents/hermes-nan/leak.md
    run git -C "$HERMES_VAULT_PATH" commit -m "leak"
    [ "$status" -ne 0 ]
    [[ "$output" == *"token-like"* ]]
}

@test "guardrail: pre-push hook installed and rejects non-fast-forward" {
    provision
    [ -x "$HERMES_VAULT_PATH/.git/hooks/pre-push" ]
    grep -qF 'non-fast-forward' "$HERMES_VAULT_PATH/.git/hooks/pre-push"
}

@test "registers Hive MCP via hermes mcp add with HIVE_VAULT_PATH env" {
    export HERMES_CALLS="$BATS_TEST_TMPDIR/hermes.calls"
    stub uv; stub_git; stub_crontab; stub_hermes
    export GITHUB_TOKEN_KNOWLEDGE=dummy
    mkdir -p "$HERMES_VAULT_PATH/.git/hooks"
    PATH="$STUB:/usr/bin:/bin" run bash "$SETUP"
    [ "$status" -eq 0 ]
    grep -qF 'mcp add hive' "$HERMES_CALLS"
    grep -qF "HIVE_VAULT_PATH=$HERMES_VAULT_PATH" "$HERMES_CALLS"
}
