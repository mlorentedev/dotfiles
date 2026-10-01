---
tags: [spec, verification, templates]
created: "2026-08-25"
---

# Verification - AI-030-pi-packages-manifest

## Evidence

| AC | Proof |
|---|---|
| AC1 manifest is well-formed, unique, justified | `tests/pi-packages.bats` — "valid JSON", "no source is declared twice", "every entry says what it is for" |
| AC2 every source pinned | `tests/pi-packages.bats` — "every declared source is pinned to a version", **plus** "the pin guard rejects an unpinned source" |
| AC3 array absent from the seed, never written by setup | `tests/pi-packages.bats` — "the array is NOT declared in the seed settings.json", "setup-linux never writes the packages array itself" |
| AC4 first run installs all | `TestNewPlan`, `TestApplyConvergesAndASecondRunCallsNothing`, `TestPiPackagesApplyRemovesThenInstalls` (was `verify-reconcile.sh`, see below) |
| AC5 second run changes nothing | `TestApplyConvergesAndASecondRunCallsNothing`: the second run makes no call to pi |
| AC6 object-form entries recognised | `TestLiveSourcesReadsBothEntryForms`, `TestIdentityIgnoresTheVersion` |
| AC7 pi absent warns, never aborts | `TestPiPackagesApplyWithoutPiWarnsAndExitsZero`, `TestPiPackagesApplyWithoutNpmWarnsAndExitsZero` |
| AC8 unreadable manifest is loud | `TestLoadManifestRefusesWhatItCannotRead`, `TestPiPackagesCheckRefusesAnUnreadableManifest` |
| AC9 Linux uses `$PI_BIN` | `tests/pi-packages.bats` — "setup-linux hands the command $PI_BIN, not the shell function" |
| AC10 Windows parity, no new non-ASCII | `tests/pi-packages.bats` — "both twins reconcile through dotf pi packages apply"; `setup-windows.ps1` now has 0 non-ASCII lines |
| AC11 a declared package actually **loads** | `dotf doctor` on the real machine — `[FAIL] extension "subagent" shadows the installed package of the same name`, then `[FIX ] quarantined ~/.pi/agent/extensions/subagent/index.ts`; effect confirmed by `pi -p` going from **exit 1** (`Failed to load extension … Tool "subagent" conflicts`) to **exit 0** answering `OK`, and a third `dotf doctor` run then reporting `[pi extensions] (1 checks, all ok)`. Unit-covered by `cli/internal/doctor/checks_pi_extensions_test.go` (6 cases: FAIL on collision, WARN without, the scoping rule that leaves an external writer's files alone, SKIP with no extensions dir, quarantine landing outside the auto-discovered tree, and no-clobber on a second `--fix`) |

All on commit `2c20332` plus the spec commit that follows it.

### Re-pointed after HARNESS-139 (2026-10-01)

HARNESS-139 (#1754, #1755) moved the reconcile out of both setup scripts into
`dotf pi packages apply`. `verify-reconcile.sh` drove the shell block it
deleted, so it is retired here, and AC4-AC10 point at the tests that now hold
them. The table above lists them; `features.json` carries one command per
criterion. Each Go command names its tests with `-run '^(...)$'`, and `go test
-v` confirmed each named test ran: a `-run` or `bats -f` that matches nothing
exits 0, which is how three of these commands had been passing without running
anything since #1754. AC10's ASCII count is now 0, since the file has none.

## Test status

Current, at the archive head (after the HARNESS-139 re-point):

```
$ bats tests/pi-packages.bats
1..14   all ok
```

The `features.json` commands f1-f10 each exit 0 when run by hand at the same head.

Historical, from the original implementation (2026-08). `verify-reconcile.sh` and
the shell block it drove were retired by HARNESS-139, so this output can no longer
be reproduced. It is kept as the record of what was measured then:

```
$ bats tests/pi-packages.bats
1..16   all ok

$ specs/AI-030-pi-packages-manifest/verify-reconcile.sh
[OK] AC4  first run installed all 9 declared packages
[OK] AC5  second run installed 0 (changed=0)
[OK] AC6  object-form entries recognised, 0 reinstalled
[OK] AC7  pi absent: warned, exit 0, bootstrap continues
[OK] AC4-AC7 verified against the block extracted from setup-linux.sh:954-1003
exit 0
```

**No regressions**: the full suite is green on this branch (see the PR body for
the run and its exit status).

**Manual smoke test — NOT performed, deliberately.** The reconcile has not been
run against the real `pi` on this machine, because doing so installs nine
third-party packages that upstream documents as running with full system access,
into the owner's live agent. That is the owner's call, not the implementing
session's. What the stub proves is which call is made and that the set
difference is correct; what only a real run proves is that `pi install` accepts
these arguments. That gap is stated rather than papered over — it is exactly the
limitation `tests/stub-real-pairing.bats` exists to keep visible (BUG-055).

## Decisions made during implementation

- **The `packages` array does not go in `ai/pi/settings.json`.** That is the
  obvious placement and it is wrong: the file is seed-if-missing (#754), so a
  declaration there reaches a fresh machine and never an existing one — the
  opposite of the requirement. `pi install` writes the live array itself, and it
  also unpacks the package to disk, so an array entry written by setup would
  name something that was never installed. One mechanism, one owner.
- **Keyed on `$PI_BIN`, not `pi`.** Measured mid-implementation: `pi` on this
  machine is a shell function wrapping `dotf secrets run`, and it fails with
  `bitwarden vault is locked` while `~/.local/bin/pi --version` returns `0.84.2`.
  A bootstrap must not require an unlocked vault to install an extension.
- **The reconcile is additive, never subtractive.** A package present but
  undeclared is left alone. Uninstalling a human's own extension is not setup's
  decision, and "converge exactly" would make this manifest an authority it has
  not earned.
- **`jq -er`, not `jq -r`.** A malformed manifest yields an empty want-list, and
  an empty want-list installs nothing while logging exactly like "everything is
  already present" — silent success on a broken input.
- **The packaged `subagent` wins; the hand-wired link is quarantined** (#1243).
  Two extensions provided the tool. The decision is for the one the manifest
  declares — `npm:pi-subagents@0.56.0` — and against the 2026-08-09 symlink into
  pi's bundled examples, on four grounds. It is **reproducible**: setup installs
  it on every run and on a fresh machine, while nothing recreates the symlink.
  It is **versioned**: the manifest pins `@0.56.0`, so upstream's publish passes
  a diff and a reviewer, whereas the link tracks whatever the examples directory
  happens to contain. It is **portable**: the link hard-codes
  `~/.nvm/versions/node/v24.16.0/`, so it rots when that node version is removed
  and is invisible to any other node. And it is **maintained**: the packaged
  version is 0.56.0 against examples last touched in the pi release that shipped
  them. The loser is quarantined rather than deleted — the choice is about which
  one loads, not about discarding the user's; it is restorable with `mv` and its
  target inside pi's npm package is never touched.
- **The spec was written after the implementation.** The Discipline Gate trigger
  was found while measuring the diff for the PR, not while scoping. Recorded in
  `tasks.md` rather than disguised; a back-dated task list would make this folder
  a worse record than none.
- **`created:` reads 2026-08-25 for work done on 2026-08-24.** `dotf spec init`
  stamps that field from a UTC clock — the defect #1217 fixed for `dotf mem`,
  still live in `spec`. Filed as **#1225** and left uncorrected here so the
  evidence survives.

## Promotion candidates

- **Nothing for the vault.** The seed-if-missing trap, the `$PI_BIN` wrapper trap
  and the manifest shape are all specific to this repository's deployment of
  pi — build/operate detail, which belongs in the repo (`ai/pi/README.md`,
  `ai/pi/packages.json`'s own `$comment`) and not in the cross-project store.
- The one candidate that is genuinely cross-project — *"a seed-if-missing config
  cannot carry new declarations to an existing machine"* — is already the
  generalisation of `pattern-decision-persistence` rather than a new pattern, and
  a second instance should exist before it is promoted.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-231-a-hand-wired-dev-symlink-outranks-the-managed-instal.md
- [x] ADR-worthy decision? no: the manifest and its declaration semantics are recorded in `ai/pi/README.md` and on epic #1625
- [x] New pattern candidate for `00_meta/patterns/`? no: the one cross-project candidate needs a second instance first, as said above

## Round-1 review (2026-10-01)

Verdict FAIL (`nan/qwen3.8-flash`). The spec text still declares removal out of scope, while the code removes undeclared packages since HARNESS-139 (D-2 on #1625).

| Finding | Disposition |
|---|---|
| F-01 Major REAL: "Removing packages" declared out of scope, code removes | Open. A contract edit to `proposal.md`, followed by round 2. |
| F-02 Major REAL: Windows twin has no `~\.local\bin\dotf.exe` fallback | Ticketed: #1925 |
| F-03 Minor REAL: `1..16` and the retired `verify-reconcile.sh` block shown as current | Applied: "Test status" now shows the current `1..14` and labels the old block as historical |
| F-04 Minor REAL: f2 duplicates the pin regex | Open, in the same contract round as F-01 |
| F-05 Minor THEORETICAL: `LiveSources` drops a source-less object entry | Ticketed: #1926 |
| Unnumbered Minor SPECULATIVE: `Identity()` and dist-tags or paths | Declined. The CI pin guard admits only `npm:<name>@<semver>`, so the case cannot reach the parser. |
| F-06 Minor REAL: `ai/pi/README.md` says install-only | Applied: both README sentences name the removal |
| Question: `features.json` all `pending` | The harness writes `passing`; the commands were run by hand and exit 0 |
