---
tags: [spec, verification, templates]
created: "2026-06-13"
---

# Verification - CLI-005-retire-spec-shell-twins

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] **AC1 — the 4 shells + `tests/init-spec.bats` no longer exist**: `git rm` removed `scripts/{init,archive}-spec.{sh,ps1}` and `tests/init-spec.bats` (status `D` in the diff).
- [x] **AC2 — no live `init-spec`/`archive-spec` reference outside historical artifacts**: `grep -rnE 'init-spec|archive-spec'` returns only `CHANGELOG.md` + `specs/**` (provenance) + `docs/adr/adr-018,adr-020` + `docs/lessons.md` (point-in-time records). Zero live source/skill/script references remain.
- [x] **AC3 — AGENTS.md / check-spec-gate.sh / spec SKILL.md direct users to `dotf spec`**: AGENTS.md §389 (reframed from "shell fallback" to single cross-platform entry) + §406 (`dotf spec init <id> --issue <N>`); `check-spec-gate.sh:193` hint; `SKILL.md` 117/261.
- [ ] **AC4 — `tests/agents-md.bats` asserts the `dotf spec` guidance and passes**: assertion updated to `grep -qF 'dotf spec init'`; both pinned literals (`dotf spec init`, `11-tasks.md`) confirmed present in AGENTS.md. **Run `bats tests/agents-md.bats` to capture the green.**
- [ ] **AC5 — full bats suite green (minus deleted bats) + shellcheck clean + `dotf spec init/archive` work e2e**: **pending user run** (commands below).

## Test status

Run before merge (kept for the user per hands-off-testing preference):

```
bats tests/*.bats                              # init-spec.bats deleted; expect all green
~/.local/bin/shellcheck scripts/check-spec-gate.sh scripts/check-md-escapes.sh
(cd cli && go test ./...)                       # unchanged Go path, smoke
(cd cli && go run ./cmd/dotf spec init TEST-001-smoke --force-no-gate && go run ./cmd/dotf spec archive TEST-001-smoke --abandoned)
```

- Guard grep: clean (only CHANGELOG / specs / ADRs / lessons remain — all historical).
- Manual smoke: `dotf spec init --help` / `spec archive` flags confirmed against the Go source (`--issue`, `--force-no-gate`, `--pr`, `--abandoned`, `--force-with-drafts`).
- No regressions expected: only deletions + reference repoints; the Go `dotf spec` path is unchanged from CLI-007/008.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **The acceptance guard-grep is the completeness oracle, not the spec's hand-enumerated file list.** The proposal listed 5 repoint targets; the `grep -rE 'init-spec|archive-spec'` acceptance check surfaced **two more live references the list missed**: `scripts/check-md-escapes.sh` (a comment naming `init-spec.sh` as a line-based parser) and `harness/skills/adversarial-review/SKILL.md` (L13/L165 naming `archive-spec.sh` as a command to run). Both were repointed — required to meet AC2, not scope creep.
- **`harness/skills/*/SKILL.md` are committed RENDERS; the edit-SSOT is the vault** (`00_meta/skills/<name>/SKILL.md`, per `compile-harness.sh` §12/§22). The render edits here are correct for CI `--check`/`--deploy`, but the vault sources for `spec` and `adversarial-review` still said `init-spec`/`archive-spec` and would have **reverted this repoint on the next `compile-harness.sh --refresh`**. **DONE:** both vault `SKILL.md` sources repointed to `dotf spec`, now on vault `origin/master` — obsidian-git auto-persists vault edits, so no manual git is needed there (a manual worktree+push was an over-correction this session; just edit in place).
- **`11-tasks.md` residual drift is OUT OF SCOPE.** ADR-018 retired the vault `11-tasks.md` task store; AGENTS.md:405 still names it as historical context, `agents-md.bats:33` pins that word, and the embedded `verification.md` template (this file, line "Backlog entry in vault 11-tasks.md") still instructs ticking it. That is a distinct drift from the spec-twin retirement — filed as its own ticket **REFACTOR-013 #370**, not folded in (CLI-005 acceptance: "no unrelated changes in the diff").

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "When a migration's spec lists repoint targets by hand, the acceptance guard-grep is the real completeness oracle; run it before claiming done — it caught 2 surfaces the list missed."
- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "Editing a committed render (`harness/skills/`) without its vault SSOT is a half-migration that `--refresh` silently reverts." (sibling of the GEMINI→AGY incomplete-migration lesson.)
- [ ] ADR-worthy decision? no — executes ADR-020 §5, no new architecture.
- [ ] New pattern candidate for `00_meta/patterns/`? no.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-005-retire-spec-shell-twins/` -> `specs/archive/CLI-005-retire-spec-shell-twins/` (via `dotf spec archive CLI-005-retire-spec-shell-twins --pr <url>`)
- [x] Vault skill sources synced (`spec`, `adversarial-review`) → vault `origin/master`; 11-tasks ticket filed → REFACTOR-013 #370
- [ ] Promotions above executed (2 lessons)
