# Verification

Output of `pi -v` after changes (no warnings):

```text
1.0.0
```

## Independent review follow-up (2026-10-03)

The review (`review.md`, PASS WITH GAPS, agy/gemini-3.1-pro-high) found one REAL
minor gap: AC1 also moved `pi-mcp-client` into `retire`, but `retire` archives
paths under `~/.pi/agent/` and no such path exists. Removing the package from
`packages` is what uninstalls it. The `retire` entry is dropped in the archiving
PR, so AC1's "added to `retire`" half no longer describes the tree; the rest of
AC1 stands. `ai/pi/README.md` still described `pi-mcp-client` as the MCP client
and now describes pi's native one.

Evidence after the fix, on Linux (msi):

```text
$ go test ./internal/pi/...
ok  	github.com/mlorentedev/dotfiles/cli/internal/pi	0.005s

$ bats tests/pi-packages.bats tests/pi-config.bats
38 tests, 0 failures

$ go run ./cmd/dotf pi packages check; echo "exit=$?"
pi packages already reconciled (9 declared, 0 changed)
exit=0

$ pi mcp list   # native client, no extension package
hive: connected, 13 tools (codemode, global)
context7: connected, 2 tools (codemode, global)
sequential-thinking: connected, 1 tool (codemode, global)
```
