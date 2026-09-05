---
tags: [spec, verification]
created: "2026-09-05"
---

# Verification - SEC-002-secrets-run-pty

## Evidence

| Criterion | Proof |
|---|---|
| Child gets a pty when the parent has one, a pipe when not | `TestRunChildPTY_ChildSeesATerminal` + `TestRunChild_ChildSeesAPipe`. Both run `sh -c 'test -t 1'` — the same question `pi` asks — and assert opposite answers on the two paths. |
| Secret redacted through the pty, split across writes | `TestRunChildPTY_RedactsASecretSplitAcrossWrites`. Child emits `mock-open` / sleep 0.2 / `router-test-token-val`, so the value genuinely crosses a write boundary. |
| Prefix-aware hold-back | `TestRedactWriter_ReleasesFrameWithNoSecretPrefixImmediately` (released without a Flush) and `TestRedactWriter_HoldsBackATrailingSecretPrefix` (a real prefix is still withheld). |
| #1459 guard on the new path | `TestRunChildPTY_HonoursTheIntrospectionGuard`. |
| Windows keeps the pipe path | `GOOS=windows go vet ./...` clean; `secrets_child_windows.go` returns `interactiveChildSupported() == false`. |

### Mutation proof (lesson 267)

The new tests were proven in the failing direction, and the mutation was proven to have **landed**
(the patch asserts its anchor before applying, so a silent no-op cannot pass as a result):

```
mutation applied            <- holdBack reverted to the old fixed-window rule
--- FAIL: TestRedactWriter_ReleasesFrameWithNoSecretPrefixImmediately
--- FAIL: TestRedactWriter_HoldsBackATrailingSecretPrefix
mutated_rc=1
restored_rc=0
```

### End-to-end, from a binary rebuilt from this branch

The installed `dotf` is `vcs.modified=true` (built from a dirty tree), so all evidence comes from a
fresh build at a scratchpad path. Reported as exit status and byte counts — never the captured
stream, which for a TUI is ~40KB of escape sequences and is the channel `secrets-never-in-output`
governs.

| Case | Old binary | New binary |
|---|---|---|
| `secrets run -- pi` under a real pty (40x120) | `bytes=0`, dead in 1.5s | **`bytes=41904`, alive at timeout** |
| `pi` directly (reference) | — | `bytes=50969`, alive |
| `secrets run -- pi -p 'reply with exactly: OK'` | exit 124 @90s | **exit 0, `OK`, 118s** |
| `secrets run -- opencode --version` (canary) | — | exit 0 |

Harness: `pty.fork()` + `TIOCSWINSZ` set to 40x120 after the fork. A bare `pty.fork()` leaves the
window at 0x0 and the TUI refuses to draw, which is a false negative.

## Test status

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` — all clean
- `go test ./...` — all packages pass
- `golangci-lint run` at the pinned **2.12.2** (`versions.conf`) — **0 issues**
- No regression: the four pre-existing `runChild` tests still pass unchanged

## Decisions made during implementation

- **`pi -p` at 90s looked like a regression and was not.** The old binary timed out identically, and
  at a 240s budget the new one returns `OK` in 118s. Recorded because the first measurement said
  "regression" and only the controlled comparison against the old binary said otherwise — a
  conclusion drawn from the new binary alone would have been wrong in either direction.
- **The pty was the easy half.** The blocking defect was `redactWriter` withholding a fixed
  `maxSecretLen-1` bytes on every write. A pty alone would have produced a TUI that renders late and
  misplaced rather than not at all.
- **One redactWriter on the pty path, not two**, because a pty carries a single stream. This also
  removes a latent interleaving hazard between two writers each holding their own tail.
- **Windows keeps the pipe path deliberately.** ConPTY is a different API that the Linux CI leg
  cannot prove, and a half-working pseudo-console fails the same silent way as the bug being fixed.

## Promotion candidates

- [ ] Lesson for `docs/lessons/`? **yes** — "an `io.Writer` that is not an `*os.File` silently
      becomes a pipe", and the wider shape: a security wrapper that breaks the tool it protects gets
      routed around, restoring the exposure it existed to prevent.
- [ ] ADR? no — this implements ADR-028's existing decision, it does not change it.
- [ ] Pattern for `00_meta/patterns/`? no — Go-specific.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/SEC-002-secrets-run-pty/`
- [ ] #1506 closed with the PR link
- [ ] Independent adversarial review (reviewer != implementer) passed
