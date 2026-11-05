---
spec: "CLI-135-init-stack-directories"
verdict: "PASS"
reviewed_sha: "ecf307deb1f73b822beef599e8353d213612e0a2"
reviewer: "nan/glm5.3-flash"
date: "2026-11-04"
---

## Adversarial review

**Scope**: CLI-135-init-stack-directories
**Sources**: specs/CLI-135-init-stack-directories/{proposal,tasks,verification}.md, features.json, review-request.json + `git diff 944e6f55b1902970c52a1a2ff1c8cbb64e1ae816...HEAD` (one branch commit: ecf307de "feat(init): let each stack declare its directories, and add a workflows stack"; `git log --first-parent --no-merges` confirms no other work in the range).

### Spec and task alignment

- The diff implements exactly the four "What" bullets: `stacks.go` is the single table (`ci` + `dirs` per stack), `ciTemplates` is folded in (deleted from `ci.go`), `scaffoldDirs` keeps only the shared directories (`specs`, `docs/**`, `.claude`), and the stack flows through `ScaffoldOpts(root, stack, dryRun)` from `orchestrator.go:53`.
- `workflows` profile: `{dirs: ["tests"], ci: ""}` — matches proposal (tests only, no src/scripts, no ci.yml; lint.yml still written for every stack by the untouched `WriteLintOpts`).
- `StackInit` gained `"workflows"` in the `none`/`""` no-op case (stack.go:44); `--stack` help now reads `go|python|node|ts|workflows|none` (init.go:103) — both as tasked.
- Default preservation verified by code read + tests: `stackOf` falls back to `codeDirs` for any value missing from the table (unknown stacks included); `"none"`, `""` and every existing stack map to `codeDirs`; `WriteCIOpts` for those values yields no ci.yml, same as the old missing-map-entry behaviour.
- No test was deleted anywhere in the range; `TestScaffoldCreatesStructureAndFiles` and `TestInitOrchestratesFullScaffold` pass unchanged (full suite green).
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec artifacts.
- Contract digests in `review-request.json` match `dotf spec digest CLI-135-init-stack-directories` exactly (all three files). `reviewed_sha` equals HEAD; the working tree outside the spec folder is clean.

### Findings

Evidence run fresh in this session: `go test ./...` (all packages ok), the three features.json commands with `-count=1 -v` (all pass, one subtest row per stack including `cobol` and `""`), `go vet ./...` and `GOOS=windows go vet ./...` clean, `gofmt -l` empty, `golangci-lint run` (pinned 2.12.2) → `0 issues.` Mutation re-check reproduced independently: workflows→codeDirs, workflows dropped from StackInit's no-op case, and orchestrator passing `"none"` — each killed the expected test; all mutations reverted, tree verified clean.

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | comment drift | `InitOptions.Stack` field comment still reads "go \| python \| node \| ts \| none" — `workflows` added everywhere else but this line, now misleading | cli/internal/initrepo/orchestrator.go:20 | UNTESTED (comment) | code (one-line comment, outside contract set) |
| Minor | REAL (pre-existing) | help drift | `dotf vault project --stack` help reads "go\|python\|node\|none" — already missing `ts` at the base commit; this change adds a further unlisted value. Vault accepts any token so behaviour is correct | cli/internal/cmd/vault.go:227; confirmed identical at 944e6f5 | UNTESTED | code (follow-up; out of this spec's declared scope) |
| Minor | THEORETICAL | docs | ADR-022:76 stack list (`{go,python,node,none}`) is now two entries stale. verification.md documents the deliberate decision to leave a dated record as written and treat the flag help as the live list — accepted as declared, surfaced so the next ADR-022 contact rewrites it | docs/adr/adr-022-dotf-init-flagship.md:76; verification.md "Decisions made" | UNTESTED | none now (decision recorded in verification.md; code/doc on future contact) |

No Blocker, no Major. Adversarial angles argued and closed: slice aliasing (`stackOf` result is copied before append in `ScaffoldOpts`, so the shared `codeDirs` backing array cannot be mutated); empty-string stack behaves as before (full dirs, no ci.yml — pinned by the `""` table row); dry-run path unchanged (no dirs written, file list only); unknown-stack CI behaviour covered by pre-existing `TestWriteCIByStack` / `TestWriteCINoneIsNoOp` semantics plus `stackOf`'s `ci: ""` default; workflows vault entry accepts the new token (plain string replacement in `vault.WriteProjectEntry`).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All four ACs hold with named tests including negative rows (unknown stack, empty stack, no ci.yml, no src/scripts); no observed defects |
| Verification       | A | features.json carries reproducible per-AC commands; all re-executed fresh in this session with matching results, mutation kills reproduced independently |
| Scope              | A | 7 code files + spec artifacts, each mapping to a proposal bullet; no unrelated changes in the range |
| Reliability        | A | No new error paths; existing degrade-to-skip and skip-if-present semantics preserved; MkdirAll idempotent |
| Maintainability    | A | One table as the single stack definition, functions well under limits, comments explain why (embed/gitignore history, ADR-022 deferral) |
| Handoff-readiness  | B | Spec artifacts complete and decisions recorded; only the PR box is unticked, deliberately documented in verification.md (HARNESS-081) |

### Verdict

PASS

### Recommended next steps

Route by set — the contract set (`proposal.md`, `tasks.md`, `features.json`) is closed by this verdict; all three are outside it:

- (code, this PR or follow-up) Update the `InitOptions.Stack` comment in `orchestrator.go` to include `workflows` — a one-line, non-contract edit that does not invalidate this review.
- (follow-up ticket) Refresh the stale `--stack` enumerations in `cli/internal/cmd/vault.go:227` (pre-existing: missing `ts` and now `workflows`) and, on next contact, ADR-022:76.
- (verification.md) Disposition the two Minor findings above (applied / ticketed / declined with reason) so the archive record carries them.

**Completion:** (a) verdict **PASS**; (b) `dotf spec archive` is **advisable** in the current state — no blockers or majors, rubric all B or above; the archive gate's own remaining pre-flight (PR opened, board state) is outside this review's scope; (c) n/a — no FAIL.
