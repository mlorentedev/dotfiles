# Lesson 218 — The Go build cache does not see the data file your test reads

**Date:** 2026-08-21
**Context:** Mutation-testing new guards over `harness/model-map.schema.json` while closing
HARNESS-075's round-5 review.

## What happened

A new guard was added and the standard proof was run: break the thing the guard protects, confirm
the test fails, restore. The mutation deleted `propertyNames` from the shipped schema. The test
reported:

```
ok  github.com/mlorentedev/dotfiles/cli/internal/harness  (cached)
```

`(cached)` is the whole finding. The guard was never re-executed, so the experiment established
nothing — and it read exactly like a passing experiment establishing that the guard was vacuous.

Go's test cache keys a result on the package's own inputs and on files the test opens **through
paths the toolchain can attribute to the module at build time**. These tests read the shipped
registry through a path computed at run time:

```go
root := repoRootForTest(t)
raw, err := os.ReadFile(filepath.Join(root, ModelMapSchemaFile))
```

Nothing in that chain is a build input. Edit `harness/model-map.json` or its schema, re-run
`go test ./...`, and the cache answers for the previous contents of a file it never saw.

## The fix

Two, and both are needed because they cover different moments:

```bash
# proving a guard bites, or verifying after editing any shipped registry
go test -count=1 ./...
```

And in `specs/*/features.json`, every `go test` verification command carries `-count=1`. Without it
a cached run reprints `--- PASS: <Name> (0.00s)` verbatim, so the lesson-217 predicate —
`grep -q -- '--- PASS: <Name> '` — succeeds against a run that predates the edit under test. That
predicate is non-vacuous against a *missing* test and was never non-vacuous against a *stale* one.

CI is unaffected: it starts with a cold cache. This is a local-verification hazard, which is worse,
because local is where the claim "I ran it and it passed" is produced.

## The lesson

**A test that reads a file the build system does not know about is not covered by the build
system's idea of freshness.** The general shape: any caching layer invalidates on the inputs it can
see, and a test fixture loaded by run-time path is invisible to all of them. The same applies to
golden files opened relative to a discovered repo root, and to anything a test pulls from
`$HOME` or a deployed directory.

**Corollary for mutation testing.** The experiment has three arms, not two: the guard must fail on
the mutation, pass on the original, *and* the runner must be shown to have actually re-run. `ok
(cached)` satisfies neither of the first two — it answers a question nobody asked.

Two vacuous guards were caught by this mutation loop in the same session, both of which five rounds
of adversarial review had passed over. The first failed for five reasons unrelated to the property
under test. The second was rejected by a neighbouring pattern rule before the guard was reached.
**A "this is rejected" assertion is worth nothing without a control arm asserting that a document
differing in exactly one attribute is accepted** — and "exactly one" has to be checked, not assumed.

## See also

- `docs/lessons/lesson-217-go-test-run-passes-when-the-test-name-matches-not.md` — the same
  `features.json` predicate, against a missing test rather than a stale run
- `docs/lessons/lesson-214-a-declared-status-is-not-evidence-probe-the-syst.md`
- `docs/lessons/lesson-209-every-layer-reported-a-health-none-of-them-had-est.md`
