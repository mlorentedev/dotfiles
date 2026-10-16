---
id: "lesson-446-herdr-isolates-a-second-server-only-by-home"
type: lesson
status: active
title: "herdr isolates a second server only by HOME; its variables move the sockets, not the session"
created: "2026-10-16"
owner: manu
tags: [herdr, testing, isolation, incident]
---

# herdr isolates a second server only by HOME; its variables move the sockets, not the session

**Context:** CLI-118's `dotf worktree add --agent` drives herdr, and its live check was meant to run against a throwaway server so the operator's own session stayed untouched. herdr 0.9.3 reads `HERDR_SOCKET_PATH`, `HERDR_HOME` and `HERDR_CONFIG_PATH`, so a server started with all three pointed at a scratch directory looked isolated.

**Problem:** It was not. On 2026-10-16 at 01:11Z the second `herdr server` listened on the scratch socket, as asked, and then loaded `~/.config/herdr/session.json`: all ten of the operator's workspaces. It relaunched every agent in them with its saved resume id, six `claude` and one `agy`, and saved `session.json` back. One resumed `claude` was a copy of the session that started the server; within seconds it wrote an interrupted-tool marker into that live session's transcript, forking its tree. `kill -9` ended it before anything else ran (SIGTERM would have let it save again). A first attempt had failed earlier with `sun_path` too long: a socket under a deep scratch directory exceeds macOS's 104 bytes, which is why the short path that did work was chosen. Its log lines also went into the operator's `herdr-server.log`.

**Solution:** No second herdr server on a machine with a live one, whatever its variables say. A test that needs a real server gets a different `HOME` (and so a fresh `~/.config/herdr` with no session to restore), with a socket path under 104 bytes; an agent started there has none of the operator's config, so it exercises the refusal paths, not the full one. Anything that must see the operator's real session reads the running server and starts nothing. The coordination note on #2550 carries this to CLI-127, whose live check had the same shape.

**Why:** "Isolated" was inferred from the variables a tool exposes, not measured from what it reads. herdr resolves its socket from the variables and its persisted state from `HOME`, and only the first was tried. Before calling a second instance of any stateful tool isolated, list what it loads at start (sessions, caches, logs, credentials) and check each one lives under the override, by watching it start in a sandbox with nothing to find.
