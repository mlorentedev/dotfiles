---
tags: [spec, verification, templates]
created: "2026-11-11"
id: "HARNESS-192-draft-push-warns"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-192-draft-push-warns

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (draft warns, exit 0, both script shells and `dotf gate spec`) -> `ef7080ed` / `AC9: a draft PR at push time is warned, not refused, and CI keeps the refusal (#3364)` (the parity helper runs the dotf leg on the same event and env)
- [x] AC2 (`false`/unset refused; `skip-archive` without rationale refused on a draft) -> `ef7080ed` / `AC1: closing an issue whose active spec is not archived fails the gate`, `AC9: a PR that is not a draft is refused as before, whatever the flag says`, `AC9: a draft does not waive a skip-archive label without its rationale`
- [x] AC3 (adapter hands `SDD_PR_DRAFT`) -> `ef7080ed` / `spec-gate-prepush: a draft PR hands the gate SDD_PR_DRAFT=true (#3364)`, `…: a PR that is ready, or an answer without the field, hands the gate SDD_PR_DRAFT=false`, `…: the --jq expression picks the first PR…`, `…: a failed lookup falls through, gate still runs with no PR context` (asserts the variable is unset)
- [x] AC4 (CI keeps the refusal) -> `ef7080ed` / `AC9: dotf gate spec does not read the event's draft field, so a draft PR in CI is still refused`, `AC9: CI's adapter never hands the gate the draft flag, so the merge-time refusal stays`

## Test status

- Every bats suite that names the two scripts (`git grep -l check-spec-gate.sh spec-gate-prepush.sh -- tests`, nine files): `bats tests/check-knowledge-gate.bats tests/check-spec-gate.bats tests/guard-zsh-bash-rematch.bats tests/spec-gate-adjacency.bats tests/spec-gate-archive.bats tests/spec-gate-pr-real.bats tests/spec-gate-pr.bats tests/spec-gate-prepush-real.bats tests/spec-gate-prepush.bats` -> 177 tests, 0 failures (2026-11-12, on `ef7080ed`).
- `shellcheck scripts/check-spec-gate.sh scripts/spec-gate-prepush.sh` and `bash -n` on both: clean.
- `cd cli && go test ./internal/cmd/ ./internal/gate/ -count=1` -> ok, ok; `GOOS=windows go vet ./internal/cmd/` clean; `golangci-lint run ./internal/cmd/...` (2.12.2, the pin, cache cleaned first) 0 issues on both `GOOS`.
- Red first: the five new archive tests and three prepush tests were run before the implementation; AC9 draft-warn, the `--jq` test and the two adapter tests failed, the three pins passed (they assert the pre-existing refusal).
- Manual smoke test: none beyond the tests; the parity leg runs the real `dotf gate spec` binary on a real repository fixture.
- No regressions in existing test suite: yes.
- Production lines: 55 (insertions plus deletions) across the two scripts and `gate_spec_rules.go`, which is why this spec exists (lesson 585).

## Decisions made during implementation

- The waiver rides on an environment variable, not on the event's `draft` field: CI's adapters never set the variable, so the merge-time refusal cannot weaken, and a PR marked ready without a push cannot carry a stale green.
- The Go twin prints the same text from one function so `tests/lib/spec-gate-parity.bash` (CI-018) keeps holding both to the same output; the parity helper passes the environment through, so the draft case runs on both legs without a change to the helper.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-590-a-merge-time-invariant-checked-at-push-time-blocks-the-fix-it-needs.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: one exit added to an existing gate; BUG-061 and CI-018 already hold the two decisions it sits between
- [x] New pattern candidate for `00_meta/patterns/`? no: the gate's two-moment shape is specific to this repository's SDD tooling

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-192-draft-push-warns/` -> `specs/archive/HARNESS-192-draft-push-warns/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
