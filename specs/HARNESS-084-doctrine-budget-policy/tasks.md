---
tags: [spec, tasks]
created: "2026-09-05"
---

# Tasks - HARNESS-084-doctrine-budget-policy

- [x] Split `render_region` into `render_region_raw` (markers intact) and a marker-stripping
      wrapper; point `render_region_compact` at the raw form, since it needs the markers.
- [x] Teach `render_region_compact` to drop `full-only` regions, on the same
      exactly-identified-lines rule it already used for the provenance blockquote.
- [x] Mark the auto-merge exception and the branch-name paragraph `full-only` in
      `harness/enforced/no-auto-merge.md`.
- [x] Guard (HARNESS-056) asserting BOTH directions, that the region closes, and that the
      markers reach no surface.
- [x] Remove the pipe from `render_region` so a missing source-of-record still fails.

## Not done, deliberately

- [ ] The budget POLICY #1241 asks for. This is the mechanism a policy would need; deciding
      which records may mark regions is a separate judgement.
- [ ] Any enforcement of *which* regions qualify. The boundary is stated, not checked.
