---
tags: [spec, tasks, ideas-002]
created: "2026-05-25"
---

# Tasks - IDEAS-002-shell-functions

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/ideas-002-shell-functions`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (R1-R5 all have mitigations spec'd)
- [x] Decide sourcing strategy: wait for IDEAS-003 (loop) OR explicit `source` in `.zshrc`/`.bashrc`?

## Implementation

> TDD order. One function per cycle: write failing bats test → implement → shellcheck → next.

- [x] Write failing bats test for `mkd`. Implement `mkd()` in new file `.zsh/functions.sh`. Wire up sourcing in `.zshrc` + `.bashrc`. Confirm pass + shellcheck clean.
- [x] Write failing bats test for `gz`. Implement `gz()`. Confirm pass + shellcheck clean.
- [x] Write failing bats test for `dataurl` (with graceful MIME fallback for R2). Implement `dataurl()`. Confirm pass + shellcheck clean.
- [x] Write failing bats test for `targz` (gzip-only path; document zopfli/pigz preference in code). Implement `targz()`. Confirm pass + shellcheck clean.
- [x] Add smoke test for `server` (skippable if python3 missing) + `getcertnames` (skippable if no network). Implement both functions. Confirm shellcheck clean.
- [x] Cross-shell parity check: matrix-run all bats tests under bash and zsh.
- [x] Update README with "Shell helpers" section (6 one-liners + name-collision caveat referencing IDEAS-001).
- [x] Run `healthcheck.sh` — confirm no regressions in cumulative tool detection.
- [x] Refactor pass: extract common helpers if any duplication emerged.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Type checks N/A (shell)
- [x] Lint: `shellcheck .zsh/functions.sh` exits 0
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/IDEAS-002-shell-functions/features.json`):

```json
[
  {
    "id": "IDEAS-002-shell-functions-f1",
    "behavior": "mkd creates and cd's into a nested path",
    "verification": "bats tests/shell-functions.bats --filter 'mkd'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-002-shell-functions-f2",
    "behavior": "dataurl emits a valid data: URL with detected MIME",
    "verification": "bats tests/shell-functions.bats --filter 'dataurl'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-002-shell-functions-f3",
    "behavior": "targz produces a gzip-decompressible tarball",
    "verification": "bats tests/shell-functions.bats --filter 'targz'",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-002-shell-functions-f4",
    "behavior": "gz reports original and gzipped sizes",
    "verification": "bats tests/shell-functions.bats --filter 'gz '",
    "state": "pending",
    "evidence": ""
  },
  {
    "id": "IDEAS-002-shell-functions-f5",
    "behavior": "shellcheck passes on functions file",
    "verification": "shellcheck .zsh/functions.zsh",
    "state": "pending",
    "evidence": ""
  }
]
```
