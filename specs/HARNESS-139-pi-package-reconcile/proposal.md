---
id: "HARNESS-139-pi-package-reconcile"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#1628"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-139-pi-package-reconcile

> **Naming**: file lives at `<repo>/specs/HARNESS-139-pi-package-reconcile/proposal.md`. `HARNESS-139-pi-package-reconcile` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

pi's `npm:pi-memory@0.4.2` is provisioned from `ai/pi/packages.json`, but its retrieval dependency, `qmd`, was never provisioned. So `memory_search` answers with an install hint, while `~/.pi/agent/memory/` keeps growing (236 KB on msi, measured 2026-09-25). A memory layer that reads as present and cannot search is worse than none. It is also a third memory store beside the vault, outside the single-sink doctrine (GUARD-001, D-8 on #1625). Decision D-2 on #1625 is to purge it.

The purge cannot be a manifest edit. The code that applies `ai/pi/packages.json` is a shell twin pair, `setup-linux.sh` (about 100 lines) and `setup-windows.ps1`, and it only **installs** the difference between the manifest and the live `settings.json`. It never removes anything. Deleting the entry would leave the package installed on every existing machine. The owner's decision (2026-09-24, #1628): port the reconciler to Go, converging in both directions, and make the purge a declaration. No uninstall logic goes into the twins, and there is no manual purge.

## What

1. **`dotf pi packages check`** reads `ai/pi/packages.json` and the live `~/.pi/agent/settings.json`, and prints what is missing and what is undeclared. It exits non-zero on drift, and on a manifest it cannot read: an unreadable manifest must never read as "nothing declared".
2. **`dotf pi packages apply [--dry-run]`** converges both ways through pi's own CLI, in this order:
   1. `pi remove <source>` for each live entry whose **package** the manifest does not declare at any version;
   2. `pi install <source>` for each declared entry that is not live at exactly that source, which covers a version bump, because `pi install` replaces the entry of the same package (measured below);
   3. the retired paths (item 3). They move after the removals, so a purged extension is gone before its data is.

   It never writes `settings.json` itself, because pi owns that file (#754). A second run changes nothing and says `changed=0`.
   - It keeps what CI-003 established for every install and removal: the output is captured, the elapsed time is logged on every outcome, and the output is printed fenced when the command fails or takes more than 120 s. The threshold gates verbosity only.
   - `DOTFILES_SKIP_PI_PACKAGES` keeps its CI-002 contract. When it is set, the command skips loudly, says what was not verified, and exits 0.
   - The pi binary is resolved explicitly: `--pi <path>`, else `~/.local/bin/pi` (the prefix setup installs into), else the first `pi` on `PATH`. A missing pi or npm is a warning with exit 0, as in the twins, so setup degrades instead of breaking.
3. **Retired paths are declared in the manifest.** A new `retire` list names paths under `~/.pi/agent/`, each with a `why`. `apply` moves each existing one to `~/.pi/agent/archive/<name>-<UTC date>/` and never deletes it. The first entry is `memory`. A second run finds nothing to move.
4. **The retrieval dependency is declared too.** A package entry may carry `requires`, a list of executables it needs on `PATH`. A new check in `dotf doctor`'s pi section, reading the manifest through the same loader as the reconciler, FAILs for any declared package whose requirement does not resolve, which is the state pi-memory was in. With pi-memory purged, the check passes on today's manifest. #1628 asks for this "for any harness". pi is the only harness with a package manifest, so the check is scoped to it; a second harness that grows one would reuse the `requires` field.
5. **Both twins call the command.** Each setup script's pi package block is replaced by one `dotf pi packages apply` call (Linux passes `--pi "$PI_BIN"`). No uninstall logic is added to either script, and about 100 lines of shell and its PowerShell twin are deleted. `tests/pi-packages.bats` keeps its manifest-level tests. The tests that pinned the shell block are replaced by Go tests, plus a bats test that each twin calls the command (ADR-020 §5).
6. **`npm:pi-memory@0.4.2` is removed from the manifest**, and the next `apply` on each machine removes it.

## Delivery: two PRs

Measured after the first draft, the change is about 320 executable lines, over the ~300 cap. It ships in two PRs, split where the second adds a declaration kind rather than more of the same logic:

- **PR-A**: the reconciler, `check` and `apply`, both twins replaced, the CI filter, and pi-memory removed from the manifest (AC1-AC4, AC7). After it, the next `apply` on each machine removes pi-memory; its data directory is left untouched.
- **PR-B**: `retire` and `requires`, with the doctor check (AC5, AC6), and `memory` declared retired. After it, AC8 can be closed live.

The spec spans both and archives with PR-B.

## Out of scope

- The pi CLI's own install and upgrade (the synthesis's P0.4 on #1628). It stays in setup until a separate port.
- **Fanning the installs out** (P1.4). Rejected for now: `pi install` writes the live `settings.json`, and two concurrent installs would race on that file. Parallelism needs pi to serialise its own writes, or a measured proof that it does.
- Other harnesses' packages. No other harness has a package manifest today.

## Risks / open questions

- **Measured on pi 0.87.1, 2026-09-25, in a scratch `HOME`:**
  - `pi install npm:pi-effort@0.0.5` over a live `npm:pi-effort@0.0.8` **replaces** the entry, leaving one entry at the new version. A version bump is therefore an install, never a removal plus an install.
  - `pi remove npm:pi-effort@0.0.5` removes an object-form entry (`{"source": …, "extensions": […]}`) by its `source`.
  - `pi remove` of a source that is not installed exits 1 ("No matching package found"). The plan only removes what it read as live.
  - Install output is noisy (npm's audit lines), which is one more reason it is captured rather than streamed.
- **AI-030 is superseded in part.** Its spec (`specs/AI-030-pi-packages-manifest/`, #1224, still active) introduced the manifest and the install-only reconcile. Its reconcile criteria (AC4-AC7) move here, and its `verify-reconcile.sh` drives the shell block this change deletes, so those verifiers stop being runnable. The manifest criteria still hold. AI-030 is dispositioned by the zombie-spec sweep (W1.4, #1626), not in this change, so this diff stays one concern.
- **The first live `apply` is deploy-class.** It removes a package and moves a directory on a machine where other sessions run pi. It is announced to peers first, and the PR carries the live `check` and `apply --dry-run` output rather than a run from the branch.

- **Removal is total.** A package installed by hand, and not declared, is removed on the next `apply`. That is the declaration semantics the owner chose. `check` and `--dry-run` name every removal first, and a package someone wants to keep is declared.
- **A live entry in object form** (`{"source": …, …}`, upstream's per-resource filter form) is compared by its `source`, as the twins did, and removed by its `source`.
- **Windows.** pi on Windows is `pi.cmd` on `PATH`. The resolution order finds it, and the Windows CI leg runs setup, and so the command, for real.

## Acceptance criteria

- [ ] **AC1** — `apply` installs a declared missing entry and removes an undeclared live one, through `pi install` and `pi remove` only, and a second run reports `changed=0` with no call to pi. Tested against a fake pi that edits a real `settings.json`.
- [ ] **AC2** — `check` exits non-zero naming each missing and undeclared entry, and exits non-zero on an unreadable or empty manifest.
- [ ] **AC3** — Every install and removal logs its elapsed time. A failed or slow one prints its captured output inside a fence, and the threshold never stops an install.
- [ ] **AC4** — `DOTFILES_SKIP_PI_PACKAGES` skips loudly and exits 0 before any probe. A missing pi or npm is a warning with exit 0.
- [ ] **AC5** (PR-B) — A declared `retire` path is moved under `~/.pi/agent/archive/` with its contents intact and is never deleted, and a second run moves nothing.
- [ ] **AC6** (PR-B) — The doctor check FAILs on a fixture that declares a package whose `requires` does not resolve, and PASSes on the shipped manifest.
- [ ] **AC7** — Both setup twins call `dotf pi packages apply`, and neither carries the reconcile loop any more. The CI path filter that selects the pi job covers `cli/internal/pi/**` as well as the manifest and both twins.
- [ ] **AC8** (live, msi) — After `apply`: `jq '.packages' ~/.pi/agent/settings.json | grep -c pi-memory` is `0`, `~/.pi/agent/memory` is under `~/.pi/agent/archive/`, a fresh `pi` session registers no `memory_*` tool, and a second `apply` reports `changed=0`.

## References

- #1628 (HARNESS-139), epic #1625 (W1.7, decisions D-2 and D-8, and the owner's decision of 2026-09-24).
- AI-030 (#1224): the manifest and the install-only reconcile. CI-002 (#1478): `DOTFILES_SKIP_PI_PACKAGES`. CI-003 / #1486: captured output and timing.
- ADR-020 §5: strangler-fig, where a touched twin is ported and its tests go with it.
- `forge protection {check,apply}` (GUARD-017): the check/apply shape this follows.
