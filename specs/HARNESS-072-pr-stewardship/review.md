---
spec: "HARNESS-072-pr-stewardship"
verdict: "PASS"
reviewed_sha: "1320efa1b384c2d90023cf9b3e08bcada76bef73"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-15"
---

## Adversarial review

**Scope**: HARNESS-072-pr-stewardship (PR #986, merged at 62d2e84)
**Sources**: `specs/HARNESS-072-pr-stewardship/{proposal,tasks,verification}.md`, `features.json`, `harness/enforced/pr-stewardship.md`, `harness/manifest.json`, `scripts/compile-harness.sh`, `tests/compile-harness.bats`, `harness/skills/pr-review-triage/SKILL.md`, `AGENTS.md`, `ai/claude/CLAUDE.md`, `~/.claude/CLAUDE.md`, `~/.gemini/GEMINI.md`, `~/.codex/AGENTS.md`, vault `00_meta/patterns/pattern-change-lifecycle.md` (HEAD at 2e351f1a)

### Spec and task alignment

All 8 acceptance criteria are addressed by the implementation. The proposal, tasks, and verification form a coherent chain. Two tasks claim a scope discipline that does not fully hold (see Findings).

### Verified evidence

Run against the current tree at `1320efa` (a descendant of the merged PR commit `62d2e84`; spec contract files unchanged since `62d2e84`):

| Check | Result | Command |
|---|---|---|
| 47/47 bats tests | ✅ | `bats tests/compile-harness.bats` |
| shellcheck | ✅ | `shellcheck scripts/compile-harness.sh` |
| `bash -n` | ✅ | `bash -n scripts/compile-harness.sh` |
| `--check` no drift | ✅ exit 0 | `./scripts/compile-harness.sh --check` |
| All 8 features pass | ✅ | `for i in 1..8; jq .verification features.json; bash -c "$cmd"` |
| Vault section matches record | ✅ byte-identical | `diff /tmp/vault-section.md harness/enforced/pr-stewardship.md` |
| AC2: 5 surfaces, 1 hit each | ✅ | `grep -c 'the disposition, not the waiting'` on AGENTS.md, ai/claude/CLAUDE.md, ~/.claude/CLAUDE.md, ~/.gemini/GEMINI.md, ~/.codex/AGENTS.md |
| AC2: caps hold | ✅ | GEMINI 6503/12000, codex 6503/32768 (`wc -c`) |
| Mutation: remove from inject → GAP | ✅ | `jq` removed pr-stewardship from ai/claude/CLAUDE.md inject, `--check` → `[GAP]` + exit 1; reverted clean |
| No [AGENT-DRAFT] tags | ✅ | g[r]ep -rn AGENT-DRAFT specs/HARNESS-072/ → none |
| Vault pattern present | ✅ | `grep -c "## PR Stewardship"` at vault HEAD (contains 2e351f1a) |
| pr-review-triage body gh-only | ✅ | uses `gh pr checks`, `gh pr view`, `gh api`, `gh run view` — no Claude-specific primitives |
| Reviewer-pool claim verified | ✅ | `harness/reviewer-pool.json` exists; `archive.go` calls `checkReviewGate`; spec-gate.yml enforces archive-on-merge |
| Working tree clean | ✅ | `git status --short` → empty |

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|-------------|
| Minor | REAL | .gitignore completeness | PR adds `specs/*/review-transcript.jsonl` but `review_launch.go`'s `StderrPath` also writes `.jsonl.stderr` | `git show 62d2e84:.gitignore` shows only the `.jsonl` line; `.stderr` file appeared untracked at session start. Fixed by a concurrent session in 1320efa (not yet on origin/main) | UNTESTED | code (already fixed as 1320efa; note that the fix is not yet on origin/main) |
| Minor | REAL | Scope claim | tasks.md closing checklist: "[x] No unrelated changes in the diff — the record sync is its own commit". The diff (6252eba..62d2e84) includes `harness/skills/dispatching-parallel-agents/SKILL.md` (16 lines, reconciliation block — substantive unrelated content), `harness/agents/curator/AGENT.md` (owner field), `harness/skills/systematic-debugging/defense-in-depth.md` + `root-cause-tracing.md` (frontmatter). These are unavoidable vault-drift syncs (the PR author kept six others out at b678103), but the claim is inaccurate. | `git diff 6252eba..62d2e84 --stat` shows these files changed; the dispatching-parallel-agents change is a real feature addition (mandatory reconciliation block) unrelated to PR stewardship | UNTESTED | spec artifacts (correct the tasks.md claim to acknowledge the drift sync) |
| Minor | THEORETICAL | Verification | features.json f2 checks only the 2 committed surfaces + doctrine.inject, not the 3 $HOME deployed surfaces that AC2 requires. The verification.md supplies session evidence for those, but the machine-readable check is weaker than the AC. | `jq '.[1].verification'` shows f2 checks AGENTS.md, ai/claude/CLAUDE.md, and doctrine.inject — not the agy/codex payloads. The grep on $HOME payloads confirms the sentence is present, but the f2 command would not fail if a deploy regressed | UNTESTED | tests (extend f2 to also check deployed surfaces, or document the proxy) |
| Minor | THEORETICAL | Tests | `check_coverage`'s doctrine branch is untested by fixtures. The 3 bats tests use a manifest without a `doctrine` section. The real tree passes (pr-sizing is in doctrine.inject, coverage OK), but a regression wouldn't be caught. | `sed -n '851,880p' scripts/compile-harness.sh` shows the doctrine surface handling; the 3 bats tests use `write_two_surface_manifest` which has no doctrine section | UNTESTED | tests (add a 4th bats test with a doctrine surface) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All acceptance criteria verified; negative paths covered (AC8's partial-injection case proven by 3 bats tests + real-tree mutation). Minor gap: middle-ground opt_out interpretations (e.g., a truthful but misleading opt_out reason) are not tested — but that's a human judgement, not a mechanical gap. |
| Verification | B | Evidence is thorough and mostly reproducible (commands + outputs). AC2's $HOME payload evidence is session-dependent but verifiable. Features.json f2 is a proxy for the full AC2. One lesson captured in docs/lessons.md. |
| Scope | B | Diff is focused on the spec. The vault-drift sync (dispatching-parallel-agents reconciliation block, curator owner field, sysdebug frontmatter) is unavoidable generated-file drift, but the tasks.md claim contradicts the diff. |
| Reliability | B | Error paths handled (GAP messages, DRIFT detection, exit codes). The coverage guard is idempotent. The "ten minutes" default is acknowledged as a guess — the mitigation (demotion to default mechanism) is sound. |
| Maintainability | A | `check_coverage` is a clean 30-line function with a WHY comment explaining the blind spot it exists for. 3 bats tests are clear with comments. Region text is terse. The lesson in `docs/lessons.md` is comprehensive. |
| Handoff-readiness | B | Spec updates included (tasks.md, verification.md, archive checklist). Lesson captured in docs/lessons.md. The "six stale records" finding is documented. Minor: the scope claim in tasks.md is inaccurate. |

### Verdict

**PASS**

- No **Blocker** or **Major** findings. All findings are **Minor**.
- **No C or D** in any rubric dimension. All grades are B or above.
- The coverage guard (`check_coverage`) is the spec's most important contribution and works correctly on the real tree (verified by mutation test).
- The region text is injected into all 5 surfaces, satisfies all acceptance criteria, and matches the vault source byte-for-byte.
- The pr-review-triage skill is correctly updated to cover the reviewer bot.
- The `dotf spec archive` gate is `advisable` in the current state — the spec meets the mechanical requirements for archiving.

### Recommended next steps (before archive)

1. **Correct the tasks.md scope claim** — the diff does contain unrelated generated-record syncs (dispatching-parallel-agents reconciliation block, curator owner field, sysdebug frontmatter). Acknowledge them as unavoidable vault-drift syncs rather than claiming "no unrelated changes".
2. **(Optional) Add a 4th bats test** — exercise `check_coverage`'s doctrine surface branch with a fixture manifest that includes a `doctrine` section.
3. **(Optional) Extend f2** — have the features.json verification command also confirm the $HOME doctrine payloads are grep-positive, or document that f2 is a proxy for the committed-world subset.
4. **Note the .gitignore .stderr gap** — the `.stderr` companion is not yet ignored on `origin/main` (the fix at 1320efa is on a fix branch, not merged). If the spec is archived before that fix lands, the next `dotf spec review` run will leave an untracked `.stderr` file.