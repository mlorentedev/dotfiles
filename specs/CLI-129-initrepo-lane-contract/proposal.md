---
id: "CLI-129-initrepo-lane-contract"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-04"
issue: "mlorentedev/dotfiles#2552"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# CLI-129-initrepo-lane-contract

## Why

<!-- from issue #2552: CLI-129: dotf initrepo scaffolds the lane contract in every repo -->

ADR-050 (accepted in #3143) makes the lane contract the one way parallel sessions coordinate: a committed `.dotf/lane.json` queue, a prompt template, and a shared section in `AGENTS.md`. Only dotfiles has them. Measured 2026-10-15: kubelab's `AGENTS.md` has no `dotf` reference and no other repository has `.dotf/lane.json`. `dotf init` scaffolds `AGENTS.md` and the SDD section but not the lane contract, so every new repository starts outside the contract and every old one needs a hand-copied file. This serves the standing orchestrator goal: the lane contract becomes a `dotf` primitive that reaches every repository, with no per-agent copy.

## What

`dotf init` gains a lane step, and a sibling `dotf init lane` command runs it alone. Against a target repository it writes, idempotently:

- **`.dotf/lane.json`** with exactly the shape `lane.Config` reads today: an empty `queue`, `branch_prefix` and `base` at their defaults, `template` pointing at the prompt. A fresh repo starts with an empty queue; dotfiles' own queue is never copied.
- **`.dotf/lane-prompt.md`**, a shared template rendered by `lane` with `PromptData` (`.Number`, `.Title`, `.Branch`, `.Remaining`) and nothing else. Verification is generic ("run the repository's verify target"), not a Go toolchain list, so one template serves every stack.
- **A `## Lanes` section in `AGENTS.md`**: lanes, worktrees, claim and landing, in the same append-if-absent shape as the SDD section.

Existing files are left alone. `--force` replaces the lane prompt and the AGENTS.md section in place and leaves `lane.json` alone, since its queue belongs to the repository. A second run reports every step unchanged. The embedded templates are self-contained static scaffolding (no vault SSOT, no `$VAULT_PATH`).

## Out of scope

- `.github/merge-grant.yml`. Scaffolding it into every repository widens agent merge authority everywhere; it is owner-lands and ADR-050 §6 does not list it.
- Copying dotfiles' live queue into another repository.
- The per-repository rollout (kubelab first, then the rest). Each is its own PR in its own repository, tracked as follow-up tickets, not tasks of this spec.
- Any change to `lane.Config`. The issue text mentions a "bootstrap command"; no such field exists and ADR-050 does not name one, so it is dropped as stale. If a repo needs one it is a separate schema change.
- Stack-specific prompt variants. The prompt is generic.

## Risks / open questions

- A scaffolded key `lane.LoadConfig` does not know would break every `dotf lane` call in that repo (`DisallowUnknownFields`). Mitigated by AC1, which loads the scaffolded file with the real loader.
- The prompt template may only use fields `PromptData` passes. AC2 renders it with the real renderer.
- The command is `dotf init`, not `initrepo` (the issue's old name). `--force` is the existing flag; there is no `--refresh`.
- `AGENTS.md` wording is the public face of ADR-050 in other repositories. The section states the contract in a few lines and points at `dotf lane --help`; it does not restate the ADR.

## Acceptance criteria

- [x] AC1: `dotf init lane` on an empty repository writes `.dotf/lane.json`, and `lane.LoadConfig` loads it with an empty queue and the documented defaults.
- [x] AC2: it writes `.dotf/lane-prompt.md`, and the lane renderer renders it with `PromptData` without error.
- [x] AC3: it appends the `## Lanes` section to `AGENTS.md`, creating the file when it is absent and keeping prior content when it exists.
- [x] AC4: a second run changes nothing: every file is byte-identical and the report says `unchanged`.
- [x] AC5: an existing `.dotf/lane.json`, prompt or section is left alone without `--force`.
- [x] AC6: `--force` replaces the prompt and the `## Lanes` section in place and leaves `.dotf/lane.json` untouched.
- [x] AC7: `dotf init` runs the lane step after the AGENTS.md step, and `--dry-run` writes nothing.
- [x] AC8: the follow-up ticket for the kubelab rollout exists and is linked from `verification.md`.

## References

- Bitácora board: mlorentedev/dotfiles#2552, part of epic #2546
- Related ADR: `docs/adr/adr-050-one-lane-contract-for-every-repo-agent-and-forge.md` (§6 Distribution)
- Code: `cli/internal/lane/queue.go` (`Config`, `PromptData`), `cli/internal/initrepo/agents.go` (the section pattern), `cli/internal/initrepo/orchestrator.go`
