---
tags: [spec, verification, templates]
created: "2026-08-11"
---

# Verification - DOCS-013-doc-path-guard

## Evidence

| AC | Claim | Proof |
|---|---|---|
| 1 | Every repo path in `.claude/CLAUDE.md` resolves | `./scripts/check-doc-paths.sh .claude/CLAUDE.md` → `OK`. Before the change the same command reported **8** dead paths |
| 2 | Secrets docs describe ADR-028 | `grep -n env-mapping .claude/CLAUDE.md` → only plain-text mentions naming it as retired; the two "adding a secret" recipes now use `dotf secrets set/verify/run` |
| 3 | Verification Commands covers the Go layer | New *Go layer* block: `go build/vet/test` + `golangci-lint run`, with the `GOLANGCI_LINT_VERSION` pin and the reason an unpinned local run proves nothing (BUG-071) |
| 4 | No hardcoded vault literal | `grep -n 'Projects/knowledge' .claude/CLAUDE.md` → no matches; three occurrences replaced with `$VAULT_PATH` |
| 5 | Guard catches, on real and seeded inputs | `tests/check-doc-paths.bats` cases 3 (dead path), 4 (empty glob), 7 (ALL-CAPS with extension); README's dead `load-secrets.sh` was found by the guard, not by reading |
| 6 | Zero false positives on `AGENTS.md` | `./scripts/check-doc-paths.sh AGENTS.md` → `OK`. An earlier revision reported **13** on the same file; case 6 pins nine of the token shapes that caused them |
| 7 | Suite pins both behaviours | `bats tests/check-doc-paths.bats` → **8/8** |

## Test status

- `bats tests/check-doc-paths.bats` → **8 passed, 0 failed** (re-run against the
  final tree after the last edit, not an earlier state).
- `bats tests/*.bats` → **1205 passed, 1 failed** of 1206.
  The one failure is `not ok 402 converges over a running dotf: a live binary in
  dest is replaced, not refused` — **#807 (BUG-054)**, pre-existing and
  unrelated. Reproduced this session on clean `main` @ `a3f9a10` with no
  working-tree changes, same `coreutils: unknown program 'dotf'` cause.
- `shellcheck scripts/check-doc-paths.sh` → clean.
- `bash -n` and `zsh -n` on the new script → clean; the script was also
  **executed** under `zsh -c`, not merely parsed, because the failure class this
  repo keeps hitting is a construct that runs and answers wrongly rather than
  erroring (`docs/lessons.md`, 2026-08-09).
- `./scripts/check-bats-names.sh tests/` → OK, 82 files.
- `./scripts/check-spec-gate.sh --explain` → `Spec folder touched: yes`,
  active-spec LOC 199 (floor 10).
- Guard clean on all six instruction files: `.claude/CLAUDE.md`, `AGENTS.md`,
  `README.md`, `ai/claude/CLAUDE.md`, `ai/copilot/copilot-instructions.md`,
  `.github/copilot-instructions.md`.

## What was found while verifying

Two defects in the guard itself, both caught by running it rather than
reasoning about it, and both now pinned as tests:

1. **False positives (26, then 13).** The first revision flagged `&>/dev/null`,
   `ai/<agent>/`, the model id `opencode-go/qwen3.6-plus` and more. The control
   run on `AGENTS.md` — a file with no staleness — was what exposed the scale.
2. **A false negative.** The ALL-CAPS placeholder rule intended for
   `sensitive/KEYNAME.secret.age` also swallowed `SKILL.md`, so the guard
   silently skipped `ai/skills/*/SKILL.md`, a genuinely dead glob it existed to
   catch. Fixed by applying the placeholder reading only when the token lacks a
   known extension.

The second is the one worth remembering: a guard that under-reports looks
identical to a clean repo.

## Not verified

- **Deployment.** This change is repo-side only; `.claude/CLAUDE.md` is read
  from the checkout, so no deploy step applies. `README.md` is not deployed.
- **Windows.** The guard is bash and runs in the Linux `test` job. It is not
  wired into any PowerShell path, and nothing on Windows consumes it.

## Sign-off

- [ ] Independent `/adversarial-review` — **owed, not performed.** This session
      implemented the change and therefore cannot review it. `dotf spec archive`
      refuses without a fresh passing `review.md`, so this must be supplied by
      another session before the spec can be archived.

---

## Round 2 — after the adversarial review

The round-1 review (`review.md`, verdict **FAIL**, reviewed_sha `d6681a6`) found
one Major that neither the author nor CodeRabbit caught, plus six Minors. All
were reproduced independently before being accepted.

### The Major

`.claude/CLAUDE.md:51` documented the golangci-lint install as
`@v$(. versions.conf; …)`. A slashless argument to the `.` builtin is searched
on `$PATH` only; bash additionally falls back to the cwd, zsh does not. Under
zsh — the default interactive shell here — it resolved to **empty**, producing
`@v`:

```console
$ bash -c 'echo "v$(. versions.conf; echo "$GOLANGCI_LINT_VERSION")"'
v2.12.2
$ zsh -c 'echo "v$(. versions.conf; echo "$GOLANGCI_LINT_VERSION")"'
zsh:.:1: no such file or directory: versions.conf
v
```

A wrong instruction, shipped in the PR whose purpose was to stop instructions
being wrong, by the same bash/zsh divergence class the file's own
prohibited-patterns table documents twenty lines above it.

Fixed to `. ./versions.conf`; verified `2.12.2` under zsh. The pattern is now a
row in that table, and two tests pin it: one on `versions.conf` specifically,
one class-level over all three instruction files.

### Minors applied

| Finding | Resolution |
|---|---|
| "backticked path is a live claim" oversells the guard (bare names unchecked) | Callout scoped to slash-containing paths, with the blind spot stated |
| Case 6 pins 9 shapes but 3 pass via `is_repo_rooted`, not the filter | Not papered over — the filter is now labelled defense-in-depth in the script, and the test comment says it pins the outcome, not the mechanism |
| `versionMatches` described as callable | "append an entry to the `versionMatches` table" |
| `#!/bin/bash` vs the repo's documented `#!/usr/bin/env bash` | Changed |
| `setup-macos.sh` — live instance of the bare-name blind spot | De-backticked both README mentions |
| red `spec-gate` not disclosed in Evidence | Disclosed here; archive-on-merge resolves when this review lands |

### CodeRabbit findings (PR #922), all reproduced

- README:42 claimed secrets are "auto-loaded at login", contradicting the
  ADR-028 text this PR added at :95. Also stale in the same block: "21 custom
  skills" (37) and "316 BATS tests" (1206).
- The script header still described basename resolution that was removed.
- **Path traversal**: `scripts/../../dotfiles/README.md` passed `is_repo_rooted`
  and was reported `OK` while resolving outside the repo — a false negative.
  CodeRabbit's stated mechanism was slightly off (it predicted a wrong success
  on a nonexistent path); the real shape is silent acceptance. Rejected now,
  with a regression test.

### Round-2 evidence

- `bats tests/check-doc-paths.bats` → **11/11**
- Mutation: a bare `. versions.conf` added to a live README code block turns
  case 11 red; removing it turns it green again
- `zsh -c '. ./versions.conf; echo $GOLANGCI_LINT_VERSION'` → `2.12.2`
- `shellcheck` clean; `bash -n` + `zsh -n` clean
- Guard clean on all instruction files

### Still owed

A **fresh** adversarial review at the new sha. A Major cannot be waived by
re-reading the reviewed commit, so `dotf spec archive` stays blocked until a
passing `review.md` exists for this head.
