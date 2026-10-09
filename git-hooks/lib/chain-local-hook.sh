#!/usr/bin/env bash
#
# chain-local-hook.sh <hook-type> [args...] — run the repo's own gate for the
# given hook type. A global core.hooksPath makes git ignore .git/hooks/ for ALL
# hook types; this restores per-repo hooks by exec-ing the literal path.
#
# Repos managed by pre-commit have no such file to chain to. The same global
# core.hooksPath makes `pre-commit install` refuse ("Cowardly refusing to install
# hooks with core.hooksPath set"), so .git/hooks/<type> is never written and the
# repo's gates silently never run — that is BUG-036, which left the knowledge
# vault's gitleaks pre-push secret scan inactive. When there is no local hook but
# the repo carries a .pre-commit-config.yaml, hand the stage to pre-commit
# directly; that restores every such gate machine-wide with no per-repo install
# and no change to core.hooksPath.
#
# `hook-impl` — not `run --hook-stage` — because git delivers a pre-push hook's
# ref list on stdin and only hook-impl parses it into --from-ref/--to-ref. A
# `run --hook-stage pre-push` would fall back to the staged file set and report
# green on the wrong input, which is worse than the no-op it replaces. It is the
# same entry point pre-commit's own generated hook uses, and --hook-dir is passed
# for the same reason that hook passes it: hook_impl feeds the value straight into
# os.path.join() to look for a <stage>.legacy hook, BEFORE running anything. Omit
# it and pre-commit 4.4.0 raises TypeError on None and the stage exits 3 — which,
# on a dispatcher wired machine-wide, aborts `git commit` in every repo that has a
# config and no locally-installed hook (BUG-055). An earlier comment here claimed
# omitting it was an upstream-supported dispatcher path; it is not, and nothing
# tested the claim because the suite stubs pre-commit out.
#
# Which hooks a stage actually has is pre-commit's decision, not ours: a config
# declaring nothing for this stage exits 0 by itself, so there is no YAML to
# parse here.
#
# Both paths are `exec` on purpose — that is what makes stdin reach the child and
# its exit status become the hook's, and the exit status is the whole feature.
# A repo with no local hook and no .pre-commit-config.yaml is a clean no-op:
# this dispatcher is wired machine-wide, and most repos on a box declare no gate.
# A repo that DOES declare one fails closed on the stages that can block
# (pre-commit, pre-push, commit-msg) when pre-commit is missing. A no-op there
# was a silent bypass: on a Mac without pre-commit, every vault commit and push
# skipped gitleaks while `dotf doctor` reported the gate active (2026-10-08).
#
# pre-commit is looked up on PATH, then in uv's tool bin dir, where `dotf tools`
# installs it. GUI launchers do not read the shell rc: obsidian-git commits the
# vault under launchd's PATH, which has no ~/.local/bin.

set -u

hook_type="${1:?hook type required}"
shift

toplevel="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

# BUG-090: pre-commit's own env-build clones live under its store and must
# never re-enter this dispatcher. When a hook repo (e.g. pre-commit-hooks)
# needs its environment built, pre-commit clones it into the store and checks
# out the pinned rev — an ordinary git checkout, which fires the GLOBAL
# post-checkout hook same as any other. If that clone carries its own
# .pre-commit-config.yaml (upstream dogfoods itself), the fallback below would
# hand it straight back to `pre-commit hook-impl`, re-entering the SAME shared
# store the outer invocation is already holding locked to build that very
# environment. Parent waits on the checkout it triggered; child waits on the
# lock the parent holds. Deadlock, reproduced live: `pre-commit install-hooks`
# hung indefinitely on a first-time env build, confirmed via the process tree
# (a nested `hook-impl --hook-type post-checkout` sleeping inside the store
# clone) with no unrelated cause (no leaked GIT_DIR/GIT_INDEX_FILE in its
# environ). These clones are not user repos; their gates must never run here,
# the same reasoning BUG-036 already applies to pre-commit's OWN config.
# Store path per pre-commit's own Store.get_default_directory().
precommit_store="${PRE_COMMIT_HOME:-${XDG_CACHE_HOME:-$HOME/.cache}/pre-commit}"
# $toplevel is the physical path, so a store reached through a symlink (macOS's
# /var -> /private/var, a linked ~/.cache) would never match as written.
precommit_store="$(cd "$precommit_store" 2>/dev/null && pwd -P || printf '%s' "$precommit_store")"
case "$toplevel" in
    "$precommit_store"/*) exit 0 ;;
esac

# Hooks are shared repo state, not per-worktree state: git keeps them in the
# COMMON git dir. In a linked worktree — and under --separate-git-dir —
# $toplevel/.git is a `gitdir:` pointer FILE, not a directory, so assuming that
# layout resolved to a path that cannot exist and silently skipped every local
# hook there (BUG-043). `--git-common-dir` answers relative to the cwd in an
# ordinary checkout and absolute in a linked worktree; Git for Windows emits
# that absolute path as `C:/...`, which is absolute despite lacking a leading
# slash. Asking from $toplevel makes all forms resolve without requiring
# --path-format=absolute (git 2.31+) and therefore without a version floor.
common_dir="$(cd "$toplevel" && git rev-parse --git-common-dir 2>/dev/null)"
case "$common_dir" in
    /*|[[:alpha:]]:/*) ;;                      # POSIX or Windows absolute path
    ?*) common_dir="$toplevel/$common_dir" ;; # ordinary checkout: relative to $toplevel
esac                                          # empty (probe failed): left empty on purpose
# Two ways the probe lies, both ending in a silently skipped hook:
#
#   - `git rev-parse` echoes back an option it does not understand and still exits
#     0, so a git predating --git-common-dir (< 2.5) hands us the literal flag;
#   - the probe can fail outright (a $toplevel that vanished between the two calls
#     — there is no `set -e` here), leaving the answer empty. That is the input
#     that matters: an empty value joined above would have become "$toplevel/",
#     which always passes a directory test, so it would walk straight through the
#     guard below and resolve hooks under "<toplevel>//hooks".
#
# Require a non-empty path that is a real directory; otherwise fall back to the
# classic layout rather than resolving hooks somewhere they cannot be.
[ -n "$common_dir" ] && [ -d "$common_dir" ] || common_dir="$toplevel/.git"

local_hook="$common_dir/hooks/$hook_type"
[ -x "$local_hook" ] && exec "$local_hook" "$@"

pre_commit_config="$toplevel/.pre-commit-config.yaml"
[ -f "$pre_commit_config" ] || exit 0

# uv's documented order for its tool executables: UV_TOOL_BIN_DIR, then
# XDG_BIN_HOME, then ~/.local/bin.
pre_commit="$(command -v pre-commit 2>/dev/null)"
if [ -z "$pre_commit" ]; then
    uv_bin="${UV_TOOL_BIN_DIR:-${XDG_BIN_HOME:-$HOME/.local/bin}}"
    # pre-commit.exe on Windows, the name doctor looks for there too.
    for candidate in "$uv_bin/pre-commit" "$uv_bin/pre-commit.exe"; do
        if [ -x "$candidate" ]; then pre_commit="$candidate"; break; fi
    done
fi
if [ -n "$pre_commit" ]; then
    exec "$pre_commit" hook-impl \
        --config "$pre_commit_config" \
        --hook-type "$hook_type" \
        --hook-dir "$common_dir/hooks" \
        -- "$@"
fi

case "$hook_type" in
    pre-commit|pre-push|commit-msg)
        printf '%s\n' \
            "$hook_type: $toplevel declares .pre-commit-config.yaml, but pre-commit is not installed, so its gates (gitleaks among them) cannot run." \
            "Install it with: dotf tools install pre-commit" \
            "To skip the gates once, deliberately: --no-verify" >&2
        exit 1
        ;;
esac
exit 0
