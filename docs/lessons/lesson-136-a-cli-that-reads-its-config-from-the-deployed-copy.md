---
id: lesson-136-a-cli-that-reads-its-config-from-the-deployed-copy
type: lesson
status: active
created: "2026-06-27"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 136: A CLI that reads its config from the *deployed* copy, not the checkout, silently reverts its own writes

**Context**: `dotf secrets` resolves `secrets/registry.yaml` (the mapping SSOT) to drive `show`/`run`/`migrate`/`set`. The first cut read and wrote the **deployed** copy at `~/.dotfiles/secrets/registry.yaml` — the same path setup rsyncs from the checkout on every redeploy. The first real C8 migrate (#635) never ran, so the footgun stayed latent until the `sync ci` smoke surfaced it.

**Problem**: Setup deploys by copying checkout → `~/.dotfiles`, so the deployed copy is a *derived artifact*, not a source. A `migrate` that flipped `backend: age → bw` in the deployed copy produced a write that (1) the next redeploy silently reverts and (2) never reaches git at all — a durability black hole that looks successful in the moment. Worse, it is a two-tier split-brain: `#636` fixed the registry to resolve checkout-first, but left the secret *files* (`sensitive/*.age`, `SecretsDir`) deployed-only, so the identical bug recurred for values — a token re-encrypted in the checkout was invisible to `dotf` until a redeploy, hit live during the RELEASE_TOKEN rotation (`#642`). (The same PR also caught an ADR-number collision — the new model was renumbered 029→030.)

**Solution**: Split the seam by intent. Reads prefer the checkout and fall back to the deployed copy (`env.ResolveRegistryPath` / `ResolveSensitiveDir`, "live-on-pull" with a graceful degrade); **writes** resolve the checkout only and **fail loud** when no checkout is found (`env.RepoRegistryPath`), so a mutation can never land in a derived artifact. `env.RepoDir` is the shared resolver (`DOTFILES_REPO_DIR` or a `.git` walk-up). Registry and values now share one source, so a checkout-side rotation or migrate is authoritative immediately and is committable.

**Rule**: When a tool both *reads* and *writes* a file that a deploy step copies from a source-of-truth, the deployed copy is a cache, not a store — never write to it. Resolve writes to the checkout/SSOT and fail loud if it is absent; reads may fall back to the deployed copy for ergonomics, but the read and write seams must be **separate** so a write never silently targets a derived artifact. And when you fix this for one file (the registry), audit every sibling the same deploy touches (the secret values) — a split-brain rarely has exactly one half.
