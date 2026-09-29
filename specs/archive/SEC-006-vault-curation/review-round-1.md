---
spec: "SEC-006-vault-curation"
verdict: "FAIL"
reviewed_sha: "f7abf1998005ea3cb951f48aba7284358abada4e"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-28"
---

## Adversarial review

**Scope**: SEC-006-vault-curation (issue mlorentedev/dotfiles#1784, PR #1790)
**Sources**: `specs/SEC-006-vault-curation/{proposal,tasks,verification,features}.md`; `git diff c114094003855544fe1d399d4dd996064a2ae52b...HEAD`; `git log c114094..HEAD`.

### Spec and task alignment

- All implementation and closing boxes in `tasks.md` are `[x]`, and each acceptance criterion has a named test in `features.json` that I ran (all PASS). `proposal.md` carries no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags, so the archive tag gate is clear.
- Fresh evidence produced this session: `cd cli && go build ./...` exit 0; `go vet ./...` exit 0; `go test ./...` exit 0 (all packages ok); `golangci-lint run ./...` → `0 issues`; the six `features.json` commands all PASS. `verification.md`'s claims are reproducible.
- AC1/AC3/AC4/AC5/AC6 are met as tested. **AC2 is only partially met**: of the proposal's blocking rules, the registry-ownership rule is enforced for the row's target but **not for a `merge-delete` keeper that the op writes to** (finding 1). The proposal's own rule is unambiguous ("the item is one the registry declares, and the op is not `hide` or `reprompt`"), so the code, not the spec, is what diverges — no contract-set edit is required to fix it.
- Deliberate design decisions I checked and accepted as stated: passkey loss is prevented by refusal and only *detected* after `--apply` (AC6 says "fails if", not "prevented by"); `hide`/`reprompt` reaching registry-declared items is declared in `verification.md`; `drop=uris` is a declared loss, not a passkey-rule exception.
- Test-retention check on the out-of-spec `dotfiles-sync` changes in range: the removed execution check in `scripts/test.sh` was replaced by a static source-inspection bats case, and `tests/dotfiles-sync.bats` gained a real "never copies `sensitive/` in either direction" case against temp dirs. Net retention is **stronger**, not weaker — no test-deletion finding. The deleted assertion was itself the hazard (#1795: the pre-commit suite synced the real `$HOME`).
- `review.md` was absent before this run (`review-request.json` carries `review_digest_before: ""`), so this is round 1.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | authorization / registry boundary | `merge-delete` writes the carried URIs to its **keeper** with no ownership check, so a registry-declared keeper is silently edited — exactly the op the proposal forbids on a declared item. `planCurateRow` consults `in.Owned` only for the row target `t`; `planMerge` never checks the keeper `a`, and `applyCurateStep` then calls `s.Curate(st.ArgID, …)`. | Reproduced with a probe test in `internal/secrets` (reverted after): keeper named `registry-item`, duplicate `bbbb0002`; plan `merge-delete bbbb0002 aaaa0001` planned `apply`, `carry=[https://bbbb0002.example]`, and `ApplyCurate` left `writes=2` with the registry item's URIs mutated. `TestCurateBlocks`'s "registry-owned item" case covers only the target (`add-uri`), never a keeper. | **UNTESTED** — no named test asserts a registry-declared keeper blocks (needs e.g. `TestCurateBlocksRegistryOwnedMergeKeeper`) | **code** (`planMerge`/`planCurateRow`: apply the `Owned(a.Name)` gate to the keeper when `Carry` is non-empty, i.e. whenever the op would write to it) **+ tests** |
| Minor | REAL | destructive scope | `delete <id> <same-id>` (or a kept arg resolving to the target) plans `apply` and deletes the item the row declares is *kept instead*; the `kept` precondition only checks presence, never identity. | Reproduced with a probe test (reverted): `delete aaaa0001 aaaa0001` → `state=apply`, and the item was gone after `ApplyCurate`. | **UNTESTED** | **code** (reject `a.ID == t.ID` on `delete`) + **tests** |
| Minor | THEORETICAL | digest completeness | The digest pins rows, resolved ids, item `revisionDate`s and states, but not the **folder namespace**: `folder` resolves by name only at apply time (`s.ResolveFolder(row.Arg)`), so a folder created or renamed between review and apply is invisible to the digest and can file the item elsewhere (or split into a second folder of the same name) without a refusal. | Code read of `PlanCurate` (digest) vs `applyCurateStep`/`ResolveFolder`; no repro attempted. | **UNTESTED** | **code** (include the folder name→id resolution in the digest, or re-check it at apply) |
| Minor | THEORETICAL | quality (repo rule) | Four functions exceed the repo's `< 40 lines` rule: `planOp` 69, `parseCurateRow` 51, `applyMutation` 46, `planMerge` 44. Lint is clean because the `gocyclo` gate does not flag them, so nothing else will catch this. | Measured brace-matched function lengths; `golangci-lint run ./...` = `0 issues`. | n/a (not a test-coverage gap) | **code** (extract the op switch in `planOp`; the rules themselves are correct) |
| Minor | SPECULATIVE | evidence overclaim | `verification.md` says the 12-guard mutation check covers "registry ownership", but disabling that guard is killed only for the target row; the keeper seam had no mutant and no test, so the claim is broader than the coverage (finding 1 is the consequence). Separately, AC5's "every error path" is asserted by three `assertNoValue` call sites, not literally every path. | Finding 1's probe; code read of `curateVault`/`assertNoValue` call sites. | **UNTESTED** | **tests** (+ `verification.md` wording; `verification.md` is outside the contract set) |
| Question | — | scope | The reviewed range contains four other merged PRs' work (WIN-014 spec files, harness mirror, gofmt gate, `dotfiles-sync`/`sensitive/`, ZOHO retirement), not SEC-006's own diff, because base `c114094` predates them. `tasks.md` asserts "No unrelated changes in the diff", which is true of SEC-006's own commits but not of the resolved range. Confirm the range is the intended review surface; if only `5fdf472` is SEC-006, the extra changes have already had their own reviews. | `git log c114094..HEAD` (6 commits); `git diff --stat`. | — | **spec/process** (no code change; no contract edit needed) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All six ACs are tested and green, but one of the proposal's own blocking rules (registry ownership) is unenforced on the merge-keeper seam — a reproduced defect. |
| Verification       | B | Named, reproducible tests plus a live dry-run record; the 12-guard mutation claim is incomplete for exactly the guard that failed. |
| Scope              | B | SEC-006's own diff matches the proposal; the resolved range also carries four other merged PRs, an artifact of the launcher's base, disclosed as a question. |
| Reliability        | B | Idempotent and converge-on-replan; merge carry is non-transactional but re-plans to convergence; passkey loss is detected after the write by deliberate design. |
| Maintainability    | B | Clear naming and WHY-comments; four functions exceed the repo's 40-line rule, invisible to the current lint config. |
| Handoff-readiness  | B | Lessons captured (`lesson-312`, plus the cannot-fail/can-do-damage lesson), spec and `features.json` complete; the archive checklist correctly still open. |

### Verdict

FAIL

One **Blocker**, REAL and reproduced: a `merge-delete` whose keeper the registry declares writes to that item, violating the proposal's registry-ownership rule and AC2's "every blocking rule blocks". Per the severity × reality rule a REAL Blocker forces FAIL regardless of the rubric, and the verdict would be FAIL even on the rubric's own C grade.

### Recommended next steps

- **Fix in code + tests (contract set untouched, so this verdict's contract digests stay valid):** gate `merge-delete` on `in.Owned(a.Name)` whenever the op would write to the keeper (`len(st.Carry) > 0`), and add a named regression test — e.g. `TestCurateBlocksRegistryOwnedMergeKeeper` (keeper registry-declared, duplicate carries a URI the keeper lacks → `blocked`), plus the corresponding mutant in the recorded mutation run. Re-review after the fix; this is the mechanism, not an inconvenience.
- **Fix in code + tests:** reject `delete <id>` when the `kept` arg resolves to the target id; name the test (e.g. `TestCurateRejectsDeleteKeptIsItself`).
- **Track (code):** digest the folder resolution used by `folder` ops so a folder rename between review and apply refuses; and split `planOp`/`parseCurateRow`/`applyMutation`/`planMerge` under the 40-line rule.
- **Correct the record (tests / `verification.md`):** state the mutation check per seam rather than as a count, and either extend `assertNoValue` to the remaining error paths or scope AC5's wording to the paths actually covered. `verification.md` is outside the staleness-checked contract set, so it can be updated without a re-review.
- **Do not archive yet.** `dotf spec archive` is **not** advisable in this state; a `review.md` with `verdict: FAIL` (and, if the gate is age-based, a stale one) will be refused. The minimum set of actions that flips this to PASS: fix finding 1 in code with a named test and re-run the full `go test ./...` + `dotf spec review SEC-006-vault-curation` (the other findings are trackable and need not block).
- **Disposition in `verification.md`:** the implementer should record each finding above as applied, ticketed, or declined with a reason — including the scope question — before the next round.
