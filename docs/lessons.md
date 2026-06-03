---
id: "dotfiles-lessons"
type: lesson
status: active
tags: [project, dotfiles, lessons]
created: "2025-12-15"
owner: manu
---

# Dotfiles - Lessons Learned

> Patterns learned, mistakes avoided, and best practices discovered during development.
>
> **Protocol**: Update after every correction or important discovery.

---

## Format

```markdown
### [YYYY-MM-DD] Title

**Context**: What I was doing

**Problem**: What went wrong or what I discovered

**Solution**: How I resolved it

**Rule**: Pattern to follow going forward
```

---

## Entries

### [2025-12-15] echo -e breaks in zsh

**Context**: Shell scripts used `echo -e "\033[32mDone\033[0m"` for colored output

**Problem**: zsh does not support the `-e` flag on `echo`. Color codes printed as literal text instead of being interpreted.

**Solution**: Replace all `echo -e` with `printf '%b'` which handles escape sequences portably.

**Rule**: Never use `echo -e`. Always use `printf '%b' "..."` for colored or escaped output.

### [2025-12-15] &>/dev/null is not POSIX

**Context**: Scripts used `&>/dev/null` to suppress both stdout and stderr

**Problem**: `&>` is a bash-ism. Not POSIX-compliant and can fail in strict zsh configurations or when scripts are sourced in unexpected contexts.

**Solution**: Replace all `&>/dev/null` with `>/dev/null 2>&1` across every script.

**Rule**: Always use `>/dev/null 2>&1` for output suppression. Never use `&>`.

### [2025-12-15] ((count++)) exits with code 1 when count is 0

**Context**: Counter variables used `((count++))` inside scripts with `set -e`

**Problem**: In bash, `((0))` evaluates to false and returns exit code 1. When `count=0`, `((count++))` evaluates the pre-increment value (0), triggering `set -e` to abort the script.

**Solution**: Replace `((count++))` with `count=$((count + 1))`. The `$((...))` form always returns exit code 0.

**Rule**: Never use `((...))` for arithmetic that might evaluate to 0 under `set -e`. Use `var=$((expr))` assignment form instead.

### [2025-12-15] ${BASH_SOURCE[0]} is empty in zsh

**Context**: Scripts used `${BASH_SOURCE[0]}` to determine their own file path for relative directory resolution

**Problem**: zsh does not populate `BASH_SOURCE`. Scripts that relied on it for path resolution silently got empty strings, breaking relative path calculations.

**Solution**: Use `${BASH_SOURCE[0]:-$0}` which falls back to `$0` (populated by zsh) when `BASH_SOURCE` is empty.

**Rule**: Always use `${BASH_SOURCE[0]:-$0}` when a script needs to know its own path. Test path resolution in both bash and zsh.

### [2026-02-26] Claude Code auto-memory path encoding

**Context**: Needed to symlink auto-memory directories from dotfiles repo to `~/.claude/projects/*/memory/`

**Problem**: Claude Code encodes project paths by replacing all `/` with `-`. The path `/home/manu/Projects/dotfiles` becomes `-home-manu-Projects-dotfiles`. This encoding is deterministic but machine-specific (depends on username and directory structure).

**Solution**: Compute the encoded path dynamically in setup scripts: `printf '%s' "$project_path" | sed 's|/|-|g'`. Symlink from encoded path to vault directory (`~/Projects/knowledge/10_projects/<project>/memory/`).

**Rule**: When symlinking Claude Code internal paths, always compute the encoding at setup time — never hardcode the encoded path. Memory lives in the vault (not in the dotfiles repo) following the Neural Hive protocol: code in git, knowledge in vault.

### [2026-02-26] set -u requires ${1:-} for optional positional parameters

**Context**: Adding `set -euo pipefail` to all standalone scripts. Several scripts used bare `$1` in argument parsing (e.g., `case "$1" in`, `[[ -z "$1" ]]`).

**Problem**: `set -u` treats unset variables as errors. When a script is invoked with no arguments, `$1` is unbound and the script aborts before reaching its usage message. Tests that checked "shows usage with no args" all failed.

**Solution**: Use `${1:-}` (default to empty string) wherever a positional parameter might be unset. For scripts with a `case "$1"` dispatch, assign to a local variable first: `ACTION="${1:-}"` then `case "$ACTION" in`.

**Rule**: When using `set -u`, always guard optional positional parameters with `${1:-}` or `${1:-default}`. The `$#` check (`[ $# -ne 2 ]`) before access is also safe. Only bare `$1` inside a `while [[ $# -gt 0 ]]` loop is inherently safe.

### [2026-02-26] ${VAR:-fallback} pattern for sourced config files

**Context**: Tool versions (Java 21.0.4, Go 1.26.0, etc.) were hardcoded in both `.zshrc` and `.bashrc` — 12 duplicated strings with no single source of truth.

**Problem**: Updating a tool version required editing two files. A missed edit caused silent path mismatches. Also, on a fresh machine before running setup, `versions.conf` might not exist yet, and the shell would fail to set up PATH correctly.

**Solution**: Created `versions.conf` at repo root (KEY=VALUE, no export, no quotes — simultaneously sourceable by bash/zsh and parseable by PowerShell). Shell RCs source it with a guard (`[[ -f ... ]] && . ...`) then use `${JAVA_VERSION:-21.0.4}` to construct paths, providing a safe fallback if the file is missing.

**Rule**: When sourcing external config files that may not exist, always: (1) guard the source with a file-existence check, (2) use `${VAR:-default}` for every variable consumed from the config. This ensures the shell works on fresh machines before setup runs. The config file should use bare `KEY=VALUE` format (no `export`, no quotes) for maximum cross-tool compatibility.

### [2026-02-27] Always edit the repo copy, never the deployed system copy

**Context**: Adding an `obsidian --no-sandbox` alias to `.zshrc` and `.bashrc`

**Problem**: Edited `~/.dotfiles/.zshrc` and `~/.dotfiles/.bashrc` (the deployed system copy) first, instead of `~/Projects/dotfiles/.zshrc` and `~/Projects/dotfiles/.bashrc` (the canonical repo). Changes in `~/.dotfiles/` are not tracked by git and will be overwritten on next sync.

**Solution**: Always edit files in `~/Projects/dotfiles/` (the repo), then commit there. The sync/install script deploys repo → `~/.dotfiles/`.

**Rule**: The canonical source of truth for dotfiles is `~/Projects/dotfiles/`. The `~/.dotfiles/` directory is a deployment target — never edit it directly. Flow: edit repo → commit → sync to system.

### [2026-02-28] grep -c with 0 matches outputs "0" AND exits with code 1

**Context**: Writing `dedup_current_date()` in `knowledge-crystallize.sh`. Used `count=$(grep -c 'pattern' "$file" 2>/dev/null || echo 0)` to count matching lines.

**Problem**: `grep -c` with zero matches prints `"0"` to stdout AND exits with code 1 (not found). The `|| echo 0` then fires, appending a SECOND `"0"` to stdout. Result: `count="0\n0"` — a multi-line string. Downstream `[ "$count" -le 1 ]` fails with "integer expression expected".

**Solution**: Use assignment outside the subshell: `count=$(grep -c 'pattern' "$file" 2>/dev/null) || count=0`. The `$()` captures grep's stdout (`"0"`). If grep exits 1, `|| count=0` overwrites the variable. Either way, `count` is a single clean integer.

**Rule**: Never use `|| echo VALUE` to provide a default for a command that BOTH outputs and exits non-zero on "empty" results (grep -c, wc, etc.). Use `command || var=default` (assignment form) to overwrite after the subshell, not append.

### [2026-02-28] Claude Code path encoding: Linux vs Windows differ

**Context**: Implementing `--all` auto-discovery in `knowledge-crystallize.sh/.ps1`. Both scripts need to decode `~/.claude/projects/<encoded>/` back to real project paths.

**Problem**: The encoding differs by OS. Linux uses `tr '/' '-'` on the full absolute path (leading `/` becomes leading `-`). Windows uses `.Replace('\', '-').Replace(':', '')` which strips the drive colon (no leading dash). A decode strategy that works on one OS breaks on the other.

**Solution**: Two-stage decode in each script: (1) simple character substitution + `Test-Path`/`-d` existence check (handles 95% of cases where dir names have no dashes); (2) filesystem walk under `$HOME`/`$env:USERPROFILE` up to depth 5, encode each directory, compare (handles project names with dashes like `kasa-provisioner`). The walk is O(dirs under HOME, depth ≤ 5) — fast enough in practice.

**Rule**: When round-tripping Claude Code project paths: Linux encodes with leading `-` (from `/`), Windows strips `:` so no leading char. Keep OS-specific encode/decode functions separate. Always test with a project whose name contains a dash.

### [2026-02-28] bash set -e does not exit on [ with integer error when in && chain

**Context**: In `knowledge-crystallize.sh`, `[ "$count" -le 1 ] && return 0` was at the top of `dedup_current_date`. When `count="0\n0"` (bug above), `[` exited with code 2 (error).

**Problem**: Expected `set -euo pipefail` to abort the script on the `[` error. Instead, the script continued. Bash's `set -e` doesn't always trigger on `[` exits with code 2 inside `&&` chains — the `&&` compound absorbs the error differently than a simple command.

**Solution**: Fixed the root cause (`grep -c` bug). Also added `|| log_warning "..."` in `run_all` around `process_project` so one bad project can't kill the batch run.

**Rule**: Don't rely on `set -e` to catch every `[` evaluation error — especially inside `&&`/`||` compounds. Validate that numeric variables are actually integers before using them in arithmetic comparisons. Wrap batch loops with per-item `|| warn` to isolate failures.

### [2026-02-27] Claude Code SessionStart hook for vault health context

**Context**: `vault-health.sh` was created to report Obsidian vault health (orphans, unresolved links, frontmatter coverage) but required manual invocation.

**Problem**: Claude had no automatic awareness of vault health state at session start. Users had to remember to run the script and paste results.

**Solution**: Created `claude-session-start.sh` as a Claude Code `SessionStart` hook. The hook detects if CWD is inside an Obsidian vault (walks up directories looking for `.obsidian/`), runs `vault-health.sh` if found, and returns health summary as `additionalContext` via the hook JSON output format. Registered in `~/.claude/settings.json` under `hooks.SessionStart`.

**Rule**: Claude Code hooks live in `~/.claude/settings.json` (global scope). Scripts they invoke live in dotfiles (`~/.dotfiles/scripts/`). On new machines: (1) deploy dotfiles (gets the script), (2) add the hook entry to `~/.claude/settings.json`. The hook must tolerate Obsidian GUI being down (exit code 2 from vault-health.sh) and non-vault directories (exit 0 silently).

### [2026-03-10] Aider requires Python 3.12 — audioop removed in 3.13

**Context**: Installing aider-chat via `uv tool install` on a system with Python 3.13.

**Problem**: Python 3.13 removed the `audioop` stdlib module. Aider depends on `pydub` which imports `audioop` at module load time. The error manifests as `ModuleNotFoundError: No module named 'audioop'` on any aider command, even `--version`.

**Solution**: Pin Python 3.12 in the install command: `uv tool install --python 3.12 aider-chat`. Both `setup-linux.sh` and `setup-windows.ps1` use this pinned version.

**Rule**: When installing Python tools that depend on deprecated/removed stdlib modules, always pin the Python version explicitly in `uv tool install --python X.Y`. Check release notes for stdlib removals before upgrading the pinned version.

### [2026-03-12] Single-quoted shell strings prevent variable expansion in JSON

**Context**: `setup-linux.sh` built a JSON hook entry with `HOOK_ENTRY='{"command":"$HOME/.dotfiles/scripts/..."}'`. The literal `$HOME` was written into `settings.json`.

**Problem**: Claude Code reads the hook path as-is. The literal string `$HOME/...` is not a valid path, so the SessionStart hook silently failed on every fresh install since it was added.

**Solution**: Replaced string concatenation with `jq -n --arg cmd "$HOME/.dotfiles/scripts/claude-session-start.sh" '{"command":$cmd}'`. Shell expands `$HOME` in the argument, `jq` handles JSON escaping safely.

**Rule**: Never embed shell variables inside single-quoted JSON strings. Use `jq -n --arg` to build JSON with dynamic values — it handles both variable expansion and proper JSON escaping.

### [2026-03-12] grep -c '.' counts 1 on empty input (newline matches dot)

**Context**: `vault-health.sh` counted orphan/dead-end links with `echo "$output" | grep -c '.'`. When output was empty, the count should be 0.

**Problem**: `echo ""` emits a newline character. The regex `.` matches a newline in this pipeline context, so `grep -c '.'` returns 1 instead of 0. Every vault health check reported at least 1 orphan even on a clean vault.

**Solution**: Changed all `grep -c '.'` to `grep -c '[^[:space:]]'` which only counts lines with visible characters.

**Rule**: When counting non-empty lines from command output, use `grep -c '[^[:space:]]'` not `grep -c '.'`. The dot regex matches newlines from `echo` even when the content is empty.

### [2026-03-12] Plaintext secrets must never touch disk — pipe to age directly

**Context**: `secrets_add` and `secrets_rotate` in `load-secrets.sh` wrote the secret value to a plaintext file, encrypted it with age, then deleted the plaintext.

**Problem**: Between write and delete, the secret exists unencrypted on disk. On SSDs with wear leveling or systems with filesystem journaling, `rm` doesn't guarantee data erasure. An interrupted script leaves the plaintext file permanently.

**Solution**: Piped the value directly to age via stdin: `printf '%s' "$value" | age_encrypt "$encrypted_file" "$key_path"`. No temporary plaintext file is ever created.

**Rule**: Never write secrets to disk before encryption. Always pipe directly to the encryption tool's stdin. This eliminates both the crash-window vulnerability and the data-remanence risk.

### [2026-03-12] Uninitialized variable under set -u in conditional-only assignment

**Context**: `claude-session-start.sh` used `VAULT_NAME` which was only assigned inside a vault-detection `if` block.

**Problem**: Under `set -euo pipefail`, referencing `VAULT_NAME` after the conditional would abort the script with "unbound variable" when no vault was detected — the variable was never defined in that code path.

**Solution**: Initialize `VAULT_NAME=""` at the top of the script, before any conditionals. The variable is always bound regardless of which branch executes.

**Rule**: When using `set -u`, any variable assigned inside a conditional block must also be initialized at a wider scope (script top or function top). If only one branch of an `if/else` assigns the variable, the other branch leaves it unbound.

### [2026-03-12] \s is not POSIX — use `[[:space:]]` in bash regex

**Context**: `utils.sh` used `\s` in `[[ =~ ]]` regex patterns inside `load_env_file()` and `debug_print_env()`.

**Problem**: Bash's `[[ =~ ]]` uses POSIX Extended Regular Expressions (ERE), which do not define `\s` as a whitespace shorthand. `\s` is a Perl-Compatible Regular Expression (PCRE) extension. On some bash versions/platforms, `\s` silently fails to match whitespace, causing functions to behave incorrectly without any error message.

**Solution**: Replace all `\s` with the POSIX character class `[[:space:]]`, which is universally supported in ERE and functionally equivalent.

**Rule**: Never use `\s`, `\d`, `\w` or other PCRE shorthands in bash `[[ =~ ]]` or `grep -E` patterns. Use POSIX character classes instead: `[[:space:]]`, `[[:digit:]]`, `[[:alnum:]]`. Only `grep -P` (PCRE mode) supports the shorthand forms.

### [2026-03-12] Stray bare word causes silent set -e abort

**Context**: `github-secrets-manager.sh` had an accidental bare word `tmp` on its own line, immediately before a valid `tmp=$(create_temp_file "ssh_key")` assignment.

**Problem**: Under `set -e`, bash interprets a bare word as a command to execute. `tmp` is not a valid command, so it exits non-zero, and `set -e` aborts the script immediately. The SSH_PRIVATE_KEY_BASE64 branch would always fail silently with no error message — the "command not found" error went to stderr but was easily missed.

**Solution**: Delete the stray bare-word line. The variable assignment on the next line was already correct.

**Rule**: Under `set -e`, any bare word on its own line is treated as a command. A stale variable name from a copy-paste or incomplete edit becomes an invisible script-killer. Always review diffs for orphaned identifiers — they compile silently but crash at runtime.

### [2026-03-16] ShellCheck treats "# shellcheck" comments as directives

**Context**: `setup-linux.sh` had a comment `# shellcheck (shell script linter)` describing the tool being installed.

**Problem**: ShellCheck parses any comment starting with `# shellcheck` as a directive (like `# shellcheck disable=SC2012`). The parenthesized description text is not valid directive syntax, causing SC1073/SC1072 parse errors that halt further checking of the entire file.

**Solution**: Capitalize or rephrase the comment to avoid the `# shellcheck` prefix, e.g., `# ShellCheck (shell script linter)` or `# Install shellcheck`.

**Rule**: Never start a comment with the literal text `# shellcheck` unless it is an actual ShellCheck directive. The tool intercepts any comment matching that prefix. Use capitalization (`# ShellCheck`) or different phrasing to describe the tool in human-readable comments.

### [2026-03-25] Secrets mapping and file inventory must be reconciled automatically

**Context**: `sensitive/` contained 35 encrypted `.secret.age` files but only 17 had entries in `env-mapping.conf`. 14 were app-specific secrets (mlorentedev) that didn't belong in dotfiles at all.

**Problem**: `load-secrets.sh` silently skipped missing files (`file_exists || return 1`) and never checked for orphans. No automated way to detect mapping↔file drift. Secrets accumulated without cleanup.

**Solution**: (1) Added `log_warning` on missing files and orphan detection to `secrets_load()` — runs passively at every shell startup. (2) Added healthcheck section 8/8 that validates every mapping entry has a file and every file has a mapping. (3) Classified secrets: personal cross-machine credentials stay, app-specific envs move to project SOPS.

**Rule**: Any system with a mapping file (config ↔ resource) needs bidirectional reconciliation: mapping→resource (missing?) and resource→mapping (orphan?). Run both checks automatically at load time (passive warning) and in CI/healthcheck (active audit). Silent failure on missing resources is a bug, not a feature.

### [2026-03-18] cp fails when source and destination resolve to the same file via symlink

**Context**: `setup-linux.sh` used `safe_copy` (which wraps `cp`) to deploy `.gitconfig` from `$DOTFILES_DIR/.gitconfig` to `$HOME/.gitconfig`.

**Problem**: `~/.gitconfig` was already a symlink pointing to `~/.dotfiles/.gitconfig` (the same file). `cp` cannot copy a file over a symlink that resolves to the same source — it fails silently or with an error. This broke the "Setting up Git configuration" step on re-runs.

**Solution**: Replaced `safe_copy` with `ln -sf` for `.gitconfig`, consistent with how `.zshrc` and `.profile` are already deployed. `ln -sf` silently replaces any existing symlink. Updated `verify-setup.bats` to assert symlink behavior instead of regular file.

**Rule**: When deploying dotfiles, prefer `ln -sf` over `cp` for idempotency. A symlink can always be replaced atomically, while `cp` fails when the destination is a symlink to the source. Use the same deployment mechanism (symlink) for all managed dotfiles to avoid inconsistent behavior on re-runs.

### [2026-03-26] PSScriptAnalyzer fails on non-ASCII chars outside here-strings

**Context**: Added `-WorkSdk` mode to `init-project.ps1` with em dashes (`—`) in `Write-Host` strings and an arrow (`→`) in a `Write-Success` call. Also added an em dash in a comment line.

**Problem**: PSScriptAnalyzer flags any non-ASCII character that appears in `.ps1` files outside of here-strings (`@"..."@`). The lint-powershell CI job (`Invoke-ScriptAnalyzer -Severity Error,Warning`) caught it and exited 1. Syntax check (test 84) passed because PowerShell parses non-ASCII fine — only PSScriptAnalyzer rejects them.

**Solution**: Replace `—` with `-` and `→` with `->` in string literals, `Write-*` calls, and comments. Non-ASCII inside `@"..."@` here-strings is fine (pre-existing `→` on line 288 was already passing).

**Rule**: In `.ps1` files, keep all non-here-string code (comments, regular quoted strings, `Write-*` calls) to pure ASCII. Em dashes, arrows, and similar typography must live only inside `@"..."@` blocks (template content). When adding display text to a PowerShell script, use ASCII punctuation: `-` not `—`, `->` not `→`.

### [2026-03-29] File deployment requires delete-then-copy, not additive-only copy

**Context**: Skills ecosystem overhaul deleted 9 skill directories from the dotfiles repo. Setup scripts (`setup-linux.sh`, `setup-windows.ps1`) deployed skills by copying source to destination.

**Problem**: The copy loop only added new/updated files — it never removed entries at the destination that no longer existed in the source. After deleting 9 skills from the repo, those stale skill directories persisted at all deployment targets (`~/.claude/skills/`, Gemini prompts dir) indefinitely.

**Solution**: Changed both setup scripts to a three-step sync pattern: (1) enumerate destination entries, (2) delete any entry not present in the source, (3) copy all source entries. Linux uses `basename` + `[ -d ]` checks; Windows uses `$item.BaseName` + `Test-Path`.

**Rule**: Any file deployment pipeline that copies a directory must also handle deletions. Additive-only copy creates ghost artifacts that accumulate silently. The canonical pattern is: enumerate destination → diff against source → delete orphans → copy current. Apply this to skills, configs, prompts, or any mirrored directory.

### [2026-03-25] Config deployment guards vs tool installation guards

**Context**: Made Gemini config deployment conditional on `command -v gemini`. CI integration tests failed because the Docker container doesn't have gemini installed.

**Problem**: Conflated two concerns: (1) installing a tool's dependencies (needs the tool ecosystem present) and (2) deploying config files (just copying markdowns). The guard prevented harmless config from being deployed, breaking CI and also preventing pre-deployment on machines where the tool will be installed later.

**Solution**: Removed the CLI guard from config deployment. Config files are always deployed. Guards remain only around actual tool installation commands (e.g., `gh extension install github/gh-copilot`).

**Rule**: Separate "deploy config" from "install tool". Config file deployment (copying markdown, YAML, JSON) is always safe and should run unconditionally. Only guard commands that install binaries, extensions, or packages. A machine without the CLI benefits from pre-deployed config — it's ready when the tool arrives.

### [2026-05-08] Self-heal third-party plugin breakage at SessionStart

**Context**: `thedotmack/claude-mem` shipped v12.7.4 and v13.0.0 to the marketplace with two independent bugs that prevent the `mcp-search` MCP server and worker from starting on a fresh install: (1) `.mcp.json` embeds `${_R%/}` shell parameter expansion which Claude Code's MCP loader misreads as a missing env var (upstream #2385), and (2) v13.0.0's `bun.lock` and shipped `node_modules/` omit the `zod` dep declared in `package.json`, so the worker crashes with `Cannot find module 'zod/v3'`.

**Problem**: A manual edit of the cached `.mcp.json` plus a manual `npm install zod` fixes both, but `/plugin update` (or any reinstall) wipes the workaround. Documenting the manual steps in a vault troubleshooting note relies on me remembering to re-run them, which violates the "automate, don't instruct" standing order. Pinning to v10.6.3 loses upstream fixes and eventually stops working when the marketplace stops serving the old version. Forking the plugin is heavyweight for two trivial patches.

**Solution**: Encoded both fixes into `scripts/claude-mem-heal.sh` (POSIX `sh`, idempotent: `grep -F '${_R%/}'` to detect bug 1, `[ -d node_modules/zod ]` to detect bug 2; silent on healthy installs). Wired into `claude-session-start.sh` so it runs on every session start before vault detection. Heal output (when something was actually fixed) is surfaced via `additionalContext` so the user sees what was repaired. Iterates over all cached versions plus the marketplace fallback copy. Cost on healthy installs: <50ms (just two filesystem checks per cached version).

**Rule**: When a third-party tool ships a broken artifact and the fix is small and idempotent, encode it as a self-heal script wired into the relevant lifecycle hook (SessionStart, shell init, install verification). Three properties are required: idempotent (re-running on healed state is a no-op), silent on success (don't pollute every session with status spam), and surfacing only when it acts (so the user knows the workaround fired). Document the bug, the fix, and the retire-when criteria in `50-troubleshooting/`. Promote to a `00_meta/patterns/` pattern only after the second occurrence — one instance is a workaround, two is a pattern.


### [2026-05-11] tmux clipboard needs an external bridge — and that bridge is display-server-specific

**Context**: After enabling `set -g mouse on` and `mode-keys vi`, selections inside tmux still did not appear in the system clipboard. `Ctrl+V` outside tmux pasted stale content.

**Problem**: tmux's `copy-pipe` writes to its own internal buffer, not to the OS clipboard. The buffer is only exposed externally if you pipe the selection through an out-of-process tool. That tool is also display-server-specific:

- X11 → `xclip` (or `xsel`)
- Wayland → `wl-copy` (from `wl-clipboard`)
- macOS → `pbcopy` (stdlib, but irrelevant for this Linux-only setup)

`xclip` does not work on a pure Wayland session. `wl-copy` does not work on X11.

**Solution**: Pipe selections to `xclip` via `copy-pipe-and-cancel` and install `xclip` as a system package (added a warning block to `setup-linux.sh`, matching the existing `tmux` pattern — no sudo from the script). Bindings live in `tmux.conf`:

```tmux
bind -T copy-mode-vi y                 send-keys -X copy-pipe-and-cancel 'xclip -selection clipboard -in'
bind -T copy-mode-vi MouseDragEnd1Pane send-keys -X copy-pipe-and-cancel 'xclip -selection clipboard -in'
```

**Rule**: When migrating display server, the clipboard bridge must change in lockstep:
- X11 → keep `xclip`.
- Wayland → swap to `wl-copy` in `tmux.conf` **and** install `wl-clipboard` instead of `xclip`.

Check current server with `echo $XDG_SESSION_TYPE`. Full operational walkthrough lives in [`runbooks/guide-tmux.md`](runbooks/guide-tmux.md) under the "Copy / paste" section.


### [2026-05-11] Editing a dotfile in the repo does not take effect until `setup-linux.sh` runs

**Context**: After editing `tmux.conf` in `~/Projects/dotfiles/` to add clipboard bindings (`copy-pipe-and-cancel` piped to `xclip`), tmux still behaved like the old config. Mouse selection produced nothing in the system clipboard.

**Problem**: The dotfiles repo uses a **two-tier deploy**, not a direct symlink to the repo:

```
~/Projects/dotfiles/<file>   ← canonical, git-tracked (you edit here)
        │ safe_copy in setup-linux.sh
        ▼
~/.dotfiles/<file>           ← deploy-dir intermediate (what the symlink resolves to)
        ▲ ln -sf in setup-linux.sh
~/.<file>                    ← active symlink in $HOME
```

The symlink `~/.tmux.conf → ~/.dotfiles/tmux.conf` was intact, but `~/.dotfiles/tmux.conf` was stale — `grep -c "copy-pipe-and-cancel" ~/.tmux.conf` returned `0` despite the repo file having the bindings. tmux was reading the old version of the file.

This applies to **every** dotfile deployed via `safe_copy` in `setup-linux.sh`: `tmux.conf`, `.gitconfig`, `.zshrc`, `.bashrc`, etc.

**Solution**:

```sh
cd ~/Projects/dotfiles
./setup-linux.sh                  # refreshes ~/.dotfiles/<file> from the repo
tmux source-file ~/.tmux.conf     # for tmux: reload running session
```

For other dotfiles, the appropriate reload command (e.g. `exec zsh`, `source ~/.zshrc`) after the redeploy.

**Rule**: After editing **any** file in `~/Projects/dotfiles/` that's tracked by `setup-linux.sh`, the change is not live until both:
1. `./setup-linux.sh` runs (refreshes the deploy-dir middle layer).
2. The relevant tool reloads its config.

Verification check before claiming the change is live:

```sh
grep -c "<new content>" ~/.<file>   # should match what's in the repo file
```

Cross-ref: [`runbooks/guide-tmux.md`](runbooks/guide-tmux.md) documents the same flow under "How the config gets deployed".

### [2026-05-12] Env-vs-disk drift after secret mutation
**Context:** While diagnosing dotfiles issue #7 (https://github.com/mlorentedev/dotfiles/issues/7) where `secrets_rotate` appeared to silently fail to update the encrypted file. Investigation showed the encrypted .age file WAS being updated correctly on disk (mtime + SHA256 confirmed change after rotate), but every consumer of the secret saw the old value.
**Problem:** load-secrets.sh exports `$VAR` once at shell startup. `secrets_rotate` updated the on-disk .age file but did NOT re-export `$VAR` in the current shell. Any subsequent read of `$VAR` (gh CLI, curl, scripts, even `secrets_show` without `--raw`) returned the cached old value, indistinguishable from a real failure. This wasted ~30min of debugging adding instrumentation to age_encrypt before realizing the encryption was fine. The issue was env-vs-disk drift between shell state and disk state, with no automatic reconciliation. Compounded by: `_secrets_sync_to_repo` silently no-op'd when `DOTFILES_REPO_DIR` was unset, so the user's primary verification step (git status in the repo) could miss real updates. Also the project lacked a `secrets_remove` function, so deletion was a manual three-step (edit mapping + rm .age + manual sync) that bypassed audit logging.
**Solution:** After any mutating secret operation (add/rotate/remove/add_file), auto-update the current shell so `$VAR` matches disk immediately — eliminate the manual `secrets_refresh` step that users forget. Concretely: rotate/add do `export_var "$var" "$value"`; remove does `unset_var`; add_file calls `secrets_refresh` (file deployment is non-trivial). Make `_secrets_sync_to_repo` warn loudly to stderr when no repo is resolved, and add an auto-detect default of `$HOME/Projects/dotfiles` when `DOTFILES_REPO_DIR` is unset. Add a missing `secrets_remove VAR_NAME [--yes]` that handles plain + file secrets, syncs deletions to repo, and updates audit log. Generalizes to: any system where the source of truth (disk) and the working copy (env, cache, deployed file) can diverge MUST either auto-reconcile after mutation OR loudly warn — silent drift is the worst failure mode because it produces false bug reports and wastes investigation time.
**Tags:** `#secrets` `#shell` `#env-drift` `#ssot` `#false-positive-debugging`

### [2026-05-12] git log --pretty=format: drops last commit silently in while-read pipelines
**Context:** Building scripts/changelog-gen.sh to bucket commits by Conventional Commit type. Read `git log --no-merges --pretty=format:'%cs|%h|%s'` via `while IFS='|' read -r date hash subject; do ...; done < <(git log ...)`. All commits except the very oldest one made it into the output. Tests caught it: the assertion that "feat(core): initial commit" appeared under "## Features" failed.</context>
<parameter name="problem">`git log --pretty=format:'...'` writes the format string between commits but does NOT terminate the LAST commit with a newline (matches the docstring of `format:` in `git log` man page). When piped into a `while read` loop, `read` returns non-zero on the unterminated final line, so the loop exits BEFORE processing it. Result: the oldest commit is silently dropped. Easy to miss — the output looks correct, just one entry shorter.</problem>
<parameter name="solution">Use `--pretty=tformat:'...'` instead of `--pretty=format:'...'`. The `t` prefix means "terminator" — git appends a newline after every commit, including the last. The `while read` loop then sees the newline and processes the final entry. Alternatively, use the canonical idiom `while read || [[ -n "$line" ]]; do ...; done` to consume the unterminated line, but `tformat:` is cleaner because it fixes the data, not the consumer. Generalizes: any `while read` loop fed by a tool with optional trailing newlines (printf, awk, custom scripts) needs either the `|| [[ -n "$var" ]]` guard or the producer to always terminate. When in doubt, write a test that asserts the FIRST and LAST records make it through.</solution>
<parameter name="tags">["bash", "git", "shell-pipelines", "off-by-one", "test-discovery"]

### [2026-05-15] MCP transport state and daemon state can disagree per-conversation

**Context:** Mid-session on a fresh Windows 11 laptop, after rejecting the very first `mcp__hive__session_briefing` call in the permission prompt. Every subsequent Hive tool call returned `MCP error -32000: Connection closed`, then `No such tool available`. Spent a few minutes assuming the Hive server had crashed.

**Problem:** `claude mcp list` in a separate terminal reported `hive: uvx hive-vault - ✓ Connected`. The daemon process was healthy; only the current conversation's handle to it was dead. There is no in-session command to re-attach. Filed as [mlorentedev/hive#75](https://github.com/mlorentedev/hive/issues/75); see also [`troubleshooting/hive-mcp-rejection-disconnect.md`](troubleshooting/hive-mcp-rejection-disconnect.md).

**Solution:** Two layers. Operationally: always accept the first MCP tool call in a fresh conversation; you can deny later ones safely. If the transport is already poisoned, restart the conversation — that is the only recovery. Diagnostically: when an MCP-backed tool stops working mid-session, the very first signal to capture is `claude mcp list` in a separate terminal. If the daemon is `✓ Connected` but the conversation cannot reach the tools, you have a session-state vs daemon-state divergence — not a server crash, not a config error, just a per-conversation transport handle that Claude Code does not recover on its own. The fallback is filesystem reads of the vault for the rest of the session.

**Rule:** Treat "MCP server appears dead" and "MCP server actually died" as two different failures and check them separately. The daemon-side check (`claude mcp list`) is free, takes seconds, and decides between "restart conversation" and "actually investigate the server". Doing this check first reframes the rest of the debugging — and prevents 20 minutes of poking at a healthy server.

**Tags:** `#mcp` `#claude-code` `#hive` `#transport-state` `#diagnostic-first-move`

### [2026-05-15] bash `IFS=$'\t' read` collapses consecutive tabs (whitespace IFS chars never preserve empty fields)

**Context:** Building `scripts/doctor.sh` to iterate `env-contract.json` entries. Used jq's `@tsv` to emit one TSV record per env-var, with `(.required_on // "")` for an optional column, then parsed with `while IFS=$'\t' read -r name required required_on default validation`. On entries where `required_on` was empty (most of them), every subsequent column shifted by one — `$default` got the validation value, `$validation` got an empty string, and the script ran silently wrong for ~10 minutes before a `bash -x` trace pinpointed it. Raw TSV had the right columns (`cat -A` confirmed); the read loop was eating them.

**Problem:** Bash's `read` treats *whitespace* IFS characters specially: even when you explicitly set `IFS=$'\t'`, consecutive tabs are collapsed into a single delimiter (the same rule applies to space and newline). So a TSV row like `name<TAB>true<TAB><EMPTY><TAB>$HOME/.dotfiles<TAB>path_exists` is read as if the empty third column did not exist, shifting every later assignment by one slot. POSIX behaviour, deeply confusing in practice because the documentation calls IFS "the field separator" and a quick reader assumes "tab means tab". The bug is silent: the script never errors, it just operates on wrong data.

**Solution:** Use a *non-whitespace* delimiter for TSV-like output whenever any column can be empty. Switched jq to `... | join("|")` and bash to `IFS='|' read -r ...`. Non-whitespace IFS chars do NOT collapse, so empty fields are preserved exactly. Pipe is safe for our values (paths, version strings, regex patterns) but for arbitrary content prefer the ASCII Unit Separator `$'\x1f'` — chosen by ASCII itself for this purpose, guaranteed absent from any sane string. Either way, **never use `IFS=$'\t'` (or `' '` or `$'\n'`) for `read` if an empty field is even possible**.

**Rule:** When a `read` loop must preserve empty fields, the IFS character must be non-whitespace (`|`, `;`, `:`, or `$'\x1f'`). For jq pipelines: replace `@tsv` with `[...] | join("|")`. For ad-hoc shell: never assume `IFS=$'\t' read` round-trips a TSV with empty columns — it doesn't, and the failure is invisible.

**Tags:** `#bash` `#ifs` `#read` `#tsv` `#silent-failure` `#jq` `#posix-gotcha` 

### [2026-05-16] PSScriptAnalyzer fails on non-ASCII in .ps1 without BOM
**Context:** Editing PowerShell scripts (`setup-windows.ps1`, `powershell/profile.ps1`, `scripts/*.ps1`) in this repo. CI runs PSScriptAnalyzer in the `lint-powershell` job with default rules.
**Problem:** Any non-ASCII character (em dash `—`, en dash `–`, arrows `→`, smart quotes `"" ''`, ellipsis `…`) in a `.ps1` file without a BOM trips the rule `PSUseBOMForUnicodeEncodedFile`, fails the lint-powershell CI job, and blocks the PR. Hit twice in two months: commit `464eecf` (Mar 2026, em dash in `setup-windows.ps1`) and commit `9d284b9` (May 2026 PR #36, em dash in `powershell/profile.ps1`). The bug surfaces only in CI — local edits and grep look fine, the byte sequence is just multi-byte UTF-8.</problem>
<parameter name="solution">Project policy: **ASCII-only in `.ps1` files; do not add a BOM**. Substitutions when writing/editing: em dash -> `--`, arrows -> `->`, smart quotes -> ASCII `'` `"`, ellipsis -> `...`. Pre-commit safety net: `grep -nP '[^\x00-\x7F]' powershell/*.ps1 scripts/*.ps1 setup-windows.ps1` must return zero hits. Comments in `.ps1` are as constrained as code — an em dash in a comment block is enough to fail CI. Note `.sh`, `.md`, and vault files are NOT subject to this constraint.</solution>
<parameter name="tags">["powershell", "ci", "lint", "psscriptanalyzer", "encoding"]
**Solution:** 

### [2026-05-18] Verify post-checks with hardcoded strings rot when the verified file is refactored
**Context:** setup-windows.ps1 had Select-String post-checks (lines 142 and 466) grepping for the literal "CORE PRINCIPLE" in deployed CLAUDE.md/GEMINI.md to confirm the AI-013 pointer-style refactor (2026-05-16) landed correctly. AI-013 actually replaced that content with pointers starting "First, read `AGENTS.md`" -- the string "CORE PRINCIPLE" no longer existed in any deployed file. Setup kept emitting two spurious [ERROR] lines on every run despite the actual Copy-Item succeeding. Discovered empirically on 2026-05-18 during WIN-003 validation re-run.</context>
<problem>Hardcoded verify strings create an invisible coupling between two files (the deploy source and the verifier). When the deploy source is refactored, the verifier silently lies: deploy succeeds (Copy-Item ran), post-check says "failed" (string not found), and the script appears partially broken. Same class as BUG-001 (copilot-instructions verify, fixed in PR #40 with the exact same pattern fix). Both bugs lived in main for weeks -- they were only caught by empirical re-run of setup on a clean machine where someone read the output carefully.</problem>
<solution>Use a durable marker tied to the *file format convention*, not arbitrary body content. For pointer-style files the convention is the first-line marker `'First, read \`AGENTS.md\`'`. Match it with `-SimpleMatch` (PowerShell Select-String) or `grep -F` (POSIX) to avoid regex interpretation of backticks. The marker survives content refactors because it IS the convention, not an arbitrary string in the content. setup-linux.sh already had this; setup-windows.ps1 lagged in 2 places (BUG-002, fixed PR #47). Lock the parity in tests/setup-windows.bats with cross-OS asserts so future drift fails CI not production.</solution>
<tags>["bash", "powershell", "verification", "setup-scripts", "refactor-drift", "parity", "silent-failure"]</tags>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-18] detect-and-act scripts go silently inert when upstream products change their surface
**Context:** dotfiles setup-windows.ps1 and setup-linux.sh detected GitHub Copilot CLI via `gh extension list | grep github/gh-copilot`. GitHub released a new standalone `copilot` CLI (winget GitHub.Copilot, agentic interface closer to Claude Code than to the legacy suggest/explain wrappers). On machines with the new v2 installed and operative, the script logged "extension not installed, skipping" and never deployed ~/.copilot/copilot-instructions.md. The AI-013 pointer-style refactor was functionally inert on those machines. Discovered 2026-05-18 only when the user typed `gh extension install github/gh-copilot` and got "matches built-in alias" error -- revealing both products existed.</context>
<problem>Detect-and-act scripts assume the upstream product's surface stays stable. When the vendor ships a new product with the same conceptual purpose but different binary/CLI/install path, the detection silently misses it. Two failure modes: (1) deploy never fires (config missing); (2) the script LOGS "success" because the detect branch was simply skipped (no fail, no warn -- log says "not installed, skipping"). The longer the gap between vendor change and detection update, the more machines drift silently.</problem>
<solution>Three layers. (1) Detect on the BINARY not on a package-manager extension list: `Get-Command copilot` / `command -v copilot`. Binaries are more stable as a surface than extension manager state. (2) Re-audit upstream products at least once per sprint -- vendor changes are frequent; quarterly audit is too slow. Add to sprint checklists: "any AI/dev-tool integration deployed in past 6 months -- verify upstream hasn't shipped a replacement product." (3) Annotate every detect-and-act block with an inline comment: `# Upstream: https://...  Last validated: YYYY-MM-DD`. Forces the re-audit habit when the comment goes stale.</solution>
**Tags:** `#setup-scripts` `#detect-and-act` `#upstream-drift` `#copilot` `#silent-failure`

### [2026-05-18] When an invariant changes, dead code emerges silently downstream

**Context:** SDD-001 (PR #49) added an unconditional `[sdd]` reminder to `$ContextLines` / `CONTEXT_LINES` at the start of both `claude-session-start.{ps1,sh}`. The new invariant: the context buffer is NEVER empty. Two pre-existing branches gated on the OLD invariant (`if (-not $VaultRoot -and -not $ContextLines) { exit 0 }` and the bash equivalent) became unreachable code -- they fired the `exit 0` branch only when both were empty, which can no longer happen. Also: the claude-mem heal block's `$ContextLines = "..."` overwrite (instead of append) would have wiped the new reminder when heal output existed.

**Problem:** Dead code born from an invariant change is silently broken in a way that compiles and runs but does the wrong thing. Three failure modes: (1) the dead branch is never executed (silent waste), (2) the dead branch IS executed and produces incorrect behavior because its precondition is now impossible (rare, but happens when state is computed by side-effects), (3) downstream blocks that share state with the upstream invariant get clobbered (the claude-mem overwrite case). The bug surfaces only on the next refactor or in production when a corner case finally triggers the dead path.

**Solution:** When changing an invariant (especially one as fundamental as "this buffer is always non-empty"), do a Pre-Flight Audit per the AGENTS.md Socratic Guardrail: grep the entire file for references to the OLD invariant's preconditions (in this case, `(-not $ContextLines)` and `[ -z "$CONTEXT_LINES" ]`) and decide for each: (a) is this block now unreachable? remove it explicitly; (b) does this block read the buffer state and act on it? verify the new invariant doesn't break it; (c) does this block WRITE to the buffer? change overwrite to defensive append. Document the invariant change in a comment at the original assignment site so future readers see "this was made invariant by SDD-001 -- downstream blocks rely on it being non-empty".

**Rule:** Invariant changes are interface changes in disguise. Audit every reader AND writer of the affected state. A grep for the OLD invariant's predicate is the cheap mechanical step that catches the dead-code class. Skipping the audit = future-you debugs a 5-line silent regression for 30 minutes.

**Tags:** `#refactor` `#invariants` `#dead-code` `#pre-flight-audit` `#shell-state`

### [2026-05-18] Bulk-copy operations collide silently with per-file deploy logic

**Context:** SDD-002 (PR #51) introduced a per-file deploy for `ai/claude/settings.json`: read template, substitute `__HOOK_COMMAND__` placeholder, merge with existing target using per-key policy. Both setup scripts also had a pre-existing bulk-copy `Copy-Item ai/claude/* ~/.claude/` (PowerShell) / `cp -rf ai/claude/* ~/.claude/` (bash) that would copy ALL files from the source dir, including the new `settings.json` template -- which contained the literal `__HOOK_COMMAND__` placeholder AND would have wiped the user's customizations.

**Problem:** The bulk-copy + per-file collision is invisible until the per-file logic introduces a placeholder or merge invariant that the bulk-copy can't honor. While both deploys produce "a file at the target", the bulk-copy version produces semantically wrong content (placeholder unsubstituted, user customizations lost). No error fires because both operations succeed at the filesystem level. The bug surfaces only when the user opens the deployed file and finds garbage, OR when downstream logic chokes on the placeholder.

**Solution:** When introducing per-file deploy semantics for a file previously covered by a bulk-copy, ALWAYS add an explicit exclusion to the bulk-copy at the same time. PowerShell: `Copy-Item ... -Exclude 'settings.json'`. POSIX bash: explicit loop with `[ "$(basename "$src")" = "settings.json" ] && continue`. Document the exclusion next to the bulk-copy with the reason ("handled by per-file logic in <function>"). A bats parity assert that grep-checks for the exclusion locks it in.

**Rule:** Per-file deploy + bulk-copy of the parent dir is a guaranteed collision. The first-PR-after-refactor often misses it because the symptom is "file exists at target, looks OK on glance". Pair every new per-file deploy with the exclusion edit in the same atomic PR. Generalizes beyond settings.json: skills, MCP configs, opencode commands, anywhere a curated subset of a directory needs per-file handling within an otherwise-bulk deploy.

**Tags:** `#setup-scripts` `#deploy` `#bulk-copy` `#collision` `#parity` `#shell` `#powershell`

### [2026-05-19] Defensive monitors are not fixes — trigger fix and monitor are siblings, not substitutes

**Context:** dotfiles#33 was the "original" fix for the upstream `anthropics/claude-code#59870` truncation bug — every `claude plugin install` call rewrites `~/.claude/.claude.json` and silently drops subscription metadata (organizationType, organizationRateLimitTier, projects map, onboarding flags), shrinking the file from ~75 KB to ~1.5 KB and forcing re-authentication. The "fix" was an idempotence guard: don't call install if the plugin already appears in `claude plugin list` output. Six months later SDD-021 (2026-05-18) added a session-start canary (`Test-ClaudeJsonSize` in `claude-session-start.{sh,ps1}`) that flags the symptom if it ever recurs, with a 10 KB threshold. Today (2026-05-19) I noticed the canary firing on every session: file at 3444 bytes, re-login prompt in every project.

**Problem:** The idempotence guard had a false negative for `claude-mem@thedotmack` — it never appears in `claude plugin list` output (different marketplace, `@thedotmack` vs `@claude-plugins-official`), so the literal-match check `grep -qF "claude-mem@thedotmack"` returns false on every setup run, triggering one real install call and one real truncation. The SDD-021 canary CORRECTLY surfaced this — the warning text was in the session-start additionalContext I was reading at boot — but the canary is a detector, not a preventer. I had been blaming the recurrence on something I had "just broken" instead of recognising the canary was doing its job and the trigger fix from dotfiles#33 was incomplete.

**Solution:** Add a snapshot/restore wrapper around the install call (BUG-004 / PR pending) as a defense-in-depth layer beneath the existing idempotence guard. The wrapper snapshots `.claude.json` to a tempfile before the install, restores from snapshot in `finally` iff the post-call size dropped >50% from a baseline ≥10 KB (same threshold as SDD-021's canary — single SSOT for "anomalously small"). Now there are THREE layers that have to fail for the user to see re-login: (1) idempotence guard catches the common case, (2) wrapper catches the false-negative case, (3) canary alarms at next session start if both fail.

**Rule:** When you ship a monitor for a bug you "fixed", the monitor firing is evidence the fix was incomplete, not evidence the fix was undone. Before assuming "I broke it again", check whether the monitor was designed for the exact failure mode now appearing. Three-layer thinking: prevention (the trigger fix), detection (the monitor), recovery (the auto-restore). Each layer guards against the others failing. The presence of a monitor does NOT discharge the obligation to find the residual trigger — it just gives you a finite time window before the bug bites the user.

**Tags:** `#defense-in-depth` `#monitoring` `#claude-cli` `#setup-scripts` `#three-layer-thinking` `#upstream-bug`

### [2026-05-19] Wide try/catch misclassifies the error and misleads the next reader

**Context:** SDD-002 (PR #51) wrapped `ConvertFrom-Json -AsHashtable` in `Merge-ClaudeSettings` with `try { ... } catch { Write-Warn "Claude settings template is not valid JSON after placeholder substitution: $_"; return }`. Under Windows PowerShell 5.1 (the default `PowerShell` interpreter on Windows), `-AsHashtable` does not exist — it was added in PowerShell 7.0. The actual exception thrown is `ParameterBindingException` ("A parameter cannot be found that matches parameter name 'AsHashtable'"), NOT a JSON parse error.

**Problem:** The catch was wide and the user-facing message anchored on "not valid JSON". For a debugging human reading the warning, the natural next step is to inspect the template file for JSON syntax errors — which are nonexistent. The actual root cause (PS version mismatch) is buried in the trailing `$_` interpolation that most observers skip. This was the entire surface area of BUG-005: not the missing parameter, but the misleading log line that prevented someone from finding the missing parameter for hours.

**Solution:** Two layers: (1) at the FIX site, replaced the catch by an explicit version check at script entry that re-execs under pwsh (BUG-005 / PR #58). (2) At the PATTERN site, catches should be either narrow (`catch [System.Management.Automation.ParameterBindingException]` for the specific case) OR the user-facing message should NOT assert a cause it cannot verify (write "Claude settings merge failed" + the raw `$_` exception type, not "is not valid JSON"). Better still: use `Test-Json` to check structure before parsing, so a JSON failure is a separate code path from any other parse-pipeline failure.

**Rule:** A wide `catch { ... }` is fine for "swallow and continue"; it is NOT fine when paired with a user-facing message that asserts a cause. If the message names a cause, the catch must be narrow enough that the cause is the only possibility. Otherwise the error message becomes a lie that costs more debugging time than no message at all. Auxiliary lesson: when a script must run under a newer runtime (PS 7+, Python 3.12+, Node 22+), the cheapest portability fix is to detect-and-reexec at the front door (single point of policy) rather than to backfill compat into every helper (N points, easy to miss one).

**Tags:** `#error-handling` `#powershell` `#portability` `#log-quality` `#wide-catch-trap` `#auto-reexec`


### [2026-05-19] Incident → guard pattern (red-team thyself)
**Context:** During a single 2026-05-19 session, Hive's vault_patch MCP wrote the literal 2-character sequence backslash-n into dotfiles/11-tasks.md four separate times instead of interpreting it as a newline. Each occurrence corrupted a markdown bullet list by merging two items into one physical line — invisible in rendered markdown but breaking init-spec.sh's vault-gate grep (which anchors on `^- [ ] **<id>**`) and any downstream line-based parser. The user surfaced the meta-issue: "la red de seguridad tiene que ir mejorándose a sí misma" — the safety net must keep improving itself.</context>
<parameter name="problem">Each corruption was fixed manually with the Edit tool. No guard was added until the 4th occurrence. By then, the bug class had already burned ~10 minutes of cumulative friction. The general failure: when a bug class hits, we tend to fix the immediate symptom and move on instead of adding a CI assertion / health check / parity test that prevents the next occurrence. Three sibling instances in the same session reinforce the pattern: (1) AI-019 missed `.github/copilot-instructions.md` Model Tier section — fixed in SDD-005 with `tests/docs-drift.bats`; (2) BUG-001 + BUG-002 verify-string drift between setup-linux.sh and setup-windows.ps1 — fixed earlier in PR #40 + #47 with bats parity asserts; (3) Hive vault_patch literal `[BS-n]` — fixed in SDD-006 with `scripts/check-md-escapes.sh` + bats. All three are the same meta-pattern: a class of failure recurs because each occurrence was patched without adding the structural guard.
**Problem:** 
**Solution:** **General rule (incident → guard, red-team thyself):** every bug class encountered MUST emit a CI assertion or health check in the SAME PR that fixes the symptom. Three signals that the rule is being violated: (a) you hit the same bug class twice in one session, (b) the fix is "I'll edit the file manually" with no test added, (c) the PR body says "I'll add the guard later". When you hit a bug class for the 2nd time, STOP. Don't fix the symptom — add the guard. The guard prevents the 3rd through Nth occurrence. **Concrete artefacts shipped under this rule in dotfiles:** `tests/docs-drift.bats` (mirror-file parity, SDD-005), `tests/check-md-escapes.bats` + `scripts/check-md-escapes.sh` (vault_patch corruption, SDD-006), `tests/setup-windows.bats` parity asserts (verify-string drift, BUG-002). **Pre-promotion to 00_meta/patterns/:** the pattern is dotfiles-specific for now; promote to a global pattern when a second project applies it.
**Tags:** `#safety-net` `#ci` `#testing` `#vault-patch` `#incident-driven` `#meta-pattern`

### [2026-05-19] Filename glob *.lock does NOT match package-lock.json (basename matters)
**Context:** Building scripts/check-spec-gate.sh for SDD-003: the LOC-exclusion list needed to skip lockfiles. Initial regex used a single glob pattern `*.lock` in a bash `case` statement. A bats test for `package-lock.json` failed because the file ends in `.json`, not `.lock` — the npm convention puts the lock-marker in the middle of the name, not the suffix.</context>
<problem>The naive pattern `*.lock` only matches files whose filename ends in literal `.lock`. It matches `Cargo.lock`, `poetry.lock`, `Pipfile.lock`, `yarn.lock`, `Gemfile.lock` — all the suffix-style conventions. It does NOT match npm's `package-lock.json`, pnpm's `pnpm-lock.yaml`, or Go's `go.sum`. Lockfile filtering with `*.lock` alone produces silent false negatives — exactly the kind of bug that ships green and breaks in production.</problem>
<parameter name="solution">Use basename-aware matching with explicit literals for non-suffix conventions. Pattern from scripts/check-spec-gate.sh:

```bash
_excluded() {
    local path="$1"
    local base="${path##*/}"
    case "$path" in
        tests/*|specs/archive/*) return 0 ;;
        *generated*) return 0 ;;
    esac
    case "$base" in
        *.lock|*.lockb) return 0 ;;
        package-lock.json|pnpm-lock.yaml|go.sum) return 0 ;;
        .gitignore|CHANGELOG.md) return 0 ;;
    esac
    return 1
}
```

Two `case` blocks: first checks path prefixes (tests/, specs/archive/) and substring matches (*generated*); second checks basename for filename conventions. ${path##*/} is bash parameter expansion for basename — no external `basename` call needed.

General rule: when filtering filenames by convention, list the literal exceptions BEFORE relying on a suffix glob. Suffix globs are a lower bound, not a complete set.</parameter>
<parameter name="tags">["bash", "globs", "lockfiles", "ci", "spec-gate"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-19] Numeric bats threshold drift is invisible — comment the bump inline
**Context:** AI-019 (model-tier policy) added a Model Tier subsection to ai/claude/CLAUDE.md, pushing the file from 70 to 78 lines. The existing bats assertion `wc -l < CLAUDE.md -le 70` started failing. Two options: compact existing content to fit under 70, or bump the threshold.</context>
<problem>If you silently bump a numeric threshold in a test (70 → 80 lines, 50 → 100 tests, etc.) without leaving a trace, the next contributor sees the new number and has no way to know whether (a) the threshold is calibrated to real constraints, or (b) it was raised to accommodate scope creep that should have been resisted. Threshold drift is invisible — every bump compounds; six months later, the assertion has become meaningless rubber. The classic "boiled frog" failure mode.</problem>
<parameter name="solution">When raising any numeric threshold in a bats test (or any CI assertion), add an inline comment in the SAME line/block stating: which spec/PR caused the bump, by how much, and the justification. Example from tests/opencode.bats line 148:

```bash
@test "ai/claude/CLAUDE.md is a pointer to AGENTS.md (≤ 80 lines)" {
    # Threshold bumped 70→80 in AI-019 (model-tier overlay added ~8 lines).
    # Future per-agent extensions should justify each bump in the spec.
    grep -q "First, read \`AGENTS.md\`" "$DOTFILES_DIR/ai/claude/CLAUDE.md"
    [[ $(wc -l < "$DOTFILES_DIR/ai/claude/CLAUDE.md") -le 80 ]]
}
```

Now any future contributor reading the test sees: previous threshold, new threshold, what caused the change, and the implicit rule for next-bump justification. The comment also makes the audit trail visible to `git blame` so PR review can challenge unjustified bumps. Apply this to ALL thresholds — function-length linters, coverage minimums, perf budgets, file-size caps.</parameter>
<parameter name="tags">["testing", "bats", "thresholds", "code-review"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-19] "chore: close spec lifecycle" pattern — for features that shipped piecemeal before archive
**Context:** TERM-001-ghostty-bootstrap had its proposal scaffolded on 2026-05-17 but the implementation shipped piecemeal across PR #38 (tmux truecolor) + commit b00353e (full ghostty bootstrap) + commit 7424731 (config translation) before the spec lifecycle was formally closed. By 2026-05-19 the feature was 100% live on main with bats green, but the spec folder still sat in specs/ (not archive/) with tasks.md as a skeleton.</context>
<problem>SDD-001's archive criterion ("move folder to specs/archive/ on merge") is straightforward when a single PR ships the feature. It's awkward when implementation lands across multiple commits over multiple days — there's no single "merge" event to trigger archival, and tasks.md / verification.md don't get filled because the work was done. The risk: spec folders accumulate in active state indefinitely after the feature is shipped, polluting `ls specs/` and breaking the "active spec = WIP" invariant that the spec-gate CI relies on.</problem>
<parameter name="solution">Apply the "chore: close spec lifecycle" pattern: a small atomic PR that introduces ZERO production code changes and only does the archive housekeeping. Three artefacts in this PR: (1) fill tasks.md retroactively as a map from existing artefacts to the original TDD plan; (2) fill verification.md with the evidence map (commit hashes for each AC + bats output); (3) features.json with the harness-facing contract; (4) git mv specs/<id>/ → specs/archive/<id>/.

Example: PR #66 for TERM-001 — branched as `chore/TERM-001-close-spec-lifecycle`, single commit `chore(spec): close TERM-001-ghostty-bootstrap lifecycle`, 6 files changed (3 archive moves + 3 spec-folder additions), 160 insertions / 98 deletions, zero production code.

The pattern is NOT a workaround for SDD discipline. The proposal WAS filled before implementation (2026-05-17). The pattern is the final step that turns "shipped artefacts" into "archived spec + audit trail". The risk to guard against: scope creep — be strict about ZERO production code in close-out PRs.

The PR title prefix `chore(spec):` makes the intent obvious at review time.</parameter>
<parameter name="tags">["sdd", "spec-lifecycle", "housekeeping", "atomic-prs"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-19] JSONC native // comments beat _commentKey JSON convention for documentation
**Context:** AI-019 needed to document the model-tier mapping inside ai/opencode/opencode.jsonc. The proposal weighed two conventions: (a) a `_modelTierComment` JSON key with underscore prefix (convention says parsers ignore it); (b) native JSONC `//` line comments. OpenCode reads the file as JSONC (the file already used `//` for the schema URL + 6 other comment blocks).</context>
<problem>Documenting structured config inside JSON has no native syntax — comments aren't part of the JSON spec. Many projects invent the `_commentKey` convention (`"_comment": "..."`, `"_doc": "..."`). It works in practice because consumers ignore unknown keys, but it has two downsides: (1) it pollutes the parsed JSON namespace, so any tooling that enumerates keys sees noise; (2) underscore-key convention is unofficial — a future JSON schema validator might reject it. JSONC (JSON with Comments) is a different file format where `//` and `/* */` are first-class syntax.</problem>
<parameter name="solution">When the file is `.jsonc` (or any consumer accepts JSONC), prefer native `//` comments over `_commentKey` JSON keys. Rationale: (a) comments stay out of the parsed key namespace; (b) diff-friendly (line comments don't bracket-wrap content); (c) consistent with existing convention in that file. Example from ai/opencode/opencode.jsonc head:

```jsonc
// OpenCode configuration — managed by dotfiles.
// SSOT: ai/opencode/opencode.jsonc (this file).
// Schema: https://opencode.ai/config.json
//
// Model Tier (per AGENTS.md "Model Selection"):
//   Top: opencode-go/deepseek-v4-pro
//   Mid: opencode-go/qwen3.6-plus
//   Low: opencode-go/deepseek-v4-flash
{
  "$schema": "https://opencode.ai/config.json",
  ...
}
```

Risk mitigation: if the consumer ever rejects line comments (unlikely for any tool that accepts the .jsonc extension), fall back to a sibling Markdown file (e.g. ai/opencode/MODEL_TIERS.md) — explicit documentation file, no convention games inside the data.

Inverse: if the file is pure .json (no JSONC support), use a top-of-file `_comment: ` key as a least-bad option. The pollution is real but acceptable when the alternative is no documentation at all.</parameter>
<parameter name="tags">["json", "jsonc", "config", "documentation"]</parameter>
</invoke>
**Problem:** 
**Solution:** 

### [2026-05-20] Audit ALL call sites of a vulnerable upstream API when guarding one
**Context:** BUG-004 (PR #57, 2026-05-19) wrapped `claude plugin install` with snapshot/restore against upstream truncation bug `anthropics/claude-code#59870`. One week later (2026-05-20) the user reported `.claude.json` truncated AGAIN after a setup-windows.ps1 run. Diagnosis revealed BUG-004 missed ~9 sibling call sites that go through the same vulnerable deserialize-modify-serialize path: `claude mcp get`, `claude mcp add` (each ~9 times per setup), and `claude plugin list`. BUG-011 (PR #69) had to wrap all of them belatedly.</context>
<parameter name="problem">Fixing one symptom of a CLI-level vulnerability without enumerating siblings creates a guaranteed recurrence. The cost of audit-all-call-sites is small (~30 min grep + wrap); the cost of skipping it is a recurring incident class one week later.
**Problem:** 
**Solution:** When patching ANY guard around a vulnerable upstream API call, MUST in the same PR: (1) grep both setup scripts (sh + ps1) for every invocation of the vulnerable binary/subcommand family; (2) wrap each invocation with the same guard; (3) add bats parity asserts that fail CI if a future call site lands without the guard. The audit step is non-negotiable; "wrap one and move on" is a smell.
**Tags:** `#incident` `#guard-pattern` `#audit-discipline` `#claude-cli` `#cross-os-parity`

### [2026-05-20] Claude Code marketplace dir naming follows GitHub repo, NOT declared name field
**Context:** BUG-012 (PR #70, 2026-05-20). Diagnosing `UserPromptSubmit operation blocked by hook` on Windows. Hook command was claude-mem plugin's `bun-runner.js` discovery script. It searched at `~/.claude/plugins/marketplaces/thedotmack/plugin/scripts/` (the marketplace's declared `name` from `marketplace.json`), but Claude Code had cloned the marketplace under `~/.claude/plugins/marketplaces/thedotmack-claude-mem/` (GitHub repo name `thedotmack/claude-mem` flattened with `-`). When `CLAUDE_PLUGIN_ROOT` is unset/stale (Windows: cache `plugins/cache/thedotmack/claude-mem/` stays empty post-install), the hook falls through to the broken fallback → `exit 1` → blocked.</context>
<parameter name="problem">A plugin can declare `name: "thedotmack"` in `marketplace.json` but Claude Code names the install dir after the GitHub repo (`<owner>-<repo>` in some cases, just `<repo>` in others — naming logic not fully documented). Plugins that hardcode `marketplaces/<declared-name>/plugin/scripts/...` in fallback paths break silently on machines where the env-var path fails and the fallback is consulted. Symptom on Windows is `printf: write error: Permission denied` (Git Bash + Claude Code hook subprocess sandbox quirk); on Linux it's a cleaner `claude-mem: plugin scripts not found`. The cross-OS root cause is identical.
**Problem:** 
**Solution:** Defense in depth: create a junction (Windows, no admin) / symbolic link (Linux) from the declared name to the actual install dir during heal. Guard creation on `source exists AND target absent` for idempotence. Once the link exists, ALL code that hardcodes the declared name (plugin's bundled hooks.json, .mcp.json, etc.) resolves correctly without modifying upstream files. The link survives `/plugin update` operations. Pattern lives in `scripts/claude-mem-heal.{sh,ps1}` as `ensure_marketplace_compat_symlink` / `Repair-MarketplaceCompatJunction`.
**Tags:** `#claude-code` `#plugin-discovery` `#claude-mem` `#cross-os-parity` `#junction-pattern`

### [2026-05-20] vault_patch timeout != patch not applied
**Context:** During WIN-001 (PR #71) and the post-merge vault tick, two `mcp__hive__vault_patch` calls to `10_projects/dotfiles/11-tasks.md` returned `vault_patch timed out after 60s. Server may be under load or a lock is contended; retry shortly.` In BOTH cases the patch had actually committed; the file was already in its new state when the retry ran, and the retry then failed with `patch N: find text not found.` (because the original anchor text no longer existed). Pattern observed twice in the same session.</context>
<parameter name="problem">The naïve retry path on `vault_patch` timeout (call the same patch again with the same find/replace) produces a false `find text not found` failure that masks the fact that the first call succeeded. Wasted ~30s per occurrence verifying after the fact. If the file was not idempotent under the new state, a retry could also corrupt content.
**Problem:** 
**Solution:** On any `vault_patch` timeout, do NOT immediately retry. Instead: (a) `mcp__hive__vault_query` the file to inspect current state, (b) determine if the patch text is already applied (by checking for the post-state content), (c) only retry if the original anchor still exists. This is the same defensive pattern as setup-script idempotence: read-then-write, never blind-write-twice. Treat the Hive timeout response as "outcome unknown" not "outcome failed".
**Tags:** `#hive` `#vault` `#mcp` `#idempotence`

### [2026-05-21] Fix ALL surfaces in the same PR when a bug class spans multiple call sites
**Context:** Today's claude-mem upstream bug cascade. BUG-016 fixed `.mcp.json` cascade-pipe EPIPE race but deferred `hooks.json` as "future BUG-017". Minutes after BUG-016 merged, user hit hooks.json fail → BUG-017 had to ship. Then BUG-017 fixed the pipe but UserPromptSubmit still blocked because hook command lacked `{"continue":true}` directive → BUG-018 narrow (only session-init). Minutes later, Stop hook failed 9x in a row → BUG-018 extended via regex to all 5 hooks. Three deferrals, ~30 min user pain each.
**Problem:** When patching a bug class affecting multiple call sites of the same upstream system, "fix one surface and ship" creates a guaranteed cascade. Each subsequent surface fail is a separate ticket, separate context-switch, separate ~30min of user-visible breakage. The cost of audit-all-surfaces-once is low (10-20 min grep + apply); the cost of skipping is unbounded.
**Solution:** When discovering a bug pattern (e.g. `break; }; done` cascade race; missing `{"continue":true}` directive), the SAME PR that fixes the first surface MUST: (a) `grep -E '<broken-pattern>' <relevant-files>` to enumerate every callsite; (b) apply the same fix or regex-substitute across all of them; (c) bats parity asserts that lock the substitution count. Companion to BUG-011's "audit all call sites of vulnerable upstream API" lesson — applies the same principle one layer down (pattern instead of API).
**Tags:** `#bug-class` `#audit-discipline` `#claude-mem` `#regex-substitution` `#cascade-cost`

### [2026-05-21] The detection probe MUST use the race-free pattern, not the upstream broken one
**Context:** BUG-015 (PR #81, today) added a healthcheck probe to detect when claude-mem's path-resolution cascade fails. The probe used the EXACT SAME `{ printf; ls; printf; } | while ... break` pattern as the upstream hooks it was checking. After BUG-017 patched the upstream hooks with `head -n1`, the probe itself STILL raced — reporting false-positive FAIL because the probe-internal EPIPE fires before resolution completes. BUG-022 had to ship a separate fix to apply the same `head -n1` to the probe.
**Problem:** When writing a detection/observability layer for a known upstream bug pattern, copying the broken pattern verbatim "to faithfully reproduce what the hook does" preserves the same failure mode in the detection layer. The probe inherits the bug it was designed to detect → false-positive FAILs masking real state.
**Solution:** Probes MUST use the canonical race-free version of the pattern being detected. The probe is not the same as the upstream — its job is to ACCURATELY REPORT state, not faithfully reproduce broken behaviour. If the bug-class signature still appears in source (e.g. `break; }; done` in hooks.json), the probe should grep for the broken signature in the FILE (cheap, deterministic) rather than re-execute the broken logic. Cross-OS parity matters: even when the race is silent on Linux (SIGPIPE clean), use the race-free form for consistency.
**Tags:** `#observability` `#detection-layer` `#race-condition` `#healthcheck` `#cross-os-parity`

### [2026-05-21] Healthcheck must validate end-state, not proxy artifacts
**Context:** BUG-014 (PR #75, today). The pre-existing healthcheck assertion for claude-mem (BUG-012 era) checked whether the filesystem JUNCTION existed at `~/.claude/plugins/marketplaces/thedotmack/`. The junction exists if BUG-012's heal ran. But the heal only runs if the marketplace dir is present, and the marketplace dir presence does NOT imply the plugin is installed in `installed_plugins.json` (Claude Code's canonical install record). Result: healthcheck reported `PASS: claude-mem marketplace legacy junction present` while `/mem-search` was unavailable, session-start hook never fired, and `installed_plugins.json` had zero `@thedotmack` entries. False positive that hid the real bug for days.
**Problem:** Asserting a proxy artifact (filesystem junction = consequence of heal) instead of canonical state (installed_plugins.json = source of truth for "is the plugin actually installed") makes the healthcheck unable to detect a whole class of failures. The asymmetry is dangerous: proxy artifacts can exist WITHOUT the canonical state being correct.
**Solution:** Every healthcheck assertion should validate the END-STATE that the user cares about, not a proxy. For "is plugin X installed?" → grep `installed_plugins.json`. For "is service Y running?" → query the service status, not "config file exists". For "is alias Z configured?" → invoke the alias and check exit, not "alias line present in profile". When a proxy is the only available signal, the assertion message should explicitly say "proxy" — and a PRIMARY canonical assertion should come first.
**Tags:** `#healthcheck` `#observability` `#false-positive` `#end-state-vs-proxy`

### [2026-05-21] PowerShell single-quoted strings + grep BRE: backslash counting
**Context:** BUG-017 PR #84 first CI run failed on bats test 471. The grep pattern was `'hooks\\\\hooks\.json'` in single-quoted bash to match a PowerShell single-quoted path `'hooks\hooks.json'`. With 4 backslashes the regex matched `hooks\\hooks.json` (two literal backslashes) — wrong. Reducing to 2 backslashes made it match `hooks\hooks.json` (one literal backslash) — correct.
**Problem:** The "double the backslashes" rule of thumb (count literal `\` then × 2 for regex escape) over-applies when the source string is ALREADY single-quoted bash. In bash single-quotes, every char is literal — no shell-level escape happens. So `'\\'` is exactly 2 chars sent to grep. Grep BRE treats `\\` as one literal `\`. Hence 2 backslashes in bash single-quote == 1 literal backslash matched. Adding 4 backslashes gives 2 literal matches, which over-shoots when the source has only one.
**Solution:** Counting rule for grep BRE inside bash single-quote: target literal backslashes × 2 (NOT × 4). Examples: `'\\'` → matches `\`; `'\\\\'` → matches `\\`; `'\\\\\\\\'` → matches `\\\\`. Tip: extract the pattern into a `pat=$'...'` ANSI-C variable for double-quote interpolation rules instead, OR use `grep -F` (fixed string, no regex) when you don't need wildcards — sidesteps the entire escape rabbit hole.
**Tags:** `#bash` `#powershell` `#grep` `#regex-escape` `#bats`

### [2026-05-21] Heal scripts versioned against the upstream bug class they paper over
**Context:** BUG-016 (PR #83, today). `claude-mem-heal.{sh,ps1}::Repair-McpJson` was authored against v12.7.4's broken `${_R%/}` literal in `.mcp.json` (PR #57, 2026-05-19). v13.0.0+ shipped a different broken pattern: cascading-printf via `sh -c` triggering the EPIPE race. The heal silently no-oped against v13.3.0 installs because the v12.7.4 signature was absent. User hit the upstream MCP failure repeatedly while the heal kept exiting clean — `[claude-mem-heal] .mcp.json already healthy: ...` was a false claim.
**Problem:** Heal scripts are bug-class-specific by design (they paper over a SPECIFIC broken upstream pattern). When the upstream changes its bug pattern (intentionally or accidentally — v12 → v13 shipped different brokenness), the heal's detection regex no longer matches, and the heal becomes a no-op. The script reports "healthy" while the install is broken. Worst: silent failure.
**Solution:** When the upstream version changes AND a bug class is still being papered over, the heal's detection MUST be refreshed in the same investigation that discovers the new pattern. (a) Detect each known broken signature with an OR (don't replace the v12 detection with v13 — keep both, since rollbacks happen). (b) Log which signature was patched so future audits can map heal output → upstream version. (c) Bats parity asserts must lock BOTH detection signatures (each on its own assert) so regressions surface in CI. (d) Add a stronger assertion in healthcheck that validates the canonical fix actually landed (the `head -n1` substring in the patched file), not just that the heal "ran". Companion to BUG-014's end-state-not-proxy lesson.
**Tags:** `#heal-pattern` `#upstream-versioning` `#defensive-scripting` `#claude-mem`

### [2026-05-21] Safety-net fixes must be audited against the same bug-class they paper over
**Context:** BUG-022 (PR #87, 2026-05-21) was a fix for BUG-015's hook-resolution probe. The probe originally re-executed the same `break; }; done` EPIPE-race pattern as the upstream claude-mem hooks. BUG-022 appended `head -n1` after the while-loop to make the pipeline race-free. The same day, hours later, BUG-023 surfaced: the BUG-022 fix STILL raced under `set -euo pipefail` when 2+ candidates matched in cache (this user had 6 versions: 12.7.4 -> 13.3.0). `head -n1` closed the consumer; leftover printfs in the while loop got EPIPE; pipefail propagated 141; `set -e` killed healthcheck.sh mid-section 4/12. setup-linux.sh logged a false-positive WARNING. The fix was a half-fix for the same bug-class.
**Problem:** When a safety-net fix patches a bug-class (race condition, escape bug, error-suppression pattern), it's tempting to believe the patch fully closes the class. But a partial patch can leave a structurally identical sub-case open -- exactly the same bug-class, just in a different scenario count (0-1 matches vs 2+ matches; single producer vs multiple producers; single-arg vs varargs). The next time someone reads the code, the fix looks defensive and complete, masking the residual vulnerability. Cascade-cost lesson applies recursively: every safety-net iteration is itself a candidate for the same audit.
**Solution:** When shipping a fix for a bug-class, before merging, ask: 'could the same bug-class symptom still fire in a different cardinality, a different producer/consumer count, or a different shell mode (pipefail on/off, set -e on/off, strict-mode on/off)?' If the answer is maybe, explicitly enumerate the cases and add a bats parity assert with BOTH positive (new pattern present) AND negative (old broken pattern absent) for each cardinality. Concrete pattern for pipe-with-early-close races: materialize candidates into a variable first, then iterate in pure bash with `break` -- no pipe at all means no consumer-close, no EPIPE, no pipefail propagation. Avoid `done | head -n1` form entirely; it is a half-fix dressed up as a full one. Companion to the 'detection probe must use the race-free pattern' lesson -- applies it recursively to the fix itself.
**Tags:** `#safety-net` `#audit-discipline` `#race-condition` `#bash` `#pipefail` `#cascade-cost` `#claude-mem`

### [2026-05-21] Classify "extract boilerplate" audit findings as bootstrap (chicken-and-egg) vs logic before estimating LOC savings
**Context:** AUDIT-005 (REFACTOR-001 scripts/ audit, 2026-05-21) proposed POLISH-001: extract get_script_dir + utils.sh source-fallback boilerplate to utils.sh, estimating -75-85 LOC reduction. During closeout investigation, the actual numbers came out very differently: the boilerplate is bootstrap code that runs BEFORE utils.sh is sourced. A helper inside utils.sh cannot replace bootstrap that's needed to find utils.sh in the first place — chicken-and-egg. The only extraction path is a new _bootstrap.sh file each script sources via a 1-liner; net ~-10 LOC after counting the bootstrap file content, with +1 file overhead and non-trivial script-loading risk. Decision: WONTFIX.
**Problem:** Audit-style analysis tends to flag "repeated code across N files" without distinguishing whether the repetition is structural (must happen before any extraction target exists) or logical (after extraction target is available). The two have very different extractability and ROI characteristics. Conflating them produces misleadingly high LOC-saving estimates that look like quick wins but require either a new file (with its own ongoing maintenance cost) or sophisticated meta-tricks like self-locating libraries. The audit-005 agent treated the 11-script SCRIPT_DIR pattern + 10-script utils.sh source pattern as if they were ordinary code duplication, but both are bootstrap.
**Solution:** When an audit (your own or an agent's) flags "extract boilerplate" or "shared-pattern extraction", classify EACH proposed extraction before estimating value: (1) Is the repeated code BOOTSTRAP — runs before the would-be helper file is loaded (e.g., resolving where to find the helper)? Inextricable without adding new files or shell meta-tricks. The honest LOC saving is near-zero. (2) Is the repeated code LOGIC — runs after the helper is loaded? Extractable with a normal function. The LOC saving is real. For bootstrap-class patterns, the cost of extraction (new file + ongoing maintenance + loading-order risk) usually exceeds the LOC saving. Document the classification in the audit so future readers don't re-litigate. Concrete test: can the audit explain how the helper itself would be loaded inside the same boilerplate it's trying to replace? If not, it's bootstrap.
**Tags:** `#audit-discipline` `#refactoring` `#bash` `#boilerplate` `#bootstrap` `#chicken-and-egg` `#loc-estimation`

### [2026-05-21] Setup-time mutations to repo-symlinked files create permanent drift false-positives
**Context:** BUG-024 (PR #93). After REFACTOR-001 audit chain closed earlier today, the next ./setup-linux.sh fresh run reported `[12/12] Repo ↔ Deploy-Dir Drift FAIL`. diff-check.sh (PR #10) showed 3 lines drifting in both .bashrc and .zshrc: opencode PATH, project-init alias, and ~/.dotfiles/scripts on PATH.</context>
<parameter name="problem">setup-linux.sh appended those 3 lines to ~/.bashrc and ~/.zshrc via `ensure_line_in_file` *after* symlinking them to the repo (L431-433 opencode PATH, L903-905 project-init alias, L922-924 scripts PATH). Because the rc files are symlinks into the deploy-dir, the appends always wrote through to the deploy-dir copies — making those copies diverge from the (clean) repo source on every fresh setup. The drift detector then flagged it on the very next CI run. The repo source rc files also lacked a trailing newline, which caused the first appended line to concatenate onto `fi`.</problem>
<parameter name="solution">Make the repo the SINGLE writer for any file the drift detector watches. Bake the 3 lines directly into the repo source .bashrc/.zshrc (with trailing newline), delete the 3 `ensure_line_in_file` blocks from setup-linux.sh. `ensure_line_in_file` remains a valid pattern for *external* rc files (files the dotfiles repo does NOT own), but NEVER for files the setup has already symlinked from the repo. Test invariant rewritten in tests/opencode.bats #5 to assert repo-as-SSOT and forbid the old `ensure_line_in_file` pattern for these specific lines. Sibling of `lesson_dotfiles_two_tier_deploy.md` — same family of two-tier-deploy bugs where a writer bypasses the SSOT.</solution>
<parameter name="tags">["drift-detection", "setup-idempotence", "two-tier-deploy", "BUG-024"]
**Problem:** 
**Solution:** 

### [2026-05-21] Byte-equivalence assertions require SCRIPT_DIR control, not just literal diff
**Context:** SDD-004 (PR #97). Claimed acceptance criterion: refactor preserves byte-identical output. First attempt at the assertion ran `git show main:scripts/claude-session-start.sh > $(mktemp /tmp/...sh)` then diffed PRE vs POST. False-positive diff appeared (claude-mem heal block missing in PRE, vault-health line different in PRE).</context>
<parameter name="problem">The `mktemp /tmp/...sh` for PRE put the script in /tmp, which changed its SCRIPT_DIR resolution. The pre-refactor script looked for sibling helpers (`claude-mem-heal.sh`, `vault-health.sh`, `doctor.sh`) in /tmp/, didn't find them, so silently skipped those injectors. POST ran from real `scripts/` directory and found them. The diff was 100% methodology artifact — both versions of the script behaved IDENTICALLY when given identical sibling-script paths. Worse failure mode masked by this: a real refactor regression could be hidden under the same diff noise.</parameter>
<parameter name="solution">For any refactor that asserts byte-identical output via PRE-vs-POST diff: the PRE script copy MUST live in the same directory as POST so `SCRIPT_DIR`-relative lookups (sibling scripts, configs, fixtures) resolve identically. Pattern: write PRE to `<script-dir>/<name>.sh.pre-refactor`, run, diff, delete. Captured in tests/session-start-config.bats #14 (byte-equivalence test) + verification.md row 2. Sibling caveat: if the script's output ALSO includes live state queries (vault-health unresolved-links count changes per minute), pure twice-run-deterministic isn't guaranteed — but PRE-vs-POST at the SAME MOMENT cancels that drift, so the SCRIPT_DIR fix is sufficient. Reframe of R1 from "literal byte-equivalence" to "code-controlled byte-equivalence at fixed SCRIPT_DIR".</parameter>
<parameter name="tags">["byte-equivalence", "testing-methodology", "SDD-004", "verify-before-act", "refactor-safety-net"]
**Problem:** 
**Solution:** 

### [2026-05-25] Batch-scaffold N specs in one PR from a research worktree, defer implementation

**Context:** Did a research worktree comparing 3 reference dotfiles repos (fmontes / holman / mathiasbynens) against this repo. Research surfaced 6 actionable ideas with clear ROI tiering. Two paths forward: (a) open 6 separate PRs, one per spec, paid out over weeks; (b) batch-scaffold all 6 in one PR, defer implementation to per-spec branches.

**Problem:** If each spec gets its own scaffolding PR, the cross-spec dependencies discovered during research (IDEAS-003 enables clean IDEAS-001/002 integration; IDEAS-005 + IDEAS-004 interlock on fresh-machine setup; IDEAS-006 has an abandon gate) get rediscovered or lost across review cycles. Also: 6 small scaffolding PRs each pay the spec-gate / CI / review overhead, but produce zero shippable code. The research doc itself (`research/dotfiles-survey.md`) is the durable index — splitting it across 6 PRs scatters it.

**Solution:** Bundle the research doc + all N spec folders into ONE PR (PR #101, 1,629 LOC, 19 files, 0 code changes). Outcomes: (1) spec-gate workflow PASSED with NO `skip-sdd` label because the new `specs/<id>/` folders ARE the active specs satisfying the gate — confirmed empirically on PR #101; (2) cross-spec dependencies + ordering recommendations live in the PR body as the canonical handoff; (3) implementation work is fully isolated per-spec on `feat/IDEAS-NNN-*` branches with full reviewer attention; (4) the research worktree is disposable after merge. Each spec includes BLOCKER-classified risks, testable AC, features.json skeleton — same shape as SDD-004 so reviewers parse them identically.

**Why:** research outputs go stale fast; batch-capturing them as specs locks in the analytical work before the context window or session memory loses it. **How to apply:** any future research session (compare-X-against-Y, audit-N-instances-of-Z, survey-the-field) → bundle the research doc + every surfaced spec into one scaffold PR before opening implementation branches. Use Tier-1/2/3 + LOC estimate + cross-spec dependencies in the PR body to drive the implementation order.

**Tags:** `#sdd` `#specs` `#research-pattern` `#pr-strategy` `#dotfiles-survey`

### [2026-05-26] Incomplete migration: file rename leaves callers stale
**Context:** SDD-007 renamed GEMINI.md → AGY.md across the dotfiles project. Three separate cleanup PRs (#105, #108, #109) all post-merge found different surfaces still referencing the old name or stale assumptions.
**Problem:** When migrating an identity file (X.md → Y.md), it's tempting to think the work is "rename the file in repo + deploy from new path". But the FILE rename touches 3 distinct surfaces that all need updating in lockstep:

1. **The filesystem**: pre-migration installs accumulate the orphan file. Setup must `rm -f old_name` BEFORE copying new_name, idempotently.
2. **Tooling that READS by name**: healthcheck assertions, init-project copies, bats tests — every callsite that grep'd for "X.md" or did `Test-Path X.md` is now broken or asserting stale state.
3. **The CONTENT inside the new file**: the H1, body text, and embedded references may still describe the old tool. The rename ships a hollow shell unless the migration also refreshes prose.

In SDD-007 specifically: PR #102 did the file rename. PR #105 (orphan + healthcheck refs + init-project refs), PR #108 (stale content inside AGY.md), and PR #109 (hc path mismatch) each unwound one of the three surfaces — three separate post-merge fix PRs for what should have been one cohesive migration.</problem>
<parameter name="solution">For any rename/migration PR going forward, the checklist BEFORE merge is:

```bash
# 1. Find every caller of the old name (every surface)
grep -rIn "OLD_NAME" --exclude-dir={node_modules,.git,specs/archive} .

# 2. Confirm every hit is either:
#    - Updated to NEW_NAME, OR
#    - An intentional historical reference (archive, vault lesson), OR
#    - A cleanup statement (rm -f OLD_NAME with explanatory comment)

# 3. Check the NEW file's content, not just its filename
head -20 path/to/NEW_NAME  # Does the H1 / body still describe OLD?

# 4. For each callsite, ask: "what happens on the next setup run for a
#    user whose machine has the pre-migration state?" — if the answer
#    isn't "the orphan gets removed and the new state is reconciled",
#    the migration is incomplete.
```

Better still: add a `tests/` regression guard that bans the old name in the canonical SSOT (e.g. `! grep -qF '# GEMINI.md' "$DOTFILES_DIR/ai/agy/AGY.md"`). That makes "completion" mechanical rather than reviewer-vigilance.

The pattern is "incomplete migration" — same root cause class as "set -u: requires ${1:-} for optional positional parameters" (one-place fix masking missed-callsite bugs). When you change something, all its callers need to come with you.
**Solution:** 
**Tags:** `#migration` `#refactoring` `#pre-merge-discipline` `#regression-guards` `#dotfiles`

### [2026-05-26] Stop fighting agent filesystem expectations

**Context:** SDD-007 was triggered by BUG-100 — `agy` (Antigravity CLI) v1.0.2 collided with our deploy strategy: agy writes in-place to `~/.gemini/...` paths, but our `setup-linux.sh` had placed symlinks at those paths pointing back to the repo. Symptoms: EEXIST errors (forum #145851), circular link traversal (gemini-cli issue #10960), silent state corruption. The reflex fix would have been "patch each collision case as it surfaces". The root-cause fix was different.

**Problem:** Our deploy strategy was *fighting the agent's expected filesystem layout*, not the agent being buggy. Every new agent we adopt (today: agy, opencode, claude-mem; tomorrow: Cursor, Codex, Devin, whatever) has its own conventions for what it writes-in-place, what it expects to be a regular file, what it tolerates as a symlink. If our deploy strategy hardcodes a single mechanism (symlinks pointing back to the repo), we will hit BUG-100-class incidents N times — once per agent that disagrees with that mechanism.

**Solution:** Default deploy mechanism = **copy** (atomic via `deploy_file` helper, idempotent via `cmp -s`). Symlinks become the *exception*, reserved for: (a) intentional vault↔home bindings (Obsidian vault content → memory paths), (b) secret files needing canonical absolute paths (`~/.ssh/id_ed25519`, `~/.config/age/key.txt`). All other config paths get copies. The "edit-in-`~`-silently-loses-change" failure mode that copies introduce gets neutralized by a `check_deployed` drift assertion in `scripts/healthcheck.sh` — drift surfaces loud red. ADR-012 captures the decision.

**Why:** Symlinks express "this path IS the repo file" — that's a strong claim about filesystem identity that not every consumer of the path respects. Copies express "this path is *a copy of* the repo file at deploy time" — a weaker, more universal claim. The weaker claim is portable across N agents; the stronger one only works as long as all consumers cooperate. SDD-008 will extend this discipline to skills (vault `00_meta/skills/` → render via copy to each agent's native path, no symlinks).

**How to apply:** When adding a new deploy target (any path under `~/.*/` that the repo owns), default to `deploy_file` (copy). Only use symlinks when there's a SPECIFIC, documented reason (cross-system binding that must stay live; secret with hard-coded absolute path requirement). Update the `DEPLOYED_FILES` registry so the drift assertion covers the new path. If the target is consumed by a CLI tool that writes-in-place to its own configuration directory, *always* use copy — don't even debate it.

**Tags:** `#deploy` `#cross-agent` `#filesystem` `#sdd-007` `#bug-100` `#adr-012` `#dotfiles`

### [2026-05-26] pkill -f self-kill: pattern matches the pkill command line itself

**Context:** During a cleanup session uninstalling several AI CLIs (opencode, agy, gemini-cli), I ran `pkill -TERM -f 'opencode|antigravity|/agy\b|gemini-cli'` to terminate any live processes. The shell exited with code 144 (128 + SIGTERM 16) — pkill killed the shell that invoked it.

**Problem:** `pkill -f PATTERN` matches against each process's **full command line** (argv as a single string). The shell running my pkill command had a command line containing the literal string `pkill -TERM -f 'opencode|antigravity|/agy\b|gemini-cli'` — and `opencode` substring of that command line matches the pattern `opencode|...`. So pkill cheerfully matched ITS OWN INVOKING SHELL and signaled it. The shell got SIGTERM mid-command and died. This is silent and confusing: the cleanup appears to have failed (exit 144) but actually nothing was killed except the wrapper itself.

This is a recurring class — anyone writing a script that uses `pkill -f` with a pattern broad enough to match human-readable words will hit this on certain shell invocations (especially when the pattern is constructed dynamically from user input or config). The bash-tool-in-an-AI-loop case is particularly prone because each command is launched in its own subshell whose argv contains the full command string.

**Solution:** Three safer alternatives in order of robustness:

```bash
# 1. Match the basename only (no -f flag) — kills by program name not command line.
#    Best when you know the exact binary name.
pkill -x opencode 2>/dev/null
pkill -x agy 2>/dev/null

# 2. With -f, explicitly exclude $$ (current shell PID).
MYPID=$$
pgrep -f 'opencode|/agy|gemini-cli' | grep -v "^$MYPID$" | xargs -r kill -TERM

# 3. Use a sentinel that the pattern won't match in your own command.
#    Wrap the pattern in [oo]pencode trick used by ps/grep — the brackets aren't
#    literal in regex but the LITERAL string '[o]pencode' doesn't match 'opencode'.
pkill -f '[o]pencode|[a]ntigravity'   # matches OUR processes only
```

**Why:** `pkill -f` is one of those tools where the obvious mental model ("match programs whose name contains X") is wrong — it actually matches "processes whose entire command line contains X", and your own pkill invocation IS such a process. The semantics are documented but counter-intuitive, and the failure mode (exit 144, no error message, your script terminated) gives no clue what happened.

**How to apply:** Default to `pkill -x BASENAME` when you can. Only use `pkill -f PATTERN` when you genuinely need command-line matching (e.g., killing one specific instance of a daemon with distinguishing args), AND in that case either exclude `$$` explicitly OR use the `[X]regex` self-exclusion trick. In any AI-agent loop or wrapper script context, never trust `pkill -f` with broad patterns without one of these guards — the wrapper's own argv WILL match common English words.

**Tags:** `#shell` `#process-management` `#pkill` `#self-kill` `#exit-codes` `#dotfiles` `#ai-loop-gotcha`

### [2026-05-27] MEMORY.md files break YAML parsing with `# currentDate` and `---` separators

**Context**: 23 `MEMORY.md` files across all projects had invalid frontmatter. `vault_health --checks frontmatter` reported errors on every one.

**Problem**: Three distinct YAML-breaking patterns:
1. **`# currentDate` after frontmatter** — YAML sees `# currentDate` as a second document (comments after `---` close the first doc).
2. **`> Updated:` block scalar** — YAML interprets `>` as a folded block scalar, then chokes on the next line (e.g., `**Last task:**` doesn't start with a comment or line break).
3. **`---` in body** — Session handoff blocks use `---` as dividers, which YAML interprets as a new document separator.

**Solution**: Wrap the entire body in a single YAML `content: |` literal block scalar under the frontmatter. **Do NOT use a trailing `---`** — that starts a second document. The file is still valid Markdown (Obsidian renders it correctly). Pattern:
```yaml
---
id: "<project>-memory"
type: memory
status: active
tags: [memory, <project>, claude-code]
content: |
  # Title
  
  > block scalar text
  > more text
  ---
  another section
```
Note: the body's `---` is safe because it's inside the `|` scalar.

**Rule**: When a Markdown file needs both valid YAML frontmatter AND body content that may contain YAML-reserved characters (`>`, `---`, `#` at start of line), wrap the body in a `content: |` literal block scalar. Never use a trailing `---` after frontmatter — it starts a second YAML document. For Obsidian compatibility: the `content` key is ignored by Obsidian's markdown renderer (it only reads frontmatter + body), so the file renders identically.

**Tags:** `#yaml` `#frontmatter` `#obsidian` `#memory` `#validation` `#structural-debt`

### [2026-05-26] PowerShell -replace with [\s\S]*? expands large strings instead of replacing
**Context:** setup-windows.ps1 profile-section block used -replace regex to update dotfiles section in PowerShell profile
**Problem:** PowerShell's -replace operator with [\s\S]*? regex pattern EXPANDS large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single -replace run, then 30+ markers on subsequent runs. The -replace and [regex]::Replace both failed — same behavior. Root cause: PowerShell regex engine backtracking on large strings with non-greedy [\s\S]*?.
**Solution:** Replace regex-based replace with index-based split/join: use IndexOf() to find marker positions, Substring() to extract before/after, and string concatenation to build the result. Also add -NoNewline to Set-Content to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.
**Tags:** `#powershell` `#regex` `#replace` `#idempotency` `#profile` `#bug`
### [2026-05-27] PowerShell -replace with [\s\S]*? expands large strings instead of replacing

**Context**: `setup-windows.ps1` profile-section block used `-replace` regex to update dotfiles section in PowerShell profile.

**Problem**: PowerShell's `-replace` operator with `[\s\S]*?` regex pattern **expands** large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single `-replace` run, then 30+ markers on subsequent runs. Both `-replace` and `[regex]::Replace` failed identically — same behavior.

**Solution**: Replace regex-based replace with index-based split/join: use `IndexOf()` to find marker positions, `Substring()` to extract before/after, and string concatenation to build the result. Also add `-NoNewline` to `Set-Content` to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.

**Rule**: Never use PowerShell's `-replace` operator with `[\s\S]*?` on strings >10KB. Use `IndexOf()` + `Substring()` + concatenation instead.
### [2026-05-27] PowerShell -replace with [\s\S]*? expands large strings instead of replacing

**Context**: `setup-windows.ps1` profile-section block used `-replace` regex to update dotfiles section in PowerShell profile.

**Problem**: PowerShell's `-replace` operator with `[\s\S]*?` regex pattern **expands** large strings (>10KB) instead of replacing. A profile with 1 marker and 0 errors became 4 markers and 5 errors after a single `-replace` run, then 30+ markers on subsequent runs. Both `-replace` and `[regex]::Replace` failed identically.

**Solution**: Replace regex-based replace with index-based split/join: use `IndexOf()` to find marker positions, `Substring()` to extract before/after, and string concatenation. Also add `-NoNewline` to `Set-Content` to prevent trailing newline drift (2 bytes per run). Verified: 5 consecutive runs = 0 accumulation, 0 parse errors, constant size.

**Rule**: Never use PowerShell's `-replace` operator with `[\s\S]*?` on strings >10KB. Use `IndexOf()` + `Substring()` + concatenation instead.### [2026-05-27] Obsidian CLI package name @vorillaz/obsidian-cli does not exist on npm

**Context**: Both `setup-linux.sh` and `setup-windows.ps1` attempted `npm install -g '@vorillaz/obsidian-cli'` but the package returns 404 on npm registry.

**Problem**: The Obsidian CLI was referenced with a wrong package name (`@vorillaz/obsidian-cli`) that does not exist on npm. The correct package is `obsidian-cli` (no scope). Both Linux and Windows setup scripts silently failed to install it, causing `FAIL: Obsidian CLI not in PATH` in healthcheck.

**Solution**: Updated both setup scripts to use `npm install -g 'obsidian-cli'`. Updated all bats tests that grep for the old package name.

**Rule**: Always verify npm package names exist before committing them to setup scripts. `npm view <package> name version` before using in automated install.
### [2026-05-27] Split-Path -LiteralPath and -Parent are mutually exclusive parameter sets in PowerShell
**Context:** Fixing setup-windows.ps1 post-run errors. utils.ps1:58 used Split-Path -LiteralPath $Destination -Parent which threw Parameter set cannot be resolved.
**Problem:** Split-Path -LiteralPath and -Parent are mutually exclusive parameter sets in PowerShell 5.1 and 7.x. Using both together throws: "Parameter set cannot be resolved using the specified named parameters."
**Solution:** Split-Path has distinct parameter sets: LiteralPathSet (accepts -LiteralPath WITHOUT -Parent) and ParentSet (accepts -Path WITH -Parent). Replace -LiteralPath with -Path when using -Parent. The fix: $dstDir = Split-Path -Path $Destination -Parent (1 line change).
**Tags:** `#powershell` `#bugfix` `#cross-platform` `#split-path`

### [2026-05-30] Verify a plan's file-map against the code before editing load-bearing scripts
**Context:** SDD-008 skill pipeline (PR #179). The implementation plan specified which loops to remove from setup-linux.sh / setup-windows.ps1 to migrate skill deploy to render-at-deploy ("agy ~L428, claude ~L505").
**Problem:** The plan's file-map was incomplete/wrong: the loop that actually produced the symlinks AC1 forbids was the vault-skill symlink loop (setup-linux.sh:1087 / setup-windows.ps1:594), which the plan did not mention. There were 5 skill-deploy loops per OS, not 2. Editing only the loops the plan named would have left AC1 (zero symlinks) silently failing, and opencode deployed from a separate pipeline (skills-to-opencode.sh -> ai/opencode/commands) the plan also missed.
**Solution:** Before editing, grep the actual code for every site that touches the thing being migrated (here: every loop writing to ~/.claude/skills, ~/.config/opencode/commands, ~/.gemini/*, and every vault-symlink loop) and reconcile that against the plan. Treat a plan's line numbers / file list as a hypothesis to verify, not ground truth. A wrong file-map makes an acceptance criterion fail silently because the obvious edits look complete.
**Tags:** `#sdd` `#verification` `#deploy` `#shell`

### [2026-05-30] A whole-file transform must inspect the data shape before assuming a uniform model
**Context:** SDD-008: migrating skill deploy to render-at-deploy. The render kind 'skill' was first implemented to render only SKILL.md to the agent path.
**Problem:** Two vault skills carry auxiliary files (systematic-debugging has 5: scripts + reference .md; test-driven-development has 1). The old claude deploy did `cp -rf "$skill_dir"*` (whole directory), so a SKILL.md-only render would have silently dropped those reference files — a functional regression invisible to a test that only checks SKILL.md.
**Solution:** Before committing to a transform model, enumerate the inputs (here: `find vault/00_meta/skills -type f ! -name SKILL.md`). For dir-based renders, copy the whole record dir then overlay the rendered SKILL.md; single-file renders (opencode command, agy prompt) legitimately take only SKILL.md. Add a test that asserts an auxiliary file lands at the dir-based target and does NOT at the single-file targets.
**Tags:** `#sdd` `#skills` `#deploy` `#testing`

### [2026-05-30] Don't commit to a shared vault that has another session's staged work — use an isolated worktree
**Context:** SDD-011 close-out. The vault (`~/Projects/knowledge`) was being edited in parallel by another session that had a feature branch (`rfd-001/pdf-modifier-mcp`) checked out with *staged* work (a pollex restructure). I committed a scoped 2-file SDD-011 change there.
**Problem:** A scoped `git commit -- <paths>` correctly isolated my 2 files (it did NOT sweep the other session's staged work), but the commit landed on the *other session's* feature branch. When that session switched branches, the vault working tree reverted and my change was stranded off `master` (the canonical branch). Committing into a repo another agent/human is actively branch-switching is racy and misplaces the work.
**Solution:** When you must read or operate on a specific commit of a repo another session is using, use `git worktree add --detach <tmp> <commit>` — an isolated checkout that shares only the object store, never the live HEAD/index. To place a change on the right branch without disturbing the active checkout: `git worktree add <tmp> master` + `git cherry-pick <commit>`, then `git worktree remove`. Detect the hazard early: a `git status` showing *staged* changes you did not make means another session owns this checkout — do not commit there.
**Tags:** `#vault` `#git` `#worktree` `#parallel-sessions` `#sdd`

### [2026-05-31] A structural integrity guard surfaces latent issues you didn't know you had
**Context:** SDD-012 built a backlog-integrity guard (`check-backlog-integrity.sh`) to stop the dotfiles `11-tasks.md` from drifting (the same ticket listed twice with diverging status).
**Problem:** Run on the real file, the guard flagged not only the expected view-duplication but **9 number collisions** — two *different* tickets sharing one number (`BUG-020-pwsh` vs `BUG-020-splitpath`, `IDEAS-007-promote` vs `IDEAS-007-cross-provider`, ...). Nobody had noticed; enforcing the invariant exposed them. It also forced a guard-design refinement: match by FULL id (slug-aware), not by number, so legitimate-but-untidy reuse becomes an advisory NOTE instead of a false "duplicate" that would demand history-desyncing renumbers.
**Solution:** When you add a structural guard (incident→guard), expect it to surface adjacent latent issues beyond the bug class it targets — budget for that, and decide up front which are hard-fails vs advisories. Match the guard to the real identity of the thing (full id), distinguishing "same thing listed twice" (drift → hard-fail) from "two things share a label" (collision → advisory). An over-strict guard demands harmful fixes.
**Tags:** `#sdd` `#incident-to-guard` `#backlog` `#guard-design`

### [2026-05-31] A SCRIPT_DIR root-resolution fix breaks CWD-pinned fixture tests — add an env-override seam, not a code branch
**Context:** PR #192 changed compile-harness.sh root resolution from `git rev-parse --show-toplevel` (CWD-based) to SCRIPT_DIR-based (`cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.."`), to fix the section-12 healthcheck drift false-fail on the non-git ~/.dotfiles copy-deploy (ADR-012), where `git rev-parse` errors.
**Problem:** 26 compile-harness.bats tests run the REAL script against a throwaway fixture they `cd` into, relying on CWD-based (git-toplevel) root resolution to make the fixture the root. SCRIPT_DIR resolution instead pointed the script at the LIVE repo, so --refresh/--deploy operated on the wrong tree → mass failure. The two goals (robust resolution for the deploy copy vs. arbitrary fixture root for tests) are irreconcilable with a single hard-coded REPO_ROOT. A second, brittle test also broke: a guard that greps healthcheck.sh for an exact failure-message string, after #192 reworded the message.
**Solution:** Introduce an explicit env override: `REPO_ROOT="${HARNESS_REPO_ROOT:-$(SCRIPT_DIR-based)}"` — the SAME idiom the script already uses for VAULT_PATH. Tests `export HARNESS_REPO_ROOT="$REPO"` once in setup() to pin the fixture; production never sets it and keeps the SCRIPT_DIR default. One seam, two correct behaviors, zero production code branch. Generalize the existing override idiom rather than special-casing tests. Corollary: prefer grepping a stable SUBSTRING of a message (or asserting behavior) over an exact full-string match, so message rewording doesn't break the guard.
**Tags:** `#testing` `#bats` `#root-resolution` `#deploy-model` `#env-override` `#compile-harness`

### [2026-05-31] A skip-guarded test is green in CI but a real assertion locally — it can hide a genuine cross-OS parity gap
**Context:** While running the full bats suite locally to verify PR #192, tests/antigravity.bats test 64 ("setup-windows.ps1 syncs Shared Skills to ~/.gemini/skills") FAILED locally, yet the very same commit was GREEN in CI.
**Problem:** The test carries `skip "agy not installed (CI / fresh machine)"`. On CI (fresh runner) agy is absent → the test skips → counts as `ok`. On a dev box with agy installed, the skip condition is false → the real assertion runs → red. The assertion is a static grep proving setup-windows.ps1 contains the `sharedSkillsDir` / "Synced Shared Skills to" sync that setup-linux.sh has; those strings are currently ABSENT from setup-windows.ps1. So a genuine Windows-parity gap (or a stale test left behind when SDD-008's compile-harness --deploy subsumed the old skills-sync block) is invisible in CI and surfaces only locally.
**Solution:** Treat skip-guarded tests as CI-BLIND: green-in-CI does NOT mean the contract holds when the skip condition is "tool not installed on CI". For a cross-OS-parity-of-SOURCE assertion (static grep of a script's text), there is no reason to skip on a missing runtime tool — gate it on file presence, not on `agy` being installed, so CI actually runs it. Reserve skips for tests that exercise the tool's RUNTIME behavior. When such a local-only red appears, record it as an open thread / Windows-empirical ticket rather than assuming the suite is clean.
**Tags:** `#testing` `#ci` `#skip-guard` `#cross-os-parity` `#false-green` `#windows`

### [2026-05-31] Onboarding a junior/remote agent: verify its self-authored docs, enforce boundaries mechanically
**Context:** HERMES-001 — integrating Hermes, a low-capability remote ops agent (Debian 13 on NaN infra, Telegram), into the dotfiles ecosystem via ai/hermes/setup.sh + a curated vault SSOT at 80_agents/hermes-nan/.
**Problem:** The agent's own self-authored vault docs had silently drifted from reality: validate.sh checked filenames (memory.md, skills.md) the folder no longer used (numbered 10-memory.md, 11-skills.md), a constitution AGENTS.md was referenced everywhere but did not exist, and the vault clone lived in ephemeral /tmp so a reboot lost the auto-push git hook. Separately, the write-zone boundary (commit only within 80_agents/) was instruction-only — a junior agent can ignore instructions, and prompt-injection or a bug could push anywhere.
**Solution:** Treat a junior/remote agent's own docs as CLAIMS to verify, not truth. Probe the real box before designing provisioning — config path (/hermes-home/config.yaml, not ~/.hermes/), the commit mechanism (git CLI vs MCP — determines whether git hooks fire), and credential handling (token was embedded in the remote URL) were ALL different from assumptions; one probe round saved a wrong design. Convert soft instruction-only boundaries into MECHANICAL guardrails: once the probe confirmed Hermes commits via git CLI, install local git hooks in its clone — pre-commit rejecting paths outside the write-zone + token-like content, pre-push rejecting non-fast-forward (force) pushes — each with a functional test. Hooks are local to the agent's clone (never tracked/cloned), so they harden one consumer without touching others.
**Tags:** `#hermes` `#agent-onboarding` `#guardrails` `#remote-agent` `#verify-before-trust` `#git-hooks`

### [2026-05-31] A cross-environment SSOT validator must split "content drift" (fail) from "runtime absent off-box" (warn)
**Context:** HERMES-001 Track B. `80_agents/hermes-nan/scripts/validate.sh` checks the vault SSOT for the Hermes agent; AC6 is "vault SSOT consistent, validate.sh green". The script also checked box-only runtime facts (post-commit hook, cron entry, `uvx`).
**Problem:** Those runtime checks only pass on the provisioned Hermes box. As hard failures they made AC6 unprovable anywhere else — the script could never be green from a dev checkout or CI, so "green" had no portable meaning.
**Solution:** Split the exit policy by what the artifact is SSOT for. `fail` (exit 1) = vault SSOT inconsistency (a required file missing/malformed), content the vault owns and that holds everywhere. `warn` (exit 0) = box-runtime advisory (hook/cron/tool), environmental and absent off-box. A green run then means "SSOT internally consistent" regardless of where it runs, and the same script still does the full check on the box. Generalize: a validator that mixes content invariants with environment state must rank them, or it asserts nothing portable.
**Tags:** `#validation` `#ssot` `#hermes` `#exit-codes` `#cross-environment`

### [2026-06-02] A harness consumer spec scoped on a retired axis must be reconciled, not implemented (WORKMODE-001)

**Context:** Picking up the HARNESS-001 backlog, the next planned consumer was WORKMODE-001 (#159) — "the harness adapts its knowledge-SSOT target to repo type (personal → vault, work → repo + Project)". Before implementing, a vault+repo sweep checked whether the work/personal model had already been decided.

**Problem:** It had. `pattern-knowledge-placement` (KPM-001, 2026-05-28) replaced the **work/personal** axis with a **decide-vs-operate-by-layer** axis: build/operate docs → repo `docs/` for *every* placement-model repo; cross-project brain → vault; collaborate → forge. WORKMODE-001's premise ("personal → vault") was the retired axis. Implementing it as written would have hard-coded the obsolete split into the deploy engine — the opposite of the fix. Worse, `AGENTS.md` (the cross-agent SSOT) still carried the old axis in Standing Orders #2/#3/#7 + the Document Dynamic / Lessons sections, contradicting its own Neural Hive section and mis-routing artifacts (the kubelab regression: a personal+placement repo's lesson sent back to the vault). The open reconciliation ticket #197 already named this.

**Solution:** Fuse #197 + #159 into one spec that retires the axis instead of cementing it: make decide-vs-operate primary across all of AGENTS.md, demote work/personal to its one residual (where the cross-project *brain* lives + whether tasks sit on a shared board) as a defaulted `## Knowledge Placement` declaration, re-key tooling guards from "for work projects" to "for any placement-model repo", and ship an incident→guard (`tests/agents-md.bats`) that fails if any build/operate class is routed to the vault again. Verify a held spec's *premise* against later patterns/ADRs before writing code — a spec is a hypothesis with a shelf life, not a work order.

**Tags:** `#sdd` `#knowledge-placement` `#decide-vs-operate` `#agents-md` `#ssot` `#incident-to-guard` `#reconcile-dont-implement`
