---
spec: "SKILL-001-portfolio-consolidation"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "451ddc195cf723082b505dc8855f89c6eed1fe27"
reviewer: "nan/glm5.3-flash"
date: "2026-09-25"
---

## Adversarial review

**Scope**: SKILL-001-portfolio-consolidation — retire eight skills, fold their rules into three survivors, slim the reviewer/planner/builder rosters without losing a severity, and prove the retirement reached every harness surface. Fourth review round; rounds 1–3 and their dispositions are committed beside the spec as `review-round-{1,2,3}.md`.

**Sources**: `specs/SKILL-001-portfolio-consolidation/{proposal,tasks,verification}.md` + `features.json` (all six features re-run this session); `git diff 737c83c5567fd899c525fb0b13c59a4c34732b70...HEAD` (51 files, +1018/−954; base as resolved and stated by the launcher — the parent of the spec-introducing commit `3c2e393`, not substituted with `main`); `git log 737c83c..HEAD` (7 commits; the delta since round 3, `bd78cd2..HEAD`, is exactly the dispositioned round-3 fixes plus the three committed round records); the vault at `$VAULT_PATH`; `harness/manifest.json`; the deployed surfaces under `$HOME`.

Every claim below was tested by execution, not by reading assertions. Mutations were run and reverted; after the last one `git status --porcelain` shows only the launcher's untracked `review-request.json`.

### Spec and task alignment

- **AC1 holds.** `find 00_meta/skills -mindepth 2 -name SKILL.md` = 31 (the two extra `ls` entries are `README.md`/`CURRENT-STATE.md`, not skills); all eight retired names under `90_archive/skills/` with `status: archived`. f1 re-run: exit 0.
- **AC2 holds, and the round-3 fix bites — red-green verified by me in both YAML forms.** Flow form: `requires: [audit]` injected into `pattern-loader`'s frontmatter → `TestEverySkillTheRouterNamesHasARecord` FAILS naming `audit`. Block form: `requires:` + `- audit` → same FAIL. Appending outside the frontmatter is correctly ignored (records parse frontmatter). f2 re-run: exit 0. A format-agnostic `git grep` over `harness/`, `cli/internal/harness` finds no live router naming a retired skill; the remaining `audit` hits are the trigger id `security-and-quality-audit` (an id and keyword, not a skill — its `skills` list is `["adversarial-review"]`), provenance prose ("once `enrich-us`"), and the word "audit" in unrelated skill prose.
- **AC3 holds.** `dotf harness resolve-skills` prints the three slimmed rosters exactly as the AC lists them; `enforce: warn` counts 3/1/2 by direct frontmatter read; `TestRoleJoinDrift` passes. f3 re-run: exit 0.
- **AC4 holds.** f4's command — which checks each folded section heading *and* a load-bearing rule from each body (round-2 F5 fix) — re-run: exit 0. The surviving records byte-match their vault sources (verified by round 3; `--check` reports no drift at this head, which pins record↔vault equality).
- **AC5 holds.** This session: `go build ./...`, `go vet ./...`, `go test ./... -count=1` exit 0; `compile-harness.sh --check` → "no harness drift"; the six f5-named bats files plus `verify-setup.bats` → **200 ok, 0 failures**. f5 re-run equivalent: exit 0. The full serial suite (1,607 run, 1,606 pass, 1 environmental #1641 failure) was **not re-run this round** — UNVERIFIED here, standing evidence is round 3's run; the delta since touches only the script and its tests, both of which I ran.
- **AC6 holds on this machine.** f6 (`check-retired-skills.sh` with all eight names) re-run: exit 0. The script's mark regex matches what the real renderer actually emits for all three render types I could inspect: `generated_from:` in skill and opencode-command frontmatter, `; from:` in the gemini prompt comment. Fail-closed arms re-verified by my own mutations, not only the suite's: a CRLF-emitting jq shadow with a marked `audit` copy present → exit 1; `del(.skills)` on the manifest → exit 2. The three round-3 arms are each pinned by a named bats case, and the suite ran green.
- **Task ticks match reality.** The `[ ] Independent review before archiving` box is correctly open; this review is it. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tag in any spec file. Issue `mlorentedev/dotfiles#1692` is OPEN and self-assigned; no PR exists on `chore/archive-skill-001`, so there is no CI or reviewer output to disposition. The documented redeploy pending (deployed `adversarial-review` copies still read the pre-round-2 "in this session" wording — confirmed on disk at `~/.pi/agent/skills/adversarial-review/SKILL.md`) matches its recorded plan: redeploy from main after the archive merges.

### Findings

| # | Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|---|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| 1 | Minor | THEORETICAL | fail-closed coverage | `check-retired-skills.sh` guards emptiness of two of its three manifest target lists but not the third: `deploy` and `files` each `fail` when their jq query yields nothing, while `agent_dirs` (the `agents.deploy` → `agent-md` surface) silently checks nothing when empty. The asymmetry is the residue of round-3 finding 3's fix, which added the surface but not its guard. A manifest schema change or key rename on `agents.deploy` would drop the rendered-agent-definition surface from the check **and** from its test: `tests/check-retired-skills-real.bats` "a rendered agent definition that names a retired skill is found" *skips* (`[ -n "$dir" ] \|\| skip`) in exactly that state, so the regression is silent on both sides. | My mutation: `.agents.deploy = []` in a manifest copy + `MUST consume: [adversarial-review, audit]` seeded **only** in `$HOME/.claude/agents/reviewer.md` → **exit 0** (false green). With the manifest intact the same seed → exit 1. Mitigation keeping this Minor: the deploy and instruction-file surfaces still fail closed when they are the dropped ones, the same rosters are mirrored into the `agents.presence` files, and today's manifest does declare the surface — nothing observed, only a rename away. | UNTESTED — no case seeds a manifest whose `agents.deploy` is empty *while* a roster-only copy is present; the existing case skips instead of failing. | code + tests: add `[ -n "$agent_dirs" ] \|\| fail "…no agent-md target…"` (or a deliberate skip-with-reason if a manifest may legitimately render none), plus one real-jq bats case for the arm. Outside the contract set; disposition in `verification.md` or carry into a follow-up ticket. |
| 2 | Minor | SPECULATIVE | verification currency | `verification.md`'s "Test status" cites "the five files this change touches … 192 run" for the round-3 tree; at this head I measured 200 ok across the six f5 files plus `verify-setup.bats`. The discrepancy is bookkeeping (file set and the `check-retired-skills.bats` rewrite changed the count), not a failure — 0 failures either way — and f5, the actual criterion, is exact and re-run green. | f5 re-run this session; the count line predates the final `check-retired-skills.bats` split. | f5 (named, passing) | verification.md (excluded from the staleness check): update the count when recording the next disposition. |
| 3 | Question | — | residual, already dispositioned | The three carried residuals from earlier rounds remain open by design and stay tracked: the pending post-merge redeploy (round-3 #4, confirmed still pending on disk), `pr-review-triage` at `warn` on a reviewer with no PR (round-3 #6 — this review is itself that case, and the warning is harmless), and the count-only routing guard (round-3 #7, exit ticketed as HARNESS-154 #1713). None regressed; none needs action this round. | Direct checks above | each has its recorded disposition | none — tracked |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All six ACs verified true by independent execution at this head; both `requires:` YAML forms and three fail-closed arms mutation-tested; one THEORETICAL coverage asymmetry found and it is narrow. |
| Verification       | A | Every criterion maps to a reproducible command that I re-ran green this session; only the full-suite supporting run was not repeated (round 3's run stands; delta since is the script and its tests, both re-run). |
| Scope              | A | The delta since round 3 is exactly the dispositioned fixes and the three committed round records; nothing unrelated rode along. |
| Reliability        | B | The instrument now fails closed on every arm I could construct except the unguarded `agent_dirs` emptiness (finding 1); the Go side is fail-loud with a fail-loud "nothing was checked" guard. |
| Maintainability    | B | The script is 60 documented lines, shellcheck-clean, runs under bash and zsh (both re-run this session on the clean and detect paths); it reads its targets from the manifest rather than duplicating a list, with the one asymmetry of finding 1. |
| Handoff-readiness  | A | All three prior rounds persisted as committed artifacts, every finding dispositioned with a reason, follow-ups ticketed (#1693, #1712, #1713, #1722), and the pending post-merge redeploy has a recorded owner and trigger. |

Aggregation: no D, no C → PASS by rubric; no Blocker, no REAL Major → no severity escalation. One open Major-class requirement does not exist; the sole open finding is a THEORETICAL Minor.

### Verdict

**PASS WITH GAPS**

The round-3 verdict's minimum set is satisfied and re-proven: the script fails closed (CRLF jq, unreadable manifest, empty target lists — each reproduced or mutation-tested by me, not just asserted), the Go guard reads records' `requires:` in both YAML styles (red-green verified in both), the agent-md surface is read, the round records are committed, and every feature command, the Go suite, `--check` and the touched bats files are green at the reviewed head. What keeps this from a plain PASS is finding 1: the fail-closed doctrine that rounds 2 and 3 forced onto this script still has one unguarded list, and the test that should catch it skips instead of failing. It is theoretical, narrow, and tracked — it does not block the archive, but it must not be lost.

**`dotf spec archive` is advisable in the current state**, with finding 1 dispositioned in `verification.md` (applied here, or ticketed with a root cause) or carried into a follow-up ticket — an edit to `verification.md` does not invalidate this verdict; an edit to `proposal.md`, `tasks.md` or `features.json` would. The post-merge redeploy of the doctrine files must happen as recorded.

### Recommended next steps

- Disposition finding 1 in `verification.md` — fix the script and its test, or file the ticket. It is one guard line and one bats case; it is the same failure shape this spec has spent three rounds eliminating from this check.
- After the archive PR merges: run `compile-harness.sh --deploy` from main and record it in `verification.md` (round-3 #4), updating the "192 run" count while in the file (finding 2).
- No contract-file change is requested by this review; do not touch `proposal.md`, `tasks.md` or `features.json` while this verdict stands.
