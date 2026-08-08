---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - BUG-041-spec-tag-preflight

## Evidence

### The false negative, proven against the tree

```console
$ grep -o '\[AGENT-SUGGESTION[^]]*\]' specs/archive/CLI-002-repo-structure/proposal.md
[AGENT-SUGGESTION — accept or remove]
```

That spec is **archived**. The pattern it passed required `]` immediately after
the keyword, so the canonical emitted form never matched and the lock never
engaged. Establishing the emitted form from `harness/skills/spec/SKILL.md` rather
than from the pattern is what surfaced this — the pattern was the artefact under
suspicion, so it could not also be the reference.

### The live repro for the reported symptom

Session-start flagged `AI-028-hive-install-model-migration` as carrying an
unresolved marker. The sole occurrence is at `tasks.md:17`, inside a **completed**
checkbox whose text states the marker is resolved, and inside a code span.

After the fix, the patched binary emits:

```
[specs] 29 active, 119 archived
```

with no unresolved-tag clause — where the unpatched one named AI-028.

### Tests

`TestScanUnresolvedTags` is table-driven over both directions:

| Must NOT fire | Must fire |
|---|---|
| inside an inline code span | the canonical suffixed form in prose |
| on a completed `- [x]` item | the canonical draft form in a comment |
| inside a fenced block | the bare form in a plain HTML comment |
| inside a tilde-fenced block | on an UNticked checklist item |
| | after a fenced block has closed |
| | with a longer closing fence |

Plus two integration cases at the level the user experiences: a spec that only
quotes the markers archives cleanly, and a spec carrying the canonical emitted
form is still refused by name (the red direction — without it, a scanner matching
nothing at all would satisfy every green case above).

`gofmt` clean on the touched files, `go build ./...` and `go vet ./...` pass, and
the full Go suite is green across 12 packages.

## Self-application

The old binary cannot archive this spec. Its prose necessarily discusses the
markers, so the unpatched pre-flight refuses — the exact bug being fixed. The
archive commit in this PR was produced with the binary built from this branch,
which makes the fix a precondition of the PR's own completion.
