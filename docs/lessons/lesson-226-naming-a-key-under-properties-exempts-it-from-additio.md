# Lesson 226 — Naming a key under `properties` exempts it from `additionalProperties`, so adding a constraint there can loosen the schema

**Date:** 2026-08-23
**Area:** registries / JSON Schema / guards
**Severity:** medium — the edit reads as a tightening in review and in the diff, and it is a loosening

## What happened

`harness/model-map.json` declares `chains`, an ordered per-tier fallback. Every
tier's value was held to one shape by `additionalProperties`:

```jsonc
"chains": {
  "type": "object",
  "minProperties": 1,
  "additionalProperties": {
    "type": "array", "minItems": 1,
    "items": { "type": "string", "pattern": "^[^:[:space:]]+:[^:[:space:]]+$" }
  }
}
```

ADR-032 §4 says the top tier never degrades, so `chains.top` must not be allowed
to declare a fallback. The obvious edit — cap it at one entry:

```jsonc
"properties": {
  "top": { "maxItems": 1, "description": "the top tier declares no fallback" }
},
"additionalProperties": { …unchanged… }
```

That edit adds a constraint and removes four. In JSON Schema,
`additionalProperties` applies **only to properties not matched by `properties`
or `patternProperties`**. Naming `top` moves it out of that set, so it keeps
`maxItems: 1` and loses `type: array`, `minItems: 1` and the `pool:model`
pattern that every other tier is still held to.

Measured by deleting the fix and re-running the cases:

| `chains.top` | With the `$ref` | Without it |
|---|---|---|
| `["claude:opus", "nan:deepseek-v4-flash"]` | rejected (`maxItems`) | rejected |
| `["claude-opus"]` — not `pool:model` | rejected (`pattern`) | **accepted** |
| `"claude:opus"` — a bare string | rejected (`array`) | **accepted** |
| `[]` — nothing to route to | rejected (`minItems`) | **accepted** |

`maxItems` does not apply to a string, so the one constraint that survived was
also inert on two of the three newly-legal values. The key the rule was written
to protect became the only key in the file with no shape at all.

## Why it survives inspection

**The diff is a pure addition.** Nothing was deleted, so a reviewer scanning for
removals sees none. What was removed is an *application* of a rule that is still
sitting there in the file, four lines below, looking as though it covers
everything.

**The real map still validates.** `chains.top` is `["claude:opus"]`, which
passes either way, so the existing suite stayed green and the shipped registry
was never in danger. The hole is only reachable by a value nobody has written
yet — which is exactly what a schema is for.

**Both keywords are present and correct.** There is no typo, no invalid schema,
no warning from the validator. `properties` and `additionalProperties` are doing
precisely what the specification says.

## The fix

Extract the shape once and reference it from both places, so the named key is
constrained *in addition to* its cap rather than *instead of* it:

```jsonc
"$defs": { "chain": { "type": "array", "minItems": 1, "items": { … } } },

"chains": {
  "properties": { "top": { "$ref": "#/$defs/chain", "maxItems": 1 } },
  "additionalProperties": { "$ref": "#/$defs/chain" }
}
```

In draft 2020-12 `$ref` composes with sibling keywords, so this needs no
`allOf`. (Under draft-07 it would: `$ref` there replaces the schema it sits in,
and `maxItems` beside it is ignored — the same class of silent loss.)

## The rule

**When you add a key to `properties` in a schema that relies on
`additionalProperties`, the new key inherits nothing. Reference the shared shape
explicitly, and write the test that proves the *unchanged* constraints still
apply to it** — not only the new one you came to add.

`TestChainsTopIsCappedWithoutLosingTheChainShape` is that test: five cases, of
which four assert constraints this edit was never meant to touch. Three of them
go red when the `$ref` is removed, which is how the hole was found.

## Relation to Lessons 204 and 220

[204](lesson-204-a-check-that-cannot-fail-the-way-you-cite-it.md) — a check that
cannot fail the way you cite it. This is that shape in a schema: the constraint
is cited as "top is capped and validated like any chain" and can only fail on the
first half.

[220](lesson-220-four-defects-one-shape-a-thing-verified-by-a-proxy.md) — its
diagnostic question, *what would this still pass on if the thing it checks were
broken?*, is what surfaced it. Answering it required deleting the fix and running
the cases, not re-reading the schema; re-reading is how the bug got written.

## Evidence

- Mutation run, 2026-08-23: `$ref` removed from `chains.top` → 3 of 5 cases in
  `TestChainsTopIsCappedWithoutLosingTheChainShape` fail, with
  `chains.top: "claude:opus"` and `chains.top: []` both accepted
- `harness/model-map.schema.json` — `$defs.chain` and the two references to it
- JSON Schema draft 2020-12, §10.3.2.3: `additionalProperties` applies to
  instance names not matched by `properties` or `patternProperties`
- `specs/CLI-042-dotf-agent-run/` — the PR B change that needed the cap
