#!/usr/bin/env bash
# Shared case runner for the vault-health golden corpus (CLI-021 increment 2).
#
# ONE runner, used by BOTH capture.sh and the .bats suites, for the same reason
# the crystallize corpus does it: if capture and verify normalised differently
# the goldens would be meaningless, and the drift would be invisible because
# both sides would stay self-consistent.
#
# What makes THIS corpus different from crystallize's: four of the script's
# seven sections shell out to `obsidian`, which talks over IPC to a running GUI.
# So the oracle is not just "what the script prints" — it is also "how the
# script invokes obsidian". The stub therefore logs its argv, and that log is a
# compared artefact. Without it the Go port could drift in the flags it passes
# while stdout stayed byte-identical.

# gvh_run_case CASE_DIR OUT_DIR
#   Builds an isolated vault + a stubbed `obsidian`, runs the implementation,
#   and writes normalised artefacts to OUT_DIR:
#       stdout        merged stdout+stderr, normalised
#       exit          the exit status
#       obsidian-argv every obsidian invocation, in order, normalised
gvh_run_case() {
    local case_dir="$1" out_dir="$2"

    local sandbox vault_dir stub_bin
    sandbox=$(mktemp -d)
    vault_dir="$sandbox/vault"
    stub_bin="$sandbox/bin"
    mkdir -p "$stub_bin"

    # A fixture tree must never contain a link out. Isolation is a claim that
    # needs its own assertion (docs/lessons.md, 2026-08-09) — this script family
    # nearly wrote through a symlinked "sandbox" once already.
    if [ -n "$(find "$case_dir" -type l 2>/dev/null)" ]; then
        printf 'gvh_run_case: fixture tree contains a symlink: %s\n' "$case_dir" >&2
        return 1
    fi

    if [ -d "$case_dir/vault" ]; then
        cp -r "$case_dir/vault" "$vault_dir"
    fi

    # ── the obsidian stub ────────────────────────────────────────────────────
    #
    # There is a REAL obsidian binary on a developer machine (~/.local/bin ->
    # an AppImage). Prepending the stub is not enough: the `obsidian absent`
    # case would find the real one, and any case could in principle launch the
    # actual GUI against the actual vault. So PATH is REPLACED, not extended,
    # and the replacement is then verified below rather than assumed.
    local obs_log="$sandbox/obsidian-argv.log"
    if [ ! -f "$case_dir/no-obsidian" ]; then
        {
            printf '#!/usr/bin/env bash\n'
            printf 'printf "%%s\\n" "$*" >> "%s"\n' "$obs_log"
            printf 'sub=""\n'
            printf 'for a in "$@"; do\n'
            printf '  case "$a" in vault|orphans|dead-ends|unresolved|tags) sub="$a"; break ;; esac\n'
            printf 'done\n'
            printf '[ -n "$sub" ] && [ -f "%s/stub/$sub" ] && cat "%s/stub/$sub"\n' "$case_dir" "$case_dir"
            printf 'exit 0\n'
        } > "$stub_bin/obsidian"
        chmod +x "$stub_bin/obsidian"
    fi

    local sandbox_path="$stub_bin:/usr/bin:/bin"
    # The isolation assertion. `obsidian` must resolve into the sandbox, or
    # nowhere at all for the absent case — never to the developer's AppImage.
    local resolved
    resolved=$(PATH="$sandbox_path" bash -c 'command -v obsidian' 2>/dev/null || true)
    if [ -f "$case_dir/no-obsidian" ]; then
        if [ -n "$resolved" ]; then
            printf 'gvh_run_case: PATH leak — obsidian still resolves to %s\n' "$resolved" >&2
            return 1
        fi
    elif [ "$resolved" != "$stub_bin/obsidian" ]; then
        printf 'gvh_run_case: PATH leak — obsidian resolved to %s, not the stub\n' "$resolved" >&2
        return 1
    fi

    # ── git state for section 1 ──────────────────────────────────────────────
    # Absent  -> not a repo, the section skips.
    # clean   -> a committed tree.
    # deleted -> committed, then a file removed from disk but not from HEAD,
    #            which is the 2026-05-13 incident this check exists for.
    local gitmode=""
    [ -f "$case_dir/gitmode" ] && gitmode=$(cat "$case_dir/gitmode")
    if [ -n "$gitmode" ] && [ -d "$vault_dir" ]; then
        git -C "$vault_dir" init -q -b main
        git -C "$vault_dir" -c user.email=t@t -c user.name=t add -A
        git -C "$vault_dir" -c user.email=t@t -c user.name=t commit -q -m fixture
        if [ "$gitmode" = "deleted" ]; then
            rm -f "$vault_dir/$(cat "$case_dir/deleted-file")"
        fi
    fi

    local -a args=()
    if [ -f "$case_dir/args" ]; then
        # Fixture-authored, not user input; word-splitting is the point.
        # shellcheck disable=SC2207
        args=($(cat "$case_dir/args"))
    fi

    mkdir -p "$out_dir"

    local raw="$sandbox/raw.out" rc=0
    local mode="${GVH_IMPL_MODE:-shell}"
    case "$mode" in
        shell)
            # `|| rc=$?` rather than relying on the caller's flags — the lesson
            # the reconciler cost us (docs/lessons.md, 2026-08-09).
            PATH="$sandbox_path" VAULT_DIR="$vault_dir" \
                bash "$GVH_ORACLE_SH" "${args[@]}" >"$raw" 2>&1 || rc=$?
            ;;
        go)
            PATH="$sandbox_path" VAULT_DIR="$vault_dir" \
                "$GVH_DOTF_BIN" vault health "${args[@]}" >"$raw" 2>&1 || rc=$?
            ;;
        *)
            printf 'gvh_run_case: unknown GVH_IMPL_MODE: %s\n' "$mode" >&2
            return 1
            ;;
    esac

    printf '%s\n' "$rc" > "$out_dir/exit"
    _gvh_normalize "$vault_dir" < "$raw" > "$out_dir/stdout"

    : > "$out_dir/obsidian-argv"
    if [ -f "$obs_log" ]; then
        _gvh_normalize "$vault_dir" < "$obs_log" >> "$out_dir/obsidian-argv"
    fi

    rm -rf "$sandbox"
    return 0
}

# Normalise only what legitimately varies between runs: the sandbox path and
# ANSI colour. Nothing date-dependent is emitted by this script, so unlike the
# crystallize corpus there is no <TODAY> substitution to make.
_gvh_normalize() {
    local vault_dir="$1"
    sed -e 's/\x1b\[[0-9;]*m//g' \
        -e "s|${vault_dir}|<VAULT>|g"
}
