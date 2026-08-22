---
spec: "HARNESS-076-model-map-tier-render"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "b644acc0d02eadca125b39008b2f7d949264b6e8"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-21"
---

## Adversarial review

**Scope**: HARNESS-076-model-map-tier-render — PR mlorentedev/dotfiles#1165 (open, head = `b644acc`).
**Sources**:
- `specs/HARNESS-076-model-map-tier-render/{proposal,tasks,verification,features}.json/.md`
- `git diff main...HEAD` (single commit `b644acc`), plus live verification runs in this session:
  `go build/vet/test`, `GOOS=windows go vet`, `golangci-lint run`, `shellcheck`,
  `bash -n`/`zsh -n`, `bats tests/*.bats` (1415/1415), per-feature `features.json` verifier commands,
  end-to-end `--deploy` against the real binary with a mutated (reverted) model-map.json.

### Spec and task alignment

Every acceptance criterion (AC1–AC8) is backed by at least one named test, and I re-ran the named
tests and the end-to-end paths rather than trusting the verification.md claims:

- **AC1/AC2/AC3** — `TestHarnessResolveTier` (7 subtests incl. `the_tier_the_only_deployed_agent_declares`
  top/claude→`opus`) and `TestHarnessResolveTierFailsLoudWithoutAMap` (4 subtests: no map, no schema
  beside it, schema-rejected ghost-pool `chains`, non-JSON) all PASS. Manual smoke of the built binary:
  `top/claude`→`opus` rc 0; `mid/opencode`→`qwen3.6-plus`; `top/copilot` rc 1, stdout empty, error names
  both; `ultra/claude` rc 1.
- **AC4** — bats `agents: --deploy resolves the neutral model tier into the rendered frontmatter` and the
  real-binary `a full deploy renders the resolved model id into the agent file` PASS.
- **AC5** — bats `an unresolvable tier fails the deploy…` + `…leaves the PREVIOUS agent definition intact`
  PASS; I re-ran this end-to-end with the real dotf and a map whose `top` tier names a different harness:
  exit 1, error names `claude` and `top`, pre-seeded `curator.md` (with `model: sonnet`) byte-identical
  afterwards.
- **AC6** — bats `an absent dotf warns…` and `a dotf too old to know resolve-tier warns…` PASS; I also
  exercised the absent-resolver path end-to-end (deploy with an empty PATH): exit 0, `[WARN]` naming the
  tier/harness, `curator.md` rendered without a model line.
- **AC7** — bats `neutral keys dropped` asserts `kind|capabilities|skills|targets` still dropped; the
  updated record render keeps `model` as the only addition.
- **AC8** — new bats cases in `tests/compile-harness.bats`, 5 in `tests/compile-harness-real.bats`
  (capability probe + stdout contract against the built binary), plus the Go table tests.
- Feature verifiers f1–f9 all ran green as written.

The tasks.md implementation and closing boxes are **all unticked** even though the work is done and
verified (`[x]` only on the three Setup lines). The sibling archived spec HARNESS-075 ticks its
implementation boxes, so this is a deviation from repo convention, not a template quirk.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | diagnostics | On the fail-loud path (`rc=1`), `resolve_model_tier` swallows the resolver's stderr (`2>/dev/null`), so a map that cannot be READ (schema-invalid, e.g. ghost pool in `chains`; absent schema) is reported as "tier top does not resolve for harness claude" — a misattribution: the tier resolves fine, the MAP is unreadable. C15 invariants all hold (exit non-zero, nothing rendered, previous definition intact — reproduced), and one manual `dotf harness resolve-tier …` run recovers the true message, so it is a diagnostic-quality gap, not a correctness failure. | Reproduced: deployed with a ghost-pool `chains.top` map; deploy printed only `tier "top" does not resolve for harness "claude" — see harness/model-map.json` while the resolver alone says `chains.top[0] names "ghost" — the \`pools\` block does not declare`. | UNTESTED — no bats/Go case asserts the resolver's own message surfaces in the deploy output. Add one to `tests/compile-harness.bats` (e.g. extend the unresolvable case with a schema-invalid map branch) and a one-line fix capturing/echoing the resolver's stderr on rc=1. | code + tests |
| Minor | REAL | spec artifacts | `tasks.md` implementation and Closing boxes are all `- [ ]` despite completed, verified work (HARNESS-075 archived spec ticks them). Any archive audit reads the spec as mid-implementation. Each task is verifiably done; the boxes should be ticked with the evidence above (per repo convention), not re-implemented. | `grep -c '^- \[ \]'` = 24 vs HARNESS-075's fully ticked implementation list. | N/A (doc state) | spec (tasks.md) |
| Minor | REAL | spec artifacts | `verification.md` "Deviation from AC6 as written" contradicts the committed `proposal.md`: it claims the criterion said the deploy FAILS when `dotf` is absent, but AC6 as committed says it WARNS and renders without a model line (which is what is implemented). The paragraph describes an earlier draft; proposal/tasks/implementation all agree, so this is stale prose, not a behavioral deviation. | Text of `proposal.md` AC6 vs the deviation paragraph in `verification.md`. | N/A | spec (verification.md) |
| Minor | REAL | spec artifacts | `features.json` f6 verifier is positionally brittle and under-covers its own behavior string. `-f 'dotf'` matches three tests — including pre-existing `ENGINE-002: … dotf doctor wires …` — and `grep '^ok 2 '` lands on "absent dotf" by file position; "too-old" is asserted only by the bats suite, not by any feature verifier. All `grep '^ok N '` patterns also pass when the other matched tests fail. | `bats tests/compile-harness.bats -f 'dotf'` → ok 1 ENGINE-002, ok 2 absent-dotf, ok 3 too-old; f6 verification only greps `^ok 2 `. | N/A (the suite itself covers both shapes; the verifier does not) | spec (features.json) |
| Minor | REAL | scope | `ai/claude/settings.json` gains `"outputStyle": "Concise"` — a one-line unrelated change. It is not mentioned in proposal "Risks"/"What", verification.md "Decisions", or the commit message's "supporting changes". No interaction with the tier render (settings.json is not produced by `harness/compile-harness.sh`). | `git diff main..HEAD -- ai/claude/settings.json` → +`"outputStyle": "Concise"` only. | N/A | code (drop to a separate PR) or spec (document the why) |
| Question | THEORETICAL | reliability | `dotf_knows_resolve_tier`'s `dotf harness --help \| grep -q …` under `set -euo pipefail` could, with a large help output, exit 141 via SIGPIPE even after grep matched (grep -q closes the pipe early). Cobra help for this small tree fits the pipe buffer and the real-binary deploy test exercises this exact path repeatedly with zero failures, so no reproduction exists. | No repro; analysis of pipefail + `grep -q` semantics on >64 KiB output. | `tests/compile-harness-real.bats` "harness --help lists resolve-tier" exercises the equivalent grep on the real binary | — (surface only; do not gate) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All criteria met and independently re-verified (Go + stub + real-binary + end-to-end negative with a mutated map); only gap is diagnostic precision on the schema-invalid sub-branch. |
| Verification       | B | Evidence is command-level and reproducible; every claim I spot-ran was green (1415 bats, gofmt only pre-existing #1154, linters at pin); minor count drift (1413 vs 1415; "6 new" vs 10 new) and one stale deviation paragraph. |
| Scope              | B | Diff maps to proposal (Go subcommand, shell render, schema prose, tests, lesson); one undocumented unrelated one-liner (`outputStyle`) pulls it below A. |
| Reliability        | B | Fail paths loud, atomic temp-file write, previous definition proven intact on failure; swallowed-stderr misattribution is a message-precision gap, not an unhandled path. |
| Maintainability    | A | Small pure functions, WHY-comments throughout, single resolution seam, pinned probe string with a Go tripwire test. |
| Handoff-readiness  | A | Lesson 219 in `docs/lessons/`, `features.json` present and non-vacuous, follow-ups #1162/#1163/#1164 filed, archive checklist accurate modulo the unticked boxes. |

### Verdict
PASS WITH GAPS

### Recommended next steps (before archive)
- **Code (optional but cheap)**: capture `dotf harness resolve-tier`'s stderr on `rc=1` and re-emit it under the `[ERROR]` block so a schema-invalid map names the real cause (e.g. the ghost `chains` pool) instead of blaming the tier; add the corresponding bats assertion. Less than 20 lines.
- **Spec artifacts (required for a clean gate)** — apply AFTER this review, and note that each will invalidate `reviewed_sha`, so they belong in a follow-up commit before archive, and the review would then need a refresh:
  - tick the implementation + Closing boxes in `tasks.md` (each verified as done in this review);
  - rewrite the "Deviation from AC6" paragraph in `verification.md` to match the committed AC6 (drop/annotate it as superseded drafting history);
  - tighten `features.json` f6's `-f` filter and `ok N` pins to unique names, and add a verifier that covers the too-old shape.
  - Optionally: drop `ai/claude/settings.json`'s `outputStyle` into a separate PR, or add a "why" line to proposal so the diff is self-explaining.
2. Proceed to `dotf spec archive HARNESS-076-model-map-tier-render` only AFTER the dispositions above are applied or explicitly declined.

Note on hygiene: all mutation edits made during this review (model-map.json variants, renderer/probe/stdout changes in `scripts/compile-harness.sh` and `cli/internal/cmd/harness_resolve_tier.go`) were reverted; the working tree is clean except for this `review.md`, and `harness/model-map.json` at the reviewed sha is pristine.