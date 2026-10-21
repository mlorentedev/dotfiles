---
id: "lesson-505-a-directory-glob-sorts-by-the-slash-not-by-the-name"
type: lesson
status: active
title: "A directory glob sorts by the slash, not by the name"
created: "2026-10-21"
---

# A directory glob sorts by the slash, not by the name

## Context

`compile-harness.sh`'s `build_skill_catalog` walks the skill records with
`for sk_dir in "$recdir"/*/` and writes one catalog bullet per record. The Go
port (`harness.catalogBullets`, CLI-026 slice 2) walks the same records in
byte order of their names, and the parity oracle runs the shell under
`LC_ALL=C` so both sides should sort alike.

## Problem

They did not. The glob sorts the matched paths, and a `*/` match ends in `/`,
so it compares `test-driven-development/` with `test/`. `-` is 0x2d and `/` is
0x2f, so `test-driven-development` came first, even in the C locale. Byte order
of the names puts `test` first. Setting `LC_ALL=C` fixes the collation, not the
string being collated.

## Solution

Go keeps byte order of the names, which is what the spec declares. The parity
oracle compares the catalog bullets as a set with the sha left out, and a unit
test pins byte order with the `a` / `a-b` pair.

## Rule

When a shell loop's order matters, look at what the glob actually matches, not
the name you strip from it afterwards: a trailing `/` (or an extension) is part
of the sort key. Sort the names explicitly when the order is part of an output.
