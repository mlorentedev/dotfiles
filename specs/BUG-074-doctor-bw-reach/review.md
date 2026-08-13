---
spec: "BUG-074-doctor-bw-reach"
verdict: "FAIL"
reviewed_sha: "13b3d121b40d9aaa3266e724d727a5f0caf571b3"
reviewer: "claude-opus-5"
date: "2026-08-12"
---

## Adversarial review

**Scope**: BUG-074-doctor-bw-reach / PR #950 (branch `fix/doctor-bw-reach`, 2 commits: `145516c`, `13b3d12`)
**Sources**: `specs/BUG-074-doctor-bw-reach/{proposal,tasks,verification}.md` + `features.json`;
`git diff origin/main...HEAD` (merge base `22cc7c5`); `cli/internal/doctor/checks_bw_reach.go`,
`checks_bw_reach_test.go`, `system.go`, `doctor.go`, `testhelpers_test.go`; `cli/internal/env/env.go`,
`cli/internal/secrets/`, `cli/internal/spec/review.go`; `secrets/registry.yaml`; `setup-linux.sh`,
`tests/Dockerfile.integration`, `.github/workflows/ci.yml`; `gh pr view 950` + `gh run view 31663317955`.

### Spec and task alignment

- The three tiers described in `## What` are all present and in the stated order: `bw status`
  (`checks_bw_reach.go:82-111`), `lastSync` age (`checkBWSyncAge`, `:142-160`), authenticated
  `bw sync` round-trip (`:132-136`). `bw list` is correctly rejected in favour of `sync` — the
  reasoning (local cache passes on a dead token) is sound and documented in the code, not only in
  the spec.
- Severity keying is implemented as specified and reads the checkout-preferring
  `env.ResolveRegistryPath()` (`env.go:176-183`), not `cfg.DotfilesDir`. Verified the schema the
  count depends on: `Entry.Backend` with value `"bw"` is the correct predicate
  (`cli/internal/secrets/secrets.go:21`, `registry_write.go SetBackendBW`), and all 34 current
  registry entries are `backend: age`, so today's exposure is genuinely 0 → advisory. AC1-AC3 are
  implemented as written.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file.
- Independently re-ran every `features.json` verification command: 8/8 tests PASS. `go build ./...`,
  `go vet ./...` clean; `golangci-lint run` → `0 issues` on the **pinned** 2.12.2 (matches
  `versions.conf`); `check-spec-gate.sh --base-ref origin/main --head-ref HEAD` → OK (493 prod LOC,
  spec touched). `verification.md`'s "220 production LOC" was captured at `145516c`, before the spec
  commit — both numbers are correct at their own sha; not a finding.
- Two `[ ]` boxes remain: `tasks.md` "PR opened referencing this spec folder" (PR #950 *is* open —
  stale tick), and `verification.md`'s archive checklist including "Promotions above executed". The
  `docs/lessons.md` promotion is correctly marked a *candidate*, not a completed capture, and the
  diff touches no `docs/` file — that is consistent, but it is pre-archive work still outstanding.
- **PR state (not a finding):** `spec-gate` is RED on #950 with "SDD archive-on-merge violation —
  this PR closes an issue whose spec is still active". That is the designed lifecycle ordering: the
  archive is meant to land in this same PR, and `dotf spec archive` gates on this review. All other
  checks are green (lint, test ubuntu + windows, lint-powershell, goreleaser, CodeRabbit,
  GitGuardian). It is process state pending this verdict, not a defect of the change.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Major | REAL | severity source / coverage | The production `bwBackedSecrets()` producer — the seam the proposal itself calls load-bearing — has **zero** test coverage. Every test injects the `BWBackedSecrets` fake, so the `s.Backend == "bw"` predicate has never executed anywhere: not in CI (all fakes), and not in the live smoke either (34/34 registry entries are `age`, so the branch is unreachable today). A wrong predicate, a wrong path, or a parse regression silently returns 0 and pins severity at advisory for the whole #585 migration — the exact failure the risk section says the seam exists to prevent, rebuilt one level down. This is a verification gap, not a shipped defect: the predicate is correct as written. | **Reproduction:** mutated `s.Backend == "bw"` → `"bitwarden"` in `checks_bw_reach.go:183`; mutant compiles; `go test ./...` over all of `cli/` → 13 packages `ok`, **exit 0**, no failing package. The mutant survives the entire suite. (Tree reverted; `git status` clean.) Also: `verification.md`'s mutation row "severity forced to always-advisory → detected" mutated the *consumer* (`if live > 0`, `:99`), not the producer, so the table overstates what was proven. | **UNTESTED** — no test names `bwBackedSecrets`. Needs e.g. `TestBWBackedSecrets_CountsOnlyBWBackend` (temp registry fixture + `DOTFILES_REPO_DIR` override, asserting a mixed age/bw registry counts only the bw entries). | tests (+ correct the mutation row in `verification.md`) |
| Major | THEORETICAL | reliability / timeouts | The two `bw` shell-outs are unbounded. `CommandOutput` is a bare `exec.Command(...).CombinedOutput()` (`system.go:79-82`) with no context or deadline, and `bw sync` is the **only network-bound shell-out doctor makes** — every other `CommandOutput` caller is a local `--version`/`git config` probe. Doctor's own network seam is explicitly capped (`HTTPGet`: `http.Client{Timeout: 5 * time.Second}`, `system.go:99`), so this change introduces a network call that ignores the file's own established precedent. A stalled TLS connection (captive portal, DNS blackhole, VPN half-open) hangs `dotf doctor` indefinitely — and `dotf doctor` is the final step of `setup-linux.sh` (`:1503-1505`), so it hangs a bootstrap too. | Code read of `system.go:79-82` vs `:99`; `grep` of all `CommandOutput(` callers in `internal/doctor/` confirms `bw status`/`bw sync` are the only remote ones. No hang observed — THEORETICAL. | **UNTESTED** — no timeout/hang test exists, and none can, because there is no deadline to assert on. | code (`exec.CommandContext` with a bw-appropriate bound, 30-60s, not HTTPGet's 5s) **or** spec (a documented accepted-risk line in `proposal.md`) |
| Major | REAL | spec accuracy | `proposal.md` risk 3 states: *"Verified that no workflow and no `verify-setup.bats` case runs `dotf doctor`, so the FAIL severity cannot turn CI permanently red once secrets migrate."* The premise is false. `setup-linux.sh:1503-1505` runs `dotf doctor`, and `tests/Dockerfile.integration` ends with `RUN cd /home/testuser/dotfiles-repo && bash setup-linux.sh`, which the `integration` job builds and runs. Doctor **does** run in CI. The conclusion happens to survive, but for two reasons the spec does not record: (a) the invocation is `dotf doctor \|\| log_warning …` — non-fatal, it cannot fail the image build; (b) `bw` is not installed in the container, so the check hits `rep.Skip` before any severity is computed. Both are load-bearing and both are undocumented, so the next person to add `bw` to that image (or to make doctor fatal) will read a risk marked resolved and be wrong. | `setup-linux.sh:1497-1509`; `tests/Dockerfile.integration` final `RUN`; `.github/workflows/ci.yml:246-267`; `grep -n doctor tests/verify-setup.bats` → only a comment, so that half of the claim is true. | UNTESTED (nothing asserts doctor stays non-fatal in setup, or that bw stays absent from the image) | spec (`proposal.md` risk 3 rewritten with the real safety margin) |
| Minor | REAL | reporting | `checks_secrets_tooling.go:37` still emits `rep.Pass("bw (Bitwarden CLI — live secrets SSOT) found")` — verbatim the green line `proposal.md` cites as the misleading artifact of the incident. Doctor now prints presence-dressed-as-SSOT-health *and* a truthful reach section in the same report. The exit code and the findings are correct; the residual is a reading hazard for anyone skimming for green. Deliberate (separate section, to leave the existing tests untouched) but worth one word: "installed" / "on PATH". | `cli/internal/doctor/checks_secrets_tooling.go:37`, unchanged in the diff; `proposal.md` lines 21-23. | `TestBWReach_AbsentBinarySkips` covers the ownership split, but nothing asserts the tooling line's wording. | code (one-line reword) |
| Minor | THEORETICAL | clock | A `lastSync` in the future (clock skew, a restored VM, a machine crossing a DST/NTP correction) yields a negative `age`, so `checkBWSyncAge` reports `rep.Pass("Bitwarden synced -3d ago")`. It passes, which is the safe direction, but prints a nonsense figure and silently treats an impossible timestamp as healthy. | `checks_bw_reach.go:152-159` — `days := int(age.Hours() / 24)` with no lower bound. | UNTESTED (`bwSyncFresh`/`bwSyncStale` are both in the past). | code (clamp or warn on `age < 0`) + tests |
| Minor | REAL | spec hygiene | `tasks.md` "Closing" leaves `[ ] PR opened referencing this spec folder` unticked although PR #950 is open and references the spec folder in its body. | `tasks.md:45`; `gh pr view 950`. | n/a | spec |
| Question | — | keep-alive claim | The code and spec both argue `bw sync` inside doctor "makes a periodic `dotf doctor` the keep-alive that would have prevented this incident outright." That holds only if doctor is actually run periodically *with an unlocked vault* — the sync tier is skipped entirely on a locked vault, which the spec itself calls the normal resting state. Is there an intended cadence (cron, shell hook) that satisfies both conditions, or is the keep-alive incidental to manual operator runs? If the latter, the claim is weaker than stated and should be softened rather than relied on. | `checks_bw_reach.go:115-121` (locked → early return) vs `:128-131`. | n/a | spec (confirm or soften) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All three tiers and the severity policy behave as specified; negative paths (bad JSON, unknown status, never-synced, unparseable stamp, unreadable registry, sync failure) are genuinely covered — the gap is coverage of the producer, not incorrect shipped behaviour. |
| Verification       | C | A producer-side mutant survives the entire `cli/` suite, and `verification.md`'s mutation table claims a stronger result than what was actually mutated. |
| Scope              | A | Diff is exactly the check, its seam, its registration and its tests; no unrelated changes (`git diff --stat` over 5 code files, 4 spec files). |
| Reliability        | B | Degradation paths are careful and never claim reach on unusable input; the unbounded network call is the one unhandled failure mode. |
| Maintainability    | B | Comments explain WHY (why `sync` and not `list`, why the checkout registry, why a separate section); functions are short; `firstLine` reused rather than redeclared. Minor: the misleading legacy PASS line was left in place. |
| Handoff-readiness  | B | Spec artifacts are complete and detailed; the `docs/lessons.md` capture is correctly declared a pending promotion rather than claimed done. |

### Verdict

**FAIL**

Driven by the severity axis, which the skill applies mechanically: one Major → FAIL. The load-bearing
one is the first — Major, **REAL** (a surviving mutant is a reproduction, not an argument), and
**UNTESTED**, which the skill says cannot be closed by the implementer's assurance alone. The rubric
path agrees independently (Verification = C → PASS-WITH-GAPS floor); the more severe path governs.

Note what this verdict is *not* saying: no shipped behaviour was found wrong, the change is
well-scoped and unusually well-reasoned, and every acceptance criterion's *implementation* checks
out. The gap is that the one predicate deciding whether doctor ever escalates to FAIL is proven by
nothing — which is the same class of defect (#898: a check never observed failing is not evidence)
that this spec exists to fix.

### Recommended next steps (before archive)

`dotf spec archive BUG-074-doctor-bw-reach` is **not advisable** in the current state and will refuse
(`checkReviewGate` blocks on `verdict: FAIL`). Minimum set to flip to PASS:

1. **Kill the surviving mutant.** Add a named test for the production producer — e.g.
   `TestBWBackedSecrets_CountsOnlyBWBackend`: write a temp `secrets/registry.yaml` with a mix of
   `age`, `age-offline` and `bw` entries, point `DOTFILES_REPO_DIR` at it, assert the count equals
   the number of `backend: bw` entries. Confirm it goes red under
   `s.Backend == "bitwarden"` before landing it. Then correct `verification.md`'s mutation row to say
   which side (consumer / producer) each mutation exercised.
2. **Resolve the unbounded `bw` calls** — either bound them (`exec.CommandContext` with a
   bw-appropriate deadline; note `bw sync` legitimately needs longer than `HTTPGet`'s 5s), or record
   the accepted risk explicitly in `proposal.md` alongside the `bw sync`-mutates-state entry. Either
   satisfies the gate; silently leaving it does not.
3. **Rewrite `proposal.md` risk 3** with the true chain (`ci.yml integration` →
   `Dockerfile.integration` → `setup-linux.sh:1505` → `dotf doctor`) and the two real safeguards
   (non-fatal `|| log_warning`; `bw` absent from the image), so the escape hatch the risk asks for is
   documented against the actual conditions that hold.
4. Optional, non-gating: reword `checks_secrets_tooling.go:37` to "installed"; clamp negative sync
   age; tick `tasks.md`'s PR box; answer the keep-alive question.
5. Then execute the archive checklist's own pending items — write the `docs/lessons.md` entry
   (declared a promotion candidate: *a health check that reads local state proves liveness of
   nothing*), set `status: archived`, move the folder, close the board ticket.

**Re-review is required, not optional:** steps 1-3 touch `proposal.md`, `tasks.md` and add tests, and
`proposal.md`/`tasks.md` are contract files — so this review goes stale by construction the moment
they change (`cli/internal/spec/review.go`, `contractFiles`). The path to green is fix → one fresh
`/adversarial-review` against the new head, not patch-then-archive against this file.
