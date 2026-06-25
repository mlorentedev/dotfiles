---
tags: [spec, verification, secrets, shell]
created: "2026-06-25"
---

# Verification - CLI-024-secrets-retire-loadsecrets

> PR-B1 only. Branch `feat/nan-scripts-via-dotf-secrets` (off main). B2/C verified
> with their own PRs.

## Evidence

- [x] **AC1** — nan-* resolve `NAN_API_KEY` via `dotf secrets show nan-api-key`
  → test `tests/nan-scripts-secrets.bats` (7/7), edits to
  `scripts/nan-{bench,debug,quality-bench}.sh`.
- [ ] **AC2/AC3** — setup migration + twin deletion → PR-B2 / PR-C.
- [x] **AC4** — neighbouring suites green; no regressions.
- [ ] **AC5** — PAT lesson archived → PR-C.

## TDD log — AC1

`tests/nan-scripts-secrets.bats` is structural (grep-based), matching the other
RC/script contract tests in this suite.

- **RED** (before edit): 5/7 failing —
  `not ok 2 … do not source the retired load-secrets twin`,
  `not ok 3 … do not call secrets_refresh`,
  `not ok 4 … self-fetch via dotf secrets show`,
  `not ok 5 … guard with command -v dotf`,
  `not ok 6 … hint 'dotf secrets run'`.
- **GREEN** (after editing the 3 scripts): **7/7 ok**.

The fallback in each script is now:

```sh
if [ -z "${NAN_API_KEY:-}" ] && command -v dotf >/dev/null 2>&1; then
    NAN_API_KEY="$(dotf secrets show nan-api-key 2>/dev/null || true)"
    export NAN_API_KEY
fi
```

`|| true` sits inside the substitution so `set -euo pipefail` never trips. The
`[ -z … ] && command -v dotf` guard means a `dotf secrets run -- <script>`
invocation (key already injected) skips the block — no double-fetch.

## Test status

- `bats tests/nan-scripts-secrets.bats` → **7/7**.
- `bats tests/shell-wrapper-dedup.bats` → **13/13** (asserts the `dbg`→nan-debug
  alias dedup; unaffected by the scripts' internal secret resolution).
- `bats tests/powershell-profile.bats` → **14/14**.
- `bash -n` on all three scripts → clean. `shellcheck` → clean.
- **Live fetch deferred:** deployed `dotf` is 0.18.0 (no `secrets show` yet — this
  PR is the `feat(secrets):` commit that lets release-please cut 0.19.0). The
  end-to-end `dotf secrets show nan-api-key` is exercised at the post-deploy smoke
  (issue #587 / spec task #3). Until then a direct run degrades gracefully:
  `show` fails → `2>/dev/null || true` → empty key → the `dotf secrets run` hint.

## Decisions made during implementation

- **`show` (self-fetch), not a shell wrapper.** nan-* read a single var, so the
  in-script `dotf secrets show nan-api-key` keeps them directly runnable AND
  wrappable under `dotf secrets run` — no new alias/function surface, aliases
  (`dbg`, …) keep calling the script on PATH untouched.
- **Split out of the user's "PR-B".** The setup eager-load is a high-risk
  critical-path refactor; bundling it with this trivial swap would violate the
  atomic-PR rule. → PR-B2.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no (mechanical migration; the pattern is the
  ADR-028 idiom, already documented).
- [ ] ADR-worthy? no (executes ADR-028).
- [ ] New cross-project pattern? no.

## Archive checklist (deferred to PR-C, which closes #587)

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-024-secrets-retire-loadsecrets/`
- [ ] Promotions above executed (if any)
