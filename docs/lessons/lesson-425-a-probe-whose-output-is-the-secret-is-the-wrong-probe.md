---
id: "lesson-425-a-probe-whose-output-is-the-secret-is-the-wrong-probe"
type: lesson
status: active
title: "A probe whose output is the secret is the wrong probe"
created: "2026-10-14"
owner: manu
tags: [security, secrets, gh, transcript, deny-list]
---

# A probe whose output is the secret is the wrong probe

**Context:** An agent was checking whether the real-`gh` bats suites (#2442) would find a GitHub token under an isolated `GH_CONFIG_DIR`. It ran `gh auth token` with that variable set. It tried to discard stdout and keep only the error text: `cmd 2>&1 >/dev/null | head -2`.

**Problem:** The live GitHub CLI OAuth token was printed into the agent transcript. The transcript is stored, synced and read by later sessions, and nothing scans it. Two causes stacked:

1. **`gh auth token` does not honour `GH_CONFIG_DIR` for the keychain.** On macOS it reads the token from the login keychain, whatever config directory it is pointed at. The "isolated" probe was not isolated, so it found the real credential.
2. **zsh's `MULTIOS` sends stdout to the pipe too.** In bash, `cmd 2>&1 >/dev/null | head` passes only stderr down the pipe. In zsh, with the default `MULTIOS` option, an explicit `>/dev/null` together with the pipe makes *two* outputs, and zsh copies stdout to both. The token went to `/dev/null` and to `head`. Measured: `f(){ print out; print err >&2; }; f 2>&1 >/dev/null | cat` prints `err` in bash and `err` then `out` in zsh. The agent's shell is zsh.

The token was revoked and `gh auth login` re-run, so the exposure is closed. That cost a rotation, and the next one would cost the same.

**Solution:**

- `Bash(*gh auth token*)` joins the Claude deny list (`ai/claude/settings.json`), beside `env`, `printenv` and `dotf secrets show`. So does `gh auth status` with its token flag (long and short form), which prints the same credential. A committed test (`tests/claude-settings-template.bats`, "the deny list keeps the commands that print a secret") fails if any of them is dropped.
- To check that a credential is usable, run something that *uses* it and read the exit status. For example, `gh auth status` exits 0 or not. To check that one is *present*, test the variable without expanding it: `[ -n "${GH_TOKEN:-}" ]`.

**Why:** The probe asked "is there a token?" with a command whose answer *is* the token. A redirection or a filter downstream is no protection: the shell may route the value somewhere the author did not expect, as zsh did here, and the harness captures what reaches it. The safe question is one whose output cannot contain the secret: an exit code, a boolean or a length. A deny rule matches spellings, so it is a backstop and not a sandbox. The working rule is to choose a probe that cannot print the value.

The deny list protects Claude only. The other five harnesses have no equivalent rule from a shared source; that gap is HARNESS-182 (#2472).
