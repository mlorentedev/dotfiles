---
spec: "TOOL-013-pr-agent-reviewer"
verdict: "FAIL"
reviewed_sha: "a6db7222af428cb4d1b80572c7dba91ed835614f"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-09"
---

## Adversarial review

**Scope**: TOOL-013-pr-agent-reviewer — PR-Agent on NaN inference as a second PR reviewer.
**Sources**: `specs/TOOL-013-pr-agent-reviewer/{proposal,tasks,verification,features}.md`;
`.pr_agent.toml`; `.github/workflows/pr-agent.yml`; `scripts/pr-agent-push-gate.sh`;
`scripts/pr-agent-model-preflight.sh`; `harness/pr-agent-upstream-contract.json`;
`harness/reviewer-pool.json`; `docs/runbooks/guide-pr-agent-reviewer.md`;
`tests/pr-agent-{config,push-gate,publish-guard,model-preflight,queue-skip}.bats`;
upstream `The-PR-Agent/pr-agent@8e5a9295973b24af4b70cafd0b660a230811ef9e` (the pin) read live
from the GitHub API: `file_filter.py`, `ignore.toml`, `configuration.toml`, `config_loader.py`,
`pr_reviewer.py`, `inline_comment_dedup.py`.

### Spec and task alignment

- **Diff scope.** The stated base `2f7bfd5b` is 616 commits and 2,309 files (~175k insertions)
  behind `HEAD` — the three-dot diff is dominated by other specs' work already merged to `main`.
  The 45 commits that touch this spec's owned files (`23c57169 feat(ci): add PR-Agent on NaN
  inference … #1032` → `a6db7222`) are the change; everything else in the range is context.
  I read the spec-owned files at `HEAD` and their whole history in range. See finding Q1.
- **All 9 `features.json` verifications were re-run fresh, not read:** each exits 0 (f1–f9), which
  matches `verification.md`'s claim. 116 bats cases across the five pr-agent suites pass;
  `shellcheck -S warning` on both scripts exits 0; `actionlint` on the workflow exits 0;
  `zsh -n` on both scripts exits 0.
- **tasks.md**: every implementation box is `[x]` except "Adversarial review passes before archive",
  which is this run. Two follow-on boxes are `[x]` and one of them is not true of the shipped
  config — see F1. `proposal.md`'s nine acceptance-criterion boxes are all `[ ]`.
- **No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` markers** in the authored spec files. The
  `review-transcript.jsonl` in this folder does carry them (it echoes this skill), and it is
  correctly excluded from the archive tag scan by `ReviewStateFiles` in
  `cli/internal/spec/archive.go:160` — verified, not assumed, so the archive gate is not
  unpassable by construction.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| **Major** | **REAL** | AC1 — inline delivery | The automatic path posts **no inline comments**. AC1 requires "inline review comments generated through NaN"; what ships is one summary comment. `github_action_config.auto_improve: "false"` (`.github/workflows/pr-agent.yml:~530`, decision #1107) removes the only inline pass that was configured, and `.pr_agent.toml` never sets `[pr_reviewer] inline_key_issues`, whose upstream default at the pinned commit is **false** — so `/review` itself cannot publish inline either. | Fetched `pr_agent/settings/configuration.toml` at `8e5a929…`: line 178 `inline_key_issues=false`; `pr_agent/tools/pr_reviewer.py:1174` gates inline publication behind that key; no `[pr_reviewer] inline_key_issues` in `.pr_agent.toml`. The repo's own record agrees: `verification.md` AC1 — "The findings arrive as one review comment that links each file and line, **not** as inline diff comments"; `tasks.md` follow-on "[x] Inline suggestions actually enabled … Measured across #1042, #1047 and #1051: 0 inline comments" is false again for the unasked path since #1107. | The guard `pr-agent: inline suggestions are actually enabled, not merely claimed` (`tests/pr-agent-config.bats:394`) asserts `[pr_code_suggestions].dual_publishing_score_threshold`, a section `auto_improve=false` keeps off the automatic path — so it passes while AC1 is unmet. The AC1 inline path is **UNTESTED**. | code (`.pr_agent.toml`: `inline_key_issues = true` — upstream's GithubProvider implements both `get_persistent_comment_bodies` and `get_recent_inline_comment_bodies`, so `can_verify_inline_comment_publication` is true on the Action path and findings publish inline **from the same single `/review` call**, which preserves #1107's demand-halving rationale) **+ tests** (a guard that the active config publishes inline on the unasked path); spec only if the config route is declined. |
| Minor | REAL | Spec/code claim drift | `.pr_agent.toml`'s `[pr_reviewer]` comment still says "Inline comments on the diff, which is the half CodeRabbit's free tier withholds on private repos **and the reason this work exists**", and `[pr_code_suggestions]` says `dual_publishing_score_threshold = 7` "is the line that makes the `[pr_reviewer]` claim true". Neither holds on the automatic path. This is the exact defect class the same file catalogues for `ignore_pr_source_branches` ("read like a decision and asserted nothing"). | Read of `.pr_agent.toml` + the F1 mechanism above. | `pr-agent: inline suggestions are actually enabled, not merely claimed` (same mis-scoped guard as F1) | code (comments) + tests, alongside F1 |
| Minor | THEORETICAL | AC3 — ignore-list replacement | Declaring `[ignore] glob = [...]` **replaces** upstream's default list, which at the pin is `glob = ['vendor/**']`. `sensitive/**` still applies, but `vendor/**` is dropped, so a PR touching a `vendor/` tree would now reach the model. No negation, no widening of the load-bearing entry — impact is narrow because this repo has no `vendor/`. | Upstream `pr_agent/settings/ignore.toml` at `8e5a929…` (`glob = ['vendor/**']`); `pr_agent/config_loader.py` comment: "`merge_enabled: False` … a section-level `set()` replaces the whole section, so always pass a full one". | `pr-agent: never sends sensitive/ to an inference endpoint` covers `sensitive/**` only (mutation M1 removing it turned it **and** `an escrow-only PR is recognised…` red — both bite); no guard names `vendor/**` → that half is UNTESTED | code (`.pr_agent.toml`: carry `vendor/**` alongside the repo's entries, or state the replacement in the comment) |
| Minor | REAL | Handoff / traceability | `features.json` records `state: "pending"` for all nine entries and `proposal.md` leaves all nine AC boxes `[ ]`, while `verification.md` asserts the criteria are met. Pass-state gating means only the harness may write `passing`, so this is expected — but the archive run must execute the harness first, and the AC boxes stay unreadable as a status record. | `grep` of `features.json` / `proposal.md` at `HEAD`; the nine verifications re-run here exit 0. | the nine `features.json` `verification` commands (all exit 0, run fresh in this session) | spec (`proposal.md` AC ticks) + harness pass-state; no code |
| Question / assumption | — | Review base resolution | The launcher-stated base produced a 2,309-file "whole change" diff for a spec whose owned surface is 8 files. For a spec whose work has partly landed on `main`, the stated base should be the spec's own first commit, or the launcher should say which subset is the change. Not a defect in this change; it belongs to whoever owns review-base selection (`specs/archive/HARNESS-112-review-base-and-reality`). | `git rev-list --count 2f7bfd5b..HEAD` → 616; `git diff --stat` → 2,309 files / 175,239 insertions; `git log --oneline <base>..HEAD -- <owned files>` → 45 commits. | n/a | vault / other spec (not TOOL-013) |

### What I tried to break, and what held (evidence, not praise)

Each of these was attacked with a mutation and reverted; the tree is clean apart from the
launcher's untracked `review-request.json`.

- **AC3** — M1: deleted `"sensitive/**"` from `[ignore] glob` → `never sends sensitive/ to an
  inference endpoint` **and** `an escrow-only PR is recognised as having nothing PR-Agent may read`
  both went red. The runtime emulation in the `Skip when PR-Agent would receive an empty diff` step
  matches upstream `file_filter.py` exactly (`fnmatch.translate(glob)`, plus the glob without a
  leading `**/`, drop on `re.match`), read from the pinned commit — so the filter is real, not a
  copy of a claim.
- **AC4** — M2: added `EXTRA__KEY: ${{ secrets.SOMETHING_ELSE }}` → `the workflow takes exactly one
  secret` went red. The guard counts distinct names and admits `secrets.GITHUB_TOKEN`, which is not
  an inference credential, so it matches AC4's wording; the publication guard deliberately reads
  `github.token` to keep that count honest. `NAN_API_KEY is declared as a CI consumer of this repo`
  passes against the registry.
- **AC2** — M3: `fallback_models = ["openai/qwen3.6"]` → both `the fallback is not a model the
  reviewer pool excludes by name` and `the chain the preflight probes equals the toml's` went red.
  Separately, I checked the one mechanism that could hand the review to a cheap model behind the
  config's back: upstream `[model_routing]` ("send a small pull request to a cheaper primary model")
  is `enable = false`, `rules = []` at the pin, so `model_weak = "openai/qwen3.6"` cannot become the
  review's primary. `qwen3.8-flash` and `glm5.3-flash` are pool members; `qwen3.6` is not, and the
  config does not use it for review.
- **AC5** — `gh api repos/The-PR-Agent/pr-agent/git/ref/tags/v0.47.0 --jq .object.sha` returns
  `8e5a9295973b24af4b70cafd0b660a230811ef9e`, the exact pin. Independent of the annotation.
- **AC8** — M4: changed one blob in `harness/pr-agent-upstream-contract.json` → the network leg
  `pinned upstream review contracts match their approved source identities` went red with
  `pr_agent/agent/pr_agent.py changed since the approved upstream contract`. With the file untouched
  it exits 0. The runtime filter derives `PR_AGENT_REF` from `github.workflow_sha` (the executing
  workflow), and fails closed when the sed yields anything other than one 40-hex sha.
- **AC6/AC9** — exactly one `uses: The-PR-Agent/pr-agent@…` step; `fail_on_tool_errors: "true"`;
  no `continue-on-error` on the Action; the guard treats a failed/cancelled Action as an error,
  distinguishes an API outage from an absent marker (`gh_api` retry + `api_errors` channel, #2069),
  binds the marker to the run's start stamp and to `github-actions[bot]` authorship, and skips
  itself after a credential or preflight failure so the wrong diagnosis is not stacked on a red job.
  The push gate's `continue-on-error: true` fails **toward** reviewing, not toward skipping.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Eight of nine criteria verified with negative tests and live runs; AC1's "inline" half is not met on the automatic path — a substantial, demonstrated gap on the criterion that names the tool's reason to exist. |
| Verification       | B | Evidence is reproducible: the nine `features.json` commands re-run here exit 0, live run ids and triage links are cited, and `verification.md` discloses the AC1 shortfall instead of papering over it. Not A: the AC1 box rests on a claim the config contradicts. |
| Scope              | B | The owned files match the proposal; later growth (`TOOL-023` push gate, `AI-045` preflight, `#1417` type filter) is declared in tasks.md sections with its own specs, and the rollout/retirement non-goals are respected. |
| Reliability        | B | Fail-closed on ambiguous failure, fail-open toward reviewing, retries with an outage/absent distinction, bounded Action timeout inside a longer job, queue-skip for a PR that died while waiting. |
| Maintainability    | B | shellcheck and actionlint clean, `zsh -n` clean, small functions in both scripts, guards bound to the executing pin rather than copies; deducted for two config comments and one follow-on task box that assert behavior the shipped config does not perform. |
| Handoff-readiness  | B | Runbook, ADR-040/042 lineage and promotion answers are written; `features.json` states and AC ticks are still `pending`/`[ ]` for the harness to set. |

### Verdict

**FAIL** — one **REAL** Major (F1: AC1 inline delivery, UNTESTED on its actual path), which forces
FAIL under the severity × reality rule regardless of the rubric. The rubric independently caps this
at PASS WITH GAPS (Correctness C, no D); the more severe path decides.

`dotf spec archive TOOL-013-pr-agent-reviewer` is **not advisable** in this state — it would refuse
this verdict anyway, and the contract files would need the changes below first.

### Recommended next steps

A FAIL makes the contract set fair game again; these are ordered so the smallest set flips the
verdict.

**Blocking (must land before a passing review):**

1. **Code** — `.pr_agent.toml` `[pr_reviewer]`: add `inline_key_issues = true`. Upstream v0.47.0
   publishes each review finding as an inline comment from the *same* `/review` response when the
   provider can verify inline publication, and `GithubProvider` at the pin can — so this satisfies
   AC1 without restoring the second inference call #1107 removed. If the owner declines that route,
   the alternative is re-enabling `auto_improve` for a bounded class of PRs, which costs the NaN
   slot that decision bought.
2. **Tests** — add a named bats guard that the *automatic* path is configured to publish inline
   findings (assert `pr_reviewer.inline_key_issues` is true, or `auto_improve` true, whichever route
   is taken). Retire or re-scope `pr-agent: inline suggestions are actually enabled, not merely
   claimed` so it cannot pass while AC1 is unmet; it currently guards a section the automatic path
   never runs. Then re-run the mutation: flip the new key off and confirm that guard goes red.
3. **Spec / code comments** — correct `.pr_agent.toml`'s `[pr_reviewer]` "the reason this work
   exists" block, the `[pr_code_suggestions]` "makes the claim true" line, and the `tasks.md`
   follow-on box marked `[x]` for inline suggestions, to state what the shipped config actually does.
4. **Re-review** — `dotf spec review TOOL-013-pr-agent-reviewer`. F1's fix touches `.pr_agent.toml`
   and `tests/` (outside the contract set), so if step 3 is limited to the tasks.md box and comments,
   one round should be enough.

**Tracked, non-blocking (disposition in `verification.md` — apply, ticket, or decline with a reason):**

- Carry `vendor/**` in `[ignore] glob`, or record that declaring the key replaces upstream's list.
- Run the harness so `features.json` states leave `pending`, and tick the nine AC boxes in
  `proposal.md` against the evidence.
- File the review-base question (Q1) against `HARNESS-112`'s owner: a stated base that resolves to
  2,309 unrelated files makes the "whole change" instruction unreadable for a spec this narrow.
