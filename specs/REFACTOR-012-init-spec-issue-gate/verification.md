---
tags: [spec, verification]
created: "2026-06-09"
---

# Verification - REFACTOR-012-init-spec-issue-gate

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Open issue scaffolds + injects context -> bats "scaffolds with an open issue gate" + "injects issue context into proposal ## Why"
- [x] Missing/closed issue fails exit 3, no dir -> bats "gate fails (exit 3) without --issue / when the issue does not exist / when the issue is CLOSED"
- [x] Both bypass flags work without gh -> bats "--force-no-gate bypasses the gate without calling gh" (poisoned stub) + "legacy --force-no-vault still works"
- [x] No 11-tasks.md reference; suite green -> bats "ADR-018: no 11-tasks.md reference remains in init-spec.{sh,ps1}"

## Test status

- Test suite: `bats tests/init-spec.bats` -> 12/12 ok (gh stubbed on PATH, no network)
- Full suite: `bats tests/*.bats` -> only pre-existing failures (6 pwsh-skips + 3 shell-profile env failures, reproduced identically on main)
- shellcheck `scripts/init-spec.sh` -> clean; init-spec.ps1 ASCII-only verified (PSSA runs in CI)
- No regressions in existing test suite: yes

## Decisions made during implementation

- Gate failure is always exit 3 (same code the vault gate used) -- callers only need "gate failed", the message differentiates missing flag / unknown issue / closed issue / no gh.
- `--task` removed outright instead of deprecated: it only parameterized the vault lookup, which no longer exists.
- `--force-no-vault` kept as a warning-emitting alias rather than removed, since AGENTS.md and muscle memory still reference it.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? <yes / no>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no>
- [ ] New pattern candidate for `00_meta/patterns/`? <yes / no>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/REFACTOR-012-init-spec-issue-gate/` -> `specs/archive/...`
- [ ] Issue #304 closed by the PR
- [ ] Promotions above executed (if any)
