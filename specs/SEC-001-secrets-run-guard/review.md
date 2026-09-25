---
spec: "SEC-001-secrets-run-guard"
verdict: "PASS"
reviewed_sha: "5897b829920db4d46e473af504b0d472206f2040"
reviewer: "nan/mimo-v2.5"
date: "2026-09-25"
---

## Adversarial review

**Scope**: SEC-001-secrets-run-guard (full change from base `0105662daf0c04c77f43adc9b02f588604b148c6`)
**Sources**: `specs/SEC-001-secrets-run-guard/{proposal,tasks,verification,features}.json`, `git diff 0105662daf0c04c77f43adc9b02f588604b148c6...HEAD`

### Spec and task alignment

All 10 acceptance criteria are claimed complete. I verified each against the diff and the test suite:

- **AC1–AC5** (introspection guard + unit tests): `assertSafeChildCommand` in `secrets.go`, 62 table-driven subtests in `TestAssertSafeChildCommand`, plus `TestRunChildPTY_HonoursTheIntrospectionGuard`, `TestSecretsRun_RefusesBeforeResolvingSecrets`, and the differential `TestSnippetGuard_RefusesEveryShapeTheRealShellRuns` (18 shapes × real shells). All pass.
- **AC6** (Claude deny list + openrouter): `ai/claude/settings.json` denies `env`, `printenv`, `export -p`, `sudo`, `git clean`, `secrets show`, pipe-to-shell; `ai/pi/models.json` registers `openrouter`. Verified by `features.json` f6 (jq exits 0).
- **AC7** (redactWriter): `TestRedactWriter_*` (6 named tests including tiny-chunk and split-across-writes), `TestRunChildPTY_RedactsASecretSplitAcrossWrites`. All pass.
- **AC8** (lesson): `docs/lessons/lesson-261-…` exists and is non-empty.
- **AC9** (secrets show solutions 1-2-3 + agent refusal): `TestSecretsShow_RejectsAgentSession`, `_TTYMasking`, `_RevealFlag`, `_ClipFlag`; `TestDetectAgentSession_EachMarkerIsSufficient`; `TestAgentSessionMarkers_CoverEveryHarness` (independently holds the harness→marker table against `harness/model-map.json`). All pass.
- **AC10** (deny list hardening): Same `features.json` f6 verification. Template-scoped.

No ticking-boxes-without-evidence: every `[x]` in `tasks.md` has corresponding diff and test coverage. The five review rounds' findings are all dispositioned in `verification.md` with specific commits and test names.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Minor | THEORETICAL | guard/setsCFlag | `setsCFlag` treats any `-`/`+` prefixed arg containing the letter `c` as setting the shell's c flag. Non-shell options like `--color`, `--noclobber`, `--rcfile` trigger the flag, causing all subsequent args to be inspected. This over-blocks legitimate commands (e.g. `bash --color=auto script.sh` inspects `script.sh` for introspection words). Accepted as the cost of failing closed (rounds 3+4 decision). | code read of `setsCFlag` | `TestAssertSafeChildCommand/c_inside_a_cluster`, `TestSnippetGuard_RefusesEveryShapeTheRealShellRuns/bash_-c_--_X_` | tests (behavior is by design) |
| Minor | THEORETICAL | guard/snippetIntrospection | `snippetIntrospection` refuses `bash -c 'set -x'` and `bash -c 'echo env'` even though `set -x` enables tracing (not env dump) and `echo env` prints the literal string. The guard matches whole-word `set` and `env` without parsing arguments. Accepted as over-block under the tripwire model (round 1, F4, declined). | code read of `introspectionWords` and `snippetIntrospection` | `TestAssertSafeChildCommand/bash -c set`, `TestAssertSafeChildCommand/allowed echo env word` | tests (over-block is by design) |

Both findings are THEORETICAL — documented, tested, and accepted by the owner across multiple review rounds. No REAL findings.

### Evaluator rubric

| Dimension | Grade | Rationale (one line) |
|-----------|-------|----------------------|
| Correctness        | B | All 10 ACs verified with passing tests; minor over-blocks are the accepted cost of the tripwire model |
| Verification       | A | 62+ table tests, differential shell testing against real bash/zsh/sh/dash, 3 mutation rounds (27+ mutants killed), 7 features.json verifiers, negative controls |
| Scope              | B | SEC-001 changes match proposal exactly; the large 430-file diff includes other specs' work carried along in the branch |
| Reliability        | A | Mutex on redactWriter, prefix-aware hold-back (not fixed window), Flush handles partial-secret-at-EOF, both pty and pipe paths tested, guard runs before decryption |
| Maintainability    | A | All SEC-001 functions under 40 LOC, clear naming, doc comments explain why not just what, no dead code, zero golangci-lint findings in scope |
| Handoff-readiness  | A | 5 review rounds documented with dispositions, lessons 289/290 captured, ADR-028 corrected in place, features.json has runnable verifiers |

### Verdict
PASS

### Recommended next steps
- Archive is advisable. The archive checklist in `verification.md` has one remaining box: "Round 5 review passes" — this review satisfies it. The two other boxes (`dotf spec archive` and #1626 disposition) are post-archive actions.
- No contract-set changes required. The findings are all minor, THEORETICAL, and documented as accepted design decisions. `proposal.md`, `tasks.md`, and `features.json` do not need modification.
- Carry the two minor findings into `verification.md` as tracked dispositions (applied as design decisions, declined under tripwire model) for completeness, if desired.
