---
spec: "TOOL-023-pr-agent-incremental-push-review"
verdict: "FAIL"
reviewed_sha: "db856f90772b6446e484812d0e847da21fa548c2"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: TOOL-023-pr-agent-incremental-push-review — the push gate (#1757) plus its #1757-triage hardening (#1773), as it stands at `db856f90`.
**Sources**: `specs/TOOL-023-pr-agent-incremental-push-review/{proposal,tasks,verification}.md`, `features.json`; `git diff 4dad5eadd8857c084499e2474a67b4ac87024ea7...HEAD` (11 spec-relevant files); the gate itself (`scripts/pr-agent-push-gate.sh`), `.github/workflows/pr-agent.yml`; upstream `The-PR-Agent/pr-agent` at `v0.45.0` and `v0.46.0`.

**Evidence produced in this session** (not read from the spec):

| Command | Result |
|---|---|
| `bats tests/pr-agent-push-gate.bats` | exit 0, 22/22 |
| `bats tests/pr-agent-config.bats` | exit 0, 52/52 (tail showed `ok 45..52`, no `not ok`) |
| Mutation: `THRESHOLD=3` → `99` in the gate, re-run push-gate tests, revert | mutation killed — 4 tests red (`not ok`); file restored (`git status` clean) |
| `grep` for any test that reads upstream PR-Agent source | none; `tests/pr-agent-config.bats:133` only pattern-matches the 40-hex pin |
| Upstream source read at both tags | `github_provider.get_commit_range` at `v0.45.0` uses `commit.author.date`; at `v0.46.0` uses `_commit_timeline_date` = **committer** date |

### Spec and task alignment

- AC1–AC4 are all implemented and all have offline coverage: the gate's decision table (`push-gate.bats`, incl. CWE-345 and `head_sha` cases), the workflow wiring and marker registry (`pr-agent-config.bats`), and the ADR/pr-stewardship artifacts. Every `tasks.md` box is `[x]` and traced to `verification.md`.
- `verification.md` is candid: it separates what was verified offline from the three live checks it defers, and it discloses the incremental-baseline rebase limitation and the cherry-pick divergence before I found them. That disclosure is real work and it is what makes the version-drift finding below visible at all.
- Scope caveat (not the author's fault): the launcher's base `4dad5ea` predates ~100 unrelated commits, so the raw diff is 379 files. I reviewed the TOOL-023 surface (`pr-agent-push-gate.sh`, `pr-agent.yml`, `.pr_agent.toml`, `harness/review-attestation.json`, tests, ADR-040, runbook, spec). The unrelated bulk is not attributed to this spec.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | Version drift / correctness of the mirror | The gate mirrors **PR-Agent v0.45.0** (`gate.sh:14,112`; `proposal.md` Risks) but the workflow pins **v0.46.0** (`pr-agent.yml:297`). Between those tags upstream changed `get_commit_range`'s date basis: v0.45.0 compares `.commit.author.date`, v0.46.0 compares `_commit_timeline_date` = **committer** date, with a docstring saying exactly why ("rebasing rewrites content but preserves the author date"). The gate's date fallback still compares `.commit.author.date` (`gate.sh`, date-fallback `jq`). The divergence is present in the reviewed tree and is not disclosed as a version drift anywhere. | Upstream `v0.45.0/pr_agent/git_providers/github_provider.py:214` vs `v0.46.0/.../github_provider.py:245-255`; gate date fallback; `pr-agent.yml:297` | **UNTESTED** — no test reads upstream; no fixture exercises author-date-old/committer-date-new | code (use committer date, matching the pinned tool) + tests |
| Major | THEORETICAL | Silent skip (consequence of the row above) | On the date fallback — reached whenever the baseline review has no `head_sha`, i.e. after any **incremental** review, which the spec itself discloses — a rebase that preserves author dates and rewrites committer dates now makes the gate count **0** new commits (`run=false`) while the pinned v0.46.0 counts them as new. The gate skips, `pr-agent.yml` skips PR-Agent **and the "no review published" guard** (`steps.push_gate.outputs.run != 'false'`), so a genuinely new push lands green and unreviewed — the #1053 symptom, in the opposite direction. At v0.45.0 the blind spot was symmetric (both sides used author date, so `run=false` matched PR-Agent's own "no new commits"); the version bump made it asymmetric. I did **not** reproduce this end-to-end against a live push. | Reasoning from the two pinned implementations above; `pr-agent.yml` guard `if:` | **UNTESTED** — the fixture would be: baseline = incremental review (no state block), commits with old author dates / new committer dates | code + tests |
| Major | REAL | Verification claim in a contract file | `proposal.md` (Risks) states "the tests pin its headings and identities, so a version bump that changes them fails here." That is false: no test reads the pinned action's source or its identity/marker strings; the tests only pin the gate's own copies. The v0.45.0→v0.46.0 bump (commit `25ee6299`, inside this range) changed the identity module and `get_commit_range` and the suite stayed green. The claim is what let the drift go unexamined. | `grep` across `tests/` finds no upstream read; `tests/pr-agent-config.bats:133` checks only the 40-hex pin shape | **UNTESTED** | spec (proposal Risks — contract set; only editable on this FAIL) + tests |
| Minor | REAL | Requirement vs artifact | `proposal.md` is in the contract set and would need the correction above; `verification.md`'s deferred live checks (a below-threshold push stays green, `triage-queue` reads an incremental comment, a timeout on a gated push) remain unexercised in the wild — one of the three now has evidence (#1850), the other two do not. | `verification.md` "Archive pass" | `tests/pr-agent-config.bats` covers the offline half; live half UNTESTED | tests (regression once a live incremental comment exists) |
| Minor | THEORETICAL | Workflow expression executed only at runtime | `github_action_config.push_commands` is now an expression reading `steps.push_gate.outputs.mode`. It is validated textually + by `actionlint`, never executed; if GitHub rejects `steps.*` in a Docker action's `env:` at runtime, the push path silently reverts to the expression's fallback (`["/review"]`, full) — safe direction, but not the intended one. The f7 note argues correctness from the existing `STARTED`/`HEAD_SHA` precedent, which is sound. | `pr-agent.yml` push_commands; `pr-agent-config.bats:303` is a YAML/text check | UNTESTED (runtime) | tests (live push) |
| Minor | SPECULATIVE | Output injection surface | `head_sha` is interpolated into `reason=` written to `$GITHUB_OUTPUT`; a state-block value containing a newline would append a second key (e.g. `mode=incremental`). The value originates from PR-Agent's own state block on a bot-authored comment, so I found no path for a PR author to control it. Surface only. | `gate.sh` `decide()` + `tee -a "$GITHUB_OUTPUT"` | UNTESTED | — (do not gate) |

No Blocker. No injection, secret, or auth defect in the reviewed surface: the gate reads JSON via `jq --arg`/`--argjson` (no string-built shell), fails **open** on every unreadable input, and the baseline author filter (CWE-345) is a genuine fix for a public-repo forgery.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All ACs verified, but the gate's mirror of the pinned tool has drifted (author vs committer date) and no test can detect it. |
| Verification       | B | Reproducible offline evidence (I re-ran it) and honest disclosure of deferred checks; but the "version bump fails here" claim is untrue. |
| Scope              | B | The spec's own surface matches the proposal; the launcher's stale base makes the raw diff far wider than the change, which I excluded explicitly. |
| Reliability        | B | Fails open on every unreadable path and is idempotent, but the date fallback can now skip a push the pinned tool would review, silently. |
| Maintainability    | A | Small functions, low branching, comments explain WHY at length; one stale version reference in a comment. |
| Handoff-readiness  | A | `verification.md` complete, promotion candidates answered, limitations disclosed rather than buried. |

### Verdict
**FAIL** — one **REAL Major**: the gate mirrors v0.45.0 while the workflow pins v0.46.0, and upstream changed the commit-date basis the gate's fallback still uses, with no test able to catch the drift. Rubric has no D, so the severity path is what decides this, as it should.

### Recommended next steps
- **Code** — make the gate's date fallback use the committer date (`_commit_timeline_date`'s basis: `commit.committer.date` falling back to `commit.author.date`), so it mirrors the pinned v0.46.0; or, better, stop mirroring a moving upstream at all and read the reviewed `head_sha` unconditionally.
- **Tests** — add a named case proving the divergence: a baseline with no state block plus commits whose author date is old and committer date is new must not return `run=false`. A test that reads the pinned action's tag and fails when the gate's cited version drifts would close the class.
- **Spec (contract set — permitted on this FAIL)** — correct `proposal.md`'s Risks claim that the tests pin the upstream headings/identities, and record the v0.46.0 pin and its date basis in `verification.md`. A re-review follows the contract edit.
- **Tracked, not blocking on their own** — the live checks deferred in `verification.md`, the untested `push_commands` expression, and the `$GITHUB_OUTPUT` newline surface. Disposition each in `verification.md`: applied, ticketed, or declined with a reason.
- **Not reached / UNVERIFIED** — I did not exercise the live workflow, did not fetch a real incremental review comment (none exists in the wild), and did not verify v0.46.0's `get_pr_review_comment_identifiers`/`get_previous_review` kind-filtering against the gate's "either identity" selection; that is a second, unconfirmed way the mirror may have drifted and belongs in the same fix.
