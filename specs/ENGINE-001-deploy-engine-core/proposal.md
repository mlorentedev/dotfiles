---
id: "ENGINE-001-deploy-engine-core"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-05-28"
tags: [spec, proposal, engine-001, harness-001, deploy-engine, cross-agent, tracer-bullet]
template_version: "1.0"
---

# ENGINE-001-deploy-engine-core

> **PR-1 of the HARNESS-001 epic** (`specs/HARNESS-001-unified-cross-agent-harness/`, GH #162). The engine core, delivered as the *no-attribution tracer-bullet*: smallest payload through the whole pipeline. Consumers (SDD-008, IDEAS-007, #156, #159) build on what this ships.

## Why

<!-- from 11-tasks.md: PR-1 of HARNESS-001 (#162): manifest-driven agent-artifact deploy engine, delivered as the no-attribution tracer-bullet. -->

The no-attribution policy regressed silently (#156) because there is no pipeline from the SSOT rules (`00_meta/patterns/`) to the instruction files agents read; harness defaults override absent rules at runtime. This PR builds the **engine core** and proves it end-to-end on the smallest possible payload — the no-attribution rule — so the full pipeline (manifest → source-marker → compile → commit → drift guard) exists and the #156 regression class cannot recur undetected. Scaling to all agents and all artifact kinds is the consumers' job; this PR ships the spine.

## What

Observable behaviour after this PR (Linux; Windows parity is PR-2):

1. **`harness/manifest.json`** declares enforced rules and targets:
   ```json
   { "version": 1,
     "enforced": [ { "id": "no-attribution",
                     "source": "00_meta/patterns/pattern-git-workflow.md#attribution-policy" } ],
     "targets":  [ { "agent": "claude", "kind": "pointer", "file": "ai/claude/CLAUDE.md", "inject": ["no-attribution"] },
                   { "agent": "agents", "kind": "native",  "file": "AGENTS.md",          "inject": ["no-attribution"] } ] }
   ```
2. **`compile-harness.sh --refresh`** (needs the vault): extracts the named section from the vault pattern, writes a **committed source-of-record** at `harness/enforced/no-attribution.md`, and injects a marker-delimited override block into each target file.
3. **`compile-harness.sh --check`** (fully offline, no vault): renders from the committed source-of-record and diffs against the deployed blocks; non-zero exit on any drift.
4. **Marker-delimited regions** are replaced idempotently; everything outside the markers stays hand-authored:
   ```
   <!-- BEGIN GENERATED no-attribution (src: pattern-git-workflow.md#attribution-policy sha256:<16>) -->
   …generated override text…
   <!-- END GENERATED no-attribution -->
   ```
5. **Healthcheck guard** — `scripts/healthcheck.sh` gains a `check_harness` assertion that fails when a deployed block ≠ render(source-of-record). Runs locally with no vault (the type-A / #156 guard; per the 2026-05-28 decision, the guard lives here, not in CI).
6. **Line-cap assertion** — `--refresh` fails if injecting a block pushes `ai/claude/CLAUDE.md` over 80 lines.
7. **`setup-linux.sh`** invokes `compile-harness.sh --refresh` as part of deploy, so the build IS setup.

## Out of scope

- **Windows `compile-harness.ps1` + Pester** — PR-2 (sequenced in the umbrella). PR-1 is the Linux tracer-bullet to prove the pipeline.
- **Agents beyond Claude + AGENTS** — agy/Copilot/OpenCode/Pi targets are PR-3.
- **Consumers** — skills (SDD-008), `.agent/<id>/` registry (IDEAS-007), full `enforce:true` discovery (#156), work/personal mode (#159).
- **Auto-discovery of enforced patterns** — PR-1 lists the one rule explicitly in the manifest; frontmatter `enforce:true` scanning is the #156 consumer.
- **Splitting patterns into per-rule files** — D3 decision: extract by section anchor now; the split belongs to #156.

## Risks / open questions

- **FM1 — marker tampering**: a missing/duplicated `END` marker must make `--refresh` *fail loudly*, never silently append (that reintroduces the drift class). Guarded by AC5.
- **FM2 — line-cap breach**: injection can push `ai/claude/CLAUDE.md` over 80 lines (opencode.bats #34). `--refresh` asserts the cap post-injection and fails. Guarded by AC4.
- **FM3 — nondeterministic order**: multiple injected rules must sort stably by `id`, or every recompile churns the diff. (Only one rule in PR-1, but the sort lands now.)
- **Section-anchor fragility (D3)**: if the heading named by the anchor is absent, the extractor errors clearly rather than emitting an empty block.
- **Vault dependency of `--refresh`**: reuse SDD-008's preflight — if the vault is absent, `--refresh` aborts with an actionable message; `--check` still works offline.

## Acceptance criteria

- [ ] **AC1** — `compile-harness.sh --check` exits 0 on a freshly-refreshed tree; exits non-zero (with a diff summary) after a deployed block is hand-edited. *(bats)*
- [ ] **AC2** — Re-running `--refresh` is idempotent (no diff); generated blocks carry `BEGIN/END GENERATED no-attribution` markers with source path + sha256 prefix. *(bats)*
- [ ] **AC3** — `harness/enforced/no-attribution.md` exists and is committed; `--check` renders from it with **no vault access** (test runs with `VAULT_PATH` pointed at an empty dir). *(bats)*
- [ ] **AC4** — Injecting past the 80-line cap on `ai/claude/CLAUDE.md` makes `--refresh` fail with a clear error. *(bats fixture)*
- [ ] **AC5** — A missing `END` marker makes `--refresh` fail loudly (no silent append). *(bats fixture)*
- [ ] **AC6** — `healthcheck.sh` flags a tampered deployed block (offline). *(bats / healthcheck test)*
- [ ] **AC7** — The no-attribution override text appears in deployed `ai/claude/CLAUDE.md` + `AGENTS.md` between the markers. *(grep test)*

## References

- Umbrella: `specs/HARNESS-001-unified-cross-agent-harness/` (epic AC1, AC3, AC4, AC5, partial AC7)
- ADR: `docs/adr/adr-013-agent-artifact-deploy-engine.md` (this engine); `docs/adr/adr-012-deploy-strategy-copy-with-drift-assertion.md` (copy substrate + healthcheck drift)
- SSOT rule: vault `00_meta/patterns/pattern-git-workflow.md` §Attribution Policy
- Tracker: GH #162 (epic), #156 (the regression this proves a guard for)
