---
id: "lesson-367-a-gate-that-needs-a-binary-is-only-as-present-as-the-binary"
type: lesson
status: active
title: "A gate that needs a binary is only as present as the binary"
created: "2026-10-08"
---

# A gate that needs a binary is only as present as the binary

## Context
The knowledge vault's only secret gate is gitleaks, which runs through pre-commit. A global
`core.hooksPath` sends every stage to the GUARD dispatcher (`git-hooks/`), and the dispatcher hands
the stage to `pre-commit hook-impl` in any repo that declares `.pre-commit-config.yaml`.
`dotf doctor` checks the gate (`checkVaultHooks`).

## The Trap
The Mac had the dispatcher deployed and no pre-commit installed, because no bootstrap installed
it. Three things then failed silently, each in a way that looked like a pass:

- The dispatcher exited 0 when the binary was missing. A no-op was meant for repos that declare
  no gate, but it applied to repos that declare one too. Every vault commit and push skipped
  gitleaks.
- Doctor asked whether each stage resolved to the dispatcher, and it did. It looked for the binary
  only under `--fix`, so check mode printed "gitleaks gate active".
- The only installer, `scripts/install-precommit.sh`, used `pip install` and then ran
  `pre-commit install`. That command refuses outright while `core.hooksPath` is set ("Cowardly
  refusing…"), so the script failed on every provisioned machine, and nothing called it anyway.

## The Solution
- pre-commit is a `uv-tool` entry in `packages.json` on every OS, installed by
  `dotf tools install`.
- Doctor FAILs in check mode when the binary is absent, before it probes any stage.
- The dispatcher fails closed on the stages that can block (pre-commit, pre-push, commit-msg) and
  names the install command. It also looks in uv's tool bin dir, because GUI launchers such as
  obsidian-git do not read the shell rc.
- The installer is deleted: the dispatcher does its hook half, and the catalog does its install
  half (#2183).

## The Rule
A check that a gate is wired must also check that the gate can run. Probe every link down to the
binary that does the scanning: a hook path pointing at a dispatcher proves only that the
dispatcher runs. And a hook that cannot run its gate fails closed on the stages that can block.
`--no-verify` is the deliberate way past it.
