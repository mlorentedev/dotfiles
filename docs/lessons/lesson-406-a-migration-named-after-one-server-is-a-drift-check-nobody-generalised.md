---
id: "lesson-406-a-migration-named-after-one-server-is-a-drift-check-nobody-generalised"
type: lesson
status: active
title: "A migration named after one server is a drift check nobody generalised"
created: "2026-10-13"
---

# A migration named after one server is a drift check nobody generalised

## Context
`dotf deploy` registers Claude Code's MCP servers from `mcp-servers.json`. It skipped every
name `claude mcp get` already knew, which is idempotent and wrong: it answers "is a server with
this name registered", not "is it registered the way the SSOT says". When hive moved from
`uvx hive-vault` to `hive client` (HIVE-118), that gap was closed for hive alone, with a regexp
in Go and a remove-before-loop block in each setup twin.

## The Trap
The next rename went through the same gap. `mcp-servers.json` changed pdf-modifier's package from
`uvx pdf-modifier-mcp` to `uvx pdf-modifier`; every machine that already had it kept launching
the old package. Deploy said `present`, converge said `ok`, doctor never asked. The HIVE-118 fix
looked like a migration, so nobody read it as what it was: a drift check written for one server.

## The Solution
BUG-128 (#2397) reads `claude mcp get`'s output into `transport target` (`Type:` plus `URL:`, or
`Command:` and `Args:`), renders the SSOT entry the same way, and re-registers any server whose
two strings differ. hive's old entry is now one case of the rule. An answer that does not parse
is reported as unverified and left alone, because a format change in Claude Code must never read
as "everything drifted" and remove every server.

## Takeaways
- An idempotence check keyed on presence alone freezes the first version of every entry. Compare
  the content the SSOT declares, not the name.
- A special case named after one item (`staleHive`) is the signal: ask whether the next item can
  hit the same gap, and generalise it then rather than at the second incident.
- When the comparison depends on another tool's human output, the unparseable case must fail
  safe (leave it and say so), and the test fixtures must be the measured output, not a shape
  the parser happens to accept.
- Read what the answer is about before acting on it. `claude mcp get` reports the winning scope
  (local, then project, then user), while dotf only adds and removes at user scope; acting on a
  local registration's drift would remove nothing and add a user entry it still shadows. A
  replace that removes first also owes a restore when the add fails.
