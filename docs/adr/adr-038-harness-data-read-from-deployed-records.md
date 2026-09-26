---
id: "ADR-038-harness-data-read-from-deployed-records"
type: adr
status: accepted
owner: manu
date: "2026-09-25"
supersedes: []
extends: [adr-026-agent-config-ssot-topology]
issue: mlorentedev/dotfiles#1693
tags: [architecture, decision, harness, deploy, release, skills]
created: "2026-09-25"
---

# ADR-038: Data about deployed content is read from the deployed records; a copy compiled into dotf is a tested fallback

## Context

A skill's prerequisites were declared twice: in its record's `requires:` frontmatter, and in a map compiled into `dotf`. The router read only the compiled map, and the two had drifted in three rows.

The binary and the records also deploy on different clocks. After SKILL-001 retired a skill, the records deployed the same day, but the installed binary kept suggesting the retired skill on every prompt until a release was installed. The trigger rules already worked the other way: read from disk, with an embedded copy as fallback. The dependency map was the exception.

## Decision

1. **Read at run time.** When `dotf` needs data that describes content the harness deploys (skills, triggers, personas), it reads the deployed records at run time. It reads them from the harness root it resolved: `$DOTFILES_DIR`, else the checkout.
2. **A compiled copy is only a fallback**, for a machine with no records to read. A test keeps it equal to the committed records (`TestTheCompiledDependencyMapMatchesTheRecords` for the dependency map).
3. **Records deploy before the binary that reads them:** mirror, then install. The release runbook (`docs/runbooks/release-dotf.md`) fixes that order.

## Consequences

- A retirement or a new prerequisite reaches every agent when the records deploy, not when a release is installed.
- The prompt hook reads about 31 records per prompt. Measured: 14 ms per call before, 18 ms after.
- A machine whose records are older than its binary sees the older records. The deploy order in decision 3 is what prevents it.
- Lesson 296 records the failure. The implementation is #1714.
