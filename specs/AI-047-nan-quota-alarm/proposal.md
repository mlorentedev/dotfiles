---
id: "AI-047-nan-quota-alarm"
type: spec
status: draft # draft | implementing | verifying | archived
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

- **Owner decision: where the check lives.**
  - **(A) `dotf` subcommand plus a `dotf doctor` check. Recommended.** It is agent-agnostic and runs on every machine and in CI, the declared table lives in the repo next to `model-map.json`, and it is testable in Go against a recorded `/v1/usage` fixture. hermes's cron then calls it, or its digest drops the quota section.
  - (B) Rework hermes's `budget-report.sh` to read `/v1/usage`. It reuses the existing Telegram delivery, but it only runs on hermes, keeps the logic in shell in the vault, and does not reach `dotf doctor`.
  - (A) and (B) are not exclusive: A is the source, and B can become a thin caller of it.
- **Whose usage the endpoint reports.** Per API key, or per member? If hermes and the workstation use different keys, one reading covers one key. Measure this by comparing the endpoint's totals with a known burst, before relying on it.
- **Quota period boundaries.** The docs say "month"; `glm5.3` says "period", and premium adds a rolling 4-hour window. The table records the period per model. The first cut handles calendar months only and says so for `glm5.3`.
- **Declared limits go stale.** The table carries a `checked` date, and doctor WARNs when it is older than 90 days. It fails soft (SKIP, not FAIL) when NaN is unreachable, so an outage does not read as a quota breach.
- **Rate limit.** One call per doctor run is well under 60 RPM.

## Acceptance criteria

- [ ] AC1: A Go test feeds a recorded `/v1/usage` fixture and a declared table, and gets WARN for a model at 83% of its quota, PASS for one at 10%, and nothing for an unmetered model.
- [ ] AC2: With the live endpoint, `dotf doctor` reports each metered model bound in `model-map.json` with used / quota / percent. The output is recorded in `verification.md`.
- [ ] AC3: A model id in `model-map.json` missing from `/v1/models` is a FAIL naming the id. A test mutates `rerank` back to `qwen3-rerank` and sees it fail.
- [ ] AC4: With NaN unreachable, the check SKIPs with a reason and doctor's exit code is unchanged.
- [ ] AC5: No key or header value appears in output or in the error path (a test asserts this on the error branch).
- [ ] AC6: hermes's digest no longer states a single 500M pool. It calls the new check, or its quota section is removed (vault change, same arc).

## References

- Issue: mlorentedev/dotfiles#1766 (scope comment 5851375858: the hermes digest)
- NaN catalog and limits: https://nan.builders/docs/models (read 2026-09-26)
- AI-044 (#1772), VAULT-054 (knowledge#175, `adf7b1d1`), ADR-028, ADR-005 (hermes health monitoring)
