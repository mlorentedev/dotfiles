---
tags: [spec, verification, templates]
created: "2026-08-05"
---

# Verification - HARNESS-051-copilot-native-skills

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 (Native discovery) -> commit `8f1cc08` / test `HARNESS-051: copilot gets native /spec and /handoff skills` (`tests/skills-pipeline.bats`)
- [x] Criterion 2 (Complete and filtered render) -> commit `8f1cc08` / test `HARNESS-051: copilot target filtering and auxiliary files are preserved` (`tests/skills-pipeline.bats`)
- [x] Criterion 3 (Safe convergence) -> commit `8f1cc08` / test `HARNESS-051: copilot deploy prunes only generated stale skills` (`tests/skills-pipeline.bats`)
- [x] Criterion 4 (Product recognition) -> commit `8f1cc08` / `tests/copilot-native-skills-smoke.ps1`, run manually against a real installed Copilot CLI (1.0.78) on the Windows machine in the session that produced this checkpoint; `copilot skill list` listed `handoff` after deploying to an isolated `$env:COPILOT_HOME`. Not re-run in this pass (no `pwsh`/`copilot` binary available in this Linux environment) — the code path is unchanged since that run.

## Test status

- Test suite: `bats tests/skills-pipeline.bats tests/compile-harness.bats tests/docs-drift.bats -> 56/56 passing` (re-run fresh on `origin/main` merged into this branch, Linux, `jq` 1.8.1 / `zsh` 5.9 both present — the earlier "42 pre-existing failures, unattributable" note from the checkpoint session was an artifact of `jq`/`zsh` being absent in that WSL environment, not a real regression; this run is clean and attributable)
- `scripts/compile-harness.sh --check` -> `[check] OK: no harness drift`
- Manual smoke test: AC4 only, see Evidence above — not exercised in this pass, still valid from the checkpoint session (unchanged code path)
- No regressions in existing test suite: yes

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- Closed out the spec's "Closing" checklist in a follow-up Linux session rather than re-running the PowerShell/Copilot-CLI smoke test, since that environment lacks both `pwsh` and an installed `copilot` binary and the code it exercises has not changed since the prior Windows run recorded it passing.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons.md`? no
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no
- [ ] New pattern candidate for `00_meta/patterns/`? no — single declarative manifest entry, not a recurring pattern yet

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-051-copilot-native-skills/` -> `specs/archive/HARNESS-051-copilot-native-skills/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
