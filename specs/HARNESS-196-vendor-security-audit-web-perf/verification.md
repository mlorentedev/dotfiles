---
tags: [spec, verification, templates]
created: "2026-11-13"
id: "HARNESS-196-vendor-security-audit-web-perf"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-196-vendor-security-audit-web-perf

## Evidence

Commit hashes are those of the branch before the squash; the spec's own commit follows them.

- [x] AC1 -> `bats tests/harness-generated-sha.bats` (3 of 3 ok) and `dotf harness check --repo-root ..` ("[check] OK: no harness drift"), run 2026-11-13 after the rebase onto main.
- [x] AC2 -> `TestTheCompiledDependencyMapMatchesTheRecords`. Red before the one-line change ("security-audit: the compiled map says [], its record's requires: says [test]"), green after.
- [x] AC3 -> `tests/skills-pipeline.bats` "HERMES-018: a vendored skill carries provenance and an attribution row". Red before ("security-audit is vendored but absent from ATTRIBUTION.md"), green after; the feature command also checks each row's licence against the record's `license:`.
- [x] AC4 -> `cmp` of each of the three files against `gh api .../contents/skills/security-audit/<file>?ref=c1c8a8c1471069fb0e188eeaff69b8e8db6564a8`: all three identical. That commit is upstream HEAD of `cloudflare/security-audit-skill` as of 2026-11-13, committed 2026-09-14.
- [x] AC5 -> inspection below; modes are 0644 and `git grep` for both script names in `cli/`, `scripts/`, `.github/` and `setup-linux.sh` finds nothing.

### Inspection of `harness/skills/security-audit/scripts/`

Both `.cjs` files read in full (773 and 872 lines) plus a grep of the second for the dangerous APIs.

- **Imports:** `fs`, `path` and `util` only, all Node built-ins. No `child_process`, no network module, no third-party package.
- **Execution of strings:** no `eval`, no `new Function`. Regular expressions come from the vendored schema and are compiled with `new RegExp` after a try/catch check.
- **Reads:** one file, the path in `argv[2]`, opened `O_RDONLY | O_NOFOLLOW | O_NONBLOCK`; symlinks, non-regular files and inputs over 5 MiB are refused; UTF-8 is decoded strictly. `validate-findings.cjs` also reads `report-schema.json` from its own directory.
- **Writes:** none. No `writeFile`, `appendFile`, `unlink`, `rename`, `mkdir`, and no `process.env` read.
- **Output:** PASS or FAIL lines, with control characters in diagnostics escaped; exit 0 or 1.
- **Bounds:** nesting depth 64, 1000 array items, 100 reported errors, so a hostile findings file cannot make it spin or flood output.
- **`report-schema.json`:** data, interpreted by the validator's own keyword subset, which rejects unsupported keywords.
- **Deploy path:** copied as files with the rest of the skill directory and never run by this repository. The one invocation is the instruction in SKILL.md for an agent to run `node scripts/validate-*.cjs <file>`.

Not vendored, on purpose: upstream's `*.test.cjs` and the reference documents.

## Test status

- `cd cli && go test -count=1 ./internal/harness/...` -> ok; `go test ./...` in `cli/` -> no failures (two packages have no test files).
- `bats` over `tests/harness-generated-sha.bats` and the 15 other files that name a changed skill or agent: 239 ok. The one failure, "roster: the live vault's roster and definitions agree", read a local vault checkout 2 commits behind `origin/master`; with `VAULT_PATH` on a worktree at `11bacfaf` it passes (2 of 2). CI has no vault and skips it.
- `go vet ./...`, `GOOS=windows go vet ./internal/harness/` and `golangci-lint run ./internal/harness/...` -> clean.
- No regressions in the existing suite: yes.

## Decisions made during implementation

- The vault was corrected before the render rather than the record being hand-edited here: a rendered record edited by hand fails `generated_sha` and is overwritten by the next refresh.
- `ATTRIBUTION.md` records `web-perf` as Apache-2.0, the licence of `cloudflare/skills`; the vault record had said MIT and was fixed (#3470).
- `ATTRIBUTION.md` says it vendors only `SKILL.md`; a sentence now names `security-audit` as the exception, because its `scripts/` are vendored too.
- `dotf harness refresh` skips a vault checkout without an upstream ("not confirmed current"), which a detached worktree is. The refresh was run from a scratch worktree on a branch tracking `origin/master`, removed afterwards.

## Post-review dispositions

The independent review (`review.md`, PASS, `nan/glm5.3-flash`, reviewed `08652a12`) raised three Minors. The contract files stay as reviewed.

- **Minor 1, web-perf body names MIT: applied.** The vault source's Origin line now says Apache-2.0 (vault commit `f95f7ee4`) and the record is re-rendered from it.
- **Minor 2, the ATTRIBUTION vendor date: applied.** The sentence now says the two new rows were read on 2026-11-13.
- **Minor 3, "13 reference documents" against 14: declined.** The count sits in `proposal.md`, which a PASS closes; editing it would stale the review's digests. Nothing is vendored from those documents either way, so no acceptance criterion reads on the number.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: the refresh refusal on an unconfirmed vault is already printed by the tool with its reason, and the rest is specific to this change
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: adopting two vendored skills follows the existing vendoring model (ATTRIBUTION.md, ADR-013 generate-and-commit); nothing new is decided
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: repo-specific

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-196-vendor-security-audit-web-perf/` -> `specs/archive/HARNESS-196-vendor-security-audit-web-perf/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
