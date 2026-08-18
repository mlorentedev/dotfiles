---
id: lesson-146-a-clean-local-golangci-lint-does-not-certify-ci-v1
type: lesson
status: active
created: "2026-07-10"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 146: A clean local `golangci-lint` does not certify CI — v1 default-excludes errcheck Close/Remove, v2 does not

**Context**: BUG-029 (#696) added an atomic `machine.json` writer with `defer os.Remove(tmpName)` and a `tmp.Close()` in an error branch. `golangci-lint run` was clean locally (the machine's binary was v1.62.2), so the change was pushed as done.

**Problem**: The `cli.yml` `lint` job (`golangci-lint-action@v8`) failed with exactly two errcheck findings — `os.Remove` and `tmp.Close` return values not checked — on those two lines. The action pins golangci-lint **v2**, and v2 dropped v1's default-on exclusion set (`issues.exclude-use-default`), whose built-in list suppressed the stock "Error return value of `(*os.File).Close`/`os.Remove` is not checked" reports. So identical code passes v1 locally and fails v2 in CI: a linter **version** mismatch, not a config or code difference. `go vet` never flags these either, so the local gates were all falsely green.

**Solution**: Write the discards explicitly, matching the pattern the repo already uses (`internal/doctor/checks_secrets_tooling.go`): `_ = tmp.Close()` and `defer func() { _ = os.Remove(tmpName) }()`. The explicit-discard form satisfies errcheck on every version and documents that ignoring the error is deliberate. Confirmed by reading the failed job log (`gh run view <run> --log-failed` → "errcheck: 2") rather than re-guessing.

**Rule**: A clean local `golangci-lint run` certifies nothing when the version differs from CI — `golangci-lint-action@v8` runs golangci-lint v2, which enables errcheck for `Close`/`Remove` that v1 excluded by default. Either pin the CI version locally (check the action's major version → linter major) or, better, write `_ = x.Close()` / `defer func() { _ = os.Remove(...) }()` explicitly so the code is linter-version-agnostic. When a Go PR's `lint` job is red but local is green, read the failed job log for the exact `(linter)` tag before touching code — the fix is usually mechanical once the specific linter is known.
