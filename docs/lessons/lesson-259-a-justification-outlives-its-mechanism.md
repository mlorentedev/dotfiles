# 259 - A justification outlives its mechanism as easily as a name outlives its contract, and the cleanup that taught us the first lesson left an instance of it

**Date:** 2026-09-02
**Area:** setup scripts, doctor, technical debt

## What happened

OPS-043 (#1337) asked to delete `check_deployed` and `check_dependencies` from
`setup-linux.sh` as duplicates of `dotf doctor`. Measuring first found the
premise false in five places. Two of those failures have the same shape, at
different distances from the code.

**A name outliving its contract.** The ticket read `check_deployed` and
`checkDeployDrift` as the same check under two spellings. They are different
legs of the same chain:

| | Compares | Severity |
|---|---|---|
| `checkDeployDrift` (Go) | repo → `~/.dotfiles` | FAIL on drift |
| `check_deployed` (shell) | `~/.dotfiles` → `$HOME` | FAIL on drift **and** on a symlink |
| `checkSymlinks` (Go) | `$HOME` existence | PASS on a drifted real file |

So the shell function was the *only* byte-level assertion on the second leg
anywhere in the repo — and `setup-windows.ps1` never had it, leaving that leg
unguarded on Windows outright. Deleting as asked would have removed coverage
while the diff read as removing duplication.

**A justification outliving its mechanism.** The NOTE explaining why `.zshrc`
was exempt from the byte comparison said setup "may legitimately modify it (e.g.
stripping stale gh-copilot eval lines via `sed -i`)". There is no `sed -i` left
in the script; the only match in the whole file is inside that comment. The
OPS-040 purge removed the mechanism a day earlier and left the sentence.

That is the part worth recording. **OPS-040's entire method was lesson 256 —
probe the target, never read the block's own comment.** It still left behind a
comment describing a thing it had just deleted. Knowing the lesson, in writing,
while applying it, was not enough to avoid producing another instance of it.

## Why this class survives knowing about it

A false comment is written by someone who was wrong. **These comments were
true when written.** Nobody introduced an error; the world moved underneath a
correct sentence, and the sentence has no link to the thing it describes. There
is no moment where a careful reviewer would have caught it, because at every
individual commit the text was accurate.

Both halves are also *cheap to leave*: a stale justification still justifies
something plausible, and a misleading name still names a real function. Nothing
fails. The cost lands later, on a reader who takes the text as evidence — which
is how #1337 came to be filed in the first place.

## What to do instead

**Put the justification next to the data it justifies, and make its absence a
test failure.** The exemptions that used to live in a shell comment now live in
`homeDeployMap` as a required field:

```go
{src: ".gitconfig", dst: ".gitconfig",
    exemptReason: "every `git config --global` rewrites it; measured drifting on a converged box 2026-09-02"},
```

`TestHomeDeployExemptionsAreReasoned` fails on an exemption with an empty
reason. That does not detect a *stale* reason — nothing can, cheaply — but it
forces the reason to be re-read whenever the entry is touched, which is the only
moment anyone would notice.

**Guard the join, not the two sides.** Nothing in the repo spanned
`setup-linux.sh`'s deploy calls and doctor's coverage lists, which is precisely
the gap #1337 came through: a claim about coverage that no test could refute.
`TestHomeDeployMapCoversSetup` parses the script and fails in both directions —
an uncovered `deploy_file` call, *and* a map entry setup no longer deploys. The
second direction is the one that catches this lesson's class.

**When a ticket asserts duplication, compare direction and severity before
deleting.** Two checks with similar names are duplicates only if they compare
the same things at the same strength. Here they shared neither.

## Related

- #1337 — this spec; #1447 — the Windows half of the leg, still unguarded
- Lesson 256 — a cleanup block's own description of what it deletes is not
  evidence. This is that lesson recurring inside the work that established it.
- Lesson 257 — the same purge; a one-time migration with no removal date
- Lesson 232 — detect the shape that is wrong, not the shape that is right
