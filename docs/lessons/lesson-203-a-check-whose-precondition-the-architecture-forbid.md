---
id: lesson-203-a-check-whose-precondition-the-architecture-forbid
type: lesson
status: active
created: "2026-08-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 203: A check whose precondition the architecture forbids reports SKIP forever, and SKIP reads as nothing-to-check

**Context**: `dotf doctor`'s PAT-expiry section probes each GitHub PAT for liveness and days-to-expiry. It was written when secrets were loaded into the login shell by `load-secrets.sh`, so it read its token with `sys.Getenv(...)` and SKIPped when the var was unset — "fresh shell, no alarm", which was true at the time. ADR-028 then retired the loader: secrets are injected into one child process on demand and **never** exported into the ambient environment. Nobody revisited the check.

**Problem**: the SKIP branch became the only reachable branch. On every correctly configured machine the token resolved to `""`, so the section reported SKIP for every PAT — and its remediation told the reader to run `secrets_refresh`, a command retired with the loader and present nowhere in the repo. Neither half was noticed for months, because both failure modes are silent by construction: a SKIP is indistinguishable from "nothing to check here", and nobody runs a remediation command for a check that is not complaining. A second, independent defect hid inside the first — selection matched the age blob's `github.` filename prefix, so `BITACORA_PAT` fell out of monitoring the day it migrated to Bitwarden and acquired no filename to match. Repairing both and running it once found a **genuinely dead token**: the Bitwarden copy of `BITACORA_PAT` returns HTTP 401, and had been dead behind a check that could not report it.

**Solution**: resolve through the sanctioned read path (a `System` seam over the secrets `Loader`, the same one `dotf secrets run` uses) instead of reading an environment the architecture guarantees is empty; select on the registry's declared `validate: github-token` marker instead of a filename convention, so the check is backend-neutral by construction rather than by a second prefix rule. The severity contract was deliberately left alone — HTTP 401 stays the only FAIL — because `doctor` is the last step of `setup-linux.sh` and a new non-zero branch fails the bootstrap of every mid-migration machine. The concern that argued for stricter was answered in wording instead: both resolution branches state outright that the expiry is **not** being monitored, so the SKIP cannot be read as "nothing to do".

**Rule**: when an architecture change removes a *precondition* that existing checks depend on, the checks do not fail — they go quiet, in the branch that was written to mean "this is fine". Grep for consumers of what you removed (an env var, a file, a daemon, a login step) and ask of each: on a correctly configured machine after this change, which branch does it take? A check that can only take its no-op branch is not a weakened check, it is a deleted one that still prints. Two tells worth trusting: a remediation string naming a command that no longer exists is a dead check with a timestamp on it — grep the repo for the command before believing the message — and a health section that has never once complained is a claim about the section, not about the system.

**Tags**: `verification`, `secrets`, `doctor`, `architecture-migration`, `observability`
