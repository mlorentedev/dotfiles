---
id: "lesson-365-a-cli-that-exits-0-on-its-own-errors-is-answered-by-content"
type: lesson
status: active
title: "A CLI that exits 0 on its own errors is answered by content, not by output"
created: "2026-10-08"
---

# A CLI that exits 0 on its own errors is answered by content, not by output

## Context
`dotf vault health` checks that Obsidian is reachable with `obsidian vault` before it runs the
link and orphan sections. It inherited the check from `vault-health.sh`:
`obsidian_cmd vault 2>/dev/null | grep -q .`. Any output meant "connected".

## The Trap
On macOS, the `obsidian` on PATH is a separate binary (`obsidian-cli` inside the app bundle). It
reads the Linux-only `--no-sandbox` flag as a command. It also prints its own errors on stdout
("Vault not found.", `Error: Command "--no-sandbox" not found.`) and exits 0. So the
connectivity section printed `PASS: Obsidian CLI connected`, and the failure surfaced two sections
later as "cannot read the obsidian CLI's JSON". Until #2154 parsed that JSON, the error text was
only counted as lines, and every section after the check reported a wrong number rather than a
failure.

## The Solution
The connection now passes only when the answer names the vault (`name<TAB>knowledge`), and the
flag is sent everywhere but darwin. Both are pinned by golden cases that fail against the old
code.

When a tool's exit status carries no information, "it printed something" is not a result. Check
for a token only a successful answer contains, and print what the tool said when it is missing.
