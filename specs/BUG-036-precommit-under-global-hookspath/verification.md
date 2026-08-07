---
tags: [spec, verification, templates]
created: "2026-08-06"
---

# Verification - BUG-036-precommit-under-global-hookspath

## Evidence

All twelve tests live in `tests/precommit-fallback.bats`. Six of them were red before the fix and green after; the other six were green throughout — they pin the behaviour that must NOT change (local-hook precedence, the no-op paths), so their staying green is the regression proof.

- [x] AC1 — dispatcher invokes `pre-commit hook-impl --hook-type <stage>` -> `AC1: with no local hook, a pre-commit config routes the stage to hook-impl`, `AC1: the remote name and url are forwarded to pre-commit`, `AC1: the config is passed explicitly, not left to the caller's cwd` (all red before)
- [x] AC2 — non-zero exit propagates -> `AC2: a failing pre-commit blocks the operation (exit status propagates)` (red before)
- [x] AC3 — stdin reaches pre-commit unmodified -> `AC3: the pre-push ref list on stdin reaches pre-commit unmodified` (red before; asserts byte equality of the ref line)
- [x] AC4 — local hook still wins, no double run -> `AC4: an executable repo-local hook still wins and pre-commit is not also run`, `AC4: a failing repo-local hook still blocks, with no pre-commit fallback` (green before and after)
- [x] AC5 — no config / no binary is a clean exit 0 -> `AC5: no pre-commit config is a clean no-op`, `AC5: a missing pre-commit binary is a clean no-op, not a broken commit` (green before and after)
- [x] AC6 — stage-generic -> `AC6: the fallback is stage-generic, not pre-push-only`, `AC6: commit-msg forwards the message file argument` (red before)
- [x] AC7 — GUARD-001 still vetoes -> `tests/guard-memory-sink.bats` failure set unchanged (see below)

## Test status

- `bats tests/precommit-fallback.bats` -> 12/12 pass. Red run before the fix: 6 failed (1, 2, 3, 4, 9, 10, 11 by name), 6 passed.
- Full suite: **986 tests, 116 failures** vs a `main` baseline of **974 tests, 116 failures** — +12 tests, identical failure count. No regression.
- `shellcheck -x git-hooks/lib/chain-local-hook.sh` -> clean.
- `tests/guard-memory-sink.bats`: 5 of its 8 tests fail **both before and after** this change, in the pre-existing baseline set. Cause is #761 (extensionless hooks get CRLF on a Windows checkout, so `#!/usr/bin/env bash\r` fails with exit 127 under a non-MSYS bash) — unrelated to this work. The signature confirms it: the three that pass are precisely the ones asserting a *non-zero* exit, which a hook that always dies with 127 satisfies by accident.

### Bound on what the tests prove

CI installs only `age`, `zsh` and `bats`, so the suite drives a **stub** `pre-commit` on `PATH`. That is a deliberate limit: these tests verify the dispatcher's *decision and the command it builds* — which stage, which config, which arguments, whose exit status, whether stdin survives — not pre-commit's own behaviour, which is upstream's contract. Whether a planted secret is actually caught (issue #748's first task) is a property of the vault's gitleaks hook, verifiable only on a machine with pre-commit installed.

## Decisions made during implementation

- **`hook-impl`, not the `run --hook-stage` the issue proposed.** `pre-commit run` accepts no stdin, and git delivers a pre-push hook's ref list *on stdin*; only `hook_impl._pre_push_ns()` parses it into `--from-ref`/`--to-ref`. `run --hook-stage pre-push` would have fallen back to the staged file set and reported green on the wrong input — a gate that passes for the wrong reason, which is worse than the silent no-op it replaced. Verified against upstream `hook_impl.py` and `resources/hook-tmpl`.
- **`--hook-dir` omitted deliberately.** Upstream branches on `if hook_dir is None:  # git 2.54+ hooks`, so a dispatcher-driven invocation is a supported first-class path, not a workaround.
- **No YAML parsing.** Whether a stage has hooks is pre-commit's decision; a config declaring nothing for the stage exits 0 on its own. Parsing YAML in shell to pre-filter would add a fragile second source of truth.
- **Fails open, on purpose.** A missing binary or config exits 0. This dispatcher is wired machine-wide via a global `core.hooksPath`, so failing closed would break `git commit` in every unrelated repo on the box.
- **Config passed explicitly** rather than relying on the caller's cwd, even though git runs hooks at the toplevel.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **Yes** — a guard that installs itself machine-wide can disable other guards, and the failure is silent in both directions: `core.hooksPath` made `pre-commit install` refuse, and the dispatcher's "chain to the local hook" no-op'd because there was nothing to chain to. Sibling instance of the same theme in #761. The generalisable rule: a security control must assert it is *effective*, not that its file exists.
- [ ] ADR-worthy decision? No — this implements GUARD-001's existing intent rather than changing a decision.
- [ ] New pattern candidate for `00_meta/patterns/`? Not yet. It needs a second project before it is a pattern rather than an anecdote.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/BUG-036-precommit-under-global-hookspath/` -> `specs/archive/BUG-036-precommit-under-global-hookspath/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
