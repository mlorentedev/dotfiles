---
id: "BUG-081b-pi-runtime-secret"
type: spec
status: draft
created: "2026-08-16"
issue: "mlorentedev/dotfiles#987"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — BUG-081b-pi-runtime-secret

TDD order. The guard is written before the source flips, because the guard's
whole job is to make the old state visible — and the old state is the one on
this machine right now, which makes it the only chance to observe it red against
reality rather than a fixture.

## 1. Guard, red first

- [ ] Test: a deployed pi config containing `{env:` FAILs, naming the file and
      the fact that pi cannot resolve it.
- [ ] Test: a deployed pi config whose `apiKey` is a bare literal FAILs, naming
      it as a materialised secret. **The literal must not be echoed** — report
      its presence, never its value (CLI-038's rule applies to the guard too).
- [ ] Test: `${VAR}` and `$VAR` both PASS. Both are pi syntax; accepting only the
      braced form would fail a correct config.
- [ ] Test: no deployed pi config at all → SKIP, not FAIL. A machine that never
      installed pi has nothing broken.
- [ ] Observe the FAIL against the real deployed config on this machine, which
      currently carries a materialised literal.

## 2. Source flip

- [ ] `ai/pi/models.json`: `"apiKey": "{env:NAN_API_KEY}"` → `"${NAN_API_KEY}"`.
- [ ] Test: `dotf secrets render` leaves the file byte-identical — it substitutes
      only `{env:VAR}`, so this is a passthrough. Pins the claim the whole
      "no setup edits" argument rests on.
- [ ] Confirm no other `{env:` remains in the file (there is exactly one today).

## 3. ADR

- [ ] Short ADR: agent-config secrets resolve at **runtime**, not at deploy time;
      deploy-time materialisation retires per config as each converts.
- [ ] Record the considered alternative — pi's `!command` form, rejected because
      it depends on `dotf secrets show` (#952) — and the assumption that the
      injection wrappers exist on all three platforms.

## 4. Verification

- [ ] `go build`, `go vet`, `go test ./...`, `golangci-lint run` at the
      `versions.conf` pin.
- [ ] Prove pi resolves `${NAN_API_KEY}`: import its shipped resolver and compare
      fingerprints (never values). The passing case, since #987's own evidence
      describes a machine state that has since been reverted.
- [ ] Doctor observed FAILing on the real deployed config, and PASSing against a
      corrected copy.
- [ ] `git diff --stat setup-linux.sh setup-windows.ps1` is empty.

## 5. Knowledge

- [ ] `docs/lessons.md`: the transferable half is not "pi uses `${}`" but that
      **three layers each reported a health none had established** — setup's
      "resolved at runtime", pi's `isConfigValueConfigured` returning true for the
      broken form, and a 401 that cannot distinguish a bad credential from a
      malformed one.
- [ ] Close #987 with the change that closed it.

## Out of scope reminders

The setup scripts (#1023 retires those blocks); opencode (untested assumption);
a macOS bootstrap; `dotf secrets show`'s contract (#952).
