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

`ResolveTier` errors rather than returning `""` when a tier declares no model for a harness, **and
also when it declares a blank one** — resolving to an empty model id would render a definition
naming no model at all.

**That second half was not true when this line was first written**, and an adversarial review caught
it (`agy/gemini-3.1-pro-high`, Major/REAL). The type assertion on `string` succeeds for `""`, so
only a *missing key* errored while an *empty value* returned `""` with a nil error. The claim was
broader than the code — the fourth instance in this session of a verification asserting more than it
checks, and the first one found by someone other than its author.

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

Shell layer, run to completion with its **own** exit code captured rather than a pipeline's:

```
$ ~/.local/bin/bats tests/*.bats > bats-full.txt 2>&1; echo "BATS_EXIT=$?"
BATS_EXIT=0
1..1394        ok: 1394        not ok: 0
```

The eight new cases ran as 631-638 — checked by their numbers in the output, not assumed from the
file existing.

**An earlier attempt at this evidence was worthless and is recorded rather than replaced.** The
first run was `bats tests/*.bats | tail -6`, whose exit code belongs to `tail`, not to `bats`. It
reported 0 while proving nothing, and the captured file held nine lines. Same class as finding 3
below, and the second time in one session that a truncating pipe swallowed the exit code being
reported on.

## Three defects found in this spec's own verification

Recorded because they are the exact class the spec exists to prevent, committed inside the spec.
The third is the worst of them and was found last, by running the commands rather than reading them.

### 3. Five of the eight verification commands could pass vacuously

`go test -run <name>` **exits 0 when the name matches nothing**:

```
$ go test ./internal/harness/ -run TestThisNameDoesNotExistAnywhere
ok  github.com/mlorentedev/dotfiles/cli/internal/harness  0.002s [no tests to run]
$ echo $?
0
```

So `f2`, `f3`, `f5`, `f6` and `f7` — every Go-backed criterion — would have reported PASS the moment
its test was renamed or deleted. The features.json contract says a criterion is verified; what it
actually asserted was that the package compiles.

Fixed by requiring the test to have reported a pass, which fails on a **missing** test and on a
**failing** one:

```bash
cd cli && go test ./internal/<pkg>/ -run '^<Name>$' -v 2>&1 | grep -q -- '--- PASS: <Name> '
```

Here the pipeline's exit code being `grep`'s is the point rather than the hazard. Verified across
all three states:

| state | exit |
|---|---|
| test exists and passes | 0 |
| test name matches nothing | 1 |
| test exists and fails | 1 |

And mutation-verified on the real command — renaming `TestModelMapValidatesAgainstSchema` away made
`f2` exit 1, and restoring it returned 0.

**A first attempt at the fix was itself wrong**, which is worth recording: the pattern
`'--- PASS: <Name>$'` never matched, because `go test -v` appends a duration (`--- PASS: TestX (0.00s)`).
It failed closed rather than open, so it was caught immediately — but a check that always fails gets
disabled rather than fixed, which is its own failure mode.

### 1 and 2, found before any code was written



Recorded because they are the exact class the spec exists to prevent, committed inside the spec.

**f1 could never pass.** `$comment` was embedded in a **double-quoted**
   `python3 -c` argument, so the shell expanded it to the empty string before python saw it: the
   required-key set became `{'', 'version', …}` and the check exited 1 on a *correct* map,
   permanently. Measured against a seven-key stub — exit 1 under the old form, exit 0 under the
   fixed one, and exit 1 on a stub missing `chains`.
**f4 would have failed a correct map.** It forbade the string `openrouter` anywhere in the file,
   while `tasks.md` asks the `$comment` to carry the measured rationale — and *"openrouter was
   deleted upstream"* is that rationale. Replaced with a structural check.

A verification that reports a state the system is not in is the thing constraint C15 names. Finding
two of them inside the spec written to enforce C15 is worth recording rather than quietly fixing.

## Independent adversarial review — three rounds, three FAILs, all findings reproduced first

`dotf spec review` on the primary (`nan/deepseek-v4-flash`) died with
`429: deepseek-v4-flash concurrency limit: max 5 simultaneous requests` after three auto-retries —
PR-Agent was reviewing this very PR and holding the slots. That is the case `reviewer-pool.json`
predicted in writing, so the run fell through to the **provider-diverse fallback**:

```yaml
verdict: "FAIL"          # first pass
reviewed_sha: "cd79d936618eaac21b6a2faa15ff4379a725fefb"
reviewer: "agy/gemini-3.1-pro-high"
```

**All three findings were reproduced before being accepted**, and all three were real.

| # | Severity | Finding | Reproduced | Fixed |
|---|---|---|---|---|
| 1 | Major | `checkPoolReferences` walked only `harnesses`; a ghost pool in `chains` or `services` validated cleanly | ghost in both blocks passed `TestModelMapValidatesAgainstSchema` | walks all three; `chains` entries split on `pool:model` and a malformed entry is itself an error |
| 2 | Major | `ResolveTier` returned `""` with a nil error for a blank declared model | `ResolveTier returned "", err=<nil>` | blank ids rejected loudly; the overstated claim above corrected |
| 3 | Minor | `additionalProperties: "false"` (a string) hit neither switch arm, so undeclared keys passed | undeclared key accepted | `default` arm errors; `nil` handled separately as JSON Schema's real default |

Finding 1 mattered most for `chains`: it is what a dispatcher walks at run time, so a ghost there is
a fallback that fails at the exact moment the primary already has.

Finding 3 is the keyword allow-list argument one level down — **be loud about a schema you cannot
read, whether the unreadable part is an unknown key or a malformed value.** The original allow-list
covered unknown keys and left malformed values failing open, which is the same defect it was written
to prevent.

Re-run of the reviewer's own mutations against the fixed code:

```
services.embeddings.pool names "ghost"
chains.mid[0] names "ghost"
additionalProperties is string, which this validator cannot interpret
```

### Round 2 — `nan/deepseek-v4-flash`, FAIL

It confirmed all three round-1 findings were closed **and found that one of them was closed at the
wrong layer**.

| # | Severity | Finding | Reproduced | Fixed |
|---|---|---|---|---|
| 1 | Major | Blank model ids were still **schema**-valid. Round 1 guarded `ResolveTier`, the loader path; the schema constrained every id slot to `type: string` with no length assertion | `tiers.mid.claude=""`, `chains.mid=["nan:"]`, `services.embeddings.model=""` all validated and `dotf doctor` printed OK | validator gains `minLength` and `pattern`; the schema carries both. `minLength` alone does not cover chains — `"nan:"` is four characters — so the shape is the assertion |
| 2 | Minor | `x-poolReferencesResolve` read through a bool-only helper, so the **string** `"true"` silently disabled the whole cross-block rule | dangling `ghost` accepted, `err=nil` | errors on any non-boolean; the helper it used has no callers left |
| 3 | Question | The doctor check reads the **deployed** copy, so a repo ahead of the deploy dir reports `[FAIL] not found` | `~/.dotfiles/harness/` holds the other six registries | `setup-linux.sh` copies the whole directory (`cp -rf harness/.`), so no wiring is needed — but the message now says the check reads the deployed copy and that re-running setup mirrors it |

Finding 1 was **worse than the bug it replaced**: the map said a fallback existed, the fallback was
nothing, and the check written to catch exactly that certified it healthy.

### Round 3 — `agy/gemini-3.1-pro-high`, FAIL

Launched on the primary, which died with `429: deepseek-v4-flash concurrency limit` — PR #1150's own
CI run was holding the slots, which is the contention #1150 exists to end. Fell through to the
provider-diverse fallback, as `reviewer-pool.json` prescribes.

| # | Severity | Finding | Reproduced | Fixed |
|---|---|---|---|---|
| 1 | **Blocker** | A **known** keyword carrying a malformed value silently disabled its rule — `required: "pools"`, `minLength: "2"`, `minItems: "5"`, `enum: {}`, `pattern: 42` | **5 of 5 validated cleanly** | structural, not another special case: `implementedKeywords` now maps each keyword to its expected value type and the up-front walk asserts it. **0 of 5 remain** |
| 2 | Major | A tier in `tiers` with no entry in `chains` validated, then failed when a dispatcher resolved it — at run time, under load | adding a `ghost` tier passed | new `x-tiersHaveChains` rule, declared in the schema like its sibling |
| 3 | Minor | The chains pattern ended `.*`, so `"nan:deepseek   "` and `"nan:deep:seek"` matched | both matched | pattern ends `+` and excludes whitespace |

**The Blocker is the lesson of this spec.** Rounds 1 and 2 each closed this hole for one keyword —
`additionalProperties`, then `x-poolReferencesResolve` — and round 3 proved keyword-by-keyword was
the wrong shape entirely: five more were still open. **Three rounds of patching instances before
fixing the class.**

A defect of my own surfaced while fixing it and is recorded rather than quietly corrected: the first
revision `return`ed from the pool-reference rule, so a schema requesting **both** cross-block rules
got only the first checked. A rule that is declared and never runs is the same shape as one that
passed. `TestBothCrossBlockRulesRun` pins it.

**Reviewer hygiene note:** the round-3 reviewer left its mutation battery in the tree
(`zz_review_mutations_test.go`) despite the file's own header saying *"Deleted after the review
run"*. Removed by hand. Worth knowing that the transcript is the record and the tree is not
guaranteed clean afterwards.

**All three rounds stamp `date: "2026-08-20"`, and it was the 21st.** GUARD-004's review stamped
2026-08-22. The model writes that field, so it is invention, not a clock — recorded on #1138
(comment `5364846335`), together with a second launcher gap the same rounds measured: the runner
retries the SAME model three times on a 429, which cannot work against a per-model limit, while the
ordered pool exists precisely so something can walk it. It was walked by hand twice tonight.

**This sentence previously claimed the note had been posted when it had not.** It was written at the
same time as the intention and never checked — a verification record asserting a state the system
was not in, inside the spec that exists to make that impossible. Corrected by posting the comment,
not by softening the claim.

### Round 4 — `nan/deepseek-v4-flash`, FAIL — and the threshold fired

| # | Severity | Finding | Reproduced | Disposition |
|---|---|---|---|---|
| 1 | Major | Malformed `properties` **elements** and non-string `required` **members** skipped silently — the class round 3's Blocker was meant to seal, one level deeper | `properties: {"a": 5}` and `required: ["a", 5]` both validated | **library** |
| 2 | Minor | `minimum` unimplemented, so `concurrency: -5` validated | reproduced on the shipped schema | **library** + `minimum: 0` on the budget fields |
| 3 | Minor | `$comment` carried a truncated sentence, in the file whose purpose is carrying rationale | direct read | fragment removed |
| 4 | Minor | `tasks.md` boxes stale — PR open, review box unticked | `gh pr list --head` | corrected |
| 5 | Question | `tiers` keying rule unstated: `pi` has no tier entry while `nan` (a pool) carries one | code read | **answered in the `$comment`**: tiers key by whatever *consumes* the id. `claude`/`opencode` render it into their own definition, so they key by harness; `nan` keys by pool because `pi` and `agy` take the model as a launcher flag. A harness whose render is `adapter` has no rendered field to hold an id, which is why `ResolveTier(m, t, "pi")` errors rather than inventing one |

**Findings 1 and 2 are both interpreter-semantics, and the threshold declared before round 4 ran
said one was enough.** It fired. The hand-rolled interpreter was replaced by
`santhosh-tekuri/jsonschema/v6` for standard keywords; the schema file stays the SSOT and both
custom cross-block rules stay, because no schema language expresses them. Full reasoning, the
measured dependency cost, and what the swap changed in the tests is in `proposal.md`'s Risks
section — including a **real regression the swap introduced and fixed**: the library treats our two
`x-` keywords as annotations, which is the tolerance the custom rules need, and it means the library
does not police their types.

**Every named regression test from rounds 1–4 passes against the library-backed validator.**

```
go build ./...            OK
go vet ./...              OK
GOOS=windows go vet ./... OK
go test ./...             17 packages ok
golangci-lint run         0 issues   (v2.12.2, matching the pin)
bats tests/model-map.bats 8/8
go test -fuzz             60s, 821k executions, no crash
dotf doctor (real binary) healthy map -> INFO; ghost pool -> FAIL naming chains.mid[0]
```

### What four rounds cost, and what they bought

Eight interpreter-semantics defects, closed one at a time across three rounds before the fourth
proved the class was not hand-fixable. **Two domain-rule defects, both mine, both closed on the
first attempt** — `checkPoolReferences` missing `chains`/`services`, and the early-`return` that let
one cross-block rule disable the other.

That split is the finding worth keeping: **the standard half kept losing, the domain half did not.**
It is the frontier the swap now draws — standard keywords delegated, domain rules written here.

## Not done here, and stated rather than implied

- **Level 2 budget enforcement.** ADR-035 splits it; level 2 needs a dispatcher to decrement a
  semaphore and `dotf agent` is still an unknown command. Declaring a budget nothing decrements is
  what ADR-035 itself calls advisory, so this ships level 1 and says so in the map, in the schema
  description, and in the doctor output.
- **`harness/capability-map.json`** — re-scoped to #560.
- **The independent adversarial review** the archive gate requires. Not run yet; the reviewer must
  not be the implementer.
