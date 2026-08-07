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
- [x] Criterion 5 (Detect-and-act) -> follow-up commit / test `BUG-771: copilot native skills are not deployed when the copilot binary is absent` (`tests/skills-pipeline.bats`), plus a direct filesystem check (see Test status below).

## Test status

- First full-CI run for this branch (the checkpoint session's WSL environment lacked `jq`/`zsh`, so it never got real signal): `integration` failed. Root cause: the native skill deploy added here writes to `~/.copilot/skills/` unconditionally, but `tests/verify-setup.bats`'s pre-existing "copilot config NOT deployed when gh-copilot extension is absent" (BUG-001/PR#40) asserts `~/.copilot` must not exist when Copilot isn't installed — the container has no `copilot` binary, so the assertion fired.
- Fix: `harness/manifest.json`'s copilot `skills.deploy[]` entry gained `"requires_command": "copilot"`; both `scripts/compile-harness.sh` (`deploy_skills`) and `setup-windows.ps1` (`Deploy-SkillRecord`) skip a target when its `requires_command` isn't on PATH. `opencode`/`agy`/`pi` have no such field (this repo auto-installs them; Copilot is explicitly "no auto-install" per BUG-003), so their deploy is unchanged.
- Verified directly: `env HOME=$FAKEHOME scripts/compile-harness.sh --deploy` with no `copilot` on PATH creates no `.copilot` directory at all (confirmed by `ls`, not just test assertions) — so `verify-setup.bats:279` needed no changes at all.
- `bats tests/skills-pipeline.bats tests/compile-harness.bats tests/docs-drift.bats` -> **57/57 passing** (added test 14, "BUG-771: copilot native skills are not deployed when the copilot binary is absent"; the two existing HARNESS-051 tests now stub a fake `copilot` on PATH via a new `stub_copilot` helper, since they test the deploy itself, not the gate).
- `scripts/compile-harness.sh --check` -> `[check] OK: no harness drift`
- `shellcheck -x scripts/compile-harness.sh` -> clean. `bash -n` -> clean.
- `setup-windows.ps1`'s `Deploy-SkillRecord` change mirrors the bash fix structurally (same manifest field, same skip-and-continue) but is **not executable-verified** here (no `pwsh`). Written defensively against `Set-StrictMode -Version Latest` (this file's own `ContainsKey` comment on the winget install block documents that direct property access on a JSON-deserialized object without a key throws under strict mode) using `.PSObject.Properties.Match(...)` rather than direct `$d.requires_command` access. `test-windows` CI is the real gate for this half.
- Manual smoke test: AC4 only, see Evidence above — not exercised in this pass, still valid from the checkpoint session (unchanged code path).
- No regressions in existing test suite: yes, beyond the two tests updated to stub `copilot` (which now test what they always meant to: the deploy logic, with presence assumed).

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Option A over Option B** (gate vs. retire the old invariant): considered making Copilot's skill deploy unconditional like opencode/agy/pi, and rewriting the old `verify-setup.bats` test instead. Rejected once `setup-linux.sh` was checked directly: opencode/agy/pi are auto-installed by this repo's own setup (unconditional deploy is safe because they're guaranteed present after setup runs); Copilot has a written, deliberate "no auto-install" policy (BUG-003). Gating on presence is the coherent choice for a tool this repo treats as optional, not a historical relic to remove.
- **Manifest-declared `requires_command`, not a hardcoded `if agent == "copilot"` check**: keeps `deploy_skills`/`Deploy-SkillRecord` agent-agnostic (the design principle this spec's own proposal already states: "one declarative target reaches Linux and Windows without changing either engine"). Any future tertiary tool can opt into the same gate via manifest data alone.
- **`deploy_prune` left ungated deliberately**: pruning only removes files this pipeline's own provenance marker (`generated: true`) is found on, and only if `~/.copilot/skills/` already exists — on a box without Copilot that directory never gets created in the first place, so the question is moot. Gating prune too would be speculative scope beyond what the failing test required.
- Closed out the spec's "Closing" checklist in a follow-up Linux session rather than re-running the PowerShell/Copilot-CLI smoke test for AC4, since that environment lacks both `pwsh` and an installed `copilot` binary and the code it exercises has not changed since the prior Windows run recorded it passing.

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
