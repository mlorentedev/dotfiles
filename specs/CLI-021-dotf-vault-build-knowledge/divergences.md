---
id: "CLI-021-dotf-vault-build-knowledge"
type: spec
status: draft
created: "2026-08-09"
issue: "mlorentedev/dotfiles#490"
tags: [spec, divergences]
template_version: "1.0"
---

# Twin divergences — `knowledge-crystallize.sh` vs `.ps1`

Task 1's last bullet: *"Enumerate every behaviour where `.sh` and `.ps1` disagree today. Linux is
the reference; anything `.ps1`-only must be listed before it is dropped."*

**Method.** Static read of both twins at oracle revision `9caedc1` (see
`tests/golden/crystallize/ORACLE`). `pwsh` is **not installed on this machine** — the suite's
PowerShell cases skip here — so the `.ps1` column is read, not measured. **Empirical capture of the
`.ps1` goldens is deliberately deferred to a Windows session**, batched per the standing practice of
draining Linux-doable work first. This is a stated deferral, not an omission: until that capture
happens, the Go port is characterization-proven on Linux only.

## Divergences

| # | Behaviour | `.sh` (reference) | `.ps1` | Disposition for the port |
|---|---|---|---|---|
| 1 | `[INFO]` prefix padding | `[INFO] ` — one space (`utils.sh` `printf '%b[INFO]%b %s\n'`) | `[INFO]    ` — **four** spaces | Follow `.sh`. See note below: the `.ps1` matches the `.sh`'s *fallback*, not its live path. |
| 2 | Checklist bullets | `— audit observation backlog` (em-dash) | `- audit observation backlog` (ASCII hyphen) | Follow `.sh` on Linux. The `.ps1` deviation is **deliberate and must stay** — `pattern-powershell-ascii-only`: non-ASCII without a BOM fails PSScriptAnalyzer in CI. |
| 3 | Line-limit warning | `... (limit: 150) — run /crystallize to trim` | `... (limit: $Limit) - run /crystallize to trim` | Same as #2 — same root cause. |
| 4 | BUG-062 refusal output | **Four** `[ERROR]` lines, then `return 1` | A **single** `throw` with the text joined onto one line | Follow `.sh`. The shapes are not reconcilable by accident; the port must pick one and the goldens pin the four-line form. |
| 5 | Error emitter | `log_error` from `utils.sh` | No `Write-Err` function exists; uses `throw` | Go port emits a uniform error line; `throw` semantics (terminating, formatted by the host) have no Go equivalent worth reproducing. |

## Agreements worth pinning anyway

Both twins already agree on the two behaviours most likely to be broken by a naive port, and the
golden corpus locks them:

- `Processed N / M projects (K skipped)` — and a BUG-062 refusal counts as **skipped**, never
  processed (`.sh` L286-292, `.ps1` L282-290).
- `Add-SectionBeforeHandoff` / `append_before_handoff` — HARNESS-029 parity, the invariant BUG-060
  was filed for.

## Note on divergence #1

The `.sh` defines four-space fallbacks (`log_info() { printf '[INFO]    %s\n' ... }`) used **only
when `utils.sh` is absent**, which never happens in the repo. The `.ps1` hardcodes that same
four-space form. So the twins agree on the path that never runs and disagree on the one that always
does — a drift that is invisible from either file alone.

The same block is why divergence #5 matters more than it looks: the fallback set defines
`log_info` / `log_success` / `log_warning` but **not `log_error`**, which the BUG-062 refusal path
calls. Standalone without `utils.sh`, that path would die on `command not found` under `set -e`
instead of refusing cleanly. Ticketed separately rather than fixed here — a port that improves while
translating cannot be characterization-tested.
