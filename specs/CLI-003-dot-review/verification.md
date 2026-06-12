---
tags: [spec, verification, templates]
created: "2026-05-13"
---

# Verification - CLI-003-dot-review

## Evidence

- [x] Live review returned -> `git diff HEAD -- cli/ | go run ./cmd/dot review --provider nan` produced a markdown review (NaN, deepseek-v4-flash); full 12KB staged diff reviewed live via `--provider openrouter` in ~10s. Windows leg -> same code path via `httptest` mock in CI `test (windows-latest)`.
- [x] Unit tests -> `TestReviewErrorContract` (empty stdin, missing NAN_API_KEY/NAN_BASE_URL/OPENROUTER_API_KEY naming the variable, unknown provider, max-bytes exceeded), `TestReviewHappyPath` (auth header + default model asserted), `TestReviewModelOverride`, `TestReviewNoChoices`, `TestReviewHTTPError` (5xx), `TestReviewTimeout` — green on Linux local; both OSes in CI.
- [x] `dot review --help` documents providers, default models, required env vars, exit codes and the privacy note.
- [x] Timeout default (120s, `--timeout`) + bounded `max_tokens` (4096 const `reviewMaxTokens`) -> `TestReviewTimeout`, `TestReviewMaxTokensBounded`.
- [ ] Lint + goreleaser snapshot stay green -> CI on the PR (pending first run; gofmt/vet clean locally).
- [x] Manual QA pass -> see QA findings below; no product issues to file (one provider-side limitation documented in README).

## Test status

- Test suite: `cd cli && go test ./...` -> `ok github.com/mlorentedev/dotfiles/cli/cmd/dot` (root + 12 review subtests, 0 failures)
- Manual smoke (live APIs, Linux):
  - Small diff via NaN -> review returned, "Verdict: Clean diff."
  - 12KB staged diff via NaN -> request dropped mid-generation (gateway closes long non-streaming responses; reproduced at 120s client timeout and at 300s with a TCP read timeout ~168s) — provider-side, documented as Known limitation in `cli/README.md`.
  - Same 12KB diff via OpenRouter -> full review in ~10s; the cross-model reviewer flagged the privacy concern (already documented) and asked for a malformed-response test -> `TestReviewNoChoices` added in response.
- No regressions: scaffold tests untouched and green.

## Decisions made during implementation

- **Default provider stays `nan`** despite the large-diff limitation: fixed-cost subscription vs metered OpenRouter credit; `--provider openrouter` is the documented escape hatch for large diffs.
- **Default `--timeout` kept at 120s**: raising it does not help (the gateway, not the client, drops the connection); evidence recorded above.
- **No confirmation prompt for privacy** (suggested by the cross-model review): the CLI must work in pipes; an interactive prompt would break `git diff | dot review`. The privacy note lives in `--help` and README instead.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? yes — "Non-streaming chat endpoints behind CDN gateways drop long generations: timeout flags cannot fix a server-side cut; test providers with realistic payload sizes during QA, not hello-world ones"
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — provider/default decisions recorded here and in proposal
- [ ] New pattern candidate for `00_meta/patterns/`? no — single-tool concern so far

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-003-dot-review/` -> `specs/archive/CLI-003-dot-review/`
- [ ] Backlog entry: close issue #337 (bitácora auto-moves to Done)
- [ ] Promotions above executed (if any)
