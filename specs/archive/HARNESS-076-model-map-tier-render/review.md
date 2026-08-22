---
spec: "HARNESS-076-model-map-tier-render"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "09511cf2ca75a6f682edad346d351b4e0dd04053"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-21"
---

## Adversarial review

**Scope**: HARNESS-076-model-map-tier-render — branch `feat/model-map-tier-render`, base `main`, HEAD `09511cf` (two commits: `b644acc` feature, `09511cf` review dispositions). The previous adversarial review ran at `b644acc` (PASS-WITH-GAPS, six findings); this is a fresh review of the dispositioned state.
**Sources**:
- `specs/HARNESS-076-model-map-tier-render/{proposal,tasks,verification,features.json,review.md}`
- `git diff main...HEAD` (20 files, +1461/−17), both commits read individually
- Fresh runs in this session: `go build/vet/test ./...` (all packages ok), `GOOS=windows go vet ./...`,
  `golangci-lint run` @ pin `2.12.2` (0 issues), `shellcheck --severity=error` on
  `scripts/compile-harness.sh` + `setup-linux.sh` (clean), `bash -n`/`zsh -n` (clean),
  full `bats tests/*.bats` → **1418 ok / 0 not ok** (exactly the verification.md count),
  `scripts/check-bats-names.sh tests/` → OK (99 files), every `features.json` verifier run
  as written (f1, f4, f6, f6b, f6c, f8, f9, f10 all green), a real end-to-end `--deploy`
  against the dotf built from this tree, and five mutation experiments on a throwaway
  fixture tree (whitespace-id map, tier removed, map absent, map non-JSON) with the real
  binary — all reverted afterwards, working tree clean.

### Spec and task alignment

All eight acceptance criteria are backed by named tests, and I re-ran each rather than
trusting verification.md:

- **AC1/AC2/AC3** — Go table tests `TestHarnessResolveTier` (7 subtests) and
  `TestHarnessResolveTierFailsLoudWithoutAMap` (4 subtests) PASS; manual smoke of the built
  binary confirms `top/claude → opus` rc 0, `top/copilot` rc 1 with both names on stderr and
  empty stdout, `ultra/claude` rc 1. `TestHarnessHelpListsResolveTier` pins the probe string.
- **AC4** — bats `resolves the neutral model tier into the rendered frontmatter` and real-binary
  `a full deploy renders the resolved model id into the agent file` PASS; my own end-to-end run
  of the real script + real binary wrote exactly one `model: opus` line into `curator.md`.
- **AC5** — bats `unresolvable tier fails the deploy` + `leaves the PREVIOUS agent definition
  intact` PASS. Mutations with the real binary confirmed: an unresolvable/schema-rejecting map
  → deploy exits 1, wrapper `[ERROR]` names tier+harness, the pre-seeded definition stays
  byte-intact, zero `.tmp.$$` files left behind.
- **AC6** — bats `absent dotf warns...` + `too old to know resolve-tier warns...` PASS. I also
  reproduced the too-old path for real: this machine's installed `~/.local/bin/dotf` predates
  the subcommand, and a deploy that found it (not the fixture's stub) warned loudly and
  rendered without a model line — exactly C15's inverse branch, working as designed.
- **AC7** — bats `neutral keys dropped` asserts `kind/capabilities/skills/targets` still
  dropped; `model` is the sole addition. Confirmed against the rendered artifact.
- **AC8** — 11 new bats cases in `tests/compile-harness.bats`, 5 in
  `tests/compile-harness-real.bats` (probe + stdout contract against the built binary), 3 Go
  tests (12 subtests). Every `features.json` verifier (f1–f10) runs green as written and now
  propagates exit status (`out=$(...) && grep`) and pins by unique test NAME — the exact two
  gaps the prior review's f6 finding asked to close.

**Prior-review disposition check.** The six findings of the `b644acc` review are all
dispositioned at this head, and I verified each by running, not reading: (1) the resolver's
stderr is no longer swallowed — mutations M3/M4 and bats `the resolver's own diagnosis reaches
the deploy output` prove the cause (e.g. `ghost` pool / `invalid character`) survives into the
deploy output; (2) the probe is pipe-free (here-string) — code read + stale-binary bats case;
(3) `agent_model_line` extraction — measured ≤20 executable lines each in the changed set;
(4) `outputStyle` merge policy now named on both platforms with a cross-OS guard test — the
negative control (removing the jq clause) went red, then I reverted it; (5) `features.json`
verifiers exit-status/name-pinned — f4/f6/f6b/f6c/f8/f9/f10 all green as written; (6) the
SIGPIPE question — eliminated by the here-string. `tasks.md` boxes are all ticked, and the
AC6 deviation prose is gone from `verification.md`. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]`
tags remain in any spec file. `features.json` has zero `passing` entries, so the
pass-state-gating rule is respected.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | code (setup-linux.sh) | In `merge_claude_settings`, the guard `if ($tmpl.outputStyle // empty)` does NOT "leave the existing value alone if a template ever omits it" as its comment claims — in jq, a condition evaluating to `empty` makes the whole `if` expression (and thus the whole merge pipeline) yield nothing, so a template lacking `outputStyle` silently disables the ENTIRE settings merge (model + effortLevel included), not just that key. Reproduced on this machine (jq 1.8.1): `echo '{"a":1}' | jq '.a=2 | (if (null \|\| empty) then .b=3 else . end) | .c=4'` → zero output. Unreachable today only because the shipped template always carries `outputStyle: "Concise"` and a bats test pins that; the next template edit that drops or nulls the key trips it. The `[ -z "$merged" ]` guard turns it into a warning + no write, so no corruption — a silent full-merge no-op with a misleading comment. | Live jq reproduction above (2026-08-21, jq 1.8.1). | UNTESTED — no bats/Go case runs the merge with a template lacking `outputStyle`; the existing policy test only asserts the key's presence. | code + tests (use `has("outputStyle")` in the jq, and/or add a merge-with-missing-key bats case) |
| Minor | THEORETICAL | code (scripts/compile-harness.sh) | The model-id sniff `case "$out" in ''\|*[[:space:]]*) return 2` reuses the rc=2 warning text ("dotf is absent, or predates the resolve-tier subcommand") for a resolver that RAN FINE and returned a whitespace-bearing id. The schema's `pattern: "^\\S.*$"` permits trailing/interior whitespace (`"opus "`, `"opus 4"`), so a typo'd hand-edited map historically reaches this sniff. Reproduced: map `"opus "` → deploy warns "dotf is absent…" while dotf is current and answered, renders no model line, exits 0. Misattributed cause in a corner that is otherwise well-guarded. | Reproduction: MAP mutation (reverted) with `"opus "` for top/claude; script emitted the absent-dotf warning. | UNTESTED — no bats case drives a whitespace-bearing id through the sniff (the stub covers only empty-output and fail shapes). | code (distinct message/rc for malformed output) or explicitly documented as belt-and-braces |
| Minor | REAL | spec artifacts (verification.md) | Count drift: verification.md AC8 says "6 new bats cases in `tests/compile-harness.bats`" and "5 in `tests/compile-harness-real.bats`"; the diff adds **11** new cases to compile-harness.bats (real suite: exactly 5 ✓). Go counts check out (3 tests / 12 subtests ✓), and the 1418-passing claim is exact, so this is prose drift, not an evidence gap. | `git diff main...HEAD -- tests/compile-harness.bats` → `+@test` count = 11 | N/A (count wording) | spec artifacts (verification.md) |
| Minor | REAL | maintainability | `deploy_agents` is 52 raw lines vs the repo's own <40-line rule (28 on main; the temp-file/mv discipline added ~24). Resolution moved correctly into `agent_model_line` (13 executable lines), so this is the deploy loop's remaining bulk — and 6 legacy functions in the same file (do_refresh 112, do_check 85, deploy_skills 68, deploy_doctrine 57, render_skill 55) already exceed 40, so the rule is demonstrably not enforced in this file. Noted for the record; not a gate. | measured line counts on HEAD | n/a (structural) | code (optional extraction of the per-record temp-write into a helper) |
| Question | THEORETICAL | design (deploy semantics) | `deploy_agents` `return 1`s on the FIRST unresolvable record, aborting the whole agent loop — earlier agents' freshly-written files stay, later agents never deploy. With today's single-agent `agents.deploy` set (claude only) this is unreachable; when the set grows, one broken record blocks every agent behind it. This matches AC5's fail-loud decision; the expand-and-flag alternative was never formally chosen. Surface only. | code read of the `while ... done < <(jq ...)` loop | bats `an unresolvable tier fails the deploy` (single-agent fixture) | spec note (no action required today) |

**(no injection vector found.)** `tier` and `agent` flow from repo-controlled AGENT.md frontmatter and manifest.json into a quoted command argument (`"$tier"`), so no shell re-evaluation occurs; the whitespace sniff additionally rejects any id containing whitespace. A malicious record could at worst write a literal `model: $(...)` string into the rendered file — inert text, never executed. Noted as considered-and-excluded.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All ACs 1–8 met and re-verified end-to-end with the built binary and 4 fixture mutations; two reachable-but-pinned corner diagnostics are imprecise (jq `// empty`, whitespace sniff mislabel). |
| Verification       | A | Every claim independently reproduced: 1418 bats, Go + double cross-vet, lint at pin, features verifiers green as written, negative control on the settings guard, real too-old-dotf reproduction; one factual count typo in the prose. |
| Scope              | A | Diff maps one-to-one to the proposal; the single unrelated change (`outputStyle`) is declared in the proposal's Out of scope, justified, and now policy-guarded on both platforms. |
| Reliability        | B | Fail paths loud, C15 invariants verified under 4 mutation shapes, atomic temp+mv with cleanup on both failure legs, previous definition proven intact; the two Minor diagnostics are message-level, not unhandled paths. |
| Maintainability    | B | Small pure renderer (13 lines), one seam, WHY-comments, pinned probe string with a Go tripwire; `deploy_agents` remains over the repo's 40-line rule (as do 6 legacy siblings). |
| Handoff-readiness  | A | All six prior findings dispositioned and re-verified, lesson 219 written, features.json non-vacuous and exit-propagating, follow-ups #1162/#1163/#1164 filed, archive checklist accurate. |

### Verdict
**PASS WITH GAPS** — no blockers, no majors, no C/D rubric grade; four Minor + one Question,
all non-gating, all tracked above. The rubric is all-A/B, so this could stand as PASS; the
four minors each ask for a concrete follow-up (one 1-line jq change + one test case, one diagnostics
message, one doc-count fix, one optional extraction), so the honest label — matching
this repo's disposition cadence — is PASS WITH GAPS.

`dotf spec archive` should treat this review as **fresh** (`reviewed_sha` = HEAD; no change
to `proposal.md`/`tasks.md`/`features.json` after it) and **advisable**: the archive gate's
two pre-flights are satisfied, all prior dispositions landed, and none of the four minors
blocks archiving. Apply them post-archive as follow-up issues, or land them first as a
follow-up commit and re-run the review — either is defensible.

### Recommended next steps (before archive)
1. (Optional, cheap) **Code fix for the jq guard**: change `(if ($tmpl.outputStyle // empty) …)`
   to `(if ($tmpl | has("outputStyle")) …)` — the simplest correct guard — OR
   add a bats case driving `merge_claude_settings` with a template that omits `outputStyle` and
   asserting the rest of the merge still applies. The negative control test
   already proves the current policy latch works for the present template.
2. (Optional, cosmetic) distinguish the whitespace-sniff WARN from the absent/too-old message
   (e.g. "resolver returned a malformed model id (has whitespace)") + one bats case.
3. (Trivial) correct the "6 new bats cases" count in verification.md to 11 (or "a dozen").
4. (Record only) note the single-resolver abort semantics of `deploy_agents` in the proposal
   when `agents.deploy` ever grows beyond one entry.
5. Proceed to `dotf spec archive` — the gate's two pre-flights (fresh `review.md` signed by a
   pool member, no stale contract files) are satisfied at `09511cf`.

Note on hygiene: all mutation edits were confined to a `/tmp` fixture tree (never the repo);
the settings-guard negative control edit to `setup-linux.sh` was reverted in the same session,
and `git status` is clean except this `review.md`. The `review-transcript.jsonl` beside this
file is the harness's live session-annotation target for the review run and was left untouched.