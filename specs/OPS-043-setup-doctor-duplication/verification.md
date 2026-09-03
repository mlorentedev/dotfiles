---
tags: [spec, verification, templates]
created: "2026-09-02"
---

# Verification - OPS-043-setup-doctor-duplication

## Evidence

- [x] **AC1** (byte drift on the deploy-dir→`$HOME` leg FAILs; symlink FAILs) →
      commit `7482384`, `TestCheckHomeDeployDrift` in
      `cli/internal/doctor/checks_home_deploy_test.go`. Eight rows: content
      agreement, drift on `.zsh/functions.sh` (the file in no doctor list before
      this), drift on `tmux.conf` (the entry whose relative path differs between
      the two legs), the symlink case, and the missing/unprovisioned branches.
- [x] **AC2** (exempt set is existence-only; stale `sed -i` clause corrected) →
      commit `7482384` for the exemptions plus
      `TestHomeDeployExemptionsAreReasoned`, which fails on an exemption with no
      stated mechanism; commit `c61d607` removed the NOTE carrying the dead
      clause. `grep -n 'sed -i' setup-linux.sh` now returns nothing.
- [x] **AC3** (equal-or-stronger parity for all 17 items) → commit `69dc2da`,
      `TestSetupShellParity` (17 subtests, all pass) + `TestCheckDockerCompose`
      (4 rows).
- [x] **AC4** (six REFACTOR-002 pre-exports guarded) → commit `69dc2da`,
      `tests/guard-setup-preexports-present.bats`, 3 tests. Covers both setup
      scripts and asserts the exports still precede the `dotf doctor` handoff.
- [x] **AC5** (join guard on the map ↔ setup script) → commit `7482384`,
      `TestHomeDeployMapCoversSetup`. Fails in both directions: an uncovered
      `deploy_file` call, and a map entry setup no longer deploys.
- [x] **AC6** (shell calls gone, both layers green) → commit `c61d607`. See
      Test status below.

## Test status

- `cd cli && go build ./... && go vet ./... && GOOS=windows go vet ./...` → clean.
- `cd cli && go test ./...` → all packages ok, no FAIL lines.
- `golangci-lint run` with the pinned 2.12.2 (matches `versions.conf`, BUG-071
  checked before running) → **0 issues**.
- `~/.local/bin/bats tests/*.bats` → **1534 tests, exit 0**, run against the tree
  with the deletion already applied.
- `shellcheck --severity=error setup-linux.sh scripts/utils.sh` → clean (CI's
  severity; at default severity the repo is non-clean on `main` too, verified by
  stashing — this change adds nothing new).
- `bash -n` and `zsh -n` on `setup-linux.sh` → both clean.
- **Smoke test on msi:** `dotf doctor` built from this branch reports
  `[Deploy-dir↔$HOME drift] (11 checks, all ok)` on a converged box — including
  `.gitconfig`, which genuinely differs from its deploy-dir copy and passes via
  its exemption rather than by accident.
- **Mutation test** (the parity claim is not vacuous): flipping
  `.zsh/functions.sh` to `contentChecked: false` turns `TestSetupShellParity`
  red with "…would lose the assertion". Reverted; `git diff` clean afterwards.
- No regressions: yes.

## Decisions made during implementation

- **The ticket's premise was false in five places, and the fix order is the whole
  finding.** `check_deployed` and `checkDeployDrift` cover *different legs* of
  `repo → ~/.dotfiles → $HOME`; no doctor section byte-compared the second leg
  (all 40 enumerated); `check_dependencies` only ever `log_warning`d;
  `.zsh/functions.sh` was in no doctor list; and the six pre-exports are BUG-021's
  fix, not duplication. Doing what #1337 asked, in the order it asked, removes
  coverage rather than duplication.
- **`isManagedDeployPath` was not reusable.** The second leg renames
  (`ssh/config` → `.ssh/config`, `tmux.conf` → `.tmux.conf`), so the map is
  explicit — and explicit means it can drift, which is why AC5 guards the join
  itself. Nothing in the repo spanned a setup script and a doctor list before.
- **Compose is probed as a plugin, not a PATH entry.** `command -v
  docker-compose` — what the shell call did — reports missing on a current
  install where `docker compose` works. Measured on msi: v1 binary *and* plugin
  v2.39.1 both present, so either probe alone describes the box wrongly.
- **Absence of compose is SKIP, not FAIL**, because the repo provisions it in no
  installer, pin or contract binary. Same reasoning BUG-052 applied to terraform.
- **The exemption list was measured, not inherited.** Deriving it from
  `check_deployed`'s comments would have missed `.gitconfig`, the only one of the
  eleven files actually observed drifting, since that function never watched it.
- **`Test-FileDrift` was already dead.** Written as PowerShell parity with
  `check_deployed` for `healthcheck.ps1`; CLI-018 retired that script and left
  the helper with zero call sites. Deleted as part of the same pair.
- **Windows does not get the check.** Every map entry is POSIX-only, so the
  section SKIPs there and the leg stays unguarded — stated as an explicit
  non-goal and filed as **OPS-046 (#1447)** rather than silently implied.
- Setup scripts 3991 → 3988 lines; the shell layer as a whole (`*.sh` + `*.ps1`)
  is **net −66**. The setup-script figure moves little because the deleted calls
  were replaced by comments recording where the behaviour went — which is what
  would have prevented #1337 being filed against them.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? **yes** — a justification can outlive
      its mechanism exactly as a name outlives its contract: the `sed -i` clause
      was true when written, and OPS-040's purge (whose own method was "probe the
      target, never read the block's comment") removed the mechanism and left the
      comment. Lesson 256 violated by the change that established it.
- [ ] ADR-worthy decision? **no** — ADR-020 §5 already governs strangler-fig on
      contact; this is an application of it, not a new decision.
- [ ] New pattern candidate for `00_meta/patterns/`? **no** — the finding is
      about this repo's own comments and tickets; not yet observed in a second
      project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Independent adversarial review PASS recorded in `review.md` (reviewer must
      not be the implementer)
- [ ] Folder moved: `specs/OPS-043-setup-doctor-duplication/` -> `specs/archive/OPS-043-setup-doctor-duplication/`
- [ ] Bitácora board ticket #1337 closed with PR link (ADR-018)
- [ ] Lesson above written to `docs/lessons.md`
