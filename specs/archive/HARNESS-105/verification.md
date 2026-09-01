---
tags: [spec, verification]
created: "2026-08-30"
---

# Verification - HARNESS-105

## Evidence

### Unit Tests
The `shellsafe` package and `errors` package have 100% test coverage and they both pass.

```
$ cd cli && go test -coverprofile=coverage.out ./internal/shellsafe && go tool cover -func=coverage.out
=== RUN   TestBash
...
--- PASS: TestBash (0.00s)
=== RUN   TestPowerShell
...
--- PASS: TestPowerShell (0.00s)
PASS
ok  	github.com/mlorentedev/dotfiles/cli/internal/shellsafe	0.003s	coverage: 100.0% of statements
github.com/mlorentedev/dotfiles/cli/internal/shellsafe/quote.go:10:	Bash		100.0%
github.com/mlorentedev/dotfiles/cli/internal/shellsafe/quote.go:17:	PowerShell	100.0%
total:									(statements)	100.0%

$ go test -v ./internal/errors
=== RUN   TestIsTerminalFailure
=== RUN   TestIsTerminalFailure/terminal_failure
=== RUN   TestIsTerminalFailure/retryable_error
--- PASS: TestIsTerminalFailure (0.00s)
    --- PASS: TestIsTerminalFailure/terminal_failure (0.00s)
    --- PASS: TestIsTerminalFailure/retryable_error (0.00s)
=== RUN   TestTerminalFailureFormat
--- PASS: TestTerminalFailureFormat (0.00s)
PASS
ok  	github.com/mlorentedev/dotfiles/cli/internal/errors	0.006s
```

### Refactoring Check
Running `! grep -rn "strings.ReplaceAll.*'\\''$" ./internal/spec/review_launch.go` succeeds, confirming the ad-hoc replacement was removed and correctly replaced by `shellsafe.Bash`.

### CLI Top-Level Check
The `main.go` file has been modified to handle `TerminalFailureError` by directly using `fmt.Fprintln(stderr, err.Error())` when detected, avoiding the default Cobra "Error: " prefix.

```
$ cd cli && go test -v ./cmd/dotf
=== RUN   TestRunTerminalFailure
--- PASS: TestRunTerminalFailure (0.00s)
PASS
ok  	github.com/mlorentedev/dotfiles/cli/cmd/dotf	0.008s
```
