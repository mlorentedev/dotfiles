---
id: "lesson-395-an-environment-token-is-not-a-stored-login"
type: lesson
status: active
title: "An environment token is not a stored login"
created: "2026-10-11"
---

# An environment token is not a stored login

## Context
`dotf converge` installs gh's git credential helper only when `gh auth status --hostname
github.com` succeeds, and `dotf doctor` FAILs when the helper is missing while that same query
says it could be installed (#2319).

## The Trap
gh reads `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN` and `GITHUB_ENTERPRISE_TOKEN` before
its stored login, so `gh auth status` answers "logged in" for whatever process carries one. On the
Windows CI leg, converge ran in the setup step, which has no token, and left the helper out; the
doctor gate ran with `GH_TOKEN` set and called the same machine repairable. The disagreement was
filed as runner-only and carried as a permanent known failure of the gate (#2212, #2225). On a
real machine the same split is a wrong answer, not a CI quirk: a terminal with a token in its
profile reports a login that a GUI app or a scheduled task, which never sees that profile, does
not have.

## The Solution
Both converge and doctor ask about the stored login: the query runs with the four token variables
removed from its environment (`gitconfig.WithoutEnvTokens`). The two now agree with each other and
with every process that will later call the helper, and the gate's permanent exception matched
nothing and was removed.

## Takeaways
- When a tool resolves credentials from the environment first, a probe of "is this machine set up"
  must strip that environment, or it measures the caller instead of the machine.
- Two checks that answer the same question in different environments will eventually disagree,
  and the one that disagrees in CI is the one that runs with the extra secret.
