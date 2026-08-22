---
tags: [spec, verification, templates]
created: "2026-08-22"
---

# Verification - HARNESS-077-capability-map

## Evidence

Every acceptance criterion maps to a named test in `features.json`, and all 13 verifier commands
were executed in this session. AC1-AC5 are Go table tests over the SHIPPED map (not a fixture, so
it cannot drift from what deploys); AC6-AC8 are bats over the real script; AC9 is the real-binary
sibling suite.

The two claims worth singling out, because the whole registry rests on them:

- **The native forms genuinely differ**, from the same neutral request. `shell` resolves to
  `tools: Bash` for claude and `permission: {bash: allow}` for opencode. Asserted against the real
  binary in `tests/compile-harness-real.bats` "the two native forms really do differ".
- **The end-to-end payoff.** `curator` declares `capabilities: [read, search, edit]`; the deployed
  `curator.md` now carries `tools: Read, Glob, Bash` alongside `model: opus`, and no `capabilities:`
  line. Before this change it carried neither.

## Test status

Run in this session, in this worktree:

- `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` -> clean
- `golangci-lint run` -> `0 issues.` (v2.12.2, matching the `versions.conf` pin); `gofmt` clean
  apart from `internal/doctor/report.go`, which is #1154, pre-existing on main and untouched
- `shellcheck --severity=error scripts/*.sh setup-linux.sh` -> exit 0 (CI's own severity; the
  info-level notes on setup-linux.sh are pre-existing and are #1083)
- `bash -n` and `zsh -n` on `compile-harness.sh` -> clean
- `./scripts/check-bats-names.sh tests/` -> clean
- `bats tests/*.bats` -> **1429 passing, 0 failing**
- All 13 `features.json` verifiers executed and passing; each propagates the runner's exit status
  and pins its test by unique NAME rather than by `ok N` position (lesson 217, and the round-1
  review finding on #1165)

## Decisions made during implementation

- **The resolver takes a capability SET, not one capability.** A `csv` field is an allow-list —
  naming a tool grants it and omitting one denies it — while a `decision-map` grants without
  denying. A per-capability API would push that distinction onto every caller, and the shell caller
  is the one least able to carry it. So the map returns the complete native value for a set.
- **The resolver returns the WHOLE frontmatter line**, field name included, because the field
  differs per harness (`tools:` vs `permission:`) and the render has no business knowing which.
  This differs deliberately from `resolve-tier`, where the field is always `model:`.
- **No `x-` cross-block rule for this schema.** `model-map`'s `customRules` is a package-level
  closed set bound to one schema, and `TestShippedSchemaDeclaresEveryCustomRule` asserts that
  schema declares every rule in it — so a second registry cannot declare one without refactoring a
  file hardened over six adversarial review rounds. The one cross-block invariant here
  (`checkVocabularyCoverage`) runs in the loader instead. Refactoring that plumbing as a side
  effect of adding a registry is exactly the kind of drive-by change that destabilises hardened
  code, so it is filed separately.
- **The capability probe was generalised over the subcommand name.** With two registry consumers,
  each field must probe independently — a binary new enough for one and not the other is the
  realistic staleness shape, and it now degrades only the field it cannot resolve. The Go tripwire
  test was retargeted at both names accordingly.
- **Scope was cut to hold ADR-017's atomic-PR cap.** The `dotf doctor` check over this registry and
  #1164's record-vs-map tier check both live in `cli/internal/doctor/`, are diagnostics rather than
  parts of the render, and pushed the production diff past ~300 lines. They ship together as the
  immediate follow-up rather than being dropped.

## Promotion candidates

- [ ] Lesson for `docs/lessons/`? no - the transferable finding here (allow-list vs decision-map
      changes what a partial answer MEANS) is recorded in the registry's own `$comment`, which is
      where a future reader of the map will actually look.
- [ ] ADR-worthy? no - ADR-027 §2 already decided this map exists and ADR-035 decided its shape.
- [ ] Vault pattern? no - single-project so far.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/HARNESS-077-capability-map/`
- [ ] Bitacora #560 closed with the PR link
- [ ] `/adversarial-review HARNESS-077-capability-map` run and PASSing (the archive gate refuses
      without a fresh review signed by a model in `harness/reviewer-pool.json`)
