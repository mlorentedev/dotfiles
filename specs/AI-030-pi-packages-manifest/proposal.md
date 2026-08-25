---
id: "AI-030-pi-packages-manifest"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-25"
issue: "mlorentedev/dotfiles#1224"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# AI-030-pi-packages-manifest

> The `created:` date above reads 2026-08-25 and the work happened on 2026-08-24.
> `dotf spec init` stamps that field from a UTC clock — filed as **#1225**, and
> left uncorrected on purpose so the evidence survives the fix.

## Why

pi packages — extensions, skills, prompts and themes shipped over npm — were
installed by hand on one machine and recorded nowhere. A redeploy did not
reproduce them and a second machine never received them, which for a repository
whose whole purpose is reproducing an environment is a hole in the middle of it.
This repository is under active development and gets deployed repeatedly with
updates, so "install it again by hand" is not a workaround; it is the thing that
will not happen.

## What

`ai/pi/packages.json` declares the wanted packages, each pinned to a version.
`setup-linux.sh` and `setup-windows.ps1` reconcile that declaration against the
live `~/.pi/agent/settings.json` on **every** run and install the difference
through `pi install`. A machine that already has pi converges on the next setup
run; a fresh machine converges on its first. Re-running changes nothing and says
so.

## Out of scope

- **Removing packages.** The reconcile is additive: it installs what is declared
  and missing. It never uninstalls what is present and undeclared, because a
  package installed deliberately outside this manifest is not evidence of drift
  and `pi remove` on a human's own extension is not setup's decision to make.
- **Version convergence.** A declared `@0.4.6` against a live `@0.5.0` is a
  different entry, so the reconcile installs the declared one; it does not
  compare or downgrade. Upgrades are a manifest edit, which is the point —
  a diff and a reviewer between upstream's publish and this machine.
- **Auditing the nine packages.** Pinning bounds the risk; it does not review
  the code. The `why` field is where that review will be recorded when it
  happens.
- **Project-scoped packages** (`.pi/settings.json`, `pi install -l`). User scope
  only.
- **#1225** — the UTC date stamp this spec's own frontmatter carries.

## Risks / open questions

- **Supply chain.** Upstream: *"Pi packages run with full system access —
  extensions execute arbitrary code and skills can instruct the model to run
  executables."* That is inside an agent holding `NAN_API_KEY` with write access
  to these repositories. Pinning is the floor, not the answer. **Resolved for
  this PR**: every entry pinned, guard refuses an unpinned one, `why` recorded
  per entry. **Left open**: no code review of the nine.
- **`pi install` on Linux must not need an unlocked vault.** `pi` is a shell
  function wrapping `dotf secrets run` and fails on a locked Bitwarden vault.
  **Resolved**: the reconcile calls `$PI_BIN` (the raw binary), as the existing
  version check already does. Measured: `~/.local/bin/pi --version` -> `0.84.2`
  with the vault locked, while the function errors.
- **The live array holds two entry shapes.** Upstream allows `"npm:pkg"` and
  `{"source": "npm:pkg", ...}`. A reader handling only strings would reinstall
  every filtered entry on every run. **Resolved**: both forms read on both
  platforms, verified by scenario.
- **Open**: whether pi auto-installs missing packages declared in **user**-scoped
  settings at startup. Upstream documents that only for project scope. The
  reconcile loop does not depend on the answer, which is why it is the mechanism
  rather than declaring the array and hoping.

## Acceptance criteria

- [x] **AC1** — `ai/pi/packages.json` is valid JSON, declares at least one
      package, has no duplicate `source`, and every entry carries a non-empty
      `why`.
- [x] **AC2** — every declared `source` is pinned to an explicit version, and
      the guard enforcing that is itself proven to reject an unpinned source.
- [x] **AC3** — the `packages` array is absent from `ai/pi/settings.json`, and
      neither setup script writes that array directly.
- [x] **AC4** — a first run on a machine with none of them installed installs
      every declared package.
- [x] **AC5** — a second run installs nothing and reports no change
      (idempotent, `changed=0`).
- [x] **AC6** — entries already present in the live array in the **object** form
      are recognised and not reinstalled.
- [x] **AC7** — with pi absent the reconcile warns and the bootstrap continues
      (exit 0); it never aborts setup.
- [x] **AC8** — an unreadable or empty manifest is reported, never treated as
      "nothing to do".
- [x] **AC9** — Linux installs through `$PI_BIN`, not the `pi` shell function.
- [x] **AC10** — `setup-windows.ps1` reconciles the same manifest with the same
      semantics (parity), and adds no non-ASCII to the file.

## References

- Bitácora board: `mlorentedev/dotfiles#1224`
- Upstream docs: <https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md>
- Prior art in this repo: `ai/deploy.json` (CLI-039, #1023) — the same
  "declarative table, one behaviour, no per-OS twin logic" shape.
- The contract this design works around: `ai/pi/README.md` and
  `tests/pi-config.bats` on seed-if-missing (#754).
- `docs/lessons/lesson-228-...` and #1225 — the `created:` date above.
