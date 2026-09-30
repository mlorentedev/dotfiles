---
spec: "CLI-091e-dotf-as-a-product"
verdict: "PASS"
reviewed_sha: "4f77bc51bdfcd04ab8ec575c1917776ecd169553"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: CLI-091e-dotf-as-a-product (track E row E1, round 2), whole diff `0013ea5b8b45cec7ab950e91b93eb8f1defc0025...HEAD` (`4f77bc51`), not the delta since round 1.
**Sources**: `specs/CLI-091e-dotf-as-a-product/{proposal,tasks,verification}.md` + `features.json` + `round-1` artifacts; `git diff 0013ea5b...HEAD` and `git log 0013ea5b..HEAD`; `git diff 10116e85...HEAD` for the round-2 fix; `cli/internal/cmd/help_text_test.go`, `cli/internal/cmd/orca.go`, `cli/README.md`; built binary `/tmp/dotf-r2`; live `go build`, `go vet`, `go test ./...`; mutation edits reverted.

### Spec and task alignment

- **AC1 now holds, and the round-1 Blocker is genuinely closed.** `dotf orca`'s Example and `dotf orca tune-hooks`' Short and Long no longer cite `DX-006` or "lesson 111" (built binary, both `--help` outputs). I re-ran the exact round-1 mutation: reinserting `(DX-006)` into `tune-hooks`' Short turns `TestHelpTextHasNoInternalReferences` red on that line. The fix is not cosmetic — it changed the guard's source of truth.
- **The guard's declared new mechanism works.** `internalRef` now seeds its prefix list from the spec AREAs under `specs/` and `specs/archive/` (via `specArea`) on top of `fixedPrefixes`, and `t.Fatalf`s if `specs/` is unreadable or yields no new prefix. Mutation: a prefix present **only** in `specs/` (`CHORE-002`, absent from the fixed list) is caught red, proving discovery — not just the fixed list — is load-bearing. Bare `GUARD` (no number) is still green (see finding 4); that is a known, tracked gap.
- **AC2 holds.** `cli/README.md` has `## Install`, `## Commands`, `## Develop`; `TestReadmeHasNoInternalReferences` is green. PR-Agent's round-1 claim that the README advertises a non-existent `install --dry-run` is **refuted at HEAD**: the flag exists (`cli/internal/cmd/tools.go:71`, live `dotf tools install --help` prints it). PR-Agent's "duplicated guard pattern" finding is also already fixed — f2 no longer embeds its own regex, it delegates to the Go test.
- All `[x]` tasks map to diff or execution evidence. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain in the spec folder.
- Freshness: the launcher's `contract_digests` in `review-request.json` match the current `proposal.md`, `tasks.md` and `features.json` (`git diff` shows no change to those three since the request), so this review describes the current contract.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | verification evidence | `verification.md` contradicts itself: line 10 still asserts "49 failures before the rewrite", while the round-1 disposition (line 38) records that 49 is wrong and that 51 was reproduced. The correction was written in the table but never applied to the evidence line it describes ("this line is the correction"). The disposition is therefore transparent, but the artifact still carries the disproven number. | `verification.md:10` vs `verification.md:38`; `tasks.md:23` (49, contract set, left as reviewed) | UNTESTED (it is an evidence number, not a behaviour) | spec (`verification.md`, outside the contract set — free to fix) |
| Minor | REAL | spec-vs-code (scope) | The proposal declares error messages out of scope ("They are a later row"), yet the diff removes `(#770)` from `spec init`'s WIP-limit **error** string (`cli/internal/cmd/spec.go`, not `--help`). AC1 covers `--help` only, so this is a declared non-goal crossed. It is deliberate and dispositioned in `verification.md` ("Kept… recorded here as the one exception"); the flag-usage half of that same edit **is** in AC1 scope and is legitimate. | `git diff 0013ea5b...HEAD -- cli/internal/cmd/spec.go`; `verification.md` disposition table | `TestSpecInitRefusesAtWipLimit` asserts substrings, not the removed text | spec (proposal is contract set → disposition in `verification.md`, already recorded; or follow-up ticket) |
| Minor | REAL | spec staleness | The proposal's own risk note is now stale: it says "The guard's id pattern is a list of known prefixes. A new prefix would pass unnoticed, so the list sits next to the test for a reviewer to extend." Post-fix, spec areas are discovered from `specs/`, so that sentence describes the pre-fix design. The residual (non-spec prefixes in `fixedPrefixes`) is the real, narrower risk. `proposal.md` is contract set, so this cannot be fixed under a passing verdict. | `proposal.md:41`; `help_text_test.go` `internalRef` | mutation 2 (`CHORE-002` red) proves the risk text is obsolete for spec areas | spec (contract — leave as reviewed; disposition in `verification.md` or next contract touch) |
| Minor | THEORETICAL | guard completeness | The guard checks `Short`, `Long`, `Example` and flag `Usage` only — not `Use`, `Aliases`, flag default values or `Deprecated`; the issue pattern `#[0-9]{2,5}` misses 1- and 6+-digit issues; a bare prefix with no number is unmatched. No live instance: no `Use`/alias id exists, `dotf hooks install`'s bare-`GUARD` Short that PR-Agent flagged is already rewritten, and `grep 'GUARD' hooks.go` finds only comments. Latent, not observed. | mutation 3 (`(GUARD)` green); `help_text_test.go:62-67`; `grep` on `cli/internal/cmd/*.go` | UNTESTED | tests — tracked in **#1891** (OPEN, "Keep internal ids out of runtime output and the unchecked help fields") and in #1850's review triage |
| Minor | SPECULATIVE | jargon | Jargon outside the id rule survives: README/`--help` still say "bitacora", "00_meta/", "dispatcher". AC1's letter is "an internal id or 'twin'", so none is in scope; surface only, do not gate. | `cli/README.md`; live `--help` samples | UNTESTED | code (optional) / spec if intended |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Both ACs met and mutation-verified (DX and a specs-only prefix both go red); latent guard gaps remain but are latent, tracked, and outside AC1's stated fields. |
| Verification       | B | `go build`, `go vet`, `go test ./...` (27 packages green), both `features.json` commands, and three mutations reproduced here; the "49" evidence number is still inconsistent and `golangci-lint` was not re-run. |
| Scope              | B | Spec-attributable delta is tight (help text + test + README); the range additionally carries upstream merges `c462befc` (#1850) and `8d7aca59` (#1883, CLI-090 archive, TOOL-023, PS test) that are on `origin/main` and not this change's creep; one out-of-scope error string. |
| Reliability        | A | Documentation/help-text-only change plus one deliberate message edit; the guard fails closed (`t.Fatalf` on unreadable `specs/`, `t.Fatal` when no spec area is found); tree builds and vets. |
| Maintainability    | B | Guard is short, WHY-commented, and replaces a hand list with discovery; residual smell is the `len(prefixes) == len(fixedPrefixes)` "found none" proxy, which fails closed but is not "found all". |
| Handoff-readiness  | B | proposal/tasks/verification/features updated in-session, promotion lines answered with reasons, follow-ups tracked (#1891, #1890); one evidence number inconsistent and the proposal risk text stale. |

### Verdict
PASS

Round 1's **Blocker** (`DX-006` in live `--help`, guard green because `DX` was missing from a hand-kept list) is fixed and independently confirmed red-green. Every remaining finding is **Minor**, and no **REAL Major** or Blocker stands; the rubric has no C and no D, so the mechanical aggregation is PASS. The guard-completeness gap is latent (no live instance), tracked in the open #1891, and cannot by itself move the verdict.

### Recommended next steps

Route by set. The **contract set is closed** by this verdict: editing `proposal.md`, `tasks.md` or `features.json` now would invalidate this review, so the two contract-set observations (findings 2 and 3) go to `verification.md` as dispositions, not to the contract.

1. **`verification.md` (outside the contract set — free to edit):** apply the correction it already records — change the AC1 evidence line from "49 failures" to the reproduced figure, or drop the number and name the command and base commit; and record the `spec init` `(#770)` error-string edit as an in-scope exception (already partly done).
2. **Follow-up (#1891, already open):** the unchecked help fields (`Use`/`Aliases`/defaults), the bare-prefix case, and the 1-/6-digit issue pattern. Consider a committed test asserting the guard fails when a spec area is *absent* from the derived list (the current `len == len(fixedPrefixes)` check only catches "found none"), so discovery cannot silently narrow.
3. **No code change is required for this archive.** `dotf spec archive CLI-091e-dotf-as-a-product` is advisable in the current state: PASS is a recognized passing verdict, the contract digests are fresh, and no `[AGENT-DRAFT]` tag remains.

### Unverified

- `golangci-lint` was not re-run in this session; `tasks.md`'s "golangci-lint 2.12.2, 0 issues" is taken on the CI `cli-lint` claim plus round 1. UNVERIFIED here.
- `go install github.com/mlorentedev/dotfiles/cli/cmd/dotf@latest` was not executed (needs network); the README's install paths are judged from the file, not run end-to-end.
