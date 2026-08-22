---
tags: [spec, verification, templates]
created: "2026-08-22"
---

# Verification - HARNESS-076-model-map-tier-render

## Evidence

- [x] **AC1** `dotf harness resolve-tier top --harness claude` prints `opus`, exit 0
      -> Go `TestHarnessResolveTier/the_tier_the_only_deployed_agent_declares`; real-binary
      `tests/compile-harness-real.bats` "resolve-tier writes the model id to stdout, not stderr"
- [x] **AC2** an undeclared tier/harness pair exits non-zero naming both, stdout empty
      -> `TestHarnessResolveTier/a_declared_harness_that_no_tier_declares` (top/copilot),
      `/a_harness_declared_in_one_tier_asked_for_another` (top/opencode),
      `/a_tier_the_map_does_not_declare_at_all` (ultra/claude); real-binary sibling test 3
- [x] **AC3** an absent or schema-invalid map fails rather than defaulting (C15)
      -> `TestHarnessResolveTierFailsLoudWithoutAMap`, four cases: no map, no schema beside it,
      a map the schema rejects (ghost pool in `chains`), a map that is not JSON
- [x] **AC4** `model: top` on curator renders as `model: opus`
      -> bats "agents: --deploy resolves the neutral model tier into the rendered frontmatter";
      end-to-end through the real binary in "a full deploy renders the resolved model id"
- [x] **AC5** an unresolvable tier fails the render naming harness and tier
      -> bats "an unresolvable tier fails the deploy instead of rendering a model-less definition"
      and "an unresolvable tier leaves the PREVIOUS agent definition intact"
- [x] **AC6** skill deploy survives a failed agent render
      -> bats "skill deploy survives a failed agent render" (asserts the skill landed while the
      deploy exited non-zero)
- [x] **AC7** `kind`, `capabilities`, `skills`, `targets` still dropped; only `model` changed
      -> bats "agents: --deploy renders agent-md (name+description+provenance; neutral keys dropped)"
- [x] **AC8** bats covers render + unresolvable path; Go table tests cover the subcommand
      -> 6 new bats cases in `tests/compile-harness.bats`, 5 in `tests/compile-harness-real.bats`,
      3 Go tests (12 subtests) in `cli/internal/cmd/harness_resolve_tier_test.go`

**Deviation from AC6 as written, and why.** The criterion said the deploy fails when `dotf` is
absent. It does not: a missing or too-old resolver now WARNS and renders without the model line.
See "Decisions" below — the change was forced by evidence, and the failing half of AC6 (skill
deploy survives) is still asserted, against the unresolvable-tier failure instead.

## Test status

Every command below was run in this session, in this worktree.

- `cd cli && go build ./... && go vet ./...` -> clean
- `GOOS=windows go vet ./...` -> clean (the Windows CI leg compiles the same tree)
- `go test ./...` -> all packages ok; `internal/cmd` 0.459s
- `golangci-lint run` -> `0 issues.` (v2.12.2, matching the `versions.conf` pin exactly)
- `gofmt -l .` -> reports only `internal/doctor/report.go`, which is **#1154, pre-existing on
  main** and untouched here
- `shellcheck scripts/*.sh setup-linux.sh` -> clean (one pre-existing SC1091 info on an unrelated file)
- `bash -n` on every script, `zsh -n scripts/compile-harness.sh` -> clean
- `./scripts/check-bats-names.sh tests/` -> `OK (99 file(s) clean)`
- `bats tests/*.bats` -> **1413 passing, 0 failing**
- Baseline confirmed: the same suite on `main` before this branch -> 0 failing, so the 17
  failures seen mid-implementation were this change's and are now resolved rather than tolerated

Manual smoke: the real `dotf` built from this branch resolves `top/claude` -> `opus`,
`mid/opencode` -> `qwen3.6-plus`, and refuses `top/copilot`. A full `--deploy` into a throwaway
`$HOME` writes `model: opus` into `curator.md`.

## Decisions made during implementation

- **Fail-loud was narrowed, and the test suite is why.** The first implementation failed the render
  whenever a tier could not be resolved, including when `dotf` was missing. That broke 17 tests in
  `tests/skills-pipeline.bats`, which drive the real deploy — because the `dotf` deployed on this
  machine predates the subcommand (#1158). The rule now splits by cause: a map that is absent,
  invalid, or silent about a tier fails loudly (that is C15, which governs a map that cannot be
  READ); a resolver that is absent or too old warns loudly and renders without the model line,
  because that is a bootstrap state and `setup-linux.sh` installs `dotf` best-effort. Making it
  fatal would promote a warned-past dependency into a hard prerequisite of the whole harness
  deploy. The degraded output is exactly what this script produced before the change, so the path
  is never worse than the status quo, and it still SAYS SO (ADR-032's honest-degradation driver).
- **Exit status cannot tell the two apart, so the probe asks a different question.** Measured
  2026-08-21: a `dotf` predating the subcommand answers
  `harness resolve-tier top --harness claude` with `unknown flag: --harness` and **exit 1** —
  identical in status to a genuine routing refusal. `dotf_knows_resolve_tier` therefore greps
  `dotf harness --help` for the subcommand name, which is the only question whose answer does not
  depend on the arguments the old binary failed to parse. `TestHarnessHelpListsResolveTier` pins
  the string, so a rename cannot silently make the probe answer "too old" forever.
- **`render_agent` stayed a pure renderer; resolution moved to the deploy caller.** `--check` also
  renders, and it runs in the CI `lint` job, which installs no Go. Resolving inside the renderer
  would make a drift gate report drift on a perfectly good record purely because the machine has
  no `dotf` — conflating a property of the deploy ENVIRONMENT with a property of the committed
  RECORD. The caller resolves and passes the finished line in.
- **The render writes through a temp file.** `render_agent ... > "$outp"` truncates the target
  before the renderer runs, so a failed resolution left an EMPTY agent definition behind: a file
  naming no model, the exact failure the change exists to prevent. It now renders to `mktemp` and
  moves on success, leaving the previous definition intact on failure.
- **A stub was paired with a real test rather than exempted.** `tests/stub-real-pairing.bats`
  (BUG-055) flagged the new `dotf` stub. Unlike the `gh`/`hive` suites in its exemption table,
  `dotf` is our own binary and a real test mutates nothing, so `tests/compile-harness-real.bats`
  builds it and drives the real path end to end.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? **yes** - a stale CLI's refusal is indistinguishable
      from a legitimate one by exit status, so a version gate must probe capability, not outcome.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no - ADR-035 already decided the
      two consumer classes; this implements the compile-time one and decides nothing new about them.
- [ ] New pattern candidate for `00_meta/patterns/`? no - single-project so far. If a second repo
      hits the stale-binary-probe problem, promote the lesson above.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-076-model-map-tier-render/` -> `specs/archive/HARNESS-076-model-map-tier-render/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
- [ ] `/adversarial-review HARNESS-076-model-map-tier-render` run and PASSing (the archive gate
      refuses without a fresh review signed by a model in `harness/reviewer-pool.json`)
