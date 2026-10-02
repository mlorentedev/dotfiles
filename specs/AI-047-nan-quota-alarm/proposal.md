---
id: "AI-047-nan-quota-alarm"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-26"
issue: "mlorentedev/dotfiles#1766"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, nan, quota, doctor]
template_version: "1.0"
---

# AI-047-nan-quota-alarm

## Why

Every NaN model we route traffic to has its own monthly quota, and a spent one answers 402 until the month resets. On 2026-09-26, `qwen3.8-flash` stood at 83% of its 500M quota, and nothing had noticed: opencode's default and pi's routing were sending bulk traffic there (fixed by AI-044, #1772). The one alarm that exists is hermes's daily digest (`00_meta/agents/scripts/budget-report.sh`), and it is wrong three ways. It sums deepseek, mimo and `glm-5.2` against a single 500M pool. It never watches `qwen3.8-flash`. And it self-accounts from hermes's own `state.db`, so it cannot see opencode, pi, PR-Agent or Claude-side usage on the same member.

## What

A check that answers "which bound NaN model is near its quota" from NaN's own numbers:

- It reads `GET /v1/usage` through the secrets facade (`dotf secrets run`, ADR-028). The key never touches argv or output.
- It sums `total_tokens` per model over the current quota period. The endpoint returns per-day, per-model rows plus `totals.by_model` (measured 2026-09-26). It does not return limits.
- It compares each model against a **declared** quota table, with one entry per metered model and its source URL and date. It WARNs at a threshold (default 80%) and FAILs at 100%. An unmetered model is never flagged.
- It also checks that every NaN id named in `harness/model-map.json` appears in `/v1/models`. That is the guard for the class that bound `qwen3-rerank` for weeks while NaN answered 401.
- hermes's digest either reads the same numbers or is retired. Two alarms that disagree are worse than one.

## Out of scope

- Throttling or rerouting traffic automatically. This is an alarm; AI-044 was the reroute.
- pi's `/nan-usage` (AI-046 keeps it out: it needs a second credential).
- Billing or cost in money. NaN quotas are tokens.

## Risks / open questions

- **Owner decision: where the check lives. DECIDED 2026-09-26: (A).**
  - **(A) `dotf` subcommand plus a `dotf doctor` check. Recommended.** It is agent-agnostic and runs on every machine and in CI, the declared table lives in the repo next to `model-map.json`, and it is testable in Go against a recorded `/v1/usage` fixture. hermes's cron then calls it, or its digest drops the quota section.
  - (B) Rework hermes's `budget-report.sh` to read `/v1/usage`. It reuses the existing Telegram delivery, but it only runs on hermes, keeps the logic in shell in the vault, and does not reach `dotf doctor`.
  - (A) and (B) are not exclusive: A is the source, and B can become a thin caller of it.
- **Whose usage the endpoint reports.** Per API key, or per member? If hermes and the workstation use different keys, one reading covers one key. Measure this by comparing the endpoint's totals with a known burst, before relying on it. Still unmeasured after PR-1; `verification.md` records it as open.
- **Quota period boundaries.** The docs say "month"; `glm5.3` says "period", and premium adds a rolling 4-hour window. The table records the period per model. The first cut handles calendar months only and says so for `glm5.3`. Measured 2026-09-26: `/v1/usage` takes `start_date` and `end_date` as UTC dates and defaults to a rolling 30-day window, which does not say which period NaN meters against. The check asks for the UTC calendar month and names that as an assumption.
- **Declared limits go stale.** The table carries a `checked` date, and doctor WARNs when it is older than 90 days. It fails soft (SKIP, not FAIL) when NaN is unreachable, so an outage does not read as a quota breach.
- **Rate limit.** One call per doctor run is well under 60 RPM.

## Acceptance criteria

- [x] AC1: A Go test feeds a recorded `/v1/usage` fixture and a declared table, and gets WARN for a model at 83% of its quota, a visible report (INFO) for one at 10%, and nothing above PASS for an unmetered model.
- [x] AC2: With the live endpoint, `dotf doctor` reports each metered model with used / quota / percent, without `--verbose`. The output is recorded in `verification.md`.
- [x] AC3: A model id in `model-map.json` missing from `/v1/models` is a FAIL naming the id. A test mutates `rerank` back to `qwen3-rerank` and sees it fail.
- [x] AC4: With NaN unreachable, the check SKIPs with a reason and doctor's exit code is unchanged.
- [x] AC5: No key or header value appears in output or in the error path (a test asserts this on the error branch).
- [ ] AC6: hermes's digest no longer states a single 500M pool. It calls the new check, or its quota section is removed (vault change, same arc). Resolved 2026-09-26: the hermes-nan MicroVM carries no `dotf`, registry or age key (`80_agents/hermes-nan/context.md`), so the quota section is removed rather than wired to a check it cannot run.

- [x] AC7: The quota is the account's, not a binding's. The check watches every model the table meters and every model with usage this period, as well as the bound ones. A test with nothing binding `qwen3.8-flash` at 83% still gets its WARN. Added 2026-10-01: on 2026-09-26 `qwen3.8-flash` reached 83% while pi, not `model-map.json`, was spending it, and the bound-only check could not have seen it. An unbound model the key cannot see (`/v1/models` hides premium models by tier) spends nothing and is not reported.

## References

- Issue: mlorentedev/dotfiles#1766 (scope comment 5851375858: the hermes digest)
- NaN catalog and limits: https://nan.builders/docs/models (read 2026-09-26)
- AI-044 (#1772), VAULT-054 (knowledge#175, `adf7b1d1`), ADR-028, ADR-005 (hermes health monitoring)
