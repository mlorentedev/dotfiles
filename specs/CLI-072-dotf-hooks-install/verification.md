---
tags: [spec, verification, templates]
created: "2026-09-02"
---

# Verification - CLI-072-dotf-hooks-install

## Evidence

- [x] **AC1 — deploys and wires** → `TestInstallHandlesAFirstInstall`,
      `TestInstallMakesEntrypointsExecutable`, `TestInstallWiresHooksPath`, plus
      `TestHooksInstallIsReachableFromRoot` for the path a user actually takes.
- [x] **AC2 — the union of both suites** → the 13 bats and 9 Pester cases are
      covered by `TestInstallRefusesUnsafeDestinations`,
      `TestDeployRefusesADestinationOutsideGitHooks`,
      `TestInstallDoesNotDestroyItsOwnSource`, `TestInstallHandlesAFirstInstall`,
      `TestInstallRefusesABadSource`, `TestInstallCleanMirrorsRatherThanMerging`,
      `TestInstallNormalisesCRLF`, `TestInstallMakesEntrypointsExecutable` and
      the five subtests of `TestInstallWiresHooksPath`.
- [x] **AC3 — no test touches the real global git config** → every case drives
      `fakeGit`; `TestInstallAgainstRealGit` is the single exception and runs
      against a throwaway `GIT_CONFIG_GLOBAL` with `GIT_CONFIG_NOSYSTEM=1`.
- [x] **AC4 — CR-free deploys** → `TestInstallNormalisesCRLF`.
- [x] **AC5 — twins and test twins deleted, both setups repointed** →
      `scripts/install-git-hooks.{sh,ps1}` and
      `tests/install-git-hooks.{bats,Tests.ps1}` are gone; the prose sweep below.
- [x] **AC5b — Windows ordering** → `tests/guard-setup-hooks-order.bats`.
- [x] **AC6 — doctor and the installer agree** →
      `TestDoctorAgreesWithTheInstallerOnTheDispatcherPath`.
- [x] **AC7 — the metric moved** → twin pairs **6 → 5**.

### What the four Linux-only behaviours became

The reason this ticket existed. Each now runs on both CI legs from one source:

| Verified on Linux only before | Now |
|---|---|
| `is_guard_dispatcher` — an equivalent dispatcher elsewhere is ACTIVE, and a `pre-commit` without the guard warns | `TestInstallWiresHooksPath` subtests 4 and 5 |
| `Test-SameHooksPath` — a trailing-slash variant counts as wired | `TestInstallWiresHooksPath` subtest 3 |
| the `#695` self-mirror guard | `TestInstallDoesNotDestroyItsOwnSource`, plus the first-install case bats could not reach |
| a missing source dispatcher directory | `TestInstallRefusesABadSource` |

The Pester-only drive-root refusal is kept too, guarded on `GOOS`.

## Test status

```
cd cli
go build ./...            → clean      go test ./... -count=1 → all ok
go vet ./...              → clean      go test -race          → clean
GOOS=windows go vet ./... → clean      golangci-lint run      → 0 issues (v2.12.2 pin)
gofmt -l internal/hooks internal/cmd internal/doctor → clean

shellcheck setup-linux.sh → clean (2 pre-existing SC1091 infos on sourced files)
bash -n setup-linux.sh    → OK        zsh -n setup-linux.sh → OK
bats tests/*.bats         → see below
setup-windows.ps1         → ASCII-only; LF in the index (.gitattributes eol=crlf on checkout)
```

**A false green caught rather than reported.** The first bats run was
`bats tests/*.bats | tail -15`, and the harness reported "exit code 0" — which is
`tail`'s exit status, not the suite's, and the captured file held only the last 15
lines. Re-run without the pipe, capturing `$?` directly. The same shape as the
`go test -run <no-match>` trap: a command that cannot report failure is not
verification.

### Mutation testing

Every guard mutated; each caught by a named test.

| Mutation | Detected by |
|---|---|
| `#695`: drop the self-mirror short-circuit | `TestInstallDoesNotDestroyItsOwnSource` |
| drop the filesystem/drive-root refusal | `TestInstallRefusesUnsafeDestinations` |
| drop the `$HOME` refusal | `TestInstallRefusesUnsafeDestinations` |
| drop the empty-dir refusal | `TestInstallRefusesUnsafeDestinations` |
| drop the `*/git-hooks` shape check | `TestDeployRefusesADestinationOutsideGitHooks` |
| drop the missing-`pre-commit` refusal | `TestInstallRefusesABadSource` |
| skip CR stripping | `TestInstallNormalisesCRLF` |
| merge instead of clean-mirror | `TestInstallCleanMirrorsRatherThanMerging` |
| make `samePath` a byte comparison | `TestInstallWiresHooksPath` |
| treat any `hooksPath` as a dispatcher | `TestInstallWiresHooksPath` |
| clobber an unrelated `hooksPath` | `TestInstallWiresHooksPath` |
| swallow a failed `hooksPath` write | `TestInstallReportsAWiringFailure` |
| unregister the command from `root.go` | `TestHooksInstallIsReachableFromRoot` |
| installer writes to a path doctor does not check | `TestDoctorAgreesWithTheInstallerOnTheDispatcherPath` |
| invert the setup ordering comparison | `guard-setup-hooks-order.bats` (both legs) |

**Two escaped the first battery, and both were defects in my tests rather than in
the code.** Both had the same shape: asserting *that* something was refused
rather than *which* diagnosis came back.

- Removing the empty-dir guard still refused — `filepath.Clean("")` is `"."` and
  the root branch catches it — while telling a user with an unset `DOTFILES_DIR`
  that `"."` is a filesystem root.
- A byte comparison in `samePath` fell through to the equivalent-dispatcher
  branch, which **also** declines to write. Identical observable outcome; only
  the message differs.

Both tests now read the message. Where two branches produce the same effect and
differ only in what the user is told, the message *is* the behaviour.

## Decisions made during implementation

**`//go:embed` was the first design and was measured out.** `git-hooks/` sits at
the repo root, outside the Go module, so embedding meant relocating a directory
the README documents as a landmark — and with it `.gitattributes`'s `eol=lf` rule,
the CI path filter, `architecture.md`, three audit ADRs, and three unrelated bats
suites that read hook files from the root. A relocation change wearing a port's
clothes. Keeping a `--source` directory also keeps three guards *reachable* that
embedding would have silently retired, `#695` among them.

**The Windows step moved rather than being repointed.** `setup-windows.ps1`
installed hooks ~270 lines before `Install-Dotf`. Harmless while it sourced a
PowerShell twin; a silent failure the moment it invoked `dotf`, since the
surrounding `Write-Warn` would have turned "binary not found" into a skipped
memory-sink guard on every fresh box. Same class as everything else in this
ticket: the block was not written expecting `dotf` to exist there, because when it
was written it did not need it to.

**Two guards came out stronger than their originals.** `checkDotfilesDir` refuses
a root with `filepath.Dir(p) == p`, true of `/` and `C:\` alike — where the shell
needed a POSIX check plus a separate PowerShell one, which is exactly why the
drive-root case existed only in Pester. `sameDir` uses `os.SameFile` and
short-circuits when the destination does not exist, the first-install case the
bats test for `#695` could not reach because it assumes both paths exist.

**A branch nothing could distinguish was deleted.** `samePath` had a
cleaned-string fast path; no test could tell whether it was present, because
wiring always runs after the deploy and `os.SameFile` already answers. Removed
rather than kept with a comment.

**`dotf` is resolved by path, not by name, in both setups.** Reusing the idiom
`setup-linux.sh` already documents for #1202: `install_dotf` places it in
`~/.local/bin`, which the rc files add to `PATH` but the running process may not
carry — the integration container hits exactly that.

**The setup LOC number went up, and that is stated rather than netted away.**
3992 → 4025 (+33): the Windows block grew to record why it moved, the Linux block
to resolve `dotf` by path. Net repo surface is −317 (`scripts/` 7921 → 7571). The
drive tracks two metrics and they moved in opposite directions here.

## Prose sweep (lesson 259)

Grepped for `install-git-hooks` / `install_git_hooks` / `Install-GitHooks` across
the repo, **comments included**, not just call sites:

- **Fixed:** `tests/setup-linux.bats:492-494` asserted setup *sources* the deleted
  script — a live test, not prose, and the only real breakage found.
- **Fixed:** `checks_guard.go`'s FAIL remedy and `checks_tools.go`'s comment.
- **Kept deliberately:** past-tense references in `hooks.go`, `cmd/hooks.go` and
  the two setup blocks. Recording where behaviour went is what lesson 259 asks
  for; a reference is stale only when it claims the thing still exists. The
  ordering guard strips comment lines before checking for invocations, for
  exactly this reason.
- **Left alone:** `docs/adr/audit-007` and `audit-010`. Frozen audit snapshots.
  Worth noting that `audit-010:165` describes the `#695` bug *and* calls doctor's
  remedy text circular — both fixed here, in the audit's own words, three months
  after it was written.

## Promotion candidates

- [ ] Lesson for `docs/lessons/`? **Not on its own.** The findings here are
      instances of lessons 259 and 260, already written. A third instance of "the
      message is the behaviour" would be worth its own lesson; this is the first.
- [x] ADR-worthy? **Yes, as an amendment, not a new ADR** — the corollary this
      PR is the first deliberate instance of: *when a shell function ports to
      `dotf`, its bats and Pester tests port to `go test` in the same PR and are
      deleted with the twin.* Belongs in ADR-020 §5, which already governs
      port-on-contact.
- [ ] New pattern for `00_meta/patterns/`? **No** — not yet seen in a second
      project.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/CLI-072-dotf-hooks-install/`
- [ ] Bitácora ticket #1460 closed with the PR link (ADR-018)
- [ ] Independent adversarial review recorded in `review.md` with a passing verdict
- [ ] ADR-020 §5 amendment landed

## AC8 — the rule became a mechanism

**The amendment this PR was going to propose already existed.** The plan was to
add a corollary to ADR-020 §5: *when a shell function ports to `dotf`, its bats
and Pester tests port to `go test` in the same PR and are deleted with the twin.*
Reading §5 before writing it showed the sentence is already there, verbatim:

> Port a script into `dot` only when it is next touched; **in the same PR, delete
> the `.sh` + `.ps1` pair and their bats/Pester tests. Never leave the three
> coexisting** (no triple-maintenance).

Plus **C2 — One test suite** and decision 4, *"`go test` (table-driven) replaces
bats + Pester for every migrated script"*. Writing the amendment would have
duplicated a decision into two places — the drift shape this repo keeps fighting.

**What was actually missing is enforcement.** OPS-043 and CLI-072 both obeyed §5
because the agent remembered, which is not a mechanism. So
`scripts/check-twin-test-retirement.sh` fails a PR that retires a
`scripts/X.{sh,ps1}` pair and leaves `tests/X.{bats,Tests.ps1}` alive. Nine
fixture-driven bats cases, run in both shells; shellcheck clean.

**Diff-based, and that was measured rather than assumed.** A state-based version
("a test whose subject script is missing") was tried first and rejected: most of
`tests/*.bats` covers invariants, docs and CI config rather than a same-stem
script, so it produced ~40 false positives on the current tree. §5 says *in the
same PR*, so the PR's diff is the right unit.

**The guard shipped with the bug it exists to prevent, and its own test caught
it.** The deletion list was read as `$(deleted_paths)` inside the loop's heredoc.
That runs in a subshell, so `exit 2` on an unreadable list — or a failing
`git diff` on an unreachable merge-base — killed the subshell and left the outer
script iterating an **empty** list, which exits 0. The header claimed it failed
closed; it did not. Test 9 ("a missing deletion-list file is an error, never an
empty list") was written before the fix and failed exactly as intended.

Verified against this PR's real diff, both directions: it passes on the branch as
it stands, and restoring the deleted `tests/install-git-hooks.bats` makes it fail
and name the file. Reported once rather than twice — a retired pair deletes two
scripts that resolve to the same surviving test.

**Also still open and worth naming:** #672 (CLI-031) is the *planned* other half —
golden characterization tests captured against a twin **before** porting and kept
as the executable contract. It is open, unstarted, and its pilot (#668) shipped
without it. Not folded in here; this guard enforces the deletion, not the
parity.
