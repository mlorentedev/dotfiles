---
id: lesson-153-a-guard-installed-machine-wide-can-silently-disabl
type: lesson
status: active
created: "2026-08-06"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 153: A guard installed machine-wide can silently disable every other guard

**Context**: GUARD-001 enforces its memory-sink check everywhere by setting a **global** `core.hooksPath`. Its dispatcher was written knowing that this makes git ignore `.git/hooks/` entirely, so it deliberately chains onward — the comment in `git-hooks/pre-push` says it exists so "per-repo guards (gitleaks) survive". The intent was right and the code did exactly what it said.

**Problem**: `pre-commit install` refuses to run while `core.hooksPath` is set (`Cowardly refusing to install hooks with core.hooksPath set`), so `.git/hooks/pre-push` was never created in the first place. The dispatcher chained to a file that did not exist and returned its clean no-op, exit 0. The knowledge vault's `gitleaks` secret scan had therefore not run on a single push — one guard had disabled another, and both were reporting success. Worse, `dotf doctor --fix` offered `pre-commit install` as the remedy for that exact FAIL: the repair was the very command the first guard blocks, so the diagnosis was correct, the fix was impossible, and nothing in the output said so. The same theme surfaced twice more the same day (#761): the dispatchers are extensionless, so `.gitattributes`' `*.sh eol=lf` misses them and a Windows checkout gives them a `\r` shebang that dies with exit 127 — a guard whose liveness depends on which `bash` happens to run it.

**Solution**: Make the dispatcher hand the stage to `pre-commit hook-impl` when there is no local hook but a `.pre-commit-config.yaml` exists, which restores every repo's gates without touching `core.hooksPath`. Notably **not** `pre-commit run --hook-stage`, which the issue had proposed: `run` accepts no stdin, and git delivers a pre-push hook's ref list *on stdin*, so it would have scanned the staged file set instead of the commits being pushed — green for the wrong reason.

**Rule**: A security control must assert it is **effective**, not that it is installed. "The hook file exists", "the setting is set", "the dispatcher ran" are all satisfiable while nothing is being checked; the only honest test fires a real violation and watches it get blocked. Two corollaries. When one enforcement mechanism claims a shared resource machine-wide — `core.hooksPath`, a PATH shim, a global config — enumerate what else uses that resource, because it is now yours to keep alive. And a repair action that cannot succeed is worse than no repair action: if a fix is structurally blocked by another component, say so in the output instead of proposing it every run.
