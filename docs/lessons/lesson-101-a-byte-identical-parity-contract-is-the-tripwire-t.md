---
id: lesson-101-a-byte-identical-parity-contract-is-the-tripwire-t
type: lesson
status: active
created: "2026-06-17"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 101: A byte-identical parity contract is the tripwire that exposes a template divergence masquerading as a rename

**Context**: CLI-015 PR2 (#395/#403) extracted `dotf init`'s inlined vault-entry renderer into `cli/internal/vault` as `WriteProjectEntry`, moving three `vault-*` templates with it. The inherited plan (from the prior session's handoff) was "Full SSOT + drift": vendor the templates into the vault SSOT and drift-test them, mirroring PR1's work-SDK precedent.

**Problem**: Going to reconcile the embedded `vault-context.md` against the vault SSOT's `project-context.md` — to drift-test one against the other — revealed they were **never the same artifact**. Different token schemes (`{{repo}}`/`{{stack}}` vs `{{project_name}}`/`{{git_url}}`), different structure (the SSOT version carries the HARNESS-006 orientation contract + a `/context-refresh` patchable frontmatter block; the embedded one is an older generation). Same for `vault-memory.md` vs `agent-memory.md`; `vault-roadmap.md` had no SSOT counterpart at all. "Drift-test them" silently assumed a 1:1 source existed. The `#395` byte-identical acceptance criterion was the tripwire: making the embed equal the SSOT would *change* `dotf init`'s output (a behavior change), while vendoring the embed into the SSOT as-is would duplicate `project-context` (an SSOT violation). The divergence was invisible at build/test time — only the parity contract forced the question "are these actually the same file?".

**Solution**: Re-scope PR2 to move the templates **embed-only** (no drift guard), preserving byte-identical output, and ticket the reconciliation (#400) as a separate behavior-changing PR that owns the design decision (which generation wins). Confirmed parity empirically: the `git mv` rename shows 100% similarity, and `dotf vault project` vs `dotf init` emit an identical `00-context.md` modulo the repo name. Rejected: (a) vendor-as-is into the SSOT (duplicates `project-context` → SSOT violation); (b) adopt the SSOT generation now (breaks the byte-identical contract + busts the atomic-PR cap).

**Rule**: A "vendor + drift-test against the SSOT" plan silently assumes the embedded copy and the SSOT are one artifact — a shared *name* is not a shared *lineage*. Diff them before adopting the plan. A byte-identical / behavior-preserving acceptance criterion is the cheapest tripwire: if making the two equal would change output, they were never one source that drifted but two generations that diverged — and reconciling them is a design decision (which generation wins), not a mechanical re-vendor. Defer it to its own PR; never smuggle a behavior change into a move. (Generalizes "verify-before-act on agent audits": re-weigh an inherited plan against the evidence the moment the evidence contradicts its premise.)
