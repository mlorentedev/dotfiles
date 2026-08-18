---
id: lesson-050-the-detection-probe-must-use-the-race-free-pattern
type: lesson
status: active
created: "2026-05-21"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 050: The detection probe MUST use the race-free pattern, not the upstream broken one

**Context:** BUG-015 (PR #81, today) added a healthcheck probe to detect when claude-mem's path-resolution cascade fails. The probe used the EXACT SAME `{ printf; ls; printf; } | while ... break` pattern as the upstream hooks it was checking. After BUG-017 patched the upstream hooks with `head -n1`, the probe itself STILL raced — reporting false-positive FAIL because the probe-internal EPIPE fires before resolution completes. BUG-022 had to ship a separate fix to apply the same `head -n1` to the probe.
**Problem:** When writing a detection/observability layer for a known upstream bug pattern, copying the broken pattern verbatim "to faithfully reproduce what the hook does" preserves the same failure mode in the detection layer. The probe inherits the bug it was designed to detect → false-positive FAILs masking real state.
**Solution:** Probes MUST use the canonical race-free version of the pattern being detected. The probe is not the same as the upstream — its job is to ACCURATELY REPORT state, not faithfully reproduce broken behaviour. If the bug-class signature still appears in source (e.g. `break; }; done` in hooks.json), the probe should grep for the broken signature in the FILE (cheap, deterministic) rather than re-execute the broken logic. Cross-OS parity matters: even when the race is silent on Linux (SIGPIPE clean), use the race-free form for consistency.
**Tags:** `#observability` `#detection-layer` `#race-condition` `#healthcheck` `#cross-os-parity`
