---
id: "lesson-357-a-mode-convergence-must-only-narrow"
type: lesson
status: active
title: "A mode convergence must only narrow"
created: "2026-10-07"
---

# A mode convergence must only narrow

## Context
CLI-055 taught `dotf deploy` to fix the mode of a file whose content was already in sync. On
Windows, a 0600 file deployed by an older binary had kept the ACL its directory handed down, and
nobody would ever have tightened it. The fix compared the file's bits with the declared ones and
applied the declared mode whenever they differed (#1664).

## The Trap
"Differs" runs in both directions. An operator who ran `chmod 0600` on a deployed 0644 file had
it set back to 0644 by the next `dotf deploy`, and the only report was `mode fixed`. The tool
undid deliberate hardening and described that as a repair. Doctor never noticed, because it asked
a different question: `PlanConfig` compared bytes only. Doctor said "in sync" while
`dotf deploy --dry-run` said `would fix mode`, even though doctor's own comment claimed the two
"can never disagree".

## The Solution
Converge toward the declared mode only along the axis where the declaration protects something.
The owner gets exactly the declared bits, because a missing owner bit breaks the tool. Group and
other keep a declared bit only while the file still grants it, so tightening survives. A bit the
declaration does not grant is removed everywhere. `deploy.ModeDrift` computes that target. It is
the single predicate behind both deploy's fix and doctor's drift line, so a tightened file is not
drift for either of them. A content rewrite installs the same narrowed mode, or the next change to
the source would undo the tightening the in-sync path had kept.

When two commands must agree on a state, give them one function that answers it, not two
compares that happen to match today. A comment that claims parity is not a test of it.
