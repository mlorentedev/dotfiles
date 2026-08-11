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
