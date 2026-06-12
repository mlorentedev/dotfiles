---
id: "CLI-003-dot-review"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-06-12"
issue: "dotfiles#337"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-003-dot-review

> **Naming**: file lives at `<repo>/specs/CLI-003-dot-review/proposal.md`. `CLI-003-dot-review` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #337: CLI-003: `dot review` — cross-model code review (NaN/OpenRouter) -->

Today the only AI code review available is from the same model family that writes the code (Claude reviewing Claude) — correlated blind spots. `dot review` provides a decorrelated second opinion (deepseek/qwen via NaN, or OpenRouter) invokable by any agent or by hand: a portable CLI, deliberately NOT a Claude hook (hooks do not port across agents — the trigger that produced ADR-020). It is also the greenfield first subcommand that proves the `dot` CLI pattern end-to-end before any shell-twin migration risks regressions.

## What

1. New subcommand `dot review`: reads a unified diff from **stdin** (`git diff main...HEAD | dot review`), sends it to an OpenAI-compatible chat-completions endpoint, prints the review (markdown) to stdout.
2. Providers: `--provider nan` (default; consumes existing `NAN_BASE_URL` + `NAN_API_KEY`) and `--provider openrouter` (`OPENROUTER_API_KEY`, fixed base `https://openrouter.ai/api/v1`). `--model` overrides the per-provider default: `deepseek-v4-flash` on NaN (grounded in the repo's NaN catalog), an equivalent low-cost deepseek on OpenRouter. Defaults visible in `--help`.
3. Error contract: empty stdin -> non-zero exit with usage hint; missing env var -> clear error naming the variable; HTTP/timeout -> error to stderr, never a silent partial review.
4. Zero new dependencies: stdlib `net/http` — one endpoint, one JSON payload; an OpenAI SDK is not justified (Decision Hierarchy: stdlib first).

## Out of scope

- **No GitHub/PR integration** (inline comments): the review goes to stdout; piping it to `gh pr comment` is the caller's job, not the CLI's.
- **No streaming** of the response — one complete answer; add only if latency actually hurts.
- **No provider fallback chains** — OpenRouter's native model-array already gives fallback for free (per issue #337); no custom retry-across-providers.
- **No configurable prompt** (no template file, no Viper) — one curated built-in review prompt; configurability when a real need appears.
- **No twin migration, no other subcommands** — this PR only proves the pattern.

## Risks / open questions

- **R1 — Large diff vs model context window.** RESOLVED (must hold before code): fail above a size threshold with non-zero exit and a hint (`git diff --stat`, split the range) — never truncate silently; a review of a truncated diff is a silently incomplete review. Flag `--max-bytes`, sensible default (~200KB).
- **R2 — CI cannot test the live API** (no secrets on PR CI, provider rate limits). RESOLVED: unit tests run against an in-process `httptest` mock of the OpenAI-compatible endpoint on both OSes; the live-API smoke is manual QA recorded in `verification.md` (standing per-CLI-PR QA task, CLI-001 R5 convention).
- **R3 — Privacy**: the feature's purpose is sending diffs to a third-party API; `cli/README.md` states it explicitly (think before piping client-repo diffs).
- **R4 — Windows stdin behavior** (CRLF/encoding): covered by table-driven tests + the `windows-latest` CI job exercising piped stdin.

## Acceptance criteria

- [ ] `git diff main...HEAD | dot review --provider nan` returns a markdown review — live evidence on Linux (manual QA in `verification.md`); same code path verified on Windows in CI against the mock + piped-stdin smoke.
- [ ] Unit tests green on Linux and Windows (existing matrix): provider selection, env validation, stdin parsing, and the full error contract — empty stdin, missing env var (names the variable), HTTP 5xx, timeout, diff over `--max-bytes`.
- [ ] `dot review --help` documents providers, default models, required env vars and exit codes.
- [ ] HTTP client enforces a default timeout (`--timeout`, ~120s) and a bounded `max_tokens` cost guard; both covered by tests.
- [ ] Lint + goreleaser snapshot stay green (scaffold checks untouched).
- [ ] Manual QA pass on both OSes recorded in `verification.md`; issues filed for findings (CLI-series standing convention).

## References

- Work-gate: issue #337 (epic #131 — Go CLI convergence)
- Related ADR: `docs/adr/adr-020-tooling-cli-go-convergence.md` (names `dot review` as the proving PR)
- Predecessor: `specs/archive/CLI-001-dot-scaffold/` (module, CI, release pipeline)
- Env wiring: `sensitive/env-mapping.conf` (`NAN_API_KEY`, `OPENROUTER_API_KEY`), `.zshrc` (`NAN_BASE_URL`)
