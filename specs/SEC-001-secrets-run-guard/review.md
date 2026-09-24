---
spec: "SEC-001-secrets-run-guard"
verdict: "FAIL"
reviewed_sha: "a720b9d2b40569b73c96aadc5ceadbd8a41d8f7f"
reviewer: "nan/glm5.3-flash"
date: "2026-09-23"
---

## Adversarial review

**Scope**: SEC-001-secrets-run-guard — full diff `0105662daf0c04c77f43adc9b02f588604b148c6...HEAD` (single commit `a720b9d`, PR #1459).
**Sources**: `specs/SEC-001-secrets-run-guard/{proposal,tasks,verification}.md`, `features.json`, `cli/internal/cmd/secrets{,_test}.go`, `cli/internal/cmd/secrets_registry_test.go`, `ai/claude/settings.json`, `ai/pi/models.json`, `AGENTS.md`, `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`, `docs/lessons/lesson-261-*`.

All claims below were verified by running things in this session (per lesson-261 doctrine, only synthetic data — the live registry was never touched): `go build ./...`, `go vet ./...`, `go test ./...` (green except my own scratch probes, which were deleted afterwards), plus temporary adversarial unit probes against `assertSafeChildCommand` and `redactWriter`.

### Spec and task alignment

- AC1/AC2 (refuse `env`, `printenv`, `/usr/bin/env`, `export` as direct child): implemented in `assertSafeChildCommand`; covered by named tests in `TestAssertSafeChildCommand` (`bare env`, `path to env`, `bare printenv`, `bare export`). Verified.
- AC3 (shell wrappers detected): partially implemented. `sh -c 'env'`, quote/escape forms, bundled/interleaved `-c` flags are covered by named tests and pass. **Demonstrated bypasses exist** (finding F2).
- AC4 (legitimate tools unhindered): holds for direct binaries (named tests: `allowed tool`, `allowed python`, `allowed dotf review`); **does not hold inside shell snippets** — legitimate `bash -c 'echo env'` / `echo $ENV` / `grep env file.txt` are refused (finding F4, unit-reproduced).
- AC5 (table-driven unit tests): present, table-driven, good flag-coverage matrix. Missing negative-path coverage for chunked redaction and in-snippet path/redirect forms (findings F1/F2 are UNTESTED).
- AC6 (Claude deny list + Pi models catalog): both updated. OpenRouter provider registered with `${OPENROUTER_API_KEY}`. Not runtime-verified (out of unit reach); deny-pattern globs flagged as a Question (F9).
- AC7 (redactWriter): implemented and correct for whole-chunk writes (boundary straddling via `tail` holdback is handled: verified — an 11-char secret split as `"value="` + secret was fully redacted). **Fails for chunked writes smaller than the longest secret** (finding F1, full plaintext leak reproduced).
- AC8 (lesson 261): exists, well-written, root causes and doctrine recorded.
- AC9 (show 1-2-3): implemented with four named tests (`TestSecretsShow_RejectsAgentSession`, `TestSecretsShow_TTYMasking`, `TestSecretsShow_RevealFlag`, `TestSecretsShow_ClipFlag`); order of checks is correct (clip → agent refusal → TTY mask → print; `--reveal` bypasses only TTY mask, never agent refusal). Verified.
- AC10 (deny-list hardening): entries added; glob syntax plausibility flagged as Question (F9).
- **Contract drift**: `verification.md` is an unfilled template (all boxes `[ ]`, placeholders like `commit <hash>` intact) while `tasks.md` ticks `[x] verification.md filled in` and `[x] Every acceptance criterion from proposal.md is covered by tests`; `features.json` still carries the scaffold `verification: "echo 'not implemented' && exit 1"` with `state: pending` (finding F3).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Blocker | REAL | redaction (AC7) | `redactWriter` leaks injected secrets that arrive split across writes smaller than the longest secret. The holdback only engages when `len(data) >= maxSecretLen` (`secrets.go` `Write`); any shorter chunk is written verbatim after `ReplaceAll`, so a secret straddling chunk boundaries escapes (partially or entirely). Reproduced: 6-char secret emitted byte-by-byte produced full plaintext `ABCDEF` in the sink. This defeats the feature's stated purpose ("scrubs **any** injected secret value") under realistic small-chunk/unbuffered writers, and it is the second defense layer that lesson-261 relies on. | Scratch probe this session (deleted after run): full suite green, probe FAILed with `LEAK: secret escaped when emitted byte-by-byte: "ABCDEF"`. | UNTESTED — no shipped test writes in chunks; `TestRedactWriter_RedactsInjectedSecrets` uses one whole-chunk write | code (always retain `min(len(data), maxSecretLen-1)` bytes when pairs non-empty) + chunked-write regression test |
| Major | REAL | guard bypass (AC3) | Shell-snippet inspection misses two demonstrated forms: `sh -c '/usr/bin/env'` (absolute path — `/` is not in the boundary class) and `sh -c 'env>x'` (redirect with no space — `>` is not in the end-separator class). Both returned `blocked=false` in unit probes. AC3 says wrappers "are detected and refused"; these are not. | Scratch probe this session: `abs path env in sh -c blocked=false`, `env redirect no space blocked=false` (synthetic argv, no execution). | UNTESTED — `TestAssertSafeChildCommand` covers path form only as direct argv, never inside `-c` | code (e.g. tokenize `cleanCmd` on non-alphanumerics and match whole tokens, which also covers `env>x`, `(env)`, `\nenv`) + tests |
| Major | REAL | spec artifacts | Contract not fulfilled at bookkeeping level: `verification.md` is an unfilled template; `tasks.md` falsely ticks `[x] verification.md filled in` and `[x] Every acceptance criterion from proposal.md is covered by tests`; `features.json` still contains the scaffold `"verification": "echo 'not implemented' && exit 1"` / `state: pending`. A `[x]` without evidence is a process finding on its own, and two of the false ticks sit in the contract set. | Direct read of `specs/SEC-001-secrets-run-guard/{tasks.md,verification.md,features.json}` this session. | n/a (artifact finding) | spec artifacts — `verification.md` freely; `tasks.md`/`features.json` are contract-set → fix in next round, re-review follows (already forced by F1) |
| Minor | REAL | over-blocking (AC4) | Inside shell snippets the regex over-blocks legitimate commands: `bash -c 'echo env'`, `bash -c 'echo $ENV'`, `bash -c 'grep env file.txt'` all refused (unit-reproduced). Fails-closed is the right bias, but AC4's "legitimate tools run unhindered" is not honored for wrappers, and the AC4 tests only exercise direct binaries. | Scratch probe this session: all three `blocked=true`. | UNTESTED (no test documents the over-block as accepted behavior) | tests (pin accepted over-blocks) or code (token-match instead of boundary regex) |
| Minor | THEORETICAL | guard scope (AC3) | Shells outside the wrapper list bypass inspection: `fish -c 'env'`, `pwsh -c 'printenv'` not blocked (unit-reproduced at function level; relevance depends on those shells existing on the host). The proposal's out-of-scope covers "arbitrary binary payload content" but does not clearly waive other *shells*. | Scratch probe: both `blocked=false`. | UNTESTED | code (extend shell list) or spec (next round: declare non-POSIX shells out of scope explicitly) |
| Minor | THEORETICAL | guard scope | Indirection bypass: `bash -c 'e=en; ${e}v'` is not blocked. Arguably within the declared out-of-scope (payload-content inspection), but the proposal's wording covers "arbitrary binary payload", not shell-variable indirection inside the `-c` argument itself. Layer-2 redaction is the designed backstop — which is why F1 matters so much. | Scratch probe: `blocked=false`. | UNTESTED | spec (clarify scope boundary next round); code change optional |
| Minor | REAL | maintainability | `buildChildEnv` is production-dead after the refactor: only `secrets_registry_test.go:139,181` call it; the live path inlines `append(stripBackendAuth(os.Environ()), injected...)` in `RunE`. Its docstring (backend-auth stripping rationale) now describes logic that lives elsewhere. | `grep -rn buildChildEnv --include='*.go'` this session. | n/a | code (inline into tests or delete + move comment) |
| Minor | REAL | docs drift | ADR-028 amendment lists agent-session env vars `CLAUDE_CODE, ANTIGRAVITY_AGENT, AGENT_SESSION` but the code also checks `ANTIGRAVITY_CLI` (`isAgentSession`, `secrets.go`). Doc omits one of the four tripwires. | Code read + ADR text this session. | `TestSecretsShow_RejectsAgentSession` covers the mechanism, not the doc | spec artifacts (repo `docs/adr/` — editable, outside contract set) |
| Question | SPECULATIVE | AC6/AC10 | Deny entries like `Bash(*dotf secrets show*:*)` and `Bash(curl *|*sh*)` mix mid-pattern globs with the `:*` suffix form. Claude Code docs indicate Bash rules support glob `*` patterns, but the mixed form should be verified against the live permission matcher once; a never-matching deny entry is dead security configuration. | Not verified at runtime this session (no Claude Code matcher available). | UNTESTED | tests or manual verification; no code change indicated yet |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | Ten of eleven ACs verified on happy paths, but the core redaction criterion (AC7) has a demonstrated full-leak failure mode and AC3 has demonstrated bypasses. |
| Verification       | C | `verification.md` is an unfilled template; evidence exists only as reproducible tests/commits, not in the artifact that must prove the criteria. |
| Scope              | B | Diff matches the proposal's What/AC list; one undocumented side change (`Bash(source:*)` removed from allow) otherwise clean. |
| Reliability        | B | Guard fails closed; flush errors are silently discarded (`_ = stdout.Flush()`); redactor degradation is silent by design. |
| Maintainability    | B | Small functions, table-driven tests, clear comments; production-dead `buildChildEnv` and duplicated env-assembly logic. |
| Handoff-readiness  | C | Lesson 261 and ADR amendment are strong, but verification.md/features.json left in scaffold state contradict the ticked tasks. |

### Verdict
FAIL

### Recommended next steps

Minimum set that would flip this to PASS (the first three are required; the re-review itself is forced because F1/F2 need code+tests and F3 touches the contract set):

1. **F1 (code + tests)** — make the holdback unconditional when pairs are non-empty: retain `min(len(data), maxSecretLen-1)` bytes on every write instead of gating on `len(data) >= maxSecretLen`. Add a regression test that feeds a ≥6-char secret in chunks of 1–3 bytes and asserts nothing longer than the redaction marker escapes (my scratch probes are a ready-made template: split-write and byte-by-byte cases).
2. **F2 (code + tests)** — close the two demonstrated wrapper bypasses: normalize the snippet by tokenizing on non-alphanumeric characters (covers `/usr/bin/env`, `env>x`, `(env)`, leading newline) and match whole tokens; add table rows for `sh -c '/usr/bin/env'` and `sh -c 'env>x'`.
3. **F3 (spec artifacts)** — fill `verification.md` with per-AC evidence (commit hashes + named tests, as AC-wise they mostly exist), correct the two false `[x]` ticks in `tasks.md`, and replace the `features.json` placeholder. Note for the implementer: `tasks.md`/`features.json` are contract-set, so these edits happen in the round that also lands F1/F2 and are re-reviewed together.
4. **F4–F7 (disposition in `verification.md` or a follow-up ticket)** — pin or refine the wrapper over-blocks (F4), declare or extend the shell list (F5), clarify the indirection scope boundary in the next proposal revision (F6), and delete/inline production-dead `buildChildEnv` (F7).
5. **F8 (docs)** — add `ANTIGRAVITY_CLI` to the ADR-028 amendment's env-var list; `docs/adr/` is outside the staleness set, so this can land any time.
6. **F9 (question)** — empirically verify the mixed glob+`:*` deny patterns against a live Claude Code matcher once; demote to a finding only if a pattern proves inert.
