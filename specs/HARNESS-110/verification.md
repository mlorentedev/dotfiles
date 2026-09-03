---
tags: [spec, verification, templates]
created: "2026-09-02"
---

# Verification - HARNESS-110

## Work-gate: satisfied by another transport, NOT skipped

This spec was scaffolded with `--force-no-gate`, so the record of the gate belongs here rather than
in the tool's output. The gate's own assertion (`spec.Gate`: the issue exists and is OPEN) was run
by hand over REST, because GitHub's GraphQL endpoint was under a **secondary** rate limit — two
agent sessions sharing one account quota — while REST was unaffected:

```
$ gh api repos/mlorentedev/dotfiles/issues/1436 --jq '.state, (.pull_request != null)'
open
false                      # measured 2026-09-02, ~18:40 local
```

`state=open` and `pull_request=null` together are exactly what `Gate` checks, plus the PR exclusion
`Gate` gets for free by calling `gh issue view`. The gate was met; only its transport differed.

**Do not "fix" `spec.Gate` to use REST on the strength of this.** Two measurements say the
`gh issue view` path at `cli/internal/spec/spec.go:73-102` is deliberate, as its comment claims:

1. `state` case differs — REST returns `open`, the code compares against `"OPEN"`. A naive swap
   fails closed on every issue.
2. REST `/issues/{n}` **returns pull requests too** — `#1419` reads `state=open, is_pr=true`. A REST
   gate would accept a PR as a work-gate unless it also asserted `.pull_request == null`. That is
   the same defect class as the reviewer that treated PR #1433 as a ticket to satisfy on #1440.

## Known before the hook half is written

Recorded here rather than discovered during AC5-AC7:

- **The hook does not run from the checkout.** `ResolveRoles`' tests load personas from
  `../../../harness/agents` and triggers from the repo root, which is right for a test and wrong for
  the runtime: the hook runs as a deployed binary with cwd set to whatever repo the user is in.
  `--from-hook` must resolve personas the way `harness_gate.go` does (manifest `record_dir` under the
  deploy dir), not by a repo-relative path.
- **A non-dotfiles repo has no `harness/triggers.json` at cwd.** That is an AC7 fail-open case —
  print nothing, exit 0 — and it belongs as a row in the exit-status table, not as an error path.

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

-
-

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-110/` -> `specs/archive/HARNESS-110/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
