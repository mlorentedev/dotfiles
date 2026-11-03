---
tags: [spec, verification, templates]
created: "2026-10-21"
---

# Verification - TOOL-031-pr-agent-reusable-workflow

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`
- [ ] Criterion 3 -> commit `<hash>` / test `<name>`

- AC3, partly -> PR #3015, run 41232841929: `Build dotf`, the preflight, the draw, the guard probe and the final guard all ran from the PR and passed; the risk route drew Sonnet, which published. The push gate was skipped on `ready_for_review`. No weight changed in that PR, so the task's run on a weight change is still open.

- AC6, D's half (#3016) -> `cli/internal/prreview/{reviewable,filefilter}_test.go`, `cli/internal/cmd/pr_reviewable_test.go`; the cases of `tests/pr-agent-queue-skip.bats` and the matcher cases of `tests/pr-agent-config.bats` are ported there, and the bats file keeps the wiring.
  - Mutations: 48 of the verb and 5 of the workflow wiring, each applied alone, every one failing a test. The first pass left 4 alive: two test gaps closed (a failed read with a body on stdout, an unpaginated listing), one literal case added, and one equivalent mutation that showed the code was redundant, now deleted.
  - Live, PR #3037: run 41376549715 built from head and skipped the draft ("PR #3037 is draft now"), with the draw skipped; run 41376625169 on `ready_for_review` built, decided, drew Sonnet and published, with no incomplete-list warning, so the real lists read at the audited commit. The empty-diff skip has not run on a real PR yet: its evidence is the Go tests over upstream's real files.
  - Differential, one-off, before the Python left: `translateGlob` against Python 3.13's `fnmatch.translate` on 240,000 random glob and path pairs. 239,844 agreed, none differed, 156 are globs RE2 refuses, which are dropped and only narrow.

### `dotf pr reviewable`: interior globstars (#3036)

Measured on 2026-11-01 against upstream's own `translate_globs_to_regexes` at the audited commit, Python 3.13.16, with `pr_agent`'s two imports stubbed out of `file_filter.py`.

- Differential: 20,000 random glob lists over `**`, `*`, sets, and unclosed and stray brackets, four paths each. 80,000 cases, 6,608 of them ignored upstream, 0 differences in either direction.
- The table in `TestIgnorePatterns_ExpandsInteriorGlobstars` and the counts in `TestGlobSources_*` were each read from upstream, not derived from the port.
- The bounds are ported, not relaxed: past six globstars, 256 runes or 256 variants PR-Agent keeps a file, so dropping it here would skip a real diff.
- Mutations: 20 on the new code in the first pass, of which 4 exposed code with no effect (`lastClose`, the `**/` stride, and the cap's seen check and `break`), deleted rather than tested; the remaining survivors were closed by tests, and the 11 rerun on the final code were all killed.
- Python 3.13's `fnmatch` compresses a run of stars, so `translateGlob` does too: it changes no match, only which forms count against the quota.

### `dotf pr declare-unreviewed`

- AC4 and AC6, D's half -> `cli/internal/prreview/declare_test.go`, `cli/internal/cmd/pr_declare_unreviewed_test.go`, and `TestReviewable_DiffSaysOnlyWhatWasRead`. kubelab's four step tests map to `TestDeclare_NeverHoldsTheEscapeAsALiteral`, `TestRationale_PointsAtNoDeadEnd` and the `empty` and `reviewable` branches of `TestPrDeclareUnreviewed_*`.
- Mutations: 19 on the decisive lines, each applied alone. 18 failed a test; the survivor, the blank line under a filled heading, is now asserted.
- Live, on the verb's own PR #3044, built from its head: `reviewable` answered `diff=reviewable`; `--diff reviewable` with no label wrote nothing; `--diff empty` wrote the section and `merged-unreviewed`; a second run left one section; `--diff reviewable` removed the label and warned that the section stays. The body was then restored. Against hive's real registry, `--diff empty` exited 1 with `harness/review-attestation.json at master declares no escape.label/escape.section`.
- Not run: the verb under a `GITHUB_TOKEN` in a real workflow, which waits for ci-workflows `v0.2.0`, and a real PR of only ignored files.

## Test status

- #3016, `cli/`: `go build ./... && go vet ./... && go test ./...` -> exit 0, no failing package; `GOOS=windows go vet ./...` -> exit 0; pinned `golangci-lint` 2.12.2, Linux and `GOOS=windows` -> `0 issues.`
- #3016, shell: `bats` on `tests/pr-agent-config.bats`, `tests/stub-real-pairing.bats` and every suite naming these files -> 110 tests, none failing.
- Whole spec: `<command> -> <output / coverage %>`
- Manual smoke test: what was exercised, what was observed
- No regressions in existing test suite: yes / no (if no, document)

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- `dotf pr declare-unreviewed` takes `--diff` from a new `diff=` output of `dotf pr reviewable`, not its boolean. `reviewable=false` also means a draft, a closed PR or a forfeited branch, which kubelab's `open` step filtered before its declare step, and `reviewable=true` also means a read that failed. The advisor proposed `skip_reason` with a `--clear` flag; that covers the first and not the second, and leaves the choice of mode in the workflow.
- `diff=reviewable` needs every filter source read. A lost source narrows the filter, which keeps files the whole one might drop, so that run says `unread`, and a declared label survives an outage.
- The section check follows `check-review-attestation.sh`, not kubelab's `grep -qxF`: CRLF and trailing blanks do not count, and a heading with nothing under it is filled in place, because the gate reads the first heading and refuses an empty one. kubelab's step would report "already declared" and leave the gate red.
- Departs from kubelab's test on one point. It asserts `lesson-376` and `#1527` in the rationale; those are kubelab's, and would link to something else in another repository. The rationale keeps the method, decrypt both sides and diff the key names, with no reference.
- `dotf pr preflight --pool` orders the chain by descending weight, pool order breaking a tie. Document order would have made deepseek the primary, and the pool's own comment says its order is provenance, not precedence. A `forge_rank` field exists but no member sets it, and `harness/**` was out of scope for this change, so weight is what reproduces the declared chain.
- A pool with no NaN member exits 2, not 1. Exit 1 is the workflow's signal that NaN is down, which moves the review to the other provider, so a bad pool must not look like an outage.
- With a pool as the source, a refused model's remedy names the pool. The old text sent the reader to dotfiles' `.pr_agent.toml` and workflow, which a consumer does not have.
- The chain the embedded pool yields, pinned by `TestPrPreflight_PoolYieldsTheDeclaredChainInOrder`:

  ```
  model=openai/mimo-v2.6-flash
  fallbacks=["openai/deepseek-v4-flash","openai/glm5.3-flash"]
  ```

  dotfiles' own workflow keeps its literals, so `pr-agent-config.bats` still compares them with the pool.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [ ] Lesson for the repo's `docs/lessons/`? <yes: path / no: reason>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes: path / no: reason>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes: path / no: reason>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/TOOL-031-pr-agent-reusable-workflow/` -> `specs/archive/TOOL-031-pr-agent-reusable-workflow/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
