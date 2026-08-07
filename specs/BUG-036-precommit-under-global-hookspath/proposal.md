---
id: "BUG-036-precommit-under-global-hookspath"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-06"
issue: "mlorentedev/dotfiles#748"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, security]
template_version: "1.0"
---

# BUG-036-precommit-under-global-hookspath

## Why

GUARD-001 sets a **global** `core.hooksPath` so its memory-sink guard runs in every repo. That setting makes `pre-commit install` refuse (`Cowardly refusing to install hooks with core.hooksPath set`), so `.git/hooks/<type>` is never written — and the GUARD-001 dispatcher, whose stated job is to "chain to the repo-local hook so per-repo guards (gitleaks) survive", chains to a file that does not exist and exits 0. One guard silently disabled every other repo's pre-commit gate, including the knowledge vault's `gitleaks` pre-push secret scan. `dotf doctor --fix` proposes `pre-commit install` as the remedy, which is the very command the first guard blocks: the repair path is circular.

## What

The GUARD-001 stage dispatchers stop being a no-op when a repo has pre-commit-managed hooks that were never installable. When no repo-local hook exists for a stage but the repo has a `.pre-commit-config.yaml`, the dispatcher invokes pre-commit directly, so per-repo gates fire again — in **every** repo on the machine, with no per-repo install step and no change to the global `core.hooksPath`.

Observable: with a global `core.hooksPath` set and no `.git/hooks/pre-push`, a repo whose `.pre-commit-config.yaml` declares a failing `pre-push` hook now fails the push instead of silently passing.

## Design decision: `hook-impl`, not `run --hook-stage`

Issue #748 proposed `pre-commit run --hook-stage <stage>`. **That would be wrong for `pre-push`**, which is the stage the vault's gitleaks gate actually uses.

`pre-commit run` takes no stdin. Git hands a pre-push hook its ref list on stdin, and pre-commit's own `hook_impl._pre_push_ns()` parses exactly that to compute `--from-ref`/`--to-ref`, i.e. *which commits are being pushed*. Invoked as `run --hook-stage pre-push` with no refs, the run falls back to the staged file set — so it would scan the wrong thing and report success on a push containing a secret. A gate that reports green on the wrong input is worse than the current silent no-op.

`pre-commit hook-impl` is what pre-commit's own generated hook calls (`pre_commit/resources/hook-tmpl`):

```sh
ARGS=(hook-impl --config=... --hook-type=...)
ARGS+=(--hook-dir "$HERE" -- "$@")
exec pre-commit "${ARGS[@]}"
```

`--hook-dir` is **optional**, and omitting it is a supported first-class path rather than a workaround — `hook_impl._run_legacy()` branches on `if hook_dir is None:  # git 2.54+ hooks`, i.e. exactly the case of pre-commit being driven by a dispatcher rather than by a file it installed. So the dispatcher calls `pre-commit hook-impl --hook-type <stage> -- "$@"` with stdin inherited, and gets identical semantics to an installed hook for every stage.

### Rejected alternatives

- **`doctor --fix` unsets `core.hooksPath`, installs, restores it** — rejected in the issue: racy, and only repairs repos the doctor happens to visit.
- **`pre-commit install --git-dir <dir>`** — `install()` does skip the refusal when `git_dir` is passed, but that parameter is not exposed on the `install` CLI (only `init-templatedir` uses it), so it is not reachable.
- **Dropping the global `core.hooksPath`** — reopens GUARD-001, whose whole design is machine-wide enforcement.

## Out of scope

- `dotf doctor` changes (issue tasks 4 and 5: stop emitting the impossible `pre-commit install` remedy; add an effectiveness check that a hook actually fires). Go-side work in `cli/internal/doctor`, split to a second PR to keep this one atomic.
- `scripts/install-precommit.sh`, which calls the same blocked `pre-commit install`. Tracked with the doctor work.
- Fixing the vault's own hook state. Once the dispatcher fires, the vault needs no per-repo action — that is the point.
- The CRLF defect in the same dispatchers (#761) — independent root cause, independent fix.

## Risks / open questions

- **Double execution.** If a repo-local hook *does* exist (installed before `core.hooksPath` was set, or hand-written), the dispatcher must keep chaining to it and must NOT also run pre-commit, or hooks run twice. Resolved: the fallback is reachable only when no executable local hook is present, preserving today's precedence exactly.
- **Repos with no pre-commit.** The fallback must be a clean no-op when `pre-commit` is absent from PATH or no config exists — a global dispatcher that failed closed would break `git commit` in every unrelated repo on the machine. Resolved: both are guarded, and the failure mode is exit 0.
- **Exit status is the whole feature.** A gate that runs but swallows a non-zero exit blocks nothing. The dispatcher must propagate pre-commit's status verbatim, which `exec` gives for free.
- **stdin.** `pre-push` semantics depend on the ref list reaching pre-commit; `exec` inherits stdin, so it must stay an `exec` and not a captured subshell.
- **No pre-commit in CI.** The bats job installs `age`, `zsh` and `bats` only, so tests must drive a stub `pre-commit` on `PATH` rather than the real tool. This bounds what the tests can prove: they verify the dispatcher's *decision and invocation*, not pre-commit's internal behaviour.

## Acceptance criteria

- [ ] With a global `core.hooksPath`, no `.git/hooks/pre-push`, and a `.pre-commit-config.yaml` present, the dispatcher invokes `pre-commit hook-impl --hook-type pre-push`.
- [ ] A non-zero exit from pre-commit propagates out of the dispatcher (the push is blocked).
- [ ] stdin (the ref list) reaches pre-commit unmodified.
- [ ] An existing executable repo-local hook still wins, and pre-commit is not additionally invoked.
- [ ] No `.pre-commit-config.yaml`, or no `pre-commit` on PATH, is a clean exit 0.
- [ ] The fallback applies to every stage the dispatcher covers, not just `pre-push`.
- [ ] The GUARD-001 memory-sink guard still runs first and can still veto (`tests/guard-memory-sink.bats` stays green).

## References

- Bitácora board: `mlorentedev/dotfiles#748`
- Sibling defect in the same files: `#761` (CRLF shebangs disable the same dispatchers)
- `git-hooks/lib/chain-local-hook.sh`, `git-hooks/pre-push`, `git-hooks/pre-commit`
- `tests/guard-memory-sink.bats` (GUARD-001 regression suite)
- pre-commit upstream: `pre_commit/resources/hook-tmpl`, `pre_commit/commands/hook_impl.py`, `pre_commit/commands/install_uninstall.py`
