---
spec: "HARNESS-196-vendor-security-audit-web-perf"
verdict: "PASS"
reviewed_sha: "08652a12fa81e7aae600dc05887029123fa1044d"
reviewer: "nan/glm5.3-flash"
date: "2026-11-13"
---

## Adversarial review

**Scope**: HARNESS-196-vendor-security-audit-web-perf
**Sources**: `specs/HARNESS-196-vendor-security-audit-web-perf/{proposal,tasks,verification,features}.md/json`; `git diff 0bd88724f5d4835ed02cf466a0f0d3ddcfc1b69b...HEAD` (3 branch commits: b202c73d, cda7c81a, 08652a12)

### Spec and task alignment

- The diff matches the proposal exactly: two new skills (`security-audit` SKILL.md + 3 script files; `web-perf` SKILL.md), re-render of the six drifted records (builder and reviewer agents; adversarial-review, agent-lifecycle, handoff, using-git-worktrees), the one-line `DefaultSkillDependencies` entry, two ATTRIBUTION rows plus the scripts exception sentence, and the spec folder. No unrelated changes observed (the range holds only the 3 branch commits; `git log --first-parent` confirms).
- All 5 tasks-implementation boxes have diff evidence; the two claimed red-green turns were reproduced (see findings-free evidence below).
- `review-request.json` contract digests match `dotf spec digest` output for all three contract files (features.json, proposal.md, tasks.md). No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder.
- No test was deleted in the range.

### Evidence run in this review (fresh)

- `bats tests/harness-generated-sha.bats` → 3/3 ok (AC1 first half).
- `go run ./cmd/dotf harness check --repo-root ..` from `cli/` → "[check] OK: no harness drift", exit 0 (AC1 second half).
- `go test -count=1 -run TestTheCompiledDependencyMapMatchesTheRecords ./internal/harness/` → ok (AC2). Red-green reproduced: deleting the `"security-audit": {"test"}` line from `triggers.go` made it fail with exactly the claimed message ("the compiled map says [], its record's requires: says [test]"); file restored, tree clean again.
- `bats --filter 'HERMES-018' tests/skills-pipeline.bats` → 4/4 ok, plus the f3 grep chain (row licenses vs record `license:` fields: MIT and Apache-2.0 respectively) → all pass (AC3). Upstream licenses independently confirmed through the GitHub API: `cloudflare/security-audit-skill` → MIT, `cloudflare/skills` → Apache-2.0.
- AC4 re-verified byte-for-byte: `gh api .../contents/...?ref=c1c8a8c1471069fb0e188eeaff69b8e8db6564a8` piped through `cmp` for all three files → identical. The pinned commit is confirmed as upstream HEAD as of 2026-11-13 (committer date 2026-09-14T19:28:54Z).
- AC5: no file under `harness/skills/security-audit/scripts/` is `u+x`; `git grep validate-(findings|coverage-ledger)` over `cli scripts .github setup-linux.sh` → nothing.
- Scripts inspection spot-checked live in a `mktemp -d` sandbox, not just read: deeply-nested JSON (>64 depth), a symlink input, a missing input and a 6 MiB input all exit 1 with the specific refusals claimed in `verification.md` (symlink refusal, safe-open failure, 5242880-byte limit) and bounded output (74 bytes for the deep-nesting case). Static grep of both `.cjs` files (1645 lines) confirms only `fs`/`path`/`util` imports; no `child_process`, network, `eval`/`Function`, `process.env`, or any write/delete/rename API. `report-schema.json` parses as valid JSON.
- Broader suite: `go test -count=1 ./...` in `cli/` → all packages pass; `go vet ./...` and `GOOS=windows go vet ./internal/harness/` → clean; `golangci-lint run ./internal/harness/...` → 0 issues; bats over skills-pipeline, roster-consistency, agents-md, reviewer-pool, harness-suggest, guard-review-verdict-honours-reality → 65 ok, 0 failures.
- Gate claim (proposal: reviewers see `security-audit` as warn, call passes through) matches the code read (`cli/internal/harness/gate.go`: `warned` returned with `Decision: Allow`, "reported, never enforced"); the invited `inferPersonaFromSkill` mapping of `security-audit` → reviewer pre-existed, no interaction issue.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | attribution | `harness/skills/web-perf/SKILL.md:22` body still says "(`skills/web-perf`, MIT)" while the record's `license:` field and the ATTRIBUTION row both say Apache-2.0 (upstream confirmed Apache-2.0 in this review). The vault fix #3470 corrected only the license field; the prose Origin line was left stale and re-rendered as-is. | Line 22 of the committed file vs `gh api repos/cloudflare/skills/license` → Apache-2.0 | `HERMES-018` checks the ATTRIBUTION row against the record's `license:` (both Apache-2.0, so it passes); the body prose is UNTESTED | vault (SKILL.md prose is a declared non-goal of this spec) + a future re-render |
| Minor | REAL | attribution | `ATTRIBUTION.md` blanket sentence "All sources verified by reading the repository LICENSE file at vendor time (2026-06-03)" now also reads over the two new rows, whose licenses were read 2026-11-13. The durable notice overstates the vendor date for two of its rows. | ATTRIBUTION.md unchanged sentence in this diff vs proposal "Both licenses were read from the upstream repositories' LICENSE files on 2026-11-13" | UNTESTED (doc sentence) | code (`harness/skills/ATTRIBUTION.md`, outside the contract set — fixable alongside the next re-render) |
| Minor | REAL | spec prose | proposal "Out of scope" says the upstream skill ships "13 reference documents"; the tree at the pinned commit `c1c8a8c14` contains **14** reference `.md` files beside SKILL.md (AI-AND-LLM … WEB-PROTOCOL-AND-AUTH). Substance unchanged — none are vendored (the committed dir holds only SKILL.md + `scripts/`). | `gh api .../git/trees/c1c8a8c1...?recursive=1` counts 14 reference `.md` files | UNTESTED | spec (proposal.md is contract set — tracked here, not edited under a PASS) |
| Question / assumption | SPECULATIVE | deploy path | A future `dotf harness refresh` that vendored upstream's own reorder (scripts moved from skill root into `scripts/`) is assumed to keep the `requires:`/license fields stable through the render pipeline rather than needing another hand fix. | Code read of gates around refresh refusal behavior; not observed | UNTESTED | — (surface only; follow-up ticket if it recurs) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All five ACs reproduced by command in this review, including an independent red-green for AC2 and live negative-path probes of the vendored validators. |
| Verification       | A | `features.json` verification commands are all executable and pass as written; `verification.md` claims survived independent re-execution, including the byte-identity `cmp` against upstream. |
| Scope              | A | Diff matches the proposal file-for-file; the only range noise is the intended pre-spec re-render commits the launcher attributed to this work. |
| Reliability        | A | No runtime code introduced in this repo; the vendored validators refuse symlinks/oversize/over-depth inputs, verified live in a sandbox. |
| Maintainability    | A | One-line Go change plus generated records; vet, windows-vet, golangci-lint and the touched test surfaces clean. |
| Handoff-readiness  | A | Spec triad complete and consistent with the diff, promotion candidates answered with reasons, only the PR-open step (correctly) still open. |

### Verdict
PASS

No Blocker, no REAL Major; the three Minors are attribution-prose inaccuracies and a miscounted out-of-scope number, none of which any acceptance criterion reads on. Rubric holds at A across the board.

### Recommended next steps

Routed by set. **On a PASS the contract set is closed** — these are for the implementer to disposition in `verification.md` (applied / ticketed / declined) or carry into follow-up work, not contract edits.

- Ticket the web-perf body license line (`skills/web-perf`, MIT → Apache-2.0) to the vault; the next re-render will correct it here. The ATTRIBUTION date sentence can be fixed in the same pass (repo-side, no spec involvement).
- The "13 reference documents" miscount stands in the archived record; correct it opportunistically if this spec is ever amended — do not re-open the contract for it.
- The per-verification.md note stands: a local vault checkout lagging `11bacfaf` fails `roster-consistency-real.bats` until it pulls; CI is unaffected. The #3456-named drift-detection follow-up remains its own ticket, correctly out of scope here.

`dotf spec archive` is advisable in the current state, once the PR is open per the tasks checklist; the three Minors above are tracked, not gating.
