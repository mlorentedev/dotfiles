---
id: "lesson-365-a-cli-that-exits-0-on-its-own-errors-is-answered-by-content"
type: lesson
status: active
title: "A CLI that exits 0 on its own errors is answered by content, not by output"
created: "2026-10-08"
---

# A CLI that exits 0 on its own errors is answered by content, not by output

## Context
`dotf vault health` calls the Obsidian CLI for its connectivity, orphan, dead-end, unresolved-link
and tag sections. It inherited the argv and the checks from `vault-health.sh`:
`obsidian --no-sandbox <sub> --vault <name>`, and `obsidian_cmd vault | grep -q .` for "connected".

## The Trap
The CLI exits 0 on its own errors and prints them on stdout, so every check that read "any output"
or "count the lines" turned an error into a result. Measured on obsidian 1.14.4, three defects had
been hiding behind it since the shell version:

- `--vault <name>` is not a parameter. The documented form is `vault=<name>` as the first
  argument. The CLI ignored the flag and answered for whichever vault was active, or printed
  "Vault not found." when none was. The result depended on the app's state.
- `--no-sandbox` is an Electron flag. The CLI reads it as a command (`Error: Command
  "--no-sandbox" not found.`).
- `dead-ends` is not a command (`deadends` is). Its one error line was counted as one dead end.
  The section has reported `1/N` for as long as it has existed. The real figure was 88%.

The connectivity section printed `PASS: Obsidian CLI connected` over each of these errors, and the
golden corpus agreed, because its stub answered whatever argv it was given.

## The Solution
One argv on every OS, taken from the CLI's documentation rather than from the earlier script:
`vault=<name> <sub...>`. The connection passes only on the vault's own record (`name<TAB><vault>`).
The stubs answer in the real CLI's format, and the golden argv pins the documented form.

When a tool's exit status carries no information, "it printed something" is not a result. Check
for a token only a successful answer contains, and print what the tool said when it is missing.
When a port reproduces an oracle byte for byte, it also reproduces the oracle's bugs. Check the
contract with the tool against the tool's own documentation once.
