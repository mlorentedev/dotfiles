---
id: "CLI-024-secrets-fail-loud"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-26"
issue: "mlorentedev/dotfiles#612"
tags: [spec, proposal, secrets, cli, go, hardening]
template_version: "1.0"
---

# CLI-024-secrets-fail-loud

## Why

A 3-agent audit (#612, Phase A) found that `dotf secrets` resolution can fail
**silently** — the worst failure mode for a secrets path:

1. **Empty value injected/printed without error.** A bw field (or age secret)
   that resolves to `""` is injected by `run` as `VAR=` and printed by `show`
   (exit 0). The launched process runs **unauthenticated**; `KEY=$(dotf secrets
   show x)` yields an empty key — both with no signal. `render` already rejects
   empty, so the same input is "a failure" in one path and "success" in another.
2. **`render` swallows the real error and exits 0.** `resolve()` discards the
   underlying error, collapsing *locked vault*, *wrong age key*, *registry typo*,
   and *genuinely-absent secret* into one indistinguishable "unresolved" bucket,
   and the command returns 0 regardless. Under `set -e` setup, a wrong age key or
   a locked vault ships a config with literal `{env:VAR}` placeholders on a green
   exit.
3. **`--only ","` (all-empty tokens) selects nothing silently** — the child runs
   with zero secrets, exit 0, no error.

Silent failure in a secrets path means unauthenticated processes and green builds
on broken config. This PR makes resolution **fail loud**.

## What

1. **A1 — empty resolved value is a hard error.** `EnvFor` (run) and `show` reject
   an empty resolved value (env or file) fail-fast. `fieldFromItem` errors when a
   typed field (`password`/`username`/`notes`) is requested but the Bitwarden item
   has **no such block** (e.g. `field: password` against a secure-note item),
   instead of returning `""`.
2. **A2 — `render` surfaces the specific error and distinguishes absence from
   misconfiguration.**
   - **Absent** (the `sensitive/<x>.secret.age` file does not exist) stays
     non-fatal and quiet — the expected partial-setup case. The age resolver
     signals this with a sentinel `ErrSecretAbsent` (checked via `errors.Is`).
   - **Misconfiguration** (wrong key, locked vault, decrypt error, empty value,
     bw item/field typo) is **never swallowed**: the *specific* underlying error
     is printed to stderr per var.
   - A new **`--strict`** flag makes any real failure (not mere absence) exit
     non-zero, so setup/CI can abort. Default stays non-fatal (setup completes)
     but now **loud** about real errors.
3. **A3 — an explicit `--only` that resolves to zero secrets is an error**
   (`resolveOnly` rejects an all-empty/garbage selection). `--only` omitted = all
   (unchanged).

## Out of scope

- The lifecycle commands (`verify`/`set`/`migrate`/`rotate`/`sync`/`retire`) — #612
  Phase C.
- Validation hardening (global VAR uniqueness, `FileExpose.Mode`, atomic
  materialize, path-traversal, age stderr) — #612 Phase B.
- `stripBackendAuth` (the `run` child-env credential leak) — the separate open PR.

## Risks / open questions

- **Backward compatibility of "empty = error".** None of the registry secrets are
  legitimately empty; if one ever is, the fix is an explicit per-entry `allowEmpty`
  flag — never a silent default. Documented, not built now.
- **Absent vs misconfig classification.** age = `os.Stat` the `.secret.age` before
  decrypt → missing file = `ErrSecretAbsent`; a present-but-undecryptable file is a
  real error. bw errors are all treated as real (a registry-declared bw secret that
  is not in the vault *is* a misconfiguration). This keeps classification robust (no
  stderr string-parsing).
- **`render` default must stay non-fatal** so setup completes when a secret is
  genuinely absent on this machine — only `--strict` is fatal; misconfig is always
  loud regardless.
- **Existing tests change behaviour** — render/run tests that relied on silent-empty
  must be updated to expect the new errors.

## Acceptance criteria

- [ ] **AC1** — `run` and `show` exit non-zero with a clear message when a secret
  resolves empty (env or file); `field: password` against a non-login bw item errors
  clearly. *Verify:* Go tests (fake BWReader returning empty / non-login JSON).
- [ ] **AC2** — `render` prints the *specific* underlying error per misconfigured var
  (no swallow); a genuinely-absent age secret stays non-fatal + quiet (placeholder
  intact, exit 0); `--strict` makes a real failure exit non-zero. *Verify:* Go tests
  (absent → exit 0 + placeholder; locked/empty → stderr has the real error; `--strict`
  → non-zero).
- [ ] **AC3** — an explicit `--only` resolving to zero secrets returns an error.
  *Verify:* Go test (`--only ","`).
- [ ] **AC4** — `go test ./... && go vet ./... && gofmt` clean; existing secrets tests
  updated to the fail-loud contract; no scope creep.

## References

- Issue / backlog: `mlorentedev/dotfiles#612` (Phase A items A1–A3).
- Audit: parallel code-reviewer + silent-failure-hunter + design, 2026-06-26.
- Code: `cli/internal/secrets/{resolve,bw,render}.go`, `cli/internal/cmd/secrets.go`.
- ADR: `docs/adr/adr-028-secrets-two-tier-bitwarden-age.md`.
