---
id: agents-spec-section
type: template
status: active
created: "2026-05-13"
---

# AGENTS.md Spec-Driven Development Section

> Snippet to seed any repo's `AGENTS.md` so agents (Claude, Copilot, Cursor, Codex) discover the spec workflow. The snippet is **self-contained**: it depends only on the `dotf` CLI (on PATH via dotfiles), never on the vault being checked out, and carries no `$VAULT_PATH` literals — so a repo bootstrapped on a vault-less machine still gets a correct section (#248).

## Usage

**Recommended (automated):** run `dotf init agents` from anywhere inside the target repo. It:
- creates `AGENTS.md` with a minimal header if missing, or
- inserts/refreshes the `## Spec-Driven Development` section between its markers (idempotent), preserving the rest of the file.

Pass `--force` to overwrite an existing SDD section in place. On Windows, until a Windows `dotf` install path exists, the legacy `init-repo-agents.ps1` remains the fallback (#380).

**Manual fallback:** copy the section between BEGIN/END SNIPPET markers below into `AGENTS.md`, under a `## Workflow` heading or as its own section. No substitutions needed — the snippet is already self-contained.

## Bootstrap automation

`dotf init agents` (delivered via dotfiles, on PATH after install) implements the recommended flow above. It embeds this snippet in the binary and drift-tests it against this file, so the binary stays in sync with this SSOT without needing the vault at runtime.

---

## --- BEGIN SNIPPET ---

## Spec-Driven Development

**The Lane Decisor Rule:** The agent MUST declare the chosen lane on the VERY FIRST LINE of its response before taking any action (e.g., `Lane: Fast-Track Development`).

**Implementation Lanes:**
- **0. Answer:** Read-only questions, codebase exploration, debugging without writing code.
- **1. Inline Direct:** <30 LOC diff, understood change, single component.
- **2. Fast-Track Development (FTD):** 30–150 LOC, ordinary changes, low ambiguity. 1 document `spec.md`, simple branch, no worktree overhead, observable evidence required.
- **3. Full SDD:** >150 LOC, public contracts, architecture, or high ambiguity. Triad of docs (`proposal.md`, `tasks.md`, `verification.md`).

This repo follows the **Spec-Driven Development per feature** pattern: non-trivial changes are specified before they are implemented.

When the user asks to **create, fill, or archive a spec**, follow this workflow. The `dotf` CLI (installed via dotfiles, on PATH) is the canonical, self-contained interface — run `dotf spec --help` for the full surface.

| Trigger phrase | Action |
|---|---|
| "create a spec for X", "scaffold spec X", "start working on X" | `dotf spec init <feature-id> --issue <N>` |
| "fill in the proposal for X", "help me write the proposal" | edit `specs/<feature-id>/proposal.md` — the Why + acceptance criteria — before writing implementation code |
| "archive spec X", "close spec X" | `dotf spec archive <feature-id> --pr <url>` |

Per-feature specs live at `specs/<feature-id>/` in this repo; archived at `specs/archive/<feature-id>/` (never deleted — audit trail).

**Skip SDD for**: typo fixes, comment-only edits, mechanical refactors, bug fixes <20 lines with obvious cause, doc-only changes.

### Discipline Gate (NON-NEGOTIABLE)

Before creating ANY branch for code changes, Full SDD or FTD is mandatory if ANY apply:

- ~50–300 LOC of production diff (excluding tests, generated files, lockfiles). **The count is
  EXECUTABLE lines** — declarative data, schemas and comment blocks are excluded, because the cap
  rations the control flow a reviewer holds in their head, and a table is read as a table. Excluded
  is not free: declare the breakdown in the PR body whenever total added lines exceed the cap. Nor
  is it "cheap to review" — a registry where one wrong entry silently grants or denies deserves more
  care than most code. SSOT: `00_meta/patterns/pattern-git-workflow.md` §10.
- touches a public contract (API, CLI flag, exported type, alias, file path, deployed config schema)
- adds or removes a dependency
- is the first step of a multi-PR sequence

`<feature-id>` format: `^([A-Z]+[0-9]*(-[A-Z]+[0-9]*)*-[0-9]+[a-z]?(-[a-z0-9-]+)?|[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9-]+)$` (e.g., `AI-001-ollama-public`, `APP-CONFIG-015`, `ADR028-004`, `SDD-012b-guard`, `2026-05-13-cleanup`). This string is `idPattern` in dotfiles `cli/internal/spec/spec.go` verbatim; a drift test asserts every copy matches, so do not reword it.

## --- END SNIPPET ---
