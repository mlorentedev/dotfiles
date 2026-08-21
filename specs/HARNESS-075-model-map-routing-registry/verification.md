---
tags: [spec, verification, harness, orchestration]
created: "2026-08-21"
---

# Verification — HARNESS-075-model-map-routing-registry

> Every criterion below is proven by a command run in this session, with its observed output.
> Where an assertion could pass vacuously, it was **mutation-tested**: the artifact was broken
> deliberately and the guard was required to fail with a message naming the fault.

## AC1 — the map exists, parses, carries all seven blocks

```
$ ~/.local/bin/bats tests/model-map.bats
ok 1 model-map.json exists and is valid JSON
ok 3 model-map.json carries all seven declared blocks
```

## AC2 — the map validates against the schema, read as data

`ValidateModelMap` reads `harness/model-map.schema.json` and enforces what it finds there, rather
than restating its rules in Go — so the schema file remains the single source of truth and the Go
package is only its interpreter. No schema-engine dependency was added; `cli` still has three
direct dependencies (cobra, yaml, term).

```
$ cd cli && go test ./internal/harness/ -run TestModelMapValidatesAgainstSchema
ok  github.com/mlorentedev/dotfiles/cli/internal/harness
```

**Mutation — an unimplemented schema keyword must be loud, not silently skipped:**

```
$ python3 -c "...d['oneOf']=[]..."      # add a construct the validator does not implement
--- FAIL: TestModelMapValidatesAgainstSchema
    the shipped map must validate against the shipped schema:
    harness/model-map.schema.json declares "oneOf" at (root), which this validator does not implement
```

That is the property that keeps native validation honest. A validator that skipped what it does not
implement would report a document valid without checking what the schema asked for — health it
never established.

## AC3 — the schema rejects a harness naming an undeclared pool

The rule a stock JSON Schema cannot express, and the one ADR-032 §3's reference block actually
violated.

```
$ cd cli && go test ./internal/harness/ -run TestSchemaRejectsDanglingPoolReference -v
--- PASS: TestSchemaRejectsDanglingPoolReference
    --- PASS: /a_harness_naming_an_undeclared_pool_is_rejected
    --- PASS: /a_harness_naming_a_declared_pool_is_accepted
    --- PASS: /one_bad_reference_among_several_good_ones_is_still_rejected
```

**Mutation — against the shipped map, end to end:**

```
$ python3 -c "...d['harnesses']['pi']['pools']=['ghost']..."
--- FAIL: TestModelMapValidatesAgainstSchema
    harnesses.pi.pools[] names "ghost"
```

## AC4 — no retired provider is declared or referenced

`openrouter` (deleted upstream, August 2026) and `codex` (no longer used by the owner, confirmed
2026-08-21). **Structural, not textual** — the `$comment` names both on purpose so the absences read
as decisions, and a grep-based guard would fail on that very sentence.

```
$ cd cli && go test ./internal/harness/ -run TestModelMapDeclaresNoRetiredProvider
ok
$ ~/.local/bin/bats tests/model-map.bats
ok 4 no retired provider is declared as a pool or referenced by a harness
```

**Mutation, both providers:**

```
$ python3 -c "...d['pools']['openrouter']={...}..."
    model_map_test.go:138: pools declares "openrouter", which is retired
$ python3 -c "...d['pools']['codex']={...}..."
    model_map_test.go:138: pools declares "codex", which is retired
```

## AC5 — the two consumer classes are reachable separately

```
$ cd cli && go test ./internal/harness/ -run TestModelMapConsumerClasses -v
--- PASS: /tier_resolution_is_compile_time
--- PASS: /chain_resolution_is_run_time
--- PASS: /the_top_tier_has_no_fallback,_on_purpose
```

`ResolveTier` errors rather than returning `""` when a tier declares no model for a harness —
resolving to an empty model id would render a definition naming no model at all.

## AC6 — doctor distinguishes three broken states, none permissive

Verified **live through the built binary**, not only in tests:

```
$ go build -o dotf-test ./cmd/dotf

# absent
[Routing registry]
  [FAIL] harness/model-map.json not found at … — this is not an empty routing map,
         and nothing falls back to a default

# unparseable
  [FAIL] harness/model-map.json could not be parsed as JSON: unexpected end of JSON input

# schema-invalid
  [FAIL] harness/model-map.json does not satisfy harness/model-map.schema.json:
         harnesses.pi.pools[] names "ghost"
         declare each pool, or remove the reference — a harness pointing at a pool
         that does not exist routes nowhere at dispatch time

# healthy
  [INFO] declared concurrency: claude 10, nan 5 (reserve 2) — DECLARED, not enforced:
         nothing decrements these today (ADR-035 level 1)
  [INFO] pool "nan" is shared with pi-tui, qq, qf, hive-embeddings, pr-agent-ci, none of
         which routes through dotf — so the eventual guarantee is "dotf alone will never
         be the cause of exhaustion", never "exhaustion will not happen"
```

The test additionally **bans** the phrases `no pools` and `0 pools` from every broken state, and
asserts the three messages differ from one another. A fourth state is covered that C15 did not name:
the map present with its schema missing — a map that validates against nothing is not validated.

## AC7 — the budget is declaration only

```
$ cd cli && go test ./internal/harness/ -run TestModelMapBudgetIsDeclarationOnly
ok
```

`Budget.ConcurrencyDeclared` separates *not declared* from *zero*, because zero reads as "no
capacity" and absence means "this pool does not state one" — the honest answer for a seat-based pool
where concurrency is a fleet property. Nothing in the package decrements anything; the type is
called `Budget` and the accessor `DeclaredBudget` precisely so no caller reads it as enforcement.

## AC8 — guards ship in this PR (C10)

```
$ ~/.local/bin/bats tests/model-map.bats
1..8
ok 1 … ok 8       (8/8)
```

Go: `TestSchemaRejectsDanglingPoolReference`, `TestValidatorRejectsUnimplementedSchemaConstructs`,
`TestModelMapValidatesAgainstSchema`, `TestModelMapDeclaresNoRetiredProvider`,
`TestModelMapConsumerClasses`, `TestModelMapBudgetIsDeclarationOnly`,
`TestModelMapCheckThreeBrokenStates`.

## AC9 — the full local loop

```
$ cd cli
$ go build ./...            → build OK
$ go vet ./...              → vet OK
$ GOOS=windows go vet ./... → windows vet OK
$ go test ./...             → 17 packages ok, 0 FAIL
$ golangci-lint run         → 0 issues.   (v2.12.2, matching the versions.conf pin)
```

## Two defects found in this spec's own verification, before any code was written

Recorded because they are the exact class the spec exists to prevent, committed inside the spec.

1. **`features.json` f1 could never pass.** `$comment` was embedded in a **double-quoted**
   `python3 -c` argument, so the shell expanded it to the empty string before python saw it: the
   required-key set became `{'', 'version', …}` and the check exited 1 on a *correct* map,
   permanently. Measured against a seven-key stub — exit 1 under the old form, exit 0 under the
   fixed one, and exit 1 on a stub missing `chains`.
2. **f4 would have failed a correct map.** It forbade the string `openrouter` anywhere in the file,
   while `tasks.md` asks the `$comment` to carry the measured rationale — and *"openrouter was
   deleted upstream"* is that rationale. Replaced with a structural check.

A verification that reports a state the system is not in is the thing constraint C15 names. Finding
two of them inside the spec written to enforce C15 is worth recording rather than quietly fixing.

## Not done here, and stated rather than implied

- **Level 2 budget enforcement.** ADR-035 splits it; level 2 needs a dispatcher to decrement a
  semaphore and `dotf agent` is still an unknown command. Declaring a budget nothing decrements is
  what ADR-035 itself calls advisory, so this ships level 1 and says so in the map, in the schema
  description, and in the doctor output.
- **`harness/capability-map.json`** — re-scoped to #560.
- **The independent adversarial review** the archive gate requires. Not run yet; the reviewer must
  not be the implementer.
