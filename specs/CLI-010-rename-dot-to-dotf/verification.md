---
tags: [spec, verification, templates]
created: "2026-06-13"
---

# Verification - CLI-010-rename-dot-to-dotf

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof.

- [x] AC1 (binary renamed, builds, tests green) → `cli/cmd/dotf/` exists, `cli/cmd/dot/` gone (git rename); `go build -o /tmp/dotf ./cmd/dotf` → `dotf version dev`; `go test ./...` → `cmd/dotf`, `internal/cmd`, `internal/spec` all `ok`. (features.json f1 PASS)
- [x] AC2 (release pipeline names dotf) → `cli/.goreleaser.yaml`: `project_name: dotf`, `builds.id: dotf`, `main: ./cmd/dotf`, `binary: dotf`, archive `id: dotf`; `.github/workflows/cli.yml` smoke runs `./cmd/dotf`. (f2 PASS)
- [x] AC3 (bootstrap installs/checks dotf) → `versions.conf` `DOTF_VERSION=0.2.0` (no `DOT_VERSION`); `scripts/install-dotf.sh` fetches `dotf_<ver>_<os>_<arch>.tar.gz`, extracts `dotf`, installs `~/.local/bin/dotf`; `healthcheck.sh` checks `dotf` vs `DOTF_VERSION`; `setup-linux.sh` sources `install-dotf.sh` + calls `install_dotf`. (f3 PASS)
- [x] AC4 (live docs point to dotf) → `AGENTS.md` §245/249/250, `README.md:215`, `docs/architecture.md` (table + cli layout tree + rules) all say `dotf`; `tests/architecture-md.bats` greps `cmd/dotf` in lockstep. No live `dot`-as-CLI ref remains (grep guard). (f4 PASS)
- [x] AC5 (ADR-020 amended) → banner after title + `## Amendment` section documenting the Graphviz collision, the newcomer-yields decision, and the v0.2.0 release sequencing. (f5 PASS)
- [x] AC6 (smoke + lint + bats) → see Test status.

## Test status

- `cd cli && go test ./...` → all packages `ok` (`cmd/dotf`, `internal/cmd`, `internal/spec`; `internal/review` no test files).
- `shellcheck scripts/install-dotf.sh scripts/healthcheck.sh` → clean. `setup-linux.sh` → only the pre-existing `SC1091` info note on `./scripts/utils.sh` (not introduced here; the `install-dotf.sh` source line keeps its `# shellcheck source=/dev/null`).
- `bats tests/install-dotf.bats` → 5/5 (arch mapping, happy path, checksum mismatch, missing entry, idempotence). `tests/healthcheck.bats`, `tests/architecture-md.bats`, `tests/versions-conf.bats`, `tests/agents-md.bats` → all green (57 total, 0 failures).
- Manual smoke: `dotf version` → `dotf version dev`; `dotf --help` shows `dotf`; `dotf spec init SMOKE-001 --force-no-gate` in a temp git repo scaffolded proposal/tasks/verification; `dotf spec archive --help` OK.
- No regressions: the only remaining live `dot` references are in ADR-020's historical body (intentional — covered by the amendment banner), `docs/lessons.md:940` (CLI-003 lesson, provenance), `docs/adr/adr-012:51` (a shell-alias mention, not the CLI), `specs/CLI-009/*` + `CHANGELOG` (provenance), and feature-id slugs like `CLI-007-dot-spec-init`.

## Decisions made during implementation

- **Depth = full consistent rename, not minimal.** Renamed the script file (`install-dot.sh`→`install-dotf.sh`), its function (`install_dot`→`install_dotf`), the version contract (`DOT_VERSION`→`DOTF_VERSION`), `DOT_RELEASE_BASE`, the `_dot_*` locals, and the bats file. A half-renamed `install-dot.sh` that installs `dotf` is a code smell a reviewer would flag. Blast radius was bounded (4 live callers); historical `specs/CLI-009/*` left untouched.
- **ADR-020 annotated, not rewritten.** Added a dated amendment banner + section rather than editing the decision body. ADRs are point-in-time records; the standard convention is to annotate (the banner says "read every `dot` below as `dotf`"). Contrast: `AGENTS.md` and `docs/architecture.md` are current-state SSOTs and *were* edited to `dotf`.
- **`DOTF_VERSION=0.2.0`, not 0.1.0.** v0.1.0 carries `dot` artifacts; `dotf` artifacts first ship in v0.2.0. Pinning 0.2.0 bakes in the post-merge release step (merge → tag v0.2.0 → CI builds dotf → install works).
- **Scope boundary vs CLI-005.** References to the `init-spec`/`archive-spec` *shell twins* (in `SKILL.md`, `check-spec-gate.sh:193`, the architecture-map, AGENTS.md §389/§406) are CLI-005's repoint, NOT this PR — they name the shell scripts, not the `dot` binary. CLI-005 will repoint them to `dotf spec`.
- **Third collision noted, not acted on.** `docs/adr/adr-012:51` mentions a `dot` shell *alias*; no such alias exists in the current RC files (aspirational/historical). Left as-is.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons.md`? **yes** — "Name a new CLI against the host's existing `$PATH` (collision check) before the ADR locks it; the cheapest time to rename is pre-adoption." Capture during/after merge.
- [x] ADR-worthy decision? **covered** — recorded as an amendment to ADR-020 (no separate ADR; a naming correction belongs with the ADR that chose the name).
- [ ] New pattern for `00_meta/patterns/`? no — repo-specific; the lesson suffices unless it recurs.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-010-rename-dot-to-dotf/` -> `specs/archive/CLI-010-rename-dot-to-dotf/`
- [ ] Backlog issue #367 closed with PR link
- [ ] Promotions above executed (the lessons.md entry)
