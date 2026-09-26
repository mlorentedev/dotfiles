---
id: "HARNESS-024-knowledge-capture-gate"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#387"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, lessons, adr, ci]
template_version: "1.0"
---

# HARNESS-024: the PR that produces knowledge names where it went

## Why

Less of the harness's work reaches `docs/lessons/` and `docs/adr/`, the surfaces later sessions search:

- dotfiles lessons per commit fell from 0.42 in W36 to 0.18 in W39;
- W39 had 23 `fix:` commits and 11 lessons, against 24 and 32 in W36;
- there has been no dotfiles ADR since W36.

The knowledge is still written, but in ticket and PR bodies (vault note `10_projects/dotfiles/research/2026-09-25-knowledge-capture-routing.md`). Specs and reviews have mechanisms that ask for them: `spec-gate` and `review-attestation`. Lessons and ADRs have only text at the end of the work: Definition of Done §2 and the handoff's harvest step. That text loses to the next task. The archive's promotion step is the same: `dotf spec archive` accepts a "no", or no answer, without looking.

## What

1. **A knowledge gate on every PR.** The PR body carries a `## Knowledge` section with three lines:

   ```markdown
   ## Knowledge

   - Lesson: docs/lessons/lesson-301-knowledge-goes-where-a-mechanism-asks.md
   - ADR: none: no contract or policy changed
   - Runbook: none: no new procedure
   ```

   - Each line holds one or more paths, or `none: <reason>`.
   - A path counts when all three hold:
     - the PR adds, changes or renames it;
     - it sits under its kind's directory: `docs/lessons/`, `docs/adr/` or `docs/runbooks/`;
     - it is not `_index.md`.
   - A `none` needs a non-empty reason.
   - The gate fails when the section or a line is missing, or when a line holds neither a counting path nor a reasoned `none`. A section inside a fenced code block does not count.
   - It runs in CI as `knowledge-gate`, on the events `spec-gate` runs on. It reads the body live, so editing the body re-runs it.
   - Dependency-bot PRs are skipped the way `spec-gate` skips them.
   - Release PRs pass on content: the release-please footer carries the section.
2. **`dotf spec archive` enforces the promotion step.** Each promotion line in `verification.md` must be answered `yes: <path>` or `no: <reason>`.
   - An unanswered line refuses the archive, naming the line.
   - So does a `yes` whose path does not exist. Lesson and ADR paths resolve against the repo; a pattern path resolves against the vault (`VAULT_PATH`).
   - No force flag skips it. The way out is answering the line.
3. **Agents learn the section where they learn the Definition of Done.** DoD §2 names it. That is the enforced `definition-of-done` region, compiled from the vault's `pattern-change-lifecycle.md` into every harness. The PR template and the `verification.md` template name it too.

## Out of scope

- **Declaring the check required.** `spec-gate` is not required in dotfiles either (`forge/branch-protection.json`), and its red check is honoured by the human merge. `knowledge-gate` is honoured the same way until GUARD-017 (#1451) provides an apply for the protection file; its last task waits on that.
- **Other repos.** kubelab and web fell too. Their gate rides GUARD-016's delivery path (#1627), the one `spec-gate` takes, not a copy per repo.
- **A pre-push hook.** The body is edited on GitHub, and the `edited` event re-runs the gate. A pre-push check would block a code push over body text.
- **A minimum length for a reason.** Nothing shows a non-empty reason being gamed. If `none: n/a` shows up, the floor is a ratchet with data behind it, as ADR-037 §3 sets policy values.
- **A stricter Lesson rule for `fix:` PRs.** The approved design had one; it is subsumed, because every line already needs a path or a reason. A rule keyed on the title would be escaped by retitling.
- **The rest of the capture plan:**
  - the weekly crystallize job on hermes-nan and the capture metric;
  - retrieval: consulting `docs/lessons/` before writing a guard.

## Risks / open questions

- **Every open PR turns red** once the workflow is on main, because the merge ref includes it. Before the wiring PR merges, live peers are told, and the bodies of open PRs get their section. That is an edit, not a push.
- **The release PR footer.** The pinned release-please action (v5, `45996ed`) accepts `pull-request-footer`; its bundled config schema carries the key. The first release PR after the merge is the smoke test.
- **A PR that only deletes a lesson.** A deletion is not a path that counts (diff filter `AMR`), so its line takes `none: <reason>`.
- **The author judges their own work.** The reason is written by the agent whose work it describes. The gate makes the decision visible, not right; the review and the human merge judge it.

## Acceptance criteria

- [ ] **AC1.** `scripts/check-knowledge-gate.sh` exits:
  - 0 for a body whose three lines are counting paths or reasoned `none`s;
  - 1 when the section, a line or a reason is missing, or a path is not in the diff, is outside its kind's directory, or is `_index.md`;
  - 2 on a usage error, an unset body, or a diff it cannot compute.

  A `## Knowledge` inside a fenced code block does not count.
- [ ] **AC2.** A dependency-bot PR (an exact bot login plus the `dependencies` label) passes with an `[OK]` line, and a human-authored PR with the label is judged normally. The bot list equals `spec-gate`'s, and a test fails if they diverge.
- [ ] **AC3.** `.github/workflows/knowledge-gate.yml` runs the checker through `spec-gate-pr.sh --gate check-knowledge-gate.sh` on `opened`, `synchronize`, `reopened`, `labeled`, `unlabeled` and `edited`. It does not cancel a run in flight on a metadata-only event. `spec-gate-pr.sh` without `--gate` still runs `check-spec-gate.sh`.
- [ ] **AC4.** `release-please-config.json` carries a footer whose `## Knowledge` section passes the checker.
- [ ] **AC5.** `dotf spec archive`:
  - refuses a spec whose `verification.md` leaves a promotion line unanswered, answers `no` without a reason, or answers `yes` with a path that does not exist;
  - names the line in the refusal;
  - archives when every line is `yes: <existing path>` or `no: <reason>`.
- [ ] **AC6.** DoD §2, the PR template and the `verification.md` template describe the section and the answer grammar.
- [ ] **AC7.** The wiring PR passes its own gate with a lesson and an ADR, not `none`.

## References

- Bitácora board: #387 (the `issue:` field above).
- Evidence: vault `10_projects/dotfiles/research/2026-09-25-knowledge-capture-routing.md`.
- Mirrors: `scripts/check-spec-gate.sh`, `scripts/spec-gate-pr.sh` (BUG-066's live read), `.github/workflows/spec-gate.yml`.
- Related: ADR-037 §3 (policy values change with data), lesson-290 (a guard fails closed), GUARD-016 (#1627), GUARD-017 (#1451).
