---
id: "lesson-532-a-name-parsed-from-a-url-is-only-valid-on-the-host-it-came-from"
type: lesson
status: active
title: "A name parsed from a URL is only valid on the host it came from"
created: "2026-11-01"
---

# A name parsed from a URL is only valid on the host it came from

## Context

`dotf spec audit` could not run in a repository whose origin is on Gitea (#1906). The ticket asked for the
Gitea URL forms to be recognised.

## Problem

The audit asks GitHub about every spec's issue through `gh api repos/<owner>/<name>/…`, with no hostname.
Parsing `git.example.com/teledyne/projects-toolkit` into `teledyne/projects-toolkit` would have made it ask
github.com about a repository that is not there. The fix the ticket proposed would have turned a clear refusal
into wrong answers. The same value also did two jobs: completing bare `#N` links, and naming the repository
whose default branch holds `specs/archive/`. Under a cross-repo work-gate those are different repositories, and
for a Gitea repository the second is one `gh` cannot reach.

## Solution

`spechome.Resolve` returns two values. The issue home comes from the flag, then the committed
`specs/.bitacora-repo`, then `DOTF_BITACORA_REPO`, then a github.com origin. The archive home is the github.com
origin or nothing. Anything else fails closed, naming the forge and host but never the URL, which may embed a
token. Before turning a URL into an identifier, check which host the consumer will ask with it. If that host is
not the URL's host, the identifier is wrong, however cleanly it parses.
