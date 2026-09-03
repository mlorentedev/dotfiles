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

| AC | Proof |
|---|---|
| AC1 pure sorted join, consumes LoadPersona | `TestResolveRoles` — fixtures come from `LoadPersonas`, never a literal map, so the test breaks if the parse breaks |
| AC2 ambiguity returned in full | `TestResolveRolesAmbiguity` — both rules asserted, keyed by rule id with skills read from `triggers.json` |
| AC3 drift guard on the join | `TestRoleJoinDrift` — floor of 16 resolving rules, plus every persona contributing >=1 skill |
| AC4 hook emitted from the manifest | `TestManifestEmitsPromptHook` + `harness/manifest.json` `agents.bind[claude].emit_hooks[suggest-role]` |
| AC5 payload from stdin only | `TestSuggestFromHookReadsStdin` — asserts `--prompt` does not change from-hook output |
| AC6 prompt field measured, not assumed | `TestPromptFromHookPayloadAcceptsPlausibleSpellings` (8 spellings), `...ReportsUnrecognised`, `...PrefersTheMostSpecificSpelling`, `TestSuggestFromHookRecordsPromptField` |
| AC7 never exits non-zero | `TestSuggestFromHookNeverExitsNonZero` — 9 rows: malformed JSON, array payload, empty, no prompt field, blank prompt, no rule matched, missing harness root, unparseable triggers.json, unreadable persona record |
| AC8 latency budget | `TestRoleJoinLatencyBudget` — 20ms per prompt over 100 iterations |

### End-to-end, with the built binary rather than a test

```
$ echo '{"hook_event_name":"UserPromptSubmit","prompt":"add tests for this module using TDD"}' \
    | dotf harness suggest --from-hook --repo-root <checkout>
[suggest] prompt arrived as "prompt"
[persona] builder  ← pattern: pattern-testing-standards
  skills: test, test-driven-development
  → consider adopting `builder` and invoking test-driven-development
exit=0

$ echo '{"nonsense":' | dotf harness suggest --from-hook          # garbage
[suggest] payload unrecognised: no known prompt field (13 bytes)
exit=0

$ cd /tmp && DOTFILES_DIR=/nonexistent dotf harness suggest --from-hook   # non-dotfiles repo
[suggest] no personas at /nonexistent: ...no such file or directory
exit=0
```

## Corrections made during implementation

Recorded rather than silently fixed, because each was a claim that did not survive its probe:

1. **AC4's parenthetical was wrong.** It said `manifest.json` had no hook-emission key and this
   would be the first. `emit_hooks` already exists under `agents.bind[]` and carries the gate and
   both `mem` hooks. This is a fourth entry on an existing mechanism.
2. **The output labelled the pattern as a "rule".** The first end-to-end run printed
   `← rule: pattern-testing-standards`. `Suggestion` carries pattern names, not trigger-rule ids —
   `cyclomatic-complexity` matches `pattern-language-standards`, whose rule id is
   `code-complexity-and-refactor`. Relabelled `pattern:`. A field labelled as something it does not
   hold is the defect GUARD-009 (#1448) exists to detect, and it was caught by running the binary,
   not by reading the code.
3. **A test fixture asserted a claim the design never made.** `TestResolveRolesAmbiguity` first
   asserted the skill `spec` resolves to `[planner, reviewer]`; it resolves to `[planner]` alone.
   Ambiguity lives in the RULE, whose skills are `[spec, adversarial-review]`. The code was right
   and the test was wrong.

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
