---
id: "HARNESS-074-deployed-doctrine-probes"
type: spec
status: archived
created: "2026-08-18"
owner: manu
issue: "mlorentedev/dotfiles#1035"
tags: [spec, proposal, harness, doctor]
template_version: "1.0"
---

# HARNESS-074: Deployed Doctrine Probes in Doctor

> **Naming**: file lives at `<repo>/specs/HARNESS-074-deployed-doctrine-probes/proposal.md`.

## Why

AC2 of HARNESS-072 requires enforced regions (e.g. `pr-stewardship`, `definition-of-done`, `no-attribution`) to reach 5 surfaces (`AGENTS.md`, `ai/claude/CLAUDE.md`, `~/.claude/CLAUDE.md`, `~/.gemini/GEMINI.md`, `~/.codex/AGENTS.md`). While CI verifies in-repo targets, per-machine deployment state in `$HOME` has had no automated verification in `dotf doctor`. If `--deploy` fails silently or drops a region, agents run uninstructed while all checks appear green.

## What

1. Adds `checkDeployedDoctrine(sys, cfg, rep)` to `cli/internal/doctor/checks_deploy.go` (called within `checkHarnessDrift`).
2. Reads `harness/manifest.json` `enforced` regions and verifies that deployed surfaces targeted by `doctrine.deploy` (`~/.gemini/GEMINI.md`, `~/.codex/AGENTS.md`) and `agents.presence` (`~/.claude/CLAUDE.md`, etc.) contain the enforced markers when present.
3. Reports precise FAIL naming the specific region ID and deployed surface (never dumping secret/verbose content).
4. Includes comprehensive unit tests and mutation tests asserting doctor fails when an enforced region is omitted from a deployed target.

## Out of scope

- Modifying the injection mechanics of `compile-harness.sh`.
- Re-architecting `harness/manifest.json` schema.

## Risks / open questions

- Non-deployed targets on machines without certain tools: gracefully skips surfaces that are not deployed or whose tool is absent, avoiding false positives.

## Acceptance criteria

- [ ] AC1: `dotf doctor` fails when an enforced region declared in manifest is missing from a deployed surface.
- [ ] AC2: `dotf doctor` passes when all deployed surfaces carry their declared enforced regions.
- [ ] AC3: Mutation test demonstrates doctor goes RED when an enforced region is removed from a deployed target.

## References

- Bitácora work-gate: [mlorentedev/dotfiles#1035](https://github.com/mlorentedev/dotfiles/issues/1035)
- Prior work: `HARNESS-072-pr-stewardship`


<!-- archived 2026-08-18 — PR: https://github.com/mlorentedev/dotfiles/pull/1046 -->
