---
id: "lesson-495-a-heredoc-inside-a-command-substitution-still-runs-its-backticks"
type: lesson
status: active
title: "A heredoc inside a command substitution still runs its backticks"
created: "2026-10-18"
---

# A heredoc inside a command substitution still runs its backticks

## Context

Opening a GitHub issue with a markdown body full of code spans, in the shape
every agent reaches for:

```sh
gh issue create --body "$(cat <<'EOF'
... `dotf deploy` is a file table ...
EOF
)"
```

The `'EOF'` was quoted, which stops *parameter expansion*. It does not stop
**command substitution in backticks**, because the heredoc sits inside `$( )`,
which the shell parses as a script before the heredoc body is read. The stored
issue body came out with the code spans replaced by their output, and one of the
"code spans" was `dotf deploy` — so creating the issue also **ran a deploy**
against the machine (`in sync` on every config, no writes, by luck rather than by
design).

## Rule

Build a body with `write`/`cat > file <<'EOF'` in a *separate, unquoted-context-free*
step and pass the file:

```sh
gh issue create --title "…" --body-file /tmp/issue.md
```

`--body-file` has no shell in the path, so nothing in the markdown can execute.
The same applies to `gh pr create/edit`, `dotf pr` calls that take a body, and
anything else taking free text that will contain backticks or `$( )`.

## Check

After creating or editing, read it back and confirm the code spans survived:

```sh
gh issue view <n> --json body --jq '.body' | grep -c '`'
```

An empty-looking section, or a code span whose text is a command's output, is the
symptom — the fix is to rewrite the body with `--body-file`, and to treat whatever
the backticks ran as an unintended side effect to verify.
