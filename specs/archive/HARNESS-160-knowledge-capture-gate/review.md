---
spec: "HARNESS-160-knowledge-capture-gate"
verdict: "PASS"
reviewed_sha: "180294436d87ff8e7c2503074cdfba8e33f067b3"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-10"
---

## Adversarial review

**Scope**: HARNESS-160-knowledge-capture-gate, against `git diff b2364ab3ff2ac4b445575fc4983688189e2624bc...HEAD`
**Sources**: `specs/HARNESS-160-knowledge-capture-gate/{proposal,tasks,verification}.md`, `features.json`,
`scripts/check-knowledge-gate.sh`, `scripts/spec-gate-pr.sh`, `.github/workflows/knowledge-gate.yml`,
`cli/internal/spec/{promotion.go,archive.go}`, `forge/branch-protection.json`,
`release-please-config.json`, `.github/pull_request_template.md`, `harness/enforced/definition-of-done.md`,
the four spec commits (`6ceebd46` #1732, `2e73eab3` #1759, `030ade1e` #1761, `18029443` #2298).

**Scope note (stated base).** The launcher's base is the parent of the spec's first commit, so the
range spans 343 commits of main history; only five touch this spec. I reviewed the spec's own change
at HEAD (checker, adapter `--gate`, workflow, promotion pre-flight, templates, DoD, protection
declaration) and confirmed each AC against it. The unrelated commits in the range (e.g. the
`check-lessons.sh` retirement in #2211) belong to other specs' reviews; I checked only that they do
not weaken this gate, and found no interaction. No test was deleted or weakened by this spec's
commits, so the retention bar is not engaged.

### Spec and task alignment

- **AC1 (checker exit codes).** VERIFIED. Full suite `bats tests/check-knowledge-gate.bats`: 34/34
  pass at HEAD, each case run under bash and zsh with output compared. Mutation: replacing the
  "not changed by this PR" test with `elif false` turns tests 19, 20 and 23 red (31/34) — reverted,
  tree clean. Exit 2 paths (unset `SDD_PR_BODY`, undiffable range) reproduced by hand, including
  under macOS `/bin/bash` 3.2. Fenced-section refusal and the unclosed-fence diagnosis pass.
- **AC2 (bot skip).** VERIFIED. The three `dependencies:` cases plus the bot-list parity case
  (extracts `_is_dependency_bot` from both scripts and compares) pass. Human-with-label judged,
  bot-without-label judged — both fail as specified.
- **AC3 (wiring).** VERIFIED. `tests/spec-gate-pr.bats` 14/14 and `tests/knowledge-gate-workflow.bats`
  5/5 pass; triggers and the `cancel-in-progress` expression are byte-equal to `spec-gate.yml`'s
  (read both files: metadata-only events do not cancel an in-flight run; `edited` survives).
  `spec-gate-pr.sh` without `--gate` still defaults to `check-spec-gate.sh`; `--gate` with a `/`
  exits 2. Workflow YAML parses.
- **AC4 (release footer).** VERIFIED. The live `release-please-config.json` footer is fed a release
  PR body in test 34 and passes.
- **AC5 (archive promotion).** VERIFIED. `cd cli && go test ./internal/spec -run Promotion -count=1`
  passes (12 test funcs). Mutation: neutering the `no`-without-reason check fails
  `TestPromotionsRefuseANoWithoutAReason` — reverted, tree clean. Code read: `CheckPromotions` runs
  unconditionally inside `runPreflights`, before anything moves, for normal and `--abandoned` alike;
  no force flag reaches it, matching "the way out is answering the line". All three promotion lines
  in this spec's own `verification.md` answer `yes:` with paths that exist (repo lesson, repo ADR,
  vault pattern confirmed via `dotf env path VAULT_PATH`).
- **AC6 (surfaces).** VERIFIED. DoD §2 in `harness/enforced/definition-of-done.md` and the compiled
  `AGENTS.md` name the section and grammar; `scripts/compile-harness.sh --check` exits 0 (no drift);
  the PR template has `## Knowledge`; the verification template carries `yes: <path>` / `no: <reason>`.
- **AC7 (wiring PR passed its own gate).** VERIFIED live: `gh pr view 1759` body carries
  `- Lesson: docs/lessons/lesson-301-…`, `- ADR: docs/adr/adr-039-…`; both files exist at HEAD;
  `gh pr checks 1759` shows `knowledge-gate pass`.
- **Contract freshness.** `ContractDigests` recomputed from disk equals all three digests recorded
  in `review-request.json` — this review is not stale against proposal/tasks/features.
- **No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags** remain in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | checker | A code-fence marker inside an HTML comment opens the stripper's fence (fences are stripped before comments), so a body whose visible `## Knowledge` section renders fine on GitHub is refused with "code fence opened at line 3 is never closed" — a false red, fail-closed | reproduced at HEAD: body with `<!--` … \`\`\` … `-->` before a valid section exits 1 with the fence message | **UNTESTED** — no case in `tests/check-knowledge-gate.bats` covers a fence marker inside a comment | code (+ one regression test); fix by making the strip a single comment/fence-aware pass |
| Minor | REAL | spec/docs | `verification.md` "Test status" counts no longer reproduce: it claims 33 checker cases and 62 run; the suites now hold 34 + 14 + 5 + 5 = 58 named cases (all pass at HEAD) | my runs at `18029443`; the dated f1–f7 evidence lines are fine, the undated counts drifted | n/a (bookkeeping, not behavior) | `verification.md` — outside the contract set; disposition here or in a follow-up |
| Minor | REAL | spec | `proposal.md` "Out of scope: Declaring the check required" is contradicted by tasks Slice 4, ticked by `18029443`, which declares `knowledge-gate` in `forge/branch-protection.json` (line 27, app 15368) | diff of `18029443`; proposal bullet vs checked task | n/a | `proposal.md` is contract set — under PASS do not edit it; record the disposition in `verification.md` or a follow-up ticket |
| Minor | THEORETICAL | ci | `actionlint` on `knowledge-gate.yml` (claimed in `verification.md`) could not be run — not installed on this machine | `command -v actionlint` empty; YAML parsed OK instead | `tests/knowledge-gate-workflow.bats` cases 20–24 cover triggers/concurrency/naming, not actionlint | — (surface only; rerun actionlint where installed) |

No Blockers, no Majors. Every risk I could construct against the checker fails closed (missing
section, empty body, unset context, undiffable range, out-of-directory path, `_index.md`, deletion,
template placeholder, duplicate lines) rather than failing open, and the negative paths carry named
tests. The gate's own skip (bot + label) is exact-match and parity-tested against `check-spec-gate.sh`.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All seven ACs reproduced at HEAD; one fail-closed false positive (fence-in-comment) and a stale proposal bullet keep it from A |
| Verification       | A | Every feature has a runnable command; suite passes in full, and I killed two mutations myself (checker diff-check, promotion no-reason) |
| Scope              | B | Spec's own diff matches the proposal; the stated range also carries 343 unrelated main commits, reviewed only for interaction |
| Reliability        | B | Consistently fail-closed (exit 2 on missing context/undiffable diff, no force flag past the promotion check); edge cases are false reds, never false greens |
| Maintainability    | A | shellcheck clean, small helpers, comments explain why, every case run under bash and zsh |
| Handoff-readiness  | B | ADR-039 + lesson-301 written, spec evidence recorded; minor staleness in verification counts and the proposal out-of-scope bullet |

### Verdict

PASS

Severity × reality: three REAL Minors and one THEORETICAL Minor, none blocking; rubric all B or
above. The gate fails closed on every path I could attack, its tests bite under mutation, and the
promotion pre-flight runs unconditionally in the archive path.

### Recommended next steps

- All three findings go to `verification.md` dispositions (apply / ticket / decline-with-reason) —
  the contract set is closed by this verdict; do not edit `proposal.md`, `tasks.md` or
  `features.json` on account of it. The fence-in-comment false red deserves a small code fix with a
  named regression test (`tests/check-knowledge-gate.bats`) in a follow-up PR.
- Owner's remaining step, already tracked in tasks Slice 4: run `dotf forge protection apply`, then
  confirm `dotf forge protection check` is clean.
- `dotf spec archive HARNESS-160-knowledge-capture-gate` is **advisable** once the owner applies (or
  explicitly defers) the protection step: the review is fresh against the contract digests, no draft
  tags remain, and all three promotion lines answer with paths that exist.
- Definition of Done, as reviewer: **Debt** — the three Minors above are recorded here (this review
  is their ticket); **Knowledge** — this `review.md` is the artifact; **Board** — untouched by
  design, no status change owed by this run; **Review** — no PR opened, nothing to triage;
  **Evidence** — every claim above carries the command output produced in this session.
