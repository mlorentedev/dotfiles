---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - OPS-023-bitacora-board-resilience

## Evidence

### `tests/bitacora-reconcile.bats` — 9 cases, all green

This machinery had **zero** test coverage before this change, which is the
mechanical reason a dropped board item could stay invisible for three days across
two repos. The execution cases drive a stub `gh` (the script's decisions and the
commands it builds are what is under test, not GitHub) plus a stub `age` that
exits 1 with a message — so AC3 fails loudly if the decrypt step is ever reached,
rather than passing by accident.

| AC | Case |
|---|---|
| AC1 | `--backfill-only` ensures open issues and PRs are on the board |
| AC2 | `--backfill-only` does not provision — no link, no secret, no workflow push |
| AC3 | `--backfill-only` never needs the age key, so it can run in CI |
| AC4 | a full run still provisions (the flag narrows scope, not the tool) |
| AC5 | the add workflow classifies a rate limit apart from other failures |
| AC5 | the add workflow never blanket-swallows failures |
| AC6 | the add workflow takes the node id from the payload, not a lookup |
| AC7 | the reconciler validates its repo-name input before word-splitting |
| AC8 | the reconciler is scheduled and goes loud when it cannot run |

### A mistake worth recording, caught by the tests themselves

The first cut of the structural cases grepped the raw workflow files for
`continue-on-error`, `actions/add-to-project` and `API rate limit`. All three
failed — not because the code was wrong, but because each workflow's header
*explains* those constructs at length, so the assertions matched the prose that
argues against them.

That is the same failure shape as the plugin audit in `docs/lessons.md`
(2026-08-06): judging a fact from descriptive text rather than from the artifact.
Fixed by stripping comment lines before matching, and by asserting on syntactic
forms (`continue-on-error:`, `uses: actions/add-to-project`) that only appear in
code. The red run is what surfaced it.

### Lint

- `shellcheck scripts/bitacora-rollout.sh` — clean.
- `bash -n scripts/bitacora-rollout.sh` — clean.
- Both workflows parse as YAML.
- `actionlint` reports one info-level SC2016 on the GraphQL query's single
  quotes. Verified pre-existing: `main`'s copy of the file triggers the identical
  finding on the same construct, and the single quotes are correct — `$project`
  and `$contentId` are GraphQL variables that must not be shell-expanded.

### Injection review

The reconciler takes a `workflow_dispatch` repo-name input that must be
word-split to accept several repos. An unquoted expansion of that value into a
command line is the canonical workflow-injection shape, so every token is
validated against `[A-Za-z0-9._-]` and anything else is refused before the call.
AC7 pins both halves: the guard is present, and the raw variable is never passed
through unvalidated.

### Not verified here

The multi-repo rollout is deliberately out of scope — this PR changes the
canonical copy only. Propagating it touches other repositories and is an
operational step requiring an explicit go.
