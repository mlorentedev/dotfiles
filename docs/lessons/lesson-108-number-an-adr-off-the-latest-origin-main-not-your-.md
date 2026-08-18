---
id: lesson-108-number-an-adr-off-the-latest-origin-main-not-your-
type: lesson
status: active
created: "2026-06-18"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 108: Number an ADR off the latest origin/main, not your branch base — a stale base collides with ADRs shipped in parallel

**Context**: I wrote ADR-023 for cross-machine path resolution, taking the next number from my branch base (highest was ADR-022). While I worked, `main` advanced: ADR-023 (agnostic session-start) and ADR-024 (PAT-expiry) shipped. On merge, my ADR-023 collided.

**Problem**: ADR numbers are a shared append-only namespace, but I picked the next one from a *stale local base* instead of current `main`. A parallel session had already claimed 023 and 024. The collision surfaced only at integration; renumbering to 025 then meant chasing ~20 files + 3 GitHub issue/PR bodies — and surgically *not* touching the other ADR-023's references in files that carried both (the session hooks).

**Solution**: Renumber to ADR-025 (next free on merged main), case-sensitive + slug-aware; in files referencing both ADR-023s, edit only the cross-machine line.

**Rule**: Before assigning any number in a shared append-only namespace (ADRs, migrations, ticket IDs), resolve it against the *latest* `origin/main`, not your branch base — `git fetch && git ls-tree origin/main docs/adr/`. When parallel work is likely, reserve the number up front rather than guessing.
