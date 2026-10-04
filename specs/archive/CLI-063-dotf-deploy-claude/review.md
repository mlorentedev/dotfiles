---
spec: "CLI-063-dotf-deploy-claude"
verdict: "PASS"
reviewed_sha: "e04ca960dbba2febc2606db4dc2a3461e6e12988"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-03"
---

## Adversarial review

**Scope**: CLI-063-dotf-deploy-claude — Claude's four deploy capabilities (snapshot guard, plugin
sync, MCP registration, `settings.json` merge) reached through `dotf deploy` as amended, without
deleting or repointing either setup twin.

**Sources**: `specs/CLI-063-dotf-deploy-claude/{proposal,tasks,verification,divergences,features}.md`
+ commits `614ae4ed` (#1992), `0f67099a` (#1994), `944993e2` (#1996), `e04ca960` (closing docs),
on top of `ai/deploy.json` entry `claude-settings`, `ai/claude/{plugins,settings}.json`,
`mcp-servers.json`, `cli/internal/claude/{guard,plugins,mcp}.go`,
`cli/internal/cmd/deploy{,_claude}.go`, the merge engine in `cli/internal/deploy/deploy.go`, and
`tests/claude-plugins.bats`.

**Base note.** The launcher's base `ecd20efe` is the worktree fork point; the stated range spans
246 commits, the overwhelming majority of them other specs' work already merged to `origin/main`
(HEAD is 1 ahead). The change under review was therefore reconstructed as the spec's own commits
(the four above), which is what `verification.md` f8's per-commit walk measures and what
`review-request.json` (`base_sha: ecd20efe`, `reviewed_sha: e04ca960`) pins. Contract digests for
`proposal.md`/`tasks.md`/`features.json` match the state at `reviewed_sha`; this review is not
stale by construction.

### Spec and task alignment

- All eight acceptance criteria (three of them amended in-session, 2026-10-03, with the amendment
  recorded in `proposal.md` itself, not silently) map to named tests and non-vacuous
  `features.json` verification commands. Each mapped command was re-run fresh here
  (`-count=1`, packages `deploy`, `cmd`, `claude`, `mem`): all PASS. `go build ./... && go vet ./...`
  clean. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in any spec file.
- The amendments are honest and load-bearing: increment 3's re-scope (`strategy: merge` already
  expresses the per-key policy after AI-042 granularised `deepMerge`) is verified in code —
  `deepMerge` recurses objects, `unionLists` unions arrays, scalars/template win — which is
  exactly AC3's three policies. Lesson 336 records the stale measurement. This is the right way
  to shrink a spec: criterion amended *in* the contract file, old rationale kept, dates given.
- AC8 ("no twin deleted, no caller repointed") holds for every commit of this spec: none touches
  `setup-*.{sh,ps1}` or `scripts/`; both twins still carry the plugin lists, the merge function and
  the guard (`merge_claude_settings` et al. greps non-zero in both), which is also why
  `tests/claude-plugins.bats` can still extract and compare them (3/3 green).
- AC6 (no second hooks writer) is real, not aspirational: `ai/claude/settings.json` has no `hooks`
  key, `deepMerge` only writes template keys so a box's `hooks` survive untouched, and
  `TestGuardNeverAddsAHooksKey` covers the byte-copy guard. Mutation-verified below.
- #1491's Windows miscount is ported as Linux-reference behaviour (count only on success, failed
  install named and fails the run) with the divergence recorded as `divergences.md` row 1 —
  matching the proposal's "record, never absorb" discipline.

### Independent verification performed (not re-reading the implementer's claims)

- Fresh runs, all green: the union of all `features.json` verification `-run` selectors over
  `./internal/deploy ./internal/cmd ./internal/claude ./internal/mem` (30+ named tests incl.
  `TestPerKeyMergePolicy/{env,enabledPlugins,permissions.allow}`, `TestRegister*` incl.
  `TestRegisterLeavesOtherUvxServersAlone`, `TestMcpAddArgvMatchesTheTwins`, both
  manifest-refusal tests, `TestDeployCmd_AFailedPluginInstallIsNamedAndFailsTheRun`).
- `bats tests/claude-plugins.bats` → 3/3 ok.
- **Mutations run and reverted** (tree verified clean afterwards, `git status` empty but for the
  launcher's `review-request.json`):
  1. `deepMerge` object recursion disabled in `deploy.go` → `TestPerKeyMergePolicy` **FAIL**
      (AC3's "fails under top-level replace" claim confirmed).
  2. `"hooks": {...}` key injected into `ai/claude/settings.json` →
      `TestClaudeSettingsTemplateHasNoHooks` **FAIL** (AC4-inverted / AC6 guard is live).
  3. `rep.Added` appended before the install-error check in `claude.Sync` (the #1491 behaviour) →
      `TestSyncCountsOnlySuccessfulInstalls` **FAIL** (AC7 guard is live).
  4. Extra id `"drift-me@claude-plugins-official"` added to the Linux twin's loop →
      `tests/claude-plugins.bats` test 2 **FAIL** (AC2 drift guard is live; the non-empty
      assertions mean a reformatted twin fails loudly instead of vacuously passing).
- Code read of the seam: `deployCommandAvailable` gating means a box without `claude` prints
  `skipped claude-plugins (claude not installed)` and touches nothing (covered by
  `deploy_claude_test.go:198-200`); the two Claude steps are joined via `errors.Join`, so one
  failing still runs the other, as claimed; `ExpandDst` turns an unresolvable `{VAR}` into a loud
  error, never a path into `/`, and `CLAUDE_CONFIG_DIR` predates this change in `env-contract.json`.
- Version/decoder safety (AC5): the `claude-settings` entry uses only fields older released
  binaries already read (`name/src/dst/strategy/requires`), `version` stays 3, and the freeze +
  strict-parse tests pass; `TestParseManifest_RefusesWhatItCannotFullyRead` was re-run.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | REAL | reporting | Dry-run line repeats the verb: `reportCounts` sets `state="would add"` **and** `verb="would add"`, printing `would add claude-mcp 2 would add, 0 already present` | read of `deploy_claude.go` + the string is pinned verbatim at `deploy_claude_test.go:226` — so it is deliberate, but the pin freezes the awkwardness rather than the information | `TestDeployCmd_BareDeployDryRun…` (pinned; a fix updates the pin) | code (cosmetic; disposition in `verification.md`, no gate) |
| Minor | REAL | scope / AC8 wording | The stated review base (`ecd20efe`) admits `e4bedb26` (#1991), which removes 17 lines from `setup-windows.ps1` — while AC8 reads "nothing under `setup-*.{sh,ps1}`" is touched. Verified: #1991 touches no file of this spec's spec folder and its subject is the Orca hook call (CLI-093 slot), so f8's per-commit walk correctly excludes it; the twins this spec ports are intact. The wording of AC8 over claims across a stale base, not a false claim about this spec's commits | `git show --stat e4bedb26`; `git show --name-only` for #1992/#1994/#1996 show zero `setup-*`/`scripts/` paths; bats twins green | `tests/claude-plugins.bats` (proves the Claude twins survive) | spec (as AC8's wording) — **cannot be edited post-verdict; disposition this line in `verification.md` instead, or leave as-is: the check's scope is right** |
| Minor | THEORETICAL | MCP migration | `staleHive = regexp(uvx\|hive-vault)` is matched against the full `claude mcp get hive` output; a legitimate migrated entry whose displayed paths contain `uvx`/`hive-vault` (e.g. a `~/.local/share/uv/tools/hive-vault/...` command path) would remove-and-re-add once, then converge to the SSOT argv — churn, not corruption; self-correcting after one deploy | code read of `mcp.go:73-101`; `TestRegisterMigratesAStaleHiveEntry` + `TestRegisterLeavesOtherUvxServersAlone` cover the name-scope and the completed-migration half | named above for the covered half; the path-text-match variant UNTESTED (accepted risk at Minor) | tests (only if the human wants the variant pinned; no gate) |
| Minor | SPECULATIVE | plugin presence | `Sync` requires the id to be a **whole** whitespace token of `plugin list` output; ANSI-colored output would break the token equality and cause a re-install attempt per deploy. `dotf` captures via pipe (non-tty), where CLIs strip color; the twins' substring grep would have matched either way | none — no reproduction, and the pipe path makes color unlikely | UNTESTED (surface only, do not gate) | — |
| Question | REAL | process | The launcher's `base_sha` resolved to the worktree fork point rather than the spec's first parent, making the literal diff scope 246 commits of other specs' merged work. Reviewers of this repo's future rounds could rubber-stamp "the diff" of nothing relevant. Not a defect of CLI-063's implementation | `git merge-base HEAD origin/main` = `66da55b5`; `git rev-list --count ecd20efe..HEAD` = 246 | — (process observation) | vault/pattern candidate for the launcher, *outside* this change — do not attach to this spec's files |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | A | All 8 criteria (3 amended in-contract, with dates) verified by named tests re-run fresh here; negative paths (failed install, failed add, absent claude, absent prerequisite, broken `plugin list`, stale hive, hooks) each covered; 4 independent mutations killed |
| Verification | B | Evidence is command-level and reproduced almost in full by this review; one claim (f9 "10/10 mutations killed") sampled rather than re-run exhaustively — nothing contradicted |
| Scope | B | Diff matches the amended proposal; the only ambiguity is base-relative (#1991 above), which the f8 per-commit walk handles correctly by construction |
| Reliability | A | Guard restores on truncation with in-place rewrite preserving mode/ACL; list-read failure is an error, never "empty"; steps joined so one failure does not hide the other; idempotent second run evidenced (`in sync`) and consistent with `deepMerge`'s change-detection (`jsonEqual`, sorted-key encode) |
| Maintainability | A | New packages small and single-purpose (guard 62, plugins 117, mcp 123 LOC); all new funcs <40 lines, `Truncated` extracted as a pure predicate; comments carry WHY and divergence numbers |
| Handoff-readiness | A | `divergences.md`'s 24 rows, lesson 336 landed, #1993 filed as a cutover prerequisite, next steps (cutover is explicitly out of scope and gated) all in-repo |

### Verdict

**PASS**

No Blocker, no Major (any reality class). The four Minor findings and one Question are tracked
above with dispositions; none meets the bar to hold the archive. Rubric has no C or D.
Contract digests in `review-request.json` match the state at `reviewed_sha`
`e04ca960dbba2febc2606db4dc2a3461e6e12988`.

`dotf spec archive` **is advisable** in the current state.

### Recommended next steps

On a PASS the contract set (`proposal.md`, `tasks.md`, `features.json`) is closed — none of these
are contract edits, each is a `verification.md` disposition or a follow-up ticket:

- **Implementer, in `verification.md`:** disposition the dry-run double-`would add` wording —
  apply (one-line fix in `reportCounts` plus updating the pin at `deploy_claude_test.go:226`) or
  decline with reason. Outside the contract set, so it can be done freely after archiving.
- **Implementer, in `verification.md`:** record the AC8-vs-base reading (#1991 touched
  `setup-windows.ps1` inside the literal stated range but is Orca/CLI-093 work excluded by f8's
  per-commit scope; this spec's commits touch no twin) so the next reader does not re-litigate it.
- **Cutover owner, before the twins' Claude regions are deleted (#1339 follow-on):** #1993 must
  land (hive's only installer today is the twins' MCP loop) — already filed; `tasks.md` increment 2
  records the ordering. The bats drift guard is the *last* thing to go, per the proposal.
- **Reviewer-pool launcher owner (outside this change):** the base this run resolved to was the
  worktree fork point, 246 commits deep; a "commit that first touched `specs/<id>/`" resolution
  would keep the stated scope honest.
- Optionally pin the hive `mcp get` path-text variant as a table case next time `mcp_test.go` is
  touched; currently accepted risk at Minor/THEORETICAL.
