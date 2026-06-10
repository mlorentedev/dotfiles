---
tags: [spec, verification]
created: "2026-06-09"
---

# Verification - OPS-002-bitacora-rollout

## Evidence

- [x] AC1 (script) → `scripts/bitacora-rollout.sh` executable; shellcheck exit 0; `--check` and repo-args paths exercised. f1 passes.
- [x] AC2 (convergence) → link/secret/workflows/backfill sections present; workflows read from `.github/workflows/` (no embedded copies — the drift class that killed the old vault bootstrap doc). f2 greps pass.
- [x] AC3 (discovery) → `gh repo list … isArchived == false and .isFork == false`; only fork (`awesome-mcp-servers-1`) excluded. f3 grep passes.
- [x] AC4 (runbook) → §7 intro registers `bitacora-rollout.sh` as the multi-repo mechanism + decision note. f4 grep passes.
- [x] AC5 (live, post-merge) → full rollout executed across 20 repos; final re-run prints `all repos already rolled out (0 changes)`. Evidence below.

## Live run evidence (AC5)

- 2026-06-10, runs from `main` after PRs #315/#317/#320/#322 merged:
  - Run 1: 16/20 repos converged by direct contents-API push; 4 protected repos (kubelab, pollex, pdf-modifier-mcp, yt-metrics-cli) hit HTTP 409 → PR fallback added (#317).
  - Live findings fixed in canonical + propagated: fork PRs run without secrets → job-level skip (#320, Codex P2 on kubelab#237); stale `ci/bitacora-workflows` branch reuse → conflicting PRs → force-reset to base (#322).
  - Propagation PRs merged: kubelab#237/#238, pollex#37, yt-metrics-cli#18, pdf-modifier-mcp auto-merge.
  - Final run: `all repos already rolled out (0 changes)` — exit 0.
  - Backfill placed all open issues + PRs on the board; the OPS-003 "PRs pending review" view (Table, `is:pr is:open`, grouped by Repository) was created over it.
- Unrelated pre-existing failure surfaced: yt-metrics-cli `Release` (release-please) Bad credentials — expired shared token; tracked as OPS-007 (#321, per-purpose token convention).
