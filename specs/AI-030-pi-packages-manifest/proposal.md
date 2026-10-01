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
`setup-linux.sh` and `setup-windows.ps1` both call `dotf pi packages apply` on
**every** run, which reconciles that declaration against the live
`~/.pi/agent/settings.json` in both directions: it installs what is declared and
missing, and removes what is live and undeclared (HARNESS-139). A machine that
already has pi converges on the next setup run; a fresh machine converges on its
first. Re-running changes nothing and says so.

> Amended 2026-10-01 after the round-1 review. The original reconcile was
> additive, and this section declared removal out of scope. HARNESS-139 (#1628)
> replaced the shell twins with `dotf pi packages apply`, which removes
> undeclared packages so that a manifest edit can purge one (D-2 on #1625,
> pi-memory). On 2026-10-01 the owner accepted the general rule: whatever the
> manifest does not declare is removed, including a package installed by hand.

## Out of scope

- **Version convergence.** A declared `@0.4.6` against a live `@0.5.0` is a
  different entry, so the reconcile installs the declared one; it does not
  compare or downgrade. Upgrades are a manifest edit, which is the point —
  a diff and a reviewer between upstream's publish and this machine.
- **Auditing the ten packages.** Pinning bounds the risk; it does not review
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
  per entry. **Left open**: no code review of the ten.
- **`pi install` on Linux must not need an unlocked vault.** `pi` is a shell
  function wrapping `dotf secrets run` and fails on a locked Bitwarden vault.
  **Resolved**: the reconcile calls `$PI_BIN` (the raw binary), as the existing
  version check already does. Measured: `~/.local/bin/pi --version` -> `0.84.2`
  with the vault locked, while the function errors.
- **The live array holds two entry shapes.** Upstream allows `"npm:pkg"` and
  `{"source": "npm:pkg", ...}`. A reader handling only strings would reinstall
  every filtered entry on every run. **Resolved**: both forms read on both
  platforms, verified by scenario.
- **Removal is total.** The manifest is the whole declaration, so a package
  installed by hand and not declared is removed on the next setup run. That is
  the accepted cost of convergence (owner, 2026-10-01): a package worth keeping
  is declared. **Mitigated**: an empty manifest is refused rather than read as
  "remove everything", an unparseable live `settings.json` is an error rather
  than an empty list, and both setup twins pass `--repo` so a worktree's own
  manifest drives the run. `dotf pi packages check` shows the plan without
  applying it.
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
- [x] **AC7** — with pi absent, **or npm absent**, the reconcile warns once and
      the bootstrap continues (exit 0); it never aborts setup, and it never
      reports a missing toolchain as N separate package failures.
- [x] **AC8** — an unreadable or empty manifest is reported, never treated as
      "nothing to do".
- [x] **AC9** — Linux setup hands `$PI_BIN` (the raw binary) to
      `dotf pi packages apply --pi`, not the `pi` shell function.
- [x] **AC10** — `setup-windows.ps1` reconciles the same manifest through the
      same `dotf pi packages apply`, naming its own checkout (parity), and adds
      no non-ASCII to the file.
- [x] **AC11** — a package that is installed and declared is also **loaded**.
      Added 2026-08-26 for #1243, and it is the criterion whose absence this
      spec's own verification demonstrated: AC1–AC10 all hold while pi refuses
      to start. Measured that day — `pi -p` exited 1 with
      `Failed to load extension ".../pi-subagents/index.ts": Tool "subagent"
      conflicts with ~/.pi/agent/extensions/subagent/index.ts`, a hand-wired
      symlink from 2026-08-09 into pi's own bundled examples. The declared
      package was the one that never loaded, and nothing here could see it,
      because counting entries in an array is not observing effect.
      Enforced by `dotf doctor` (`cli/internal/doctor/checks_pi_extensions.go`,
      repaired under `--fix`), which is the durable surface: this is machine
      state, so CI cannot observe it and a doctor check is the only thing that
      runs where the defect lives.

- [x] **AC12** — a live package that the manifest does not declare at any
      version is removed before the declared ones are installed, and an empty
      manifest is refused rather than read as "remove everything". Added
      2026-10-01 so the removal the code performs is a criterion, not a
      side effect.

## References

- Bitácora board: `mlorentedev/dotfiles#1224`, and `#1243` for AC11.
- Upstream docs: <https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md>
- Prior art in this repo: `ai/deploy.json` (CLI-039, #1023) — the same
  "declarative table, one behaviour, no per-OS twin logic" shape.
- The contract this design works around: `ai/pi/README.md` and
  `tests/pi-config.bats` on seed-if-missing (#754).
- `docs/lessons/lesson-228-...` and #1225 — the `created:` date above.
