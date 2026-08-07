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

local_hook="$toplevel/.git/hooks/$hook_type"
[ -x "$local_hook" ] && exec "$local_hook" "$@"

pre_commit_config="$toplevel/.pre-commit-config.yaml"
if [ -f "$pre_commit_config" ] && command -v pre-commit >/dev/null 2>&1; then
    exec pre-commit hook-impl \
        --config "$pre_commit_config" \
        --hook-type "$hook_type" \
        -- "$@"
fi

exit 0
