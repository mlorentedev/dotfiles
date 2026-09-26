---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - SKILL-001-portfolio-consolidation

## Evidence

- [x] AC1 -> vault commit: eight skills under `90_archive/skills/` with `status: archived`, 31 left in `00_meta/skills/`
- [x] AC2 -> `TestEverySkillTheRouterNamesHasARecord` (new, failed first on the real triggers and dependency map), `TestEveryDeclaredSkillHasARecord`, no `requires:` naming a retired skill
- [x] AC3 -> `dotf harness resolve-skills` on the three records, and the `enforce: warn` count in each record's frontmatter
- [x] AC4 -> the folded sections in the `adversarial-review`, `spec` and `new-ticket` records
- [x] AC5 -> build, vet, `go test ./...`, lint, `--check`, full bats
- [x] AC6 -> after merge: `--deploy`, then no copy this pipeline rendered is left in any deploy target, and the Copilot catalog reads clean. The `skills-pipeline.bats` test "SKILL-001: deploy prunes a retired skill it rendered and keeps a foreign one of the same name" pins the difference. f6 runs `scripts/check-retired-skills.sh`, which reads every target from the manifest, and `tests/check-retired-skills.bats` seeds each target the manifest declares

## Test status

- `go build ./...`, `go vet ./...` (linux and windows), `go test -count=1 ./...`: pass.
- `golangci-lint run ./...` (0 issues), `shellcheck --severity=error scripts/compile-harness.sh`, `compile-harness.sh --check` (no harness drift), `check-bats-names.sh`, `check-doc-paths.sh`, `check-md-escapes.sh`: all pass.
- `bats tests/*.bats`, serial, re-run after the round-3 fixes: 1,607 run, 1,606 pass, 1 failure, the environmental oh-my-zsh snapshot (#1641), which is also red on plain main on this machine. The bats files this change touches, the six f5 runs plus `verify-setup.bats`, were re-run at the round-4 fix: 201 run, 0 failures (`verify-setup.bats` skips outside its integration container, so CI's container job is where its edited assertions run).
- Vault: `vault-validate.py` reports the same 65 issues on master and on this change; none is new.
- TDD: `TestEverySkillTheRouterNamesHasARecord` was written first and failed, naming `project-maturation`, `writing-plans`, `audit`, `verification-before-completion`, `executing-plans` (all in `DefaultSkillDependencies`) and `enrich-us` (trigger `task-and-ticket-tracking`). It passed after the remap.
- `TestRoleJoinDrift` (HARNESS-110) went red on the slimmed builder roster: 12 of 18 rules resolved, against a floor of 16. See the first decision below; after it, 16 of 18 resolve again. `TestResolveRoles` now expects `pr-review-triage` to resolve to reviewer and shipper.
- Round 1 follow-up. The new `skills-pipeline.bats` test passes. It fails when its seeded copy of ours loses the mark, because deploy then leaves the copy in place. The new f6 was run in a throwaway HOME:
  - a clean deploy: 0;
  - a foreign `audit` added: 0 (the old f6 returned 1);
  - our retired copies seeded in two targets: 1;
  - after a redeploy: 0, with ours pruned and the foreign one kept.
- Round 2 follow-up: `tests/check-retired-skills.bats` passes all 3 tests, and the check exits 0 against the real HOME. Each of these mutations of the script turns the test red:
  - dropping the doctrine files from its target list;
  - dropping the `command` render type, which also makes the script exit 2 on an unknown type;
  - dropping the `from:` marker of flat prompts.

  `shellcheck` is clean.
- Prompt hook, run against this branch's records (`DOTFILES_DIR=<worktree> dotf harness suggest --from-hook`): prompts about Go, firmware, an MCP server and asyncio still print `[persona] builder` and name the domain skill first (`golang-pro`, `debug-hardware`, `mcp-builder`, `async-python-patterns`). Control run with the new rosters and main's triggers: no output at all for the Go prompt.

## Deploy evidence (AC6)

Run on 2026-09-25 from `~/Projects/dotfiles` at main `3c2e393` (#1694 merged). Every live peer was told before and after.

- **Before.** The eight retired skills appeared 68 times: 48 rendered copies (8 skills in 6 targets) and 20 lines naming them in the Copilot catalog and the persona presence lines.
- **Mirror.** `dotf harness mirror`: 18 updated, 50 unchanged.
- **Deploy.** `compile-harness.sh --deploy`: exit 0, 48 `pruned stale` lines. `~/.gemini/GEMINI.md` is 7,653 characters and 7,653 bytes, down from 7,798.
- **Orphan records.** The mirror does not prune, so the eight retired records stayed in `~/.dotfiles/harness/skills` and doctor reported them as orphans. `dotf doctor --fix` applied exactly eight fix actions, one per orphan.
- **After.** f6 exits 0: no retired skill in any deploy target, the Copilot catalog or a presence line. `dotf harness resolve-skills` on the deployed builder, planner and reviewer records prints the slimmed rosters. Doctor's unset-severity warning reads 13 of 27 persona skills, down from 21 of 36.
- **What doctor still reports.** Re-run at the round-3 fixes: 172 passed, 13 failed, 7 warned, 8 skipped. None of the failures comes from this change. Six are zombie specs (#1626). Seven are deploy-dir drift that only setup refreshes: the four known files, `.bashrc` and `.zshrc` (whose comments this change edited), and `ssh/config` (from #1659, merged the same day).

## Post-merge deploy (round-3 finding 4)

Run on 2026-09-25 from `~/Projects/dotfiles` at main `280af58`, the 0.59.0 release. Live peers were told before and after.

- `dotf harness mirror`: 4 updated, 64 unchanged.
- `compile-harness.sh --deploy`: exit 0.
- All six rendered copies of `adversarial-review` (claude, pi, gemini skill and prompt, copilot, opencode) carry the round-2 wording, "since the last change it covers".
- `scripts/check-retired-skills.sh`, with the eight retired names, exits 0 against the real HOME.
- `dotf` 0.59.0 is installed, and the prompt hook's `skills:` line for a git-workflow prompt no longer names `verification-before-completion`.

## Size of what a persona is told to consume

Sum of the roster's `SKILL.md` files; tokens estimated at 4 characters each. Reference files a skill loads on demand are not counted.

| Roster | Before | After |
|---|---|---|
| builder | 9 skills, 53,311 B, about 13.3k tokens | 4 skills, 29,003 B, about 7.2k tokens (-46%) |
| planner | 6 skills, 61,859 B, about 15.4k tokens | 3 skills, 56,782 B, about 14.1k tokens (-8%) |
| reviewer | 4 skills, 36,955 B, about 9.2k tokens | 3 skills, 48,782 B, about 12.1k tokens (+32%: `pr-review-triage` joined, and `adversarial-review` absorbed two skills) |
| all active skills | 39, 304,054 B, about 75.6k tokens | 31, 282,514 B, about 70.2k tokens (-7%) |

## Independent review, round 1

Verdict **FAIL**, from nan/qwen3.8-flash on 2026-09-25. The review ran from 10:29 to 11:08, and its verdict rested on one REAL Major. Manu approved the dispositions below on 2026-09-25.

| # | Finding | Disposition |
|---|---|---|
| 1 | **Major, REAL.** f6 tested only that a retired skill's name was absent. A skill named `audit` that another tool installed turned it red twice during the review (10:47 and 10:50). The review showed by size that the copy was not ours: 1,892 B against a 1,551 B record, and renders are smaller than records. It then left, and f6 went green at 11:02 with nothing of ours changed. | **Applied.** f6 now counts a retired skill as present only when the copy carries this pipeline's mark (`generated_from: 00_meta/skills/<name>/SKILL.md`, or the `from:` comment in gemini prompts). AC6 and f6's behavior say so. A new bats test pins the distinction. |
| 2 | Minor, REAL. The folds dropped verification-before-completion's letter-and-spirit clause, its "Red Flags" list and its "Rationalization Prevention" table, and enrich-us's "item already done" case. | **Partly applied.** The clause is back in `adversarial-review`, and the done-item case in `new-ticket` (vault `cee00c1f`, records refreshed). The two lists are declined: they persuade rather than rule, the rule itself is the Iron Law gate and the claims table, and the reviewer's roster already grew 32%. |
| 3 | Minor, REAL. AC5 names the full bats suite, while f5 runs four files. | **Recorded.** The full suite's one failure is the oh-my-zsh snapshot test (#1641). The review reproduced it on plain main as well, so it is environmental and outside this change. |
| 4 | Minor, REAL. The README and rc files changed the example `/audit src/auth.py` to `/test src/auth.py`. | **Declined.** The line shows the syntax of a slash command, and `/test <file>` is a correct use of `test`. `adversarial-review` reviews a change, not a file. |
| 5 | Minor, SPECULATIVE. `TestResolveDependencies` spelled retired names in its synthetic fixture. | **Applied.** The fixture now uses synthetic ids. |
| 6 | Minor, SPECULATIVE. A research note in iris pointed at the retired `dispatching-parallel-agents`. | **Applied.** The note says the skill was retired and where it went (vault `cee00c1f`). The session log that also names it is history and stays. |
| 7 | Question. Four worktrees still carried the eight records, and a deploy from any of them reinstalls them. | **Coordinated and ticketed.** The live peers were told to rebase before deploying, and harness-hardening fast-forwarded. The general hazard is HARNESS-153 (#1712). |
| 8 | Question. Keep the data-level routing fix, or add a persona key? | **Data fix kept**, as the review recommended. The persona key is HARNESS-154 (#1713). |

One more residue, outside AC6's deploy targets: the prompt hook kept suggesting `verification-before-completion`. The installed `dotf` (0.58.0) compiles the dependency map in, and the map still carried the retired name. A binary built from main no longer names it, and the 0.59.0 release clears it. Recorded on HARNESS-147 (#1693).

## Independent review, round 2

Verdict **FAIL**, from nan/deepseek-v4-flash on 2026-09-25. The review ran from 12:11 to 12:42, and its verdict rested on one REAL Major. Manu approved the dispositions below on 2026-09-25.

| # | Finding | Disposition |
|---|---|---|
| F1 | **Major, REAL.** f6 hand-listed its targets. It read three of the six instruction files the harness writes, so a forced roster naming a retired skill in `~/.config/opencode/AGENTS.md` or `~/.pi/agent/AGENTS.md` left it at exit 0. | **Applied.** `scripts/check-retired-skills.sh` reads every target from `harness/manifest.json`: the six `skills.deploy` directories, and every file in `agents.presence`, `skills.catalog` and `doctrine.deploy` (opencode, pi and Codex included). f6 calls it. `tests/check-retired-skills.bats` walks the manifest too and seeds each target in turn, so a harness added there is tested without editing either file, and an unknown render type fails the test. |
| F2 | Minor. The full-suite count read 1,599 run; it was 1,600. | **Applied.** |
| F3 | Minor. The fold said "has not run in this session", which is looser than the original "in this message" and contradicted its own table ("A previous run" is not sufficient). | **Applied** (vault `8ed998cd`, record refreshed). It now reads "has not run since the last change it covers", and the table says "A run from before the last change". This is consistent with the Definition of Done's "in this session". |
| F4 | Minor. `pattern-architecture.md` still named `enrich-us` as a consumer. | **Applied** (vault `8ed998cd`). It now names the enrich step of `new-ticket`. |
| F5 | Minor. f4 checked only headings, so a folded section emptied to one line kept it green. | **Applied.** f4 also checks one load-bearing rule from each fold's body. |
| F6 | Minor. The lesson candidate was left undecided. | **Declined**, with the reason in the promotion candidates below. |
| F7 | Minor. AC5 named the full bats suite, which f5 does not run and which carries one environmental failure. | **Applied.** AC5 now names what f5 runs: Go, `--check`, the doctrine budget test and the bats files this change touches. The full-suite run stays as supporting evidence above. |

Not findings, but the review recorded two points:

- Only f3's warn counts guard against a flat roster entry disarming the gate. That is a disclosed risk, and doctor reports unset entries.
- The reviewer's roster grows by 32%, which is also disclosed.

## Independent review, round 3

Verdict **FAIL**, from nan/qwen3.8-flash on 2026-09-25. The review ran from 14:19 to 15:29, and its verdict rested on one REAL Major. The review found the retirement itself complete: no live router, record, pattern or deployed surface still names a retired skill. Manu approved the dispositions below on 2026-09-25. None of them touches the contract files.

| # | Finding | Disposition |
|---|---|---|
| 1 | **Major, REAL.** `check-retired-skills.sh` failed open. With a copy of ours present, it still exited 0 in three cases: when jq emits CRLF (the winget build on Windows), when `.skills.deploy` is empty, and when the manifest cannot be read. | **Applied.** The script requires jq and a readable manifest, strips CR from jq's output as `compile-harness.sh` does, and exits 2 when a target list is empty or the manifest is not JSON. One bats case per arm; all three were red before the fix. The CRLF case shadows jq on PATH, so it lives alone in `check-retired-skills.bats`, and every real-jq case moved to `check-retired-skills-real.bats`: the repo's `stub-real-pairing.bats` guard flagged the shadow until the suite had a real sibling. f5 names both files. |
| 2 | Minor, REAL. f2 reads only flow-style `requires:`, and no Go guard read the records' `requires:` at all. A block-form retired name passed everything. | **Applied.** `TestEverySkillTheRouterNamesHasARecord` also reads every record's `requires:` through the YAML parser. The review's own mutation (a block-form `- audit` in `pattern-loader`) now fails it. f2 is unchanged, since it is in the contract set. |
| 3 | Minor, THEORETICAL. Rendered agent definitions (`agents.deploy`, render `agent-md`) carry the roster and were not read. | **Applied.** The script also reads every `*.md` under each `agent-md` directory. A bats case seeds one. |
| 4 | Minor, REAL. The counts in this file and the deploy evidence no longer described the head. | **Applied** to the counts below. The deployed copies of `adversarial-review` predate the round-2 wording, so they get a redeploy from main after the archive merges, recorded then. |
| 5 | Minor, REAL. The FAIL verdicts left no artifact: `review.md` is overwritten each round, and the transcripts are gitignored. | **Applied and ticketed.** The three verdicts are kept beside the spec as `review-round-1.md` to `review-round-3.md`. Rounds 1 and 2 were copied aside before each relaunch. Keeping every round by design is HARNESS-157 (#1722). |
| 6 | Minor, THEORETICAL. `pr-review-triage` at `enforce: warn` on the reviewer warns even when there is no PR to triage. | **Kept, as AC3 specifies.** Flattening it would switch off the gate on the reviewer's third duty, and a warning is not a block, so a surplus warning costs a line of output. |
| 7 | Question. The routing guard counts how many rules resolve, not which ones. | **Accepted.** The schema-level exit is HARNESS-154 (#1713). |
| 8 | Minor, SPECULATIVE. The hook names the domain skill first only because it sorts first. | **Recorded, no change.** The entry skill is advisory. Since HARNESS-147 (#1714) it is chosen through the prerequisites, and the alphabetical fallback is documented in `entrySkill`. |

## Independent review, round 4

Verdict **PASS-WITH-GAPS**, from nan/glm5.3-flash on 2026-09-25. It is the first round launched under HARNESS-152's deadline (#1721): the reviewer was told to aim for 16:30 and would have been stopped at 16:45. It ran from 16:00 to 16:17. The review found that round 3's minimum set was met, and it re-proved each arm with its own mutations. It judged the archive advisable once finding 1 was dispositioned.

| # | Finding | Disposition |
|---|---|---|
| 1 | Minor, THEORETICAL. The check guarded two of its three target lists against being empty, but not the rendered agent definitions. The bats case for that surface skipped when it found none. | **Applied** after the review, outside the contract set, as the review recommended. The check exits 2 when the manifest renders no agent definition. That case now fails instead of skipping, and a new case pins the exit. |
| 2 | Minor, SPECULATIVE. The count of the touched bats files was stale. | **Applied.** It is now 201 run and 0 failures. |
| 3 | Question. Three residuals carried from round 3: the post-merge redeploy, `warn` on a reviewer with no PR, and the count-only routing guard. | **Tracked**, as recorded under round 3. The redeploy happens after this merges. |

## Decisions made during implementation

- **Four domain triggers gained a discipline skill from the builder's roster.** A prompt's persona is derived from `trigger.skills ∩ persona.skills` (HARNESS-110). With the domain skills off the builder's roster, the Go, Python, hardware and MCP rules resolved to no persona, and a rule with no persona prints nothing, not even its domain skill. That contradicts "available when their domain is detected". `golang-engineering`, `python-cli` and `mcp-tool-design` now also name `test`, and `hardware-and-embedded-debug` names `systematic-debugging`. Each is on the builder's roster, and none changes which skill the hook names first. The principled alternative is a persona key for associated skills that the join reads but the gate does not. It is a schema change, offered on the PR rather than made here.
- **Severities kept as they were.** A flat roster entry is `EnforceUnset`, which the gate neither enforces nor warns on, so rewriting the reviewer's and planner's warn-level entries flat would have switched them off.
- **Retired trigger skills remapped, not dropped.** `security-and-quality-audit` now names `adversarial-review`, which holds the checklist. `plan-authoring-and-execution` now names `spec`, which holds planning and execution. `task-and-ticket-tracking` keeps `new-ticket` and `prd-to-issues`.
- **`DefaultSkillDependencies` loses only the retired names.** `handoff` and `pr-review-triage` now depend on `adversarial-review` (the successor of what they depended on). The map was already out of step with the skills' `requires:` frontmatter before this change; that is HARNESS-147 (#1693), not widened here.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? **No.** The candidate: a forced roster is also a routing table (HARNESS-110), so slimming it silently unroutes prompts unless a guard counts resolving rules. It is already recorded where it acts: in this proposal's Risks, in HARNESS-154's problem statement (#1713), and in `TestRoleJoinDrift`, whose floor of 16 enforces it. A separate lesson would be a fourth copy (round-2 finding F6, declined).
- [ ] ADR-worthy decision for the repo's `docs/adr/`? no
- [ ] New pattern candidate for `00_meta/patterns/`? no

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/SKILL-001-portfolio-consolidation/` -> `specs/archive/SKILL-001-portfolio-consolidation/`
- [x] Bitácora board ticket moved to Done / closed with the closing PR (ADR-018): the archive PR carries `Closes #1692`
- [x] Promotions above executed: none, since all three were decided no
