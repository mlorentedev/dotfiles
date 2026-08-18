---
id: lesson-112-strangler-fig-deletion-the-parity-gate-must-cover-
type: lesson
status: active
created: "2026-06-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 112: Strangler-fig deletion: the parity gate must cover OS-specific side effects, and a "different-by-design" Go path can still be parity (CLI-020)

**Context**: First real `.ps1` deletion of the ADR-020/021 CLI convergence — repoint Windows `project-init` to `dotf init` and delete the 3 init `.ps1`. The spec gated the deletion on proving `dotf init` is at parity on Windows.

**Problem**: A naive check ("does it scaffold a repo?") would have missed two things. (1) The `.ps1` resolved `VAULT_PATH` via a hardcoded `~/Projects/knowledge` fallback; `dotf init` resolves through the ADR-025 seam — *different code, must verify it resolves on Windows*. (2) The `.ps1` eagerly created the Windows memory **junction** at init time; `dotf init`'s Go `linkMemory` is **non-Windows by design** and creates nothing on Windows — which looks like a regression.

**Solution**: Verified empirically with an isolated `VAULT_PATH=$tmp dotf init <tmp> --skip-github` (throwaway vault, zero pollution): full scaffold + vault entry produced, `VAULT_PATH` honored. The junction "gap" is not one — `claude-session-start.ps1` `Ensure-MemoryJunction` recreates it every session, and a junction's only consumer *is* a Claude session, so the transient window is harmless. Outcome: pure repoint+delete, no Go change. Left `agents-spec-section.md`'s stale ref untouched (vault-SSOT, drift-tested → #461).

**Rule**: For every strangler-fig deletion, enumerate **all** behaviors of the dying twin — not just the happy path: input resolution, seeded artifacts, and **OS-specific side effects** (symlinks/junctions, registry, PATH). A Go replacement that *deliberately* omits an effect is still at parity **iff** a downstream consumer reconstructs it — find and name that consumer, don't assume. Verify in an isolated sandbox (throwaway `VAULT_PATH`/dirs), never against the live vault.
