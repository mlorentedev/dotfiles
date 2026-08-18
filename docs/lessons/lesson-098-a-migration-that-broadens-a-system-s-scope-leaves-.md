---
id: lesson-098-a-migration-that-broadens-a-system-s-scope-leaves-
type: lesson
status: active
created: "2026-06-16"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 098: A migration that *broadens* a system's scope leaves single-repo assumptions hardcoded in the tools built against the old shape

**Context**: The bitácora began life as a per-repo idea and became, under ADR-018, one cross-repo GitHub Project spanning many repos (kubelab, knowledge, dotfiles, …). `dotf spec init --issue N` was written when "the bitácora" still effectively meant "the dotfiles repo": it ran the work-gate (`gh issue view N`) against the *current* repo's default, and it hardcoded the scaffolded frontmatter prefix to `issue: "dotfiles#N"`.

**Problem**: The scope-broadening migration (one repo -> a multi-repo Project) silently invalidated two assumptions baked into a tool built against the old shape. Scaffolding a kubelab spec gated by `knowledge#104` broke twice: the gate checked issue 104 in the *wrong* repo (kubelab's, where it is a different / closed issue), and the frontmatter recorded the wrong `dotfiles#104`. Neither was visible at build or test time — they manifest only when the gated issue actually lives in a different repo than the one you are scaffolding in, a case the original single-repo world made impossible. The narrow assumption stayed *correct for the common same-repo gate* and failed only at the edges the broadening newly allowed.

**Solution**: Resolve the host repo explicitly — `--bitacora-repo owner/repo` -> `$DOTF_BITACORA_REPO` -> the current repo's `git remote origin` slug — and thread that one resolved slug through BOTH the gate (`gh issue view --repo <slug>`) and the frontmatter (full `owner/repo#N`, never a bare `dotfiles#`). An unresolvable repo errors (pointing at `--bitacora-repo`) rather than fabricating a `#N`; the `[INFO] Work-gate OK` line names `owner/repo#N` so a wrong-repo gate is visible, not silent. Rejected: defaulting to a fixed `mlorentedev/knowledge` — that swaps one hardcode for another and breaks the common same-repo gate. Regression guards in `spec_test.go` + `cmd/spec_test.go` pin all three precedence paths (HARNESS-023, PR #393).

**Rule**: A migration that *broadens* a domain object's scope (single -> multi, local -> distributed, per-repo -> cross-repo) is more dangerous than one that merely moves it, precisely because the old narrow assumption stays valid for the common case and fails only at the edges the broadening newly permits — so tests written in the old world cannot see the gap. When you generalize a system, grep the tools built against its old shape for the now-too-narrow assumption (a hardcoded repo, a single-tenant ID, an implicit "there is only one") and re-derive it from the broadened source. Sibling of the incomplete-migration class (a rename that leaves callers stale): both are "the migration moved the world but not everything that still references it."
