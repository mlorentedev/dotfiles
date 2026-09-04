---
id: "CLI-072-dotf-hooks-install"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-02"
issue: "mlorentedev/dotfiles#1460"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-072-dotf-hooks-install

## Why

<!-- from issue #1460: CLI-072: dotf hooks install — retire the install-git-hooks twin pair, and the test twins with it -->

The `dotf` migration drive measures itself in setup-script LOC and **twin-pair
count**. Six pairs at drive start, six today: LOC has moved 4079 → 3992 while the
metric the drive declared has not moved at all. `install-git-hooks` is the
cleanest pair that can actually retire — `install-dotf` is bootstrap (ADR-020 C7),
`utils` shrinks only by attrition.

**The code twins agree; the test twins do not, and that is the real finding.**
13 bats against 9 Pester for the same behaviours. Four are verified on Linux only:
`Test-GuardDispatcher` (exists in the `.ps1` with zero coverage),
`Test-SameHooksPath` (likewise), a missing source dispatcher directory, and the
`#695` self-mirror guard — the one that stops the clean mirror's `rm -rf` from
deleting the dispatcher source and copying nothing back. **The guards exist on
both sides; only their verification is missing**, so this is a coverage gap and
not a live defect. It still means a destructive `rm -rf` runs on Windows behind a
guard nobody has exercised.

Test drift produces no symptom until the untested guard is the one that fails.
Porting both suites to `go test` removes the asymmetry by construction rather
than by discipline.

## What

1. **`dotf hooks install`** deploys the GUARD dispatcher and wires
   `core.hooksPath`, replacing both twins. It copies from a **source directory**
   (`--source`, defaulting to the checkout's `git-hooks/`) to
   `$DOTFILES_DIR/git-hooks` — the same shape both twins have today.
2. **One Go test suite** covers the union of the 13 bats and 9 Pester cases, so
   the four Linux-only behaviours gain Windows coverage from the same source.
3. `scripts/install-git-hooks.{sh,ps1}` and
   `tests/install-git-hooks.{bats,Tests.ps1}` are **deleted**, both setups
   repointed — and on Windows the step **moves** (see R6).
4. `dotf doctor`'s GUARD remedy names `dotf hooks install` instead of "run
   dotfiles setup" — a precise command where there was a re-run-everything
   instruction.

### The C7 objection, resolved

`install-git-hooks.ps1` declares itself exempt under ADR-020 C7. C7 reads
*"**Irreducible shell bootstrap.** The step that provisions **the tooling itself**
stays shell (chicken-and-egg)."* Installing git hooks does not provision the
tooling: `dotf` already exists by then, on both platforms once R6 is fixed. C7
covers `install-dotf` and does not reach this.

The `.ps1`'s supporting clause — *the deployed release carries no source tree* —
is answered by not needing one: `dotf hooks install` runs from a checkout, which
is where setup runs. A released binary on a machine with no checkout has no hooks
to install **and nothing to install them from**; that is today's behaviour, not a
regression this introduces.

### Not embedding, and why the obvious idea was dropped

`//go:embed git-hooks` was the first design and it was measured out rather than
argued out. `git-hooks/` sits at the repo root, **outside the Go module** (the
module is `cli/`), so embedding requires relocating the tree — which means editing
`.gitattributes`'s `eol=lf` rule, the CI path filter, README, `architecture.md`,
three audit ADRs, and **three unrelated bats suites** that read hook files from
the root (`board-pickup`, `precommit-fallback`, `gitattributes-eol`). That is a
relocation change wearing a port's clothes, for a directory the README documents
as a repo landmark.

Keeping the source directory also keeps three behaviours *real* that embedding
would have quietly retired: a missing source directory, a source with no
`pre-commit`, and — the important one — the `#695` self-mirror case, which is
exactly `--source X --dotfiles-dir Y` where `X` resolves to `Y/git-hooks`. Under
embedding that input could not occur; here it is the first guard to mutation-test.

## Out of scope

- **`dotf doctor`'s guard-hook probe.** #1332 (CLI-061) wants doctor to probe the
  dispatcher by consequence instead of by file presence. Adjacent, different
  concern, not folded in. This PR only updates the FAIL **remedy** string and the
  stale `install-git-hooks.ps1` comment in `checks_tools.go`.
- **The other five twin pairs.** One pair per PR; `dotfiles-sync` and `obs-cli`
  are unmeasured, `install-dotf` and `utils` are excluded above.
- **Behaviour changes.** Any bug found while porting is reproduced faithfully and
  ticketed separately — a port that improves while translating cannot be
  characterisation-tested.

## Risks / open questions

- **R1 — `git config --global` is a side effect on the developer's real machine,
  and the seam's shape is the decision.** A test that rewires the machine it runs
  on is not a test. Two candidate shapes, and the choice is made here rather than
  discovered: **a `gitRunner` seam** (`func(ctx, args...) ([]byte, error)`), the
  idiom CLI-071 just landed — a fake records `config --global core.hooksPath X`
  and returns canned `--get` output, with no process spawn and no global state —
  **plus exactly one integration test** driving real `git` against a throwaway
  `GIT_CONFIG_GLOBAL`, which is what proves the fake speaks the same dialect as
  the real binary. The fake alone would be a closed loop; the integration test
  alone would be 22 slow tests that can corrupt a developer's config.
- **R2 — file modes.** The hook entrypoints must be written `0755` explicitly
  rather than inheriting the copy's mode. On Windows the bit is inert, but
  git-for-windows runs the dispatchers through `sh`, so only the Linux leg can
  regress silently.
- **R3 — path equivalence must not become a string compare.** The `.sh` uses
  `[ "$src" -ef "$dest" ]` (same inode) and the `.ps1` resolves both paths.
  `os.SameFile` is the faithful cross-platform equivalent; comparing cleaned
  strings would pass the trailing-slash test and still miss a symlinked mirror.
  It needs two `os.Stat` calls that both succeed, so on a **first** install —
  where the destination does not exist yet — the self-mirror check must
  short-circuit to "not the same" rather than surface the stat error. The bats
  test for `#695` assumes both paths exist and would not catch that.
- **R4 — deleting the twins while `setup-*.sh` still sources them breaks setup.**
  Ordering is the finding from OPS-043 and applies unchanged: port and repoint
  first, delete last, in that order within the PR.
- **R5 — the four Linux-only behaviours have no Windows oracle.** Their expected
  Windows behaviour is derived from the `.ps1` source, not observed on a Windows
  box. Where the two twins genuinely differ (drive-root refusal is Windows-only),
  the Go test keeps both cases rather than picking one.
- **R6 — on Windows the hooks step runs BEFORE `dotf` exists, and must move.**
  Measured: `setup-windows.ps1` installs hooks at `:365-374` and runs
  `Install-Dotf` at `:633`. Repointing in place would invoke a binary 268 lines
  before it is installed, and the surrounding `Write-Warn` would swallow it — a
  silent skip of the memory-sink guard on every fresh Windows box. So the step
  **moves** after `Install-Dotf`. `setup-linux.sh` is already correct
  (`install_dotf` at `:281`, hooks at `:301`).
  This is the same class as the rest of this ticket: the `.ps1` was not written
  expecting `dotf` to exist by then, because when it was written it did not need
  it to.

## Acceptance criteria

- [x] **AC1 — `dotf hooks install` deploys and wires**, mirroring the source
      dispatcher tree to `$DOTFILES_DIR/git-hooks` and setting `core.hooksPath`,
      with the entrypoints executable.
- [x] **AC2 — every behaviour from both suites is covered by one Go suite**: the
      13 bats cases and the 9 Pester cases, including the four verified on Linux
      only today and the Windows-only drive-root refusal. Each safety guard is
      mutation-tested, starting with `#695`.
- [x] **AC3 — no test touches the real global git config.** The 22 cases drive a
      `gitRunner` fake; the single integration test drives real `git` against a
      throwaway `GIT_CONFIG_GLOBAL`. A test that would rewire the developer's
      machine fails instead.
- [x] **AC4 — deployed hooks are CR-free** even from a CRLF-tainted source tree
      (BUG-068), the same property both twins guarantee today.
- [x] **AC5 — both twins and both test files are deleted**, both setups repoint to
      `dotf hooks install`, and no stale referent to the deleted scripts remains
      anywhere — including `checks_guard.go`'s user-visible FAIL remedy and
      `checks_tools.go`'s comment. Grep covers **prose**, not just callers
      (lesson 259).
- [x] **AC5b — on Windows the hooks step runs after `Install-Dotf`** (R6), proven
      by a guard asserting the ordering in `setup-windows.ps1` rather than by
      having read it once. Nothing in the repo spans those two call sites today,
      which is why the inversion survived.
- [x] **AC6 — doctor and the installer agree on the target path** by construction,
      asserted by a test spanning both rather than by two constants that happen
      to match.
- [x] **AC7 — the metric moves: twin pairs 6 → 5**, with the setup LOC delta
      recorded in `verification.md`.
- [x] **AC8 — the rule this PR obeys becomes a mechanism.** ADR-020 §5 has always
      required a port to delete its bats/Pester tests in the same PR, and nothing
      checked it — OPS-043 and CLI-072 both complied because the agent
      remembered. `scripts/check-twin-test-retirement.sh` fails a PR that retires
      a `scripts/X.{sh,ps1}` pair and leaves `tests/X.{bats,Tests.ps1}` alive.
      Added here rather than ticketed, because this PR is the rule's first
      deliberate instance and a guard filed for later is the debt it replaces.

## References

- Bitácora board: `mlorentedev/dotfiles#1460`
- `docs/adr/adr-020-tooling-cli-go-convergence.md` §5 (port on contact) and C7
- Prior art: `cli/internal/spec` (embedded templates), `cli/internal/doctor` (the `System` seam)
- #1332 (CLI-061) — the adjacent doctor probe, deliberately not folded in
- Repo lesson 259 (a justification outlives its mechanism), lesson 260 (the path with no seam)
