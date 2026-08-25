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
| AC4 first run installs all | `verify-reconcile.sh` — `[OK] AC4 first run installed all 9 declared packages` |
| AC5 second run changes nothing | `verify-reconcile.sh` — `[OK] AC5 second run installed 0 (changed=0)` |
| AC6 object-form entries recognised | `verify-reconcile.sh` — `[OK] AC6 object-form entries recognised, 0 reinstalled` |
| AC7 pi absent warns, never aborts | `verify-reconcile.sh` — `[OK] AC7 pi absent: warned, exit 0, bootstrap continues` |
| AC8 unreadable manifest is loud | `tests/pi-packages.bats` — "refuses an unreadable manifest instead of reading it empty" |
| AC9 Linux uses `$PI_BIN` | `tests/pi-packages.bats` — "installs through $PI_BIN, not the shell function" |
| AC10 Windows parity, no new non-ASCII | `tests/pi-packages.bats` — three `setup-windows` cases; non-ASCII line count 10 before and 10 after |

All on commit `2c20332` plus the spec commit that follows it.

## Test status

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

$ bash -n setup-linux.sh && zsh -n setup-linux.sh
OK / OK

$ shellcheck setup-linux.sh
20 findings before the change, 20 after, none in the new block (lines 954-1003)

$ shellcheck specs/AI-030-pi-packages-manifest/verify-reconcile.sh
clean
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
