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
# same entry point pre-commit's own generated hook uses; omitting --hook-dir is
# its supported dispatcher path (upstream marks that branch "git 2.54+ hooks").
#
# Which hooks a stage actually has is pre-commit's decision, not ours: a config
# declaring nothing for this stage exits 0 by itself, so there is no YAML to
# parse here.
#
# Both paths are `exec` on purpose — that is what makes stdin reach the child and
# its exit status become the hook's, and the exit status is the whole feature.
# A clean no-op (exit 0) when neither a local hook nor pre-commit is available:
# this dispatcher is wired machine-wide, so failing closed would break `git
# commit` in every unrelated repo on the box.

set -u

hook_type="${1:?hook type required}"
shift

toplevel="$(git rev-parse --show-toplevel 2>/dev/null)" || exit 0

# Hooks are shared repo state, not per-worktree state: git keeps them in the
# COMMON git dir. In a linked worktree — and under --separate-git-dir —
# $toplevel/.git is a `gitdir:` pointer FILE, not a directory, so assuming that
# layout resolved to a path that cannot exist and silently skipped every local
# hook there (BUG-043). `--git-common-dir` answers relative to the cwd in an
# ordinary checkout and absolute in a linked worktree; asking from $toplevel
# makes both forms resolve against the same base, with no need for
# --path-format=absolute (git 2.31+) and therefore no version floor of our own.
common_dir="$(cd "$toplevel" && git rev-parse --git-common-dir 2>/dev/null)"
case "$common_dir" in
    /*) ;;
    *)  common_dir="$toplevel/$common_dir" ;;
esac
# `git rev-parse` echoes back an option it does not understand and still exits 0,
# so a git predating --git-common-dir (< 2.5) hands us the literal flag instead of
# a path. Require a real directory and fall back to the classic layout, rather
# than resolving hooks under a bogus path and reintroducing the silent skip this
# very change removes.
[ -d "$common_dir" ] || common_dir="$toplevel/.git"

local_hook="$common_dir/hooks/$hook_type"
[ -x "$local_hook" ] && exec "$local_hook" "$@"

pre_commit_config="$toplevel/.pre-commit-config.yaml"
if [ -f "$pre_commit_config" ] && command -v pre-commit >/dev/null 2>&1; then
    exec pre-commit hook-impl \
        --config "$pre_commit_config" \
        --hook-type "$hook_type" \
        -- "$@"
fi

exit 0
